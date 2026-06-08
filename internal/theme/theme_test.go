package theme

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestParseShorthand(t *testing.T) {
	var s StyleSpec
	if err := parseShorthand("red on black", &s); err != nil {
		t.Fatal(err)
	}
	if s.FG != "red" || s.BG != "black" {
		t.Fatalf("got %+v", s)
	}
	if err := parseShorthand("#ccc on none", &s); err != nil {
		t.Fatal(err)
	}
	if s.FG != "#ccc" || s.BG != "" {
		t.Fatalf("got %+v", s)
	}
	if err := parseShorthand("252", &s); err != nil {
		t.Fatal(err)
	}
	if s.FG != "252" || s.hasBG() {
		t.Fatalf("got %+v", s)
	}
}

func TestParseThemeFile(t *testing.T) {
	data := []byte(`
log_text: "#d0d0d0"
left_text: "252"
selected_left_text:
  fg: white
  bg: "#383838"
  bold: true
border: "240"
selected_top_border: "252"
`)
	th, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if th.LogText.FG != "#d0d0d0" {
		t.Fatalf("log_text: %+v", th.LogText)
	}
	if !th.SelectedLeftText.Bold || th.SelectedLeftText.BG != "#383838" {
		t.Fatalf("selected: %+v", th.SelectedLeftText)
	}
}

func TestLoadVibrantTheme(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	th, err := Load("vibrant", root)
	if err != nil {
		t.Fatal(err)
	}
	if th.LogText.FG != "bright-cyan" || !th.LogText.Bold {
		t.Fatalf("log_text: %+v", th.LogText)
	}
	if th.Border.FG != "bright-magenta" {
		t.Fatalf("border: %+v", th.Border)
	}
}

func TestLoadFromProjectThemes(t *testing.T) {
	dir := t.TempDir()
	themesDir := filepath.Join(dir, "themes")
	if err := os.MkdirAll(themesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themesDir, "solar.yaml"), []byte(`log_text: "red on black"`), 0o644); err != nil {
		t.Fatal(err)
	}
	th, err := Load("solar", dir)
	if err != nil {
		t.Fatal(err)
	}
	if th.LogText.FG != "red" || th.LogText.BG != "black" {
		t.Fatalf("got %+v", th.LogText)
	}
}

func TestLoadEmbeddedDefault(t *testing.T) {
	th, err := Load("default", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if th.Border.FG != "240" {
		t.Fatalf("border: %+v", th.Border)
	}
}

func TestSelectedTopBorderUsesLipglossColor(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	th := &Theme{
		Border:            StyleSpec{FG: "green"},
		SelectedTopBorder: StyleSpec{FG: "red"},
	}
	out := th.PaneBorderStyle(true).Width(10).Render("x")
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("expected ANSI colors: %q", out)
	}
	// Top row should include red (ANSI 31); left border uses green (32).
	if !strings.Contains(out, "31") {
		t.Fatalf("expected red top border in %q", out)
	}
}

func TestLogScrollNoticeStyle(t *testing.T) {
	th, err := Parse([]byte(`log_scroll_notice: grey`))
	if err != nil {
		t.Fatal(err)
	}
	if toLipglossColor(th.LogScrollNotice.FG) != "243" {
		t.Fatalf("got %q", th.LogScrollNotice.FG)
	}
}

func TestToLipglossColorNames(t *testing.T) {
	if toLipglossColor("green") != "2" {
		t.Fatalf("green -> %q", toLipglossColor("green"))
	}
	if toLipglossColor("blue") != "4" {
		t.Fatalf("blue -> %q", toLipglossColor("blue"))
	}
	if toLipglossColor("252") != "252" {
		t.Fatalf("252 -> %q", toLipglossColor("252"))
	}
}

func TestNamedColorRendersWithLipgloss(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	out := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(toLipglossColor("green"))).
		Width(8).
		Render("hi")
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("expected ANSI from green border: %q", out)
	}
}

func TestParseBorderColorNames(t *testing.T) {
	data := []byte(`
border:
  fg: green
selected_top_border:
  fg: "252"
`)
	th, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if th.Border.FG != "green" {
		t.Fatalf("border: %+v", th.Border)
	}
}

func TestStyleSpecNoBackground(t *testing.T) {
	s := StyleSpec{FG: "252", BG: "transparent"}
	if s.hasBG() {
		t.Fatal("transparent should not set background")
	}
	if strings.Contains(s.Lipgloss().Render("x"), "48;") {
		t.Fatal("should not emit background SGR")
	}
}
