package orchestrator

import (
	"context"
	"time"

	"github.com/wibus-wee/synclax/pkg/symphony/domain"
)

type RunPhase string

const (
	PhasePreparingWorkspace       RunPhase = "PreparingWorkspace"
	PhaseBuildingPrompt           RunPhase = "BuildingPrompt"
	PhaseLaunchingAgentProcess    RunPhase = "LaunchingAgentProcess"
	PhaseInitializingSession      RunPhase = "InitializingSession"
	PhaseStreamingTurn            RunPhase = "StreamingTurn"
	PhaseFinishing                RunPhase = "Finishing"
	PhaseSucceeded                RunPhase = "Succeeded"
	PhaseFailed                   RunPhase = "Failed"
	PhaseTimedOut                 RunPhase = "TimedOut"
	PhaseStalled                  RunPhase = "Stalled"
	PhaseCanceledByReconciliation RunPhase = "CanceledByReconciliation"
)

type LiveSession struct {
	SessionID          string     `json:"session_id,omitempty"`
	ThreadID           string     `json:"thread_id,omitempty"`
	TurnID             string     `json:"turn_id,omitempty"`
	CodexAppServerPID  *int       `json:"codex_app_server_pid,omitempty"`
	LastCodexEvent     *string    `json:"last_codex_event,omitempty"`
	LastCodexTimestamp *time.Time `json:"last_codex_timestamp,omitempty"`
	LastCodexMessage   *string    `json:"last_codex_message,omitempty"`
	CodexInputTokens   int        `json:"codex_input_tokens"`
	CodexOutputTokens  int        `json:"codex_output_tokens"`
	CodexTotalTokens   int        `json:"codex_total_tokens"`
	TurnCount          int        `json:"turn_count"`
}

type RunningEntry struct {
	Issue         domain.Issue `json:"issue"`
	IssueID       string       `json:"issue_id"`
	Identifier    string       `json:"issue_identifier"`
	Attempt       *int         `json:"attempt,omitempty"`
	WorkspacePath string       `json:"workspace_path"`
	StartedAt     time.Time    `json:"started_at"`
	Phase         RunPhase     `json:"phase"`
	Live          LiveSession  `json:"live"`

	cancel        context.CancelFunc
	suppressRetry bool
	cleanupOnExit bool
}

type RetryEntry struct {
	IssueID     string    `json:"issue_id"`
	Identifier  string    `json:"identifier"`
	Attempt     int       `json:"attempt"`
	DueAt       time.Time `json:"due_at"`
	Error       *string   `json:"error,omitempty"`
	DelayType   string    `json:"delay_type,omitempty"`
	timerHandle *time.Timer
}

type CodexTotals struct {
	InputTokens    int     `json:"input_tokens"`
	OutputTokens   int     `json:"output_tokens"`
	TotalTokens    int     `json:"total_tokens"`
	SecondsRunning float64 `json:"seconds_running"`
}
