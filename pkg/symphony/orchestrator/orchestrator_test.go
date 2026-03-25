package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wibus-wee/synclax/pkg/symphony/agent"
	"github.com/wibus-wee/synclax/pkg/symphony/domain"
	"github.com/wibus-wee/synclax/pkg/symphony/provider"
	"github.com/wibus-wee/synclax/pkg/symphony/runtime"
	"github.com/wibus-wee/synclax/pkg/symphony/tracker"
	"github.com/wibus-wee/synclax/pkg/symphony/workspace"
)

type fakeTracker struct {
	candidates []domain.Issue
	statesByID map[string]string
}

func (f *fakeTracker) FetchCandidateIssues(_ context.Context) ([]domain.Issue, error) {
	return append([]domain.Issue(nil), f.candidates...), nil
}

func (f *fakeTracker) FetchIssuesByStates(_ context.Context, _ []string) ([]domain.Issue, error) {
	return []domain.Issue{}, nil
}

func (f *fakeTracker) FetchIssueStatesByIDs(_ context.Context, ids []string) ([]domain.Issue, error) {
	out := make([]domain.Issue, 0, len(ids))
	for _, id := range ids {
		if st, ok := f.statesByID[id]; ok {
			out = append(out, domain.Issue{ID: id, Identifier: "X", State: st})
		}
	}
	return out, nil
}

type fakeBootstrapTracker struct {
	fakeTracker
	calls int
	err   error
}

func (f *fakeBootstrapTracker) EnsureSynclaxWorkflow(_ context.Context) error {
	f.calls++
	return f.err
}

type fakeRunner struct {
	called chan struct{}
	res    agent.Result
	err    error
}

func (r *fakeRunner) RunAttempt(_ context.Context, _ domain.Issue, _ *int, _ func(agent.Update)) (agent.Result, error) {
	select {
	case <-r.called:
	default:
		close(r.called)
	}
	return r.res, r.err
}

type fakeProvider struct{}

func (p *fakeProvider) StartSession(_ context.Context, _ string, _ *string) (provider.Session, error) {
	return nil, nil
}

func (p *fakeProvider) RunTurn(
	_ context.Context,
	_ provider.Session,
	_, _, _ string,
	_ func(event string, payload map[string]any),
) (*provider.TurnResult, error) {
	return nil, nil
}

func mustOrchestrator(t *testing.T, workflow string) (*Orchestrator, context.CancelFunc) {
	t.Helper()
	dir := t.TempDir()
	workflowPath := filepath.Join(dir, "WORKFLOW.md")
	if err := os.WriteFile(workflowPath, []byte(workflow), 0o644); err != nil {
		t.Fatal(err)
	}

	o, err := New(Options{WorkflowPath: workflowPath})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := o.runtime.Start(ctx); err != nil {
		cancel()
		t.Fatalf("runtime.Start error: %v", err)
	}
	rt, _ := o.runtime.Get()
	o.cfg = rt.Config
	o.provider = &fakeProvider{}

	ws, err := workspace.NewManager(t.TempDir(), workspace.HookScripts{})
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	o.workspace = ws

	return o, cancel
}

