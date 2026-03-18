package linear

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wibus-wee/synclax/pkg/symphony/config"
)

func TestFetchIssuesByStates_EmptyIsNoOp(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	t.Cleanup(srv.Close)

	c, err := New(Options{
		Endpoint:     srv.URL,
		APIKey:       "x",
		ProjectSlug:  "proj",
		ActiveStates: []string{"Todo"},
		PageSize:     1,
		Timeout:      2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := c.FetchIssuesByStates(context.Background(), nil)
	if err != nil {
		t.Fatalf("FetchIssuesByStates error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %#v", got)
	}
	if atomic.LoadInt32(&calls) != 0 {
		t.Fatalf("expected no HTTP calls, got %d", calls)
	}
}

func TestFetchCandidateIssues_PaginatesAndNormalizes(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		var req struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(req.Query, "project: { slugId: { eq: $projectSlug } }") {
			t.Fatalf("expected slugId filter, query=%s", req.Query)
		}
		after, _ := req.Variables["after"].(string)
		if req.Variables["after"] == nil {
			after = ""
		}

		w.Header().Set("Content-Type", "application/json")
		switch after {
		case "":
			_, _ = w.Write([]byte(`{
  "data": {
    "issues": {
      "nodes": [{
        "id": "i1",
        "identifier": "ABC-1",
        "title": "First",
        "description": "d",
        "priority": 2,
        "url": "https://linear.app/x",
        "branchName": "branch",
        "state": {"name": "Todo"},
        "labels": {"nodes": [{"name": "Bug"}]},
        "inverseRelations": {"nodes": [{
          "type": "blocks",
          "issue": {"id": "b1", "identifier": "ABC-0", "title":"", "description": null, "priority": null, "url": null, "branchName": null, "state": {"name": "Done"}, "labels":{"nodes":[]}, "inverseRelations":{"nodes":[]}, "createdAt": null, "updatedAt": null}
        }]},
        "createdAt": "2025-01-01T00:00:00Z",
        "updatedAt": "2025-01-02T00:00:00Z"
      }],
      "pageInfo": {"hasNextPage": true, "endCursor": "c1"}
    }
  }
}`))
		case "c1":
			_, _ = w.Write([]byte(`{
  "data": {
    "issues": {
      "nodes": [{
        "id": "i2",
        "identifier": "ABC-2",
        "title": "Second",
        "description": null,
        "priority": null,
        "url": null,
        "branchName": null,
        "state": {"name": "In Progress"},
        "labels": {"nodes": []},
        "inverseRelations": {"nodes": []},
        "createdAt": "2025-01-03T00:00:00Z",
        "updatedAt": "2025-01-04T00:00:00Z"
      }],
      "pageInfo": {"hasNextPage": false, "endCursor": ""}
    }
  }
}`))
		default:
			t.Fatalf("unexpected after cursor: %q", after)
		}
	}))
	t.Cleanup(srv.Close)

	c, err := New(Options{
		Endpoint:     srv.URL,
		APIKey:       "x",
		ProjectSlug:  "proj",
		ActiveStates: []string{"Todo", "In Progress"},
		PageSize:     1,
		Timeout:      2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	issues, err := c.FetchCandidateIssues(context.Background())
	if err != nil {
		t.Fatalf("FetchCandidateIssues error: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %#v", issues)
	}
	if issues[0].Identifier != "ABC-1" || issues[1].Identifier != "ABC-2" {
		t.Fatalf("unexpected order: %#v", issues)
	}
	if len(issues[0].Labels) != 1 || issues[0].Labels[0] != "bug" {
		t.Fatalf("expected lowercase label, got %#v", issues[0].Labels)
	}
	if len(issues[0].BlockedBy) != 1 || issues[0].BlockedBy[0].State == nil || *issues[0].BlockedBy[0].State != "Done" {
		t.Fatalf("expected blocker normalization, got %#v", issues[0].BlockedBy)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestFetchIssueStatesByIDs_UsesIDTyping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query string `json:"query"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if !strings.Contains(req.Query, "($ids: [ID!]!)") {
			t.Fatalf("expected [ID!] typing, query=%s", req.Query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"issues":{"nodes":[{"id":"i1","identifier":"ABC-1","state":{"name":"Done"}}]}}}`))
	}))
	t.Cleanup(srv.Close)

	c, err := New(Options{
		Endpoint:     srv.URL,
		APIKey:       "x",
		ProjectSlug:  "proj",
		ActiveStates: []string{"Todo"},
		Timeout:      2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	issues, err := c.FetchIssueStatesByIDs(context.Background(), []string{"i1"})
	if err != nil {
		t.Fatalf("FetchIssueStatesByIDs error: %v", err)
	}
	if len(issues) != 1 || issues[0].State != "Done" {
		t.Fatalf("unexpected result: %#v", issues)
	}
}

func TestNewFromConfig_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewFromConfig(config.TrackerConfig{
		ActiveStates: []string{"Todo"},
		PageSize:     10,
		Timeout:      2 * time.Second,
		Params: map[string]any{
			"endpoint":     srv.URL,
			"api_key":      "test-key",
			"project_slug": "proj",
		},
	})
	if err != nil {
		t.Fatalf("NewFromConfig error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewFromConfig_MissingAPIKey(t *testing.T) {
	t.Setenv("LINEAR_API_KEY", "")
	_, err := NewFromConfig(config.TrackerConfig{
		Params: map[string]any{
			"project_slug": "proj",
		},
	})
	if err == nil {
		t.Fatal("expected error for missing api key")
	}
}

func TestNewFromConfig_DefaultEndpoint(t *testing.T) {
	c, err := NewFromConfig(config.TrackerConfig{
		ActiveStates: []string{"Todo"},
		PageSize:     10,
		Timeout:      2 * time.Second,
		Params: map[string]any{
			"api_key":      "test-key",
			"project_slug": "proj",
		},
	})
	if err != nil {
		t.Fatalf("NewFromConfig error: %v", err)
	}
	if c.endpoint != "https://api.linear.app/graphql" {
		t.Fatalf("expected default endpoint, got %q", c.endpoint)
	}
}

func TestNewFromConfig_EnvFallback(t *testing.T) {
	t.Setenv("LINEAR_API_KEY", "env-token")
	c, err := NewFromConfig(config.TrackerConfig{
		ActiveStates: []string{"Todo"},
		PageSize:     10,
		Timeout:      2 * time.Second,
		Params: map[string]any{
			"endpoint":     "https://example.com/graphql",
			"project_slug": "proj",
			// api_key intentionally omitted
		},
	})
	if err != nil {
		t.Fatalf("NewFromConfig error: %v", err)
	}
	if c.apiKey != "env-token" {
		t.Fatalf("expected env fallback api key, got %q", c.apiKey)
	}
}
