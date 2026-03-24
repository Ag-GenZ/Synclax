package linear

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"slices"
	"strings"

	"github.com/wibus-wee/synclax/pkg/symphony/domain"
	"github.com/wibus-wee/synclax/pkg/symphony/tracker"
)

type workflowBootstrapState struct {
	Name  string
	Type  string
	Color string
}

var synclaxRequiredWorkflowStates = []workflowBootstrapState{
	{Name: "Backlog", Type: "backlog", Color: "#94a3b8"},
	{Name: "Todo", Type: "unstarted", Color: "#64748b"},
	{Name: "In Progress", Type: "started", Color: "#2563eb"},
	{Name: "Human Review", Type: "started", Color: "#7c3aed"},
	{Name: "Merging", Type: "started", Color: "#ea580c"},
	{Name: "Rework", Type: "started", Color: "#dc2626"},
	{Name: "Done", Type: "completed", Color: "#16a34a"},
}

var errWorkflowStateArchiveSkipped = errors.New("workflow_state_archive_skipped")

type workflowStateNode struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Position float64 `json:"position"`
}

type projectTeamStatesEnvelope struct {
	Projects struct {
		Nodes []struct {
			Teams struct {
				Nodes []struct {
					ID     string `json:"id"`
					States struct {
						Nodes []workflowStateNode `json:"nodes"`
					} `json:"states"`
				} `json:"nodes"`
			} `json:"teams"`
		} `json:"nodes"`
	} `json:"projects"`
}

type projectTeamStatesResult struct {
	TeamID string
	States []workflowStateNode
}

type workflowStateCreateEnvelope struct {
	WorkflowStateCreate struct {
		Success       bool              `json:"success"`
		WorkflowState workflowStateNode `json:"workflowState"`
	} `json:"workflowStateCreate"`
}

type workflowStateArchiveEnvelope struct {
	WorkflowStateArchive struct {
		Success bool `json:"success"`
		Entity  struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"entity"`
	} `json:"workflowStateArchive"`
}

type issueUpdateEnvelope struct {
	IssueUpdate struct {
		Success bool `json:"success"`
		Issue   struct {
			ID         string `json:"id"`
			Identifier string `json:"identifier"`
			State      struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"state"`
		} `json:"issue"`
	} `json:"issueUpdate"`
}

func BoolParam(params map[string]any, key string, fallback bool) bool {
	if params == nil {
		return fallback
	}
	v, ok := params[key]
	if !ok {
		return fallback
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "true", "1", "yes", "y", "on":
			return true
		case "false", "0", "no", "n", "off":
			return false
		}
	}
	return fallback
}

func (c *Client) EnsureSynclaxWorkflow(ctx context.Context) error {
	meta, err := c.fetchProjectTeamStates(ctx)
	if err != nil {
		return err
	}

	existing := make(map[string]workflowStateNode, len(meta.States))
	for _, state := range meta.States {
		name := strings.ToLower(strings.TrimSpace(state.Name))
		if name == "" {
			continue
		}
		existing[name] = state
	}

	created := make([]string, 0, len(synclaxRequiredWorkflowStates))
	requiredNames := make(map[string]struct{}, len(synclaxRequiredWorkflowStates))
	requiredStateIDs := make(map[string]string, len(synclaxRequiredWorkflowStates))
	for _, required := range synclaxRequiredWorkflowStates {
		key := strings.ToLower(strings.TrimSpace(required.Name))
		requiredNames[key] = struct{}{}
		if _, ok := existing[key]; ok {
			requiredStateIDs[key] = existing[key].ID
			continue
		}
		state, err := c.createWorkflowState(ctx, meta.TeamID, required)
		if err != nil {
			return err
		}
		existing[key] = *state
		requiredStateIDs[key] = state.ID
		created = append(created, required.Name)
	}

	archived := make([]string, 0, len(meta.States))
	skipped := make([]string, 0, len(meta.States))
	for _, state := range meta.States {
		key := strings.ToLower(strings.TrimSpace(state.Name))
		if key == "" {
			continue
		}
		if _, ok := requiredNames[key]; ok {
			continue
		}
		if err := c.rehomeWorkflowStateIssues(ctx, state, requiredStateIDs); err != nil {
			return err
		}
		if err := c.archiveWorkflowState(ctx, state.ID); err != nil {
			if errors.Is(err, errWorkflowStateArchiveSkipped) {
				skipped = append(skipped, state.Name)
				continue
			}
			return err
		}
		archived = append(archived, state.Name)
	}

	if len(created) > 0 || len(archived) > 0 || len(skipped) > 0 {
		slices.Sort(created)
		slices.Sort(archived)
		slices.Sort(skipped)
		log.Printf(
			"symphony linear_bootstrap status=updated project_slug=%s team_id=%s created=%s archived=%s skipped=%s",
			c.projectSlug,
			meta.TeamID,
			strings.Join(created, ","),
			strings.Join(archived, ","),
			strings.Join(skipped, ","),
		)
		return nil
	}

	log.Printf(
		"symphony linear_bootstrap status=ok project_slug=%s team_id=%s created= archived= skipped=",
		c.projectSlug,
		meta.TeamID,
	)
	return nil
}

