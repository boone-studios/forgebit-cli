package cmd

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("connection reset") }

func TestWriteNewFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "license.lic")

	if n, err := writeNewFile(path, strings.NewReader("first"), false); err != nil || n != 5 {
		t.Fatalf("first write: n=%d err=%v", n, err)
	}
	if _, err := writeNewFile(path, strings.NewReader("second"), false); err == nil {
		t.Fatal("expected refusal to overwrite without force")
	}
	if got, _ := os.ReadFile(path); string(got) != "first" {
		t.Fatalf("existing file was modified: %q", got)
	}
	if _, err := writeNewFile(path, strings.NewReader("2nd"), true); err != nil {
		t.Fatalf("forced write: %v", err)
	}
	if got, _ := os.ReadFile(path); string(got) != "2nd" {
		t.Fatalf("forced write left %q", got)
	}

	partial := filepath.Join(t.TempDir(), "partial.lic")
	if _, err := writeNewFile(partial, io.MultiReader(strings.NewReader("half"), failingReader{}), false); err == nil {
		t.Fatal("expected the read error")
	}
	if _, err := os.Stat(partial); !os.IsNotExist(err) {
		t.Fatal("partial file was left behind")
	}
}
