package engine

import (
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/jpmartin/multi_log_viewer/internal/config"
	"github.com/jpmartin/multi_log_viewer/internal/health"
	"github.com/jpmartin/multi_log_viewer/internal/lineparse"
	"github.com/jpmartin/multi_log_viewer/internal/pattern"
	"github.com/jpmartin/multi_log_viewer/internal/tail"
)

// Engine coordinates tails, health, and UI state.
type Engine struct {
	cfg *config.Config

	mu          sync.RWMutex
	logs        map[string]*logState
	services    map[string]*serviceState
	rows        []rowRef
	cursor      int
	openLogs    map[string]bool
	openService string
	status      string
	showHelp    bool
	subscribers []chan Event
	stopCh      chan struct{}
}

type logState struct {
	id           string
	serviceID    string
	absPath      string
	relPath      string
	label        string
	folderKey    string
	folderLabel  string
	linePattern  string
	mergeOK      bool
	tail         *tail.File
	entry        config.LogEntry
}

type serviceState struct {
	id            string
	cfg           config.Service
	health        health.Result
	runningSince  *time.Time
	wasUp         bool
	synthetic []string
	healthAt  time.Time
}

type rowRef struct {
	kind      RowKind
	serviceID string
	folderKey string
	logID     string
}

// New creates and starts the engine.
func New(cfg *config.Config) (*Engine, error) {
	e := &Engine{
		cfg:      cfg,
		logs:     make(map[string]*logState),
		services: make(map[string]*serviceState),
		openLogs: make(map[string]bool),
		stopCh:   make(chan struct{}),
	}
	for _, svc := range cfg.Services {
		e.services[svc.ID] = &serviceState{id: svc.ID, cfg: svc}
		if err := e.discoverService(svc); err != nil {
			return nil, err
		}
	}
	e.rebuildRows()
	go e.loop()
	return e, nil
}

// Close shuts down background work.
func (e *Engine) Close() {
	close(e.stopCh)
	e.mu.Lock()
	for _, l := range e.logs {
		if l.tail != nil {
			l.tail.Close()
		}
	}
	e.mu.Unlock()
}

func (e *Engine) loop() {
	healthTick := time.NewTicker(time.Duration(e.cfg.HealthPollSeconds * float64(time.Second)))
	rescanTick := time.NewTicker(time.Duration(e.cfg.PatternRescanSeconds * float64(time.Second)))
	activityTick := time.NewTicker(100 * time.Millisecond)
	defer healthTick.Stop()
	defer rescanTick.Stop()
	defer activityTick.Stop()
	e.pollHealth()
	for {
		select {
		case <-e.stopCh:
			return
		case <-healthTick.C:
			e.pollHealth()
			e.notify(Event{Kind: EventTick})
		case <-rescanTick.C:
			e.rescanPatterns()
			e.rebuildRows()
			e.notify(Event{Kind: EventTreeChanged})
		case <-activityTick.C:
			e.mu.Lock()
			for _, l := range e.logs {
				if l.tail != nil {
					l.tail.TickActivity()
				}
			}
			e.mu.Unlock()
			e.notify(Event{Kind: EventTick})
		}
	}
}

