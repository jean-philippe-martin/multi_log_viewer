package theme

import (
	"path/filepath"
	"testing"

	"github.com/jpmartin/multi_log_viewer/internal/config"
)

func TestLoadViaProjectConfig(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	cfg, err := config.Load(filepath.Join(root, "mlv.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	th, err := Load(cfg.Theme, cfg.ProjectRoot)
	if err != nil {
		t.Fatal(err)
	}
	if th.Border.FG != "green" {
		t.Fatalf("projectRoot=%q theme border=%q want green", cfg.ProjectRoot, th.Border.FG)
	}
}
