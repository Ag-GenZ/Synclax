package orchestrator

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wibus-wee/synclax/pkg/symphony/agent"
	symphonycfg "github.com/wibus-wee/synclax/pkg/symphony/config"
	"github.com/wibus-wee/synclax/pkg/symphony/domain"
	symphonylog "github.com/wibus-wee/synclax/pkg/symphony/logging"
	"github.com/wibus-wee/synclax/pkg/symphony/provider"
	"github.com/wibus-wee/synclax/pkg/symphony/runtime"
	"github.com/wibus-wee/synclax/pkg/symphony/tracker"
	"github.com/wibus-wee/synclax/pkg/symphony/tracker/linear"
	"github.com/wibus-wee/synclax/pkg/symphony/workspace"
)

type Options struct {
	WorkflowPath string
	PortOverride *int
	StatsStore   StatsStore
}

type Orchestrator struct {
	runtime *runtime.Manager

	mu sync.Mutex

	appliedRevision int64
	cfg             symphonycfg.EffectiveConfig
	tracker         tracker.Client
	workspace       *workspace.Manager
	provider        provider.Provider

	newRunner func(rt *runtime.EffectiveRuntime, tr tracker.Client, ws *workspace.Manager, prov provider.Provider) attemptRunner

	running map[string]*RunningEntry
	claimed map[string]struct{}
	retries map[string]*RetryEntry

	completed        map[string]struct{}
	completedHistory []CompletedEntry

	stateDir string
	stats    StatsStore

	agentTotals     AgentTotals
	agentRateLimits map[string]any

	httpServer *http.Server
}

type attemptRunner interface {
	RunAttempt(ctx context.Context, issue domain.Issue, attempt *int, onUpdate func(agent.Update)) (agent.Result, error)
}

type StatsStore interface {
	Load(ctx context.Context, maxAttempts int) (AgentTotals, map[string]any, []CompletedEntry, error)
	Record(ctx context.Context, totals AgentTotals, rateLimits map[string]any, entry CompletedEntry) error
}

func (o *Orchestrator) Snapshot() map[string]any {
	if o == nil {
		return map[string]any{
			"running":      []any{},
			"retrying":     []any{},
			"agent_totals": AgentTotals{},
			"rate_limits":  map[string]any{},
		}
	}
	return o.snapshot()
}

func New(opts Options) (*Orchestrator, error) {
	workflowPath := strings.TrimSpace(opts.WorkflowPath)
	if workflowPath == "" {
		return nil, errors.New("workflow path is required")
	}
	o := &Orchestrator{
		runtime:   runtime.NewManager(workflowPath),
		running:   map[string]*RunningEntry{},
		claimed:   map[string]struct{}{},
		retries:   map[string]*RetryEntry{},
	completed: map[string]struct{}{},
	stats:     opts.StatsStore,
	}
	o.newRunner = func(rt *runtime.EffectiveRuntime, tr tracker.Client, ws *workspace.Manager, prov provider.Provider) attemptRunner {
		return &agent.Worker{
			Tracker:   tr,
			Workspace: ws,
			Provider:  prov,
			Renderer:  rt.Renderer,
			Config:    rt.Config,
		}
	}
	return o, nil
}

func (o *Orchestrator) Run(ctx context.Context, portOverride *int) error {
	if err := o.runtime.Start(ctx); err != nil {
		return err
	}

	// Initial dependency build.
	if err := o.applyRuntimeLocked(portOverride); err != nil {
		return err
	}

	// Startup terminal cleanup is best-effort.
	o.startupCleanupTerminal(ctx)

	// Immediate tick, then loop.
	nextInterval := o.cfg.Polling.Interval
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			o.shutdownHTTP(context.Background())
			return nil
		case <-timer.C:
			o.runtime.RefreshIfNeeded()
			if err := o.applyRuntimeLocked(portOverride); err != nil {
				log.Printf("symphony runtime_apply status=failed error=%v", err)
			}
			o.tick(ctx)
			interval := o.getPollInterval()
			if interval <= 0 {
				interval = nextInterval
			}
			nextInterval = interval
			timer.Reset(nextInterval)
		}
	}
}

