package codex

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppServer_LinearGraphQL_ToolCallProtocol(t *testing.T) {
	var gotAuth string

	linearSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"viewer":{"id":"usr_999"}}}`))
	}))
	t.Cleanup(linearSrv.Close)

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
      # initialized notification (no response)
      ;;
    3)
      printf '%s\n' '{"id":2,"result":{"thread":{"id":"thread-1"}}}'
      ;;
    4)
      printf '%s\n' '{"id":3,"result":{"turn":{"id":"turn-1"}}}'

      # Ask the client to execute a dynamic tool call, then capture the reply.
      printf '%s\n' '{"id":102,"method":"item/tool/call","params":{"name":"linear_graphql","arguments":{"query":"query Viewer { viewer { id } }","variables":{"includeTeams":false}}}}'

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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	shQuote := func(s string) string {
		return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
	}

	command := "TRACE_FILE=" + shQuote(traceFile) + " " + shQuote(fakeCodex) + " app-server"

	srv := NewAppServer(AppServerOptions{
		Command:        command,
		ApprovalPolicy: "never",
		ThreadSandbox:  "workspace-write",
		SandboxPolicy:  map[string]any{"type": "workspaceWrite"},
		ReadTimeout:    2 * time.Second,
		TurnTimeout:    5 * time.Second,
		LinearEndpoint: linearSrv.URL,
		LinearAPIKey:   "x",
		LinearTimeout:  2 * time.Second,
	})

	session, err := srv.StartSession(ctx, workspace)
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	defer session.Close()

	_, err = srv.RunTurn(ctx, session, workspace, "MT-1: test", "hello", nil)
	if err != nil {
		t.Fatalf("RunTurn: %v", err)
	}

	if gotAuth != "x" {
		t.Fatalf("expected Linear Authorization header x, got %q", gotAuth)
	}

	traceBytes, err := os.ReadFile(traceFile)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}
	trace := string(traceBytes)

	if !strings.Contains(trace, `"method":"thread/start"`) || !strings.Contains(trace, `"dynamicTools"`) {
		t.Fatalf("expected thread/start to include dynamicTools, trace=%s", trace)
	}

	var toolReplyLine string
	for _, line := range strings.Split(trace, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "CLIENT:") {
			toolReplyLine = strings.TrimPrefix(line, "CLIENT:")
			break
		}
	}
	if strings.TrimSpace(toolReplyLine) == "" {
		t.Fatalf("missing tool reply line in trace: %s", trace)
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(toolReplyLine), &decoded); err != nil {
		t.Fatalf("decode tool reply: %v line=%q", err, toolReplyLine)
	}
	if id, _ := decoded["id"].(float64); int(id) != 102 {
		t.Fatalf("expected reply id=102, got %#v", decoded["id"])
	}
	result, _ := decoded["result"].(map[string]any)
	if result == nil {
		t.Fatalf("expected result object, got %#v", decoded["result"])
	}
	if result["success"] != true {
		t.Fatalf("expected tool success=true, got %#v", result["success"])
	}
	out, _ := result["output"].(string)
	var outDecoded map[string]any
	if err := json.Unmarshal([]byte(out), &outDecoded); err != nil {
		t.Fatalf("expected output json, err=%v output=%q", err, out)
	}
}
