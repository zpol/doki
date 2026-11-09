package inspect

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zpol/doki/internal/domain/models"
	"github.com/zpol/doki/internal/tui/charts"
	"github.com/zpol/doki/internal/tui/themes"
)

// Screen shows detailed container inspection.
type Screen struct {
	styles themes.Styles
	detail models.ContainerDetails
	stats  models.ContainerStats
	cpuHist []float64
	memHist []float64
	offset int
	width  int
	height int
	ready  bool
	err    string
}

// New creates an inspect screen.
func New(styles themes.Styles) Screen {
	return Screen{styles: styles}
}

// SetStyles updates theme styles.
func (s Screen) SetStyles(styles themes.Styles) Screen {
	s.styles = styles
	return s
}

// SetSize updates geometry.
func (s Screen) SetSize(width, height int) Screen {
	s.width = width
	s.height = height
	return s
}

// SetDetail loads inspect data.
func (s Screen) SetDetail(detail models.ContainerDetails) Screen {
	s.detail = detail
	s.ready = true
	s.err = ""
	s.offset = 0
	return s
}

// SetStats updates live resource samples and histories.
func (s Screen) SetStats(stats models.ContainerStats, cpu, mem []float64) Screen {
	s.stats = stats
	s.cpuHist = cpu
	s.memHist = mem
	return s
}

// SetError records an inspect failure.
func (s Screen) SetError(err string) Screen {
	s.err = err
	s.ready = false
	return s
}

// Clear resets content for a new inspect target while preserving terminal size.
func (s Screen) Clear() Screen {
	s.detail = models.ContainerDetails{}
	s.stats = models.ContainerStats{}
	s.cpuHist = nil
	s.memHist = nil
	s.ready = false
	s.err = ""
	s.offset = 0
	return s
}

// ContainerID returns the inspected container id when ready.
func (s Screen) ContainerID() string {
	if !s.ready {
		return ""
	}
	return s.detail.ID
}

// Update handles scrolling.
func (s Screen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		lines := strings.Split(s.content(), "\n")
		maxOffset := len(lines) - s.visibleLines()
		if maxOffset < 0 {
			maxOffset = 0
		}
		switch msg.String() {
		case "j", "down":
			if s.offset < maxOffset {
				s.offset++
			}
		case "k", "up":
			if s.offset > 0 {
				s.offset--
			}
		case "g", "home":
			s.offset = 0
		case "G", "end":
			s.offset = maxOffset
		}
	}
	return s, nil
}

// View renders inspect details.
func (s Screen) View() string {
	body := s.content()
	lines := strings.Split(body, "\n")
	visible := s.visibleLines()
	end := s.offset + visible
	if end > len(lines) {
		end = len(lines)
	}
	if s.offset > len(lines) {
		s.offset = 0
	}
	chunk := strings.Join(lines[s.offset:end], "\n")
	panel := s.styles.Panel
	if s.width > 0 {
		panel = panel.Width(s.width)
	}
	if s.height > 0 {
		panel = panel.Height(s.height)
	}
	return panel.Render(chunk)
}

func (s Screen) visibleLines() int {
	n := s.height - 2
	if n < 5 {
		n = 5
	}
	return n
}

func (s Screen) content() string {
	if s.err != "" {
		return s.styles.StatusErr.Render("Inspect failed: "+s.err) +
			"\n\n" + s.styles.Help.Render("[esc] Back")
	}
	if !s.ready {
		return s.styles.StatusMuted.Render("Loading container details…")
	}

	innerW := s.width - 4
	if innerW < 40 {
		innerW = 40
	}
	useSplit := innerW >= 96
	leftW := innerW
	rightW := 0
	if useSplit {
		rightW = innerW * 48 / 100
		if rightW < 40 {
			rightW = 40
		}
		leftW = innerW - rightW - 2
	}

	left := s.renderDetails()
	if useSplit {
		return charts.JoinHorizontal(left, s.renderGraphs(rightW), 2, leftW)
	}
	return s.renderGraphs(innerW) + "\n\n" + left
}

