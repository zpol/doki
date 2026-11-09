package docker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	"github.com/zpol/doki/internal/domain/models"
	"github.com/zpol/doki/internal/domain/ports"
)

// Client wraps the official Docker Engine SDK behind domain ports.
type Client struct {
	cli         *client.Client
	contextName string
	host        string
}

// Options configures Docker client construction.
type Options struct {
	Host    string
	Context string
}

// New creates a Docker client using environment configuration and API negotiation.
func New(opts Options) (*Client, error) {
	clientOpts := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	}
	if opts.Host != "" {
		clientOpts = append(clientOpts, client.WithHost(opts.Host))
	}

	cli, err := client.NewClientWithOpts(clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}

	ctxName := opts.Context
	if ctxName == "" {
		ctxName = "default"
	}

	return &Client{
		cli:         cli,
		contextName: ctxName,
		host:        cli.DaemonHost(),
	}, nil
}

// Close releases the underlying HTTP transport.
func (c *Client) Close() error {
	if c == nil || c.cli == nil {
		return nil
	}
	return c.cli.Close()
}

// Ping verifies Docker Engine connectivity.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.cli.Ping(ctx)
	if err != nil {
		return fmt.Errorf("docker ping: %w", err)
	}
	return nil
}

// HostInfo returns Docker Engine identity and connection state.
func (c *Client) HostInfo(ctx context.Context) (models.HostInfo, error) {
	info := models.HostInfo{
		ContextName: c.contextName,
		Name:        c.host,
		CheckedAt:   time.Now(),
		State:       models.ConnectionConnecting,
	}

	ping, err := c.cli.Ping(ctx)
	if err != nil {
		info.State = models.ConnectionError
		info.LastError = err.Error()
		return info, fmt.Errorf("docker ping: %w", err)
	}
	info.APIVersion = ping.APIVersion

	sys, err := c.cli.Info(ctx)
	if err != nil {
		info.State = models.ConnectionError
		info.LastError = err.Error()
		return info, fmt.Errorf("docker info: %w", err)
	}

	info.State = models.ConnectionConnected
	info.ServerVersion = sys.ServerVersion
	info.OperatingSystem = sys.OperatingSystem
	info.Architecture = sys.Architecture
	info.NCPU = sys.NCPU
	info.MemTotal = sys.MemTotal
	info.Containers = sys.Containers
	info.ContainersRunning = sys.ContainersRunning
	info.ContainersPaused = sys.ContainersPaused
	info.ContainersStopped = sys.ContainersStopped
	info.Images = sys.Images
	info.Driver = sys.Driver
	if sys.Name != "" {
		info.Name = sys.Name
	}
	return info, nil
}

// ListContainers returns container summaries from the Engine API.
func (c *Client) ListContainers(ctx context.Context, filter ports.ContainerFilter) ([]models.Container, error) {
	summaries, err := c.cli.ContainerList(ctx, container.ListOptions{
		All:     filter.All,
		Filters: toFilters(filter.Filter),
	})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	out := make([]models.Container, 0, len(summaries))
	for _, s := range summaries {
		out = append(out, mapSummary(s))
	}
	return out, nil
}

// InspectContainer returns detailed container configuration.
func (c *Client) InspectContainer(ctx context.Context, id string) (models.ContainerDetails, error) {
	raw, err := c.cli.ContainerInspect(ctx, id)
	if err != nil {
		return models.ContainerDetails{}, fmt.Errorf("inspect container %s: %w", id, err)
	}
	return mapInspect(raw), nil
}

// StartContainer starts a stopped container.
func (c *Client) StartContainer(ctx context.Context, id string) error {
	if err := c.cli.ContainerStart(ctx, id, container.StartOptions{}); err != nil {
		return fmt.Errorf("start container %s: %w", id, err)
	}
	return nil
}

// StopContainer stops a running container.
func (c *Client) StopContainer(ctx context.Context, id string, timeoutSeconds *int) error {
	opts := container.StopOptions{Timeout: timeoutSeconds}
	if err := c.cli.ContainerStop(ctx, id, opts); err != nil {
		return fmt.Errorf("stop container %s: %w", id, err)
	}
	return nil
}

// RestartContainer restarts a container.
func (c *Client) RestartContainer(ctx context.Context, id string, timeoutSeconds *int) error {
	opts := container.StopOptions{Timeout: timeoutSeconds}
	if err := c.cli.ContainerRestart(ctx, id, opts); err != nil {
		return fmt.Errorf("restart container %s: %w", id, err)
	}
	return nil
}

// PauseContainer pauses a running container.
func (c *Client) PauseContainer(ctx context.Context, id string) error {
	if err := c.cli.ContainerPause(ctx, id); err != nil {
		return fmt.Errorf("pause container %s: %w", id, err)
	}
	return nil
}

// UnpauseContainer resumes a paused container.
func (c *Client) UnpauseContainer(ctx context.Context, id string) error {
	if err := c.cli.ContainerUnpause(ctx, id); err != nil {
		return fmt.Errorf("unpause container %s: %w", id, err)
	}
	return nil
}

// RemoveContainer removes a container. Volumes are never removed unless opted in.
func (c *Client) RemoveContainer(ctx context.Context, id string, opts ports.RemoveOptions) error {
	if err := c.cli.ContainerRemove(ctx, id, container.RemoveOptions{
		Force:         opts.Force,
		RemoveVolumes: opts.RemoveVolumes,
	}); err != nil {
		return fmt.Errorf("remove container %s: %w", id, err)
	}
	return nil
}

// Ensure Client implements ports.DockerClient.
var _ ports.DockerClient = (*Client)(nil)

func containerName(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return strings.TrimPrefix(names[0], "/")
}

func mapStatus(state string) models.ContainerStatus {
	switch strings.ToLower(state) {
	case "created":
		return models.StatusCreated
	case "running":
		return models.StatusRunning
	case "paused":
		return models.StatusPaused
	case "restarting":
		return models.StatusRestarting
	case "removing":
		return models.StatusRemoving
	case "exited":
		return models.StatusExited
	case "dead":
		return models.StatusDead
	default:
		return models.StatusUnknown
	}
}

func parseHealthFromStatus(status string) models.HealthStatus {
	lower := strings.ToLower(status)
	switch {
	case strings.Contains(lower, "(healthy)"):
		return models.HealthHealthy
	case strings.Contains(lower, "(unhealthy)"):
		return models.HealthUnhealthy
	case strings.Contains(lower, "(health: starting)"):
		return models.HealthStarting
	default:
		return models.HealthNone
	}
}

func mapHealth(h *container.Health) models.HealthStatus {
	if h == nil {
		return models.HealthNone
	}
	switch strings.ToLower(h.Status) {
	case "healthy":
		return models.HealthHealthy
	case "unhealthy":
		return models.HealthUnhealthy
	case "starting":
		return models.HealthStarting
	default:
		return models.HealthUnknown
	}
}
