package theme

import (
	"fmt"
	"strconv"
	"strings"
)

var transparentBG = map[string]bool{
	"":            true,
	"none":        true,
	"transparent": true,
	"default":     true,
}

// ansiNameToIndex maps ANSI color names to lipgloss-compatible palette indices.
// Lipgloss accepts numeric strings and hex, but not English names like "green".
var ansiNameToIndex = map[string]string{
	"black": "0", "red": "1", "green": "2", "yellow": "3",
	"blue": "4", "magenta": "5", "cyan": "6", "white": "7",
	"bright-black": "8", "bright-red": "9", "bright-green": "10",
	"bright-yellow": "11", "bright-blue": "12", "bright-magenta": "13",
	"bright-cyan": "14", "bright-white": "15",
	"grey": "243", "gray": "243",
}

var ansiNames = func() map[string]bool {
	m := make(map[string]bool, len(ansiNameToIndex))
	for k := range ansiNameToIndex {
		m[k] = true
	}
	return m
}()

// StyleSpec is foreground/background styling for one UI element.
type StyleSpec struct {
	FG   string
	BG   string
	Bold bool
}

func (s StyleSpec) hasFG() bool { return normalizeFG(s.FG) != "" }
func (s StyleSpec) hasBG() bool { return !transparentBG[strings.ToLower(strings.TrimSpace(s.BG))] }

func normalizeFG(fg string) string {
	fg = strings.TrimSpace(fg)
	if strings.EqualFold(fg, "none") || strings.EqualFold(fg, "default") {
		return ""
	}
	return fg
}

func parseColorToken(tok string) (string, error) {
	tok = strings.TrimSpace(tok)
	if tok == "" || transparentBG[strings.ToLower(tok)] {
		return "", nil
	}
	if strings.HasPrefix(tok, "#") {
		if len(tok) != 4 && len(tok) != 7 {
			return "", fmt.Errorf("invalid hex color %q", tok)
		}
		for _, c := range tok[1:] {
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
				return "", fmt.Errorf("invalid hex color %q", tok)
			}
		}
		return tok, nil
	}
	if n, err := strconv.Atoi(tok); err == nil {
		if n < 0 || n > 255 {
			return "", fmt.Errorf("color index out of range: %d", n)
		}
		return strconv.Itoa(n), nil
	}
	lower := strings.ToLower(tok)
	if ansiNames[lower] {
		return lower, nil
	}
	return "", fmt.Errorf("unknown color %q", tok)
}

func parseShorthand(raw string, spec *StyleSpec) error {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "none") {
		*spec = StyleSpec{}
		return nil
	}
	parts := strings.SplitN(raw, " on ", 2)
	fg, err := parseColorToken(parts[0])
	if err != nil {
		return err
	}
	spec.FG = fg
	spec.Bold = false
	if len(parts) == 1 {
		spec.BG = ""
		return nil
	}
	bg, err := parseColorToken(parts[1])
	if err != nil {
		return err
	}
	spec.BG = bg
	return nil
}

// toLipglossColor converts a parsed theme token into a value lipgloss can render.
func toLipglossColor(tok string) string {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return ""
	}
	if strings.HasPrefix(tok, "#") {
		return tok
	}
	if n, err := strconv.Atoi(tok); err == nil {
		return strconv.Itoa(n)
	}
	if idx, ok := ansiNameToIndex[strings.ToLower(tok)]; ok {
		return idx
	}
	return tok
}
