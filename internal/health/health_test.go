package health

import "testing"

func TestPortOpenClosed(t *testing.T) {
	// Ephemeral port unlikely to be listening.
	if portOpen("127.0.0.1:9") {
		t.Log("port 9 accepted connection (unusual)")
	}
}

func TestNoProbes(t *testing.T) {
	r := Check("", "", false)
	if r.HasProbes || r.Up {
		t.Fatalf("%+v", r)
	}
}

func TestFormatNoProbes(t *testing.T) {
	lines := FormatProbeLine(Result{})
	if len(lines) != 1 || lines[0][0] != '-' {
		t.Fatalf("%v", lines)
	}
}
