package tail

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRingEviction(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.log")
	if err := os.WriteFile(path, []byte("line1\nline2\nline3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := NewFile(path, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	lines := f.Lines()
	if len(lines) != 2 {
		t.Fatalf("len=%d", len(lines))
	}
	if lines[0].Text != "line2" {
		t.Fatalf("first=%q", lines[0].Text)
	}
}

func TestAppend(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.log")
	if err := os.WriteFile(path, []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := NewFile(path, 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fh, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	_, _ = fh.WriteString("b\n")
	_ = fh.Close()
	time.Sleep(50 * time.Millisecond)
	_ = f.readAppend()
	lines := f.Lines()
	if len(lines) != 2 || lines[1].Text != "b" {
		t.Fatalf("lines=%+v", lines)
	}
}