func TestTick_DispatchesAndQueuesContinuationRetry(t *testing.T) {
	workflow := `---
tracker:
  kind: linear
  api_key: x
  project_slug: proj
polling:
  interval_ms: 50
agent:
  max_concurrent_agents: 10
codex:
  command: "true"
---`
	o, cancel := mustOrchestrator(t, workflow)
	t.Cleanup(cancel)

	issue := domain.Issue{ID: "i1", Identifier: "ABC-1", Title: "Test", State: "Todo"}
	o.tracker = &fakeTracker{
		candidates: []domain.Issue{issue},
		statesByID: map[string]string{"i1": "Todo"},
	}

	r := &fakeRunner{
		called: make(chan struct{}),
		res: agent.Result{
			FinalIssue:    issue,
			WorkspacePath: "",
		},
	}
	o.newRunner = func(_ *runtime.EffectiveRuntime, _ tracker.Client, _ *workspace.Manager, _ provider.Provider, _ *string) attemptRunner {
		return r
	}

	o.tick(context.Background())

	select {
	case <-r.called:
	case <-time.After(2 * time.Second):
		t.Fatal("expected worker to be called")
	}

	// Dispatch + retry scheduling happen in a goroutine; allow a short window for the retry entry
	// to be recorded in orchestrator state.
	var retry *RetryEntry
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		o.mu.Lock()
		retry = o.retries["i1"]
		o.mu.Unlock()
		if retry != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	o.mu.Lock()
	defer o.mu.Unlock()
	if _, ok := o.claimed["i1"]; !ok {
		t.Fatal("expected issue to be claimed")
	}
	if retry == nil {
		t.Fatal("expected retry entry")
	}
	if retry.Attempt != 1 || retry.DelayType != "continuation" {
		t.Fatalf("unexpected retry entry: %#v", retry)
	}
}