func (c *Client) fetchProjectTeamStates(ctx context.Context) (*projectTeamStatesResult, error) {
	raw, err := c.doGraphQL(ctx, projectTeamStatesQuery, map[string]any{
		"projectSlug": c.projectSlug,
	})
	if err != nil {
		return nil, err
	}
	return decodeProjectTeamStates(raw)
}

func (c *Client) createWorkflowState(ctx context.Context, teamID string, state workflowBootstrapState) (*workflowStateNode, error) {
	raw, err := c.doGraphQL(ctx, workflowStateCreateMutation, map[string]any{
		"input": map[string]any{
			"teamId": teamID,
			"name":   state.Name,
			"type":   state.Type,
			"color":  state.Color,
		},
	})
	if err != nil {
		return nil, err
	}
	return decodeWorkflowStateCreate(raw)
}

func (c *Client) archiveWorkflowState(ctx context.Context, stateID string) error {
	raw, err := c.doGraphQL(ctx, workflowStateArchiveMutation, map[string]any{
		"id": stateID,
	})
	if err != nil {
		if isLastStateArchiveConstraint(err) {
			return errWorkflowStateArchiveSkipped
		}
		return err
	}
	return decodeWorkflowStateArchive(raw)
}

func isLastStateArchiveConstraint(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "last state for this state type") || strings.Contains(msg, "\"message\":\"last state\"")
}

func (c *Client) rehomeWorkflowStateIssues(ctx context.Context, state workflowStateNode, requiredStateIDs map[string]string) error {
	targetStateID, ok := targetStateIDForExtraState(state, requiredStateIDs)
	if !ok || strings.TrimSpace(targetStateID) == "" {
		return nil
	}

	issues, err := c.fetchIssuesByStateIDs(ctx, []string{state.ID})
	if err != nil {
		return err
	}
	for _, issue := range issues {
		if strings.TrimSpace(issue.State) == strings.TrimSpace(state.Name) {
			if err := c.moveIssueToState(ctx, issue.ID, targetStateID); err != nil {
				return err
			}
		}
	}
	return nil
}

func targetStateIDForExtraState(state workflowStateNode, requiredStateIDs map[string]string) (string, bool) {
	name := strings.ToLower(strings.TrimSpace(state.Name))
	switch {
	case strings.Contains(name, "review"):
		return requiredStateIDs["human review"], true
	case strings.Contains(name, "merge"):
		return requiredStateIDs["merging"], true
	}

	switch strings.ToLower(strings.TrimSpace(state.Type)) {
	case "backlog":
		return requiredStateIDs["backlog"], true
	case "unstarted":
		return requiredStateIDs["todo"], true
	case "started":
		return requiredStateIDs["in progress"], true
	case "completed", "canceled", "cancelled":
		return requiredStateIDs["done"], true
	default:
		return "", false
	}
}

func (c *Client) fetchIssuesByStateIDs(ctx context.Context, stateIDs []string) ([]domain.Issue, error) {
	vars := map[string]any{
		"stateIDs": stateIDs,
		"first":    c.pageSize,
		"after":    nil,
	}
	return c.fetchIssuesPaginated(ctx, issuesByStateIDsQuery, vars, func(raw json.RawMessage) ([]domain.Issue, *pageInfo, error) {
		return decodeIssuesByStateIDs(raw)
	})
}

