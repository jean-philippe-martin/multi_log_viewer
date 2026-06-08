package engine

// RowKind identifies a sidebar row type.
type RowKind int

const (
	RowService RowKind = iota
	RowFolder
	RowLog
)

// Key is a user key action.
type Key int

const (
	KeyUp Key = iota
	KeyDown
	KeyEnter
	KeyAdd
	KeyRemove
	KeyHelp
	KeyQuit
	KeyOther
)

// SidebarRow is one visible sidebar line.
type SidebarRow struct {
	Kind       RowKind
	Depth      int // 0=service, 1=folder or log under service, 2=log under folder
	ServiceID  string
	FolderKey  string
	LogID      string
	Label      string
	Liveness   string // ● ○ -
	Activity   int
	Open       bool
}

// SidebarSnapshot is the sidebar state for rendering.
type SidebarSnapshot struct {
	Rows   []SidebarRow
	Cursor int
	Width  int
}

// PaneKind is main pane content type.
type PaneKind int

const (
	PaneLog PaneKind = iota
	PaneService
)

// Pane is one main pane section.
type Pane struct {
	Kind      PaneKind
	LogID     string
	ServiceID string
}

// MainViewState is the main pane.
type MainViewState struct {
	Panes      []Pane
	LogLines   []MergedLine
	Status     string
	ShowHelp   bool
}

// MergedLine is one line in merged log view.
type MergedLine struct {
	Prefix string
	Text   string
}

// KeyResult describes effect of a keypress.
type KeyResult struct {
	Quit bool
}

// Event notifies the TUI of changes.
type Event struct {
	Kind EventKind
}

// EventKind classifies engine events.
type EventKind int

const (
	EventTick EventKind = iota
	EventTreeChanged
)
