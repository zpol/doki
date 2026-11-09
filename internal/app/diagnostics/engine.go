package diagnostics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zpol/doki/internal/domain/models"
	"github.com/zpol/doki/internal/domain/ports"
)

// Engine produces explainable operational findings from Docker state.
type Engine struct {
	docker ports.DockerClient
}

// New creates a diagnostics engine.
func New(docker ports.DockerClient) *Engine {
	return &Engine{docker: docker}
}

// AnalyzeContainer inspects one container for operational issues.
func (e *Engine) AnalyzeContainer(ctx context.Context, id string) ([]models.DiagnosticFinding, error) {
	details, err := e.docker.InspectContainer(ctx, id)
	if err != nil {
		return nil, err
	}
	return analyze(details), nil
}

// AnalyzeAll inspects all containers.
func (e *Engine) AnalyzeAll(ctx context.Context) ([]models.DiagnosticFinding, error) {
	items, err := e.docker.ListContainers(ctx, ports.ContainerFilter{All: true})
	if err != nil {
		return nil, err
	}
	var out []models.DiagnosticFinding
	for _, item := range items {
		findings, err := e.AnalyzeContainer(ctx, item.ID)
		if err != nil {
			continue
		}
		out = append(out, findings...)
	}
	return out, nil
}

func analyze(d models.ContainerDetails) []models.DiagnosticFinding {
	var out []models.DiagnosticFinding

	if d.Health == models.HealthUnhealthy {
		out = append(out, models.DiagnosticFinding{
			ID:           "DIAG-UNHEALTHY",
			ResourceID:   d.ID,
			ResourceName: d.Name,
			Severity:     models.SeverityHigh,
			Observation:  "Container health status is unhealthy.",
			Evidence: []models.Evidence{
				{Field: "Health", Value: string(d.Health)},
				{Field: "Status", Value: d.StatusText},
			},
			PossibleCauses: []string{
				"Application is not responding to the health endpoint.",
				"Dependency is unavailable.",
				"Health check command is misconfigured.",
			},
			RecommendedVerify: []string{
				"Inspect recent health check output.",
				"Review application logs.",
				"Verify dependent services.",
			},
			SuggestedNextSteps: []string{
				"Open logs for this container.",
				"Inspect container configuration.",
			},
			ConfidenceNote: "Based on Docker health status only; root cause is not confirmed.",
		})
	}

	if d.RestartCount >= 3 && (d.Status == models.StatusRunning || d.Status == models.StatusRestarting) {
		out = append(out, models.DiagnosticFinding{
			ID:           "DIAG-RESTART-LOOP",
			ResourceID:   d.ID,
			ResourceName: d.Name,
			Severity:     models.SeverityHigh,
			Observation:  fmt.Sprintf("Container has restarted %d times.", d.RestartCount),
			Evidence: []models.Evidence{
				{Field: "RestartCount", Value: fmt.Sprintf("%d", d.RestartCount)},
				{Field: "Status", Value: string(d.Status)},
			},
			PossibleCauses: []string{
				"Application crash on startup.",
				"Failing dependency.",
				"OOM or resource exhaustion.",
			},
			RecommendedVerify: []string{
				"Check exit code and OOMKilled.",
				"Review logs around restart times.",
			},
			SuggestedNextSteps: []string{
				"Open logs.",
				"Inspect resource limits and events.",
			},
			ConfidenceNote: "Restart count alone does not prove a continuous loop.",
		})
	}

	if d.Status == models.StatusExited && d.ExitCode != 0 {
		out = append(out, models.DiagnosticFinding{
			ID:           "DIAG-EXIT-NONZERO",
			ResourceID:   d.ID,
			ResourceName: d.Name,
			Severity:     models.SeverityMedium,
			Observation:  fmt.Sprintf("Container exited with code %d.", d.ExitCode),
			Evidence: []models.Evidence{
				{Field: "ExitCode", Value: fmt.Sprintf("%d", d.ExitCode)},
				{Field: "FinishedAt", Value: formatTime(d.FinishedAt)},
			},
			PossibleCauses: []string{
				"Process returned an error.",
				"Misconfigured command or entrypoint.",
			},
			RecommendedVerify: []string{
				"Inspect logs from the last run.",
				"Verify command and environment.",
			},
			SuggestedNextSteps: []string{"Open logs", "Inspect configuration"},
			ConfidenceNote:     "Exit code indicates failure but not the underlying cause.",
		})
	}

	if d.OOMKilled {
		out = append(out, models.DiagnosticFinding{
			ID:           "DIAG-OOM",
			ResourceID:   d.ID,
			ResourceName: d.Name,
			Severity:     models.SeverityHigh,
			Observation:  "Docker reports the container was OOM killed.",
			Evidence:     []models.Evidence{{Field: "State.OOMKilled", Value: "true"}},
			PossibleCauses: []string{
				"Memory limit too low.",
				"Memory leak or unexpected workload spike.",
			},
			RecommendedVerify: []string{
				"Check memory limit and usage history.",
				"Review application memory behavior.",
			},
			SuggestedNextSteps: []string{"Inspect memory settings", "Open logs"},
			ConfidenceNote:     "Reported directly by Docker State.OOMKilled.",
		})
	}

	if d.Status == models.StatusExited {
		out = append(out, models.DiagnosticFinding{
			ID:           "DIAG-EXITED",
			ResourceID:   d.ID,
			ResourceName: d.Name,
			Severity:     models.SeverityLow,
			Observation:  "Container is in exited state.",
			Evidence: []models.Evidence{
				{Field: "Status", Value: string(d.Status)},
				{Field: "StatusText", Value: d.StatusText},
			},
			PossibleCauses:     []string{"Intentionally stopped", "Crash or failed start"},
			RecommendedVerify:  []string{"Confirm whether the container should be running"},
			SuggestedNextSteps: []string{"Start container if expected", "Inspect logs if unexpected"},
			ConfidenceNote:     "Exited state is factual; intent is unknown.",
		})
	}

	return out
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "n/a"
	}
	return t.UTC().Format(time.RFC3339)
}

// FilterBySeverity keeps findings at or above a threshold.
func FilterBySeverity(items []models.DiagnosticFinding, min models.Severity) []models.DiagnosticFinding {
	rank := map[models.Severity]int{
		models.SeverityCritical: 5,
		models.SeverityHigh:     4,
		models.SeverityMedium:   3,
		models.SeverityLow:      2,
		models.SeverityInfo:     1,
		models.SeverityUnknown:  0,
	}
	out := make([]models.DiagnosticFinding, 0, len(items))
	for _, item := range items {
		if rank[item.Severity] >= rank[min] {
			out = append(out, item)
		}
	}
	return out
}

// Summarize returns a short one-line summary.
func Summarize(f models.DiagnosticFinding) string {
	return strings.ToUpper(string(f.Severity)) + "  " + f.ResourceName + "  " + f.Observation
}
