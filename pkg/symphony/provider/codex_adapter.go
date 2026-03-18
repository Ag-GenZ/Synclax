package provider

import (
	"context"
	"errors"

	"github.com/wibus-wee/synclax/pkg/symphony/codex"
)

type codexProvider struct {
	srv *codex.AppServer
}

func newCodexProvider(srv *codex.AppServer) Provider {
	return &codexProvider{srv: srv}
}

type codexSession struct {
	inner *codex.Session
}

func (s *codexSession) SessionID() string {
	if s == nil || s.inner == nil {
		return ""
	}
	return s.inner.ThreadID()
}

func (s *codexSession) PID() *int {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.PID()
}

func (s *codexSession) Close() error {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.Close()
}

func (p *codexProvider) StartSession(ctx context.Context, workspacePath string) (Session, error) {
	if p == nil || p.srv == nil {
		return nil, &Error{Category: ErrProviderNotFound.Error(), Err: errors.New("nil provider")}
	}
	sess, err := p.srv.StartSession(ctx, workspacePath)
	if err != nil {
		return nil, wrapCodexErr(err)
	}
	return &codexSession{inner: sess}, nil
}

func (p *codexProvider) RunTurn(
	ctx context.Context,
	session Session,
	workspacePath, title, inputText string,
	onEvent func(event string, payload map[string]any),
) (*TurnResult, error) {
	if p == nil || p.srv == nil {
		return nil, &Error{Category: ErrProviderNotFound.Error(), Err: errors.New("nil provider")}
	}
	cs, ok := session.(*codexSession)
	if !ok || cs == nil || cs.inner == nil {
		return nil, &Error{Category: ErrResponseError.Error(), Err: errors.New("invalid session type")}
	}
	res, err := p.srv.RunTurn(ctx, cs.inner, workspacePath, title, inputText, onEvent)
	if err != nil {
		return nil, wrapCodexErr(err)
	}
	if res == nil {
		return nil, nil
	}
	return &TurnResult{
		TurnID:         res.TurnID,
		LastEvent:      res.LastEvent,
		LastMessage:    res.LastMessage,
		InputTokens:    res.InputTokens,
		OutputTokens:   res.OutputTokens,
		TotalTokens:    res.TotalTokens,
		RateLimits:     res.RateLimits,
		EndedWithError: res.EndedWithError,
	}, nil
}

func wrapCodexErr(err error) error {
	if err == nil {
		return nil
	}
	var ce *codex.Error
	if !errors.As(err, &ce) || ce == nil {
		return err
	}
	cat := ce.Category
	switch cat {
	case codex.ErrCodexNotFound.Error():
		cat = ErrProviderNotFound.Error()
	case codex.ErrPortExit.Error():
		cat = ErrProcessExit.Error()
	}
	return &Error{Category: cat, Err: ce.Err}
}

