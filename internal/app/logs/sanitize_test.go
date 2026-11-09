package logs

import "testing"

func TestSanitizeStripsANSI(t *testing.T) {
	in := "\x1b[31mERROR\x1b[0m boom"
	got := Sanitize(in)
	if got != "ERROR boom" {
		t.Fatalf("got %q", got)
	}
}

func TestSanitizeDropsControls(t *testing.T) {
	got := Sanitize("ok\x00\x07 there")
	if got != "ok there" {
		t.Fatalf("got %q", got)
	}
}
