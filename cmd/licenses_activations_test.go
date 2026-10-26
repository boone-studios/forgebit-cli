package cmd

import "testing"

func TestPrintableStripsControlCharacters(t *testing.T) {
	if got := printable("dev\x1b[2J\x1b]0;pwned\x07-1\n"); got != "dev[2J]0;pwned-1" {
		t.Fatalf("got %q", got)
	}
	if got := printable("café-ü 1"); got != "café-ü 1" {
		t.Fatalf("printable text was altered: %q", got)
	}
}
