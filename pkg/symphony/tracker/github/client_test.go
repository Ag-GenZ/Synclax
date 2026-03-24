package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wibus-wee/synclax/pkg/symphony/config"
)

func TestNewFromConfig_ResolvesProjectAndStateField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer env-token" {
			t.Fatalf("expected Bearer token auth, got %q", got)
		}
		var req struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(req.Query, "ResolveProject") {
			t.Fatalf("unexpected query: %s", req.Query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"organization":{"login":"octo-org","projectV2":{"id":"PVT_proj","title":"Tracker","field":{"__typename":"ProjectV2SingleSelectField","id":"PVTSSF_status","name":"Status","options":[{"id":"todo","name":"Todo"},{"id":"ip","name":"In Progress"}]}}},"user":null,"repository":{"id":"R_repo","name":"repo","owner":{"login":"octo-org"}}}}`))
	}))
	t.Cleanup(srv.Close)

	t.Setenv("GITHUB_TOKEN", "env-token")

	client, err := NewFromConfig(config.TrackerConfig{
		ActiveStates: []string{"Todo", "In Progress"},
		PageSize:     10,
		Timeout:      2 * time.Second,
		Params: map[string]any{
			"endpoint":       srv.URL,
			"project_owner":  "octo-org",
			"project_number": 7,
			"repository":     "octo-org/repo",
		},
	})
	if err != nil {
		t.Fatalf("NewFromConfig error: %v", err)
	}
	if client.projectID != "PVT_proj" {
		t.Fatalf("expected project id cache, got %q", client.projectID)
	}
	if client.stateFieldID != "PVTSSF_status" {
		t.Fatalf("expected state field id cache, got %q", client.stateFieldID)
	}
	if client.optionIDs["Todo"] != "todo" {
		t.Fatalf("expected Todo option cache, got %#v", client.optionIDs)
	}
}

func TestNewFromConfig_FieldMustBeSingleSelect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"organization":{"login":"octo-org","projectV2":{"id":"PVT_proj","title":"Tracker","field":{"__typename":"ProjectV2Field"}}},"user":null,"repository":{"id":"R_repo","name":"repo","owner":{"login":"octo-org"}}}}`))
	}))
	t.Cleanup(srv.Close)

	_, err := NewFromConfig(config.TrackerConfig{
		Timeout: 2 * time.Second,
		Params: map[string]any{
			"endpoint":       srv.URL,
			"token":          "x",
			"project_owner":  "octo-org",
			"project_number": 7,
			"repository":     "octo-org/repo",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "single-select") {
		t.Fatalf("expected single-select validation error, got %v", err)
	}
}

func TestFetchCandidateIssues_PaginatesFiltersAndCaches(t *testing.T) {
	var projectItemCalls int
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
		case strings.Contains(req.Query, "ResolveProject"):
			_, _ = w.Write([]byte(`{"data":{"organization":{"login":"octo-org","projectV2":{"id":"PVT_proj","title":"Tracker","field":{"__typename":"ProjectV2SingleSelectField","id":"PVTSSF_status","name":"Status","options":[{"id":"todo","name":"Todo"},{"id":"ip","name":"In Progress"},{"id":"done","name":"Done"}]}}},"user":null,"repository":{"id":"R_repo","name":"repo","owner":{"login":"octo-org"}}}}`))
		case strings.Contains(req.Query, "ProjectItems"):
			projectItemCalls++
			after, _ := req.Variables["after"].(string)
			if req.Variables["after"] == nil {
				after = ""
			}
			switch after {
			case "":
				_, _ = w.Write([]byte(`{"data":{"node":{"items":{"nodes":[
{"id":"ITEM_1","fieldValues":{"nodes":[{"__typename":"ProjectV2ItemFieldSingleSelectValue","name":"Todo","optionId":"todo","field":{"__typename":"ProjectV2SingleSelectField","id":"PVTSSF_status","name":"Status"}}]},"content":{"__typename":"Issue","id":"ISSUE_1","number":1,"title":"First","body":"Primary body","url":"https://github.com/octo-org/repo/issues/1","state":"OPEN","createdAt":"2025-01-01T00:00:00Z","updatedAt":"2025-01-02T00:00:00Z","repository":{"name":"repo","owner":{"login":"octo-org"}},"labels":{"nodes":[{"name":"Bug"}]},"blockedBy":{"nodes":[{"id":"ISSUE_9","number":9,"url":"https://github.com/octo-org/repo/issues/9","state":"OPEN","repository":{"name":"repo","owner":{"login":"octo-org"}},"projectItems":{"nodes":[{"id":"ITEM_9","project":{"id":"PVT_proj"},"fieldValues":{"nodes":[{"__typename":"ProjectV2ItemFieldSingleSelectValue","name":"Done","optionId":"done","field":{"__typename":"ProjectV2SingleSelectField","id":"PVTSSF_status","name":"Status"}}]}}]}},{"id":"ISSUE_10","number":10,"url":"https://github.com/octo-org/repo/issues/10","state":"OPEN","repository":{"name":"repo","owner":{"login":"octo-org"}},"projectItems":{"nodes":[]}}]}}},
{"id":"ITEM_other","fieldValues":{"nodes":[{"__typename":"ProjectV2ItemFieldSingleSelectValue","name":"Todo","optionId":"todo","field":{"__typename":"ProjectV2SingleSelectField","id":"PVTSSF_status","name":"Status"}}]},"content":{"__typename":"Issue","id":"ISSUE_other","number":30,"title":"Other repo","body":"","url":"https://github.com/other/repo/issues/30","state":"OPEN","createdAt":"2025-01-01T00:00:00Z","updatedAt":"2025-01-01T01:00:00Z","repository":{"name":"repo","owner":{"login":"other"}},"labels":{"nodes":[]},"blockedBy":{"nodes":[]}}},
{"id":"ITEM_pr","fieldValues":{"nodes":[{"__typename":"ProjectV2ItemFieldSingleSelectValue","name":"Todo","optionId":"todo","field":{"__typename":"ProjectV2SingleSelectField","id":"PVTSSF_status","name":"Status"}}]},"content":{"__typename":"PullRequest","id":"PR_1"}}
],"pageInfo":{"hasNextPage":true,"endCursor":"c1"}}}}}`))
			case "c1":
				_, _ = w.Write([]byte(`{"data":{"node":{"items":{"nodes":[
{"id":"ITEM_2","fieldValues":{"nodes":[{"__typename":"ProjectV2ItemFieldSingleSelectValue","name":"In Progress","optionId":"ip","field":{"__typename":"ProjectV2SingleSelectField","id":"PVTSSF_status","name":"Status"}}]},"content":{"__typename":"Issue","id":"ISSUE_2","number":2,"title":"Second","body":"","url":"https://github.com/octo-org/repo/issues/2","state":"OPEN","createdAt":"2025-01-03T00:00:00Z","updatedAt":"2025-01-04T00:00:00Z","repository":{"name":"repo","owner":{"login":"octo-org"}},"labels":{"nodes":[]},"blockedBy":{"nodes":[]}}},
{"id":"ITEM_draft","fieldValues":{"nodes":[{"__typename":"ProjectV2ItemFieldSingleSelectValue","name":"Todo","optionId":"todo","field":{"__typename":"ProjectV2SingleSelectField","id":"PVTSSF_status","name":"Status"}}]},"content":{"__typename":"DraftIssue","id":"DI_1"}}
],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`))
			default:
				t.Fatalf("unexpected after cursor %q", after)
			}
		default:
			t.Fatalf("unexpected query: %s", req.Query)
		}
	}))
	t.Cleanup(srv.Close)

	client, err := NewFromConfig(config.TrackerConfig{
		ActiveStates: []string{"Todo", "In Progress"},
		PageSize:     2,
		Timeout:      2 * time.Second,
		Params: map[string]any{
			"endpoint":       srv.URL,
			"token":          "x",
			"project_owner":  "octo-org",
			"project_number": 7,
			"repository":     "octo-org/repo",
		},
	})
	if err != nil {
		t.Fatalf("NewFromConfig error: %v", err)
	}

	issues, err := client.FetchCandidateIssues(context.Background())
	if err != nil {
		t.Fatalf("FetchCandidateIssues error: %v", err)
	}
	if projectItemCalls != 2 {
		t.Fatalf("expected 2 paged project item calls, got %d", projectItemCalls)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 matching issues, got %#v", issues)
	}
	if issues[0].Identifier != "octo-org/repo#1" || issues[1].Identifier != "octo-org/repo#2" {
		t.Fatalf("unexpected identifiers: %#v", issues)
	}
	if issues[0].State != "Todo" || issues[1].State != "In Progress" {
		t.Fatalf("unexpected states: %#v", issues)
	}
	if len(issues[0].Labels) != 1 || issues[0].Labels[0] != "bug" {
		t.Fatalf("expected lowercase label normalization, got %#v", issues[0].Labels)
	}
	if len(issues[0].BlockedBy) != 2 {
		t.Fatalf("expected two blockers, got %#v", issues[0].BlockedBy)
	}
	if issues[0].BlockedBy[0].State == nil || *issues[0].BlockedBy[0].State != "Done" {
		t.Fatalf("expected blocker state to resolve from project status, got %#v", issues[0].BlockedBy)
	}
	if issues[0].BlockedBy[1].State != nil {
		t.Fatalf("expected unresolved blocker state to remain nil, got %#v", issues[0].BlockedBy[1])
	}
	if client.itemIDs["ISSUE_1"] != "ITEM_1" || client.itemIDs["ISSUE_2"] != "ITEM_2" {
		t.Fatalf("expected project item cache to be populated, got %#v", client.itemIDs)
	}
}

