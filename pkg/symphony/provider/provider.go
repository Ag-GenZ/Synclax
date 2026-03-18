package provider

import "context"

type Provider interface {
	StartSession(ctx context.Context, workspacePath string) (Session, error)
	RunTurn(
		ctx context.Context,
		session Session,
		workspacePath, title, inputText string,
		onEvent func(event string, payload map[string]any),
	) (*TurnResult, error)
}

type Session interface {
	SessionID() string
	PID() *int
	Close() error
}

type TurnResult struct {
	TurnID, LastEvent, LastMessage string

	InputTokens, OutputTokens, TotalTokens int
	RateLimits                           map[string]any
	EndedWithError                       error
}

type ToolExecutor interface {
	ToolSpecs() []map[string]any
	Execute(tool string, arguments any) map[string]any
}

