package docker

import (
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/go-connections/nat"

	"github.com/zpol/doki/internal/domain/models"
)

func toFilters(in map[string][]string) filters.Args {
	args := filters.NewArgs()
	for key, values := range in {
		for _, v := range values {
			args.Add(key, v)
		}
	}
	return args
}

func mapSummary(s container.Summary) models.Container {
	labels := s.Labels
	if labels == nil {
		labels = map[string]string{}
	}

	ports := make([]models.PortBinding, 0, len(s.Ports))
	for _, p := range s.Ports {
		ports = append(ports, models.PortBinding{
			IP:          p.IP,
			PrivatePort: p.PrivatePort,
			PublicPort:  p.PublicPort,
			Type:        p.Type,
		})
	}

	return models.Container{
		ID:             s.ID,
		Name:           containerName(s.Names),
		Image:          s.Image,
		Status:         mapStatus(string(s.State)),
		StatusText:     s.Status,
		Health:         parseHealthFromStatus(s.Status),
		CreatedAt:      time.Unix(s.Created, 0).UTC(),
		Ports:          ports,
		Labels:         labels,
		ComposeProject: labels["com.docker.compose.project"],
		ComposeService: labels["com.docker.compose.service"],
	}
}

func mapInspect(raw container.InspectResponse) models.ContainerDetails {
	summary := models.Container{
		ID:     raw.ID,
		Name:   strings.TrimPrefix(raw.Name, "/"),
		Image:  "",
		Status: models.StatusUnknown,
	}
	if raw.Config != nil {
		summary.Image = raw.Config.Image
		summary.Labels = raw.Config.Labels
		if summary.Labels == nil {
			summary.Labels = map[string]string{}
		}
		summary.ComposeProject = summary.Labels["com.docker.compose.project"]
		summary.ComposeService = summary.Labels["com.docker.compose.service"]
	}
	if raw.State != nil {
		summary.Status = mapStatus(raw.State.Status)
		summary.StatusText = raw.State.Status
		summary.Health = mapHealth(raw.State.Health)
	}
	if raw.Created != "" {
		if t, err := time.Parse(time.RFC3339Nano, raw.Created); err == nil {
			summary.CreatedAt = t.UTC()
		}
	}
	summary.RestartCount = raw.RestartCount

	details := models.ContainerDetails{
		Container: summary,
		Path:      raw.Path,
		Args:      append([]string(nil), raw.Args...),
		Platform:  raw.Platform,
		Driver:    raw.Driver,
	}

	if raw.Config != nil {
		details.Env = append([]string(nil), raw.Config.Env...)
		details.Cmd = append([]string(nil), raw.Config.Cmd...)
		details.Entrypoint = append([]string(nil), raw.Config.Entrypoint...)
		details.WorkingDir = raw.Config.WorkingDir
		details.User = raw.Config.User
	}
	if raw.HostConfig != nil {
		details.NetworkMode = string(raw.HostConfig.NetworkMode)
		details.Privileged = raw.HostConfig.Privileged
		details.RestartPolicy = string(raw.HostConfig.RestartPolicy.Name)
		details.ReadonlyRootfs = raw.HostConfig.ReadonlyRootfs
		details.CapAdd = append([]string(nil), []string(raw.HostConfig.CapAdd)...)
		details.CapDrop = append([]string(nil), []string(raw.HostConfig.CapDrop)...)
		details.SecurityOpt = append([]string(nil), raw.HostConfig.SecurityOpt...)
	}
	if raw.State != nil {
		details.ExitCode = raw.State.ExitCode
		details.OOMKilled = raw.State.OOMKilled
		details.Pid = raw.State.Pid
		if t, err := time.Parse(time.RFC3339Nano, raw.State.StartedAt); err == nil {
			details.StartedAt = t.UTC()
		}
		if t, err := time.Parse(time.RFC3339Nano, raw.State.FinishedAt); err == nil {
			details.FinishedAt = t.UTC()
		}
	}

	for _, m := range raw.Mounts {
		details.Mounts = append(details.Mounts, models.Mount{
			Type:        string(m.Type),
			Name:        m.Name,
			Source:      m.Source,
			Destination: m.Destination,
			Mode:        m.Mode,
			RW:          m.RW,
		})
	}

	if raw.NetworkSettings != nil {
		details.Ports = mapPortMap(raw.NetworkSettings.Ports)
		for name, ep := range raw.NetworkSettings.Networks {
			if ep == nil {
				continue
			}
			details.Networks = append(details.Networks, models.NetworkEndpoint{
				NetworkID:   ep.NetworkID,
				NetworkName: name,
				IPAddress:   ep.IPAddress,
				Gateway:     ep.Gateway,
				MacAddress:  ep.MacAddress,
			})
		}
	}

	// Fallback: HostConfig port bindings when runtime Ports is empty
	// (e.g. stopped containers still have configured publishes).
	if len(details.Ports) == 0 && raw.HostConfig != nil && len(raw.HostConfig.PortBindings) > 0 {
		details.Ports = mapPortMap(raw.HostConfig.PortBindings)
	}
	// Also surface exposed-but-unpublished ports from image/config.
	if raw.Config != nil {
		for p := range raw.Config.ExposedPorts {
			priv, proto := parseNatPort(p)
			if priv == 0 {
				continue
			}
			if hasPrivatePort(details.Ports, priv, proto) {
				continue
			}
			details.Ports = append(details.Ports, models.PortBinding{
				PrivatePort: priv,
				Type:        proto,
			})
		}
	}

	return details
}

func mapPortMap(ports nat.PortMap) []models.PortBinding {
	if len(ports) == 0 {
		return nil
	}
	out := make([]models.PortBinding, 0, len(ports))
	for p, bindings := range ports {
		priv, proto := parseNatPort(p)
		if len(bindings) == 0 {
			out = append(out, models.PortBinding{
				PrivatePort: priv,
				Type:        proto,
			})
			continue
		}
		for _, b := range bindings {
			pub, _ := strconv.ParseUint(b.HostPort, 10, 16)
			out = append(out, models.PortBinding{
				IP:          b.HostIP,
				PrivatePort: priv,
				PublicPort:  uint16(pub),
				Type:        proto,
			})
		}
	}
	return out
}

func parseNatPort(p nat.Port) (uint16, string) {
	proto := p.Proto()
	if proto == "" {
		proto = "tcp"
	}
	n, err := nat.ParsePort(p.Port())
	if err != nil || n < 0 || n > 65535 {
		return 0, proto
	}
	return uint16(n), proto
}

func hasPrivatePort(ports []models.PortBinding, priv uint16, proto string) bool {
	for _, p := range ports {
		if p.PrivatePort == priv && strings.EqualFold(p.Type, proto) {
			return true
		}
	}
	return false
}
