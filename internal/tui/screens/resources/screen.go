package resources

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/zpol/doki/internal/domain/models"
	"github.com/zpol/doki/internal/tui/charts"
	"github.com/zpol/doki/internal/tui/themes"
)

// Kind identifies which resource list is shown.
type Kind int

const (
	KindImages Kind = iota
	KindVolumes
	KindNetworks
)

// Screen lists images, volumes, or networks.
type Screen struct {
	styles themes.Styles
	kind   Kind
	cursor int
	offset int
	width  int
	height int

	images   []models.Image
	volumes  []models.Volume
	networks []models.Network
}

func New(styles themes.Styles, kind Kind) Screen {
	return Screen{styles: styles, kind: kind}
}

func (s Screen) SetStyles(styles themes.Styles) Screen { s.styles = styles; return s }
func (s Screen) SetSize(w, h int) Screen                { s.width = w; s.height = h; return s }
func (s Screen) SetImages(items []models.Image) Screen {
	s.images = items
	s = s.clamp()
	return s
}
func (s Screen) SetVolumes(items []models.Volume) Screen {
	s.volumes = items
	s = s.clamp()
	return s
}
func (s Screen) SetNetworks(items []models.Network) Screen {
	s.networks = items
	s = s.clamp()
	return s
}

func (s Screen) Len() int {
	switch s.kind {
	case KindImages:
		return len(s.images)
	case KindVolumes:
		return len(s.volumes)
	default:
		return len(s.networks)
	}
}

// SelectedImage returns the highlighted image when available.
func (s Screen) SelectedImage() (models.Image, bool) {
	if s.kind != KindImages || s.cursor < 0 || s.cursor >= len(s.images) {
		return models.Image{}, false
	}
	return s.images[s.cursor], true
}

// SelectedVolume returns the highlighted volume when available.
func (s Screen) SelectedVolume() (models.Volume, bool) {
	if s.kind != KindVolumes || s.cursor < 0 || s.cursor >= len(s.volumes) {
		return models.Volume{}, false
	}
	return s.volumes[s.cursor], true
}

// SelectedNetwork returns the highlighted network when available.
func (s Screen) SelectedNetwork() (models.Network, bool) {
	if s.kind != KindNetworks || s.cursor < 0 || s.cursor >= len(s.networks) {
		return models.Network{}, false
	}
	return s.networks[s.cursor], true
}

func (s Screen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "j", "down":
			if s.cursor < s.Len()-1 {
				s.cursor++
				s = s.ensureVisible()
			}
		case "k", "up":
			if s.cursor > 0 {
				s.cursor--
				s = s.ensureVisible()
			}
		}
	}
	return s, nil
}

func (s Screen) View() string {
	var b strings.Builder
	title := "IMAGES"
	switch s.kind {
	case KindVolumes:
		title = "VOLUMES"
	case KindNetworks:
		title = "NETWORKS"
	}
	b.WriteString(s.styles.Title.Render(title) + "  " + s.styles.Tagline.Render(fmt.Sprintf("%d", s.Len())) + "\n")

	switch s.kind {
	case KindImages:
		b.WriteString(s.styles.TableHeader.Render(fmt.Sprintf("%-20s %-12s %-14s %-12s", "REPOSITORY", "TAG", "ID", "SIZE")) + "\n")
		visible := s.visible()
		end := min(s.offset+visible, len(s.images))
		for i := s.offset; i < end; i++ {
			img := s.images[i]
			line := fmt.Sprintf("%-20s %-12s %-14s %-12s",
				trunc(img.Repository, 20), trunc(img.Tag, 12), img.ShortID(), charts.FormatBytes(uint64(img.Size)))
			b.WriteString(s.row(line, i == s.cursor) + "\n")
		}
	case KindVolumes:
		b.WriteString(s.styles.TableHeader.Render(fmt.Sprintf("%-28s %-12s %-10s", "NAME", "DRIVER", "SCOPE")) + "\n")
		visible := s.visible()
		end := min(s.offset+visible, len(s.volumes))
		for i := s.offset; i < end; i++ {
			v := s.volumes[i]
			line := fmt.Sprintf("%-28s %-12s %-10s", trunc(v.Name, 28), trunc(v.Driver, 12), v.Scope)
			b.WriteString(s.row(line, i == s.cursor) + "\n")
		}
	default:
		b.WriteString(s.styles.TableHeader.Render(fmt.Sprintf("%-24s %-12s %-10s %-8s", "NAME", "DRIVER", "SCOPE", "CTR")) + "\n")
		visible := s.visible()
		end := min(s.offset+visible, len(s.networks))
		for i := s.offset; i < end; i++ {
			n := s.networks[i]
			line := fmt.Sprintf("%-24s %-12s %-10s %-8d", trunc(n.Name, 24), trunc(n.Driver, 12), n.Scope, n.Containers)
			b.WriteString(s.row(line, i == s.cursor) + "\n")
		}
	}

	if s.Len() == 0 {
		b.WriteString(s.styles.StatusMuted.Render("No resources found."))
	}
	return s.styles.Panel.Width(max(s.width, 0)).Height(max(s.height, 0)).Render(strings.TrimRight(b.String(), "\n"))
}

func (s Screen) row(line string, selected bool) string {
	if selected {
		return s.styles.Selected.Width(max(s.width-4, 0)).Render(line)
	}
	return s.styles.Normal.Render(line)
}

func (s Screen) visible() int {
	n := s.height - 5
	if n < 3 {
		n = 3
	}
	return n
}

func (s Screen) clamp() Screen {
	if s.cursor >= s.Len() && s.Len() > 0 {
		s.cursor = s.Len() - 1
	}
	if s.Len() == 0 {
		s.cursor = 0
	}
	return s.ensureVisible()
}

func (s Screen) ensureVisible() Screen {
	vis := s.visible()
	if s.cursor < s.offset {
		s.offset = s.cursor
	}
	if s.cursor >= s.offset+vis {
		s.offset = s.cursor - vis + 1
	}
	if s.offset < 0 {
		s.offset = 0
	}
	return s
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
