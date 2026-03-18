package config

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type EffectiveConfig struct {
	Tracker   TrackerConfig
	Polling   PollingConfig
	Workspace WorkspaceConfig
	Hooks     HooksConfig
	Agent     AgentConfig
	Provider  ProviderConfig
	Codex     CodexConfig
	Server    ServerConfig
	Logging   LoggingConfig
}

type TrackerConfig struct {
	Kind           string
	ActiveStates   []string
	TerminalStates []string
	PageSize       int
	Timeout        time.Duration
	Params         map[string]any // tracker-specific key-value pairs from YAML
}

type PollingConfig struct {
	Interval time.Duration
}

type WorkspaceConfig struct {
	Root string
}

type HooksConfig struct {
	AfterCreate  string
	BeforeRun    string
	AfterRun     string
	BeforeRemove string
	Timeout      time.Duration
}

type AgentConfig struct {
	MaxConcurrentAgents        int
	MaxRetryBackoff            time.Duration
	MaxTurns                   int
	MaxConcurrentAgentsByState map[string]int
	StallTimeout               time.Duration
}

type ProviderConfig struct {
	Kind string
}

type CodexConfig struct {
	Command           string
	ApprovalPolicy    any
	ThreadSandbox     any
	TurnSandboxPolicy any
	TurnTimeout       time.Duration
	ReadTimeout       time.Duration
}

type ServerConfig struct {
	Port *int
}

type LoggingConfig struct {
	File       string
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
}

var (
	ErrUnsupportedProviderKind = errors.New("unsupported_provider_kind")
	ErrMissingCodexCommand     = errors.New("missing_codex_command")
)

func FromWorkflowConfig(cfg map[string]any) (EffectiveConfig, error) {
	// Collect the entire raw tracker map as Params.
	trackerParams := rawMap(getNested(cfg, "tracker"))

	stallTimeoutAny := getNested(cfg, "agent", "stall_timeout_ms")
	if stallTimeoutAny == nil {
		// Back-compat: keep reading from codex stanza.
		stallTimeoutAny = getNested(cfg, "codex", "stall_timeout_ms")
	}

	effective := EffectiveConfig{
		Tracker: TrackerConfig{
			Kind:           str(getNested(cfg, "tracker", "kind")),
			ActiveStates:   strSlice(getNested(cfg, "tracker", "active_states")),
			TerminalStates: strSlice(getNested(cfg, "tracker", "terminal_states")),
			PageSize:       intFromAny(getNested(cfg, "tracker", "page_size"), 50),
			Timeout:        timeFromMsAny(getNested(cfg, "tracker", "timeout_ms"), 30000),
			Params:         trackerParams,
		},
		Polling: PollingConfig{
			Interval: timeFromMsAny(getNested(cfg, "polling", "interval_ms"), 30000),
		},
		Workspace: WorkspaceConfig{
			Root: str(getNested(cfg, "workspace", "root")),
		},
		Hooks: HooksConfig{
			AfterCreate:  str(getNested(cfg, "hooks", "after_create")),
			BeforeRun:    str(getNested(cfg, "hooks", "before_run")),
			AfterRun:     str(getNested(cfg, "hooks", "after_run")),
			BeforeRemove: str(getNested(cfg, "hooks", "before_remove")),
			Timeout:      timeFromMsAny(getNested(cfg, "hooks", "timeout_ms"), 60000),
		},
		Agent: AgentConfig{
			MaxConcurrentAgents:        intFromAny(getNested(cfg, "agent", "max_concurrent_agents"), 10),
			MaxTurns:                   intFromAny(getNested(cfg, "agent", "max_turns"), 20),
			MaxRetryBackoff:            timeFromMsAny(getNested(cfg, "agent", "max_retry_backoff_ms"), 300000),
			MaxConcurrentAgentsByState: stateConcurrencyMap(getNested(cfg, "agent", "max_concurrent_agents_by_state")),
			StallTimeout:               timeFromMsAny(stallTimeoutAny, 300000),
		},
		Provider: ProviderConfig{
			Kind: str(getNested(cfg, "provider", "kind")),
		},
		Codex: CodexConfig{
			Command:           str(getNested(cfg, "codex", "command")),
			ApprovalPolicy:    getNested(cfg, "codex", "approval_policy"),
			ThreadSandbox:     getNested(cfg, "codex", "thread_sandbox"),
			TurnSandboxPolicy: getNested(cfg, "codex", "turn_sandbox_policy"),
			TurnTimeout:       timeFromMsAny(getNested(cfg, "codex", "turn_timeout_ms"), 3600000),
			ReadTimeout:       timeFromMsAny(getNested(cfg, "codex", "read_timeout_ms"), 5000),
		},
		Server: ServerConfig{
			Port: intPtr(getNested(cfg, "server", "port")),
		},
		Logging: LoggingConfig{
			File:       str(getNested(cfg, "logging", "file")),
			MaxSizeMB:  intFromAny(getNested(cfg, "logging", "max_size_mb"), 10),
			MaxBackups: intFromAny(getNested(cfg, "logging", "max_backups"), 5),
			MaxAgeDays: intFromAny(getNested(cfg, "logging", "max_age_days"), 0),
			Compress:   boolFromAny(getNested(cfg, "logging", "compress"), false),
		},
	}

	applyDefaults(&effective)
	resolveEnvironment(&effective)
	normalize(&effective)

	if err := validate(effective); err != nil {
		return EffectiveConfig{}, err
	}
	return effective, nil
}

