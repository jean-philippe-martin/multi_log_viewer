package tail

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jpmartin/multi_log_viewer/internal/lineparse"
)

const maxInitialRead = 2 << 20

// Line is one log line in the ring buffer.
type Line struct {
	Seq       uint64
	Text      string
	Fields    map[string]interface{}
	HasTS     bool
	Timestamp time.Time
}

// File tails one log file.
type File struct {
	Path     string
	MaxLines int
	Pattern  *lineparse.Pattern
	OnLine   func()

	mu       sync.RWMutex
	lines    []Line
	seq      uint64
	activity int
	readOff  int64
	stopCh   chan struct{}
	watcher  *fsnotify.Watcher
	poll     *time.Ticker
}

// NewFile starts tailing path.
func NewFile(path string, maxLines int, pattern *lineparse.Pattern) (*File, error) {
	f := &File{
		Path:     path,
		MaxLines: maxLines,
		Pattern:  pattern,
		stopCh:   make(chan struct{}),
	}
	if err := f.readInitial(); err != nil {
		return nil, err
	}
	w, err := fsnotify.NewWatcher()
	if err == nil {
		f.watcher = w
		_ = w.Add(filepath.Dir(path))
		go f.watchLoop()
	} else {
		f.poll = time.NewTicker(time.Second)
		go f.pollLoop()
	}
	return f, nil
}

// Close stops tailing.
func (f *File) Close() {
	select {
	case <-f.stopCh:
	default:
		close(f.stopCh)
	}
	if f.watcher != nil {
		_ = f.watcher.Close()
	}
	if f.poll != nil {
		f.poll.Stop()
	}
}

// Lines returns a copy of buffered lines.
func (f *File) Lines() []Line {
	f.mu.RLock()
	defer f.mu.RUnlock()
	out := make([]Line, len(f.lines))
	copy(out, f.lines)
	return out
}

// ActivityLevel returns 0-7.
func (f *File) ActivityLevel() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.activity
}

// TickActivity decays activity one step.
func (f *File) TickActivity() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.activity > 0 {
		f.activity--
	}
}

// HasParsedTimestamp reports if any line has a timestamp.
func (f *File) HasParsedTimestamp() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, ln := range f.lines {
		if ln.HasTS {
			return true
		}
	}
	return false
}

func (f *File) readInitial() error {
	info, err := os.Stat(f.Path)
	if err != nil {
		if os.IsNotExist(err) {
			f.readOff = 0
			return nil
		}
		return err
	}
	if info.Size() > maxInitialRead {
		return f.readTailChunk(info.Size())
	}
	file, err := os.Open(f.Path)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := f.consumeReader(bufio.NewReader(file)); err != nil {
		return err
	}
	f.readOff = info.Size()
	return nil
}

func (f *File) readTailChunk(size int64) error {
	chunk := int64(65536)
	if chunk > size {
		chunk = size
	}
	offset := size - chunk
	file, err := os.Open(f.Path)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return err
	}
	r := bufio.NewReader(file)
	if offset > 0 {
		_, _ = r.ReadString('\n')
	}
	var buf []string
	for {
		line, err := r.ReadString('\n')
		if len(line) > 0 {
			buf = append(buf, trimNewline(line))
		}
		if err != nil {
			break
		}
	}
	if len(buf) > f.MaxLines {
		buf = buf[len(buf)-f.MaxLines:]
	}
	f.mu.Lock()
	for _, l := range buf {
		f.pushLineLocked(l)
	}
	f.readOff = size
	f.mu.Unlock()
	return nil
}

func (f *File) consumeReader(r *bufio.Reader) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for {
		line, err := r.ReadString('\n')
		if len(line) > 0 {
			f.pushLineLocked(trimNewline(line))
		}
		if err != nil {
			break
		}
	}
	return nil
}

func (f *File) readAppend() error {
	info, err := os.Stat(f.Path)
	if err != nil {
		return err
	}
	if info.Size() < f.readOff {
		f.mu.Lock()
		f.lines = nil
		f.readOff = 0
		f.mu.Unlock()
		return f.readInitial()
	}
	if info.Size() == f.readOff {
		return nil
	}
	file, err := os.Open(f.Path)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Seek(f.readOff, io.SeekStart); err != nil {
		return err
	}
	f.mu.Lock()
	r := bufio.NewReader(file)
	for {
		line, err := r.ReadString('\n')
		if len(line) > 0 {
			f.pushLineLocked(trimNewline(line))
		}
		if err != nil {
			break
		}
	}
	f.readOff = info.Size()
	f.mu.Unlock()
	return nil
}

func (f *File) pushLineLocked(text string) {
	f.seq++
	ln := Line{Seq: f.seq, Text: text}
	if f.Pattern != nil {
		if m, ok := f.Pattern.Match(text); ok {
			ln.Fields = m.Fields
			if ts, ok := lineparse.TimestampFromFields(m.Fields); ok {
				ln.HasTS = true
				ln.Timestamp = ts
			}
		}
	}
	f.lines = append(f.lines, ln)
	if len(f.lines) > f.MaxLines {
		f.lines = f.lines[len(f.lines)-f.MaxLines:]
	}
	f.activity = 7
	if f.OnLine != nil {
		go f.OnLine()
	}
}

func (f *File) watchLoop() {
	for {
		select {
		case <-f.stopCh:
			return
		case ev, ok := <-f.watcher.Events:
			if !ok {
				return
			}
			if ev.Name == f.Path {
				_ = f.readAppend()
			}
		case <-f.watcher.Errors:
		}
	}
}

func (f *File) pollLoop() {
	for {
		select {
		case <-f.stopCh:
			return
		case <-f.poll.C:
			_ = f.readAppend()
		}
	}
}

func trimNewline(s string) string {
	if len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	if len(s) > 0 && s[len(s)-1] == '\r' {
		s = s[:len(s)-1]
	}
	return s
}
