package detail

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/zpol/doki/internal/domain/models"
	"github.com/zpol/doki/internal/tui/charts"
	"github.com/zpol/doki/internal/tui/themes"
)

// Kind identifies which resource detail is shown.
type Kind int

const (
	KindImage Kind = iota
	KindVolume
	KindNetwork
)

// Screen shows inspect details for images, volumes, or networks.
type Screen struct {
	styles themes.Styles
	kind   Kind
	title  string
	lines  []string
	offset int
	width  int
	height int
	ready  bool
	err    string
	back   string // view label to restore on esc (IMAGES/VOLUMES/NETWORKS)
}

func New(styles themes.Styles) Screen {
	return Screen{styles: styles, back: "IMAGES"}
}

func (s Screen) SetStyles(styles themes.Styles) Screen { s.styles = styles; return s }
func (s Screen) SetSize(w, h int) Screen                { s.width = w; s.height = h; return s }
func (s Screen) BackView() string                       { return s.back }

func (s Screen) Clear(kind Kind, back string) Screen {
	s.kind = kind
	s.back = back
	s.title = ""
	s.lines = nil
	s.offset = 0
	s.ready = false
	s.err = ""
	return s
}

func (s Screen) SetError(err string) Screen {
	s.err = err
	s.ready = false
	return s
}

func (s Screen) SetImage(d models.ImageDetails) Screen {
	s.kind = KindImage
	s.back = "IMAGES"
	s.title = "IMAGE: " + d.Repository + ":" + d.Tag
	s.ready = true
	s.err = ""
	s.offset = 0
	var b []string
	b = append(b, kv("ID", d.ID))
	b = append(b, kv("Repository", d.Repository))
	b = append(b, kv("Tag", d.Tag))
	b = append(b, kv("Size", charts.FormatBytes(uint64(d.Size))))
	b = append(b, kv("Created", formatTime(d.CreatedAt)))
	b = append(b, kv("OS/Arch", d.OS+"/"+d.Architecture))
	b = append(b, kv("Author", empty(d.Author, "-")))
	b = append(b, kv("Working Dir", empty(d.WorkingDir, "/")))
	b = append(b, kv("Comment", empty(d.Comment, "-")))
	b = append(b, "")
	b = append(b, "REPO TAGS")
	if len(d.RepoTags) == 0 {
		b = append(b, "  (none)")
	} else {
		for _, t := range d.RepoTags {
			b = append(b, "  "+t)
		}
	}
	b = append(b, "", "EXPOSED PORTS")
	if len(d.ExposedPorts) == 0 {
		b = append(b, "  (none)")
	} else {
		for _, p := range d.ExposedPorts {
			b = append(b, "  "+p)
		}
	}
	b = append(b, "", "ENTRYPOINT")
	b = append(b, "  "+empty(strings.Join(d.Entrypoint, " "), "(none)"))
	b = append(b, "", "CMD")
	b = append(b, "  "+empty(strings.Join(d.Cmd, " "), "(none)"))
	b = append(b, "", "ENVIRONMENT")
	if len(d.Env) == 0 {
		b = append(b, "  (none)")
	} else {
		for _, e := range d.Env {
			b = append(b, "  "+redactEnv(e))
		}
	}
	b = append(b, "", "LABELS")
	if len(d.Labels) == 0 {
		b = append(b, "  (none)")
	} else {
		for k, v := range d.Labels {
			b = append(b, "  "+k+"="+v)
		}
	}
	s.lines = b
	return s
}

func (s Screen) SetVolume(d models.VolumeDetails) Screen {
	s.kind = KindVolume
	s.back = "VOLUMES"
	s.title = "VOLUME: " + d.Name
	s.ready = true
	s.err = ""
	s.offset = 0
	var b []string
	b = append(b, kv("Name", d.Name))
	b = append(b, kv("Driver", d.Driver))
	b = append(b, kv("Scope", d.Scope))
	b = append(b, kv("Mountpoint", d.Mountpoint))
	b = append(b, kv("Created", formatTime(d.CreatedAt)))
	if d.Size >= 0 {
		b = append(b, kv("Size", charts.FormatBytes(uint64(d.Size))))
	}
	if d.RefCount >= 0 {
		b = append(b, kv("Ref Count", fmt.Sprintf("%d", d.RefCount)))
	}
	b = append(b, "", "OPTIONS")
	if len(d.Options) == 0 {
		b = append(b, "  (none)")
	} else {
		for k, v := range d.Options {
			b = append(b, "  "+k+"="+v)
		}
	}
	b = append(b, "", "LABELS")
	if len(d.Labels) == 0 {
		b = append(b, "  (none)")
	} else {
		for k, v := range d.Labels {
			b = append(b, "  "+k+"="+v)
		}
	}
	s.lines = b
	return s
}