func applyDefaults(cfg *EffectiveConfig) {
	if cfg.Tracker.Kind == "" {
		cfg.Tracker.Kind = "linear"
	}
	if len(cfg.Tracker.ActiveStates) == 0 {
		cfg.Tracker.ActiveStates = []string{"Todo", "In Progress"}
	}
	if len(cfg.Tracker.TerminalStates) == 0 {
		cfg.Tracker.TerminalStates = []string{"Closed", "Cancelled", "Canceled", "Duplicate", "Done"}
	}

	if cfg.Workspace.Root == "" {
		cfg.Workspace.Root = filepath.Join(os.TempDir(), "symphony_workspaces")
	}

	if strings.TrimSpace(cfg.Provider.Kind) == "" {
		cfg.Provider.Kind = "codex"
	}
	if cfg.Codex.Command == "" {
		cfg.Codex.Command = "codex app-server"
	}
	if cfg.Hooks.Timeout <= 0 {
		cfg.Hooks.Timeout = 60 * time.Second
	}
	if cfg.Agent.MaxConcurrentAgents <= 0 {
		cfg.Agent.MaxConcurrentAgents = 10
	}
	if cfg.Agent.MaxTurns <= 0 {
		cfg.Agent.MaxTurns = 20
	}
	if cfg.Agent.MaxRetryBackoff <= 0 {
		cfg.Agent.MaxRetryBackoff = 5 * time.Minute
	}
	if cfg.Polling.Interval <= 0 {
		cfg.Polling.Interval = 30 * time.Second
	}
	if cfg.Codex.ReadTimeout <= 0 {
		cfg.Codex.ReadTimeout = 5 * time.Second
	}
	if cfg.Codex.TurnTimeout <= 0 {
		cfg.Codex.TurnTimeout = time.Hour
	}
	// StallTimeout can be <=0 to disable per spec.
	if cfg.Tracker.PageSize <= 0 {
		cfg.Tracker.PageSize = 50
	}
	if cfg.Tracker.Timeout <= 0 {
		cfg.Tracker.Timeout = 30 * time.Second
	}
}

func resolveEnvironment(cfg *EffectiveConfig) {
	// Resolve $VAR tokens in all string values within Params.
	for k, v := range cfg.Tracker.Params {
		if s, ok := v.(string); ok {
			cfg.Tracker.Params[k] = resolveEnvToken(s)
		}
	}

	cfg.Workspace.Root = expandPath(resolveEnvToken(cfg.Workspace.Root))
	cfg.Logging.File = expandPath(resolveEnvToken(cfg.Logging.File))
}

func normalize(cfg *EffectiveConfig) {
	cfg.Tracker.Kind = strings.TrimSpace(cfg.Tracker.Kind)
	cfg.Provider.Kind = strings.TrimSpace(cfg.Provider.Kind)

	for i := range cfg.Tracker.ActiveStates {
		cfg.Tracker.ActiveStates[i] = strings.TrimSpace(cfg.Tracker.ActiveStates[i])
	}
	for i := range cfg.Tracker.TerminalStates {
		cfg.Tracker.TerminalStates[i] = strings.TrimSpace(cfg.Tracker.TerminalStates[i])
	}
}

