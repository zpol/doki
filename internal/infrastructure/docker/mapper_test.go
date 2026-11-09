package docker

import (
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"

	"github.com/zpol/doki/internal/domain/models"
)

func TestMapSummary(t *testing.T) {
	s := container.Summary{
		ID:      "0123456789abcdef0123456789abcdef",
		Names:   []string{"/nginx"},
		Image:   "nginx:latest",
		State:   "running",
		Status:  "Up 3 minutes (healthy)",
		Created: 1_700_000_000,
		Ports: []container.Port{
			{IP: "0.0.0.0", PrivatePort: 80, PublicPort: 8080, Type: "tcp"},
		},
		Labels: map[string]string{
			"com.docker.compose.project": "web",
			"com.docker.compose.service": "proxy",
		},
	}

	got := mapSummary(s)
	if got.Name != "nginx" {
		t.Fatalf("name=%q", got.Name)
	}
	if got.Status != models.StatusRunning {
		t.Fatalf("status=%q", got.Status)
	}
	if got.Health != models.HealthHealthy {
		t.Fatalf("health=%q", got.Health)
	}
	if got.ComposeProject != "web" || got.ComposeService != "proxy" {
		t.Fatalf("compose labels not mapped: %#v", got)
	}
	if got.ShortID() != "0123456789ab" {
		t.Fatalf("short id=%q", got.ShortID())
	}
}

func TestParseHealthFromStatus(t *testing.T) {
	cases := map[string]models.HealthStatus{
		"Up 1 minute (healthy)":           models.HealthHealthy,
		"Up 1 minute (unhealthy)":         models.HealthUnhealthy,
		"Up Less than a second (health: starting)": models.HealthStarting,
		"Exited (0) 2 minutes ago":        models.HealthNone,
	}
	for in, want := range cases {
		if got := parseHealthFromStatus(in); got != want {
			t.Fatalf("%q => %q want %q", in, got, want)
		}
	}
}

func TestRedactAwareInspectMapping(t *testing.T) {
	raw := container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			ID:   "abc",
			Name: "/api",
			State: &container.State{
				Status:   "running",
				ExitCode: 0,
			},
			RestartCount: 2,
			HostConfig: &container.HostConfig{
				Privileged: true,
				NetworkMode: "bridge",
			},
		},
		Config: &container.Config{
			Image: "api:1",
			Env:   []string{"PATH=/usr/bin", "API_TOKEN=supersecret"},
			Labels: map[string]string{
				"com.docker.compose.project": "demo",
			},
		},
	}
	got := mapInspect(raw)
	if got.Name != "api" {
		t.Fatalf("name=%q", got.Name)
	}
	if !got.Privileged {
		t.Fatal("expected privileged")
	}
	if got.RestartCount != 2 {
		t.Fatalf("restarts=%d", got.RestartCount)
	}
	if got.ComposeProject != "demo" {
		t.Fatalf("project=%q", got.ComposeProject)
	}
}

func TestMapInspectPortsFromNetworkSettings(t *testing.T) {
	raw := container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			ID:   "abc",
			Name: "/web",
			State: &container.State{
				Status: "running",
			},
			HostConfig: &container.HostConfig{},
		},
		Config: &container.Config{
			Image: "nginx:latest",
			ExposedPorts: nat.PortSet{
				"80/tcp":  {},
				"443/tcp": {},
			},
		},
		NetworkSettings: &container.NetworkSettings{
			NetworkSettingsBase: container.NetworkSettingsBase{
				Ports: nat.PortMap{
					"80/tcp": []nat.PortBinding{
						{HostIP: "0.0.0.0", HostPort: "8080"},
					},
					"443/tcp": nil,
				},
			},
		},
	}
	got := mapInspect(raw)
	if len(got.Ports) < 2 {
		t.Fatalf("expected ports, got %#v", got.Ports)
	}
	var foundPublished, foundExposed bool
	for _, p := range got.Ports {
		if p.PrivatePort == 80 && p.PublicPort == 8080 && p.Type == "tcp" {
			foundPublished = true
		}
		if p.PrivatePort == 443 && p.PublicPort == 0 && p.Type == "tcp" {
			foundExposed = true
		}
	}
	if !foundPublished {
		t.Fatalf("missing published 80->8080 in %#v", got.Ports)
	}
	if !foundExposed {
		t.Fatalf("missing exposed 443 in %#v", got.Ports)
	}
}

func TestMapInspectPortsFromHostConfigFallback(t *testing.T) {
	raw := container.InspectResponse{
		ContainerJSONBase: &container.ContainerJSONBase{
			ID:   "abc",
			Name: "/web",
			State: &container.State{Status: "exited"},
			HostConfig: &container.HostConfig{
				PortBindings: nat.PortMap{
					"3000/tcp": []nat.PortBinding{
						{HostIP: "127.0.0.1", HostPort: "3000"},
					},
				},
			},
		},
		Config: &container.Config{Image: "app:1"},
	}
	got := mapInspect(raw)
	if len(got.Ports) != 1 {
		t.Fatalf("ports=%#v", got.Ports)
	}
	p := got.Ports[0]
	if p.PrivatePort != 3000 || p.PublicPort != 3000 || p.IP != "127.0.0.1" {
		t.Fatalf("unexpected port %#v", p)
	}
}
