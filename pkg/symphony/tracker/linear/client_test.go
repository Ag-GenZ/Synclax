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

func TestEnsureSynclaxWorkflowCreatesMissingStates(t *testing.T) {
	var created []string
	var archived []string
	var moved []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(req.Query, "query ProjectTeamStates"):
			_, _ = w.Write([]byte(`{
  "data": {
    "projects": {
      "nodes": [{
        "teams": {
          "nodes": [{
            "id": "team-1",
            "states": {
              "nodes": [
                {"id": "s1", "name": "Todo", "type": "unstarted", "position": 1},
                {"id": "s2", "name": "Done", "type": "completed", "position": 2},
                {"id": "s3", "name": "In Review", "type": "started", "position": 3}
              ]
            }
          }]
        }
      }]
    }
  }
}`))
		case strings.Contains(req.Query, "mutation WorkflowStateCreate"):
			input, _ := req.Variables["input"].(map[string]any)
			name, _ := input["name"].(string)
			created = append(created, name)
			_, _ = w.Write([]byte(`{"data":{"workflowStateCreate":{"success":true,"workflowState":{"id":"new-` + name + `","name":"` + name + `","type":"started","position":3}}}}`))
		case strings.Contains(req.Query, "query IssuesByStateIDs"):
			_, _ = w.Write([]byte(`{
  "data": {
    "issues": {
      "nodes": [
        {
          "id": "issue-1",
          "identifier": "KUN-1",
          "state": {"id": "s3", "name": "In Review"}
        }
      ],
      "pageInfo": {"hasNextPage": false, "endCursor": ""}
    }
  }
}`))
		case strings.Contains(req.Query, "mutation IssueUpdateState"):
			input, _ := req.Variables["input"].(map[string]any)
			stateID, _ := input["stateId"].(string)
			moved = append(moved, stateID)
			_, _ = w.Write([]byte(`{"data":{"issueUpdate":{"success":true,"issue":{"id":"issue-1","identifier":"KUN-1","state":{"id":"` + stateID + `","name":"Human Review"}}}}}`))
		case strings.Contains(req.Query, "mutation WorkflowStateArchive"):
			stateID, _ := req.Variables["id"].(string)
			archived = append(archived, stateID)
			_, _ = w.Write([]byte(`{"data":{"workflowStateArchive":{"success":true,"entity":{"id":"` + stateID + `","name":"archived"}}}}`))
		default:
			t.Fatalf("unexpected query: %s", req.Query)
		}
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

	if err := c.EnsureSynclaxWorkflow(context.Background()); err != nil {
		t.Fatalf("EnsureSynclaxWorkflow error: %v", err)
	}

	expected := []string{"Backlog", "In Progress", "Human Review", "Merging", "Rework"}
	if len(created) != len(expected) {
		t.Fatalf("expected %d created states, got %d (%v)", len(expected), len(created), created)
	}
	for i, want := range expected {
		if created[i] != want {
			t.Fatalf("expected created[%d]=%q, got %q", i, want, created[i])
		}
	}
	if len(archived) != 1 || archived[0] != "s3" {
		t.Fatalf("expected archived state s3, got %v", archived)
	}
	if len(moved) != 1 || moved[0] != "new-Human Review" {
		t.Fatalf("expected moved issue into Human Review, got %v", moved)
	}
}

func TestEnsureSynclaxWorkflowNoopWhenStatesAlreadyExist(t *testing.T) {
	var mutationCalls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(req.Query, "query ProjectTeamStates"):
			_, _ = w.Write([]byte(`{
  "data": {
    "projects": {
      "nodes": [{
        "teams": {
          "nodes": [{
            "id": "team-1",
            "states": {
              "nodes": [
                {"id": "s1", "name": "Backlog", "type": "backlog", "position": 1},
                {"id": "s2", "name": "Todo", "type": "unstarted", "position": 2},
                {"id": "s3", "name": "In Progress", "type": "started", "position": 3},
                {"id": "s4", "name": "Human Review", "type": "started", "position": 4},
                {"id": "s5", "name": "Merging", "type": "started", "position": 5},
                {"id": "s6", "name": "Rework", "type": "started", "position": 6},
                {"id": "s7", "name": "Done", "type": "completed", "position": 7}
              ]
            }
          }]
        }
      }]
    }
  }
}`))
		case strings.Contains(req.Query, "mutation WorkflowStateCreate"), strings.Contains(req.Query, "mutation WorkflowStateArchive"):
			atomic.AddInt32(&mutationCalls, 1)
			t.Fatalf("did not expect workflow state mutation")
		default:
			t.Fatalf("unexpected query: %s", req.Query)
		}
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

	if err := c.EnsureSynclaxWorkflow(context.Background()); err != nil {
		t.Fatalf("EnsureSynclaxWorkflow error: %v", err)
	}
	if atomic.LoadInt32(&mutationCalls) != 0 {
		t.Fatalf("expected no mutations, got %d", mutationCalls)
	}
}

func TestEnsureSynclaxWorkflow_AllowsDecimalStatePositions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(req.Query, "query ProjectTeamStates"):
			_, _ = w.Write([]byte(`{
  "data": {
    "projects": {
      "nodes": [{
        "teams": {
          "nodes": [{
            "id": "team-1",
            "states": {
              "nodes": [
                {"id": "s1", "name": "Backlog", "type": "backlog", "position": 0},
                {"id": "s2", "name": "Todo", "type": "unstarted", "position": 1},
                {"id": "s3", "name": "In Progress", "type": "started", "position": 1000},
                {"id": "s4", "name": "Human Review", "type": "started", "position": 2000},
                {"id": "s5", "name": "Merging", "type": "started", "position": 3000},
                {"id": "s6", "name": "Rework", "type": "started", "position": 4000},
                {"id": "s7", "name": "Done", "type": "completed", "position": 4946.1}
              ]
            }
          }]
        }
      }]
    }
  }
}`))
		case strings.Contains(req.Query, "mutation WorkflowStateCreate"),
			strings.Contains(req.Query, "mutation WorkflowStateArchive"),
			strings.Contains(req.Query, "mutation IssueUpdateState"):
			t.Fatalf("did not expect workflow mutations, query=%s", req.Query)
		case strings.Contains(req.Query, "query IssuesByStateIDs"):
			t.Fatalf("did not expect issue rehome query")
		default:
			t.Fatalf("unexpected query: %s", req.Query)
		}
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

	if err := c.EnsureSynclaxWorkflow(context.Background()); err != nil {
		t.Fatalf("EnsureSynclaxWorkflow error: %v", err)
	}
}
