package docker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"

	"github.com/zpol/doki/internal/domain/models"
)

// ListImages returns local image summaries.
func (c *Client) ListImages(ctx context.Context) ([]models.Image, error) {
	items, err := c.cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list images: %w", err)
	}
	out := make([]models.Image, 0, len(items))
	for _, item := range items {
		repo, tag := "<none>", "<none>"
		if len(item.RepoTags) > 0 && item.RepoTags[0] != "<none>:<none>" {
			parts := strings.SplitN(item.RepoTags[0], ":", 2)
			repo = parts[0]
			if len(parts) == 2 {
				tag = parts[1]
			}
		} else if len(item.RepoDigests) > 0 {
			repo = strings.SplitN(item.RepoDigests[0], "@", 2)[0]
			tag = "<none>"
		}
		out = append(out, models.Image{
			ID:         item.ID,
			Repository: repo,
			Tag:        tag,
			Size:       item.Size,
			CreatedAt:  time.Unix(item.Created, 0).UTC(),
			Containers: item.Containers,
		})
	}
	return out, nil
}

// ListVolumes returns Docker volumes.
func (c *Client) ListVolumes(ctx context.Context) ([]models.Volume, error) {
	resp, err := c.cli.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list volumes: %w", err)
	}
	out := make([]models.Volume, 0, len(resp.Volumes))
	for _, v := range resp.Volumes {
		if v == nil {
			continue
		}
		item := models.Volume{
			Name:       v.Name,
			Driver:     v.Driver,
			Mountpoint: v.Mountpoint,
			Scope:      v.Scope,
			Labels:     v.Labels,
		}
		if v.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339Nano, v.CreatedAt); err == nil {
				item.CreatedAt = t.UTC()
			}
		}
		out = append(out, item)
	}
	return out, nil
}

// ListNetworks returns Docker networks.
func (c *Client) ListNetworks(ctx context.Context) ([]models.Network, error) {
	items, err := c.cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list networks: %w", err)
	}
	out := make([]models.Network, 0, len(items))
	for _, n := range items {
		out = append(out, models.Network{
			ID:         n.ID,
			Name:       n.Name,
			Driver:     n.Driver,
			Scope:      n.Scope,
			Internal:   n.Internal,
			Attachable: n.Attachable,
			Containers: len(n.Containers),
		})
	}
	return out, nil
}

// InspectImage returns detailed image information.
func (c *Client) InspectImage(ctx context.Context, id string) (models.ImageDetails, error) {
	raw, err := c.cli.ImageInspect(ctx, id)
	if err != nil {
		return models.ImageDetails{}, fmt.Errorf("inspect image %s: %w", id, err)
	}
	repo, tag := "<none>", "<none>"
	if len(raw.RepoTags) > 0 {
		parts := strings.SplitN(raw.RepoTags[0], ":", 2)
		repo = parts[0]
		if len(parts) == 2 {
			tag = parts[1]
		}
	}
	details := models.ImageDetails{
		Image: models.Image{
			ID:         raw.ID,
			Repository: repo,
			Tag:        tag,
			Size:       raw.Size,
		},
		RepoTags:     append([]string(nil), raw.RepoTags...),
		RepoDigests:  append([]string(nil), raw.RepoDigests...),
		Architecture: raw.Architecture,
		OS:           raw.Os,
		Author:       raw.Author,
		Comment:      raw.Comment,
	}
	if raw.Created != "" {
		if t, err := time.Parse(time.RFC3339Nano, raw.Created); err == nil {
			details.CreatedAt = t.UTC()
		}
	}
	if raw.Config != nil {
		details.Entrypoint = append([]string(nil), raw.Config.Entrypoint...)
		details.Cmd = append([]string(nil), raw.Config.Cmd...)
		details.Env = append([]string(nil), raw.Config.Env...)
		details.WorkingDir = raw.Config.WorkingDir
		details.Labels = raw.Config.Labels
		for p := range raw.Config.ExposedPorts {
			details.ExposedPorts = append(details.ExposedPorts, string(p))
		}
	}
	return details, nil
}

// InspectVolume returns detailed volume information.
func (c *Client) InspectVolume(ctx context.Context, name string) (models.VolumeDetails, error) {
	raw, err := c.cli.VolumeInspect(ctx, name)
	if err != nil {
		return models.VolumeDetails{}, fmt.Errorf("inspect volume %s: %w", name, err)
	}
	details := models.VolumeDetails{
		Volume: models.Volume{
			Name:       raw.Name,
			Driver:     raw.Driver,
			Mountpoint: raw.Mountpoint,
			Scope:      raw.Scope,
			Labels:     raw.Labels,
		},
		Options:  raw.Options,
		RefCount: -1,
		Size:     -1,
	}
	if raw.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, raw.CreatedAt); err == nil {
			details.CreatedAt = t.UTC()
		}
	}
	if raw.UsageData != nil {
		details.RefCount = raw.UsageData.RefCount
		details.Size = raw.UsageData.Size
	}
	return details, nil
}

// InspectNetwork returns detailed network information.
func (c *Client) InspectNetwork(ctx context.Context, id string) (models.NetworkDetails, error) {
	raw, err := c.cli.NetworkInspect(ctx, id, network.InspectOptions{Verbose: true})
	if err != nil {
		return models.NetworkDetails{}, fmt.Errorf("inspect network %s: %w", id, err)
	}
	details := models.NetworkDetails{
		Network: models.Network{
			ID:         raw.ID,
			Name:       raw.Name,
			Driver:     raw.Driver,
			Scope:      raw.Scope,
			Internal:   raw.Internal,
			Attachable: raw.Attachable,
			Containers: len(raw.Containers),
		},
		Created:    raw.Created.UTC(),
		EnableIPv4: raw.EnableIPv4,
		EnableIPv6: raw.EnableIPv6,
		Ingress:    raw.Ingress,
		Options:    raw.Options,
		Labels:     raw.Labels,
	}
	if len(raw.IPAM.Config) > 0 {
		details.Subnet = raw.IPAM.Config[0].Subnet
		details.Gateway = raw.IPAM.Config[0].Gateway
	}
	for cid, ep := range raw.Containers {
		details.Endpoints = append(details.Endpoints, models.NetworkEndpointInfo{
			ContainerID:   cid,
			ContainerName: ep.Name,
			IPv4Address:   ep.IPv4Address,
			IPv6Address:   ep.IPv6Address,
			MacAddress:    ep.MacAddress,
		})
	}
	return details, nil
}
