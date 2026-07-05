package tui

import (
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jpmartin/multi_log_viewer/internal/engine"
	"github.com/jpmartin/multi_log_viewer/internal/theme"
	"github.com/muesli/termenv"
)

type focusPane int

const (
	focusLeft focusPane = iota
	focusMain
)

const scrollHint = "Scroll to the bottom to resume auto-scroll."

type model struct {
	eng           *engine.Engine
	th            *theme.Theme
	events        chan engine.Event
	width         int
	height        int
	focus         focusPane
	autoScroll          bool
	viewStart           int
	frozenDisplayLines  []string // snapshot taken when auto-scroll pauses; not refreshed until resume
}

func initColorProfile() {
	prof := termenv.NewOutput(os.Stdout).ColorProfile()
	if prof == termenv.Ascii {
		prof = termenv.ANSI256
	}
	lipgloss.SetColorProfile(prof)
}

// Run starts the TUI.
func Run(eng *engine.Engine, th *theme.Theme) error {
	initColorProfile()
	ch := make(chan engine.Event, 64)
	eng.Subscribe(ch)
	if th == nil {
		var err error
		th, err = theme.Load("default", ".")
		if err != nil {
			return err
		}
	}
	m := model{eng: eng, th: th, events: ch, focus: focusLeft, autoScroll: true}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), waitEvent(m.events))
}

func tickCmd() tea.Cmd {
	return tea.Tick(100, func(time.Time) tea.Msg { return tickMsg{} })
}

func waitEvent(ch <-chan engine.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return ev
	}
}

type tickMsg struct{}

func (m model) mainInnerH(paneH int) int {
	innerH := paneH - 2
	if innerH < 1 {
		innerH = 1
	}
	return innerH
}

func (m model) mainContentH(paneH int) int {
	innerH := m.mainInnerH(paneH)
	if !m.autoScroll {
		return innerH - 1
	}
	return innerH
}

// buildMainLines reads the current engine state, wraps log text to mainW columns,
// and returns styled strings ready to paint (one entry per terminal row).
func (m model) buildMainLines(mainW, paneH int) []string {
	mv := m.eng.MainView()
	logStyle := m.th.LogText.Lipgloss()
	mutedStyle := m.th.MutedStyle()
	if len(mv.Panes) == 0 {
		return nil
	}
	for _, p := range mv.Panes {
		if p.Kind == engine.PaneService {
			return styleWrappedLines(m.eng.ServiceDetailLines(p.ServiceID), mainW, logStyle)
		}
	}
	var out []string
	for _, ln := range mv.LogLines {
		plain := ln.Prefix + ln.Text
		for j, seg := range strings.Split(wrapLine(plain, mainW), "\n") {
			if seg == "" && j > 0 {
				continue
			}
			if ln.Prefix != "" && j == 0 && strings.HasPrefix(seg, ln.Prefix) {
				out = append(out, mutedStyle.Render(ln.Prefix)+logStyle.Render(strings.TrimPrefix(seg, ln.Prefix)))
				continue
			}
			out = append(out, logStyle.Render(seg))
		}
	}
	return out
}

// mainDisplayLines returns the line slice used for scrolling and rendering.
// While auto-scroll is on, lines are rebuilt from the live tail on every call.
// While paused, the frozen snapshot taken at pause time is returned instead.
func (m model) mainDisplayLines(mainW, paneH int) []string {
	if !m.autoScroll && m.frozenDisplayLines != nil {
		return m.frozenDisplayLines
	}
	return m.buildMainLines(mainW, paneH)
}

func (m model) pauseAutoScroll(mainW, paneH int) model {
	m.frozenDisplayLines = append([]string(nil), m.buildMainLines(mainW, paneH)...)
	return m
}

func (m model) resumeAutoScroll() model {
	m.autoScroll = true
	m.viewStart = 0
	m.frozenDisplayLines = nil
	return m
}

func styleWrappedLines(lines []string, width int, style lipgloss.Style) []string {
	wrapped := flattenWrappedLines(lines, width)
	for i, ln := range wrapped {
		wrapped[i] = style.Render(ln)
	}
	return wrapped
}

func (m model) scrollUp(n, paneH int) model {
	mainW := m.mainContentWidth()
	innerH := m.mainInnerH(paneH)
	if m.autoScroll {
		m = m.pauseAutoScroll(mainW, paneH)
		if len(m.frozenDisplayLines) == 0 {
			return m
		}
		m.autoScroll = false
		m.viewStart = maxViewStart(len(m.frozenDisplayLines), innerH) - n
	} else {
		lines := m.mainDisplayLines(mainW, paneH)
		if len(lines) == 0 {
			return m
		}
		m.viewStart -= n
		maxStart := maxViewStart(len(lines), m.mainContentH(paneH))
		if m.viewStart > maxStart {
			m.viewStart = maxStart
		}
	}
	if m.viewStart < 0 {
		m.viewStart = 0
	}
	return m
}

func (m model) scrollDown(n, paneH int) model {
	lines := m.mainDisplayLines(m.mainContentWidth(), paneH)
	if len(lines) == 0 {
		return m
	}
	contentH := m.mainContentH(paneH)
	m.viewStart += n
	maxStart := maxViewStart(len(lines), contentH)
	if m.viewStart >= maxStart {
		m = m.resumeAutoScroll()
	}
	return m
}