func TestFetchIssueStatesByIDs_UsesCachedItemIDsAndIssueFallback(t *testing.T) {
	var itemStateCalls, issueStateCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(req.Query, "ResolveProject"):
			_, _ = w.Write([]byte(`{"data":{"organization":{"login":"octo-org","projectV2":{"id":"PVT_proj","title":"Tracker","field":{"__typename":"ProjectV2SingleSelectField","id":"PVTSSF_status","name":"Status","options":[{"id":"todo","name":"Todo"},{"id":"ip","name":"In Progress"},{"id":"review","name":"Human Review"}]}}},"user":null,"repository":{"id":"R_repo","name":"repo","owner":{"login":"octo-org"}}}}`))
		case strings.Contains(req.Query, "ProjectItemStatesByItemIDs"):
			itemStateCalls++
			_, _ = w.Write([]byte(`{"data":{"nodes":[{"__typename":"ProjectV2Item","id":"ITEM_1","fieldValues":{"nodes":[{"__typename":"ProjectV2ItemFieldSingleSelectValue","name":"In Progress","optionId":"ip","field":{"__typename":"ProjectV2SingleSelectField","id":"PVTSSF_status","name":"Status"}}]},"content":{"__typename":"Issue","id":"ISSUE_1","number":1,"repository":{"name":"repo","owner":{"login":"octo-org"}}}}]}}`))
		case strings.Contains(req.Query, "IssueStatesByIssueIDs"):
			issueStateCalls++
			_, _ = w.Write([]byte(`{"data":{"nodes":[{"__typename":"Issue","id":"ISSUE_2","number":2,"repository":{"name":"repo","owner":{"login":"octo-org"}},"projectItems":{"nodes":[{"id":"ITEM_2","project":{"id":"PVT_proj"},"fieldValues":{"nodes":[{"__typename":"ProjectV2ItemFieldSingleSelectValue","name":"Human Review","optionId":"review","field":{"__typename":"ProjectV2SingleSelectField","id":"PVTSSF_status","name":"Status"}}]}}]}}]}}`))
		default:
			t.Fatalf("unexpected query: %s", req.Query)
		}
	}))
	t.Cleanup(srv.Close)

	client, err := NewFromConfig(config.TrackerConfig{
		Timeout: 2 * time.Second,
		Params: map[string]any{
			"endpoint":       srv.URL,
			"token":          "x",
			"project_owner":  "octo-org",
			"project_number": 7,
			"repository":     "octo-org/repo",
		},
	})
	if err != nil {
		t.Fatalf("NewFromConfig error: %v", err)
	}
	client.cacheItem("ISSUE_1", "ITEM_1")

	issues, err := client.FetchIssueStatesByIDs(context.Background(), []string{"ISSUE_1", "ISSUE_2"})
	if err != nil {
		t.Fatalf("FetchIssueStatesByIDs error: %v", err)
	}
	if itemStateCalls != 1 || issueStateCalls != 1 {
		t.Fatalf("expected cached item query and fallback issue query, got item=%d issue=%d", itemStateCalls, issueStateCalls)
	}
	if len(issues) != 2 {
		t.Fatalf("expected two refreshed issues, got %#v", issues)
	}
	if issues[0].ID != "ISSUE_1" || issues[0].State != "In Progress" {
		t.Fatalf("unexpected cached refresh result: %#v", issues[0])
	}
	if issues[1].ID != "ISSUE_2" || issues[1].State != "Human Review" {
		t.Fatalf("unexpected fallback refresh result: %#v", issues[1])
	}
	if client.itemIDs["ISSUE_2"] != "ITEM_2" {
		t.Fatalf("expected fallback refresh to cache project item id, got %#v", client.itemIDs)
	}
}
