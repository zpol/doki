package diagnostics

import (
	"testing"

	"github.com/zpol/doki/internal/domain/models"
)

func TestAnalyzeUnhealthy(t *testing.T) {
	d := models.ContainerDetails{
		Container: models.Container{
			ID:     "1",
			Name:   "api",
			Status: models.StatusRunning,
			Health: models.HealthUnhealthy,
		},
	}
	findings := analyze(d)
	found := false
	for _, f := range findings {
		if f.ID == "DIAG-UNHEALTHY" {
			found = true
		}
	}
	if !found {
		t.Fatalf("%#v", findings)
	}
}

func TestAnalyzeOOM(t *testing.T) {
	d := models.ContainerDetails{
		Container: models.Container{ID: "1", Name: "worker"},
		OOMKilled: true,
	}
	findings := analyze(d)
	found := false
	for _, f := range findings {
		if f.ID == "DIAG-OOM" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected OOM finding")
	}
}

func TestAnalyzeRestartCount(t *testing.T) {
	d := models.ContainerDetails{
		Container: models.Container{
			ID:           "1",
			Name:         "loop",
			Status:       models.StatusRunning,
			RestartCount: 5,
		},
	}
	findings := analyze(d)
	found := false
	for _, f := range findings {
		if f.ID == "DIAG-RESTART-LOOP" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected restart finding")
	}
}
