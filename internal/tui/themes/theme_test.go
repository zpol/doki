package themes

import (
	"testing"
)

func TestByName(t *testing.T) {
	cases := map[string]string{
		"cyberpunk":  "cyberpunk",
		"MATRIX":     "matrix",
		"nord":       "nord",
		"mono":       "monochrome",
		"unknown":    "cyberpunk",
		"":           "cyberpunk",
	}
	for in, want := range cases {
		got := ByName(in)
		if got.Name != want {
			t.Fatalf("ByName(%q)=%q want %q", in, got.Name, want)
		}
	}
}

func TestNames(t *testing.T) {
	names := Names()
	if len(names) != 4 {
		t.Fatalf("expected 4 themes, got %d", len(names))
	}
}

func TestNewStylesRespectsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	styles := NewStyles(Cyberpunk())
	if styles.Theme.Name != "monochrome" {
		t.Fatalf("expected monochrome under NO_COLOR, got %s", styles.Theme.Name)
	}
}
