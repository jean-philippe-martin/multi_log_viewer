package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/jpmartin/multi_log_viewer/internal/engine"
)

func TestSidebarTreeIndent(t *testing.T) {
	snap := engine.SidebarSnapshot{
		Rows: []engine.SidebarRow{
			{Kind: engine.RowService, Depth: 0, Label: "api", Liveness: "○"},
			{Kind: engine.RowLog, Depth: 1, Label: "log.txt"},
			{Kind: engine.RowService, Depth: 0, Label: "workers", Liveness: "●", Activity: 2},
			{Kind: engine.RowFolder, Depth: 1, Label: "worker_1", Activity: 7},
			{Kind: engine.RowLog, Depth: 2, Label: "stdout.txt", Activity: 7},
		},
	}
	bodyW := 24
	line := formatSidebarRow(snap.Rows[1], bodyW)
	if !strings.HasPrefix(line, "   - log.txt") {
		t.Fatalf("log under service: %q", line)
	}
	line = formatSidebarRow(snap.Rows[3], bodyW)
	if !strings.HasPrefix(line, "   ▷ worker_1") {
		t.Fatalf("folder: %q", line)
	}
	line = formatSidebarRow(snap.Rows[4], bodyW)
	if !strings.HasPrefix(line, "     - stdout.txt") {
		t.Fatalf("log under folder: %q", line)
	}
}

func TestOpenMarkerFixedColumn(t *testing.T) {
	const w = 20
	openRow := formatSidebarRow(engine.SidebarRow{Kind: engine.RowLog, Depth: 1, Label: "log.txt", Open: true}, w)
	closedRow := formatSidebarRow(engine.SidebarRow{Kind: engine.RowService, Depth: 0, Label: "api", Liveness: "○"}, w)
	if ansi.StringWidth(openRow) != w || openRow[len(openRow)-1] != '>' {
		t.Fatalf("open row: %q", openRow)
	}
	if ansi.StringWidth(closedRow) != w || closedRow[len(closedRow)-1] != ' ' {
		t.Fatalf("closed row: %q", closedRow)
	}
}

func TestFormatSidebarRowActivityWidth(t *testing.T) {
	const w = 24
	withPulse := formatSidebarRow(engine.SidebarRow{
		Kind: engine.RowService, Depth: 0, Label: "api", Liveness: "○", Activity: 7,
	}, w)
	withoutPulse := formatSidebarRow(engine.SidebarRow{
		Kind: engine.RowService, Depth: 0, Label: "api", Liveness: "○", Activity: 0,
	}, w)
	if ansi.StringWidth(withPulse) != w || ansi.StringWidth(withoutPulse) != w {
		t.Fatalf("visible: pulse=%d empty=%d want %d", ansi.StringWidth(withPulse), ansi.StringWidth(withoutPulse), w)
	}
}

func TestSidebarIndentDepths(t *testing.T) {
	if sidebarIndent(0) != "" {
		t.Fatal("depth 0")
	}
	if sidebarIndent(1) != "   " {
		t.Fatalf("depth 1: %q", sidebarIndent(1))
	}
	if sidebarIndent(2) != "     " {
		t.Fatalf("depth 2: %q", sidebarIndent(2))
	}
}
