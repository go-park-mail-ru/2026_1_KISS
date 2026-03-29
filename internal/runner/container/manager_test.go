package container

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/errdefs"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/config"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type fakeContainer struct {
	id      string
	name    string
	ip      string
	running bool
	labels  map[string]string
}

type fakeDocker struct {
	containersByName map[string]*fakeContainer
	containersByID   map[string]*fakeContainer
	nextID           int
	nextIP           string
	createErr        error

	lastCreatedName       string
	lastCreatedImage      string
	lastCreatedMemory     int64
	lastCreatedNanoCPUs   int64
	lastCreatedNetwork    string
	removedContainerIDs   []string
	startedContainerIDs   []string
	containerListOverride []container.Summary
}

func newFakeDocker() *fakeDocker {
	return &fakeDocker{
		containersByName: map[string]*fakeContainer{},
		containersByID:   map[string]*fakeContainer{},
		nextIP:           "172.19.0.2",
	}
}

func (f *fakeDocker) ContainerInspect(_ context.Context, containerID string) (container.InspectResponse, error) {
	c, ok := f.containersByName[containerID]
	if !ok {
		c, ok = f.containersByID[containerID]
	}
	if !ok {
		return container.InspectResponse{}, errdefs.NotFound(errors.New("container not found"))
	}

	return container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			ID: c.id,
			State: &container.State{
				Running: c.running,
			},
		},
		NetworkSettings: &container.NetworkSettings{
			Networks: map[string]*network.EndpointSettings{
				"bridge": {
					IPAddress: c.ip,
				},
			},
		},
	}, nil
}

func (f *fakeDocker) ContainerCreate(_ context.Context, cfg *container.Config, hostConfig *container.HostConfig, _ *network.NetworkingConfig, _ *ocispec.Platform, containerName string) (container.CreateResponse, error) {
	if f.createErr != nil {
		return container.CreateResponse{}, f.createErr
	}
	if _, exists := f.containersByName[containerName]; exists {
		return container.CreateResponse{}, errdefs.Conflict(errors.New("name conflict"))
	}

	f.nextID++
	id := fmt.Sprintf("id-%d", f.nextID)
	created := &fakeContainer{
		id:      id,
		name:    containerName,
		ip:      f.nextIP,
		running: false,
		labels:  cfg.Labels,
	}
	f.containersByName[containerName] = created
	f.containersByID[id] = created

	f.lastCreatedName = containerName
	f.lastCreatedImage = cfg.Image
	f.lastCreatedMemory = hostConfig.Resources.Memory
	f.lastCreatedNanoCPUs = hostConfig.Resources.NanoCPUs
	f.lastCreatedNetwork = string(hostConfig.NetworkMode)

	return container.CreateResponse{ID: id}, nil
}

func (f *fakeDocker) ContainerStart(_ context.Context, containerID string, _ container.StartOptions) error {
	c, ok := f.containersByID[containerID]
	if !ok {
		return errdefs.NotFound(errors.New("container not found"))
	}
	c.running = true
	f.startedContainerIDs = append(f.startedContainerIDs, containerID)
	return nil
}

func (f *fakeDocker) ContainerRemove(_ context.Context, containerID string, _ container.RemoveOptions) error {
	c, ok := f.containersByID[containerID]
	if !ok {
		return errdefs.NotFound(errors.New("container not found"))
	}
	delete(f.containersByID, containerID)
	delete(f.containersByName, c.name)
	f.removedContainerIDs = append(f.removedContainerIDs, containerID)
	return nil
}

func (f *fakeDocker) ContainerList(_ context.Context, options container.ListOptions) ([]container.Summary, error) {
	if f.containerListOverride != nil {
		return f.containerListOverride, nil
	}

	labelFilter := ""
	if options.Filters.Len() > 0 {
		for _, raw := range options.Filters.Get("label") {
			if strings.HasPrefix(raw, managedLabelKey+"=") {
				labelFilter = strings.TrimPrefix(raw, managedLabelKey+"=")
			}
		}
	}

	list := make([]container.Summary, 0, len(f.containersByID))
	for _, c := range f.containersByID {
		if labelFilter != "" && c.labels[managedLabelKey] != labelFilter {
			continue
		}
		list = append(list, container.Summary{ID: c.id, Labels: c.labels})
	}
	return list, nil
}

func (f *fakeDocker) Close() error {
	return nil
}

