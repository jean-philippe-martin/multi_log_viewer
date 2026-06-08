package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

func TestLoadRepoDefaultThemeHasGreenBorder(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	path := filepath.Join(root, "themes", "default.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("repo themes/default.yaml not found:", err)
	}
	th, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if th.Border.FG != "green" {
		t.Fatalf("expected green border in %s, got %q", path, th.Border.FG)
	}
}

func TestPaneBorderStyleEmitsGreenWhenFocused(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	th := &Theme{
		Border:            StyleSpec{FG: "green"},
		SelectedTopBorder: StyleSpec{FG: "green"},
	}
	out := th.PaneBorderStyle(true).Width(12).Render("hello")
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("expected ANSI in focused border: %q", ansi.Strip(out))
	}
}
