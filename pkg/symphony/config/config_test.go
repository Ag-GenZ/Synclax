package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFromWorkflowConfig_DefaultsAndEnvResolution(t *testing.T) {
	t.Setenv("LINEAR_API_KEY", "token123")

	cfg, err := FromWorkflowConfig(map[string]any{
		"tracker": map[string]any{
			"kind":         "linear",
			"project_slug": "proj",
			// api_key intentionally omitted to use LINEAR_API_KEY
		},
	})
	if err != nil {
		t.Fatalf("FromWorkflowConfig error: %v", err)
	}
	if cfg.Tracker.APIKey != "token123" {
		t.Fatalf("expected api key from env, got %q", cfg.Tracker.APIKey)
	}
	if cfg.Tracker.Endpoint == "" {
		t.Fatal("expected default endpoint")
	}
	if cfg.Workspace.Root == "" {
		t.Fatal("expected workspace root default")
	}
	if cfg.Codex.Command == "" {
		t.Fatal("expected codex command default")
	}
}

func TestFromWorkflowConfig_PathExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("no home dir")
	}
	cfg, err := FromWorkflowConfig(map[string]any{
		"tracker": map[string]any{
			"kind":         "linear",
			"project_slug": "proj",
			"api_key":      "x",
		},
		"workspace": map[string]any{
			"root": "~/.symphony_test",
		},
	})
	if err != nil {
		t.Fatalf("FromWorkflowConfig error: %v", err)
	}
	if !strings.HasPrefix(cfg.Workspace.Root, home) {
		t.Fatalf("expected expanded home prefix, got %q", cfg.Workspace.Root)
	}
	if filepath.Base(cfg.Workspace.Root) != ".symphony_test" {
		t.Fatalf("unexpected root: %q", cfg.Workspace.Root)
	}
}

func TestFromWorkflowConfig_StateConcurrencyMap(t *testing.T) {
	cfg, err := FromWorkflowConfig(map[string]any{
		"tracker": map[string]any{
			"kind":         "linear",
			"project_slug": "proj",
			"api_key":      "x",
		},
		"agent": map[string]any{
			"max_concurrent_agents_by_state": map[string]any{
				"In Progress": "2",
				"Todo":        1,
				"Bad":         0,
			},
		},
	})
	if err != nil {
		t.Fatalf("FromWorkflowConfig error: %v", err)
	}
	if cfg.Agent.MaxConcurrentAgentsByState["in progress"] != 2 {
		t.Fatalf("expected normalized key, got %#v", cfg.Agent.MaxConcurrentAgentsByState)
	}
	if cfg.Agent.MaxConcurrentAgentsByState["todo"] != 1 {
		t.Fatalf("expected todo=1, got %#v", cfg.Agent.MaxConcurrentAgentsByState)
	}
	if _, ok := cfg.Agent.MaxConcurrentAgentsByState["bad"]; ok {
		t.Fatalf("expected invalid entries ignored, got %#v", cfg.Agent.MaxConcurrentAgentsByState)
	}
}
