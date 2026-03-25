package agent

import (
	"context"
	"fmt"
	"strings"

	symphonycfg "github.com/wibus-wee/synclax/pkg/symphony/config"
	"github.com/wibus-wee/synclax/pkg/symphony/domain"
	"github.com/wibus-wee/synclax/pkg/symphony/provider"
	"github.com/wibus-wee/synclax/pkg/symphony/template"
	"github.com/wibus-wee/synclax/pkg/symphony/tracker"
	"github.com/wibus-wee/synclax/pkg/symphony/workspace"
)

type Worker struct {
	Tracker    tracker.Client
	Workspace  *workspace.Manager
	Provider   provider.Provider
	Renderer   *template.Renderer
	Config     symphonycfg.EffectiveConfig
	WorkerHost *string
}

type Update struct {
	Event   string
	Payload map[string]any
}

type Result struct {
	FinalIssue    domain.Issue
	TurnsRun      int
	WorkspacePath string
	InputTokens   int
	OutputTokens  int
	TotalTokens   int
	RateLimits    map[string]any
}

func (w *Worker) RunAttempt(ctx context.Context, issue domain.Issue, attempt *int, onUpdate func(Update)) (Result, error) {
	emit := func(event string, payload map[string]any) {
		if onUpdate == nil {
			return
		}
		onUpdate(Update{Event: event, Payload: payload})
	}
	emitPhase := func(phase string, payload map[string]any) {
		if payload == nil {
			payload = map[string]any{}
		}
		payload["phase"] = phase
		emit("symphony/phase", payload)
	}

	ws, err := w.Workspace.CreateForIssue(ctx, issue.Identifier, w.WorkerHost)
	if err != nil {
		return Result{}, err
	}
	emitPhase("PreparingWorkspace", map[string]any{"workspace_path": ws.Path})

	if err := w.Workspace.BeforeRun(ctx, ws); err != nil {
		return Result{}, err
	}
	defer w.Workspace.AfterRunBestEffort(ctx, ws)

	emitPhase("LaunchingAgentProcess", nil)
	session, err := w.Provider.StartSession(ctx, ws.Path, w.WorkerHost)
	if err != nil {
		return Result{}, err
	}
	defer session.Close()
	emitPhase("InitializingSession", map[string]any{
		"session_id": session.SessionID(),
		"thread_id":  session.SessionID(),
		"agent_pid":  session.PID(),
	})

	turns := 0
	var (
		inputTokens  = 0
		outputTokens = 0
		totalTokens  = 0
		rateLimits   map[string]any
	)
	maxTurns := w.Config.Agent.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 20
	}

	for turnNumber := 1; turnNumber <= maxTurns; turnNumber++ {
		turns++

		emitPhase("BuildingPrompt", map[string]any{"turn_count": turnNumber})
		promptText, err := w.buildTurnPrompt(issue, attempt, turnNumber, maxTurns)
		if err != nil {
			return Result{}, err
		}

		emitPhase("StreamingTurn", map[string]any{"turn_count": turnNumber})
		title := fmt.Sprintf("%s: %s", issue.Identifier, issue.Title)
		turnRes, err := w.Provider.RunTurn(ctx, session, ws.Path, title, promptText, func(event string, payload map[string]any) {
			emit(event, payload)
		})
		if err != nil {
			return Result{}, err
		}
		if turnRes != nil {
			inputTokens += turnRes.InputTokens
			outputTokens += turnRes.OutputTokens
			totalTokens += turnRes.TotalTokens
			if turnRes.RateLimits != nil {
				rateLimits = turnRes.RateLimits
			}
		}
		emit("symphony/token_update", map[string]any{
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
			"total_tokens":  totalTokens,
		})

		refreshed, err := w.Tracker.FetchIssueStatesByIDs(ctx, []string{issue.ID})
		if err != nil {
			return Result{}, err
		}
		if len(refreshed) == 0 || refreshed[0].ID != issue.ID {
			issue.State = ""
		} else {
			issue.State = refreshed[0].State
		}

		if !isActive(issue.State, w.Config.Tracker.ActiveStates, w.Config.Tracker.TerminalStates) {
			break
		}
	}

	emitPhase("Finishing", map[string]any{"turn_count": turns})
	return Result{
		FinalIssue:    issue,
		TurnsRun:      turns,
		WorkspacePath: ws.Path,
		InputTokens:   inputTokens,
		OutputTokens:  outputTokens,
		TotalTokens:   totalTokens,
		RateLimits:    rateLimits,
	}, nil
}

func (w *Worker) buildTurnPrompt(issue domain.Issue, attempt *int, turnNumber, maxTurns int) (string, error) {
	if turnNumber <= 1 {
		return w.Renderer.RenderIssuePrompt(issue, attempt)
	}
	// Continuation guidance: do not resend full task prompt.
	msg := strings.TrimSpace(fmt.Sprintf(
		"Continue working on issue %s (%s). Do not repeat the full original task prompt. "+
			"Stop if the issue is no longer in an active state. (turn %d/%d, attempt=%v)",
		issue.Identifier, issue.Title, turnNumber, maxTurns, attemptValue(attempt),
	))
	return msg, nil
}

func attemptValue(a *int) any {
	if a == nil {
		return nil
	}
	return *a
}

func isActive(state string, active []string, terminal []string) bool {
	s := strings.ToLower(strings.TrimSpace(state))
	for _, t := range terminal {
		if strings.ToLower(strings.TrimSpace(t)) == s {
			return false
		}
	}
	for _, a := range active {
		if strings.ToLower(strings.TrimSpace(a)) == s {
			return true
		}
	}
	return false
}
