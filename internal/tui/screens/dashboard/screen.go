package dashboard

import (
	"fmt"
	"image/color"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/zpol/doki/internal/app/metrics"
	"github.com/zpol/doki/internal/domain/models"
	"github.com/zpol/doki/internal/tui/charts"
	"github.com/zpol/doki/internal/tui/themes"
)

// Screen renders the overview dashboard.
type Screen struct {
	styles     themes.Styles
	host       models.HostInfo
	containers []models.Container
	stats      []models.ContainerStats
	events     []models.Event
	cpuHist    map[string][]float64
	memHist    map[string][]float64
	agg        metrics.Aggregate
	aggCPU     []float64
	aggMem     []float64
	netRx      []float64
	netTx      []float64
	blkR       []float64
	blkW       []float64
	width      int
	height     int
}

func New(styles themes.Styles) Screen {
	return Screen{
		styles:  styles,
		cpuHist: map[string][]float64{},
		memHist: map[string][]float64{},
	}
}

func (s Screen) SetStyles(styles themes.Styles) Screen { s.styles = styles; return s }
func (s Screen) SetSize(w, h int) Screen                { s.width = w; s.height = h; return s }
func (s Screen) SetHost(host models.HostInfo) Screen    { s.host = host; return s }
func (s Screen) SetContainers(items []models.Container) Screen {
	s.containers = items
	return s
}
func (s Screen) SetStats(stats []models.ContainerStats, cpu, mem map[string][]float64) Screen {
	s.stats = stats
	s.cpuHist = cpu
	s.memHist = mem
	return s
}
func (s Screen) SetAggregate(agg metrics.Aggregate, cpu, mem, netRx, netTx, blkR, blkW []float64) Screen {
	s.agg = agg
	s.aggCPU = cpu
	s.aggMem = mem
	s.netRx = netRx
	s.netTx = netTx
	s.blkR = blkR
	s.blkW = blkW
	return s
}
func (s Screen) SetEvents(events []models.Event) Screen {
	s.events = events
	return s
}

func (s Screen) View() string {
	innerW := s.width - 4
	if innerW < 40 {
		innerW = 40
	}

	useSplit := innerW >= 90
	leftW := innerW
	rightW := 0
	if useSplit {
		rightW = innerW * 55 / 100
		if rightW < 42 {
			rightW = 42
		}
		leftW = innerW - rightW - 2
		if leftW < 36 {
			leftW = 36
			rightW = innerW - leftW - 2
		}
	}

	left := s.renderLeft(leftW)
	body := left
	if useSplit {
		body = charts.JoinHorizontal(left, s.renderRightGraphs(rightW), 2, leftW)
	} else {
		body = left + "\n\n" + s.renderRightGraphs(innerW)
	}

	panel := s.styles.Panel
	if s.width > 0 {
		panel = panel.Width(s.width)
	}
	if s.height > 0 {
		panel = panel.Height(s.height)
	}
	return panel.Render(body)
}

