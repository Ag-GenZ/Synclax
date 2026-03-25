package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	symphonycfg "github.com/wibus-wee/synclax/pkg/symphony/config"
	githubtracker "github.com/wibus-wee/synclax/pkg/symphony/tracker/github"
)

func TestOrchestrator_Provider_DefaultAndExplicitCodex(t *testing.T) {
	linearSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"issues":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}`))
	}))
	t.Cleanup(linearSrv.Close)

	root := t.TempDir()
	workspaceRoot := filepath.Join(root, "workspaces")
	if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
		t.Fatalf("mkdir workspace root: %v", err)
	}

	cases := []struct {
		name     string
		provider string
	}{
		{name: "default_provider_kind", provider: ""},
		{name: "explicit_codex_provider", provider: "provider:\n  kind: codex\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			workflow := fmt.Sprintf(`---
tracker:
  kind: linear
  endpoint: %q
  api_key: x
  project_slug: proj
workspace:
  root: %q
%s---`, linearSrv.URL, workspaceRoot, tc.provider)

			workflowPath := filepath.Join(root, tc.name+".WORKFLOW.md")
			if err := os.WriteFile(workflowPath, []byte(workflow), 0o644); err != nil {
				t.Fatalf("write workflow: %v", err)
			}

			orch, err := New(Options{WorkflowPath: workflowPath})
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			if err := orch.runtime.Start(ctx); err != nil {
				t.Fatalf("runtime.Start: %v", err)
			}

			disableHTTP := -1
			if err := orch.applyRuntimeLocked(&disableHTTP); err != nil {
				t.Fatalf("applyRuntimeLocked: %v", err)
			}

			snap := orch.Snapshot()
			if _, ok := snap["agent_totals"]; !ok {
				t.Fatalf("expected agent_totals in snapshot, got keys=%v", mapKeys(snap))
			}
			if _, ok := snap["codex_totals"]; ok {
				t.Fatalf("unexpected codex_totals in snapshot, got keys=%v", mapKeys(snap))
			}
		})
	}
}

func TestOrchestrator_Provider_GitHubTrackerSmoke(t *testing.T) {
	githubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(req.Query, "ResolveProject"):
			_, _ = w.Write([]byte(`{"data":{"organization":{"login":"octo-org","projectV2":{"id":"PVT_proj","title":"Tracker","field":{"__typename":"ProjectV2SingleSelectField","id":"PVTSSF_status","name":"Status","options":[{"id":"todo","name":"Todo"},{"id":"ip","name":"In Progress"}]}}},"user":null,"repository":{"id":"R_repo","name":"repo","owner":{"login":"octo-org"}}}}`))
		default:
			t.Fatalf("unexpected query: %s", req.Query)
		}
	}))
	t.Cleanup(githubSrv.Close)

	root := t.TempDir()
	workspaceRoot := filepath.Join(root, "workspaces")
	if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
		t.Fatalf("mkdir workspace root: %v", err)
	}

	workflow := fmt.Sprintf(`---
tracker:
  kind: github
  endpoint: %q
  token: x
  project_owner: octo-org
  project_number: 7
  repository: octo-org/repo
workspace:
  root: %q
---`, githubSrv.URL, workspaceRoot)

	workflowPath := filepath.Join(root, "github.WORKFLOW.md")
	if err := os.WriteFile(workflowPath, []byte(workflow), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	orch, err := New(Options{WorkflowPath: workflowPath})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := orch.runtime.Start(ctx); err != nil {
		t.Fatalf("runtime.Start: %v", err)
	}

	disableHTTP := -1
	if err := orch.applyRuntimeLocked(&disableHTTP); err != nil {
		t.Fatalf("applyRuntimeLocked: %v", err)
	}

	if _, ok := orch.tracker.(*githubtracker.Client); !ok {
		t.Fatalf("expected github tracker client, got %T", orch.tracker)
	}
}

func TestNewTracker_UnsupportedKind(t *testing.T) {
	_, err := newTracker(symphonycfg.TrackerConfig{Kind: "jira"})
	if err == nil {
		t.Fatal("expected unsupported tracker kind error")
	}
}

func mapKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