func (m model) scrollToTop(paneH int) model {
	mainW := m.mainContentWidth()
	m = m.pauseAutoScroll(mainW, paneH)
	if len(m.frozenDisplayLines) == 0 {
		return m
	}
	m.autoScroll = false
	m.viewStart = 0
	return m
}

func (m model) scrollToBottom() model {
	return m.resumeAutoScroll()
}

func (m model) mainContentWidth() int {
	_, mainInner := splitPaneWidths(m.width)
	return mainInner
}

func (m model) paneHeight() int {
	paneH := m.height - 1
	if paneH < 5 {
		paneH = 5
	}
	if m.height > 0 && paneH > m.height-1 {
		paneH = m.height - 1
	}
	return paneH
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	paneH := m.paneHeight()
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.eng.MainView().ShowHelp {
			m.eng.HandleKey(engine.KeyHelp)
			return m, nil
		}
		key := msg.String()
		switch key {
		case "q", "ctrl+c":
			if res := m.eng.HandleKey(engine.KeyQuit); res.Quit {
				return m, tea.Quit
			}
		case "?", "shift+/":
			m.eng.HandleKey(engine.KeyHelp)
		case "right":
			if m.focus == focusLeft {
				m.focus = focusMain
			}
		case "left":
			if m.focus == focusMain {
				m.focus = focusLeft
			}
		case "up", "k":
			if m.focus == focusMain {
				m = m.scrollUp(1, paneH)
			} else {
				m.eng.HandleKey(engine.KeyUp)
			}
		case "down", "j":
			if m.focus == focusMain {
				m = m.scrollDown(1, paneH)
			} else {
				m.eng.HandleKey(engine.KeyDown)
			}
		case "pgup":
			if m.focus == focusMain {
				m = m.scrollUp(m.mainInnerH(paneH), paneH)
			}
		case "pgdown":
			if m.focus == focusMain {
				m = m.scrollDown(m.mainInnerH(paneH), paneH)
			}
		case "home", "g":
			if m.focus == focusMain {
				m = m.scrollToTop(paneH)
			}
		case "end":
			if m.focus == focusMain {
				m = m.scrollToBottom()
			}
		case "G":
			if m.focus == focusMain {
				m = m.scrollToBottom()
			}
		case "enter":
			m.eng.HandleKey(engine.KeyEnter)
			m = m.resumeAutoScroll()
		case "+":
			m.eng.HandleKey(engine.KeyAdd)
			m = m.resumeAutoScroll()
		case "-":
			m.eng.HandleKey(engine.KeyRemove)
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tickMsg:
		return m, tea.Batch(tickCmd(), waitEvent(m.events))
	case engine.Event:
		return m, tea.Batch(waitEvent(m.events))
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}
	paneH := m.paneHeight()
	innerH := m.mainInnerH(paneH)
	sideInner, mainInner := splitPaneWidths(m.width)

	leftFocused := m.focus == focusLeft
	mainFocused := m.focus == focusMain
	sidebar := renderSidebarContent(m.eng.Sidebar(), sideInner, innerH, leftFocused, m.th)
	main := m.renderMainContent(mainInner, innerH)
	body := renderSplitPane(sidebar, main, sideInner, mainInner, innerH, leftFocused, mainFocused, m.th)
	status := m.renderStatus(m.width)
	out := body + "\n" + status

	mv := m.eng.MainView()
	if mv.ShowHelp {
		overlay := m.th.HelpStyle().Width(m.width - 4).Render(helpText())
		out = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
	}
	return out
}

func helpText() string {
	return `j/k ↑↓  move / scroll
←/→     switch pane focus
Enter   show only
+       add log to merge
-       remove from view
PgUp/Dn page scroll (main pane)
g/G     top/bottom (main pane)
?       help
q       quit

(press any key to close)`
}

func (m model) renderMainContent(mainInner, innerH int) string {
	mv := m.eng.MainView()
	if len(mv.Panes) == 0 {
		return padLines(m.th.MutedStyle().Render("(no selection — press Enter on a log or service)"), innerH)
	}

	displayLines := m.mainDisplayLines(mainInner, m.paneHeight())
	visible, showHint := visibleMainLines(displayLines, innerH, m.autoScroll, m.viewStart)
	text := strings.Join(visible, "\n")
	if showHint {
		hint := scrollHint
		if len(hint) > mainInner {
			hint = hint[:mainInner]
		}
		return padLines(text, innerH-1) + "\n" + m.th.LogScrollNoticeStyle().Render(hint)
	}
	return padLines(text, innerH)
}

func defaultStatusText(focus focusPane) string {
	switch focus {
	case focusMain:
		return "↑↓ scroll  PgUp/Dn page  Home/End ends  ←/→ panes  ? help"
	default:
		return "↑↓ move  Enter show  + add  ←/→ panes  q quit  ? help"
	}
}

func (m model) renderStatus(w int) string {
	mv := m.eng.MainView()
	s := mv.Status
	if s == "" {
		s = defaultStatusText(m.focus)
	}
	return m.th.LeftText.Lipgloss().Width(w).Render(s)
}
