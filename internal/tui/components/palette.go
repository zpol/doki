package components

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/zpol/doki/internal/tui/themes"
)

// PaletteCommand is emitted when the user submits a command palette entry.
type PaletteCommand struct {
	Raw  string
	Name string
	Args []string
}

// Palette is a K9s-inspired command palette.
type Palette struct {
	styles  themes.Styles
	active  bool
	input   string
	history []string
	width   int
}

var knownCommands = []string{
	"dashboard", "containers", "images", "compose", "volumes", "networks",
	"security", "diagnostics", "contexts", "events", "theme", "refresh",
	"help", "quit",
}

// NewPalette creates a command palette.
func NewPalette(styles themes.Styles) Palette {
	return Palette{styles: styles}
}

// Active reports whether the palette is open.
func (p Palette) Active() bool { return p.active }

// SetStyles updates styles.
func (p Palette) SetStyles(styles themes.Styles) Palette {
	p.styles = styles
	return p
}

// SetSize updates width.
func (p Palette) SetSize(width int) Palette {
	p.width = width
	return p
}

// Open activates the palette.
func (p Palette) Open() Palette {
	p.active = true
	p.input = ""
	return p
}

// Close deactivates the palette.
func (p Palette) Close() Palette {
	p.active = false
	p.input = ""
	return p
}

// Update handles palette key input.
func (p Palette) Update(msg tea.Msg) (Palette, tea.Cmd) {
	if !p.active {
		return p, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			p = p.Close()
			return p, nil
		case "enter":
			raw := strings.TrimSpace(p.input)
			p.history = append(p.history, raw)
			p = p.Close()
			name, args := splitCommand(raw)
			return p, func() tea.Msg {
				return PaletteCommand{Raw: raw, Name: name, Args: args}
			}
		case "tab":
			p.input = autocomplete(p.input)
		case "backspace":
			if len(p.input) > 0 {
				p.input = p.input[:len(p.input)-1]
			}
		default:
			if len(msg.String()) == 1 {
				p.input += msg.String()
			}
		}
	}
	return p, nil
}

// View renders the palette line.
func (p Palette) View() string {
	if !p.active {
		return ""
	}
	hint := autocomplete(p.input)
	line := ":" + p.input + "█"
	if hint != "" && hint != p.input {
		line += p.styles.StatusMuted.Render("  → " + hint)
	}
	return p.styles.Panel.Width(maxInt(p.width, 0)).Render(
		p.styles.Filter.Render("COMMAND") + "\n" + line,
	)
}

func splitCommand(raw string) (string, []string) {
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return "", nil
	}
	return strings.ToLower(fields[0]), fields[1:]
}

func autocomplete(input string) string {
	in := strings.ToLower(strings.TrimSpace(input))
	if in == "" {
		return ""
	}
	for _, cmd := range knownCommands {
		if strings.HasPrefix(cmd, in) {
			return cmd
		}
	}
	return in
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
