package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/wibus-wee/synclax/pkg/symphony/config"
	"github.com/wibus-wee/synclax/pkg/symphony/domain"
	"github.com/wibus-wee/synclax/pkg/symphony/tracker"
)

type Client struct {
	endpoint      string
	token         string
	projectOwner  string
	projectNumber int
	repoOwner     string
	repoName      string
	stateField    string
	activeStates  []string

	pageSize int
	timeout  time.Duration

	projectID    string
	stateFieldID string

	httpClient *http.Client

	mu        sync.RWMutex
	itemIDs   map[string]string
	optionIDs map[string]string
}

type Options struct {
	Endpoint      string
	Token         string
	ProjectOwner  string
	ProjectNumber int
	Repository    string
	StateField    string
	ActiveStates  []string
	PageSize      int
	Timeout       time.Duration
	HTTPClient    *http.Client
}

func New(opts Options) (*Client, error) {
	if strings.TrimSpace(opts.Endpoint) == "" {
		return nil, errors.New("github graphql endpoint is required")
	}
	if strings.TrimSpace(opts.Token) == "" {
		return nil, errors.New("github token is required")
	}
	if strings.TrimSpace(opts.ProjectOwner) == "" {
		return nil, errors.New("github project owner is required")
	}
	if opts.ProjectNumber <= 0 {
		return nil, errors.New("github project number is required")
	}
	repoOwner, repoName, err := parseRepository(opts.ProjectOwner, opts.Repository)
	if err != nil {
		return nil, err
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
	stateField := strings.TrimSpace(opts.StateField)
	if stateField == "" {
		stateField = "Status"
	}
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: opts.Timeout}
	}

	c := &Client{
		endpoint:      strings.TrimSpace(opts.Endpoint),
		token:         strings.TrimSpace(opts.Token),
		projectOwner:  strings.TrimSpace(opts.ProjectOwner),
		projectNumber: opts.ProjectNumber,
		repoOwner:     repoOwner,
		repoName:      repoName,
		stateField:    stateField,
		activeStates:  append([]string(nil), opts.ActiveStates...),
		pageSize:      opts.PageSize,
		timeout:       opts.Timeout,
		httpClient:    hc,
		itemIDs:       map[string]string{},
		optionIDs:     map[string]string{},
	}
	if err := c.resolveProject(ctxBackground()); err != nil {
		return nil, err
	}
	return c, nil
}

func NewFromConfig(cfg config.TrackerConfig) (*Client, error) {
	endpoint := tracker.StringParam(cfg.Params, "endpoint", "https://api.github.com/graphql")
	token := tracker.StringParam(cfg.Params, "token", "")
	if token == "" {
		token = strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	}
	return New(Options{
		Endpoint:      endpoint,
		Token:         token,
		ProjectOwner:  tracker.StringParam(cfg.Params, "project_owner", ""),
		ProjectNumber: tracker.IntParam(cfg.Params, "project_number", 0),
		Repository:    tracker.StringParam(cfg.Params, "repository", ""),
		StateField:    tracker.StringParam(cfg.Params, "state_field", "Status"),
		ActiveStates:  cfg.ActiveStates,
		PageSize:      cfg.PageSize,
		Timeout:       cfg.Timeout,
	})
}

func (c *Client) FetchCandidateIssues(ctx context.Context) ([]domain.Issue, error) {
	return c.fetchProjectIssues(ctx, c.activeStates)
}

func (c *Client) FetchIssuesByStates(ctx context.Context, stateNames []string) ([]domain.Issue, error) {
	if len(stateNames) == 0 {
		return []domain.Issue{}, nil
	}
	return c.fetchProjectIssues(ctx, stateNames)
}

