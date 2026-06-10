package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/jpmartin/multi_log_viewer/internal/engine"
	"github.com/jpmartin/multi_log_viewer/internal/theme"
)

func TestSplitPaneWidths(t *testing.T) {
	side, main := splitPaneWidths(80)
	if side+main != 77 {
		t.Fatalf("side=%d main=%d sum=%d want 77", side, main, side+main)
	}
}

func TestRenderSplitPaneSingleDivider(t *testing.T) {
	th, err := theme.Parse([]byte(`border: "240"`))
	if err != nil {
		t.Fatal(err)
	}
	out := renderSplitPane("left", "right", 4, 5, 1, false, false, th)
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("lines=%d", len(lines))
	}
	plain := ansi.Strip(lines[1])
	if strings.Count(plain, "│") != 3 {
		t.Fatalf("want 3 vertical bars, got %q", plain)
	}
	if strings.Count(plain, "left") != 1 || strings.Count(plain, "right") != 1 {
		t.Fatalf("content row: %q", plain)
	}
}

func TestRenderSplitPaneStyledSidebarAligns(t *testing.T) {
	th, err := theme.Parse([]byte(`
left_text: bright-green
selected_left_text:
  fg: bright-white
  bg: bright-blue
  bold: true
border: green
`))
	if err != nil {
		t.Fatal(err)
	}
	sideInner := 20
	mainInner := 30
	snap := engine.SidebarSnapshot{
		Cursor: 0,
		Rows: []engine.SidebarRow{
			{Kind: engine.RowService, Depth: 0, Label: "api", Liveness: "○"},
		},
	}
	sidebar := renderSidebarContent(snap, sideInner, 1, true, th)
	main := th.LogText.Lipgloss().Render("hello")
	out := renderSplitPane(sidebar, main, sideInner, mainInner, 1, true, false, th)
	row := strings.Split(out, "\n")[1]
	want := 1 + sideInner + 1 + mainInner + 1
	if ansi.StringWidth(row) != want {
		t.Fatalf("row width=%d want %d row=%q", ansi.StringWidth(row), want, ansi.Strip(row))
	}
}

func TestFitLineStyledWidth(t *testing.T) {
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("hello")
	got := fitLine(styled, 10)
	if ansi.StringWidth(got) != 10 {
		t.Fatalf("width=%d got=%q", ansi.StringWidth(got), got)
	}
}

func TestRenderSplitTopDoubleWhenLeftFocused(t *testing.T) {
	th, err := theme.Parse([]byte(`
border: "240"
selected_top_border: "252"
`))
	if err != nil {
		t.Fatal(err)
	}
	border := th.BorderFgStyle()
	leftTop := th.TopBorderFgStyle(true)
	rightTop := th.TopBorderFgStyle(false)
	top := ansi.Strip(renderSplitTop(4, 5, true, false, leftTop, rightTop, border))
	want := "╒════╕─────┐"
	if top != want {
		t.Fatalf("top=%q want %q", top, want)
	}
}

func TestRenderSplitTopDoubleWhenMainFocused(t *testing.T) {
	th, err := theme.Parse([]byte(`
border: "240"
selected_top_border: "252"
`))
	if err != nil {
		t.Fatal(err)
	}
	border := th.BorderFgStyle()
	leftTop := th.TopBorderFgStyle(false)
	rightTop := th.TopBorderFgStyle(true)
	top := ansi.Strip(renderSplitTop(4, 5, false, true, leftTop, rightTop, border))
	want := "┌────╒═════╕"
	if top != want {
		t.Fatalf("top=%q want %q", top, want)
	}
}