func validate(cfg EffectiveConfig) error {
	switch strings.TrimSpace(cfg.Provider.Kind) {
	case "", "codex":
		if strings.TrimSpace(cfg.Codex.Command) == "" {
			return ErrMissingCodexCommand
		}
	default:
		return ErrUnsupportedProviderKind
	}
	return nil
}

func resolveEnvToken(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return v
	}
	if !strings.HasPrefix(v, "$") || len(v) == 1 {
		return v
	}
	env := os.Getenv(strings.TrimPrefix(v, "$"))
	return strings.TrimSpace(env)
}

func expandPath(v string) string {
	if v == "" {
		return v
	}
	if strings.HasPrefix(v, "~") {
		home, err := os.UserHomeDir()
		if err == nil && home != "" {
			switch v {
			case "~":
				v = home
			case "~/":
				v = home + string(filepath.Separator)
			default:
				if strings.HasPrefix(v, "~/") {
					v = filepath.Join(home, strings.TrimPrefix(v, "~/"))
				}
			}
		}
	}

	// Preserve bare strings without separators.
	if !strings.ContainsAny(v, "/\\") {
		return v
	}
	cleaned := filepath.Clean(v)
	return cleaned
}

func boolFromAny(v any, fallback bool) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		switch s {
		case "true", "1", "yes", "y", "on":
			return true
		case "false", "0", "no", "n", "off":
			return false
		}
	case int:
		return t != 0
	case int64:
		return t != 0
	case float64:
		return t != 0
	}
	return fallback
}

func getNested(root map[string]any, path ...string) any {
	cur := any(root)
	for _, seg := range path {
		m, ok := cur.(map[string]any)
		if ok {
			cur, ok = m[seg]
			if !ok {
				return nil
			}
			continue
		}
		m2, ok := cur.(map[any]any)
		if ok {
			cur, ok = m2[seg]
			if !ok {
				return nil
			}
			continue
		}
		return nil
	}
	return cur
}

func rawMap(v any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	if m, ok := v.(map[string]any); ok {
		out := make(map[string]any, len(m))
		for k, vv := range m {
			out[k] = vv
		}
		return out
	}
	if m, ok := v.(map[any]any); ok {
		out := make(map[string]any, len(m))
		for k, vv := range m {
			if ks, ok := k.(string); ok {
				out[ks] = vv
			}
		}
		return out
	}
	return map[string]any{}
}

func str(v any) string {
	if v == nil {
		return ""
	}
	s, ok := v.(string)
	if ok {
		return s
	}
	return ""
}

func strSlice(v any) []string {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case []any:
		out := make([]string, 0, len(t))
		for _, it := range t {
			if s, ok := it.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return append([]string(nil), t...)
	default:
		return nil
	}
}

func intFromAny(v any, def int) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		if err == nil {
			return n
		}
	}
	return def
}

func intPtr(v any) *int {
	switch t := v.(type) {
	case int:
		return &t
	case int64:
		n := int(t)
		return &n
	case float64:
		n := int(t)
		return &n
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		if err == nil {
			return &n
		}
	}
	return nil
}

func timeFromMsAny(v any, defMs int) time.Duration {
	n := intFromAny(v, defMs)
	return time.Duration(n) * time.Millisecond
}

func stateConcurrencyMap(v any) map[string]int {
	out := map[string]int{}
	m, ok := v.(map[string]any)
	if !ok {
		if m2, ok2 := v.(map[any]any); ok2 {
			m = map[string]any{}
			for k, vv := range m2 {
				ks, ok3 := k.(string)
				if !ok3 {
					continue
				}
				m[ks] = vv
			}
		}
	}
	for k, vv := range m {
		normalized := strings.ToLower(strings.TrimSpace(k))
		if normalized == "" {
			continue
		}
		n := intFromAny(vv, -1)
		if n <= 0 {
			continue
		}
		out[normalized] = n
	}
	return out
}