func (c *Client) FetchIssueStatesByIDs(ctx context.Context, issueIDs []string) ([]domain.Issue, error) {
	if len(issueIDs) == 0 {
		return []domain.Issue{}, nil
	}

	issuesByID := map[string]domain.Issue{}
	itemIDs, uncached := c.cachedItemIDs(issueIDs)
	if len(itemIDs) > 0 {
		cached, err := c.fetchIssueStatesByItemIDs(ctx, itemIDs)
		if err != nil {
			return nil, err
		}
		for _, issue := range cached {
			issuesByID[issue.ID] = issue
		}
	}

	missing := make([]string, 0, len(uncached))
	for _, issueID := range uncached {
		if _, ok := issuesByID[issueID]; !ok {
			missing = append(missing, issueID)
		}
	}
	if len(missing) > 0 {
		loaded, err := c.fetchIssueStatesByIssueIDs(ctx, missing)
		if err != nil {
			return nil, err
		}
		for _, issue := range loaded {
			issuesByID[issue.ID] = issue
		}
	}

	out := make([]domain.Issue, 0, len(issueIDs))
	for _, issueID := range issueIDs {
		issue, ok := issuesByID[issueID]
		if !ok {
			continue
		}
		if strings.TrimSpace(issue.State) == "" {
			continue
		}
		out = append(out, issue)
	}
	return out, nil
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

	payload, err := json.Marshal(gqlRequest{Query: query, Variables: variables})
	if err != nil {
		return nil, &tracker.Error{Category: "github_unknown_payload", Err: err}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, &tracker.Error{Category: "github_api_request", Err: err}
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &tracker.Error{Category: "github_api_request", Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4*1024))
		return nil, &tracker.Error{
			Category:   "github_api_status",
			StatusCode: resp.StatusCode,
			Err:        errors.New(strings.TrimSpace(string(body))),
		}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, &tracker.Error{Category: "github_api_request", Err: err}
	}

	var decoded gqlResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, &tracker.Error{Category: "github_unknown_payload", Err: err}
	}
	if len(decoded.Errors) > 0 {
		errBody, _ := json.Marshal(decoded.Errors)
		return nil, &tracker.Error{Category: "github_graphql_errors", Err: errors.New(string(errBody))}
	}
	if len(decoded.Data) == 0 {
		return nil, &tracker.Error{Category: "github_unknown_payload", Err: errors.New("missing data")}
	}
	return decoded.Data, nil
}

type resolveProjectEnvelope struct {
	Organization *projectOwnerNode `json:"organization"`
	User         *projectOwnerNode `json:"user"`
	Repository   *repositoryNode   `json:"repository"`
}

type projectOwnerNode struct {
	Login   string       `json:"login"`
	Project *projectNode `json:"projectV2"`
}

type projectNode struct {
	ID    string                 `json:"id"`
	Title string                 `json:"title"`
	Field *singleSelectFieldNode `json:"field"`
}

type singleSelectFieldNode struct {
	Typename string                   `json:"__typename"`
	ID       string                   `json:"id"`
	Name     string                   `json:"name"`
	Options  []singleSelectOptionNode `json:"options"`
}

type singleSelectOptionNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type repositoryNode struct {
	ID    string       `json:"id"`
	Name  string       `json:"name"`
	Owner ownerRefNode `json:"owner"`
}

type ownerRefNode struct {
	Login string `json:"login"`
}

func (c *Client) resolveProject(ctx context.Context) error {
	raw, err := c.doGraphQL(ctx, resolveProjectQuery, map[string]any{
		"projectOwner":  c.projectOwner,
		"projectNumber": c.projectNumber,
		"repoOwner":     c.repoOwner,
		"repoName":      c.repoName,
		"fieldName":     c.stateField,
	})
	if err != nil {
		return err
	}

	var env resolveProjectEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return &tracker.Error{Category: "github_unknown_payload", Err: err}
	}
	if env.Repository == nil || strings.TrimSpace(env.Repository.ID) == "" {
		return errors.New("github repository is required and must exist")
	}

	project := pickProject(env.Organization, env.User)
	if project == nil || strings.TrimSpace(project.ID) == "" {
		return fmt.Errorf("github project v2 %s/%d was not found", c.projectOwner, c.projectNumber)
	}
	if project.Field == nil {
		return fmt.Errorf("github project field %q was not found", c.stateField)
	}
	if project.Field.Typename != "ProjectV2SingleSelectField" {
		return fmt.Errorf("github project field %q must be a single-select field", c.stateField)
	}

	optionIDs := map[string]string{}
	for _, option := range project.Field.Options {
		name := strings.TrimSpace(option.Name)
		if name == "" || strings.TrimSpace(option.ID) == "" {
			continue
		}
		optionIDs[name] = option.ID
	}

	c.projectID = strings.TrimSpace(project.ID)
	c.stateFieldID = strings.TrimSpace(project.Field.ID)
	c.mu.Lock()
	c.optionIDs = optionIDs
	c.mu.Unlock()
	return nil
}

func pickProject(nodes ...*projectOwnerNode) *projectNode {
	for _, node := range nodes {
		if node != nil && node.Project != nil {
			return node.Project
		}
	}
	return nil
}

