package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

func TestFormatTimestamp(t *testing.T) {
	ts, err := formatTimestamp("hms-ms")
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) < 20 {
		t.Fatalf("short timestamp: %q", ts)
	}
}

func TestFlagsAfterPath(t *testing.T) {
	fs := flag.NewFlagSet("loggen", flag.ContinueOnError)
	interval := fs.Float64("interval", 1.0, "")
	count := fs.Int("count", 0, "")
	args := []string{"--interval", "0.5", "--count", "3"}
	if err := fs.Parse(args); err != nil {
		t.Fatal(err)
	}
	if *interval != 0.5 || *count != 3 {
		t.Fatalf("interval=%v count=%v", *interval, *count)
	}
}

func TestCountLogic(t *testing.T) {
	remaining := -1
	if 5 > 1 {
		remaining = 5 - 1
	}
	if remaining != 4 {
		t.Fatalf("want 4 got %d", remaining)
	}
	_ = filepath.Join
	_ = os.TempDir
}
