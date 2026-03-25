package codex

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDynamicTools_ToolSpecs_AdvertisesLinearGraphQLContract(t *testing.T) {
	tools := newDynamicTools(dynamicToolsOptions{
		TrackerKind:    "linear",
		LinearEndpoint: "http://example.com/graphql",
		LinearAPIKey:   "x",
		Timeout:        time.Second,
	})

	specs := tools.ToolSpecs()
	if len(specs) != 1 {
		t.Fatalf("expected 1 tool spec, got %d", len(specs))
	}
	if specs[0]["name"] != linearGraphQLToolName {
		t.Fatalf("expected tool name %q, got %#v", linearGraphQLToolName, specs[0]["name"])
	}
	schema, _ := specs[0]["inputSchema"].(map[string]any)
	if schema == nil {
		t.Fatalf("expected inputSchema map")
	}
	if schema["type"] != "object" {
		t.Fatalf("expected inputSchema.type to be object, got %#v", schema["type"])
	}
}

func TestDynamicTools_ExecuteLinearGraphQL_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "x" {
			t.Fatalf("expected Authorization header x, got %q", r.Header.Get("Authorization"))
		}
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		if q, _ := req["query"].(string); !strings.Contains(q, "viewer") {
			t.Fatalf("unexpected query: %#v", req["query"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"viewer":{"id":"usr_123"}}}`))
	}))
	t.Cleanup(srv.Close)

	tools := newDynamicTools(dynamicToolsOptions{
		TrackerKind:    "linear",
		LinearEndpoint: srv.URL,
		LinearAPIKey:   "x",
		Timeout:        2 * time.Second,
		HTTPClient:     srv.Client(),
	})

	res := tools.Execute(linearGraphQLToolName, map[string]any{
		"query":     "query Viewer { viewer { id } }",
		"variables": map[string]any{"includeTeams": false},
	})

	if res["success"] != true {
		t.Fatalf("expected success=true, got %#v (output=%v)", res["success"], res["output"])
	}
	out, _ := res["output"].(string)
	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("expected output to be json, err=%v output=%q", err, out)
	}
	if data, _ := decoded["data"].(map[string]any); data == nil {
		t.Fatalf("expected data in output, got %#v", decoded)
	}
}

func TestDynamicTools_ExecuteLinearGraphQL_MarksGraphQLErrorsAsFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[{"message":"boom"}],"data":null}`))
	}))
	t.Cleanup(srv.Close)

	tools := newDynamicTools(dynamicToolsOptions{
		TrackerKind:    "linear",
		LinearEndpoint: srv.URL,
		LinearAPIKey:   "x",
		Timeout:        2 * time.Second,
		HTTPClient:     srv.Client(),
	})

	res := tools.Execute(linearGraphQLToolName, map[string]any{
		"query": "mutation BadMutation { nope }",
	})
	if res["success"] != false {
		t.Fatalf("expected success=false, got %#v", res["success"])
	}
}

func TestDynamicTools_ToolSpecs_AdvertisesGitHubGraphQLContract(t *testing.T) {
	tools := newDynamicTools(dynamicToolsOptions{
		TrackerKind:    "github",
		GitHubEndpoint: "https://api.github.com/graphql",
		GitHubToken:    "x",
		Timeout:        time.Second,
	})

	specs := tools.ToolSpecs()
	if len(specs) != 1 {
		t.Fatalf("expected 1 tool spec, got %d", len(specs))
	}
	if specs[0]["name"] != githubGraphQLToolName {
		t.Fatalf("expected tool name %q, got %#v", githubGraphQLToolName, specs[0]["name"])
	}
}

func TestDynamicTools_ExecuteGitHubGraphQL_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer x" {
			t.Fatalf("expected GitHub Bearer header, got %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"viewer":{"login":"octocat"}}}`))
	}))
	t.Cleanup(srv.Close)

	tools := newDynamicTools(dynamicToolsOptions{
		TrackerKind:    "github",
		GitHubEndpoint: srv.URL,
		GitHubToken:    "x",
		Timeout:        2 * time.Second,
		HTTPClient:     srv.Client(),
	})

	res := tools.Execute(githubGraphQLToolName, map[string]any{
		"query": "query Viewer { viewer { login } }",
	})
	if res["success"] != true {
		t.Fatalf("expected success=true, got %#v", res["success"])
	}
}

func TestDynamicTools_ExecuteGitHubGraphQL_MarksGraphQLErrorsAsFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[{"message":"boom"}],"data":null}`))
	}))
	t.Cleanup(srv.Close)

	tools := newDynamicTools(dynamicToolsOptions{
		TrackerKind:    "github",
		GitHubEndpoint: srv.URL,
		GitHubToken:    "x",
		Timeout:        2 * time.Second,
		HTTPClient:     srv.Client(),
	})

	res := tools.Execute(githubGraphQLToolName, map[string]any{
		"query": "query Viewer { viewer { login } }",
	})
	if res["success"] != false {
		t.Fatalf("expected success=false, got %#v", res["success"])
	}
}

func TestDynamicTools_ExecuteGitHubGraphQL_MissingAuthFails(t *testing.T) {
	tools := newDynamicTools(dynamicToolsOptions{
		TrackerKind:    "github",
		GitHubEndpoint: "https://api.github.com/graphql",
		Timeout:        2 * time.Second,
	})

	res := tools.Execute(githubGraphQLToolName, map[string]any{
		"query": "query Viewer { viewer { login } }",
	})
	if res["success"] != false {
		t.Fatalf("expected success=false, got %#v", res["success"])
	}
	out, _ := res["output"].(string)
	if !strings.Contains(out, "missing GitHub auth") {
		t.Fatalf("expected missing auth message, got %q", out)
	}
}