func (s Screen) renderLeft(width int) string {
	var b strings.Builder
	running, stopped, unhealthy, paused := 0, 0, 0, 0
	for _, c := range s.containers {
		switch c.Status {
		case models.StatusRunning:
			running++
		case models.StatusPaused:
			paused++
		case models.StatusExited, models.StatusDead, models.StatusCreated:
			stopped++
		}
		if c.Health == models.HealthUnhealthy {
			unhealthy++
		}
	}

	th := s.styles.Theme
	b.WriteString(s.styles.Title.Render("DASHBOARD") + "\n")
	b.WriteString(s.styles.Tagline.Render("Container-aggregated · not host CPU/RAM") + "\n\n")

	b.WriteString(fmt.Sprintf("%s %s  %s %s  %s %s  %s %s\n\n",
		s.styles.Tagline.Render("Run"), s.styles.StatusOK.Render(fmt.Sprintf("%d", running)),
		s.styles.Tagline.Render("Stop"), s.styles.StatusMuted.Render(fmt.Sprintf("%d", stopped)),
		s.styles.Tagline.Render("Pause"), s.styles.StatusWarn.Render(fmt.Sprintf("%d", paused)),
		s.styles.Tagline.Render("Sick"), s.styles.StatusErr.Render(fmt.Sprintf("%d", unhealthy)),
	))

	barW := 14
	if width > 50 {
		barW = 18
	}
	memPct := 0.0
	if s.agg.MemoryLimit > 0 {
		memPct = float64(s.agg.MemoryUsage) / float64(s.agg.MemoryLimit) * 100
	}

	b.WriteString(s.styles.TableHeader.Render("NOW") + "\n")
	b.WriteString(fmt.Sprintf("%s %s %5.1f%%\n",
		fg(th.Primary, "CPU"),
		fg(th.Primary, charts.Bar(clampPct(s.agg.CPUPercent), barW)),
		s.agg.CPUPercent,
	))
	b.WriteString(fmt.Sprintf("%s %s %5.1f%%  %s\n",
		fg(th.Purple, "MEM"),
		fg(th.Purple, charts.Bar(memPct, barW)),
		memPct,
		charts.FormatBytes(s.agg.MemoryUsage),
	))
	b.WriteString(fmt.Sprintf("%s RX %s   TX %s\n",
		fg(th.Success, "NET"),
		charts.FormatBytes(s.agg.NetRx),
		charts.FormatBytes(s.agg.NetTx),
	))
	b.WriteString(fmt.Sprintf("%s R  %s   W  %s\n",
		fg(th.Warning, "IO "),
		charts.FormatBytes(s.agg.BlockRead),
		charts.FormatBytes(s.agg.BlockWrite),
	))
	b.WriteString(fmt.Sprintf("PIDS %d  across %d sampled\n\n", s.agg.PIDs, s.agg.Samples))

	topCPU := append([]models.ContainerStats(nil), s.stats...)
	sort.Slice(topCPU, func(i, j int) bool { return topCPU[i].CPUPercent > topCPU[j].CPUPercent })
	topMem := append([]models.ContainerStats(nil), s.stats...)
	sort.Slice(topMem, func(i, j int) bool { return topMem[i].MemoryUsage > topMem[j].MemoryUsage })
	if len(topCPU) > 5 {
		topCPU = topCPU[:5]
	}
	if len(topMem) > 5 {
		topMem = topMem[:5]
	}

	b.WriteString(s.styles.TableHeader.Render("TOP CPU") + "\n")
	if len(topCPU) == 0 {
		b.WriteString(s.styles.StatusMuted.Render("No live stats yet.") + "\n")
	} else {
		for _, st := range topCPU {
			b.WriteString(fmt.Sprintf("%-16s %5.1f%% %s\n",
				truncate(statName(st), 16),
				st.CPUPercent,
				fg(th.Primary, charts.Bar(clampPct(st.CPUPercent), 10)),
			))
		}
	}

	b.WriteString("\n" + s.styles.TableHeader.Render("TOP MEMORY") + "\n")
	if len(topMem) == 0 {
		b.WriteString(s.styles.StatusMuted.Render("No live stats yet.") + "\n")
	} else {
		for _, st := range topMem {
			b.WriteString(fmt.Sprintf("%-16s %8s %s\n",
				truncate(statName(st), 16),
				charts.FormatBytes(st.MemoryUsage),
				fg(th.Purple, charts.Bar(st.MemoryPercentOrZero(), 10)),
			))
		}
	}

	b.WriteString("\n" + s.styles.TableHeader.Render("EVENTS") + "\n")
	if len(s.events) == 0 {
		b.WriteString(s.styles.StatusMuted.Render("Waiting for Docker events…") + "\n")
	} else {
		start := 0
		if len(s.events) > 5 {
			start = len(s.events) - 5
		}
		for _, ev := range s.events[start:] {
			b.WriteString(fmt.Sprintf("%s %s\n",
				s.styles.Tagline.Render(ev.Time.Local().Format("15:04:05")),
				truncate(ev.Message, max(12, width-10)),
			))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func (s Screen) renderRightGraphs(width int) string {
	th := s.styles.Theme
	graphW := width - 1
	if graphW < 20 {
		graphW = 20
	}
	graphH := 5
	if s.height >= 30 {
		graphH = 6
	}
	if s.height < 22 {
		graphH = 4
	}

	var b strings.Builder
	b.WriteString(s.styles.Title.Render("LIVE GRAPHS") + "\n")
	b.WriteString(s.styles.Tagline.Render("braille · btop-style") + "\n\n")

	b.WriteString(charts.FrameGraph(
		"CPU",
		fmt.Sprintf("%.1f%%", s.agg.CPUPercent),
		charts.BrailleGraph(s.aggCPU, charts.GraphOptions{
			Width: graphW, Height: graphH, Max: 100,
			Color: th.Primary, LowColor: th.Accent,
		}),
		s.styles.Brand,
		s.styles.StatusOK,
	))
	b.WriteString("\n\n")

	memPct := 0.0
	if s.agg.MemoryLimit > 0 {
		memPct = float64(s.agg.MemoryUsage) / float64(s.agg.MemoryLimit) * 100
	}
	b.WriteString(charts.FrameGraph(
		"MEM",
		fmt.Sprintf("%.1f%% %s", memPct, charts.FormatBytes(s.agg.MemoryUsage)),
		charts.BrailleGraph(s.aggMem, charts.GraphOptions{
			Width: graphW, Height: graphH, Max: 100,
			Color: th.Purple, LowColor: th.Accent,
		}),
		lipgloss.NewStyle().Foreground(th.Purple).Bold(true),
		s.styles.Normal,
	))
	b.WriteString("\n\n")

	netH := graphH
	if netH > 4 {
		netH = 4
	}
	b.WriteString(charts.FrameGraph(
		"NET RX",
		charts.FormatBytes(s.agg.NetRx)+" total",
		charts.BrailleGraph(s.netRx, charts.GraphOptions{
			Width: graphW, Height: netH, Max: 0,
			Color: th.Success, LowColor: th.Primary,
		}),
		lipgloss.NewStyle().Foreground(th.Success).Bold(true),
		s.styles.Tagline,
	))
	b.WriteString("\n\n")

	b.WriteString(charts.FrameGraph(
		"NET TX",
		charts.FormatBytes(s.agg.NetTx)+" total",
		charts.BrailleGraph(s.netTx, charts.GraphOptions{
			Width: graphW, Height: netH, Max: 0,
			Color: th.Warning, LowColor: th.Error,
		}),
		lipgloss.NewStyle().Foreground(th.Warning).Bold(true),
		s.styles.Tagline,
	))
	b.WriteString("\n\n")

	ioH := 3
	b.WriteString(charts.FrameGraph(
		"DISK W",
		charts.FormatBytes(s.agg.BlockWrite)+" total",
		charts.BrailleGraph(s.blkW, charts.GraphOptions{
			Width: graphW, Height: ioH, Max: 0,
			Color: th.Error, LowColor: th.Warning,
		}),
		lipgloss.NewStyle().Foreground(th.Error).Bold(true),
		s.styles.Tagline,
	))

	return strings.TrimRight(b.String(), "\n")
}

func fg(c color.Color, text string) string {
	return lipgloss.NewStyle().Foreground(c).Render(text)
}

func statName(st models.ContainerStats) string {
	if st.Name != "" {
		return strings.TrimPrefix(st.Name, "/")
	}
	if len(st.ContainerID) > 12 {
		return st.ContainerID[:12]
	}
	return st.ContainerID
}

func clampPct(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
