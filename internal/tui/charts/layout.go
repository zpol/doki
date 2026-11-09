package charts

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// JoinHorizontal places two blocks side by side, padding the shorter one.
func JoinHorizontal(left, right string, gap int, leftWidth int) string {
	if gap < 0 {
		gap = 0
	}
	lLines := strings.Split(strings.TrimRight(left, "\n"), "\n")
	rLines := strings.Split(strings.TrimRight(right, "\n"), "\n")
	n := len(lLines)
	if len(rLines) > n {
		n = len(rLines)
	}
	spacer := strings.Repeat(" ", gap)
	var b strings.Builder
	for i := 0; i < n; i++ {
		l, r := "", ""
		if i < len(lLines) {
			l = lLines[i]
		}
		if i < len(rLines) {
			r = rLines[i]
		}
		if leftWidth > 0 {
			lw := ansi.StringWidth(l)
			if lw < leftWidth {
				l += strings.Repeat(" ", leftWidth-lw)
			} else if lw > leftWidth {
				l = lipgloss.NewStyle().Width(leftWidth).MaxWidth(leftWidth).Render(l)
			}
		}
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(l)
		b.WriteString(spacer)
		b.WriteString(r)
	}
	return b.String()
}

// VisibleWidth returns the printable width of a styled string.
func VisibleWidth(s string) int {
	return ansi.StringWidth(s)
}
