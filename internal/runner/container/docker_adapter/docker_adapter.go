//go:generate go run go.uber.org/mock/mockgen -source=docker_adapter.go -destination=../../../mocks/docker_adapter_mock.go -package=mocks
package docker_adapter

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type DockerAdapter interface {
	Close() error
	ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error)
	ContainerCreate(ctx context.Context, config *container.Config,
		hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig,
		platform *ocispec.Platform, containerName string) (container.CreateResponse, error)
	ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error
	ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error
	ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error)
	GetAvailableRuntimes() ([]string, error)
	ContainerStats(ctx context.Context, containerID string, stream bool) (container.StatsResponseReader, error)
}

type dockerAdapter struct {
	cli *client.Client
}

func NewDockerAdapter() (DockerAdapter, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &dockerAdapter{cli: cli}, nil
}

func (a *dockerAdapter) Close() error {
	return a.cli.Close()
}

func (a *dockerAdapter) ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	return a.cli.ContainerInspect(ctx, containerID)
}

func (a *dockerAdapter) ContainerCreate(ctx context.Context, config *container.Config,
	hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig,
	platform *ocispec.Platform, containerName string) (container.CreateResponse, error) {
	return a.cli.ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, containerName)
}

func (a *dockerAdapter) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	return a.cli.ContainerStart(ctx, containerID, options)
}

func (a *dockerAdapter) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	return a.cli.ContainerRemove(ctx, containerID, options)
}

func (a *dockerAdapter) ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error) {
	return a.cli.ContainerList(ctx, options)
}

func (a *dockerAdapter) ContainerStats(ctx context.Context, containerID string, stream bool) (container.StatsResponseReader, error) {
	return a.cli.ContainerStats(ctx, containerID, stream)
}

func (a *dockerAdapter) GetAvailableRuntimes() ([]string, error) {
	info, err := a.cli.Info(context.Background())
	if err != nil {
		return nil, fmt.Errorf("could not get info: %w", err)
	}

	// Собираем только названия рантаймов (ключи map)
	var runtimes []string
	for name := range info.Runtimes {
		runtimes = append(runtimes, name)
	}

	return runtimes, nil
}
