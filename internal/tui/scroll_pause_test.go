package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/jpmartin/multi_log_viewer/internal/engine"
)

func TestPauseScrollFreezesMainPaneOnLiveAppend(t *testing.T) {
	dir, eng := testEngineWithLog(t, 25)
	defer eng.Close()

	m := newTestModel(t, eng)
	eng.Subscribe(m.events)
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 14})

	m = updateModel(m, keyMsg("k"))
	if m.autoScroll {
		t.Fatal("expected auto-scroll to pause")
	}
	if m.frozenDisplayLines == nil {
		t.Fatal("expected frozen display snapshot")
	}

	pausedMain := mainPaneText(m)
	if !strings.Contains(pausedMain, scrollHint) {
		t.Fatal("expected scroll hint while paused")
	}

	logPath := filepath.Join(dir, "logs", "api.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprint(f, "line-25 NEW\nline-26 NEW\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)

	m = updateModel(m, engine.Event{Kind: engine.EventTick})
	m = updateModel(m, tickMsg{})

	if got := mainPaneText(m); got != pausedMain {
		t.Fatalf("main pane changed while scroll was paused\nbefore: %q\nafter:  %q", pausedMain, got)
	}

	live := m.buildMainLines(m.mainContentWidth(), m.paneHeight())
	if len(live) <= len(m.frozenDisplayLines) {
		t.Fatalf("live tail did not grow: frozen=%d live=%d", len(m.frozenDisplayLines), len(live))
	}
}

func TestPauseScrollResumesAtBottom(t *testing.T) {
	dir, eng := testEngineWithLog(t, 25)
	defer eng.Close()

	m := newTestModel(t, eng)
	eng.Subscribe(m.events)
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 14})
	m = updateModel(m, keyMsg("k"))

	logPath := filepath.Join(dir, "logs", "api.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprint(f, "line-25 NEW\nline-26 NEW\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)

	m = updateModel(m, engine.Event{Kind: engine.EventTick})
	m = updateModel(m, keyMsg("G"))

	if !m.autoScroll {
		t.Fatal("expected auto-scroll to resume at bottom")
	}
	if m.frozenDisplayLines != nil {
		t.Fatal("expected frozen snapshot to be cleared")
	}

	view := ansi.Strip(mainPaneText(m))
	if !strings.Contains(view, "line-26 NEW") {
		t.Fatalf("expected appended line in view after resume, got:\n%s", view)
	}
	if strings.Contains(view, scrollHint) {
		t.Fatal("scroll hint should disappear after resume")
	}
}
