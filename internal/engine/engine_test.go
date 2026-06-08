package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jpmartin/multi_log_viewer/internal/config"
)

func TestHandleKeyEnterLog(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "api.log")
	_ = os.MkdirAll(filepath.Dir(logPath), 0o755)
	_ = os.WriteFile(logPath, []byte("2026-01-01 00:00:00.000 INFO hi\n"), 0o644)
	cfgPath := filepath.Join(dir, "mlv.yaml")
	yaml := `
services:
  api:
    logs:
      - path: logs/api.log
        line_pattern: "{timestamp:YYYY-MM-DD HH:mm:ss.SSS} {level:LOGLEVEL} "
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	eng, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()
	// cursor on service, move to log
	eng.HandleKey(KeyDown)
	eng.HandleKey(KeyEnter)
	mv := eng.MainView()
	if len(mv.Panes) != 1 || mv.Panes[0].Kind != PaneLog {
		t.Fatalf("panes=%+v", mv.Panes)
	}
}

func TestHandleKeyFolderNoop(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "logs", "acme"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "logs", "acme", "log.txt"), []byte("x\n"), 0o644)
	cfgPath := filepath.Join(dir, "mlv.yaml")
	yaml := `
services:
  w:
    logs:
      - path_pattern: "logs/{client}/log.txt"
        label: "{client}"
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, _ := config.Load(cfgPath)
	eng, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer eng.Close()
	eng.HandleKey(KeyDown) // folder
	if eng.rows[eng.cursor].kind != RowFolder {
		t.Fatalf("expected folder row")
	}
	eng.HandleKey(KeyEnter)
	if eng.status != "select a log file" {
		t.Fatalf("status=%q", eng.status)
	}
}
