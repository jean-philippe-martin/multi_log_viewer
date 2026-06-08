package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/jpmartin/multi_log_viewer/internal/engine"
	"github.com/jpmartin/multi_log_viewer/internal/theme"
)

// sidebarIndent returns leading spaces for tree depth (matches docs mockup).
func sidebarIndent(depth int) string {
	if depth <= 0 {
		return ""
	}
	// depth 1 → 3 spaces, depth 2 → 5 spaces, ...
	return strings.Repeat(" ", 1+depth*2)
}

func sidebarRowContent(row engine.SidebarRow) string {
	act := engine.ActivityChar(row.Activity)
	switch row.Kind {
	case engine.RowService:
		liv := row.Liveness
		if liv == "" {
			liv = "-"
		}
		return liv + "  " + row.Label + " " + act
	case engine.RowFolder:
		return "▷ " + row.Label + " " + act
	case engine.RowLog:
		return "- " + row.Label + " " + act
	default:
		return row.Label
	}
}

// formatSidebarRow builds the body of one sidebar line (no cursor prefix).
// lineWidth is the full inner width minus the cursor column; the last character
// is the open-in-main-pane marker (">" or " ").
func formatSidebarRow(row engine.SidebarRow, lineWidth int) string {
	if lineWidth < 2 {
		lineWidth = 2
	}
	openMark := " "
	if row.Open {
		openMark = ">"
	}
	indent := sidebarIndent(row.Depth)
	content := sidebarRowContent(row)
	indentW := ansi.StringWidth(indent)
	avail := lineWidth - indentW - 1 // reserve last column for open marker
	if avail < 0 {
		avail = 0
	}
	if ansi.StringWidth(content) > avail {
		content = ansi.Truncate(content, avail, "")
	}
	pad := lineWidth - indentW - ansi.StringWidth(content) - 1
	if pad < 0 {
		pad = 0
	}
	return indent + content + strings.Repeat(" ", pad) + openMark
}

func renderSidebarLines(snap engine.SidebarSnapshot, innerW, maxLines, rowStart int, showCursor bool, th *theme.Theme) string {
	leftStyle := th.LeftText.Lipgloss()
	selectedStyle := th.SelectedLeftText.Lipgloss()
	lineWidth := innerW - 1 // first column is cursor
	if lineWidth < 2 {
		lineWidth = 2
	}
	var b strings.Builder
	end := rowStart + maxLines
	if end > len(snap.Rows) {
		end = len(snap.Rows)
	}
	for i := rowStart; i < end; i++ {
		body := formatSidebarRow(snap.Rows[i], lineWidth)
		// Keep open marker outside lipgloss styles so it stays in the last column.
		contentPart := body[:len(body)-1]
		openMark := body[len(body)-1:]
		// Highlight spans a fixed width so the background does not shrink/grow
		// when the activity glyph (or other runes) changes visible width.
		highlightW := innerW - 1
		var line string
		if showCursor && i == snap.Cursor {
			line = selectedStyle.Width(highlightW).Render(fitLine(">"+contentPart, highlightW)) + openMark
		} else {
			line = leftStyle.Width(highlightW).Render(fitLine(" "+contentPart, highlightW)) + openMark
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return padLines(strings.TrimRight(b.String(), "\n"), maxLines)
}

func renderSidebarContent(snap engine.SidebarSnapshot, innerW, innerH int, showCursor bool, th *theme.Theme) string {
	if innerH < 1 {
		innerH = 1
	}
	if innerW < 4 {
		innerW = 4
	}
	start := viewportStart(snap.Cursor, len(snap.Rows), innerH)
	return padLines(renderSidebarLines(snap, innerW, innerH, start, showCursor, th), innerH)
}
