package codex

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppServer_RequestUserInput_IsAutoAnsweredInNeverMode(t *testing.T) {
	tmp := t.TempDir()
	traceFile := filepath.Join(tmp, "codex.trace")
	fakeCodex := filepath.Join(tmp, "fake-codex")

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
      # initialized
      ;;
    3)
      printf '%s\n' '{"id":2,"result":{"thread":{"id":"thread-ui"}}}'
      ;;
    4)
      printf '%s\n' '{"id":3,"result":{"turn":{"id":"turn-ui"}}}'

      printf '%s\n' '{"id":201,"method":"item/tool/requestUserInput","params":{"questions":[{"id":"q1","options":[{"label":"Approve this Session"},{"label":"Deny"}]}]}}'

      IFS= read -r resp
      printf 'CLIENT:%s\n' "$resp" >> "$trace"

      printf '%s\n' '{"method":"turn/completed","params":{"usage":{"input_tokens":1,"output_tokens":2,"total_tokens":3}}}'
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
		Command:        command,
		ApprovalPolicy: "never",
		ThreadSandbox:  "workspace-write",
		SandboxPolicy:  map[string]any{"type": "workspaceWrite"},
		ReadTimeout:    2 * time.Second,
		TurnTimeout:    5 * time.Second,
		LinearEndpoint: "http://example.com/graphql",
		LinearAPIKey:   "x",
		LinearTimeout:  2 * time.Second,
	})

	session, err := srv.StartSession(ctx, workspace, nil)
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	defer session.Close()

	_, err = srv.RunTurn(ctx, session, workspace, "ui", "hello", nil)
	if err != nil {
		t.Fatalf("RunTurn: %v", err)
	}

	traceBytes, err := os.ReadFile(traceFile)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}
	trace := string(traceBytes)
	if !strings.Contains(trace, `"answers"`) || !strings.Contains(trace, "Approve this Session") {
		t.Fatalf("expected client answers to include approval label, got=%s", trace)
	}
}
