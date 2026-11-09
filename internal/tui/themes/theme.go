package themes

import (
	"image/color"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
)

// Theme defines the visual palette for DOKI.
type Theme struct {
	Name string

	Background    color.Color
	Panel         color.Color
	Border        color.Color
	Primary       color.Color
	Accent        color.Color
	Purple        color.Color
	Success       color.Color
	Warning       color.Color
	Error         color.Color
	Text          color.Color
	SecondaryText color.Color
	Muted         color.Color
}

// Styles are pre-built lipgloss styles for a theme.
type Styles struct {
	Theme Theme

	App      lipgloss.Style
	Header   lipgloss.Style
	Brand    lipgloss.Style
	Tagline  lipgloss.Style
	Panel    lipgloss.Style
	Border   lipgloss.Style
	Title    lipgloss.Style
	StatusOK lipgloss.Style
	StatusWarn lipgloss.Style
	StatusErr lipgloss.Style
	StatusMuted lipgloss.Style
	Selected lipgloss.Style
	Normal   lipgloss.Style
	Footer   lipgloss.Style
	Help     lipgloss.Style
	Filter   lipgloss.Style
	TableHeader lipgloss.Style
}

// Cyberpunk is the default DOKI theme.
func Cyberpunk() Theme {
	return Theme{
		Name:          "cyberpunk",
		Background:    lipgloss.Color("#0B1020"),
		Panel:         lipgloss.Color("#111827"),
		Border:        lipgloss.Color("#334155"),
		Primary:       lipgloss.Color("#22D3EE"),
		Accent:        lipgloss.Color("#3B82F6"),
		Purple:        lipgloss.Color("#A78BFA"),
		Success:       lipgloss.Color("#4ADE80"),
		Warning:       lipgloss.Color("#FBBF24"),
		Error:         lipgloss.Color("#FB7185"),
		Text:          lipgloss.Color("#E2E8F0"),
		SecondaryText: lipgloss.Color("#94A3B8"),
		Muted:         lipgloss.Color("#64748B"),
	}
}

// Matrix is a green-accent dark theme.
func Matrix() Theme {
	return Theme{
		Name:          "matrix",
		Background:    lipgloss.Color("#020805"),
		Panel:         lipgloss.Color("#07140C"),
		Border:        lipgloss.Color("#1B4332"),
		Primary:       lipgloss.Color("#00FF87"),
		Accent:        lipgloss.Color("#39FF14"),
		Purple:        lipgloss.Color("#7CFC00"),
		Success:       lipgloss.Color("#00FF87"),
		Warning:       lipgloss.Color("#D4FF00"),
		Error:         lipgloss.Color("#FF4D6D"),
		Text:          lipgloss.Color("#D8F3DC"),
		SecondaryText: lipgloss.Color("#95D5B2"),
		Muted:         lipgloss.Color("#52796F"),
	}
}

// Nord is a subdued frost-inspired theme.
func Nord() Theme {
	return Theme{
		Name:          "nord",
		Background:    lipgloss.Color("#2E3440"),
		Panel:         lipgloss.Color("#3B4252"),
		Border:        lipgloss.Color("#4C566A"),
		Primary:       lipgloss.Color("#88C0D0"),
		Accent:        lipgloss.Color("#81A1C1"),
		Purple:        lipgloss.Color("#B48EAD"),
		Success:       lipgloss.Color("#A3BE8C"),
		Warning:       lipgloss.Color("#EBCB8B"),
		Error:         lipgloss.Color("#BF616A"),
		Text:          lipgloss.Color("#ECEFF4"),
		SecondaryText: lipgloss.Color("#D8DEE9"),
		Muted:         lipgloss.Color("#4C566A"),
	}
}

// Monochrome is a minimal high-contrast theme.
func Monochrome() Theme {
	return Theme{
		Name:          "monochrome",
		Background:    lipgloss.Color("#000000"),
		Panel:         lipgloss.Color("#111111"),
		Border:        lipgloss.Color("#555555"),
		Primary:       lipgloss.Color("#FFFFFF"),
		Accent:        lipgloss.Color("#DDDDDD"),
		Purple:        lipgloss.Color("#BBBBBB"),
		Success:       lipgloss.Color("#FFFFFF"),
		Warning:       lipgloss.Color("#CCCCCC"),
		Error:         lipgloss.Color("#FFFFFF"),
		Text:          lipgloss.Color("#FFFFFF"),
		SecondaryText: lipgloss.Color("#AAAAAA"),
		Muted:         lipgloss.Color("#777777"),
	}
}

// ByName returns a theme by name, falling back to cyberpunk.
func ByName(name string) Theme {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "matrix":
		return Matrix()
	case "nord":
		return Nord()
	case "monochrome", "mono":
		return Monochrome()
	default:
		return Cyberpunk()
	}
}

// Names returns available theme identifiers.
func Names() []string {
	return []string{"cyberpunk", "matrix", "nord", "monochrome"}
}

// NewStyles builds lipgloss styles for a theme, respecting NO_COLOR.
func NewStyles(theme Theme) Styles {
	if noColorEnabled() {
		theme = Monochrome()
	}

	base := lipgloss.NewStyle().Foreground(theme.Text)
	return Styles{
		Theme: theme,
		App: lipgloss.NewStyle().
			Background(theme.Background).
			Foreground(theme.Text),
		Header: lipgloss.NewStyle().
			Background(theme.Panel).
			Foreground(theme.Text).
			Padding(0, 1),
		Brand: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true),
		Tagline: lipgloss.NewStyle().
			Foreground(theme.SecondaryText),
		Panel: lipgloss.NewStyle().
			Background(theme.Panel).
			Foreground(theme.Text).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border).
			Padding(0, 1),
		Border: lipgloss.NewStyle().Foreground(theme.Border),
		Title: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true),
		StatusOK: lipgloss.NewStyle().
			Foreground(theme.Success).
			Bold(true),
		StatusWarn: lipgloss.NewStyle().
			Foreground(theme.Warning).
			Bold(true),
		StatusErr: lipgloss.NewStyle().
			Foreground(theme.Error).
			Bold(true),
		StatusMuted: lipgloss.NewStyle().
			Foreground(theme.Muted),
		Selected: lipgloss.NewStyle().
			Foreground(theme.Background).
			Background(theme.Primary).
			Bold(true),
		Normal: base,
		Footer: lipgloss.NewStyle().
			Foreground(theme.SecondaryText).
			Background(theme.Panel).
			Padding(0, 1),
		Help: lipgloss.NewStyle().
			Foreground(theme.Accent),
		Filter: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true),
		TableHeader: lipgloss.NewStyle().
			Foreground(theme.SecondaryText).
			Bold(true),
	}
}

func noColorEnabled() bool {
	_, set := os.LookupEnv("NO_COLOR")
	return set
}
