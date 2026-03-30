package container

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	cerrdefs "github.com/containerd/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/config"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner"
)

const (
	managedLabelKey     = "kiss.runner.managed"
	sessionLabelKey     = "kiss.runner.session_id"
	startupAttemptCount = 5
)

type Manager struct {
	docker      dockerAPI
	cfg         config.RunnerConfig
	httpClient  *http.Client
	useHostPort bool // true — хост→Docker (читаем HostPort); false — Docker→Docker (внутренний IP)
	waitReady   func(ctx context.Context, httpClient *http.Client, baseURL string, timeout, interval time.Duration) error
}

func NewManager(cfg config.RunnerConfig) (*Manager, error) {
	adapter, err := NewDockerAdapter()
	if err != nil {
		return nil, fmt.Errorf("create docker adapter: %w", err)
	}
	return NewManagerWithAPI(cfg, adapter), nil
}

func NewManagerWithAPI(cfg config.RunnerConfig, docker dockerAPI) *Manager {
	// RUNNER_USE_HOST_PORT=true означает, что сервис запущен на хосте,
	// а не внутри Docker-сети — используем 127.0.0.1 + проброшенный порт.
	//useHostPort := os.Getenv("RUNNER_USE_HOST_PORT") == "true"

	// Затычка - если запускать app не из контейнера, то runner-контейнеры будут слушать localhost
	useHostPort := false
	if cfg.NetworkName == "bridge" {
		useHostPort = true
	}

	return &Manager{
		docker:      docker,
		cfg:         cfg,
		httpClient:  &http.Client{},
		useHostPort: useHostPort,
		waitReady:   waitUntilReady,
	}
}

func (m *Manager) Close() error {
	return m.docker.Close()
}

func (m *Manager) GetContainerAddress(ctx context.Context, sessionID string) (string, error) {
	inspect, err := m.inspectByName(ctx, m.containerName(sessionID))
	if err != nil {
		return "", err
	}
	if inspect.State == nil || !inspect.State.Running {
		return "", runner.ErrContainerNotFound
	}
	return m.addressFromInspect(inspect)
}

func (m *Manager) StartSession(ctx context.Context, sessionID string) (string, error) {
	name := m.containerName(sessionID)

	for attempt := 0; attempt < startupAttemptCount; attempt++ {
		inspect, err := m.inspectByName(ctx, name)
		if err == nil {
			if inspect.State != nil && inspect.State.Running {
				return m.waitAndReturnAddress(ctx, inspect)
			}
			if err := m.removeContainer(ctx, inspect.ID); err != nil {
				return "", err
			}
			continue
		}
		if !errors.Is(err, runner.ErrContainerNotFound) {
			return "", err
		}

		createResp, err := m.createContainer(ctx, sessionID, name)
		if err != nil {
			if cerrdefs.IsConflict(err) {
				continue
			}
			return "", fmt.Errorf("create container %s: %w", name, err)
		}

		if err := m.docker.ContainerStart(ctx, createResp.ID, container.StartOptions{}); err != nil {
			return "", fmt.Errorf("start container %s: %w", name, err)
		}

		inspect, err = m.docker.ContainerInspect(ctx, createResp.ID)
		if err != nil {
			return "", fmt.Errorf("inspect started container %s: %w", name, err)
		}
		return m.waitAndReturnAddress(ctx, inspect)
	}

	return "", fmt.Errorf("start session %s: container name conflict", sessionID)
}

func (m *Manager) StopSession(ctx context.Context, sessionID string) error {
	inspect, err := m.inspectByName(ctx, m.containerName(sessionID))
	if err != nil {
		if errors.Is(err, runner.ErrContainerNotFound) {
			return nil
		}
		return err
	}
	return m.removeContainer(ctx, inspect.ID)
}

func (m *Manager) CleanupSessions(ctx context.Context) {
	args := filters.NewArgs(filters.Arg("label", managedLabelKey+"=true"))
	containers, err := m.docker.ContainerList(ctx, container.ListOptions{All: true, Filters: args})
	if err != nil {
		return
	}

	for _, c := range containers {
		err = m.removeContainer(ctx, c.ID)
		if err != nil {
			fmt.Printf("failed to remove container %s: %v\n", c.ID, err)
		}
	}
}

func (m *Manager) inspectByName(ctx context.Context, name string) (container.InspectResponse, error) {
	inspect, err := m.docker.ContainerInspect(ctx, name)
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return container.InspectResponse{}, runner.ErrContainerNotFound
		}
		return container.InspectResponse{}, fmt.Errorf("inspect container %s: %w", name, err)
	}
	return inspect, nil
}