func (s Screen) SetNetwork(d models.NetworkDetails) Screen {
	s.kind = KindNetwork
	s.back = "NETWORKS"
	s.title = "NETWORK: " + d.Name
	s.ready = true
	s.err = ""
	s.offset = 0
	var b []string
	b = append(b, kv("ID", d.ID))
	b = append(b, kv("Name", d.Name))
	b = append(b, kv("Driver", d.Driver))
	b = append(b, kv("Scope", d.Scope))
	b = append(b, kv("Created", formatTime(d.Created)))
	b = append(b, kv("Internal", fmt.Sprintf("%v", d.Internal)))
	b = append(b, kv("Attachable", fmt.Sprintf("%v", d.Attachable)))
	b = append(b, kv("Ingress", fmt.Sprintf("%v", d.Ingress)))
	b = append(b, kv("IPv4/IPv6", fmt.Sprintf("%v / %v", d.EnableIPv4, d.EnableIPv6)))
	b = append(b, kv("Subnet", empty(d.Subnet, "-")))
	b = append(b, kv("Gateway", empty(d.Gateway, "-")))
	b = append(b, "", "CONNECTED CONTAINERS")
	if len(d.Endpoints) == 0 {
		b = append(b, "  (none)")
	} else {
		for _, ep := range d.Endpoints {
			b = append(b, fmt.Sprintf("  %s  ipv4=%s  mac=%s",
				empty(ep.ContainerName, short(ep.ContainerID)),
				empty(ep.IPv4Address, "-"),
				empty(ep.MacAddress, "-"),
			))
		}
	}
	b = append(b, "", "OPTIONS")
	if len(d.Options) == 0 {
		b = append(b, "  (none)")
	} else {
		for k, v := range d.Options {
			b = append(b, "  "+k+"="+v)
		}
	}
	b = append(b, "", "LABELS")
	if len(d.Labels) == 0 {
		b = append(b, "  (none)")
	} else {
		for k, v := range d.Labels {
			b = append(b, "  "+k+"="+v)
		}
	}
	s.lines = b
	return s
}

func (s Screen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		vis := s.visibleLines()
		maxOff := len(s.lines) - vis
		if maxOff < 0 {
			maxOff = 0
		}
		switch msg.String() {
		case "j", "down":
			if s.offset < maxOff {
				s.offset++
			}
		case "k", "up":
			if s.offset > 0 {
				s.offset--
			}
		case "g", "home":
			s.offset = 0
		case "G", "end":
			s.offset = maxOff
		}
	}
	return s, nil
}

func (s Screen) View() string {
	var body string
	switch {
	case s.err != "":
		body = s.styles.StatusErr.Render("Inspect failed: "+s.err) + "\n\n" + s.styles.Help.Render("[esc] Back")
	case !s.ready:
		body = s.styles.StatusMuted.Render("Loading details…")
	default:
		var b strings.Builder
		b.WriteString(s.styles.Title.Render(s.title) + "\n\n")
		vis := s.visibleLines()
		end := s.offset + vis
		if end > len(s.lines) {
			end = len(s.lines)
		}
		for _, line := range s.lines[s.offset:end] {
			if line == "REPO TAGS" || line == "EXPOSED PORTS" || line == "ENTRYPOINT" ||
				line == "CMD" || line == "ENVIRONMENT" || line == "LABELS" || line == "OPTIONS" ||
				line == "CONNECTED CONTAINERS" {
				b.WriteString(s.styles.TableHeader.Render(line) + "\n")
				continue
			}
			if strings.Contains(line, ":") && !strings.HasPrefix(line, "  ") {
				parts := strings.SplitN(line, ":", 2)
				b.WriteString(s.styles.Tagline.Render(fmt.Sprintf("%-16s", parts[0])) + s.styles.Normal.Render(parts[1]) + "\n")
				continue
			}
			b.WriteString(s.styles.Normal.Render(line) + "\n")
		}
		b.WriteString("\n" + s.styles.Help.Render("[esc] Back  [j/k] Scroll"))
		body = b.String()
	}
	panel := s.styles.Panel
	if s.width > 0 {
		panel = panel.Width(s.width)
	}
	if s.height > 0 {
		panel = panel.Height(s.height)
	}
	return panel.Render(strings.TrimRight(body, "\n"))
}

func (s Screen) visibleLines() int {
	n := s.height - 5
	if n < 5 {
		n = 5
	}
	return n
}

func kv(k, v string) string { return k + ":" + v }

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

func short(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}

func redactEnv(env string) string {
	parts := strings.SplitN(env, "=", 2)
	if len(parts) != 2 {
		return env
	}
	lower := strings.ToLower(parts[0])
	for _, s := range []string{"password", "secret", "token", "key", "credential", "auth", "api_key", "apikey"} {
		if strings.Contains(lower, s) {
			return parts[0] + "=********"
		}
	}
	return env
}