func (s Screen) renderDetails() string {
	d := s.detail
	var b strings.Builder
	b.WriteString(s.styles.Title.Render("INSPECT: "+d.Name) + "\n\n")

	b.WriteString(s.renderResourceMeters())
	b.WriteString("\n")

	b.WriteString(kv(s, "ID", d.ID))
	b.WriteString(kv(s, "Image", d.Image))
	b.WriteString(kv(s, "Status", strings.ToUpper(string(d.Status))+" — "+d.StatusText))
	b.WriteString(kv(s, "Health", string(d.Health)))
	b.WriteString(kv(s, "Created", formatTime(d.CreatedAt)))
	b.WriteString(kv(s, "Started", formatTime(d.StartedAt)))
	b.WriteString(kv(s, "Restarts", fmt.Sprintf("%d", d.RestartCount)))
	b.WriteString(kv(s, "Exit Code", fmt.Sprintf("%d", d.ExitCode)))
	b.WriteString(kv(s, "OOM Killed", fmt.Sprintf("%v", d.OOMKilled)))
	b.WriteString(kv(s, "User", empty(d.User, "n/a")))
	b.WriteString(kv(s, "Platform", empty(d.Platform, "n/a")))
	b.WriteString(kv(s, "Network Mode", empty(d.NetworkMode, "n/a")))
	b.WriteString(kv(s, "Privileged", fmt.Sprintf("%v", d.Privileged)))
	b.WriteString(kv(s, "Restart Policy", empty(d.RestartPolicy, "n/a")))
	b.WriteString(kv(s, "Working Dir", empty(d.WorkingDir, "/")))
	b.WriteString(kv(s, "Compose Project", empty(d.ComposeProject, "-")))
	b.WriteString(kv(s, "Compose Service", empty(d.ComposeService, "-")))

	b.WriteString("\n" + s.styles.TableHeader.Render("COMMAND") + "\n")
	b.WriteString(s.styles.Normal.Render(strings.Join(append([]string{d.Path}, d.Args...), " ")) + "\n")

	b.WriteString("\n" + s.styles.TableHeader.Render("PORTS") + "\n")
	if len(d.Ports) == 0 {
		b.WriteString(s.styles.StatusMuted.Render("(none)") + "\n")
	} else {
		for _, p := range d.Ports {
			proto := p.Type
			if proto == "" {
				proto = "tcp"
			}
			if p.PublicPort == 0 {
				b.WriteString(fmt.Sprintf("  %d/%s (exposed, not published)\n", p.PrivatePort, proto))
				continue
			}
			b.WriteString(fmt.Sprintf("  %s:%d -> %d/%s\n", empty(p.IP, "0.0.0.0"), p.PublicPort, p.PrivatePort, proto))
		}
	}

	b.WriteString("\n" + s.styles.TableHeader.Render("MOUNTS") + "\n")
	if len(d.Mounts) == 0 {
		b.WriteString(s.styles.StatusMuted.Render("(none)") + "\n")
	} else {
		for _, m := range d.Mounts {
			rw := "ro"
			if m.RW {
				rw = "rw"
			}
			b.WriteString(fmt.Sprintf("  [%s] %s -> %s (%s)\n", m.Type, m.Source, m.Destination, rw))
		}
	}

	b.WriteString("\n" + s.styles.TableHeader.Render("NETWORKS") + "\n")
	if len(d.Networks) == 0 {
		b.WriteString(s.styles.StatusMuted.Render("(none)") + "\n")
	} else {
		for _, n := range d.Networks {
			b.WriteString(fmt.Sprintf("  %s  ip=%s  gw=%s\n", n.NetworkName, empty(n.IPAddress, "-"), empty(n.Gateway, "-")))
		}
	}

	b.WriteString("\n" + s.styles.TableHeader.Render("ENVIRONMENT") + "\n")
	b.WriteString(s.styles.StatusMuted.Render("Secrets are redacted by default.") + "\n")
	if len(d.Env) == 0 {
		b.WriteString(s.styles.StatusMuted.Render("(none)") + "\n")
	} else {
		for _, env := range d.Env {
			b.WriteString("  " + redactEnv(env) + "\n")
		}
	}

	b.WriteString("\n" + s.styles.Help.Render("[esc] Back"))
	return strings.TrimRight(b.String(), "\n")
}

