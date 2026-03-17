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
	Codex     CodexConfig
	Server    ServerConfig
}

type TrackerConfig struct {
	Kind           string
	Endpoint       string
	APIKey         string
	ProjectSlug    string
	ActiveStates   []string
	TerminalStates []string
	PageSize       int
	Timeout        time.Duration
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
}

type CodexConfig struct {
	Command           string
	ApprovalPolicy    any
	ThreadSandbox     any
	TurnSandboxPolicy any
	TurnTimeout       time.Duration
	ReadTimeout       time.Duration
	StallTimeout      time.Duration
}

type ServerConfig struct {
	Port *int
}

var (
	ErrUnsupportedTrackerKind    = errors.New("unsupported_tracker_kind")
	ErrMissingTrackerAPIKey      = errors.New("missing_tracker_api_key")
	ErrMissingTrackerProjectSlug = errors.New("missing_tracker_project_slug")
	ErrMissingCodexCommand       = errors.New("missing_codex_command")
)

func FromWorkflowConfig(cfg map[string]any) (EffectiveConfig, error) {
	effective := EffectiveConfig{
		Tracker: TrackerConfig{
			Kind:           str(getNested(cfg, "tracker", "kind")),
			Endpoint:       str(getNested(cfg, "tracker", "endpoint")),
			APIKey:         str(getNested(cfg, "tracker", "api_key")),
			ProjectSlug:    str(getNested(cfg, "tracker", "project_slug")),
			ActiveStates:   strSlice(getNested(cfg, "tracker", "active_states")),
			TerminalStates: strSlice(getNested(cfg, "tracker", "terminal_states")),
			PageSize:       intFromAny(getNested(cfg, "tracker", "page_size"), 50),
			Timeout:        timeFromMsAny(getNested(cfg, "tracker", "timeout_ms"), 30000),
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
		},
		Codex: CodexConfig{
			Command:           str(getNested(cfg, "codex", "command")),
			ApprovalPolicy:    getNested(cfg, "codex", "approval_policy"),
			ThreadSandbox:     getNested(cfg, "codex", "thread_sandbox"),
			TurnSandboxPolicy: getNested(cfg, "codex", "turn_sandbox_policy"),
			TurnTimeout:       timeFromMsAny(getNested(cfg, "codex", "turn_timeout_ms"), 3600000),
			ReadTimeout:       timeFromMsAny(getNested(cfg, "codex", "read_timeout_ms"), 5000),
			StallTimeout:      timeFromMsAny(getNested(cfg, "codex", "stall_timeout_ms"), 300000),
		},
		Server: ServerConfig{
			Port: intPtr(getNested(cfg, "server", "port")),
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
	if cfg.Tracker.Kind == "linear" && cfg.Tracker.Endpoint == "" {
		cfg.Tracker.Endpoint = "https://api.linear.app/graphql"
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
	if cfg.Tracker.APIKey == "" {
		// canonical env for Linear
		cfg.Tracker.APIKey = os.Getenv("LINEAR_API_KEY")
	} else {
		cfg.Tracker.APIKey = resolveEnvToken(cfg.Tracker.APIKey)
	}

	cfg.Workspace.Root = expandPath(resolveEnvToken(cfg.Workspace.Root))
}

func normalize(cfg *EffectiveConfig) {
	cfg.Tracker.Kind = strings.TrimSpace(cfg.Tracker.Kind)

	for i := range cfg.Tracker.ActiveStates {
		cfg.Tracker.ActiveStates[i] = strings.TrimSpace(cfg.Tracker.ActiveStates[i])
	}
	for i := range cfg.Tracker.TerminalStates {
		cfg.Tracker.TerminalStates[i] = strings.TrimSpace(cfg.Tracker.TerminalStates[i])
	}
}

func validate(cfg EffectiveConfig) error {
	if cfg.Tracker.Kind != "linear" {
		return ErrUnsupportedTrackerKind
	}
	if strings.TrimSpace(cfg.Tracker.APIKey) == "" {
		return ErrMissingTrackerAPIKey
	}
	if strings.TrimSpace(cfg.Tracker.ProjectSlug) == "" {
		return ErrMissingTrackerProjectSlug
	}
	if strings.TrimSpace(cfg.Codex.Command) == "" {
		return ErrMissingCodexCommand
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