type projectItemsEnvelope struct {
	Node *struct {
		Items struct {
			Nodes    []projectItemNode `json:"nodes"`
			PageInfo pageInfo          `json:"pageInfo"`
		} `json:"items"`
	} `json:"node"`
}

type pageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type projectItemNode struct {
	ID          string               `json:"id"`
	FieldValues fieldValueConnection `json:"fieldValues"`
	Content     projectItemContent   `json:"content"`
}

type fieldValueConnection struct {
	Nodes []fieldValueNode `json:"nodes"`
}

type fieldValueNode struct {
	Typename string `json:"__typename"`
	Name     string `json:"name"`
	OptionID string `json:"optionId"`
	Field    struct {
		Typename string `json:"__typename"`
		ID       string `json:"id"`
		Name     string `json:"name"`
	} `json:"field"`
}

type projectItemContent struct {
	Typename   string          `json:"__typename"`
	ID         string          `json:"id"`
	Number     int             `json:"number"`
	Title      string          `json:"title"`
	Body       string          `json:"body"`
	URL        string          `json:"url"`
	State      string          `json:"state"`
	CreatedAt  string          `json:"createdAt"`
	UpdatedAt  string          `json:"updatedAt"`
	Repository repositoryRef   `json:"repository"`
	Labels     labelConnection `json:"labels"`
	BlockedBy  struct {
		Nodes []blockedIssueNode `json:"nodes"`
	} `json:"blockedBy"`
}

type repositoryRef struct {
	Name  string       `json:"name"`
	Owner ownerRefNode `json:"owner"`
}

type labelConnection struct {
	Nodes []struct {
		Name string `json:"name"`
	} `json:"nodes"`
}

type blockedIssueNode struct {
	ID           string                     `json:"id"`
	Number       int                        `json:"number"`
	URL          string                     `json:"url"`
	State        string                     `json:"state"`
	Repository   repositoryRef              `json:"repository"`
	ProjectItems issueProjectItemConnection `json:"projectItems"`
}

type issueProjectItemConnection struct {
	Nodes []issueProjectItemNode `json:"nodes"`
}

type issueProjectItemNode struct {
	ID      string `json:"id"`
	Project struct {
		ID string `json:"id"`
	} `json:"project"`
	FieldValues fieldValueConnection `json:"fieldValues"`
}

func (c *Client) fetchProjectIssues(ctx context.Context, stateNames []string) ([]domain.Issue, error) {
	stateSet := normalizeStates(stateNames)
	var all []domain.Issue
	after := ""

	for {
		vars := map[string]any{
			"projectId": c.projectID,
			"first":     c.pageSize,
			"after":     nil,
		}
		if after != "" {
			vars["after"] = after
		}

		raw, err := c.doGraphQL(ctx, projectItemsQuery, vars)
		if err != nil {
			return nil, err
		}
		var env projectItemsEnvelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return nil, &tracker.Error{Category: "github_unknown_payload", Err: err}
		}
		if env.Node == nil {
			return nil, &tracker.Error{Category: "github_unknown_payload", Err: errors.New("missing project node")}
		}

		for _, item := range env.Node.Items.Nodes {
			issue, ok := c.normalizeProjectItem(item, stateSet)
			if !ok {
				continue
			}
			all = append(all, issue)
		}

		if !env.Node.Items.PageInfo.HasNextPage {
			break
		}
		if strings.TrimSpace(env.Node.Items.PageInfo.EndCursor) == "" {
			return nil, &tracker.Error{Category: "github_missing_end_cursor", Err: errors.New("missing endCursor")}
		}
		after = env.Node.Items.PageInfo.EndCursor
	}
	return all, nil
}

func normalizeStates(states []string) map[string]struct{} {
	out := make(map[string]struct{}, len(states))
	for _, state := range states {
		s := strings.ToLower(strings.TrimSpace(state))
		if s == "" {
			continue
		}
		out[s] = struct{}{}
	}
	return out
}

