package host

import (
	"context"

	"github.com/zpol/doki/internal/domain/models"
	"github.com/zpol/doki/internal/domain/ports"
)

// Service provides Docker host connectivity information.
type Service struct {
	docker ports.DockerClient
}

// New creates a host application service.
func New(docker ports.DockerClient) *Service {
	return &Service{docker: docker}
}

// Info returns current Docker Engine connection details.
func (s *Service) Info(ctx context.Context) (models.HostInfo, error) {
	return s.docker.HostInfo(ctx)
}

// Ping checks Docker Engine reachability.
func (s *Service) Ping(ctx context.Context) error {
	return s.docker.Ping(ctx)
}
