package lineparse

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Pattern matches log line prefixes.
type Pattern struct {
	original string
	re       *regexp.Regexp
	fields   []fieldSpec
}

type fieldSpec struct {
	name      string
	typ       fieldType
	groupIdx  int
}

type fieldType int

const (
	typeString fieldType = iota
	typeTimestampISO8601
	typeTimestampHMS
	typeTimestampHMSS
	typeLogLevel
)

// Compile builds a line pattern.
func Compile(pattern string) (*Pattern, error) {
	if pattern == "" {
		return nil, fmt.Errorf("empty line_pattern")
	}
	var regex strings.Builder
	regex.WriteString("^")
	var fields []fieldSpec
	group := 1
	for i := 0; i < len(pattern); {
		if pattern[i] == '\\' && i+1 < len(pattern) {
			regex.WriteString(regexp.QuoteMeta(string(pattern[i+1])))
			i += 2
			continue
		}
		if pattern[i] == '{' {
			end := strings.IndexByte(pattern[i:], '}')
			if end < 0 {
				return nil, fmt.Errorf("unclosed {")
			}
			end += i
			inner := pattern[i+1 : end]
			name, typ, err := parseFieldDecl(inner)
			if err != nil {
				return nil, err
			}
			fs := fieldSpec{name: name, typ: typ, groupIdx: group}
			switch typ {
			case typeTimestampISO8601:
				regex.WriteString(`(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?:\.\d{1,9})?(?: [+-]\d{2}:\d{2}|Z)?)`)
			case typeTimestampHMS:
				regex.WriteString(`(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})`)
			case typeTimestampHMSS:
				regex.WriteString(`(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3})`)
			case typeLogLevel:
				regex.WriteString(`(DEBUG|INFO|WARN|ERROR)`)
			default:
				regex.WriteString(`(.*?)`)
			}
			fields = append(fields, fs)
			group++
			i = end + 1
			continue
		}
		regex.WriteString(regexp.QuoteMeta(string(pattern[i])))
		i++
	}
	re, err := regexp.Compile(regex.String())
	if err != nil {
		return nil, fmt.Errorf("compile line_pattern: %w", err)
	}
	return &Pattern{original: pattern, re: re, fields: fields}, nil
}

func parseFieldDecl(inner string) (string, fieldType, error) {
	name := inner
	typ := typeString
	if idx := strings.IndexByte(inner, ':'); idx >= 0 {
		name = inner[:idx]
		typeName := inner[idx+1:]
		switch typeName {
		case "TIMESTAMP_ISO8601":
			typ = typeTimestampISO8601
		case "YYYY-MM-DD HH:mm:ss":
			typ = typeTimestampHMS
		case "YYYY-MM-DD HH:mm:ss.SSS":
			typ = typeTimestampHMSS
		case "LOGLEVEL":
			typ = typeLogLevel
		default:
			return "", 0, fmt.Errorf("unknown line_pattern type: %s", typeName)
		}
	}
	if name == "" || !validName(name) {
		return "", 0, fmt.Errorf("invalid field name %q", name)
	}
	return name, typ, nil
}

func validName(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range s {
		if i == 0 {
			if c != '_' && !isLetter(c) {
				return false
			}
			continue
		}
		if c != '_' && !isLetter(c) && !isDigit(c) {
			return false
		}
	}
	return true
}

func isLetter(c rune) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }
func isDigit(c rune) bool  { return c >= '0' && c <= '9' }

// MatchResult holds parsed fields from a line.
type MatchResult struct {
	Fields map[string]interface{}
}

// Match applies the pattern at the start of line.
func (p *Pattern) Match(line string) (MatchResult, bool) {
	m := p.re.FindStringSubmatch(line)
	if m == nil {
		return MatchResult{}, false
	}
	fields := make(map[string]interface{})
	for _, fs := range p.fields {
		val := m[fs.groupIdx]
		switch fs.typ {
		case typeTimestampISO8601, typeTimestampHMS, typeTimestampHMSS:
			t, err := parseTimestamp(fs.typ, val)
			if err != nil {
				return MatchResult{}, false
			}
			fields[fs.name] = t
		case typeLogLevel:
			fields[fs.name] = val
		default:
			fields[fs.name] = val
		}
	}
	return MatchResult{Fields: fields}, true
}

// parseTimestamp parses a captured timestamp string. ISO8601 values may use T or
// a space, optional Z or numeric offsets, and fractional seconds; naive
// values without a zone are interpreted as UTC.
func parseTimestamp(typ fieldType, val string) (time.Time, error) {
	switch typ {
	case typeTimestampISO8601:
		val = strings.TrimSpace(val)
		if strings.HasSuffix(val, " Z") {
			val = strings.TrimSuffix(val, " Z") + "Z"
		}
		if strings.Contains(val, "T") {
			return time.Parse(time.RFC3339Nano, val)
		}
		// "2006-01-02 15:04:05.123 -07:00" or Z
		if strings.HasSuffix(val, "Z") {
			val = strings.TrimSuffix(val, "Z") + "Z"
			return time.Parse("2006-01-02 15:04:05.999999999Z07:00", val)
		}
		if idx := strings.LastIndex(val, " "); idx > 10 {
			tz := val[idx+1:]
			if len(tz) == 6 && (tz[0] == '+' || tz[0] == '-') {
				base := val[:idx]
				layout := "2006-01-02 15:04:05"
				if strings.Contains(base, ".") {
					layout = "2006-01-02 15:04:05.999999999"
				}
				return time.Parse(layout+" -07:00", base+" "+tz)
			}
		}
		layout := "2006-01-02 15:04:05"
		if strings.Contains(val, ".") {
			layout = "2006-01-02 15:04:05.999999999"
		}
		t, err := time.ParseInLocation(layout, val, time.UTC)
		return t, err
	case typeTimestampHMS:
		t, err := time.ParseInLocation("2006-01-02 15:04:05", val, time.UTC)
		return t, err
	case typeTimestampHMSS:
		t, err := time.ParseInLocation("2006-01-02 15:04:05.000", val, time.UTC)
		return t, err
	default:
		return time.Time{}, fmt.Errorf("not a timestamp")
	}
}

// TimestampFromFields returns the timestamp field if present.
func TimestampFromFields(fields map[string]interface{}) (time.Time, bool) {
	if t, ok := fields["timestamp"].(time.Time); ok {
		return t, true
	}
	return time.Time{}, false
}
