package components

import (
	"fmt"
	"strings"

	"github.com/zpol/doki/internal/domain/models"
	"github.com/zpol/doki/internal/tui/themes"
)

// Header renders the top status bar.
type Header struct {
	styles  themes.Styles
	host    models.HostInfo
	width   int
	view    string
	errText string
}

// NewHeader creates a header component.
func NewHeader(styles themes.Styles) Header {
	return Header{styles: styles, view: "CONTAINERS"}
}

// SetStyles updates theme styles.
func (h Header) SetStyles(styles themes.Styles) Header {
	h.styles = styles
	return h
}

// Styles returns the current styles (for callers that need restyling).
func (h Header) Styles() themes.Styles { return h.styles }

// SetSize updates width.
func (h Header) SetSize(width int) Header {
	h.width = width
	return h
}

// SetHost updates connection info.
func (h Header) SetHost(host models.HostInfo) Header {
	h.host = host
	return h
}

// SetView updates the active view label.
func (h Header) SetView(view string) Header {
	h.view = view
	return h
}

// SetError updates an optional error banner fragment.
func (h Header) SetError(errText string) Header {
	h.errText = errText
	return h
}

// View renders the header.
func (h Header) View() string {
	brand := h.styles.Brand.Render("◈ DOKI")
	tagline := h.styles.Tagline.Render("DOCKER OPERATIONS COCKPIT")

	state := h.styles.StatusMuted.Render("● DISCONNECTED")
	switch h.host.State {
	case models.ConnectionConnected:
		state = h.styles.StatusOK.Render("● CONNECTED")
	case models.ConnectionConnecting:
		state = h.styles.StatusWarn.Render("● CONNECTING")
	case models.ConnectionError:
		state = h.styles.StatusErr.Render("● ERROR")
	}

	dockerVer := h.host.ServerVersion
	if dockerVer == "" {
		dockerVer = "n/a"
	}
	hostName := h.host.Name
	if hostName == "" {
		hostName = "local"
	}
	ctxName := h.host.ContextName
	if ctxName == "" {
		ctxName = "default"
	}

	line1 := fmt.Sprintf("%s  %s", brand, tagline)
	line2 := fmt.Sprintf("Context: %s   %s    Docker: %s    Host: %s",
		h.styles.Normal.Render(ctxName),
		state,
		h.styles.Normal.Render(dockerVer),
		h.styles.Normal.Render(hostName),
	)

	tabs := h.renderTabs()
	out := line1 + "\n" + line2 + "\n" + tabs
	if h.errText != "" {
		out += "\n" + h.styles.StatusErr.Render("⚠ "+h.errText)
	}
	return h.styles.Header.Width(max(h.width, 0)).Render(out)
}

func (h Header) renderTabs() string {
	// Keep this order in sync with app.cycleView — mismatched order makes Tab
	// feel like it jumps backwards around NETWORKS/VOLUMES.
	items := []string{"DASHBOARD", "CONTAINERS", "COMPOSE", "IMAGES", "NETWORKS", "VOLUMES", "SECURITY", "DIAGNOSTICS"}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		if item == h.view {
			parts = append(parts, h.styles.Brand.Render(item))
			continue
		}
		parts = append(parts, h.styles.StatusMuted.Render(item))
	}
	return strings.Join(parts, "  ")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
