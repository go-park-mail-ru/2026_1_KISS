package container

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type dockerAPI interface {
	Close() error
	ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error)
	ContainerCreate(ctx context.Context, config *container.Config,
		hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig,
		platform *ocispec.Platform, containerName string) (container.CreateResponse, error)
	ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error
	ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error
	ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error)
	GetAvailableRuntimes() ([]string, error)
}

type DockerAdapter struct {
	cli *client.Client
}

func NewDockerAdapter() (*DockerAdapter, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerAdapter{cli: cli}, nil
}

func (a *DockerAdapter) Close() error {
	return a.cli.Close()
}

func (a *DockerAdapter) ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error) {
	return a.cli.ContainerInspect(ctx, containerID)
}

func (a *DockerAdapter) ContainerCreate(ctx context.Context, config *container.Config,
	hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig,
	platform *ocispec.Platform, containerName string) (container.CreateResponse, error) {
	return a.cli.ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, containerName)
}

func (a *DockerAdapter) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	return a.cli.ContainerStart(ctx, containerID, options)
}

func (a *DockerAdapter) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	return a.cli.ContainerRemove(ctx, containerID, options)
}

func (a *DockerAdapter) ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error) {
	return a.cli.ContainerList(ctx, options)
}

func (a *DockerAdapter) GetAvailableRuntimes() ([]string, error) {
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
