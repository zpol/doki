package containers

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/zpol/doki/internal/domain/models"
	"github.com/zpol/doki/internal/tui/themes"
)

// Screen is the container list view.
type Screen struct {
	styles     themes.Styles
	items      []models.Container
	cursor     int
	offset     int
	width      int
	height     int
	filterMode bool
	filter     string
	statusMsg  string
	marked     map[string]struct{}
}

// New creates a containers screen.
func New(styles themes.Styles) Screen {
	return Screen{styles: styles, marked: map[string]struct{}{}}
}

// SetStyles updates theme styles.
func (s Screen) SetStyles(styles themes.Styles) Screen {
	s.styles = styles
	return s
}

// SetSize updates available geometry.
func (s Screen) SetSize(width, height int) Screen {
	s.width = width
	s.height = height
	s = s.ensureVisible()
	return s
}

// SetItems replaces the container list while preserving selection when possible.
func (s Screen) SetItems(items []models.Container) Screen {
	var selectedID string
	if s.cursor >= 0 && s.cursor < len(s.items) {
		selectedID = s.items[s.cursor].ID
	}
	s.items = items
	s.cursor = 0
	if selectedID != "" {
		for i, item := range items {
			if item.ID == selectedID {
				s.cursor = i
				break
			}
		}
	}
	if s.cursor >= len(s.items) && len(s.items) > 0 {
		s.cursor = len(s.items) - 1
	}
	// Drop marks for containers that disappeared.
	if len(s.marked) > 0 {
		alive := make(map[string]struct{}, len(items))
		for _, item := range items {
			alive[item.ID] = struct{}{}
		}
		for id := range s.marked {
			if _, ok := alive[id]; !ok {
				delete(s.marked, id)
			}
		}
	}
	s = s.ensureVisible()
	return s
}

// SetStatus sets a transient status message.
func (s Screen) SetStatus(msg string) Screen {
	s.statusMsg = msg
	return s
}

// Selected returns the currently highlighted container, if any.
func (s Screen) Selected() (models.Container, bool) {
	if s.cursor < 0 || s.cursor >= len(s.items) {
		return models.Container{}, false
	}
	return s.items[s.cursor], true
}

// Marked returns containers toggled with space (bulk selection).
func (s Screen) Marked() []models.Container {
	if len(s.marked) == 0 {
		return nil
	}
	out := make([]models.Container, 0, len(s.marked))
	for _, item := range s.items {
		if _, ok := s.marked[item.ID]; ok {
			out = append(out, item)
		}
	}
	return out
}

// MarkedCount returns how many containers are marked.
func (s Screen) MarkedCount() int {
	return len(s.marked)
}

// Targets returns marked containers when any exist, otherwise the cursor row.
func (s Screen) Targets() []models.Container {
	if marked := s.Marked(); len(marked) > 0 {
		return marked
	}
	if c, ok := s.Selected(); ok {
		return []models.Container{c}
	}
	return nil
}

// ClearMarks removes all bulk marks.
func (s Screen) ClearMarks() Screen {
	s.marked = map[string]struct{}{}
	return s
}

// Filter returns the active filter query.
func (s Screen) Filter() string {
	return s.filter
}

// Filtering reports whether filter input is active.
func (s Screen) Filtering() bool {
	return s.filterMode
}

// Update handles list navigation and filter input.
func (s Screen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if s.filterMode {
			return s.updateFilter(msg)
		}
		switch msg.String() {
		case "j", "down":
			if s.cursor < len(s.items)-1 {
				s.cursor++
				s = s.ensureVisible()
			}
		case "k", "up":
			if s.cursor > 0 {
				s.cursor--
				s = s.ensureVisible()
			}
		case "g", "home":
			s.cursor = 0
			s = s.ensureVisible()
		case "G", "end":
			if len(s.items) > 0 {
				s.cursor = len(s.items) - 1
				s = s.ensureVisible()
			}
		case " ", "space":
			s = s.toggleMark()
		case "A":
			s = s.markAll()
		case "U":
			s = s.ClearMarks()
		case "/":
			s.filterMode = true
		case "esc":
			if len(s.marked) > 0 {
				s = s.ClearMarks()
				return s, nil
			}
			if s.filter != "" {
				s.filter = ""
				return s, filterChangedCmd("")
			}
		}
	}
	return s, nil
}

func (s Screen) toggleMark() Screen {
	c, ok := s.Selected()
	if !ok {
		return s
	}
	if s.marked == nil {
		s.marked = map[string]struct{}{}
	}
	if _, ok := s.marked[c.ID]; ok {
		delete(s.marked, c.ID)
	} else {
		s.marked[c.ID] = struct{}{}
	}
	return s
}

func (s Screen) markAll() Screen {
	if s.marked == nil {
		s.marked = map[string]struct{}{}
	}
	for _, item := range s.items {
		s.marked[item.ID] = struct{}{}
	}
	return s
}

func (s Screen) updateFilter(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		s.filterMode = false
		return s, nil
	case "enter":
		s.filterMode = false
		return s, filterChangedCmd(s.filter)
	case "backspace":
		if len(s.filter) > 0 {
			s.filter = s.filter[:len(s.filter)-1]
		}
		return s, filterChangedCmd(s.filter)
	default:
		if len(msg.String()) == 1 {
			s.filter += msg.String()
			return s, filterChangedCmd(s.filter)
		}
	}
	return s, nil
}

