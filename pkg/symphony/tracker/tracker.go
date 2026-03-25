package tracker

import (
	"context"

	"github.com/wibus-wee/synclax/pkg/symphony/domain"
)

// Supported reports whether kind names a built-in tracker implementation.
func Supported(kind string) bool {
	switch kind {
	case "github", "linear":
		return true
	default:
		return false
	}
}

type Client interface {
	FetchCandidateIssues(ctx context.Context) ([]domain.Issue, error)
	FetchIssuesByStates(ctx context.Context, stateNames []string) ([]domain.Issue, error)
	FetchIssueStatesByIDs(ctx context.Context, issueIDs []string) ([]domain.Issue, error)
}

type Error struct {
	Category   string
	StatusCode int
	Err        error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return e.Category
	}
	return e.Category + ": " + e.Err.Error()
}

func (e *Error) Unwrap() error { return e.Err }
