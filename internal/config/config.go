package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jpmartin/multi_log_viewer/internal/pattern"
	"gopkg.in/yaml.v3"
)

const (
	defaultHealthPoll      = 2.0
	defaultPatternRescan   = 5.0
	defaultRingBufferLines = 2000
	defaultTheme           = "default"
	minRingBufferLines     = 100
	maxRingBufferLines     = 100_000
)

// Config is the resolved application configuration.
type Config struct {
	ConfigPath           string
	ConfigDir            string
	ProjectRoot          string
	HealthPollSeconds    float64
	PatternRescanSeconds float64
	RingBufferLines      int
	Theme                string
	Services             []Service
}

// Service is one configured service.
type Service struct {
	ID      string
	Logs    []LogEntry
	Health  HealthConfig
	Start   Command
	Stop    Command
	Cwd     string
	BaseDir string
}

// LogEntry describes how to find a log file.
type LogEntry struct {
	Path        string
	PathPattern string
	Label       string
	LinePattern string
}

// HealthConfig holds optional health probes.
type HealthConfig struct {
	Port            string // host:port dial string
	ProcessContains string
	HasProbes       bool
}

// Command is a start or stop command.
type Command struct {
	Shell string
	Argv  []string
}

type rawConfig struct {
	ProjectRoot          string                `yaml:"project_root"`
	HealthPollSeconds    *float64              `yaml:"health_poll_seconds"`
	PatternRescanSeconds *float64              `yaml:"pattern_rescan_seconds"`
	RingBufferLines      *int                  `yaml:"ring_buffer_lines"`
	Theme                string                `yaml:"theme"`
	Services             map[string]rawService `yaml:"services"`
}

type rawService struct {
	Logs   []rawLogEntry          `yaml:"logs"`
	Health map[string]interface{} `yaml:"health"`
	Start  yaml.Node              `yaml:"start"`
	Stop   yaml.Node              `yaml:"stop"`
	Cwd    string                 `yaml:"cwd"`
}

type rawLogEntry struct {
	Path        string `yaml:"path"`
	PathPattern string `yaml:"path_pattern"`
	Label       string `yaml:"label"`
	LinePattern string `yaml:"line_pattern"`
}

// Load reads and validates a config file.
func Load(configPath string) (*Config, error) {
	absConfig, err := filepath.Abs(configPath)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(absConfig)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config not found: %s", configPath)
		}
		return nil, err
	}
	configDir := filepath.Dir(absConfig)

	var raw rawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if len(raw.Services) == 0 {
		return nil, fmt.Errorf("'services' must be a mapping")
	}

	projectRoot := configDir
	if strings.TrimSpace(raw.ProjectRoot) != "" {
		projectRoot = filepath.Clean(filepath.Join(configDir, raw.ProjectRoot))
	}
	projectRoot, err = filepath.Abs(projectRoot)
	if err != nil {
		return nil, err
	}

	healthPoll := defaultHealthPoll
	if raw.HealthPollSeconds != nil {
		healthPoll = *raw.HealthPollSeconds
	}
	patternRescan := defaultPatternRescan
	if raw.PatternRescanSeconds != nil {
		patternRescan = *raw.PatternRescanSeconds
	}
	ringLines := defaultRingBufferLines
	if raw.RingBufferLines != nil {
		ringLines = *raw.RingBufferLines
	}
	if ringLines < minRingBufferLines || ringLines > maxRingBufferLines {
		return nil, fmt.Errorf("ring_buffer_lines must be between %d and %d", minRingBufferLines, maxRingBufferLines)
	}

	cfg := &Config{
		ConfigPath:           absConfig,
		ConfigDir:            configDir,
		ProjectRoot:          projectRoot,
		HealthPollSeconds:    healthPoll,
		PatternRescanSeconds: patternRescan,
		RingBufferLines:      ringLines,
		Theme:                themeName(raw.Theme),
	}

	for _, id := range sortedKeys(raw.Services) {
		svc, err := parseService(id, raw.Services[id], projectRoot)
		if err != nil {
			return nil, err
		}
		cfg.Services = append(cfg.Services, svc)
	}
	return cfg, nil
}

