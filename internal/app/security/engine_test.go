package security

import (
	"testing"

	"github.com/zpol/doki/internal/domain/models"
)

func TestPrivilegedFinding(t *testing.T) {
	d := models.ContainerDetails{
		Container:  models.Container{ID: "1", Name: "agent"},
		Privileged: true,
	}
	findings := privilegedChecker{}.Check(d)
	if len(findings) != 1 || findings[0].Severity != models.SeverityHigh {
		t.Fatalf("%#v", findings)
	}
}

func TestDockerSocketFinding(t *testing.T) {
	d := models.ContainerDetails{
		Container: models.Container{ID: "1", Name: "ci"},
		Mounts: []models.Mount{{
			Source:      "/var/run/docker.sock",
			Destination: "/var/run/docker.sock",
			RW:          true,
		}},
	}
	findings := dockerSocketChecker{}.Check(d)
	if len(findings) != 1 {
		t.Fatalf("%#v", findings)
	}
}

func TestRootUserUnset(t *testing.T) {
	d := models.ContainerDetails{Container: models.Container{ID: "1", Name: "web"}}
	findings := rootUserChecker{}.Check(d)
	if len(findings) != 1 {
		t.Fatalf("expected finding for unset user")
	}
}

func TestPortAllInterfaces(t *testing.T) {
	d := models.ContainerDetails{
		Container: models.Container{
			ID:   "1",
			Name: "api",
			Ports: []models.PortBinding{
				{IP: "0.0.0.0", PublicPort: 8080, PrivatePort: 80, Type: "tcp"},
			},
		},
	}
	findings := portExposureChecker{}.Check(d)
	if len(findings) != 1 || findings[0].Severity != models.SeverityInfo {
		t.Fatalf("%#v", findings)
	}
}

func TestDisclaimer(t *testing.T) {
	if Disclaimer() == "" {
		t.Fatal("empty disclaimer")
	}
}