func (c *Client) normalizeProjectItem(item projectItemNode, allowedStates map[string]struct{}) (domain.Issue, bool) {
	if item.Content.Typename != "Issue" {
		return domain.Issue{}, false
	}
	if !c.matchesRepository(item.Content.Repository) {
		return domain.Issue{}, false
	}

	state, optionID := c.extractState(item.FieldValues)
	if state == "" {
		return domain.Issue{}, false
	}
	if len(allowedStates) > 0 {
		if _, ok := allowedStates[strings.ToLower(state)]; !ok {
			return domain.Issue{}, false
		}
	}

	c.cacheItem(item.Content.ID, item.ID)
	c.cacheOption(state, optionID)

	labels := make([]string, 0, len(item.Content.Labels.Nodes))
	for _, label := range item.Content.Labels.Nodes {
		name := strings.ToLower(strings.TrimSpace(label.Name))
		if name == "" {
			continue
		}
		labels = append(labels, name)
	}

	blockedBy := make([]domain.BlockerRef, 0, len(item.Content.BlockedBy.Nodes))
	for _, blocker := range item.Content.BlockedBy.Nodes {
		if !c.matchesRepository(blocker.Repository) {
			continue
		}
		blockedBy = append(blockedBy, c.normalizeBlocker(blocker))
	}

	createdAt, err := parseGitHubTime(item.Content.CreatedAt)
	if err != nil {
		return domain.Issue{}, false
	}
	updatedAt, err := parseGitHubTime(item.Content.UpdatedAt)
	if err != nil {
		return domain.Issue{}, false
	}

	title := strings.TrimSpace(item.Content.Title)
	if title == "" {
		return domain.Issue{}, false
	}
	description := normalizeOptionalString(item.Content.Body)
	url := normalizeOptionalString(item.Content.URL)

	return domain.Issue{
		ID:          item.Content.ID,
		Identifier:  c.issueIdentifier(item.Content.Repository, item.Content.Number),
		Title:       title,
		Description: description,
		State:       state,
		URL:         url,
		Labels:      labels,
		BlockedBy:   blockedBy,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, true
}

func (c *Client) normalizeBlocker(blocker blockedIssueNode) domain.BlockerRef {
	id := strings.TrimSpace(blocker.ID)
	identifier := c.issueIdentifier(blocker.Repository, blocker.Number)
	state := c.resolveStateFromProjectItems(blocker.ProjectItems)

	ref := domain.BlockerRef{}
	if id != "" {
		ref.ID = &id
	}
	if identifier != "" {
		ref.Identifier = &identifier
	}
	if state != "" {
		ref.State = &state
	}
	return ref
}

func (c *Client) resolveStateFromProjectItems(items issueProjectItemConnection) string {
	for _, item := range items.Nodes {
		if strings.TrimSpace(item.Project.ID) != c.projectID {
			continue
		}
		state, optionID := c.extractState(item.FieldValues)
		if state == "" {
			continue
		}
		c.cacheOption(state, optionID)
		return state
	}
	return ""
}

func (c *Client) extractState(values fieldValueConnection) (string, string) {
	for _, value := range values.Nodes {
		if value.Typename != "ProjectV2ItemFieldSingleSelectValue" {
			continue
		}
		if strings.TrimSpace(value.Field.ID) == c.stateFieldID || strings.EqualFold(strings.TrimSpace(value.Field.Name), c.stateField) {
			return strings.TrimSpace(value.Name), strings.TrimSpace(value.OptionID)
		}
	}
	return "", ""
}

type nodesProjectItemsEnvelope struct {
	Nodes []projectItemStateNode `json:"nodes"`
}

type projectItemStateNode struct {
	Typename    string               `json:"__typename"`
	ID          string               `json:"id"`
	Content     projectItemContent   `json:"content"`
	FieldValues fieldValueConnection `json:"fieldValues"`
}

func (c *Client) fetchIssueStatesByItemIDs(ctx context.Context, itemIDs []string) ([]domain.Issue, error) {
	raw, err := c.doGraphQL(ctx, projectItemStatesByItemIDsQuery, map[string]any{"ids": itemIDs})
	if err != nil {
		return nil, err
	}
	var env nodesProjectItemsEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, &tracker.Error{Category: "github_unknown_payload", Err: err}
	}

	out := make([]domain.Issue, 0, len(env.Nodes))
	for _, node := range env.Nodes {
		if node.Typename != "ProjectV2Item" || node.Content.Typename != "Issue" {
			continue
		}
		if !c.matchesRepository(node.Content.Repository) {
			continue
		}
		state, optionID := c.extractState(node.FieldValues)
		if state == "" {
			continue
		}
		c.cacheItem(node.Content.ID, node.ID)
		c.cacheOption(state, optionID)
		out = append(out, domain.Issue{
			ID:         node.Content.ID,
			Identifier: c.issueIdentifier(node.Content.Repository, node.Content.Number),
			State:      state,
		})
	}
	return out, nil
}

