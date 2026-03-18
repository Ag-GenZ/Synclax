package orchestrator

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func mapKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