func parseService(id string, rs rawService, projectRoot string) (Service, error) {
	if len(rs.Logs) == 0 {
		return Service{}, fmt.Errorf("service '%s': logs required", id)
	}
	base := projectRoot
	if strings.TrimSpace(rs.Cwd) != "" {
		base = filepath.Clean(filepath.Join(projectRoot, rs.Cwd))
	}
	svc := Service{
		ID:      id,
		Cwd:     rs.Cwd,
		BaseDir: base,
	}
	for _, rl := range rs.Logs {
		le, err := parseLogEntry(id, rl)
		if err != nil {
			return Service{}, err
		}
		svc.Logs = append(svc.Logs, le)
	}
	hc, err := parseHealth(rs.Health)
	if err != nil {
		return Service{}, fmt.Errorf("service '%s': %w", id, err)
	}
	svc.Health = hc
	svc.Start, err = parseCommand(rs.Start)
	if err != nil {
		return Service{}, fmt.Errorf("service '%s': start: %w", id, err)
	}
	svc.Stop, err = parseCommand(rs.Stop)
	if err != nil {
		return Service{}, fmt.Errorf("service '%s': stop: %w", id, err)
	}
	return svc, nil
}

func parseLogEntry(serviceID string, rl rawLogEntry) (LogEntry, error) {
	hasPath := strings.TrimSpace(rl.Path) != ""
	hasPattern := strings.TrimSpace(rl.PathPattern) != ""
	if !hasPath && !hasPattern {
		return LogEntry{}, fmt.Errorf("service '%s': log entry requires path or path_pattern", serviceID)
	}
	if hasPath && hasPattern {
		return LogEntry{}, fmt.Errorf("service '%s': log entry cannot have both path and path_pattern", serviceID)
	}
	le := LogEntry{
		Path:        strings.TrimSpace(rl.Path),
		PathPattern: strings.TrimSpace(rl.PathPattern),
		Label:       rl.Label,
		LinePattern: rl.LinePattern,
	}
	if hasPattern {
		if err := pattern.Validate(le.PathPattern); err != nil {
			return LogEntry{}, fmt.Errorf("service '%s': invalid path_pattern: %w", serviceID, err)
		}
	}
	return le, nil
}

func parseHealth(h map[string]interface{}) (HealthConfig, error) {
	if len(h) == 0 {
		return HealthConfig{}, nil
	}
	var hc HealthConfig
	if v, ok := h["port"]; ok && v != nil {
		hc.HasProbes = true
		switch p := v.(type) {
		case int:
			hc.Port = fmt.Sprintf("127.0.0.1:%d", p)
		case int64:
			hc.Port = fmt.Sprintf("127.0.0.1:%d", p)
		case float64:
			hc.Port = fmt.Sprintf("127.0.0.1:%d", int(p))
		case string:
			if strings.Contains(p, ":") {
				hc.Port = p
			} else {
				hc.Port = "127.0.0.1:" + p
			}
		default:
			return HealthConfig{}, fmt.Errorf("invalid port type")
		}
	}
	if v, ok := h["process_contains"]; ok && v != nil {
		hc.HasProbes = true
		s, ok := v.(string)
		if !ok {
			return HealthConfig{}, fmt.Errorf("process_contains must be a string")
		}
		hc.ProcessContains = s
	}
	return hc, nil
}

func parseCommand(node yaml.Node) (Command, error) {
	if node.Kind == 0 {
		return Command{}, nil
	}
	var c Command
	if node.Kind == yaml.ScalarNode {
		c.Shell = node.Value
		return c, nil
	}
	if node.Kind == yaml.SequenceNode {
		for _, n := range node.Content {
			c.Argv = append(c.Argv, n.Value)
		}
		return c, nil
	}
	return Command{}, fmt.Errorf("command must be string or list")
}

// ResolveLogPath returns absolute path for a fixed log path entry.
func (s Service) ResolveLogPath(relPath string) string {
	return filepath.Clean(filepath.Join(s.BaseDir, relPath))
}

// EnsureLogFile creates parent dirs and empty file if missing.
func EnsureLogFile(absPath string) error {
	if _, err := os.Stat(absPath); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(absPath, os.O_CREATE|os.O_RDONLY, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}

func themeName(raw string) string {
	if t := strings.TrimSpace(raw); t != "" {
		return t
	}
	return defaultTheme
}

func sortedKeys(m map[string]rawService) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
