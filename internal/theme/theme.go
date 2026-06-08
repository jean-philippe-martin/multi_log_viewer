package theme

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

//go:embed builtin/default.yaml
var embeddedDefault []byte

// Theme holds colors for UI elements.
type Theme struct {
	LogText           StyleSpec `yaml:"log_text"`
	LogScrollNotice   StyleSpec `yaml:"log_scroll_notice"`
	LeftText          StyleSpec `yaml:"left_text"`
	SelectedLeftText  StyleSpec `yaml:"selected_left_text"`
	Border            StyleSpec `yaml:"border"`
	SelectedTopBorder StyleSpec `yaml:"selected_top_border"`
}

// Load resolves and parses a theme by name or path under projectRoot/themes/.
func Load(name, projectRoot string) (*Theme, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "default"
	}
	path := resolvePath(name, projectRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && name == "default" {
			data = embeddedDefault
		} else if os.IsNotExist(err) {
			return nil, fmt.Errorf("theme not found: %s", name)
		} else {
			return nil, err
		}
	}
	return Parse(data)
}

func resolvePath(name, projectRoot string) string {
	if strings.Contains(name, "/") || strings.Contains(name, string(os.PathSeparator)) || strings.HasSuffix(name, ".yaml") {
		if filepath.IsAbs(name) {
			return name
		}
		return filepath.Clean(filepath.Join(projectRoot, name))
	}
	return filepath.Join(projectRoot, "themes", name+".yaml")
}

// Parse reads a theme YAML document.
func Parse(data []byte) (*Theme, error) {
	var raw map[string]yaml.Node
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse theme: %w", err)
	}
	t := &Theme{}
	fields := map[string]*StyleSpec{
		"log_text":            &t.LogText,
		"log_scroll_notice":   &t.LogScrollNotice,
		"left_text":           &t.LeftText,
		"selected_left_text":  &t.SelectedLeftText,
		"border":              &t.Border,
		"selected_top_border": &t.SelectedTopBorder,
	}
	for key, spec := range fields {
		node, ok := raw[key]
		if !ok {
			continue
		}
		if err := unmarshalStyle(node, spec); err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
	}
	return t, nil
}

func unmarshalStyle(node yaml.Node, spec *StyleSpec) error {
	switch node.Kind {
	case yaml.ScalarNode:
		return parseShorthand(node.Value, spec)
	case yaml.MappingNode:
		var raw map[string]interface{}
		if err := node.Decode(&raw); err != nil {
			return err
		}
		if v, ok := raw["fg"]; ok {
			fg, err := parseColorToken(scalarString(v))
			if err != nil {
				return fmt.Errorf("fg: %w", err)
			}
			spec.FG = fg
		}
		if v, ok := raw["bg"]; ok {
			bg, err := parseColorToken(scalarString(v))
			if err != nil {
				return fmt.Errorf("bg: %w", err)
			}
			spec.BG = bg
		}
		if v, ok := raw["bold"]; ok {
			b, ok := v.(bool)
			if !ok {
				return fmt.Errorf("bold must be a boolean")
			}
			spec.Bold = b
		}
		return nil
	default:
		return fmt.Errorf("expected string or mapping")
	}
}

func scalarString(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.Itoa(int(x))
	case float64:
		return strconv.Itoa(int(x))
	default:
		return fmt.Sprint(v)
	}
}

// Lipgloss builds a lipgloss style from a spec.
func (s StyleSpec) Lipgloss() lipgloss.Style {
	st := lipgloss.NewStyle()
	if s.hasFG() {
		st = st.Foreground(lipgloss.Color(toLipglossColor(s.FG)))
	}
	if s.hasBG() {
		st = st.Background(lipgloss.Color(toLipglossColor(s.BG)))
	}
	if s.Bold {
		st = st.Bold(true)
	}
	return st
}

func borderColorOrDefault(fg string) string {
	if fg == "" {
		return "240"
	}
	return toLipglossColor(fg)
}

// BorderFgStyle colors ordinary border segments.
func (t *Theme) BorderFgStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(borderColorOrDefault(t.Border.FG)))
}

// TopBorderFgStyle colors a focused pane's top border segment.
func (t *Theme) TopBorderFgStyle(focused bool) lipgloss.Style {
	if focused && t.SelectedTopBorder.FG != "" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(toLipglossColor(t.SelectedTopBorder.FG)))
	}
	return t.BorderFgStyle()
}

// PaneBorderStyle returns the pane box border; top is double when focused.
// selected_top_border applies only to the top edge; other edges use border.
func (t *Theme) PaneBorderStyle(focused bool) lipgloss.Style {
	borderColor := borderColorOrDefault(t.Border.FG)
	topColor := borderColor
	if focused && t.SelectedTopBorder.FG != "" {
		topColor = toLipglossColor(t.SelectedTopBorder.FG)
	}
	normal := lipgloss.NormalBorder()
	if !focused {
		return lipgloss.NewStyle().Border(normal).BorderForeground(lipgloss.Color(borderColor))
	}
	double := lipgloss.DoubleBorder()
	custom := lipgloss.Border{
		Top:         double.Top,
		Bottom:      normal.Bottom,
		Left:        normal.Left,
		Right:       normal.Right,
		TopLeft:     double.TopLeft,
		TopRight:    double.TopRight,
		BottomLeft:  normal.BottomLeft,
		BottomRight: normal.BottomRight,
	}
	st := lipgloss.NewStyle().Border(custom)
	if topColor == borderColor {
		return st.BorderForeground(lipgloss.Color(borderColor))
	}
	return st.
		BorderTopForeground(lipgloss.Color(topColor)).
		BorderBottomForeground(lipgloss.Color(borderColor)).
		BorderLeftForeground(lipgloss.Color(borderColor)).
		BorderRightForeground(lipgloss.Color(borderColor))
}

// HelpStyle styles the help overlay.
func (t *Theme) HelpStyle() lipgloss.Style {
	bg := t.SelectedLeftText
	if !bg.hasBG() {
		bg.BG = "236"
	}
	if !bg.hasFG() {
		bg.FG = t.LeftText.FG
	}
	return bg.Lipgloss()
}

// LogScrollNoticeStyle styles the auto-scroll pause notice in the main pane.
func (t *Theme) LogScrollNoticeStyle() lipgloss.Style {
	if t.LogScrollNotice.FG == "" && !t.LogScrollNotice.hasBG() && !t.LogScrollNotice.Bold {
		return t.MutedStyle()
	}
	return t.LogScrollNotice.Lipgloss()
}

// MutedStyle is secondary text in the main pane (prefixes, empty state).
func (t *Theme) MutedStyle() lipgloss.Style {
	spec := t.LeftText
	if spec.FG == "" {
		spec.FG = "243"
	}
	spec.BG = ""
	spec.Bold = false
	return spec.Lipgloss()
}
