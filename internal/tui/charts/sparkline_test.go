package charts

import "testing"

func TestSparklineEmpty(t *testing.T) {
	got := Sparkline(nil, 5)
	if len([]rune(got)) != 5 {
		t.Fatalf("got %q", got)
	}
}

func TestBarBounds(t *testing.T) {
	if Bar(0, 10) != "░░░░░░░░░░" {
		t.Fatalf("zero bar unexpected: %q", Bar(0, 10))
	}
	if Bar(100, 4) != "████" {
		t.Fatalf("full bar unexpected: %q", Bar(100, 4))
	}
}

func TestFormatBytes(t *testing.T) {
	if FormatBytes(512) != "512 B" {
		t.Fatalf("%q", FormatBytes(512))
	}
	got := FormatBytes(1536)
	if got == "" {
		t.Fatal("empty")
	}
}