type nodesIssuesEnvelope struct {
	Nodes []issueStateNode `json:"nodes"`
}

type issueStateNode struct {
	Typename     string                     `json:"__typename"`
	ID           string                     `json:"id"`
	Number       int                        `json:"number"`
	Repository   repositoryRef              `json:"repository"`
	ProjectItems issueProjectItemConnection `json:"projectItems"`
}

func (c *Client) fetchIssueStatesByIssueIDs(ctx context.Context, issueIDs []string) ([]domain.Issue, error) {
	raw, err := c.doGraphQL(ctx, issueStatesByIssueIDsQuery, map[string]any{"ids": issueIDs})
	if err != nil {
		return nil, err
	}
	var env nodesIssuesEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, &tracker.Error{Category: "github_unknown_payload", Err: err}
	}

	out := make([]domain.Issue, 0, len(env.Nodes))
	for _, node := range env.Nodes {
		if node.Typename != "Issue" {
			continue
		}
		if !c.matchesRepository(node.Repository) {
			continue
		}
		state := ""
		itemID := ""
		for _, item := range node.ProjectItems.Nodes {
			if strings.TrimSpace(item.Project.ID) != c.projectID {
				continue
			}
			stateName, optionID := c.extractState(item.FieldValues)
			if stateName == "" {
				continue
			}
			state = stateName
			itemID = item.ID
			c.cacheOption(stateName, optionID)
			break
		}
		if state == "" {
			continue
		}
		if itemID != "" {
			c.cacheItem(node.ID, itemID)
		}
		out = append(out, domain.Issue{
			ID:         node.ID,
			Identifier: c.issueIdentifier(node.Repository, node.Number),
			State:      state,
		})
	}
	return out, nil
}

func parseRepository(projectOwner, repository string) (string, string, error) {
	repository = strings.TrimSpace(repository)
	if repository == "" {
		return "", "", errors.New("github repository is required")
	}
	if owner, name, ok := strings.Cut(repository, "/"); ok {
		owner = strings.TrimSpace(owner)
		name = strings.TrimSpace(name)
		if owner == "" || name == "" {
			return "", "", errors.New("github repository must be in owner/repo format when a slash is provided")
		}
		return owner, name, nil
	}
	return strings.TrimSpace(projectOwner), repository, nil
}

func parseGitHubTime(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func normalizeOptionalString(raw string) *string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	return &raw
}

func (c *Client) matchesRepository(repo repositoryRef) bool {
	return strings.EqualFold(strings.TrimSpace(repo.Owner.Login), c.repoOwner) &&
		strings.EqualFold(strings.TrimSpace(repo.Name), c.repoName)
}

func (c *Client) issueIdentifier(repo repositoryRef, number int) string {
	return fmt.Sprintf("%s/%s#%d", strings.TrimSpace(repo.Owner.Login), strings.TrimSpace(repo.Name), number)
}

func (c *Client) cacheItem(issueID, itemID string) {
	issueID = strings.TrimSpace(issueID)
	itemID = strings.TrimSpace(itemID)
	if issueID == "" || itemID == "" {
		return
	}
	c.mu.Lock()
	c.itemIDs[issueID] = itemID
	c.mu.Unlock()
}

func (c *Client) cacheOption(state, optionID string) {
	state = strings.TrimSpace(state)
	optionID = strings.TrimSpace(optionID)
	if state == "" || optionID == "" {
		return
	}
	c.mu.Lock()
	if _, exists := c.optionIDs[state]; !exists {
		c.optionIDs[state] = optionID
	}
	c.mu.Unlock()
}

func (c *Client) cachedItemIDs(issueIDs []string) ([]string, []string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	itemIDs := make([]string, 0, len(issueIDs))
	missing := make([]string, 0, len(issueIDs))
	seen := map[string]struct{}{}
	for _, issueID := range issueIDs {
		itemID, ok := c.itemIDs[issueID]
		if !ok || strings.TrimSpace(itemID) == "" {
			missing = append(missing, issueID)
			continue
		}
		if _, ok := seen[itemID]; ok {
			continue
		}
		seen[itemID] = struct{}{}
		itemIDs = append(itemIDs, itemID)
	}
	return itemIDs, missing
}

func ctxBackground() context.Context {
	return context.Background()
}
