package resources

import (
	"context"
	"strings"

	"github.com/zpol/doki/internal/domain/models"
	"github.com/zpol/doki/internal/domain/ports"
)

// Service lists Docker images, volumes, and networks.
type Service struct {
	docker ports.DockerClient
}

// New creates a resources service.
func New(docker ports.DockerClient) *Service {
	return &Service{docker: docker}
}

// Images returns filtered images.
func (s *Service) Images(ctx context.Context, query string) ([]models.Image, error) {
	items, err := s.docker.ListImages(ctx)
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return items, nil
	}
	out := make([]models.Image, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Repository), q) ||
			strings.Contains(strings.ToLower(item.Tag), q) ||
			strings.Contains(strings.ToLower(item.ShortID()), q) {
			out = append(out, item)
		}
	}
	return out, nil
}

// Volumes returns filtered volumes.
func (s *Service) Volumes(ctx context.Context, query string) ([]models.Volume, error) {
	items, err := s.docker.ListVolumes(ctx)
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return items, nil
	}
	out := make([]models.Volume, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), q) ||
			strings.Contains(strings.ToLower(item.Driver), q) {
			out = append(out, item)
		}
	}
	return out, nil
}

// Networks returns filtered networks.
func (s *Service) Networks(ctx context.Context, query string) ([]models.Network, error) {
	items, err := s.docker.ListNetworks(ctx)
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return items, nil
	}
	out := make([]models.Network, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), q) ||
			strings.Contains(strings.ToLower(item.Driver), q) ||
			strings.Contains(strings.ToLower(item.ShortID()), q) {
			out = append(out, item)
		}
	}
	return out, nil
}

// InspectImage returns detailed image information.
func (s *Service) InspectImage(ctx context.Context, id string) (models.ImageDetails, error) {
	return s.docker.InspectImage(ctx, id)
}

// InspectVolume returns detailed volume information.
func (s *Service) InspectVolume(ctx context.Context, name string) (models.VolumeDetails, error) {
	return s.docker.InspectVolume(ctx, name)
}

// InspectNetwork returns detailed network information.
func (s *Service) InspectNetwork(ctx context.Context, id string) (models.NetworkDetails, error) {
	return s.docker.InspectNetwork(ctx, id)
}
