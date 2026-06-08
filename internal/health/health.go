package health

import (
	"context"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

const dialTimeout = 500 * time.Millisecond

// Result of a health check for one service.
type Result struct {
	Up                 bool
	HasProbes          bool
	PortOpen           bool
	ProcessFound       bool
	Port               string
	ProcessContains    string
	PID                int
	PortPID            int
	ProcessPID         int
}

// Check runs configured probes.
func Check(port, processContains string, hasProbes bool) Result {
	r := Result{
		Port:              port,
		ProcessContains:   processContains,
		HasProbes:         hasProbes,
	}
	if !hasProbes {
		return r
	}
	if port != "" {
		r.PortOpen = portOpen(port)
		if r.PortOpen {
			r.PortPID = pidForPort(port)
		}
	}
	if processContains != "" {
		r.ProcessPID = findProcessPID(processContains)
		r.ProcessFound = r.ProcessPID > 0
	}
	r.PID = r.PortPID
	if r.PID == 0 {
		r.PID = r.ProcessPID
	}
	if port != "" && processContains != "" {
		if r.PortOpen && r.ProcessFound {
			if r.PortPID > 0 && r.ProcessPID > 0 && r.PortPID == r.ProcessPID {
				r.PID = r.PortPID
			} else if r.PortPID > 0 {
				r.PID = r.PortPID
			}
		}
	}
	r.Up = true
	if port != "" && !r.PortOpen {
		r.Up = false
	}
	if processContains != "" && !r.ProcessFound {
		r.Up = false
	}
	return r
}

func portOpen(addr string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func findProcessPID(needle string) int {
	needle = strings.ToLower(needle)
	procs, err := process.Processes()
	if err != nil {
		return 0
	}
	for _, p := range procs {
		cmd, err := p.Cmdline()
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(cmd), needle) {
			return int(p.Pid)
		}
	}
	return 0
}

func pidForPort(addr string) int {
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		if strings.HasPrefix(addr, ":") {
			portStr = strings.TrimPrefix(addr, ":")
		} else {
			return 0
		}
	}
	switch runtime.GOOS {
	case "darwin":
		return pidLsof(portStr)
	default:
		return pidLsof(portStr)
	}
}

func pidLsof(port string) int {
	out, err := exec.Command("lsof", "-i", ":"+port, "-sTCP:LISTEN", "-t").Output()
	if err != nil {
		return 0
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return 0
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(lines[0]))
	return pid
}

// FormatProbeLine returns a human-readable probe status line.
func FormatProbeLine(r Result) []string {
	var lines []string
	if !r.HasProbes {
		return []string{"- no liveness detection configured for this service"}
	}
	portCfg := r.Port != ""
	procCfg := r.ProcessContains != ""
	if portCfg && procCfg {
		if r.Up && r.PID > 0 {
			lines = append(lines, "● port "+portDisplay(r.Port)+" bound to process name \""+r.ProcessContains+"\", PID "+itoa(r.PID))
		} else if r.PortOpen && r.ProcessFound && r.PortPID != r.ProcessPID {
			lines = append(lines, "○ port "+portDisplay(r.Port)+", bound but not to process name \""+r.ProcessContains+"\"")
		} else if !r.PortOpen && r.ProcessFound {
			lines = append(lines, "○ port "+portDisplay(r.Port)+", not bound. Process name \""+r.ProcessContains+"\" exists.")
		} else if r.PortOpen && !r.ProcessFound {
			lines = append(lines, "○ port "+portDisplay(r.Port)+", bound but not to process name \""+r.ProcessContains+"\"")
		} else if !r.PortOpen && !r.ProcessFound {
			lines = append(lines, "○ port "+portDisplay(r.Port)+", nothing bound")
			lines = append(lines, "○ process name \""+r.ProcessContains+"\" not found")
		}
		return lines
	}
	if portCfg {
		if r.PortOpen && r.PID > 0 {
			return []string{"● port " + portDisplay(r.Port) + " bound to PID " + itoa(r.PID)}
		}
		return []string{"○ port " + portDisplay(r.Port) + ", nothing bound"}
	}
	if procCfg {
		if r.ProcessFound {
			return []string{"● process name \"" + r.ProcessContains + "\" bound to PID " + itoa(r.ProcessPID)}
		}
		return []string{"○ process name \"" + r.ProcessContains + "\" not found"}
	}
	return lines
}

func portDisplay(addr string) string {
	_, port, err := net.SplitHostPort(addr)
	if err == nil {
		return port
	}
	return addr
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
