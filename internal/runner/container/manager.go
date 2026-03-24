package container

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	cerrdefs "github.com/containerd/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/config"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner"
)

const (
	managedLabelKey = "kiss.runner.managed"
	sessionLabelKey = "kiss.runner.session_id"
)

type Manager struct {
	docker     dockerAPI
	cfg        config.RunnerConfig
	httpClient *http.Client
	waitReady  func(ctx context.Context, httpClient *http.Client, baseURL string, timeout, interval time.Duration) error
}

func NewManager(cfg config.RunnerConfig) (*Manager, error) {
	adapter, err := NewDockerAdapter()
	if err != nil {
		return nil, fmt.Errorf("create docker adapter: %w", err)
	}
	return NewManagerWithAPI(cfg, adapter), nil
}

func NewManagerWithAPI(cfg config.RunnerConfig, docker dockerAPI) *Manager {
	return &Manager{
		docker:     docker,
		cfg:        cfg,
		httpClient: &http.Client{},
		waitReady:  waitUntilReady,
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

	for attempt := 0; attempt < 3; attempt++ {
		// пробуем подключиться к уже запущенному контейнеру, если он есть
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
		// если косяк не в отсутствии контейнера, а в чем-то другом - падаем
		if !errors.Is(err, runner.ErrContainerNotFound) {
			return "", err
		}
		// если просто нет контейнера - создаем новый
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
	containerConfig := &container.Config{
		Image: m.cfg.Image,
		Labels: map[string]string{
			managedLabelKey: "true",
			sessionLabelKey: sessionID,
		},
	}

	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			Memory:   m.cfg.MemoryLimitBytes,
			NanoCPUs: m.cfg.NanoCPUs,
		},
		NetworkMode: container.NetworkMode("bridge"),
	}

	networkConfig := &network.NetworkingConfig{}

	return m.docker.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, name)
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
	baseURL := "http://" + address + ":" + m.cfg.AgentPort
	if err := m.waitReady(ctx, m.httpClient, baseURL, m.cfg.StartupTimeout, m.cfg.HealthCheckInterval); err != nil {
		return "", err
	}
	return address, nil
}

func (m *Manager) addressFromInspect(inspect container.InspectResponse) (string, error) {
	if inspect.NetworkSettings == nil {
		return "", fmt.Errorf("container has no network settings")
	}

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
