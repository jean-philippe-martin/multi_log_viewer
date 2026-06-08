package pattern

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompileAndMatch(t *testing.T) {
	tmpl, err := Compile("logs/{client}/log.txt")
	if err != nil {
		t.Fatal(err)
	}
	caps, ok := tmpl.MatchCaptures("logs/acme/log.txt")
	if !ok || caps["client"] != "acme" {
		t.Fatalf("caps=%v ok=%v", caps, ok)
	}
}

func TestDiscover(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "logs", "acme"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "logs", "acme", "log.txt"), []byte("x"), 0o644)
	tmpl, _ := Compile("logs/{client}/log.txt")
	paths, err := Discover(dir, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || paths[0] != "logs/acme/log.txt" {
		t.Fatalf("paths=%v", paths)
	}
}