func (m *Manager) createContainer(ctx context.Context, sessionID, name string) (container.CreateResponse, error) {
	port := nat.Port(m.cfg.AgentPort + "/tcp")

	containerConfig := &container.Config{
		Image: m.cfg.Image,
		Labels: map[string]string{
			managedLabelKey: "true",
			sessionLabelKey: sessionID,
		},
		ExposedPorts: nat.PortSet{
			port: struct{}{},
		},
	}

	runtimes, err := getAvailableRuntimes()
	if err != nil {
		panic(err)
	}
	var runtimeName string
	if slices.Contains(runtimes, "runsc") {
		runtimeName = "runsc"
	} else {
		fmt.Println(fmt.Errorf("WARNING: runsc runtime not found, using runc instead"))
		runtimeName = "runc"
	}
	hostConfig := &container.HostConfig{
		AutoRemove: true, // сносим контейнер после окончания сессии
		Runtime:    runtimeName,
		Resources: container.Resources{
			Memory:   m.cfg.MemoryLimitBytes,
			NanoCPUs: m.cfg.NanoCPUs,
		},
		NetworkMode: container.NetworkMode(m.cfg.NetworkName), // "bridge", // container.NetworkMode(m.cfg.NetworkName),
		PortBindings: nat.PortMap{
			port: []nat.PortBinding{
				{HostIP: "0.0.0.0", HostPort: "0"}, // случайный свободный порт или 8081
			},
		},
	}

	return m.docker.ContainerCreate(ctx, containerConfig, hostConfig, &network.NetworkingConfig{}, nil, name)
}

func (m *Manager) removeContainer(ctx context.Context, containerID string) error {
	err := m.docker.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true, RemoveVolumes: true})
	if err != nil && !cerrdefs.IsNotFound(err) {
		return fmt.Errorf("remove container %s: %w", containerID, err)
	}
	return nil
}

func (m *Manager) containerName(sessionID string) string {
	prefix := m.cfg.NamePrefix
	if prefix == "" {
		prefix = "runner-"
	}
	return prefix + sessionID
}

func (m *Manager) waitAndReturnAddress(ctx context.Context, inspect container.InspectResponse) (string, error) {
	address, err := m.addressFromInspect(inspect)
	if err != nil {
		return "", err
	}
	baseURL := "http://" + address
	if !m.useHostPort {
		// В режиме внутреннего IP порт фиксированный (AgentPort контейнера)
		baseURL += ":" + m.cfg.AgentPort
	}
	if err := m.waitReady(ctx, m.httpClient, baseURL, m.cfg.StartupTimeout, m.cfg.HealthCheckInterval); err != nil {
		return "", err
	}
	return address, nil
}

// addressFromInspect возвращает адрес контейнера в зависимости от режима:
//   - useHostPort=false (Docker→Docker): внутренний IP из bridge-сети, порт — AgentPort
//   - useHostPort=true  (хост→Docker):   127.0.0.1:HostPort из PortBindings
func (m *Manager) addressFromInspect(inspect container.InspectResponse) (string, error) {
	if inspect.NetworkSettings == nil {
		return "", fmt.Errorf("container has no network settings")
	}

	if m.useHostPort {
		return m.hostPortAddress(inspect)
	}
	return m.bridgeIPAddress(inspect)
}

// bridgeIPAddress — адрес внутри Docker-сети (для Docker→Docker).
func (m *Manager) bridgeIPAddress(inspect container.InspectResponse) (string, error) {
	if endpoint, ok := inspect.NetworkSettings.Networks["bridge"]; ok && endpoint.IPAddress != "" {
		return endpoint.IPAddress, nil
	}
	for _, endpoint := range inspect.NetworkSettings.Networks {
		if endpoint.IPAddress != "" {
			return endpoint.IPAddress, nil
		}
	}
	return "", fmt.Errorf("container has no ip address")
}

// hostPortAddress — адрес 127.0.0.1:HostPort (для хост→Docker).
// Docker пробрасывает AgentPort контейнера на случайный порт хоста.
func (m *Manager) hostPortAddress(inspect container.InspectResponse) (string, error) {
	port := nat.Port(m.cfg.AgentPort + "/tcp")

	bindings, ok := inspect.NetworkSettings.Ports[port]
	if !ok || len(bindings) == 0 {
		return "", fmt.Errorf("no host port binding for container port %s", port)
	}
	hostPort := bindings[0].HostPort
	if hostPort == "" {
		return "", fmt.Errorf("empty host port for container port %s", port)
	}
	return "127.0.0.1:" + hostPort, nil
}

func getAvailableRuntimes() ([]string, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	info, err := cli.Info(context.Background())
	if err != nil {
		return nil, err
	}

	// Собираем только названия рантаймов (ключи map)
	var runtimes []string
	for name := range info.Runtimes {
		runtimes = append(runtimes, name)
	}

	return runtimes, nil
}