func (s Screen) renderResourceMeters() string {
	th := s.styles.Theme
	var b strings.Builder
	b.WriteString(s.styles.TableHeader.Render("LIVE RESOURCES") + "\n")
	if !s.stats.Available && s.detail.Status != models.StatusRunning {
		b.WriteString(s.styles.StatusMuted.Render("Stats unavailable (container not running).") + "\n")
		return b.String()
	}
	if !s.stats.Available {
		b.WriteString(s.styles.StatusMuted.Render("Collecting stats…") + "\n")
		return b.String()
	}

	barW := 16
	b.WriteString(fmt.Sprintf("%s %s %5.1f%%\n",
		lipgloss.NewStyle().Foreground(th.Primary).Render("CPU"),
		lipgloss.NewStyle().Foreground(th.Primary).Render(charts.Bar(s.stats.CPUPercent, barW)),
		s.stats.CPUPercent,
	))
	b.WriteString(fmt.Sprintf("%s %s %5.1f%%  %s",
		lipgloss.NewStyle().Foreground(th.Purple).Render("MEM"),
		lipgloss.NewStyle().Foreground(th.Purple).Render(charts.Bar(s.stats.MemoryPercentOrZero(), barW)),
		s.stats.MemoryPercentOrZero(),
		charts.FormatBytes(s.stats.MemoryUsage),
	))
	if s.stats.MemoryLimit > 0 {
		b.WriteString(" / " + charts.FormatBytes(s.stats.MemoryLimit))
	}
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s RX %s  TX %s\n",
		lipgloss.NewStyle().Foreground(th.Success).Render("NET"),
		charts.FormatBytes(s.stats.NetRx),
		charts.FormatBytes(s.stats.NetTx),
	))
	b.WriteString(fmt.Sprintf("%s R  %s  W  %s\n",
		lipgloss.NewStyle().Foreground(th.Warning).Render("DISK"),
		charts.FormatBytes(s.stats.BlockRead),
		charts.FormatBytes(s.stats.BlockWrite),
	))
	b.WriteString(fmt.Sprintf("PIDS %d\n", s.stats.PIDs))
	return b.String()
}

func (s Screen) renderGraphs(width int) string {
	th := s.styles.Theme
	graphW := width - 1
	if graphW < 24 {
		graphW = 24
	}
	graphH := 7
	if s.height < 24 {
		graphH = 5
	}

	var b strings.Builder
	b.WriteString(s.styles.Title.Render("HISTORY") + "\n")
	b.WriteString(s.styles.Tagline.Render("braille · live") + "\n\n")

	if !s.stats.Available && len(s.cpuHist) == 0 {
		b.WriteString(s.styles.StatusMuted.Render("Graphs appear once stats arrive."))
		return b.String()
	}

	b.WriteString(charts.FrameGraph(
		"CPU",
		fmt.Sprintf("%.1f%%", s.stats.CPUPercent),
		charts.BrailleGraph(s.cpuHist, charts.GraphOptions{
			Width: graphW, Height: graphH, Max: 100,
			Color: th.Primary, LowColor: th.Accent,
		}),
		lipgloss.NewStyle().Foreground(th.Primary).Bold(true),
		s.styles.StatusOK,
	))
	b.WriteString("\n\n")

	b.WriteString(charts.FrameGraph(
		"MEM",
		fmt.Sprintf("%.1f%% %s", s.stats.MemoryPercentOrZero(), charts.FormatBytes(s.stats.MemoryUsage)),
		charts.BrailleGraph(s.memHist, charts.GraphOptions{
			Width: graphW, Height: graphH, Max: 100,
			Color: th.Purple, LowColor: th.Accent,
		}),
		lipgloss.NewStyle().Foreground(th.Purple).Bold(true),
		s.styles.Normal,
	))
	return strings.TrimRight(b.String(), "\n")
}

func kv(s Screen, key, value string) string {
	return fmt.Sprintf("%s  %s\n",
		s.styles.Tagline.Render(fmt.Sprintf("%-16s", key)),
		s.styles.Normal.Render(value),
	)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "n/a"
	}
	return t.Local().Format(time.RFC3339)
}

func empty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func redactEnv(env string) string {
	parts := strings.SplitN(env, "=", 2)
	if len(parts) != 2 {
		return env
	}
	key := parts[0]
	lower := strings.ToLower(key)
	sensitive := []string{"password", "secret", "token", "key", "credential", "auth", "api_key", "apikey"}
	for _, s := range sensitive {
		if strings.Contains(lower, s) {
			return key + "=********"
		}
	}
	return env
}
