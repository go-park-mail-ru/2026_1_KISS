package container

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/errdefs"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/mocks"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/config"
	"go.uber.org/mock/gomock"
)

func noopWait(_ context.Context, _ *http.Client, _ string, _ time.Duration, _ time.Duration) error {
	return nil
}

func notFoundErr() error {
	return errdefs.NotFound(errors.New("container not found"))
}

func runningInspect(id, ip string) container.InspectResponse {
	return container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			ID:    id,
			State: &container.State{Running: true},
		},
		NetworkSettings: &container.NetworkSettings{
			Networks: map[string]*network.EndpointSettings{
				"bridge": {IPAddress: ip},
			},
		},
	}
}

func stoppedInspect(id, ip string) container.InspectResponse {
	return container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			ID:    id,
			State: &container.State{Running: false},
		},
		NetworkSettings: &container.NetworkSettings{
			Networks: map[string]*network.EndpointSettings{
				"bridge": {IPAddress: ip},
			},
		},
	}
}

func TestStartSession_CreatesAndStartsContainer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	docker := mocks.NewMockDockerAdapter(ctrl)

	gomock.InOrder(
		docker.EXPECT().ContainerInspect(gomock.Any(), "runner-s-1").
			Return(container.InspectResponse{}, notFoundErr()),
		docker.EXPECT().GetAvailableRuntimes().
			Return([]string{}, nil),
		docker.EXPECT().ContainerCreate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "runner-s-1").
			Return(container.CreateResponse{ID: "id-1"}, nil),
		docker.EXPECT().ContainerStart(gomock.Any(), "id-1", gomock.Any()).
			Return(nil),
		docker.EXPECT().ContainerInspect(gomock.Any(), "id-1").
			Return(runningInspect("id-1", "172.19.0.2"), nil),
	)

	mgr := NewManagerWithAPI(config.RunnerConfig{
		Images:              map[string]string{"python": "kiss-python-runner", "r": "kiss-r-runner"},
		NamePrefix:          "runner-",
		AgentPort:           "8080",
		MemoryLimitBytes:    128 * 1024 * 1024,
		NanoCPUs:            500_000_000,
		StartupTimeout:      time.Second,
		HealthCheckInterval: time.Millisecond,
	}, docker, noopWait)

	address, err := mgr.StartSession(context.Background(), "s-1", "python")
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}
	if address != "http://172.19.0.2:8080" {
		t.Fatalf("want address http://172.19.0.2:8080, got %s", address)
	}
}

func TestStartSession_ReusesRunningContainer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	docker := mocks.NewMockDockerAdapter(ctrl)

	docker.EXPECT().ContainerInspect(gomock.Any(), "runner-s-2").
		Return(runningInspect("id-existing", "172.19.0.8"), nil)

	mgr := NewManagerWithAPI(
		config.RunnerConfig{
			Images:     map[string]string{"python": "kiss-python-runner"},
			NamePrefix: "runner-",
			AgentPort:  "8080",
		},
		docker, noopWait)

	address, err := mgr.StartSession(context.Background(), "s-2", "python")
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}
	if address != "http://172.19.0.8:8080" {
		t.Fatalf("want existing address http://172.19.0.8:8080, got %s", address)
	}
}

func TestStartSession_RecreatesStoppedContainer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	docker := mocks.NewMockDockerAdapter(ctrl)

	gomock.InOrder(
		docker.EXPECT().ContainerInspect(gomock.Any(), "runner-s-3").
			Return(stoppedInspect("id-old", "172.19.0.5"), nil),
		docker.EXPECT().ContainerRemove(gomock.Any(), "id-old", gomock.Any()).
			Return(nil),
		docker.EXPECT().ContainerInspect(gomock.Any(), "runner-s-3").
			Return(container.InspectResponse{}, notFoundErr()),
		docker.EXPECT().GetAvailableRuntimes().
			Return([]string{}, nil),
		docker.EXPECT().ContainerCreate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "runner-s-3").
			Return(container.CreateResponse{ID: "id-new"}, nil),
		docker.EXPECT().ContainerStart(gomock.Any(), "id-new", gomock.Any()).
			Return(nil),
		docker.EXPECT().ContainerInspect(gomock.Any(), "id-new").
			Return(runningInspect("id-new", "172.19.0.9"), nil),
	)

	mgr := NewManagerWithAPI(
		config.RunnerConfig{
			Images:     map[string]string{"python": "kiss-python-runner"},
			NamePrefix: "runner-",
			AgentPort:  "8080",
		},
		docker, noopWait)

	address, err := mgr.StartSession(context.Background(), "s-3", "python")
	if err != nil {
		t.Fatalf("StartSession() error = %v", err)
	}
	if address != "http://172.19.0.9:8080" {
		t.Fatalf("want recreated address http://172.19.0.9:8080, got %s", address)
	}
}

func TestGetContainerAddress_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	docker := mocks.NewMockDockerAdapter(ctrl)

	docker.EXPECT().ContainerInspect(gomock.Any(), "runner-missing").
		Return(container.InspectResponse{}, notFoundErr())

	mgr := NewManagerWithAPI(
		config.RunnerConfig{NamePrefix: "runner-"},
		docker, noopWait)

	_, err := mgr.GetContainerAddress(context.Background(), "missing")
	if !errors.Is(err, ErrContainerNotFound) {
		t.Fatalf("want ErrContainerNotFound, got %v", err)
	}
}

func TestStopSession_RemovesContainer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	docker := mocks.NewMockDockerAdapter(ctrl)

	gomock.InOrder(
		docker.EXPECT().ContainerInspect(gomock.Any(), "runner-s-4").
			Return(runningInspect("id-4", "172.19.0.4"), nil),
		docker.EXPECT().ContainerRemove(gomock.Any(), "id-4", gomock.Any()).
			Return(nil),
	)

	mgr := NewManagerWithAPI(
		config.RunnerConfig{NamePrefix: "runner-"},
		docker, noopWait)

	if err := mgr.StopSession(context.Background(), "s-4"); err != nil {
		t.Fatalf("StopSession() error = %v", err)
	}
}

func TestCleanupSessions_RemovesOnlyManaged(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	docker := mocks.NewMockDockerAdapter(ctrl)

	docker.EXPECT().ContainerList(gomock.Any(), gomock.Any()).
		Return([]container.Summary{
			{ID: "id-a", Labels: map[string]string{managedLabelKey: "true"}},
		}, nil)

	docker.EXPECT().ContainerRemove(gomock.Any(), "id-a", gomock.Any()).
		Return(nil)

	mgr := NewManagerWithAPI(
		config.RunnerConfig{NamePrefix: "runner-"},
		docker, noopWait)

	mgr.CleanupSessions(context.Background())
}
