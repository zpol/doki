package charts

import (
	"strings"
	"testing"
)

func TestBrailleGraphDimensions(t *testing.T) {
	vals := []float64{10, 20, 40, 80, 50, 30, 70, 90, 60, 40}
	rows := BrailleGraph(vals, GraphOptions{Width: 8, Height: 4, Max: 100})
	if len(rows) != 4 {
		t.Fatalf("rows=%d", len(rows))
	}
	for i, row := range rows {
		// Strip ANSI — plain braille width should be 8
		plain := stripANSI(row)
		if len([]rune(plain)) != 8 {
			t.Fatalf("row %d width=%d %q", i, len([]rune(plain)), plain)
		}
	}
}

func TestBrailleGraphEmpty(t *testing.T) {
	rows := BrailleGraph(nil, GraphOptions{Width: 5, Height: 2, Max: 100})
	if len(rows) != 2 {
		t.Fatal(rows)
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == 0x1b {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
