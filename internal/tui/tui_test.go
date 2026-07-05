package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/jpmartin/multi_log_viewer/internal/config"
	"github.com/jpmartin/multi_log_viewer/internal/engine"
	"github.com/jpmartin/multi_log_viewer/internal/theme"
)

func newTestModel(t *testing.T, eng *engine.Engine) model {
	t.Helper()
	th, err := theme.Load("default", "")
	if err != nil {
		t.Fatal(err)
	}
	return model{
		eng:        eng,
		th:         th,
		events:     make(chan engine.Event, 64),
		width:      80,
		height:     14,
		focus:      focusMain,
		autoScroll: true,
	}
}

func testEngineWithLog(t *testing.T, lineCount int) (dir string, eng *engine.Engine) {
	t.Helper()
	dir = t.TempDir()
	logPath := filepath.Join(dir, "logs", "api.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatal(err)
	}
	var seed strings.Builder
	for i := 0; i < lineCount; i++ {
		fmt.Fprintf(&seed, "line-%02d message\n", i)
	}
	if err := os.WriteFile(logPath, []byte(seed.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, "mlv.yaml")
	yaml := `
services:
  api:
    logs:
      - path: logs/api.log
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	eng, err = engine.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	eng.HandleKey(engine.KeyDown)
	eng.HandleKey(engine.KeyEnter)
	return dir, eng
}

func updateModel(m model, msg tea.Msg) model {
	next, _ := m.Update(msg)
	return next.(model)
}

func keyMsg(key string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

func mainPaneText(m model) string {
	_, mainInner := splitPaneWidths(m.width)
	innerH := m.mainInnerH(m.paneHeight())
	return ansi.Strip(m.renderMainContent(mainInner, innerH))
}
