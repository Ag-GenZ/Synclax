package template

import (
	"errors"
	"strings"

	"github.com/osteele/liquid"
	"github.com/wibus-wee/synclax/pkg/symphony/domain"
)

const defaultPrompt = "You are working on an issue from Linear."

var (
	ErrTemplateParseError  = errors.New("template_parse_error")
	ErrTemplateRenderError = errors.New("template_render_error")
)

type Renderer struct {
	engine  *liquid.Engine
	source  string
	parsed  *liquid.Template
	enabled bool
}

func Compile(promptTemplate string) (*Renderer, error) {
	engine := liquid.NewEngine()
	engine.StrictVariables()

	promptTemplate = strings.TrimSpace(promptTemplate)
	if promptTemplate == "" {
		return &Renderer{
			engine:  engine,
			source:  "",
			parsed:  nil,
			enabled: false,
		}, nil
	}

	tpl, err := engine.ParseString(promptTemplate)
	if err != nil {
		return nil, ErrTemplateParseError
	}
	return &Renderer{
		engine:  engine,
		source:  promptTemplate,
		parsed:  tpl,
		enabled: true,
	}, nil
}

func (r *Renderer) RenderIssuePrompt(issue domain.Issue, attempt *int) (string, error) {
	if r == nil || !r.enabled || r.parsed == nil {
		return defaultPrompt, nil
	}

	bindings := liquid.Bindings{
		"issue":   issueToMap(issue),
		"attempt": attemptToAny(attempt),
	}
	outBytes, err := r.parsed.Render(bindings)
	if err != nil {
		return "", ErrTemplateRenderError
	}
	out := strings.TrimSpace(string(outBytes))
	if out == "" {
		return defaultPrompt, nil
	}
	return out, nil
}

func attemptToAny(attempt *int) any {
	if attempt == nil {
		return nil
	}
	return *attempt
}

func issueToMap(issue domain.Issue) map[string]any {
	m := map[string]any{
		"id":         issue.ID,
		"identifier": issue.Identifier,
		"title":      issue.Title,
		"state":      issue.State,
		"labels":     issue.Labels,
	}
	if issue.Description != nil {
		m["description"] = *issue.Description
	} else {
		m["description"] = nil
	}
	if issue.Priority != nil {
		m["priority"] = *issue.Priority
	} else {
		m["priority"] = nil
	}
	if issue.BranchName != nil {
		m["branch_name"] = *issue.BranchName
	} else {
		m["branch_name"] = nil
	}
	if issue.URL != nil {
		m["url"] = *issue.URL
	} else {
		m["url"] = nil
	}

	blockers := make([]map[string]any, 0, len(issue.BlockedBy))
	for _, b := range issue.BlockedBy {
		entry := map[string]any{
			"id":         nil,
			"identifier": nil,
			"state":      nil,
		}
		if b.ID != nil {
			entry["id"] = *b.ID
		}
		if b.Identifier != nil {
			entry["identifier"] = *b.Identifier
		}
		if b.State != nil {
			entry["state"] = *b.State
		}
		blockers = append(blockers, entry)
	}
	m["blocked_by"] = blockers
	if issue.CreatedAt != nil {
		m["created_at"] = issue.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	} else {
		m["created_at"] = nil
	}
	if issue.UpdatedAt != nil {
		m["updated_at"] = issue.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	} else {
		m["updated_at"] = nil
	}
	return m
}
