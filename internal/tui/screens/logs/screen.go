package logs

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/zpol/doki/internal/domain/models"
	"github.com/zpol/doki/internal/tui/themes"
)

const maxLines = 2000

// Screen is a live log viewer.
type Screen struct {
	styles     themes.Styles
	name       string
	lines      []models.LogLine
	offset     int
	width      int
	height     int
	follow     bool
	paused     bool
	showTime   bool
	searchMode bool
	search     string
	status     string
}

// New creates a logs screen.
func New(styles themes.Styles) Screen {
	return Screen{styles: styles, follow: true, showTime: true}
}

func (s Screen) SetStyles(styles themes.Styles) Screen { s.styles = styles; return s }
func (s Screen) SetSize(w, h int) Screen                { s.width = w; s.height = h; return s }
func (s Screen) SetTarget(name string) Screen {
	s.name = name
	s.lines = nil
	s.offset = 0
	s.follow = true
	s.paused = false
	s.status = ""
	return s
}

// Append adds log lines respecting pause/follow and bound.
func (s Screen) Append(lines ...models.LogLine) Screen {
	if s.paused {
		return s
	}
	s.lines = append(s.lines, lines...)
	if len(s.lines) > maxLines {
		s.lines = s.lines[len(s.lines)-maxLines:]
	}
	if s.follow {
		s.offset = s.maxOffset()
	}
	return s
}

func (s Screen) SetStatus(msg string) Screen { s.status = msg; return s }

// Update handles log viewer keys.
func (s Screen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if s.searchMode {
			return s.updateSearch(msg)
		}
		switch msg.String() {
		case "j", "down":
			s.follow = false
			if s.offset < s.maxOffset() {
				s.offset++
			}
		case "k", "up":
			s.follow = false
			if s.offset > 0 {
				s.offset--
			}
		case "g", "home":
			s.follow = false
			s.offset = 0
		case "G", "end":
			s.follow = true
			s.offset = s.maxOffset()
		case "f":
			s.follow = !s.follow
			if s.follow {
				s.offset = s.maxOffset()
			}
		case "p":
			s.paused = !s.paused
		case "t":
			s.showTime = !s.showTime
		case "/":
			s.searchMode = true
		}
	}
	return s, nil
}

func (s Screen) updateSearch(msg tea.KeyPressMsg) (Screen, tea.Cmd) {
	switch msg.String() {
	case "esc":
		s.searchMode = false
		s.search = ""
	case "enter":
		s.searchMode = false
	case "backspace":
		if len(s.search) > 0 {
			s.search = s.search[:len(s.search)-1]
		}
	default:
		if len(msg.String()) == 1 {
			s.search += msg.String()
		}
	}
	return s, nil
}

// View renders logs.
func (s Screen) View() string {
	var b strings.Builder
	title := fmt.Sprintf("LOGS: %s", s.name)
	flags := []string{}
	if s.follow {
		flags = append(flags, "FOLLOW")
	}
	if s.paused {
		flags = append(flags, "PAUSED")
	}
	b.WriteString(s.styles.Title.Render(title))
	if len(flags) > 0 {
		b.WriteString("  " + s.styles.StatusWarn.Render(strings.Join(flags, " ")))
	}
	if s.status != "" {
		b.WriteString("  " + s.styles.StatusErr.Render(s.status))
	}
	b.WriteString("\n")
	if s.searchMode || s.search != "" {
		b.WriteString(s.styles.Filter.Render("search: "+s.search+"█") + "\n")
	}

	visible := s.visibleRows()
	filtered := s.filtered()
	end := s.offset + visible
	if end > len(filtered) {
		end = len(filtered)
	}
	if len(filtered) == 0 {
		b.WriteString(s.styles.StatusMuted.Render("No log lines."))
	} else {
		for _, line := range filtered[s.offset:end] {
			b.WriteString(s.renderLine(line))
			b.WriteString("\n")
		}
	}
	b.WriteString(s.styles.Help.Render("[/] Search  [f] Follow  [p] Pause  [t] Timestamp  [esc] Back"))
	return s.styles.Panel.Width(max(s.width, 0)).Height(max(s.height, 0)).Render(strings.TrimRight(b.String(), "\n"))
}

func (s Screen) renderLine(line models.LogLine) string {
	sev := models.DetectSeverity(line.Message)
	prefix := s.styles.StatusMuted.Render("INFO ")
	switch sev {
	case models.LogSeverityError:
		prefix = s.styles.StatusErr.Render("ERROR")
	case models.LogSeverityWarn:
		prefix = s.styles.StatusWarn.Render("WARN ")
	case models.LogSeverityDebug:
		prefix = s.styles.Tagline.Render("DEBUG")
	case models.LogSeverityInfo:
		prefix = s.styles.StatusOK.Render("INFO ")
	}
	ts := ""
	if s.showTime {
		t := line.Time
		if t.IsZero() {
			t = time.Now()
		}
		ts = s.styles.Tagline.Render(t.Local().Format("15:04:05")) + " "
	}
	return ts + prefix + " " + s.styles.Normal.Render(line.Message)
}

func (s Screen) filtered() []models.LogLine {
	if s.search == "" {
		return s.lines
	}
	q := strings.ToLower(s.search)
	out := make([]models.LogLine, 0, len(s.lines))
	for _, line := range s.lines {
		if strings.Contains(strings.ToLower(line.Message), q) {
			out = append(out, line)
		}
	}
	return out
}

func (s Screen) visibleRows() int {
	n := s.height - 5
	if n < 3 {
		n = 3
	}
	return n
}

func (s Screen) maxOffset() int {
	n := len(s.filtered()) - s.visibleRows()
	if n < 0 {
		return 0
	}
	return n
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