type persistedState struct {
	AgentTotals *AgentTotals   `json:"agent_totals,omitempty"`
	CodexTotals *AgentTotals   `json:"codex_totals,omitempty"` // legacy
	RateLimits  map[string]any `json:"rate_limits,omitempty"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

func (o *Orchestrator) ensureStateDirLocked() {
	if o.workspace == nil {
		return
	}
	root := strings.TrimSpace(o.workspace.Root())
	if root == "" {
		return
	}
	dir := filepath.Join(root, ".symphony_state")
	if o.stateDir == dir {
		return
	}
	_ = os.MkdirAll(dir, 0o755)
	o.stateDir = dir

	// Best-effort load on (re)bind.
	o.loadPersistedLocked()
}

func (o *Orchestrator) statePathLocked(name string) string {
	if strings.TrimSpace(o.stateDir) == "" {
		return ""
	}
	return filepath.Join(o.stateDir, name)
}

func (o *Orchestrator) loadPersistedLocked() {
	if o.stats != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if totals, rl, attempts, err := o.stats.Load(ctx, 200); err == nil {
			o.agentTotals = totals
			if rl != nil {
				o.agentRateLimits = rl
			}
			if attempts != nil {
				o.completedHistory = attempts
			}
			return
		}
	}

	// totals
	totalsPath := o.statePathLocked("totals.json")
	if totalsPath != "" {
		b, err := os.ReadFile(totalsPath)
		if err == nil && len(bytesTrimSpace(b)) > 0 {
			var st persistedState
			if json.Unmarshal(b, &st) == nil {
				if st.AgentTotals != nil {
					o.agentTotals = *st.AgentTotals
				} else if st.CodexTotals != nil {
					o.agentTotals = *st.CodexTotals
				}
				if st.RateLimits != nil {
					o.agentRateLimits = st.RateLimits
				}
			}
		}
	}

	// completed history (last N)
	attemptsPath := o.statePathLocked("attempts.jsonl")
	if attemptsPath == "" {
		return
	}
	f, err := os.Open(attemptsPath)
	if err != nil {
		return
	}
	defer f.Close()

	const maxKeep = 200
	buf := bufio.NewScanner(f)
	keep := make([]CompletedEntry, 0, maxKeep)
	for buf.Scan() {
		line := strings.TrimSpace(buf.Text())
		if line == "" {
			continue
		}
		var entry CompletedEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		keep = append(keep, entry)
		if len(keep) > maxKeep {
			keep = keep[len(keep)-maxKeep:]
		}
	}
	o.completedHistory = keep
}

func bytesTrimSpace(b []byte) []byte {
	// avoid pulling bytes just for TrimSpace
	i := 0
	for i < len(b) && (b[i] == ' ' || b[i] == '\n' || b[i] == '\r' || b[i] == '\t') {
		i++
	}
	j := len(b) - 1
	for j >= i && (b[j] == ' ' || b[j] == '\n' || b[j] == '\r' || b[j] == '\t') {
		j--
	}
	if j < i {
		return nil
	}
	return b[i : j+1]
}

func (o *Orchestrator) persistStateBestEffort(totals AgentTotals, rateLimits map[string]any) {
	o.mu.Lock()
	path := o.statePathLocked("totals.json")
	o.mu.Unlock()
	if path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	tmp := path + ".tmp"
	t := totals
	st := persistedState{
		AgentTotals: &t,
		RateLimits:  rateLimits,
		UpdatedAt:   time.Now().UTC(),
	}
	b, err := json.Marshal(st)
	if err != nil {
		return
	}
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

func (o *Orchestrator) appendAttemptBestEffort(entry CompletedEntry) {
	o.mu.Lock()
	path := o.statePathLocked("attempts.jsonl")
	o.mu.Unlock()
	if path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	b, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_, _ = f.Write(append(b, '\n'))
}

func (o *Orchestrator) recordAttemptBestEffort(totals AgentTotals, rateLimits map[string]any, entry CompletedEntry) {
	if o == nil {
		return
	}
	if o.stats != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = o.stats.Record(ctx, totals, rateLimits, entry)
		return
	}
	o.appendAttemptBestEffort(entry)
	o.persistStateBestEffort(totals, rateLimits)
}

func (o *Orchestrator) getPollInterval() time.Duration {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.cfg.Polling.Interval
}

func (o *Orchestrator) applyRuntimeLocked(portOverride *int) error {
	rt, rev := o.runtime.Get()
	if rt == nil {
		return errors.New("no runtime loaded")
	}

	o.mu.Lock()
	defer o.mu.Unlock()
	if rev == o.appliedRevision {
		return nil
	}

	o.cfg = rt.Config
	o.appliedRevision = rev
	symphonylog.Configure(o.cfg.Logging)

	ws, err := workspace.NewManager(o.cfg.Workspace.Root, workspace.HookScripts{
		AfterCreate:  o.cfg.Hooks.AfterCreate,
		BeforeRun:    o.cfg.Hooks.BeforeRun,
		AfterRun:     o.cfg.Hooks.AfterRun,
		BeforeRemove: o.cfg.Hooks.BeforeRemove,
		Timeout:      o.cfg.Hooks.Timeout,
	})
	if err != nil {
		return err
	}
	o.workspace = ws
	o.ensureStateDirLocked()

	tr, err := linear.New(linear.Options{
		Endpoint:     o.cfg.Tracker.Endpoint,
		APIKey:       o.cfg.Tracker.APIKey,
		ProjectSlug:  o.cfg.Tracker.ProjectSlug,
		ActiveStates: o.cfg.Tracker.ActiveStates,
		PageSize:     o.cfg.Tracker.PageSize,
		Timeout:      o.cfg.Tracker.Timeout,
	})
	if err != nil {
		return err
	}
	o.tracker = tr

	prov, err := provider.Build(o.cfg)
	if err != nil {
		return err
	}
	o.provider = prov

	port := o.cfg.Server.Port
	if portOverride != nil {
		if *portOverride < 0 {
			port = nil
		} else {
			port = portOverride
		}
	}
	o.ensureHTTPServerLocked(port)

	log.Printf("symphony runtime_apply status=ok revision=%d polling_interval_ms=%d workspace_root=%s", rev, o.cfg.Polling.Interval.Milliseconds(), o.cfg.Workspace.Root)
	return nil
}

func (o *Orchestrator) ensureHTTPServerLocked(port *int) {
	var old *http.Server

	if port == nil {
		old = o.httpServer
		o.httpServer = nil
		if old != nil {
			go func(s *http.Server) { _ = s.Shutdown(context.Background()) }(old)
		}
		return
	}

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	if o.httpServer != nil && o.httpServer.Addr == addr {
		return
	}
	if o.httpServer != nil {
		old = o.httpServer
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/snapshot", func(w http.ResponseWriter, _ *http.Request) {
		snap := o.snapshot()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(snap)
	})
	mux.HandleFunc("/api/v1/state", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		snap := o.snapshot()
		meta := map[string]any{
			"workflow_path": o.runtime.WorkflowPath(),
			"revision":      o.appliedRevision,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"meta":     meta,
			"snapshot": snap,
		})
	})
	mux.HandleFunc("/api/v1/refresh", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		o.runtime.RefreshIfNeeded()
		_ = o.applyRuntimeLocked(nil)
		o.tick(ctx)

		snap := o.snapshot()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(snap)
	})

	corsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		mux.ServeHTTP(w, r)
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           corsHandler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	o.httpServer = srv

	if old != nil {
		go func(s *http.Server) { _ = s.Shutdown(context.Background()) }(old)
	}
	go func() {
		log.Printf("symphony http_server status=starting addr=%s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("symphony http_server status=failed error=%v", err)
		}
	}()
}

func (o *Orchestrator) shutdownHTTP(ctx context.Context) {
	o.mu.Lock()
	srv := o.httpServer
	o.httpServer = nil
	o.mu.Unlock()
	if srv == nil {
		return
	}
	_ = srv.Shutdown(ctx)
}

func (o *Orchestrator) startupCleanupTerminal(ctx context.Context) {
	o.mu.Lock()
	tr := o.tracker
	ws := o.workspace
	terminalStates := append([]string(nil), o.cfg.Tracker.TerminalStates...)
	o.mu.Unlock()

	issues, err := tr.FetchIssuesByStates(ctx, terminalStates)
	if err != nil {
		log.Printf("symphony startup_cleanup status=warning error=%v", err)
		return
	}
	for _, is := range issues {
		ws.RemoveBestEffort(ctx, is.Identifier)
	}
	log.Printf("symphony startup_cleanup status=ok terminal_count=%d", len(issues))
}

func (o *Orchestrator) tick(ctx context.Context) {
	o.reconcile(ctx)
	if !o.dispatchPreflightOK() {
		return
	}

	o.mu.Lock()
	tr := o.tracker
	o.mu.Unlock()

	candidates, err := tr.FetchCandidateIssues(ctx)
	if err != nil {
		log.Printf("symphony fetch_candidates status=failed error=%v", err)
		return
	}
	sortForDispatch(candidates)

	o.dispatchFromCandidates(ctx, candidates)
}

func (o *Orchestrator) dispatchPreflightOK() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	if strings.TrimSpace(o.cfg.Tracker.APIKey) == "" || strings.TrimSpace(o.cfg.Tracker.ProjectSlug) == "" {
		log.Printf("symphony dispatch_validation status=failed error=missing tracker config")
		return false
	}
	if o.provider == nil {
		log.Printf("symphony dispatch_validation status=failed error=missing provider")
		return false
	}
	return true
}

func (o *Orchestrator) reconcile(ctx context.Context) {
	o.mu.Lock()
	running := make([]*RunningEntry, 0, len(o.running))
	for _, r := range o.running {
		running = append(running, r)
	}
	stallTimeout := o.cfg.Agent.StallTimeout
	tr := o.tracker
	activeStates := append([]string(nil), o.cfg.Tracker.ActiveStates...)
	terminalStates := append([]string(nil), o.cfg.Tracker.TerminalStates...)
	o.mu.Unlock()

	// Part A: stall detection
	if stallTimeout > 0 {
		now := time.Now().UTC()
		for _, r := range running {
			last := r.StartedAt
			if r.Live.LastAgentTimestamp != nil {
				last = *r.Live.LastAgentTimestamp
			}
			if now.Sub(last) > stallTimeout {
				log.Printf("symphony stall_detected issue_id=%s identifier=%s", r.IssueID, r.Identifier)
				o.stopRunning(r.IssueID, PhaseStalled, false)
			}
		}
	}

	// Part B: tracker refresh
	if len(running) == 0 {
		return
	}
	ids := make([]string, 0, len(running))
	for _, r := range running {
		ids = append(ids, r.IssueID)
	}
	refreshed, err := tr.FetchIssueStatesByIDs(ctx, ids)
	if err != nil {
		log.Printf("symphony reconcile_refresh status=failed error=%v", err)
		return
	}
	stateByID := map[string]string{}
	for _, is := range refreshed {
		stateByID[is.ID] = is.State
	}
	for _, r := range running {
		state, ok := stateByID[r.IssueID]
		if !ok {
			continue
		}
		if isTerminal(state, terminalStates) {
			log.Printf("symphony reconcile_terminal issue_id=%s identifier=%s state=%s", r.IssueID, r.Identifier, state)
			o.stopRunning(r.IssueID, PhaseCanceledByReconciliation, true)
			// Release claim immediately; retry scheduling is suppressed for reconciliation cancels.
			o.releaseClaim(r.IssueID)
			continue
		}
		if isActiveState(state, activeStates) {
			o.mu.Lock()
			if cur := o.running[r.IssueID]; cur != nil {
				cur.Issue.State = state
			}
			o.mu.Unlock()
			continue
		}
		log.Printf("symphony reconcile_non_active issue_id=%s identifier=%s state=%s", r.IssueID, r.Identifier, state)
		o.stopRunning(r.IssueID, PhaseCanceledByReconciliation, false)
		o.releaseClaim(r.IssueID)
	}
}

func (o *Orchestrator) dispatchFromCandidates(ctx context.Context, candidates []domain.Issue) {
	o.mu.Lock()
	maxGlobal := o.cfg.Agent.MaxConcurrentAgents
	perState := o.cfg.Agent.MaxConcurrentAgentsByState
	activeStates := append([]string(nil), o.cfg.Tracker.ActiveStates...)
	terminalStates := append([]string(nil), o.cfg.Tracker.TerminalStates...)
	rt, _ := o.runtime.Get()
	tr := o.tracker
	ws := o.workspace
	prov := o.provider
	newRunner := o.newRunner
	runningCount := len(o.running)
	stateCounts := map[string]int{}
	for _, r := range o.running {
		stateCounts[strings.ToLower(strings.TrimSpace(r.Issue.State))]++
	}
	o.mu.Unlock()

	available := max(maxGlobal-runningCount, 0)
	if available == 0 {
		return
	}

	for _, issue := range candidates {
		if available <= 0 {
			return
		}
		if !isDispatchEligible(issue, activeStates, terminalStates) {
			continue
		}
		if issueAlreadyClaimed(o, issue.ID) {
			continue
		}
		if !todoBlockerRuleOK(issue, terminalStates) {
			continue
		}
		// Per-state cap
		stateKey := strings.ToLower(strings.TrimSpace(issue.State))
		stateCap := maxGlobal
		if v, ok := perState[stateKey]; ok && v > 0 {
			stateCap = v
		}
		if stateCounts[stateKey] >= stateCap {
			continue
		}

		issueCopy := issue
		o.claim(issueCopy.ID)
		stateCounts[stateKey]++
		available--

		runner := newRunner(rt, tr, ws, prov)
		go o.runWorker(ctx, runner, issueCopy, nil)
	}
}

func (o *Orchestrator) runWorker(parent context.Context, w attemptRunner, issue domain.Issue, attempt *int) {
	ctx, cancel := context.WithCancel(parent)
	start := time.Now().UTC()
	entry := &RunningEntry{
		Issue:         issue,
		IssueID:       issue.ID,
		Identifier:    issue.Identifier,
		Attempt:       attempt,
		WorkspacePath: "", // updated after workspace creation
		StartedAt:     start,
		Phase:         PhasePreparingWorkspace,
		cancel:        cancel,
	}

	o.mu.Lock()
	o.running[issue.ID] = entry
	o.mu.Unlock()

	log.Printf("symphony attempt_start issue_id=%s issue_identifier=%s attempt=%v", issue.ID, issue.Identifier, attemptValue(attempt))

	res, err := w.RunAttempt(ctx, issue, attempt, func(upd agent.Update) {
		o.onWorkerUpdate(issue.ID, upd)
	})

	o.mu.Lock()
	delete(o.running, issue.ID)
	o.mu.Unlock()

	status := "ok"
	if err != nil {
		status = "error"
	}
	sessionID := strings.TrimSpace(entry.Live.SessionID)
	threadID := strings.TrimSpace(entry.Live.ThreadID)
	turnID := strings.TrimSpace(entry.Live.TurnID)
	if sessionID == "" && threadID != "" && turnID != "" {
		sessionID = threadID + "-" + turnID
	}

	log.Printf(
		"symphony attempt_end status=%s issue_id=%s issue_identifier=%s session_id=%s thread_id=%s turn_id=%s turns=%d input_tokens=%d output_tokens=%d total_tokens=%d",
		status,
		entry.IssueID,
		entry.Identifier,
		sessionID,
		threadID,
		turnID,
		res.TurnsRun,
		res.InputTokens,
		res.OutputTokens,
		res.TotalTokens,
	)

	o.onWorkerExit(parent, entry, res, err)
}

func (o *Orchestrator) onWorkerUpdate(issueID string, upd agent.Update) {
	now := time.Now().UTC()
	o.mu.Lock()
	defer o.mu.Unlock()
	r := o.running[issueID]
	if r == nil {
		return
	}
	r.Live.LastAgentTimestamp = &now
	if upd.Event != "" {
		ev := upd.Event
		r.Live.LastAgentEvent = &ev
	}

	extractMessage := func(payload map[string]any) string {
		if payload == nil {
			return ""
		}
		for _, k := range []string{
			"message",
			"text",
			"delta",
			"summaryTextDelta",
			"summary_text_delta",
			"output_text",
			"content",
		} {
			switch v := payload[k].(type) {
			case string:
				if strings.TrimSpace(v) != "" {
					return v
				}
			case map[string]any:
				if s, ok := v["text"].(string); ok && strings.TrimSpace(s) != "" {
					return s
				}
			}
		}
		if item, ok := payload["item"].(map[string]any); ok {
			for _, k := range []string{"message", "text", "delta", "content"} {
				switch v := item[k].(type) {
				case string:
					if strings.TrimSpace(v) != "" {
						return v
					}
				case map[string]any:
					if s, ok := v["text"].(string); ok && strings.TrimSpace(s) != "" {
						return s
					}
				}
			}
		}
		if delta, ok := payload["delta"].(map[string]any); ok {
			if s, ok := delta["text"].(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
			if s, ok := delta["summaryTextDelta"].(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
		return ""
	}

	// Append to event log (cap at maxEventLog entries).
	msg := extractMessage(upd.Payload)
	if upd.Event != "" || msg != "" {
		entry := LiveEvent{Timestamp: now, Event: upd.Event, Message: msg}
		r.Live.EventLog = append(r.Live.EventLog, entry)
		if len(r.Live.EventLog) > maxEventLog {
			r.Live.EventLog = r.Live.EventLog[len(r.Live.EventLog)-maxEventLog:]
		}
	}

	if upd.Payload == nil {
		return
	}

	if phase, ok := upd.Payload["phase"].(string); ok {
		if p, ok := normalizePhase(phase); ok {
			r.Phase = p
		}
	}
	if wsPath, ok := upd.Payload["workspace_path"].(string); ok && strings.TrimSpace(wsPath) != "" {
		r.WorkspacePath = wsPath
	}
	if sessionID, ok := upd.Payload["session_id"].(string); ok && strings.TrimSpace(sessionID) != "" {
		r.Live.SessionID = strings.TrimSpace(sessionID)
	}
	if threadID, ok := upd.Payload["thread_id"].(string); ok && strings.TrimSpace(threadID) != "" {
		r.Live.ThreadID = strings.TrimSpace(threadID)
	}
	if turnID, ok := upd.Payload["turn_id"].(string); ok && strings.TrimSpace(turnID) != "" {
		r.Live.TurnID = strings.TrimSpace(turnID)
	}
	if strings.TrimSpace(r.Live.SessionID) == "" && strings.TrimSpace(r.Live.ThreadID) != "" && strings.TrimSpace(r.Live.TurnID) != "" {
		r.Live.SessionID = strings.TrimSpace(r.Live.ThreadID) + "-" + strings.TrimSpace(r.Live.TurnID)
	}
	if pid := intPtrFromAny(upd.Payload["agent_pid"]); pid != nil {
		r.Live.AgentPID = pid
	} else if pid := intPtrFromAny(upd.Payload["codex_app_server_pid"]); pid != nil { // legacy
		r.Live.AgentPID = pid
	}
	if turnCount, ok := intFromAny(upd.Payload["turn_count"]); ok && turnCount > 0 {
		r.Live.TurnCount = turnCount
	}
	if m := strings.TrimSpace(extractMessage(upd.Payload)); m != "" {
		r.Live.LastAgentMessage = &m
	}

	// Token update emitted by worker after each turn.
	if upd.Event == "symphony/token_update" {
		if v, ok := intFromAny(upd.Payload["input_tokens"]); ok {
			r.Live.InputTokens = v
		}
		if v, ok := intFromAny(upd.Payload["output_tokens"]); ok {
			r.Live.OutputTokens = v
		}
		if v, ok := intFromAny(upd.Payload["total_tokens"]); ok {
			r.Live.TotalTokens = v
		}
	}

	// Best-effort usage extraction from Codex event payloads.
	applyUsage := func(usage map[string]any) {
		if usage == nil {
			return
		}
		// `thread/tokenUsage/updated` carries { tokenUsage: { total: {...}, last: {...} } }.
		// Prefer absolute totals for live display.
		if nested, ok := usage["total"].(map[string]any); ok && nested != nil {
			usage = nested
		} else if nested, ok := usage["last"].(map[string]any); ok && nested != nil {
			usage = nested
		}
		if v, ok := intFromAny(usage["input_tokens"]); ok {
			r.Live.InputTokens = v
		}
		if v, ok := intFromAny(usage["inputTokens"]); ok {
			r.Live.InputTokens = v
		}
		if v, ok := intFromAny(usage["prompt_tokens"]); ok {
			r.Live.InputTokens = v
		}
		if v, ok := intFromAny(usage["promptTokens"]); ok {
			r.Live.InputTokens = v
		}

		if v, ok := intFromAny(usage["output_tokens"]); ok {
			r.Live.OutputTokens = v
		}
		if v, ok := intFromAny(usage["outputTokens"]); ok {
			r.Live.OutputTokens = v
		}
		if v, ok := intFromAny(usage["completion_tokens"]); ok {
			r.Live.OutputTokens = v
		}
		if v, ok := intFromAny(usage["completionTokens"]); ok {
			r.Live.OutputTokens = v
		}

		if v, ok := intFromAny(usage["total_tokens"]); ok {
			r.Live.TotalTokens = v
		}
		if v, ok := intFromAny(usage["totalTokens"]); ok {
			r.Live.TotalTokens = v
		}
	}

	if usage, ok := upd.Payload["usage"].(map[string]any); ok {
		applyUsage(usage)
	}
	if usage, ok := upd.Payload["Usage"].(map[string]any); ok {
		applyUsage(usage)
	}
	if usage, ok := upd.Payload["tokenUsage"].(map[string]any); ok {
		applyUsage(usage)
	}
	if usage, ok := upd.Payload["token_usage"].(map[string]any); ok {
		applyUsage(usage)
	}

	// Some events send tokens on the top-level payload.
	applyUsage(upd.Payload)

	// Some protocols nest usage blocks (e.g. payload.response.usage, payload.metrics.tokenUsage).
	// Walk nested maps/slices and apply usage wherever we find it.
	type node struct {
		v     any
		depth int
	}
	queue := []node{{v: upd.Payload, depth: 0}}
	seen := 0
	for len(queue) > 0 && seen < 250 {
		n := queue[0]
		queue = queue[1:]
		seen++
		if n.v == nil || n.depth > 6 {
			continue
		}
		if m, ok := n.v.(map[string]any); ok {
			if usage, ok := m["usage"].(map[string]any); ok {
				applyUsage(usage)
			}
			if usage, ok := m["tokenUsage"].(map[string]any); ok {
				applyUsage(usage)
			}
			if usage, ok := m["token_usage"].(map[string]any); ok {
				applyUsage(usage)
			}
			applyUsage(m)
			for _, v := range m {
				queue = append(queue, node{v: v, depth: n.depth + 1})
			}
			continue
		}
		if xs, ok := n.v.([]any); ok {
			for _, v := range xs {
				queue = append(queue, node{v: v, depth: n.depth + 1})
			}
			continue
		}
	}
}

func (o *Orchestrator) onWorkerExit(ctx context.Context, entry *RunningEntry, res agent.Result, err error) {
	endedAt := time.Now().UTC()
	duration := endedAt.Sub(entry.StartedAt).Seconds()

	o.mu.Lock()
	cfg := o.cfg
	ws := o.workspace
	suppressRetry := entry.suppressRetry
	cleanupOnExit := entry.cleanupOnExit
	o.mu.Unlock()

	if res.WorkspacePath != "" {
		o.mu.Lock()
		entry.WorkspacePath = res.WorkspacePath
		o.mu.Unlock()
	}

	status := string(PhaseFailed)
	var errMsg *string
	if err == nil {
		status = string(PhaseSucceeded)
	} else {
		s := err.Error()
		errMsg = &s
		// Preserve reconciliation cancels / stalls for UI/history.
		if suppressRetry && entry.Phase == PhaseCanceledByReconciliation {
			status = string(PhaseCanceledByReconciliation)
		} else if entry.Phase == PhaseStalled {
			status = string(PhaseStalled)
		} else {
			if provider.IsTimeout(err) || errors.Is(err, context.DeadlineExceeded) {
				status = string(PhaseTimedOut)
			}
		}
	}

	var threadID *string
	if v := strings.TrimSpace(entry.Live.ThreadID); v != "" {
		threadID = &v
	}
	var turnID *string
	if v := strings.TrimSpace(entry.Live.TurnID); v != "" {
		turnID = &v
	}

	finalIssue := res.FinalIssue
	if strings.TrimSpace(finalIssue.ID) == "" ||
		strings.TrimSpace(finalIssue.Identifier) == "" ||
		strings.TrimSpace(finalIssue.Title) == "" ||
		strings.TrimSpace(finalIssue.State) == "" {
		finalIssue = entry.Issue
	}

	terminalAtExit := isTerminal(finalIssue.State, cfg.Tracker.TerminalStates)
	if err == nil && !suppressRetry && !terminalAtExit {
		// Queue continuation retry as early as possible to reduce races with callers that
		// observe retries immediately after dispatch.
		o.scheduleRetry(ctx, entry.IssueID, entry.Identifier, 1, "continuation", nil, 1*time.Second)
	}

	completed := CompletedEntry{
		Issue:             finalIssue,
		IssueID:           entry.IssueID,
		IssueIdentifier:   entry.Identifier,
		Attempt:           entry.Attempt,
		WorkspacePath:     strings.TrimSpace(entry.WorkspacePath),
		StartedAt:         entry.StartedAt,
		EndedAt:           endedAt,
		DurationSecs:      duration,
		Status:            status,
		Error:             errMsg,
		InputTokens:       res.InputTokens,
		OutputTokens:      res.OutputTokens,
		TotalTokens:       res.TotalTokens,
		TurnsRun:          res.TurnsRun,
		ThreadID:          threadID,
		TurnID:            turnID,
		LastAgentEvent:    entry.Live.LastAgentEvent,
		LastAgentMessage:  entry.Live.LastAgentMessage,
	}

	persistBestEffort := func(totals AgentTotals, rateLimits map[string]any, c CompletedEntry) {
		go func() {
			o.recordAttemptBestEffort(totals, rateLimits, c)
		}()
	}

	if suppressRetry {
		o.mu.Lock()
		o.completedHistory = append(o.completedHistory, completed)
		if len(o.completedHistory) > 200 {
			o.completedHistory = o.completedHistory[len(o.completedHistory)-200:]
		}
		totals := o.agentTotals
		rateLimits := o.agentRateLimits
		o.mu.Unlock()
		persistBestEffort(totals, rateLimits, completed)

		if cleanupOnExit {
			ws.RemoveBestEffort(ctx, entry.Identifier)
		}
		o.releaseClaim(entry.IssueID)
		return
	}

	if err == nil {
		o.mu.Lock()
		o.completed[entry.IssueID] = struct{}{}
		o.completedHistory = append(o.completedHistory, completed)
		if len(o.completedHistory) > 200 {
			o.completedHistory = o.completedHistory[len(o.completedHistory)-200:]
		}
		o.agentTotals.InputTokens += res.InputTokens
		o.agentTotals.OutputTokens += res.OutputTokens
		o.agentTotals.TotalTokens += res.TotalTokens
		if res.RateLimits != nil {
			o.agentRateLimits = res.RateLimits
		}
		totals := o.agentTotals
		rateLimits := o.agentRateLimits
		o.mu.Unlock()

		persistBestEffort(totals, rateLimits, completed)

		// If terminal at exit, clean + release instead of continuation retry.
		if terminalAtExit {
			ws.RemoveBestEffort(ctx, res.FinalIssue.Identifier)
			o.releaseClaim(entry.IssueID)
			return
		}

		return
	}

	o.mu.Lock()
	o.completedHistory = append(o.completedHistory, completed)
	if len(o.completedHistory) > 200 {
		o.completedHistory = o.completedHistory[len(o.completedHistory)-200:]
	}
	o.agentTotals.InputTokens += res.InputTokens
	o.agentTotals.OutputTokens += res.OutputTokens
	o.agentTotals.TotalTokens += res.TotalTokens
	if res.RateLimits != nil {
		o.agentRateLimits = res.RateLimits
	}
	totals := o.agentTotals
	rateLimits := o.agentRateLimits
	o.mu.Unlock()

	persistBestEffort(totals, rateLimits, completed)

	next := nextAttempt(entry.Attempt)
	o.scheduleRetry(ctx, entry.IssueID, entry.Identifier, next, "backoff", errMsg, backoffDelay(next, cfg.Agent.MaxRetryBackoff))
}

func (o *Orchestrator) stopRunning(issueID string, phase RunPhase, cleanupWorkspace bool) {
	o.mu.Lock()
	r := o.running[issueID]
	if r != nil {
		r.Phase = phase
		if phase == PhaseCanceledByReconciliation {
			r.suppressRetry = true
			r.cleanupOnExit = cleanupWorkspace
		}
	}
	cancel := context.CancelFunc(nil)
	if r != nil {
		cancel = r.cancel
	}
	o.mu.Unlock()
	if r == nil {
		return
	}
	if cancel != nil {
		cancel()
	}
}

func (o *Orchestrator) claim(issueID string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.claimed[issueID] = struct{}{}
}

func (o *Orchestrator) releaseClaim(issueID string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.claimed, issueID)
	if re := o.retries[issueID]; re != nil && re.timerHandle != nil {
		re.timerHandle.Stop()
		delete(o.retries, issueID)
	}
}

func issueAlreadyClaimed(o *Orchestrator, issueID string) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	_, ok := o.claimed[issueID]
	return ok
}

func (o *Orchestrator) scheduleRetry(ctx context.Context, issueID string, identifier string, attempt int, delayType string, errMsg *string, delay time.Duration) {
	timer := time.AfterFunc(delay, func() {
		o.onRetryTimer(ctx, issueID)
	})

	o.mu.Lock()
	if prev := o.retries[issueID]; prev != nil && prev.timerHandle != nil {
		prev.timerHandle.Stop()
	}
	due := time.Now().Add(delay)
	entry := &RetryEntry{
		IssueID:     issueID,
		Identifier:  identifier,
		Attempt:     attempt,
		DueAt:       due,
		Error:       errMsg,
		DelayType:   delayType,
		timerHandle: timer,
	}
	o.retries[issueID] = entry
	o.mu.Unlock()
	log.Printf("symphony retry_queued issue_id=%s identifier=%s attempt=%d due_in_ms=%d delay_type=%s", issueID, identifier, attempt, delay.Milliseconds(), delayType)
}

func (o *Orchestrator) onRetryTimer(ctx context.Context, issueID string) {
	o.mu.Lock()
	retryEntry := o.retries[issueID]
	delete(o.retries, issueID)
	tr := o.tracker
	cfg := o.cfg
	o.mu.Unlock()
	if retryEntry == nil {
		return
	}

	candidates, err := tr.FetchCandidateIssues(ctx)
	if err != nil {
		msg := "retry poll failed"
		o.scheduleRetry(ctx, issueID, retryEntry.Identifier, retryEntry.Attempt+1, "backoff", &msg, backoffDelay(retryEntry.Attempt+1, cfg.Agent.MaxRetryBackoff))
		return
	}
	var found *domain.Issue
	for i := range candidates {
		if candidates[i].ID == issueID {
			found = &candidates[i]
			break
		}
	}
	if found == nil {
		o.releaseClaim(issueID)
		return
	}

	o.mu.Lock()
	maxGlobal := o.cfg.Agent.MaxConcurrentAgents
	perState := o.cfg.Agent.MaxConcurrentAgentsByState
	activeStates := append([]string(nil), o.cfg.Tracker.ActiveStates...)
	terminalStates := append([]string(nil), o.cfg.Tracker.TerminalStates...)
	runningCount := len(o.running)
	stateCounts := map[string]int{}
	for _, r := range o.running {
		stateCounts[strings.ToLower(strings.TrimSpace(r.Issue.State))]++
	}
	rt, _ := o.runtime.Get()
	ws := o.workspace
	prov := o.provider
	newRunner := o.newRunner
	o.mu.Unlock()

	if runningCount >= maxGlobal {
		msg := "no available orchestrator slots"
		o.scheduleRetry(ctx, issueID, retryEntry.Identifier, retryEntry.Attempt+1, "backoff", &msg, backoffDelay(retryEntry.Attempt+1, cfg.Agent.MaxRetryBackoff))
		return
	}

	if !isDispatchEligible(*found, activeStates, terminalStates) || !todoBlockerRuleOK(*found, terminalStates) {
		o.releaseClaim(issueID)
		return
	}
	stateKey := strings.ToLower(strings.TrimSpace(found.State))
	stateCap := maxGlobal
	if v, ok := perState[stateKey]; ok && v > 0 {
		stateCap = v
	}
	if stateCounts[stateKey] >= stateCap {
		msg := "no available orchestrator slots"
		o.scheduleRetry(ctx, issueID, retryEntry.Identifier, retryEntry.Attempt+1, "backoff", &msg, backoffDelay(retryEntry.Attempt+1, cfg.Agent.MaxRetryBackoff))
		return
	}

	att := retryEntry.Attempt
	runner := newRunner(rt, tr, ws, prov)
	go o.runWorker(ctx, runner, *found, &att)
}

func (o *Orchestrator) snapshot() map[string]any {
	o.mu.Lock()
	defer o.mu.Unlock()
	running := make([]*RunningEntry, 0, len(o.running))
	for _, r := range o.running {
		running = append(running, r)
	}
	retrying := make([]*RetryEntry, 0, len(o.retries))
	for _, r := range o.retries {
		retrying = append(retrying, r)
	}
	completed := append([]CompletedEntry(nil), o.completedHistory...)
	return map[string]any{
		"running":      running,
		"retrying":     retrying,
		"completed":    completed,
		"agent_totals": o.agentTotals,
		"rate_limits":  o.agentRateLimits,
	}
}

func sortForDispatch(issues []domain.Issue) {
	sort.SliceStable(issues, func(i, j int) bool {
		pi := issues[i].Priority
		pj := issues[j].Priority
		if pi == nil && pj != nil {
			return false
		}
		if pi != nil && pj == nil {
			return true
		}
		if pi != nil && pj != nil && *pi != *pj {
			return *pi < *pj
		}

		ti := issues[i].CreatedAt
		tj := issues[j].CreatedAt
		if ti == nil && tj != nil {
			return false
		}
		if ti != nil && tj == nil {
			return true
		}
		if ti != nil && tj != nil && !ti.Equal(*tj) {
			return ti.Before(*tj)
		}
		return issues[i].Identifier < issues[j].Identifier
	})
}

func isDispatchEligible(issue domain.Issue, activeStates, terminalStates []string) bool {
	if issue.ID == "" || issue.Identifier == "" || issue.Title == "" || issue.State == "" {
		return false
	}
	if isTerminal(issue.State, terminalStates) {
		return false
	}
	return isActiveState(issue.State, activeStates)
}

func isActiveState(state string, activeStates []string) bool {
	s := strings.ToLower(strings.TrimSpace(state))
	for _, a := range activeStates {
		if strings.ToLower(strings.TrimSpace(a)) == s {
			return true
		}
	}
	return false
}

func isTerminal(state string, terminalStates []string) bool {
	s := strings.ToLower(strings.TrimSpace(state))
	for _, t := range terminalStates {
		if strings.ToLower(strings.TrimSpace(t)) == s {
			return true
		}
	}
	return false
}

func todoBlockerRuleOK(issue domain.Issue, terminalStates []string) bool {
	if strings.ToLower(strings.TrimSpace(issue.State)) != "todo" {
		return true
	}
	for _, b := range issue.BlockedBy {
		if b.State == nil {
			return false
		}
		if !isTerminal(*b.State, terminalStates) {
			return false
		}
	}
	return true
}

func nextAttempt(prev *int) int {
	if prev == nil {
		return 1
	}
	return *prev + 1
}

func backoffDelay(attempt int, cap time.Duration) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	delay := 10 * time.Second
	for i := 1; i < attempt; i++ {
		delay *= 2
		if cap > 0 && delay > cap {
			return cap
		}
	}
	if cap > 0 && delay > cap {
		return cap
	}
	return delay
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func normalizePhase(v string) (RunPhase, bool) {
	switch strings.TrimSpace(v) {
	case string(PhasePreparingWorkspace):
		return PhasePreparingWorkspace, true
	case string(PhaseBuildingPrompt):
		return PhaseBuildingPrompt, true
	case string(PhaseLaunchingAgentProcess):
		return PhaseLaunchingAgentProcess, true
	case string(PhaseInitializingSession):
		return PhaseInitializingSession, true
	case string(PhaseStreamingTurn):
		return PhaseStreamingTurn, true
	case string(PhaseFinishing):
		return PhaseFinishing, true
	case string(PhaseSucceeded):
		return PhaseSucceeded, true
	case string(PhaseFailed):
		return PhaseFailed, true
	case string(PhaseTimedOut):
		return PhaseTimedOut, true
	case string(PhaseStalled):
		return PhaseStalled, true
	case string(PhaseCanceledByReconciliation):
		return PhaseCanceledByReconciliation, true
	default:
		return "", false
	}
}

func intFromAny(v any) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case int32:
		return int(t), true
	case int64:
		return int(t), true
	case float64:
		return int(t), true
	case json.Number:
		if n, err := t.Int64(); err == nil {
			return int(n), true
		}
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return 0, false
		}
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return int(n), true
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return int(f), true
		}
	}
	return 0, false
}

func intPtrFromAny(v any) *int {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case *int:
		return t
	case *int32:
		n := int(*t)
		return &n
	case *int64:
		n := int(*t)
		return &n
	case int:
		return &t
	case int32:
		n := int(t)
		return &n
	case int64:
		n := int(t)
		return &n
	case float64:
		n := int(t)
		return &n
	case json.Number:
		if n64, err := t.Int64(); err == nil {
			n := int(n64)
			return &n
		}
	}
	return nil
}

func attemptValue(a *int) any {
	if a == nil {
		return nil
	}
	return *a
}
