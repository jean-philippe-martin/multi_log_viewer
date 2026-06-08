package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mlv.yaml")
	if err := os.WriteFile(path, []byte(`
health_poll_seconds: 2
services:
  api:
    logs:
      - path: logs/api.log
        line_pattern: "{timestamp:TIMESTAMP_ISO8601} {level:LOGLEVEL} "
    health:
      port: 8080
`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProjectRoot != dir {
		t.Fatalf("project root: got %q want %q", cfg.ProjectRoot, dir)
	}
	if len(cfg.Services) != 1 || cfg.Services[0].ID != "api" {
		t.Fatalf("services: %+v", cfg.Services)
	}
	if !cfg.Services[0].Health.HasProbes || cfg.Services[0].Health.Port != "127.0.0.1:8080" {
		t.Fatalf("health: %+v", cfg.Services[0].Health)
	}
	if cfg.Theme != defaultTheme {
		t.Fatalf("theme: got %q want %q", cfg.Theme, defaultTheme)
	}
}

func TestLoadThemeExplicit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mlv.yaml")
	if err := os.WriteFile(path, []byte(`
theme: vibrant
services:
  api:
    logs:
      - path: logs/api.log
`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != "vibrant" {
		t.Fatalf("theme: got %q", cfg.Theme)
	}
}

func TestLoadErrors(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{"no services", "services: {}", "'services' must be a mapping"},
		{"missing logs", "services:\n  x: {}", "logs required"},
		{"both path", "services:\n  x:\n    logs:\n      - path: a\n        path_pattern: b", "cannot have both"},
		{"neither", "services:\n  x:\n    logs:\n      - label: x", "requires path or path_pattern"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name+".yaml")
			if err := os.WriteFile(path, []byte(tc.yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path)
			if err == nil || !contains(err.Error(), tc.want) {
				t.Fatalf("err=%v want substring %q", err, tc.want)
			}
		})
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || findSub(s, sub))
}

func findSub(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
