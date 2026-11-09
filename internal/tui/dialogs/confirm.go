package dialogs

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/zpol/doki/internal/tui/themes"
)

// ConfirmResult is emitted when a confirmation dialog closes.
type ConfirmResult struct {
	ID      string
	Confirmed bool
}

// Confirm is a destructive-action confirmation overlay.
type Confirm struct {
	ID      string
	Title   string
	Message string
	Active  bool
	styles  themes.Styles
	width   int
}

// NewConfirm creates a confirmation dialog.
func NewConfirm(styles themes.Styles) Confirm {
	return Confirm{styles: styles}
}

// Show activates the dialog.
func (c Confirm) Show(id, title, message string) Confirm {
	c.ID = id
	c.Title = title
	c.Message = message
	c.Active = true
	return c
}

// Hide deactivates the dialog.
func (c Confirm) Hide() Confirm {
	c.Active = false
	c.ID = ""
	c.Title = ""
	c.Message = ""
	return c
}

// SetSize updates rendering width.
func (c Confirm) SetSize(width int) Confirm {
	c.width = width
	return c
}

// SetStyles updates theme styles.
func (c Confirm) SetStyles(styles themes.Styles) Confirm {
	c.styles = styles
	return c
}

// Update handles confirmation key events.
func (c Confirm) Update(msg tea.Msg) (Confirm, tea.Cmd) {
	if !c.Active {
		return c, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "y", "Y", "enter":
			id := c.ID
			c = c.Hide()
			return c, func() tea.Msg {
				return ConfirmResult{ID: id, Confirmed: true}
			}
		case "n", "N", "esc", "q":
			id := c.ID
			c = c.Hide()
			return c, func() tea.Msg {
				return ConfirmResult{ID: id, Confirmed: false}
			}
		}
	}
	return c, nil
}

// View renders the confirmation dialog.
func (c Confirm) View() string {
	if !c.Active {
		return ""
	}

	width := c.width
	if width < 40 {
		width = 60
	}
	inner := width - 8
	if inner < 20 {
		inner = 20
	}

	body := fmt.Sprintf("%s\n\n%s\n\n%s",
		c.styles.Title.Render(c.Title),
		wrap(c.Message, inner),
		c.styles.Help.Render("[y] Confirm   [n] Cancel"),
	)

	return c.styles.Panel.
		Width(inner + 4).
		Render(body)
}

func wrap(s string, width int) string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return s
	}
	var lines []string
	var line string
	for _, w := range words {
		if line == "" {
			line = w
			continue
		}
		if len(line)+1+len(w) > width {
			lines = append(lines, line)
			line = w
			continue
		}
		line += " " + w
	}
	if line != "" {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
