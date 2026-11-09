package containers

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/zpol/doki/internal/domain/models"
	"github.com/zpol/doki/internal/domain/ports"
)

// Service provides container operations for presentation layers.
type Service struct {
	docker ports.DockerClient
}

// New creates a container application service.
func New(docker ports.DockerClient) *Service {
	return &Service{docker: docker}
}

// ListOptions controls listing behavior.
type ListOptions struct {
	ShowStopped bool
	Query       string
}

// List returns filtered and sorted containers.
func (s *Service) List(ctx context.Context, opts ListOptions) ([]models.Container, error) {
	items, err := s.docker.ListContainers(ctx, ports.ContainerFilter{All: true})
	if err != nil {
		return nil, err
	}

	query := strings.ToLower(strings.TrimSpace(opts.Query))
	out := make([]models.Container, 0, len(items))
	for _, c := range items {
		if !opts.ShowStopped && !c.IsRunning() && c.Status != models.StatusPaused && c.Status != models.StatusRestarting {
			continue
		}
		if query != "" && !matches(c, query) {
			continue
		}
		out = append(out, c)
	}

	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := out[i].IsRunning(), out[j].IsRunning()
		if ri != rj {
			return ri
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

// Inspect returns detailed container information.
func (s *Service) Inspect(ctx context.Context, id string) (models.ContainerDetails, error) {
	if strings.TrimSpace(id) == "" {
		return models.ContainerDetails{}, fmt.Errorf("container id is required")
	}
	return s.docker.InspectContainer(ctx, id)
}

// Start starts a container.
func (s *Service) Start(ctx context.Context, id string) error {
	return s.docker.StartContainer(ctx, id)
}

// Stop stops a container.
func (s *Service) Stop(ctx context.Context, id string) error {
	timeout := 10
	return s.docker.StopContainer(ctx, id, &timeout)
}

// Restart restarts a container.
func (s *Service) Restart(ctx context.Context, id string) error {
	timeout := 10
	return s.docker.RestartContainer(ctx, id, &timeout)
}

// Pause pauses a container.
func (s *Service) Pause(ctx context.Context, id string) error {
	return s.docker.PauseContainer(ctx, id)
}

// Unpause unpauses a container.
func (s *Service) Unpause(ctx context.Context, id string) error {
	return s.docker.UnpauseContainer(ctx, id)
}

// Remove deletes a container without removing volumes.
func (s *Service) Remove(ctx context.Context, id string, force bool) error {
	return s.docker.RemoveContainer(ctx, id, ports.RemoveOptions{
		Force:         force,
		RemoveVolumes: false,
	})
}

func matches(c models.Container, query string) bool {
	fields := []string{
		c.Name,
		c.ID,
		c.ShortID(),
		c.Image,
		string(c.Status),
		c.StatusText,
		c.ComposeProject,
		c.ComposeService,
	}
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), query) {
			return true
		}
	}
	return false
}
