package charts

import "strings"

var blocks = []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// Sparkline renders a compact unicode sparkline for values.
func Sparkline(values []float64, width int) string {
	if width <= 0 {
		return ""
	}
	if len(values) == 0 {
		return strings.Repeat("·", width)
	}
	data := values
	if len(data) > width {
		data = data[len(data)-width:]
	}
	max := 0.0
	for _, v := range data {
		if v > max {
			max = v
		}
	}
	if max <= 0 {
		return strings.Repeat(string(blocks[0]), width)
	}

	var b strings.Builder
	pad := width - len(data)
	if pad > 0 {
		b.WriteString(strings.Repeat("·", pad))
	}
	for _, v := range data {
		idx := int((v / max) * float64(len(blocks)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		b.WriteRune(blocks[idx])
	}
	return b.String()
}

// Bar renders a fixed-width usage bar.
func Bar(percent float64, width int) string {
	if width <= 0 {
		return ""
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filled := int((percent / 100.0) * float64(width))
	if percent > 0 && filled == 0 {
		filled = 1
	}
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

// FormatBytes renders byte counts in human units.
func FormatBytes(v uint64) string {
	const unit = 1024
	if v < unit {
		return itoa(v) + " B"
	}
	div, exp := uint64(unit), 0
	for n := v / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	if exp >= len(units) {
		exp = len(units) - 1
	}
	val := float64(v) / float64(div)
	return trimFloat(val) + " " + units[exp]
}

func itoa(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

func trimFloat(v float64) string {
	s := strings.TrimRight(strings.TrimRight(sprintf1(v), "0"), ".")
	if s == "" {
		return "0"
	}
	return s
}

func sprintf1(v float64) string {
	// lightweight one-decimal formatting without fmt for hot path simplicity
	neg := v < 0
	if neg {
		v = -v
	}
	whole := uint64(v)
	frac := uint64((v - float64(whole)) * 10)
	s := itoa(whole) + "." + itoa(frac)
	if neg {
		return "-" + s
	}
	return s
}
