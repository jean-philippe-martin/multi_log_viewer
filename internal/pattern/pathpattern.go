package pattern

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Template describes a path_pattern with named captures.
type Template struct {
	Original   string
	Glob       string
	CaptureNames []string
}

// Validate checks path_pattern syntax.
func Validate(pathPattern string) error {
	_, err := Compile(pathPattern)
	return err
}

// Compile parses a path_pattern into a glob and capture names.
func Compile(pathPattern string) (Template, error) {
	if pathPattern == "" {
		return Template{}, fmt.Errorf("empty path_pattern")
	}
	var b strings.Builder
	var names []string
	for i := 0; i < len(pathPattern); {
		if pathPattern[i] == '\\' && i+1 < len(pathPattern) {
			b.WriteByte(pathPattern[i+1])
			i += 2
			continue
		}
		if pathPattern[i] == '{' {
			end := strings.IndexByte(pathPattern[i:], '}')
			if end < 0 {
				return Template{}, fmt.Errorf("unclosed { in path_pattern")
			}
			end += i
			name := pathPattern[i+1 : end]
			if name == "" {
				return Template{}, fmt.Errorf("empty capture name")
			}
			for _, c := range name {
				if !isNameRune(c) {
					return Template{}, fmt.Errorf("invalid capture name %q", name)
				}
			}
			names = append(names, name)
			b.WriteString("*")
			i = end + 1
			continue
		}
		b.WriteByte(pathPattern[i])
		i++
	}
	return Template{
		Original:     pathPattern,
		Glob:         b.String(),
		CaptureNames: names,
	}, nil
}

func isNameRune(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// MatchCaptures extracts capture values from a relative file path.
func (t Template) MatchCaptures(relPath string) (map[string]string, bool) {
	relPath = filepath.ToSlash(relPath)
	patParts := strings.Split(filepath.ToSlash(t.Original), "/")
	pathParts := strings.Split(relPath, "/")
	if len(patParts) != len(pathParts) {
		return nil, false
	}
	caps := make(map[string]string)
	ci := 0
	for i, pp := range patParts {
		if strings.HasPrefix(pp, "{") && strings.HasSuffix(pp, "}") {
			if ci >= len(t.CaptureNames) {
				return nil, false
			}
			name := t.CaptureNames[ci]
			ci++
			caps[name] = pathParts[i]
			continue
		}
		if pp != pathParts[i] {
			return nil, false
		}
	}
	return caps, true
}

// FolderKey returns a stable key for grouping files with the same captures.
func FolderKey(captures map[string]string, names []string) string {
	if len(names) == 0 {
		return ""
	}
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)
	var parts []string
	for _, n := range sorted {
		parts = append(parts, n+"="+captures[n])
	}
	return strings.Join(parts, "\x00")
}

// ApplyLabel substitutes {name} in a label template.
func ApplyLabel(template string, captures map[string]string) string {
	out := template
	for k, v := range captures {
		out = strings.ReplaceAll(out, "{"+k+"}", v)
	}
	return out
}

// DefaultFolderLabel joins capture values in name order.
func DefaultFolderLabel(captures map[string]string, names []string) string {
	if len(names) == 0 {
		return ""
	}
	var parts []string
	for _, n := range names {
		parts = append(parts, captures[n])
	}
	return strings.Join(parts, "/")
}

// Discover globs under baseDir and returns relative paths to regular files.
func Discover(baseDir string, tmpl Template) ([]string, error) {
	globPath := filepath.Join(baseDir, filepath.FromSlash(tmpl.Glob))
	matches, err := filepath.Glob(globPath)
	if err != nil {
		return nil, err
	}
	var rel []string
	for _, m := range matches {
		info, err := os.Lstat(m)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		r, err := filepath.Rel(baseDir, m)
		if err != nil {
			continue
		}
		if _, ok := tmpl.MatchCaptures(filepath.ToSlash(r)); !ok {
			continue
		}
		rel = append(rel, filepath.ToSlash(r))
	}
	sort.Strings(rel)
	return rel, nil
}