func (e *Engine) notify(ev Event) {
	e.mu.RLock()
	subs := append([]chan Event(nil), e.subscribers...)
	e.mu.RUnlock()
	for _, ch := range subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

// Subscribe registers for engine events.
func (e *Engine) Subscribe(ch chan Event) {
	e.mu.Lock()
	e.subscribers = append(e.subscribers, ch)
	e.mu.Unlock()
}

func (e *Engine) discoverService(svc config.Service) error {
	for _, entry := range svc.Logs {
		if entry.Path != "" {
			abs := svc.ResolveLogPath(entry.Path)
			if err := config.EnsureLogFile(abs); err != nil {
				return err
			}
			id := logID(svc.ID, abs)
			if err := e.attachLog(id, svc.ID, abs, entry.Path, "", "", entry); err != nil {
				return err
			}
			continue
		}
		tmpl, err := pattern.Compile(entry.PathPattern)
		if err != nil {
			return err
		}
		paths, err := pattern.Discover(svc.BaseDir, tmpl)
		if err != nil {
			return err
		}
		for _, rel := range paths {
			caps, _ := tmpl.MatchCaptures(rel)
			fkey := pattern.FolderKey(caps, tmpl.CaptureNames)
			flabel := entry.Label
			if flabel != "" {
				flabel = pattern.ApplyLabel(flabel, caps)
			} else {
				flabel = pattern.DefaultFolderLabel(caps, tmpl.CaptureNames)
			}
			abs := filepath.Join(svc.BaseDir, filepath.FromSlash(rel))
			id := logID(svc.ID, abs)
			if err := e.attachLog(id, svc.ID, abs, rel, fkey, flabel, entry); err != nil {
				return err
			}
		}
	}
	return nil
}

func logID(serviceID, absPath string) string {
	return serviceID + "\x00" + absPath
}

func (e *Engine) attachLog(id, serviceID, abs, rel, folderKey, folderLabel string, entry config.LogEntry) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.logs[id]; ok {
		return nil
	}
	var lp *lineparse.Pattern
	if entry.LinePattern != "" {
		var err error
		lp, err = lineparse.Compile(entry.LinePattern)
		if err != nil {
			return fmt.Errorf("service '%s': %w", serviceID, err)
		}
	}
	t, err := tail.NewFile(abs, e.cfg.RingBufferLines, lp)
	if err != nil {
		return err
	}
	ls := &logState{
		id:          id,
		serviceID:   serviceID,
		absPath:     abs,
		relPath:     rel,
		label:       filepath.Base(abs),
		folderKey:   folderKey,
		folderLabel: folderLabel,
		linePattern: entry.LinePattern,
		mergeOK:     entry.LinePattern != "",
		tail:        t,
		entry:       entry,
	}
	e.logs[id] = ls
	return nil
}

// rescanPatterns re-glob path_pattern log entries, attaches new files, and
// stops tails for pattern-discovered logs that disappeared. Fixed path logs are
// not removed here.
func (e *Engine) rescanPatterns() {
	for _, svc := range e.cfg.Services {
		var want map[string]bool
		for _, entry := range svc.Logs {
			if entry.PathPattern == "" {
				continue
			}
			tmpl, _ := pattern.Compile(entry.PathPattern)
			paths, err := pattern.Discover(svc.BaseDir, tmpl)
			if err != nil {
				continue
			}
			if want == nil {
				want = make(map[string]bool)
			}
			for _, rel := range paths {
				abs := filepath.Join(svc.BaseDir, filepath.FromSlash(rel))
				want[logID(svc.ID, abs)] = true
				caps, _ := tmpl.MatchCaptures(rel)
				fkey := pattern.FolderKey(caps, tmpl.CaptureNames)
				flabel := entry.Label
				if flabel != "" {
					flabel = pattern.ApplyLabel(flabel, caps)
				} else {
					flabel = pattern.DefaultFolderLabel(caps, tmpl.CaptureNames)
				}
				_ = e.attachLog(logID(svc.ID, abs), svc.ID, abs, rel, fkey, flabel, entry)
			}
		}
		e.mu.Lock()
		for id, l := range e.logs {
			if l.serviceID != svc.ID {
				continue
			}
			if l.entry.PathPattern == "" {
				continue
			}
			if want != nil && !want[id] {
				if l.tail != nil {
					l.tail.Close()
				}
				delete(e.logs, id)
				delete(e.openLogs, id)
			}
		}
		e.mu.Unlock()
	}
}

