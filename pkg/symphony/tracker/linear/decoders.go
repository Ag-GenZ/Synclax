package linear

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/wibus-wee/synclax/pkg/symphony/domain"
	"github.com/wibus-wee/synclax/pkg/symphony/tracker"
)

type candidateIssuesEnvelope struct {
	Issues struct {
		Nodes    []candidateIssueNode `json:"nodes"`
		PageInfo pageInfo             `json:"pageInfo"`
	} `json:"issues"`
}

type candidateIssueNode struct {
	ID          string   `json:"id"`
	Identifier  string   `json:"identifier"`
	Title       string   `json:"title"`
	Description *string  `json:"description"`
	Priority    any      `json:"priority"`
	URL         *string  `json:"url"`
	BranchName  *string  `json:"branchName"`
	State       stateRef `json:"state"`
	Labels      struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	InverseRelations struct {
		Nodes []struct {
			Type  string              `json:"type"`
			Issue *candidateIssueNode `json:"issue"`
		} `json:"nodes"`
	} `json:"inverseRelations"`
	CreatedAt *string `json:"createdAt"`
	UpdatedAt *string `json:"updatedAt"`
}

type stateRef struct {
	Name string `json:"name"`
}

func decodeCandidateIssues(raw json.RawMessage) ([]domain.Issue, *pageInfo, error) {
	var env candidateIssuesEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, nil, &tracker.Error{Category: "linear_unknown_payload", Err: err}
	}
	out := make([]domain.Issue, 0, len(env.Issues.Nodes))
	for _, n := range env.Issues.Nodes {
		issue, err := normalizeCandidateIssue(n)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, issue)
	}
	return out, &env.Issues.PageInfo, nil
}

type issuesByStatesEnvelope struct {
	Issues struct {
		Nodes []struct {
			ID         string   `json:"id"`
			Identifier string   `json:"identifier"`
			Title      string   `json:"title"`
			State      stateRef `json:"state"`
		} `json:"nodes"`
		PageInfo pageInfo `json:"pageInfo"`
	} `json:"issues"`
}

func decodeIssuesByStates(raw json.RawMessage) ([]domain.Issue, *pageInfo, error) {
	var env issuesByStatesEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, nil, &tracker.Error{Category: "linear_unknown_payload", Err: err}
	}
	out := make([]domain.Issue, 0, len(env.Issues.Nodes))
	for _, n := range env.Issues.Nodes {
		out = append(out, domain.Issue{
			ID:         n.ID,
			Identifier: n.Identifier,
			Title:      n.Title,
			State:      n.State.Name,
		})
	}
	return out, &env.Issues.PageInfo, nil
}

type issueStatesByIDsEnvelope struct {
	Issues struct {
		Nodes []struct {
			ID         string   `json:"id"`
			Identifier string   `json:"identifier"`
			State      stateRef `json:"state"`
		} `json:"nodes"`
	} `json:"issues"`
}

func decodeIssueStatesByIDs(raw json.RawMessage) ([]domain.Issue, error) {
	var env issueStatesByIDsEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, &tracker.Error{Category: "linear_unknown_payload", Err: err}
	}
	out := make([]domain.Issue, 0, len(env.Issues.Nodes))
	for _, n := range env.Issues.Nodes {
		out = append(out, domain.Issue{
			ID:         n.ID,
			Identifier: n.Identifier,
			State:      n.State.Name,
		})
	}
	return out, nil
}

func normalizeCandidateIssue(n candidateIssueNode) (domain.Issue, error) {
	priority := parsePriority(n.Priority)
	labels := make([]string, 0, len(n.Labels.Nodes))
	for _, l := range n.Labels.Nodes {
		name := strings.ToLower(strings.TrimSpace(l.Name))
		if name == "" {
			continue
		}
		labels = append(labels, name)
	}

	blockedBy := make([]domain.BlockerRef, 0)
	for _, rel := range n.InverseRelations.Nodes {
		if strings.ToLower(rel.Type) != "blocks" {
			continue
		}
		if rel.Issue == nil {
			blockedBy = append(blockedBy, domain.BlockerRef{})
			continue
		}
		id := strings.TrimSpace(rel.Issue.ID)
		ident := strings.TrimSpace(rel.Issue.Identifier)
		state := strings.TrimSpace(rel.Issue.State.Name)
		var (
			idp    *string
			identp *string
			statep *string
		)
		if id != "" {
			idp = &id
		}
		if ident != "" {
			identp = &ident
		}
		if state != "" {
			statep = &state
		}
		blockedBy = append(blockedBy, domain.BlockerRef{ID: idp, Identifier: identp, State: statep})
	}

	createdAt, err := parseTime(n.CreatedAt)
	if err != nil {
		return domain.Issue{}, &tracker.Error{Category: "linear_unknown_payload", Err: err}
	}
	updatedAt, err := parseTime(n.UpdatedAt)
	if err != nil {
		return domain.Issue{}, &tracker.Error{Category: "linear_unknown_payload", Err: err}
	}

	if n.ID == "" || n.Identifier == "" || n.Title == "" || n.State.Name == "" {
		return domain.Issue{}, &tracker.Error{Category: "linear_unknown_payload", Err: errors.New("missing required issue fields")}
	}

	return domain.Issue{
		ID:          n.ID,
		Identifier:  n.Identifier,
		Title:       n.Title,
		Description: n.Description,
		Priority:    priority,
		State:       n.State.Name,
		BranchName:  n.BranchName,
		URL:         n.URL,
		Labels:      labels,
		BlockedBy:   blockedBy,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

func parsePriority(v any) *int {
	switch t := v.(type) {
	case float64:
		n := int(t)
		return &n
	case int:
		return &t
	case int64:
		n := int(t)
		return &n
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		if err != nil {
			return nil
		}
		return &n
	default:
		return nil
	}
}

func parseTime(s *string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(*s))
	if err != nil {
		return nil, err
	}
	return &t, nil
}
