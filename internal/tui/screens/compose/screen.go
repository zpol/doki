package compose

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/zpol/doki/internal/domain/models"
	"github.com/zpol/doki/internal/tui/themes"
)

// Project aggregates containers that share a Compose project label.
type Project struct {
	Name       string
	Services   []string
	Containers []models.Container
	Running    int
	Stopped    int
	Unhealthy  int
}

// Screen lists Compose projects discovered from container labels.
type Screen struct {
	styles   themes.Styles
	projects []Project
	cursor   int
	width    int
	height   int
	status   string
}

func New(styles themes.Styles) Screen { return Screen{styles: styles} }

func (s Screen) SetStyles(styles themes.Styles) Screen { s.styles = styles; return s }
func (s Screen) SetSize(w, h int) Screen                { s.width = w; s.height = h; return s }
func (s Screen) SetStatus(msg string) Screen            { s.status = msg; return s }

func (s Screen) SetContainers(items []models.Container) Screen {
	byProject := map[string]*Project{}
	for _, c := range items {
		name := c.ComposeProject
		if name == "" {
			continue
		}
		p := byProject[name]
		if p == nil {
			p = &Project{Name: name}
			byProject[name] = p
		}
		p.Containers = append(p.Containers, c)
		if c.ComposeService != "" {
			found := false
			for _, svc := range p.Services {
				if svc == c.ComposeService {
					found = true
					break
				}
			}
			if !found {
				p.Services = append(p.Services, c.ComposeService)
			}
		}
		switch c.Status {
		case models.StatusRunning:
			p.Running++
		default:
			p.Stopped++
		}
		if c.Health == models.HealthUnhealthy {
			p.Unhealthy++
		}
	}
	out := make([]Project, 0, len(byProject))
	for _, p := range byProject {
		sort.Strings(p.Services)
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	s.projects = out
	if s.cursor >= len(out) && len(out) > 0 {
		s.cursor = len(out) - 1
	}
	return s
}

func (s Screen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "j", "down":
			if s.cursor < len(s.projects)-1 {
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
	b.WriteString(s.styles.Title.Render("COMPOSE") + "\n")
	b.WriteString(s.styles.Tagline.Render("Projects discovered from com.docker.compose.* labels on the connected Docker host.") + "\n\n")
	if s.status != "" {
		b.WriteString(s.styles.StatusWarn.Render(s.status) + "\n")
	}
	if len(s.projects) == 0 {
		b.WriteString(s.styles.StatusMuted.Render("No Compose projects detected via container labels.") + "\n")
		b.WriteString(s.styles.Tagline.Render("Remote hosts without local compose files can still show label-discovered projects.") + "\n")
	} else {
		b.WriteString(s.styles.TableHeader.Render(fmt.Sprintf("%-24s %-8s %-8s %-10s %-20s", "PROJECT", "RUN", "STOP", "UNHEALTHY", "SERVICES")) + "\n")
		for i, p := range s.projects {
			line := fmt.Sprintf("%-24s %-8d %-8d %-10d %-20s",
				trunc(p.Name, 24), p.Running, p.Stopped, p.Unhealthy, trunc(strings.Join(p.Services, ","), 20))
			if i == s.cursor {
				b.WriteString(s.styles.Selected.Width(max(s.width-4, 0)).Render(line) + "\n")
			} else {
				b.WriteString(s.styles.Normal.Render(line) + "\n")
			}
		}
		if s.cursor >= 0 && s.cursor < len(s.projects) {
			p := s.projects[s.cursor]
			b.WriteString("\n" + s.styles.TableHeader.Render("CONTAINERS IN "+p.Name) + "\n")
			for _, c := range p.Containers {
				b.WriteString(fmt.Sprintf("  %-16s %-10s %s\n", trunc(c.Name, 16), strings.ToUpper(string(c.Status)), trunc(c.Image, 28)))
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
