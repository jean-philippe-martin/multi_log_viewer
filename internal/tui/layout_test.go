package tui

import (
	"strings"
	"testing"
)

func TestClipLinesTail(t *testing.T) {
	in := "a\nb\nc\nd\ne"
	got := clipLines(in, 3)
	want := "c\nd\ne"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestMainViewportAutoScroll(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}
	got, hint := visibleMainLines(lines, 3, true, 0)
	want := "c\nd\ne"
	if strings.Join(got, "\n") != want || hint {
		t.Fatalf("got %v hint=%v", got, hint)
	}
}

func TestMainViewportScrolledUp(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}
	got, hint := visibleMainLines(lines, 3, false, 1)
	want := "b\nc"
	if strings.Join(got, "\n") != want || !hint {
		t.Fatalf("got %v hint=%v", got, hint)
	}
}

func TestMainViewportPausesWhenLinesGrow(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}
	got1, _ := visibleMainLines(lines, 3, false, 1)
	lines = append(lines, "f", "g", "h")
	got2, hint := visibleMainLines(lines, 3, false, 1)
	if strings.Join(got1, "\n") != strings.Join(got2, "\n") {
		t.Fatalf("view shifted on append: %v -> %v", got1, got2)
	}
	if !hint {
		t.Fatal("expected scroll hint")
	}
}

func TestMainViewportAutoScrollFollowsGrowth(t *testing.T) {
	lines := []string{"a", "b", "c"}
	got1, _ := visibleMainLines(lines, 2, true, 0)
	lines = append(lines, "d", "e")
	got2, _ := visibleMainLines(lines, 2, true, 0)
	if strings.Join(got1, "\n") == strings.Join(got2, "\n") {
		t.Fatalf("auto-scroll should follow tail: %v", got2)
	}
	if strings.Join(got2, "\n") != "d\ne" {
		t.Fatalf("got %v", got2)
	}
}

func TestDefaultStatusText(t *testing.T) {
	if defaultStatusText(focusLeft) != "↑↓ move  Enter show  + add  ←/→ panes  q quit  ? help" {
		t.Fatal("left focus")
	}
	if defaultStatusText(focusMain) != "↑↓ scroll  PgUp/Dn page  Home/End ends  ←/→ panes  ? help" {
		t.Fatal("main focus")
	}
}

func TestViewportStart(t *testing.T) {
	if viewportStart(0, 10, 5) != 0 {
		t.Fatal("top")
	}
	if viewportStart(9, 10, 5) != 5 {
		t.Fatal("bottom")
	}
	if viewportStart(5, 10, 5) != 1 {
		t.Fatalf("middle got %d", viewportStart(5, 10, 5))
	}
	// Cursor on second row must not hide service row when pane fits two rows.
	if viewportStart(1, 2, 2) != 0 {
		t.Fatalf("short pane: got %d", viewportStart(1, 2, 2))
	}
	if viewportStart(1, 10, 1) != 0 {
		t.Fatalf("single-line viewport: got %d", viewportStart(1, 10, 1))
	}
}