// FilterChangedMsg notifies the app that the filter query changed.
type FilterChangedMsg struct {
	Query string
}

func filterChangedCmd(query string) tea.Cmd {
	return func() tea.Msg {
		return FilterChangedMsg{Query: query}
	}
}

// View renders the container table.
func (s Screen) View() string {
	var b strings.Builder

	title := s.styles.Title.Render("CONTAINERS")
	count := s.styles.Tagline.Render(fmt.Sprintf("%d shown", len(s.items)))
	b.WriteString(title + "  " + count)
	if n := s.MarkedCount(); n > 0 {
		b.WriteString("  " + s.styles.StatusOK.Render(fmt.Sprintf("%d marked", n)))
	}
	if s.statusMsg != "" {
		b.WriteString("  " + s.styles.StatusWarn.Render(s.statusMsg))
	}
	b.WriteString("\n")

	if s.filterMode || s.filter != "" {
		mode := "filter"
		if s.filterMode {
			mode = "FILTER"
		}
		b.WriteString(s.styles.Filter.Render(fmt.Sprintf("%s: %s█", mode, s.filter)))
		b.WriteString("\n")
	}

	header := fmt.Sprintf("%-2s %-2s %-20s %-12s %-26s %-16s %-8s",
		"", "", "NAME", "STATUS", "IMAGE", "PORTS", "HEALTH")
	b.WriteString(s.styles.TableHeader.Render(header))
	b.WriteString("\n")

	if len(s.items) == 0 {
		b.WriteString(s.styles.StatusMuted.Render("No containers found."))
		return s.styles.Panel.Width(max(s.width, 0)).Height(max(s.height, 0)).Render(b.String())
	}

	visible := s.visibleRows()
	end := s.offset + visible
	if end > len(s.items) {
		end = len(s.items)
	}

	for i := s.offset; i < end; i++ {
		_, marked := s.marked[s.items[i].ID]
		row := s.renderRow(s.items[i], i == s.cursor, marked)
		b.WriteString(row)
		b.WriteString("\n")
	}

	return s.styles.Panel.Width(max(s.width, 0)).Height(max(s.height, 0)).Render(strings.TrimRight(b.String(), "\n"))
}

func (s Screen) renderRow(c models.Container, selected, marked bool) string {
	check := " "
	if marked {
		check = "✓"
	}
	dot := "○"
	switch c.Status {
	case models.StatusRunning:
		dot = "●"
	case models.StatusPaused, models.StatusRestarting:
		dot = "◐"
	case models.StatusExited, models.StatusDead:
		dot = "○"
	}

	status := strings.ToUpper(string(c.Status))
	health := strings.ToUpper(string(c.Health))
	if c.Health == models.HealthNone {
		health = "-"
	}

	plain := fmt.Sprintf("%s %s %-20s %-12s %-26s %-16s %-8s",
		check,
		dot,
		truncate(c.Name, 20),
		truncate(status, 12),
		truncate(c.Image, 26),
		truncate(formatPorts(c.Ports), 16),
		truncate(health, 8),
	)
	width := max(s.width-4, len(plain))
	plain = padRight(plain, width)

	// Selected row: one solid style over the full line. Nested ANSI
	// (status colors) would punch holes in the highlight background.
	if selected {
		return s.styles.Selected.Render(plain)
	}
	if marked {
		return s.styles.Normal.
			Foreground(s.styles.Theme.Primary).
			Bold(true).
			Render(plain)
	}

	indicator := s.styles.StatusMuted.Render(dot)
	switch c.Status {
	case models.StatusRunning:
		indicator = s.styles.StatusOK.Render(dot)
	case models.StatusPaused, models.StatusRestarting:
		indicator = s.styles.StatusWarn.Render(dot)
	case models.StatusExited, models.StatusDead:
		indicator = s.styles.StatusErr.Render(dot)
	}
	rest := fmt.Sprintf(" %-20s %-12s %-26s %-16s %-8s",
		truncate(c.Name, 20),
		truncate(status, 12),
		truncate(c.Image, 26),
		truncate(formatPorts(c.Ports), 16),
		truncate(health, 8),
	)
	return s.styles.Normal.Render("  " + indicator + rest)
}

func padRight(s string, width int) string {
	// Approximate visible width; mark/dot are single-width runes we use.
	n := len([]rune(s))
	if n >= width {
		return string([]rune(s)[:width])
	}
	return s + strings.Repeat(" ", width-n)
}

func (s Screen) visibleRows() int {
	// title + optional filter + header + padding
	rows := s.height - 6
	if rows < 3 {
		rows = 3
	}
	return rows
}

func (s Screen) ensureVisible() Screen {
	visible := s.visibleRows()
	if s.cursor < s.offset {
		s.offset = s.cursor
	}
	if s.cursor >= s.offset+visible {
		s.offset = s.cursor - visible + 1
	}
	if s.offset < 0 {
		s.offset = 0
	}
	return s
}

func formatPorts(ports []models.PortBinding) string {
	if len(ports) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(ports))
	for _, p := range ports {
		if p.PublicPort == 0 {
			parts = append(parts, fmt.Sprintf("%d/%s", p.PrivatePort, p.Type))
			continue
		}
		parts = append(parts, fmt.Sprintf("%d:%d", p.PublicPort, p.PrivatePort))
	}
	return strings.Join(parts, ",")
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