func (c *Client) moveIssueToState(ctx context.Context, issueID, stateID string) error {
	raw, err := c.doGraphQL(ctx, issueUpdateStateMutation, map[string]any{
		"id": issueID,
		"input": map[string]any{
			"stateId": stateID,
		},
	})
	if err != nil {
		return err
	}
	return decodeIssueStateUpdate(raw)
}

func decodeProjectTeamStates(raw json.RawMessage) (*projectTeamStatesResult, error) {
	var env projectTeamStatesEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, &tracker.Error{Category: "linear_unknown_payload", Err: err}
	}
	if len(env.Projects.Nodes) == 0 {
		return nil, &tracker.Error{Category: "linear_unknown_payload", Err: errors.New("project not found")}
	}
	p := env.Projects.Nodes[0]
	if len(p.Teams.Nodes) == 0 {
		return nil, &tracker.Error{Category: "linear_unknown_payload", Err: errors.New("project teams missing")}
	}
	team := p.Teams.Nodes[0]
	if strings.TrimSpace(team.ID) == "" {
		return nil, &tracker.Error{Category: "linear_unknown_payload", Err: errors.New("project team missing")}
	}
	return &projectTeamStatesResult{
		TeamID: team.ID,
		States: append([]workflowStateNode(nil), team.States.Nodes...),
	}, nil
}

func decodeWorkflowStateCreate(raw json.RawMessage) (*workflowStateNode, error) {
	var env workflowStateCreateEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, &tracker.Error{Category: "linear_unknown_payload", Err: err}
	}
	if !env.WorkflowStateCreate.Success {
		return nil, &tracker.Error{Category: "linear_unknown_payload", Err: errors.New("workflowStateCreate was not successful")}
	}
	if strings.TrimSpace(env.WorkflowStateCreate.WorkflowState.ID) == "" {
		return nil, &tracker.Error{Category: "linear_unknown_payload", Err: errors.New("workflowStateCreate missing workflow state")}
	}
	return &env.WorkflowStateCreate.WorkflowState, nil
}

func decodeWorkflowStateArchive(raw json.RawMessage) error {
	var env workflowStateArchiveEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return &tracker.Error{Category: "linear_unknown_payload", Err: err}
	}
	if !env.WorkflowStateArchive.Success {
		return &tracker.Error{Category: "linear_unknown_payload", Err: errors.New("workflowStateArchive was not successful")}
	}
	if strings.TrimSpace(env.WorkflowStateArchive.Entity.ID) == "" {
		return &tracker.Error{Category: "linear_unknown_payload", Err: errors.New("workflowStateArchive missing entity")}
	}
	return nil
}

func decodeIssuesByStateIDs(raw json.RawMessage) ([]domain.Issue, *pageInfo, error) {
	var payload struct {
		Issues struct {
			Nodes []struct {
				ID         string `json:"id"`
				Identifier string `json:"identifier"`
				State      struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"state"`
			} `json:"nodes"`
			PageInfo pageInfo `json:"pageInfo"`
		} `json:"issues"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, nil, &tracker.Error{Category: "linear_unknown_payload", Err: err}
	}
	out := make([]domain.Issue, 0, len(payload.Issues.Nodes))
	for _, node := range payload.Issues.Nodes {
		out = append(out, domain.Issue{
			ID:         strings.TrimSpace(node.ID),
			Identifier: strings.TrimSpace(node.Identifier),
			State:      strings.TrimSpace(node.State.Name),
		})
	}
	pi := payload.Issues.PageInfo
	return out, &pi, nil
}

func decodeIssueStateUpdate(raw json.RawMessage) error {
	var env issueUpdateEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return &tracker.Error{Category: "linear_unknown_payload", Err: err}
	}
	if !env.IssueUpdate.Success {
		return &tracker.Error{Category: "linear_unknown_payload", Err: errors.New("issueUpdate was not successful")}
	}
	if strings.TrimSpace(env.IssueUpdate.Issue.ID) == "" {
		return &tracker.Error{Category: "linear_unknown_payload", Err: errors.New("issueUpdate missing issue")}
	}
	return nil
}
