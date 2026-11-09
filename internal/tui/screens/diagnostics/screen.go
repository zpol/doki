package diagnostics

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	diag "github.com/zpol/doki/internal/app/diagnostics"
	"github.com/zpol/doki/internal/domain/models"
	"github.com/zpol/doki/internal/tui/themes"
)

// Screen shows diagnostic findings.
type Screen struct {
	styles   themes.Styles
	findings []models.DiagnosticFinding
	cursor   int
	width    int
	height   int
	status   string
}

func New(styles themes.Styles) Screen { return Screen{styles: styles} }

func (s Screen) SetStyles(styles themes.Styles) Screen { s.styles = styles; return s }
func (s Screen) SetSize(w, h int) Screen                { s.width = w; s.height = h; return s }
func (s Screen) SetFindings(items []models.DiagnosticFinding) Screen {
	s.findings = items
	if s.cursor >= len(items) && len(items) > 0 {
		s.cursor = len(items) - 1
	}
	return s
}
func (s Screen) SetStatus(msg string) Screen { s.status = msg; return s }

func (s Screen) Selected() (models.DiagnosticFinding, bool) {
	if s.cursor < 0 || s.cursor >= len(s.findings) {
		return models.DiagnosticFinding{}, false
	}
	return s.findings[s.cursor], true
}

func (s Screen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "j", "down":
			if s.cursor < len(s.findings)-1 {
				s.cursor++
			}
		case "k", "up":
			if s.cursor > 0 {
				s.cursor--
			}
		}
	}
	return s, nil
}

func (s Screen) View() string {
	var b strings.Builder
	b.WriteString(s.styles.Title.Render("DIAGNOSTICS") + "\n")
	b.WriteString(s.styles.Tagline.Render("Explainable operational findings from Docker state. Speculative causes are labeled as possible, not confirmed.") + "\n\n")
	if s.status != "" {
		b.WriteString(s.styles.StatusWarn.Render(s.status) + "\n")
	}
	if len(s.findings) == 0 {
		b.WriteString(s.styles.StatusMuted.Render("No diagnostic findings right now.") + "\n")
	} else {
		for i, f := range s.findings {
			line := diag.Summarize(f)
			if i == s.cursor {
				b.WriteString(s.styles.Selected.Width(max(s.width-4, 0)).Render(trunc(line, max(s.width-6, 20))) + "\n")
			} else {
				b.WriteString(s.styles.Normal.Render(trunc(line, max(s.width-6, 20))) + "\n")
			}
		}
		if f, ok := s.Selected(); ok {
			b.WriteString("\n" + s.styles.TableHeader.Render("DETAILS") + "\n")
			b.WriteString(f.Observation + "\n")
			if f.ConfidenceNote != "" {
				b.WriteString(s.styles.Tagline.Render("Note: "+f.ConfidenceNote) + "\n")
			}
			if len(f.PossibleCauses) > 0 {
				b.WriteString("\nPossible causes:\n")
				for _, c := range f.PossibleCauses {
					b.WriteString("  - " + c + "\n")
				}
			}
			if len(f.SuggestedNextSteps) > 0 {
				b.WriteString("\nNext steps:\n")
				for i, step := range f.SuggestedNextSteps {
					b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, step))
				}
			}
		}
	}
	return s.styles.Panel.Width(max(s.width, 0)).Height(max(s.height, 0)).Render(strings.TrimRight(b.String(), "\n"))
}

func trunc(v string, n int) string {
	if len(v) <= n {
		return v
	}
	return v[:n-1] + "…"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