func (e *Engine) pollHealth() {
	now := time.Now()
	e.mu.Lock()
	defer e.mu.Unlock()
	for id, ss := range e.services {
		hc := ss.cfg.Health
		r := health.Check(hc.Port, hc.ProcessContains, hc.HasProbes)
		ss.health = r
		if r.Up {
			if ss.runningSince == nil {
				t := now
				ss.runningSince = &t
			}
			if !ss.wasUp {
				ss.appendSynthetic(now, fmt.Sprintf("starts: service %s is up", id))
			} else if !ss.wasUp && ss.runningSince == nil {
				ss.appendSynthetic(now, fmt.Sprintf("running: service %s was already up", id))
			}
		} else {
			if ss.wasUp {
				ss.appendSynthetic(now, fmt.Sprintf("stops: service %s is down", id))
			}
			ss.runningSince = nil
		}
		if !ss.wasUp && r.Up && ss.synthetic == nil {
			ss.appendSynthetic(now, fmt.Sprintf("running: service %s was already up at start", id))
		}
		ss.wasUp = r.Up
		ss.healthAt = now
	}
}

func (ss *serviceState) appendSynthetic(at time.Time, msg string) {
	line := at.Format("2006-01-02 15:04:05.000") + " " + ss.id + " " + msg
	ss.synthetic = append(ss.synthetic, line)
	if len(ss.synthetic) > 500 {
		ss.synthetic = ss.synthetic[len(ss.synthetic)-500:]
	}
}