func TestTick_TodoWithNonTerminalBlockerIsNotDispatched(t *testing.T) {
	workflow := `---
tracker:
  kind: linear
  api_key: x
  project_slug: proj
agent:
  max_concurrent_agents: 10
codex:
  command: "true"
---`
	o, cancel := mustOrchestrator(t, workflow)
	t.Cleanup(cancel)

	o.tracker = &fakeTracker{
		candidates: []domain.Issue{{
			ID:         "i1",
			Identifier: "ABC-1",
			Title:      "Test",
			State:      "Todo",
			BlockedBy: []domain.BlockerRef{
				{State: strPtr("In Progress")},
			},
		}},
		statesByID: map[string]string{},
	}

	r := &fakeRunner{called: make(chan struct{})}
	o.newRunner = func(_ *runtime.EffectiveRuntime, _ tracker.Client, _ *workspace.Manager, _ provider.Provider, _ *string) attemptRunner {
		return r
	}

	o.tick(context.Background())

	select {
	case <-r.called:
		t.Fatal("did not expect worker to be called")
	case <-time.After(200 * time.Millisecond):
		// ok
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if len(o.claimed) != 0 {
		t.Fatalf("expected no claims, got %#v", o.claimed)
	}
}

func TestOnWorkerExit_SkipsContinuationWhenIssueBecomesNonActive(t *testing.T) {
	workflow := `---
tracker:
  kind: linear
  api_key: x
  project_slug: proj
  active_states:
    - Todo
    - In Progress
    - Rework
    - Merging
  terminal_states:
    - Done
codex:
  command: "true"
---`
	o, cancel := mustOrchestrator(t, workflow)
	t.Cleanup(cancel)

	o.tracker = &fakeTracker{
		statesByID: map[string]string{"i1": "Human Review"},
	}
	o.claim("i1")

	entry := &RunningEntry{
		Issue:      domain.Issue{ID: "i1", Identifier: "ABC-1", Title: "Test", State: "In Progress"},
		IssueID:    "i1",
		Identifier: "ABC-1",
		StartedAt:  time.Now().Add(-time.Second),
	}
	res := agent.Result{
		FinalIssue: domain.Issue{ID: "i1", Identifier: "ABC-1", Title: "Test", State: "In Progress"},
	}

	o.onWorkerExit(context.Background(), entry, res, nil)

	o.mu.Lock()
	defer o.mu.Unlock()
	if retry := o.retries["i1"]; retry != nil {
		t.Fatalf("expected no continuation retry, got %#v", retry)
	}
	if _, ok := o.claimed["i1"]; ok {
		t.Fatal("expected issue claim to be released")
	}
}

func TestOnWorkerExit_QueuesContinuationWhenIssueStaysActive(t *testing.T) {
	workflow := `---
tracker:
  kind: linear
  api_key: x
  project_slug: proj
  active_states:
    - Todo
    - In Progress
    - Rework
    - Merging
  terminal_states:
    - Done
codex:
  command: "true"
---`
	o, cancel := mustOrchestrator(t, workflow)
	t.Cleanup(cancel)

	o.tracker = &fakeTracker{
		statesByID: map[string]string{"i1": "In Progress"},
	}
	o.claim("i1")

	entry := &RunningEntry{
		Issue:      domain.Issue{ID: "i1", Identifier: "ABC-1", Title: "Test", State: "In Progress"},
		IssueID:    "i1",
		Identifier: "ABC-1",
		StartedAt:  time.Now().Add(-time.Second),
	}
	res := agent.Result{
		FinalIssue: domain.Issue{ID: "i1", Identifier: "ABC-1", Title: "Test", State: "In Progress"},
	}

	o.onWorkerExit(context.Background(), entry, res, nil)

	o.mu.Lock()
	retry := o.retries["i1"]
	o.mu.Unlock()
	if retry == nil {
		t.Fatal("expected continuation retry")
	}
	if retry.timerHandle != nil {
		retry.timerHandle.Stop()
	}
}

func TestReconcile_StopsRunningIssueWhenTrackerNoLongerReturnsState(t *testing.T) {
	workflow := `---
tracker:
  kind: linear
  api_key: x
  project_slug: proj
  active_states:
    - Todo
    - In Progress
  terminal_states:
    - Done
codex:
  command: "true"
---`
	o, cancel := mustOrchestrator(t, workflow)
	t.Cleanup(cancel)

	o.tracker = &fakeTracker{statesByID: map[string]string{}}
	o.running["i1"] = &RunningEntry{
		Issue:      domain.Issue{ID: "i1", Identifier: "ABC-1", Title: "Test", State: "In Progress"},
		IssueID:    "i1",
		Identifier: "ABC-1",
		StartedAt:  time.Now().Add(-time.Second),
	}
	o.claim("i1")

	o.reconcile(context.Background())

	o.mu.Lock()
	defer o.mu.Unlock()
	entry := o.running["i1"]
	if entry == nil {
		t.Fatal("expected running entry to remain until worker exit cleanup")
	}
	if entry.Phase != PhaseCanceledByReconciliation {
		t.Fatalf("expected reconcile cancel phase, got %q", entry.Phase)
	}
	if !entry.suppressRetry {
		t.Fatal("expected reconcile cancel to suppress retries")
	}
	if _, ok := o.claimed["i1"]; ok {
		t.Fatal("expected claim to be released when tracker no longer returns issue state")
	}
}

func TestBootstrapWorkflowTracker_RunsWhenEnabled(t *testing.T) {
	workflow := `---
tracker:
  kind: linear
  api_key: x
  project_slug: proj
  bootstrap_synclax_workflow: true
codex:
  command: "true"
---`
	o, cancel := mustOrchestrator(t, workflow)
	t.Cleanup(cancel)

	tr := &fakeBootstrapTracker{}
	o.tracker = tr

	if err := o.bootstrapWorkflowTracker(context.Background()); err != nil {
		t.Fatalf("bootstrapWorkflowTracker error: %v", err)
	}
	if tr.calls != 1 {
		t.Fatalf("expected 1 bootstrap call, got %d", tr.calls)
	}
}

func TestBootstrapWorkflowTracker_SkipsWhenDisabled(t *testing.T) {
	workflow := `---
tracker:
  kind: linear
  api_key: x
  project_slug: proj
codex:
  command: "true"
---`
	o, cancel := mustOrchestrator(t, workflow)
	t.Cleanup(cancel)

	tr := &fakeBootstrapTracker{}
	o.tracker = tr

	if err := o.bootstrapWorkflowTracker(context.Background()); err != nil {
		t.Fatalf("bootstrapWorkflowTracker error: %v", err)
	}
	if tr.calls != 0 {
		t.Fatalf("expected 0 bootstrap calls, got %d", tr.calls)
	}
}

func strPtr(s string) *string { return &s }
