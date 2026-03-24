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
			"api_key":      "$LINEAR_API_KEY",
		},
	})
	if err != nil {
		t.Fatalf("FromWorkflowConfig error: %v", err)
	}
	if cfg.Tracker.Params["api_key"] != "token123" {
		t.Fatalf("expected api key from env, got %q", cfg.Tracker.Params["api_key"])
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
			"kind": "linear",
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
			"kind": "linear",
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

func TestFromWorkflowConfig_ParamsEnvResolution(t *testing.T) {
	t.Setenv("MY_TOKEN", "resolved-value")

	cfg, err := FromWorkflowConfig(map[string]any{
		"tracker": map[string]any{
			"kind":    "linear",
			"api_key": "$MY_TOKEN",
			"custom":  "$MY_TOKEN",
		},
	})
	if err != nil {
		t.Fatalf("FromWorkflowConfig error: %v", err)
	}
	if cfg.Tracker.Params["api_key"] != "resolved-value" {
		t.Fatalf("expected resolved api_key, got %q", cfg.Tracker.Params["api_key"])
	}
	if cfg.Tracker.Params["custom"] != "resolved-value" {
		t.Fatalf("expected resolved custom param, got %q", cfg.Tracker.Params["custom"])
	}
}

func TestFromWorkflowConfig_GitHubTokenEnvResolution(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "gh-token")

	cfg, err := FromWorkflowConfig(map[string]any{
		"tracker": map[string]any{
			"kind":           "github",
			"project_owner":  "octo-org",
			"project_number": 7,
			"repository":     "octo-org/repo",
			"token":          "$GITHUB_TOKEN",
		},
	})
	if err != nil {
		t.Fatalf("FromWorkflowConfig error: %v", err)
	}
	if cfg.Tracker.Kind != "github" {
		t.Fatalf("expected github tracker kind, got %q", cfg.Tracker.Kind)
	}
	if cfg.Tracker.Params["token"] != "gh-token" {
		t.Fatalf("expected resolved github token, got %q", cfg.Tracker.Params["token"])
	}
}

func TestFromWorkflowConfig_EmptyKind_DefaultsToLinear(t *testing.T) {
	cfg, err := FromWorkflowConfig(map[string]any{
		"tracker": map[string]any{},
	})
	if err != nil {
		t.Fatalf("FromWorkflowConfig error: %v", err)
	}
	if cfg.Tracker.Kind != "linear" {
		t.Fatalf("expected default kind 'linear', got %q", cfg.Tracker.Kind)
	}
}