func (e *Engine) rebuildRows() {
	e.mu.Lock()
	defer e.mu.Unlock()
	var refs []rowRef
	for _, svc := range e.cfg.Services {
		refs = append(refs, rowRef{kind: RowService, serviceID: svc.ID})
		type folder struct {
			key, label string
			logs       []string
		}
		folders := map[string]*folder{}
		var bare []string
		for id, l := range e.logs {
			if l.serviceID != svc.ID {
				continue
			}
			if l.folderKey == "" {
				bare = append(bare, id)
				continue
			}
			f := folders[l.folderKey]
			if f == nil {
				f = &folder{key: l.folderKey, label: l.folderLabel}
				folders[l.folderKey] = f
			}
			f.logs = append(f.logs, id)
		}
		sort.Strings(bare)
		for _, id := range bare {
			refs = append(refs, rowRef{kind: RowLog, serviceID: svc.ID, logID: id})
		}
		fkeys := make([]string, 0, len(folders))
		for k := range folders {
			fkeys = append(fkeys, k)
		}
		sort.Strings(fkeys)
		for _, k := range fkeys {
			f := folders[k]
			refs = append(refs, rowRef{kind: RowFolder, serviceID: svc.ID, folderKey: f.key})
			sort.Strings(f.logs)
			for _, id := range f.logs {
				refs = append(refs, rowRef{kind: RowLog, serviceID: svc.ID, folderKey: f.key, logID: id})
			}
		}
	}
	e.rows = refs
	if e.cursor >= len(e.rows) {
		e.cursor = max(0, len(e.rows)-1)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (e *Engine) livenessSymbol(svcID string) string {
	ss := e.services[svcID]
	if ss == nil || !ss.cfg.Health.HasProbes {
		return "-"
	}
	if ss.health.Up {
		return "●"
	}
	return "○"
}

func (e *Engine) serviceActivity(svcID string) int {
	maxA := 0
	for _, l := range e.logs {
		if l.serviceID != svcID || l.tail == nil {
			continue
		}
		if a := l.tail.ActivityLevel(); a > maxA {
			maxA = a
		}
	}
	return maxA
}

func activityChar(level int) string {
	chars := []string{" ", "▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	if level < 0 {
		level = 0
	}
	if level >= len(chars) {
		level = len(chars) - 1
	}
	return chars[level]
}

// Sidebar returns current sidebar snapshot.
func (e *Engine) Sidebar() SidebarSnapshot {
	e.mu.RLock()
	defer e.mu.RUnlock()
	snap := SidebarSnapshot{Cursor: e.cursor, Width: 32}
	for _, ref := range e.rows {
		row := SidebarRow{Kind: ref.kind, ServiceID: ref.serviceID, FolderKey: ref.folderKey, LogID: ref.logID}
		switch ref.kind {
		case RowService:
			row.Depth = 0
			row.Label = ref.serviceID
			row.Liveness = e.livenessSymbol(ref.serviceID)
			row.Activity = e.serviceActivity(ref.serviceID)
			row.Open = e.openService == ref.serviceID
		case RowFolder:
			row.Depth = 1
			row.Label = e.folderLabel(ref.serviceID, ref.folderKey)
			row.Activity = e.folderActivity(ref.serviceID, ref.folderKey)
		case RowLog:
			if ref.folderKey == "" {
				row.Depth = 1
			} else {
				row.Depth = 2
			}
			l := e.logs[ref.logID]
			row.Label = l.label
			if l.tail != nil {
				row.Activity = l.tail.ActivityLevel()
			}
			row.Open = e.openLogs[ref.logID]
		}
		snap.Rows = append(snap.Rows, row)
	}
	return snap
}

func (e *Engine) folderLabel(svcID, fkey string) string {
	for _, l := range e.logs {
		if l.serviceID == svcID && l.folderKey == fkey {
			return l.folderLabel
		}
	}
	return fkey
}

func (e *Engine) folderActivity(svcID, fkey string) int {
	maxA := 0
	for _, l := range e.logs {
		if l.serviceID == svcID && l.folderKey == fkey && l.tail != nil {
			if a := l.tail.ActivityLevel(); a > maxA {
				maxA = a
			}
		}
	}
	return maxA
}

// MainView returns main pane state.
func (e *Engine) MainView() MainViewState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	mv := MainViewState{Status: e.status, ShowHelp: e.showHelp}
	if e.openService != "" {
		mv.Panes = append(mv.Panes, Pane{Kind: PaneService, ServiceID: e.openService})
		return mv
	}
	for id := range e.openLogs {
		if e.openLogs[id] {
			mv.Panes = append(mv.Panes, Pane{Kind: PaneLog, LogID: id})
		}
	}
	sort.Slice(mv.Panes, func(i, j int) bool { return mv.Panes[i].LogID < mv.Panes[j].LogID })
	mv.LogLines = e.buildMergedLines()
	return mv
}

// buildMergedLines merges open logs for the main pane. Lines with parsed
// timestamps are sorted across all open logs (ts, service, path, seq). Lines
// without timestamps are included only when a single log is open; they are
// omitted from multi-log merges.
func (e *Engine) buildMergedLines() []MergedLine {
	type item struct {
		ts      time.Time
		svc     string
		path    string
		seq     uint64
		text    string
		logID   string
	}
	var items []item
	for id := range e.openLogs {
		if !e.openLogs[id] {
			continue
		}
		l := e.logs[id]
		if l == nil || l.tail == nil {
			continue
		}
		for _, ln := range l.tail.Lines() {
			if !ln.HasTS {
				continue
			}
			items = append(items, item{
				ts: ln.Timestamp, svc: l.serviceID, path: l.label,
				seq: ln.Seq, text: ln.Text, logID: id,
			})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if !items[i].ts.Equal(items[j].ts) {
			return items[i].ts.Before(items[j].ts)
		}
		if items[i].svc != items[j].svc {
			return items[i].svc < items[j].svc
		}
		if items[i].path != items[j].path {
			return items[i].path < items[j].path
		}
		return items[i].seq < items[j].seq
	})
	var out []MergedLine
	for _, it := range items {
		prefix := ""
		if len(e.openLogs) > 1 {
			prefix = fmt.Sprintf("[%s/%s] ", it.svc, it.path)
		}
		out = append(out, MergedLine{Prefix: prefix, Text: it.text})
	}
	if len(e.openLogs) == 1 {
		for id := range e.openLogs {
			l := e.logs[id]
			if l == nil || l.tail == nil {
				continue
			}
			for _, ln := range l.tail.Lines() {
				if ln.HasTS {
					continue
				}
				out = append(out, MergedLine{Text: ln.Text})
			}
		}
	}
	return out
}

// HandleKey processes a key.
func (e *Engine) HandleKey(k Key) KeyResult {
	if k == KeyHelp {
		e.mu.Lock()
		e.showHelp = !e.showHelp
		e.mu.Unlock()
		return KeyResult{}
	}
	if k == KeyQuit {
		return KeyResult{Quit: true}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status = ""
	if k == KeyUp && e.cursor > 0 {
		e.cursor--
	}
	if k == KeyDown && e.cursor < len(e.rows)-1 {
		e.cursor++
	}
	if len(e.rows) == 0 {
		return KeyResult{}
	}
	ref := e.rows[e.cursor]
	switch k {
	case KeyEnter:
		e.handleEnter(ref)
	case KeyAdd:
		e.handleAdd(ref)
	case KeyRemove:
		e.handleRemove(ref)
	}
	return KeyResult{}
}

func (e *Engine) handleEnter(ref rowRef) {
	e.openLogs = make(map[string]bool)
	e.openService = ""
	switch ref.kind {
	case RowLog:
		e.openLogs[ref.logID] = true
	case RowService:
		e.openService = ref.serviceID
	case RowFolder:
		e.status = "select a log file"
	}
}

func (e *Engine) handleAdd(ref rowRef) {
	switch ref.kind {
	case RowLog:
		l := e.logs[ref.logID]
		if l == nil || l.linePattern == "" {
			e.status = "can only combine logs with parsed timestamps"
			return
		}
		if l.tail != nil && !l.tail.HasParsedTimestamp() {
			e.status = "can only combine logs with parsed timestamps"
			return
		}
		e.openService = ""
		e.openLogs[ref.logID] = true
	case RowService:
		e.status = "use Enter to show service"
	case RowFolder:
		e.status = "select a log file"
	}
}

func (e *Engine) handleRemove(ref rowRef) {
	switch ref.kind {
	case RowLog:
		delete(e.openLogs, ref.logID)
	case RowService:
		if e.openService == ref.serviceID {
			e.openService = ""
		}
	case RowFolder:
	}
}

// ServiceDetailLines returns text lines for service pane.
func (e *Engine) ServiceDetailLines(serviceID string) []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	ss := e.services[serviceID]
	if ss == nil {
		return nil
	}
	var lines []string
	liv := e.livenessSymbol(serviceID)
	lines = append(lines, liv+" "+serviceID+" "+activityChar(e.serviceActivity(serviceID)))
	lines = append(lines, "")
	lines = append(lines, health.FormatProbeLine(ss.health)...)
	if ss.health.Up && ss.runningSince != nil {
		lines = append(lines, "  running since: before "+ss.runningSince.Format("2006-1-2 15:04"))
	}
	r1, r10 := e.lineRates(serviceID)
	lines = append(lines, activityChar(e.serviceActivity(serviceID))+" 1 min activity:  "+formatRate(r1))
	lines = append(lines, activityChar(e.serviceActivity(serviceID))+" 10min activity: "+formatRate(r10))
	lines = append(lines, "")
	if ss.health.PID > 0 {
		lines = append(lines, "  [ KILL ]")
		lines = append(lines, "")
	}
	start := max(0, len(ss.synthetic)-8)
	lines = append(lines, ss.synthetic[start:]...)
	return lines
}

func formatRate(r float64) string {
	return fmt.Sprintf("%.1f lines/s", r)
}

func (e *Engine) lineRates(serviceID string) (float64, float64) {
	now := time.Now()
	var times []time.Time
	for _, l := range e.logs {
		if l.serviceID != serviceID || l.tail == nil {
			continue
		}
		for _, ln := range l.tail.Lines() {
			times = append(times, time.Now()) // approximate: count lines in buffer
			_ = ln
		}
	}
	_ = times
	// Count lines ingested - use line count in buffer as proxy
	n := 0
	for _, l := range e.logs {
		if l.serviceID == serviceID && l.tail != nil {
			n += len(l.tail.Lines())
		}
	}
	_ = now
	return float64(n) / 60.0, float64(n) / 600.0
}

// ActivityChar exports activity glyph.
func ActivityChar(level int) string {
	return activityChar(level)
}
