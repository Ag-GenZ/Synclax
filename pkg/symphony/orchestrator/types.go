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

const maxEventLog = 100

type LiveEvent struct {
	Timestamp time.Time `json:"ts"`
	Event     string    `json:"event"`
	Message   string    `json:"message,omitempty"`
}

type LiveSession struct {
	SessionID          string      `json:"session_id,omitempty"`
	ThreadID           string      `json:"thread_id,omitempty"`
	TurnID             string      `json:"turn_id,omitempty"`
	AgentPID           *int        `json:"agent_pid,omitempty"`
	LastAgentEvent     *string     `json:"last_agent_event,omitempty"`
	LastAgentTimestamp *time.Time  `json:"last_agent_timestamp,omitempty"`
	LastAgentMessage   *string     `json:"last_agent_message,omitempty"`
	InputTokens        int         `json:"input_tokens"`
	OutputTokens       int         `json:"output_tokens"`
	TotalTokens        int         `json:"total_tokens"`
	TurnCount          int         `json:"turn_count"`
	EventLog           []LiveEvent `json:"event_log,omitempty"`
}

type RunningEntry struct {
	Issue         domain.Issue `json:"issue"`
	IssueID       string       `json:"issue_id"`
	Identifier    string       `json:"issue_identifier"`
	Attempt       *int         `json:"attempt,omitempty"`
	WorkspacePath string       `json:"workspace_path"`
	WorkerHost    *string      `json:"worker_host,omitempty"`
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

type CompletedEntry struct {
	Issue           domain.Issue `json:"issue"`
	IssueID         string       `json:"issue_id"`
	IssueIdentifier string       `json:"issue_identifier"`

	Attempt       *int      `json:"attempt,omitempty"`
	WorkspacePath string    `json:"workspace_path,omitempty"`
	StartedAt     time.Time `json:"started_at"`
	EndedAt       time.Time `json:"ended_at"`
	DurationSecs  float64   `json:"duration_secs"`

	Status string  `json:"status"`
	Error  *string `json:"error,omitempty"`

	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
	TurnsRun          int `json:"turns_run"`

	ThreadID          *string `json:"thread_id,omitempty"`
	TurnID            *string `json:"turn_id,omitempty"`
	LastAgentEvent    *string `json:"last_agent_event,omitempty"`
	LastAgentMessage  *string `json:"last_agent_message,omitempty"`
}

type AgentTotals struct {
	InputTokens    int     `json:"input_tokens"`
	OutputTokens   int     `json:"output_tokens"`
	TotalTokens    int     `json:"total_tokens"`
	SecondsRunning float64 `json:"seconds_running"`
}
