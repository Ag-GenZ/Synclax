package codex

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppServer_TurnCompletedUsageAbsoluteAcrossTurnsComputesDeltas(t *testing.T) {
	tmp := t.TempDir()
	traceFile := filepath.Join(tmp, "codex.trace")
	fakeCodex := filepath.Join(tmp, "fake-codex")

	// This fake Codex emits `turn/completed` payloads where usage totals are cumulative
	// across turns (10 then 25). The app-server client should convert them into per-turn
	// deltas (10 then 15), so orchestrator-level totals do not double count.
	script := `#!/bin/sh
set -eu
trace="${TRACE_FILE:?}"
count=0

while IFS= read -r line; do
  count=$((count + 1))
  printf 'IN:%s\n' "$line" >> "$trace"

  case "$count" in
    1)
      printf '%s\n' '{"id":1,"result":{}}'
      ;;
    2)
      # initialized notification
      ;;
    3)
      printf '%s\n' '{"id":2,"result":{"thread":{"id":"thread-usage"}}}'
      ;;
    4)
      printf '%s\n' '{"id":3,"result":{"turn":{"id":"turn-1"}}}'
      printf '%s\n' '{"method":"turn/completed","params":{"usage":{"input_tokens":6,"output_tokens":4,"total_tokens":10}}}'
      ;;
    5)
      printf '%s\n' '{"id":4,"result":{"turn":{"id":"turn-2"}}}'
      printf '%s\n' '{"method":"turn/completed","params":{"usage":{"input_tokens":15,"output_tokens":10,"total_tokens":25}}}'
      exit 0
      ;;
    *)
      exit 0
      ;;
  esac
done
exit 0
`

	if err := os.WriteFile(fakeCodex, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}

	workspace := filepath.Join(tmp, "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}

	shQuote := func(s string) string {
		return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
	}
	command := "TRACE_FILE=" + shQuote(traceFile) + " " + shQuote(fakeCodex) + " app-server"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	srv := NewAppServer(AppServerOptions{
		Command:            command,
		ApprovalPolicy:     "never",
		ThreadSandbox:      "workspace-write",
		SandboxPolicy:      map[string]any{"type": "workspaceWrite"},
		ReadTimeout:        2 * time.Second,
		TurnTimeout:        5 * time.Second,
		TrackerKind:        "linear",
		LinearEndpoint:     "http://example.com/graphql",
		LinearAPIKey:       "x",
		DynamicToolTimeout: 2 * time.Second,
	})

	session, err := srv.StartSession(ctx, workspace, nil)
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	defer session.Close()

	first, err := srv.RunTurn(ctx, session, workspace, "t1", "hello", nil)
	if err != nil {
		t.Fatalf("RunTurn #1: %v", err)
	}
	if first.TotalTokens != 10 {
		t.Fatalf("expected first turn delta total=10, got %d", first.TotalTokens)
	}

	second, err := srv.RunTurn(ctx, session, workspace, "t2", "hello", nil)
	if err != nil {
		t.Fatalf("RunTurn #2: %v", err)
	}
	if second.TotalTokens != 15 {
		t.Fatalf("expected second turn delta total=15, got %d", second.TotalTokens)
	}
}
