package domain

import "time"

type BlockerRef struct {
	ID         *string `json:"id,omitempty"`
	Identifier *string `json:"identifier,omitempty"`
	State      *string `json:"state,omitempty"`
}

type Issue struct {
	ID          string       `json:"id"`
	Identifier  string       `json:"identifier"`
	Title       string       `json:"title"`
	Description *string      `json:"description,omitempty"`
	Priority    *int         `json:"priority,omitempty"`
	State       string       `json:"state"`
	BranchName  *string      `json:"branch_name,omitempty"`
	URL         *string      `json:"url,omitempty"`
	Labels      []string     `json:"labels,omitempty"`
	BlockedBy   []BlockerRef `json:"blocked_by,omitempty"`
	CreatedAt   *time.Time   `json:"created_at,omitempty"`
	UpdatedAt   *time.Time   `json:"updated_at,omitempty"`
}
