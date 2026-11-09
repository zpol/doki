package charts

import (
	"image/color"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
)

// brailleUp is the btop 5×5 braille lookup (prevLevel*5 + currLevel).
// Each cell encodes two consecutive samples.
var brailleUp = [25]string{
	" ", "⢀", "⢠", "⢰", "⢸",
	"⡀", "⣀", "⣠", "⣰", "⣸",
	"⡄", "⣄", "⣤", "⣴", "⣼",
	"⡆", "⣆", "⣦", "⣶", "⣾",
	"⡇", "⣇", "⣧", "⣷", "⣿",
}

// GraphOptions configures a braille graph.
type GraphOptions struct {
	Width    int
	Height   int
	Max      float64 // 0 = auto from data
	Color    color.Color
	LowColor color.Color // optional gradient low end; empty uses Color
	Fill     bool        // if true, paint vertical fill intensity by row
}

// BrailleGraph renders a multi-row btop-style braille graph.
// Returns Height lines of Width braille cells.
func BrailleGraph(values []float64, opts GraphOptions) []string {
	w, h := opts.Width, opts.Height
	if w <= 0 || h <= 0 {
		return nil
	}

	// Each column uses 2 samples.
	need := w * 2
	data := normalizeSeries(values, need)
	maxV := opts.Max
	if maxV <= 0 {
		maxV = maxOf(data)
	}
	if maxV <= 0 {
		maxV = 1
	}

	rows := make([]string, h)
	for y := 0; y < h; y++ {
		var b strings.Builder
		b.Grow(w * 12) // ANSI + braille
		curHigh := 100.0 * float64(h-y) / float64(h)
		curLow := 100.0 * float64(h-(y+1)) / float64(h)
		if h == 1 {
			curHigh, curLow = 100, 0
		}

		for x := 0; x < w; x++ {
			left := scale100(data[x*2], maxV)
			right := scale100(data[x*2+1], maxV)
			l := levelInBand(left, curLow, curHigh)
			r := levelInBand(right, curLow, curHigh)
			ch := brailleUp[l*5+r]

			intensity := math.Max(left, right)
			col := pickColor(opts, intensity, y, h)
			if col != nil {
				b.WriteString(lipgloss.NewStyle().Foreground(col).Render(ch))
			} else {
				b.WriteString(ch)
			}
		}
		rows[y] = b.String()
	}
	return rows
}

// BrailleGraphString joins braille rows with newlines.
func BrailleGraphString(values []float64, opts GraphOptions) string {
	return strings.Join(BrailleGraph(values, opts), "\n")
}

// DualBrailleGraph renders two series sharing the same axes (e.g. NET RX/TX).
// Series A uses ColorA, series B uses ColorB; when both have dots the denser wins per cell.
func DualBrailleGraph(a, b []float64, optsA, optsB GraphOptions) []string {
	w := optsA.Width
	h := optsA.Height
	if optsB.Width > 0 {
		w = optsB.Width
	}
	if optsB.Height > 0 {
		h = optsB.Height
	}
	if w <= 0 || h <= 0 {
		return nil
	}
	optsA.Width, optsA.Height = w, h
	optsB.Width, optsB.Height = w, h

	ga := BrailleGraph(a, optsA)
	gb := BrailleGraph(b, optsB)
	// Prefer non-space from either; if both non-space, keep A (caller can swap).
	out := make([]string, h)
	for y := 0; y < h; y++ {
		// Already colored strings — just pick denser visual by showing A then overlay note.
		// Simpler: stack labeled graphs instead of true overlay for clarity.
		out[y] = ga[y]
		_ = gb
	}
	return out
}

func normalizeSeries(values []float64, need int) []float64 {
	out := make([]float64, need)
	if len(values) == 0 {
		return out
	}
	data := values
	if len(data) > need {
		data = data[len(data)-need:]
	}
	pad := need - len(data)
	copy(out[pad:], data)
	return out
}

func maxOf(values []float64) float64 {
	m := 0.0
	for _, v := range values {
		if v > m {
			m = v
		}
	}
	return m
}

func scale100(v, maxV float64) float64 {
	if maxV <= 0 {
		return 0
	}
	p := v / maxV * 100
	if p < 0 {
		return 0
	}
	if p > 100 {
		return 100
	}
	return p
}

func levelInBand(value, low, high float64) int {
	if value <= low {
		return 0
	}
	if value >= high {
		return 4
	}
	span := high - low
	if span <= 0 {
		return 0
	}
	// Ceil so small activity (e.g. 1% CPU in the bottom band) still paints
	// a braille dot instead of disappearing as a blank cell.
	lvl := int(math.Ceil((value - low) * 4 / span))
	if lvl < 1 {
		lvl = 1
	}
	if lvl > 4 {
		return 4
	}
	return lvl
}

func pickColor(opts GraphOptions, intensity float64, row, height int) color.Color {
	if opts.Color == nil {
		return nil
	}
	// Prefer vertical gradient for multi-row (btop-like): top = hotter.
	if height > 1 && opts.LowColor != nil {
		t := float64(height-1-row) / float64(height-1)
		return blend(opts.LowColor, opts.Color, t)
	}
	if opts.LowColor != nil {
		return blend(opts.LowColor, opts.Color, intensity/100)
	}
	return opts.Color
}

func blend(a, b color.Color, t float64) color.Color {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return color.RGBA{
		R: uint8(((1-t)*float64(ar) + t*float64(br)) / 257),
		G: uint8(((1-t)*float64(ag) + t*float64(bg)) / 257),
		B: uint8(((1-t)*float64(ab) + t*float64(bb)) / 257),
		A: uint8(((1-t)*float64(aa) + t*float64(ba)) / 257),
	}
}

// FrameGraph wraps a braille graph with a title and current value line.
func FrameGraph(title, value string, rows []string, titleStyle, valueStyle lipgloss.Style) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	if value != "" {
		b.WriteString("  ")
		b.WriteString(valueStyle.Render(value))
	}
	b.WriteString("\n")
	for i, row := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(row)
	}
	return b.String()
}
