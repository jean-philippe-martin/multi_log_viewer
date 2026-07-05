package tui

import "strings"

// clipLines keeps at most maxLines, preferring the end (tail).
func clipLines(s string, maxLines int) string {
	if maxLines < 1 {
		maxLines = 1
	}
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		lines = nil
	}
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return padLines(strings.Join(lines, "\n"), maxLines)
}

// padLines ensures exactly maxLines lines.
func padLines(s string, maxLines int) string {
	if maxLines < 1 {
		maxLines = 1
	}
	var lines []string
	if s != "" {
		lines = strings.Split(strings.TrimRight(s, "\n"), "\n")
	}
	for len(lines) < maxLines {
		lines = append(lines, "")
	}
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, "\n")
}

func wrapLine(s string, w int) string {
	if len(s) <= w {
		return s
	}
	var out []string
	for len(s) > w {
		out = append(out, s[:w])
		s = s[w:]
	}
	if s != "" {
		out = append(out, s)
	}
	return strings.Join(out, "\n")
}

// flattenWrappedLines turns content lines into display lines after width wrapping.
func flattenWrappedLines(lines []string, width int) []string {
	if width < 1 {
		width = 1
	}
	var out []string
	for _, ln := range lines {
		out = append(out, strings.Split(wrapLine(ln, width), "\n")...)
	}
	return out
}

func maxViewStart(total, contentH int) int {
	if total <= contentH {
		return 0
	}
	return total - contentH
}

// visibleMainLines returns the slice of displayLines to paint in the main pane,
// and whether the paused-auto-scroll hint should occupy the bottom row.
// displayLines is either live tail content (auto-scroll) or a frozen snapshot (paused).
func visibleMainLines(displayLines []string, paneLines int, followTail bool, viewStart int) ([]string, bool) {
	if paneLines < 1 {
		paneLines = 1
	}

	contentLines := paneLines
	if !followTail {
		contentLines = paneLines - 1 // reserve bottom row for scroll hint
		if contentLines < 1 {
			contentLines = 1
		}
	}

	total := len(displayLines)
	start := viewStart
	if followTail {
		start = maxViewStart(total, contentLines)
	} else if start < 0 {
		start = 0
	} else if start > maxViewStart(total, contentLines) {
		start = maxViewStart(total, contentLines)
	}

	end := start + contentLines
	if end > total {
		end = total
	}
	visible := append([]string(nil), displayLines[start:end]...)
	for len(visible) < contentLines {
		visible = append(visible, "")
	}

	showHint := !followTail && start < maxViewStart(total, contentLines)
	return visible, showHint
}

// viewportStart picks the first visible sidebar row so the cursor stays in view.
// When the cursor is in the first "page" of rows, always start at 0 so service
// header rows are not scrolled away (fixes missing top row in short panes / Zellij).
func viewportStart(cursor, total, visible int) int {
	if visible < 1 {
		visible = 1
	}
	if total <= visible || visible <= 1 {
		return 0
	}
	if cursor < visible {
		return 0
	}
	start := cursor - visible + 1
	if start > total-visible {
		return total - visible
	}
	return start
}
