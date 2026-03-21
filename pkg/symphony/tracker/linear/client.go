package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/wibus-wee/synclax/pkg/symphony/config"
	"github.com/wibus-wee/synclax/pkg/symphony/domain"
	"github.com/wibus-wee/synclax/pkg/symphony/tracker"
)

type Client struct {
	endpoint     string
	apiKey       string
	projectSlug  string
	activeStates []string

	pageSize int
	timeout  time.Duration

	httpClient *http.Client
}

type Options struct {
	Endpoint     string
	APIKey       string
	ProjectSlug  string
	ActiveStates []string
	PageSize     int
	Timeout      time.Duration
	HTTPClient   *http.Client
}

func New(opts Options) (*Client, error) {
	if strings.TrimSpace(opts.Endpoint) == "" {
		return nil, errors.New("linear endpoint is required")
	}
	if strings.TrimSpace(opts.APIKey) == "" {
		return nil, errors.New("linear api key is required")
	}
	if strings.TrimSpace(opts.ProjectSlug) == "" {
		return nil, errors.New("linear project slug is required")
	}
	if opts.PageSize <= 0 {
		opts.PageSize = 50
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 30 * time.Second
	}
	if len(opts.ActiveStates) == 0 {
		opts.ActiveStates = []string{"Todo", "In Progress"}
	}
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: opts.Timeout}
	}

	return &Client{
		endpoint:     opts.Endpoint,
		apiKey:       opts.APIKey,
		projectSlug:  opts.ProjectSlug,
		activeStates: append([]string(nil), opts.ActiveStates...),
		pageSize:     opts.PageSize,
		timeout:      opts.Timeout,
		httpClient:   hc,
	}, nil
}

// NewFromConfig creates a Linear Client from a generic TrackerConfig.
// It reads Linear-specific parameters from cfg.Params:
//   - endpoint (string, default: "https://api.linear.app/graphql")
//   - api_key (string, required)
//   - project_slug (string, required)
func NewFromConfig(cfg config.TrackerConfig) (*Client, error) {
	endpoint := StringParam(cfg.Params, "endpoint", "https://api.linear.app/graphql")
	apiKey := StringParam(cfg.Params, "api_key", "")
	projectSlug := StringParam(cfg.Params, "project_slug", "")

	// Fallback: check LINEAR_API_KEY env var if api_key not in params.
	if apiKey == "" {
		apiKey = os.Getenv("LINEAR_API_KEY")
	}

	return New(Options{
		Endpoint:     endpoint,
		APIKey:       apiKey,
		ProjectSlug:  projectSlug,
		ActiveStates: cfg.ActiveStates,
		PageSize:     cfg.PageSize,
		Timeout:      cfg.Timeout,
	})
}

// StringParam extracts a string value from a params map with a fallback default.
func StringParam(params map[string]any, key, fallback string) string {
	if params == nil {
		return fallback
	}
	v, ok := params[key]
	if !ok {
		return fallback
	}
	s, ok := v.(string)
	if !ok {
		return fallback
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	return s
}

func (c *Client) FetchCandidateIssues(ctx context.Context) ([]domain.Issue, error) {
	vars := map[string]any{
		"projectSlug": c.projectSlug,
		"states":      c.activeStates,
		"first":       c.pageSize,
		"after":       nil,
	}
	return c.fetchIssuesPaginated(ctx, candidateIssuesQuery, vars, func(raw json.RawMessage) ([]domain.Issue, *pageInfo, error) {
		return decodeCandidateIssues(raw)
	})
}

func (c *Client) FetchIssuesByStates(ctx context.Context, stateNames []string) ([]domain.Issue, error) {
	if len(stateNames) == 0 {
		return []domain.Issue{}, nil
	}
	vars := map[string]any{
		"projectSlug": c.projectSlug,
		"states":      stateNames,
		"first":       c.pageSize,
		"after":       nil,
	}
	return c.fetchIssuesPaginated(ctx, issuesByStatesQuery, vars, func(raw json.RawMessage) ([]domain.Issue, *pageInfo, error) {
		return decodeIssuesByStates(raw)
	})
}

func (c *Client) FetchIssueStatesByIDs(ctx context.Context, issueIDs []string) ([]domain.Issue, error) {
	if len(issueIDs) == 0 {
		return []domain.Issue{}, nil
	}
	vars := map[string]any{
		"ids": issueIDs,
	}
	raw, err := c.doGraphQL(ctx, issueStatesByIDsQuery, vars)
	if err != nil {
		return nil, err
	}
	return decodeIssueStatesByIDs(raw)
}

type gqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type gqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []any           `json:"errors"`
}

func (c *Client) doGraphQL(ctx context.Context, query string, variables map[string]any) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	payload, err := json.Marshal(gqlRequest{
		Query:     query,
		Variables: variables,
	})
	if err != nil {
		return nil, &tracker.Error{Category: "linear_unknown_payload", Err: err}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, &tracker.Error{Category: "linear_api_request", Err: err}
	}
	req.Header.Set("Authorization", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &tracker.Error{Category: "linear_api_request", Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, &tracker.Error{Category: "linear_api_status", StatusCode: resp.StatusCode, Err: errors.New(string(body))}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &tracker.Error{Category: "linear_api_request", Err: err}
	}

	var decoded gqlResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, &tracker.Error{Category: "linear_unknown_payload", Err: err}
	}
	if len(decoded.Errors) > 0 {
		errBody, _ := json.Marshal(decoded.Errors)
		return nil, &tracker.Error{Category: "linear_graphql_errors", Err: errors.New(string(errBody))}
	}
	if len(decoded.Data) == 0 {
		return nil, &tracker.Error{Category: "linear_unknown_payload", Err: errors.New("missing data")}
	}
	return decoded.Data, nil
}

type pageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

func (c *Client) fetchIssuesPaginated(
	ctx context.Context,
	query string,
	variables map[string]any,
	decode func(raw json.RawMessage) ([]domain.Issue, *pageInfo, error),
) ([]domain.Issue, error) {
	var all []domain.Issue
	after := ""
	for {
		if after == "" {
			variables["after"] = nil
		} else {
			variables["after"] = after
		}
		raw, err := c.doGraphQL(ctx, query, variables)
		if err != nil {
			return nil, err
		}
		issues, pi, err := decode(raw)
		if err != nil {
			return nil, err
		}
		all = append(all, issues...)
		if pi == nil || !pi.HasNextPage {
			break
		}
		if strings.TrimSpace(pi.EndCursor) == "" {
			return nil, &tracker.Error{Category: "linear_missing_end_cursor", Err: errors.New("missing endCursor")}
		}
		after = pi.EndCursor
	}
	return all, nil
}
