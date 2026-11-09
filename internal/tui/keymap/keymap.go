package keymap

// Binding describes a keyboard shortcut shown in the footer/help.
type Binding struct {
	Keys string
	Help string
}

// Registry holds contextual key bindings.
type Registry struct {
	Global     []Binding
	Containers []Binding
	Inspect    []Binding
	Confirm    []Binding
	Filter     []Binding
}

// Default returns the Milestone 1 key bindings.
func Default() Registry {
	return Registry{
		Global: []Binding{
			{Keys: "?", Help: "Help"},
			{Keys: "r", Help: "Refresh"},
			{Keys: "q", Help: "Quit"},
			{Keys: "ctrl+c", Help: "Quit"},
		},
		Containers: []Binding{
			{Keys: "j/k", Help: "Navigate"},
			{Keys: "space", Help: "Mark"},
			{Keys: "enter", Help: "Inspect"},
			{Keys: "/", Help: "Filter"},
			{Keys: "l", Help: "Logs"},
			{Keys: "S", Help: "Shell"},
			{Keys: "s", Help: "Start"},
			{Keys: "x", Help: "Stop"},
			{Keys: "R", Help: "Restart"},
			{Keys: "D", Help: "Remove"},
			{Keys: "d", Help: "Diagnostics"},
			{Keys: "p", Help: "Pause/Unpause"},
			{Keys: "A/U", Help: "Mark all/none"},
			{Keys: ":", Help: "Palette"},
			{Keys: "Tab", Help: "Views"},
		},
		Inspect: []Binding{
			{Keys: "esc", Help: "Back"},
			{Keys: "j/k", Help: "Scroll"},
		},
		Confirm: []Binding{
			{Keys: "y", Help: "Confirm"},
			{Keys: "n/esc", Help: "Cancel"},
		},
		Filter: []Binding{
			{Keys: "enter", Help: "Apply"},
			{Keys: "esc", Help: "Cancel"},
		},
	}
}

// FormatFooter renders a compact shortcut line.
func FormatFooter(bindings []Binding) string {
	parts := make([]string, 0, len(bindings))
	for _, b := range bindings {
		parts = append(parts, "["+b.Keys+"] "+b.Help)
	}
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "  "
		}
		out += p
	}
	return out
}