func TestStartSession_CreatesAndStartsContainer(t *testing.T) {
	docker := newFakeDocker()
	mgr := NewManagerWithAPI(config.RunnerConfig{
		Image:               "kiss-runner",
		NamePrefix:          "runner-",
		AgentPort:           "8080",
		MemoryLimitBytes:    128 * 1024 * 1024,
		NanoCPUs:            500_000_000,
		StartupTimeout:      time.Second,
		HealthCheckInterval: time.Millisecond,
	}, docker)
	mgr.waitReady = func(context.Context, *http.Client, string, time.Duration, time.Duration) error {
		return nil
	}

	address, err := mgr.StartSession(context.Background(), "s-1")
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}
	if address != "172.19.0.2" {
		t.Fatalf("want address 172.19.0.2, got %s", address)
	}
	if docker.lastCreatedName != "runner-s-1" {
		t.Fatalf("want container name runner-s-1, got %s", docker.lastCreatedName)
	}
	if docker.lastCreatedImage != "kiss-runner" {
		t.Fatalf("want image kiss-runner, got %s", docker.lastCreatedImage)
	}
	if docker.lastCreatedMemory != 128*1024*1024 {
		t.Fatalf("unexpected memory limit: %d", docker.lastCreatedMemory)
	}
	if docker.lastCreatedNanoCPUs != 500_000_000 {
		t.Fatalf("unexpected nano cpus: %d", docker.lastCreatedNanoCPUs)
	}
	if len(docker.startedContainerIDs) != 1 {
		t.Fatalf("expected 1 started container, got %d", len(docker.startedContainerIDs))
	}
}

func TestStartSession_ReusesRunningContainer(t *testing.T) {
	docker := newFakeDocker()
	docker.containersByName["runner-s-2"] = &fakeContainer{id: "id-existing", name: "runner-s-2", ip: "172.19.0.8", running: true}
	docker.containersByID["id-existing"] = docker.containersByName["runner-s-2"]

	mgr := NewManagerWithAPI(config.RunnerConfig{NamePrefix: "runner-", AgentPort: "8080"}, docker)
	mgr.waitReady = func(context.Context, *http.Client, string, time.Duration, time.Duration) error { return nil }

	address, err := mgr.StartSession(context.Background(), "s-2")
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}
	if address != "172.19.0.8" {
		t.Fatalf("want existing address, got %s", address)
	}
	if len(docker.startedContainerIDs) != 0 {
		t.Fatalf("expected no starts, got %d", len(docker.startedContainerIDs))
	}
}

func TestStartSession_RecreatesStoppedContainer(t *testing.T) {
	docker := newFakeDocker()
	docker.containersByName["runner-s-3"] = &fakeContainer{id: "id-old", name: "runner-s-3", ip: "172.19.0.5", running: false}
	docker.containersByID["id-old"] = docker.containersByName["runner-s-3"]
	docker.nextIP = "172.19.0.9"

	mgr := NewManagerWithAPI(config.RunnerConfig{Image: "kiss-runner", NamePrefix: "runner-", AgentPort: "8080"}, docker)
	mgr.waitReady = func(context.Context, *http.Client, string, time.Duration, time.Duration) error { return nil }

	address, err := mgr.StartSession(context.Background(), "s-3")
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}
	if address != "172.19.0.9" {
		t.Fatalf("want recreated address 172.19.0.9, got %s", address)
	}
	if len(docker.removedContainerIDs) != 1 || docker.removedContainerIDs[0] != "id-old" {
		t.Fatalf("expected removed old container id-old, got %+v", docker.removedContainerIDs)
	}
}

func TestGetContainerAddress_NotFound(t *testing.T) {
	mgr := NewManagerWithAPI(config.RunnerConfig{NamePrefix: "runner-"}, newFakeDocker())
	_, err := mgr.GetContainerAddress(context.Background(), "missing")
	if !errors.Is(err, runner.ErrContainerNotFound) {
		t.Fatalf("want ErrContainerNotFound, got %v", err)
	}
}

func TestStopSession_RemovesContainer(t *testing.T) {
	docker := newFakeDocker()
	docker.containersByName["runner-s-4"] = &fakeContainer{id: "id-4", name: "runner-s-4", ip: "172.19.0.4", running: true}
	docker.containersByID["id-4"] = docker.containersByName["runner-s-4"]

	mgr := NewManagerWithAPI(config.RunnerConfig{NamePrefix: "runner-"}, docker)
	if err := mgr.StopSession(context.Background(), "s-4"); err != nil {
		t.Fatalf("StopSession() error = %v", err)
	}
	if len(docker.removedContainerIDs) != 1 || docker.removedContainerIDs[0] != "id-4" {
		t.Fatalf("expected id-4 removal, got %+v", docker.removedContainerIDs)
	}
}

func TestCleanupSessions_RemovesOnlyManaged(t *testing.T) {
	docker := newFakeDocker()
	docker.containersByName["runner-a"] = &fakeContainer{id: "id-a", name: "runner-a", labels: map[string]string{managedLabelKey: "true"}}
	docker.containersByID["id-a"] = docker.containersByName["runner-a"]
	docker.containersByName["other-b"] = &fakeContainer{id: "id-b", name: "other-b", labels: map[string]string{"other": "true"}}
	docker.containersByID["id-b"] = docker.containersByName["other-b"]

	mgr := NewManagerWithAPI(config.RunnerConfig{NamePrefix: "runner-"}, docker)
	mgr.CleanupSessions(context.Background())

	if len(docker.removedContainerIDs) != 1 || docker.removedContainerIDs[0] != "id-a" {
		t.Fatalf("cleanup should remove only managed container id-a, got %+v", docker.removedContainerIDs)
	}
}
