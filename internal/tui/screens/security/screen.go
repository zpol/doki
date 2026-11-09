package security

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/zpol/doki/internal/app/security"
	"github.com/zpol/doki/internal/domain/models"
	"github.com/zpol/doki/internal/tui/themes"
)

// Screen shows security findings.
type Screen struct {
	styles   themes.Styles
	findings []models.Finding
	cursor   int
	width    int
	height   int
	status   string
}

func New(styles themes.Styles) Screen { return Screen{styles: styles} }

func (s Screen) SetStyles(styles themes.Styles) Screen { s.styles = styles; return s }
func (s Screen) SetSize(w, h int) Screen                { s.width = w; s.height = h; return s }
func (s Screen) SetFindings(items []models.Finding) Screen {
	s.findings = items
	if s.cursor >= len(items) && len(items) > 0 {
		s.cursor = len(items) - 1
	}
	return s
}
func (s Screen) SetStatus(msg string) Screen { s.status = msg; return s }

func (s Screen) Selected() (models.Finding, bool) {
	if s.cursor < 0 || s.cursor >= len(s.findings) {
		return models.Finding{}, false
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
	b.WriteString(s.styles.Title.Render("SECURITY INSPECTOR") + "\n")
	b.WriteString(s.styles.Tagline.Render(security.Disclaimer()) + "\n\n")
	if s.status != "" {
		b.WriteString(s.styles.StatusWarn.Render(s.status) + "\n")
	}
	if len(s.findings) == 0 {
		b.WriteString(s.styles.StatusMuted.Render("No configuration findings detected for scanned containers.") + "\n")
		b.WriteString(s.styles.Tagline.Render("Remember: no findings ≠ safe.") + "\n")
	} else {
		for i, f := range s.findings {
			line := fmt.Sprintf("%-8s  %-18s  %s", strings.ToUpper(string(f.Severity)), trunc(f.ResourceName, 18), f.Title)
			if i == s.cursor {
				b.WriteString(s.styles.Selected.Width(max(s.width-4, 0)).Render(line) + "\n")
			} else {
				b.WriteString(colorSeverity(s, f.Severity).Render(line) + "\n")
			}
		}
		if f, ok := s.Selected(); ok {
			b.WriteString("\n" + s.styles.TableHeader.Render("DETAILS") + "\n")
			b.WriteString(f.Description + "\n")
			b.WriteString(s.styles.Tagline.Render("Recommendation: ") + f.Recommendation + "\n")
		}
	}
	return s.styles.Panel.Width(max(s.width, 0)).Height(max(s.height, 0)).Render(strings.TrimRight(b.String(), "\n"))
}

func colorSeverity(s Screen, sev models.Severity) interface{ Render(...string) string } {
	switch sev {
	case models.SeverityCritical, models.SeverityHigh:
		return s.styles.StatusErr
	case models.SeverityMedium:
		return s.styles.StatusWarn
	case models.SeverityLow, models.SeverityInfo:
		return s.styles.Tagline
	default:
		return s.styles.Normal
	}
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
