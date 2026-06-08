package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/jpmartin/multi_log_viewer/internal/theme"
)

// splitPaneWidths returns inner content widths for sidebar and main panes.
// totalW is the full terminal width; three columns are reserved for the outer
// left border, center divider, and outer right border.
func splitPaneWidths(totalW int) (sideInner, mainInner int) {
	if totalW < 15 {
		totalW = 15
	}
	innerTotal := totalW - 3
	sideInner = innerTotal * 30 / 100
	if sideInner < 22 {
		sideInner = 22
	}
	if sideInner > 46 {
		sideInner = 46
	}
	mainInner = innerTotal - sideInner
	if mainInner < 8 {
		mainInner = 8
		if sideInner > innerTotal-8 {
			sideInner = innerTotal - 8
		}
	}
	return sideInner, mainInner
}

func renderSplitPane(sidebarText, mainText string, sideInner, mainInner, innerH int, leftFocused, mainFocused bool, th *theme.Theme) string {
	sideLines := padLines(sidebarText, innerH)
	mainLines := padLines(mainText, innerH)
	sideRows := strings.Split(sideLines, "\n")
	mainRows := strings.Split(mainLines, "\n")

	border := th.BorderFgStyle()
	leftTop := th.TopBorderFgStyle(leftFocused)
	rightTop := th.TopBorderFgStyle(mainFocused)

	var b strings.Builder
	b.WriteString(renderSplitTop(sideInner, mainInner, leftFocused, mainFocused, leftTop, rightTop, border))
	b.WriteByte('\n')
	for i := 0; i < innerH; i++ {
		b.WriteString(border.Render("│"))
		b.WriteString(fitLine(sideRows[i], sideInner))
		b.WriteString(border.Render("│"))
		b.WriteString(fitLine(mainRows[i], mainInner))
		b.WriteString(border.Render("│"))
		if i < innerH-1 {
			b.WriteByte('\n')
		}
	}
	b.WriteByte('\n')
	b.WriteString(renderSplitBottom(sideInner, mainInner, border))
	return b.String()
}

func renderSplitTop(sideInner, mainInner int, leftFocused, mainFocused bool, leftTop, rightTop, border lipgloss.Style) string {
	leftCorner, leftBar := "┌", "─"
	rightCorner, rightBar := "┐", "─"
	junction := "┬"

	if leftFocused {
		leftCorner, leftBar = "╔", "═"
		if mainFocused {
			junction, rightCorner, rightBar = "╦", "╗", "═"
		} else {
			junction, rightCorner = "╤", "┐"
		}
	} else if mainFocused {
		rightCorner, rightBar = "╗", "═"
		junction = "╥"
	}

	left := leftTop.Render(leftCorner + strings.Repeat(leftBar, sideInner))
	right := rightTop.Render(strings.Repeat(rightBar, mainInner) + rightCorner)
	return left + border.Render(junction) + right
}

func renderSplitBottom(sideInner, mainInner int, border lipgloss.Style) string {
	return border.Render("└" + strings.Repeat("─", sideInner) + "┴" + strings.Repeat("─", mainInner) + "┘")
}

func fitLine(s string, w int) string {
	if w < 0 {
		w = 0
	}
	sw := ansi.StringWidth(s)
	if sw > w {
		return ansi.Truncate(s, w, "")
	}
	if sw < w {
		return s + strings.Repeat(" ", w-sw)
	}
	return s
}
