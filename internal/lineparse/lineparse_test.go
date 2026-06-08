package lineparse

import (
	"testing"
	"time"
)

func TestMatchISO8601(t *testing.T) {
	p, err := Compile("{timestamp:TIMESTAMP_ISO8601} [{worker}] {level:LOGLEVEL}")
	if err != nil {
		t.Fatal(err)
	}
	line := "2026-05-31 17:06:15.123 -07:00 [alpha] INFO the foo"
	m, ok := p.Match(line)
	if !ok {
		t.Fatal("no match")
	}
	ts, ok := TimestampFromFields(m.Fields)
	if !ok {
		t.Fatal("no timestamp")
	}
	if ts.IsZero() {
		t.Fatal("zero time")
	}
	if m.Fields["level"] != "INFO" {
		t.Fatalf("level=%v", m.Fields["level"])
	}
}

func TestMatchHMSS(t *testing.T) {
	p, err := Compile("{timestamp:YYYY-MM-DD HH:mm:ss.SSS} [{worker}]")
	if err != nil {
		t.Fatal(err)
	}
	m, ok := p.Match("2026-05-31 17:06:15.123 [x] rest")
	if !ok {
		t.Fatal("no match")
	}
	ts := m.Fields["timestamp"].(time.Time)
	if ts.Location() != time.UTC {
		t.Fatalf("tz=%v", ts.Location())
	}
}

func TestUnknownType(t *testing.T) {
	_, err := Compile("{x:BOGUS}")
	if err == nil {
		t.Fatal("expected error")
	}
}
