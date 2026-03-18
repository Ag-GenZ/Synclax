package control

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	anclaxapp "github.com/cloudcarver/anclax/pkg/app"
	"github.com/wibus-wee/synclax/pkg/config"
	"github.com/wibus-wee/synclax/pkg/symphony/orchestrator"
	"github.com/wibus-wee/synclax/pkg/zcore/model"
)

type WorkflowHealth struct {
	ID           string
	WorkflowPath string
	Running      bool
	LastError    error
	HTTPPort     *int
}

type Health struct {
	// Running is whether any workflow is running.
	Running bool

	ActiveWorkflowID string
	WorkflowPath     string
	LastError        error
	HTTPPort         *int

	Workflows []WorkflowHealth
}

type Manager struct {
	mu sync.Mutex

	defaultHTTPPort *int

	workflows map[string]*workflowEntry
	order     []string
	activeID  string
}

func NewManager(anclaxApp *anclaxapp.Application, cfg *config.Config, mdl model.ModelInterface) (*Manager, error) {
	if cfg == nil {
		return nil, errors.New("nil config")
	}

	paths := make([]string, 0, 4)
	for _, p := range cfg.Symphony.WorkflowPaths {
		if strings.TrimSpace(p) != "" {
			paths = append(paths, p)
		}
	}
	if len(paths) == 0 {
		if strings.TrimSpace(cfg.Symphony.WorkflowPath) != "" {
			paths = append(paths, cfg.Symphony.WorkflowPath)
		} else {
			paths = append(paths, "WORKFLOW.md")
		}
	}

	m := &Manager{
		defaultHTTPPort: cfg.Symphony.HTTPPort,
		workflows:       map[string]*workflowEntry{},
	}

	now := time.Now()
	for _, p := range paths {
		abs := normalizeWorkflowPath(p)
		id := workflowIDFromPath(abs)
		if _, exists := m.workflows[id]; exists {
			// On collision, keep the first and mint a deterministic alternative.
			id = workflowIDFromPath(fmt.Sprintf("%s#%d", abs, len(m.order)+1))
		}

		m.workflows[id] = &workflowEntry{
			id:           id,
			configured:   true,
			workflowPath: abs,
			startedAt:    now,
			stats:        NewDBStatsStore(mdl, id),
		}
		m.order = append(m.order, id)
	}
	if len(m.order) > 0 {
		m.activeID = m.order[0]
	}

	if anclaxApp != nil {
		anclaxApp.GetCloserManager().Register(func(ctx context.Context) error {
			return m.Stop(ctx, "")
		})
	}

	return m, nil
}

func (m *Manager) Health() Health {
	m.mu.Lock()
	defer m.mu.Unlock()

	workflows := make([]WorkflowHealth, 0, len(m.workflows))
	anyRunning := false
	for _, id := range m.order {
		w := m.workflows[id]
		if w == nil {
			continue
		}
		running := w.orch != nil && w.cancel != nil && w.done != nil
		if running {
			anyRunning = true
		}
		var port *int
		if running && w.httpPort != nil && *w.httpPort >= 0 {
			port = w.httpPort
		}
		workflows = append(workflows, WorkflowHealth{
			ID:           w.id,
			WorkflowPath: w.workflowPath,
			Running:      running,
			LastError:    w.lastErr,
			HTTPPort:     port,
		})
	}

	active := m.workflows[m.activeID]
	activeRunning := active != nil && active.orch != nil && active.cancel != nil && active.done != nil
	var activePort *int
	if activeRunning && active.httpPort != nil && *active.httpPort >= 0 {
		activePort = active.httpPort
	}

	return Health{
		Running:          anyRunning,
		ActiveWorkflowID: m.activeID,
		WorkflowPath:     safeWorkflowPath(active),
		LastError:        safeLastErr(active),
		HTTPPort:         activePort,
		Workflows:        workflows,
	}
}

func (m *Manager) Workflows() []WorkflowHealth {
	h := m.Health()
	return h.Workflows
}

func (m *Manager) Start(ctx context.Context, workflowID string, workflowPath string, httpPort *int) error {
	if ctx == nil {
		ctx = context.Background()
	}

	workflowID = strings.TrimSpace(workflowID)
	workflowPath = strings.TrimSpace(workflowPath)

	if workflowID == "" && workflowPath == "" {
		return m.startAll(ctx)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	entry, err := m.resolveWorkflowLocked(workflowID, workflowPath)
	if err != nil {
		return err
	}
	if entry.startedAt.IsZero() {
		entry.startedAt = time.Now()
	}

	// Idempotent start; allow setting params only when unchanged.
	if entry.orch != nil {
		if workflowPath != "" && normalizeWorkflowPath(workflowPath) != entry.workflowPath {
			return errors.New("symphony already running with a different workflow_path")
		}
		if httpPort != nil && (entry.httpPort == nil || *httpPort != *entry.httpPort) {
			return errors.New("symphony already running with a different http_port")
		}
		m.activeID = entry.id
		return nil
	}

	if workflowPath != "" {
		entry.workflowPath = normalizeWorkflowPath(workflowPath)
	}
	if httpPort != nil {
		entry.httpPort = httpPort
	} else if entry.httpPort == nil {
		// For dynamically-added workflows, default to disabling the debug server
		// unless explicitly requested (avoids port collisions).
		if !entry.configured && m.defaultHTTPPort != nil {
			entry.httpPort = intPtr(-1)
		} else {
			entry.httpPort = m.defaultHTTPPort
		}
	}

	orch, err := orchestrator.New(orchestrator.Options{
		WorkflowPath: entry.workflowPath,
		StatsStore:   entry.stats,
	})
	if err != nil {
		return err
	}

	runCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	entry.orch = orch
	entry.cancel = cancel
	entry.done = done
	entry.lastErr = nil
	m.activeID = entry.id

	go func() {
		err := orch.Run(runCtx, entry.httpPort)

		m.mu.Lock()
		entry.lastErr = err
		entry.orch = nil
		entry.cancel = nil
		close(done)
		entry.done = nil
		m.mu.Unlock()
	}()

	return nil
}

func (m *Manager) Stop(ctx context.Context, workflowID string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	workflowID = strings.TrimSpace(workflowID)
	if workflowID == "" {
		return m.stopAll(ctx)
	}

	m.mu.Lock()
	entry := m.workflows[workflowID]
	cancel := context.CancelFunc(nil)
	done := (<-chan struct{})(nil)
	if entry != nil {
		cancel = entry.cancel
		done = entry.done
	}
	m.mu.Unlock()

	if cancel == nil || done == nil {
		return nil
	}

	cancel()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (m *Manager) Snapshot(workflowID string) (map[string]any, error) {
	workflowID = strings.TrimSpace(workflowID)

	m.mu.Lock()
	if workflowID == "" {
		workflowID = m.activeID
	}
	if workflowID == "" && len(m.order) > 0 {
		workflowID = m.order[0]
	}
	entry := m.workflows[workflowID]
	orch := (*orchestrator.Orchestrator)(nil)
	startedAt := time.Time{}
	stats := orchestrator.StatsStore(nil)
	if entry != nil {
		orch = entry.orch
		startedAt = entry.startedAt
		stats = entry.stats
	}
	m.mu.Unlock()

	if entry == nil {
		return nil, errors.New("unknown workflow_id")
	}

	uptime := 0.0
	if !startedAt.IsZero() {
		uptime = time.Since(startedAt).Seconds()
	}

	if orch == nil {
		totals := orchestrator.AgentTotals{SecondsRunning: uptime}
		completed := []orchestrator.CompletedEntry{}
		rateLimits := map[string]any{}

		if stats != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()
			if st, rl, attempts, err := stats.Load(ctx, 200); err == nil {
				st.SecondsRunning = uptime
				totals = st
				if rl != nil {
					rateLimits = rl
				}
				if attempts != nil {
					completed = attempts
				}
			}
		}

		return map[string]any{
			"running":      []any{},
			"retrying":     []any{},
			"completed":    completed,
			"agent_totals": totals,
			"rate_limits":  rateLimits,
		}, nil
	}

	snap := orch.Snapshot()
	if v, ok := snap["agent_totals"].(orchestrator.AgentTotals); ok {
		v.SecondsRunning = uptime
		snap["agent_totals"] = v
	} else if v, ok := snap["agent_totals"].(*orchestrator.AgentTotals); ok && v != nil {
		v.SecondsRunning = uptime
	} else if v, ok := snap["agent_totals"].(map[string]any); ok && v != nil {
		v["seconds_running"] = uptime
		snap["agent_totals"] = v
	}

	return snap, nil
}

type workflowEntry struct {
	id           string
	workflowPath string
	httpPort     *int
	configured   bool

	startedAt time.Time
	stats     orchestrator.StatsStore

	orch   *orchestrator.Orchestrator
	cancel context.CancelFunc
	done   chan struct{}

	lastErr error
}

func (m *Manager) stopAll(ctx context.Context) error {
	m.mu.Lock()
	type stopTarget struct {
		cancel context.CancelFunc
		done   <-chan struct{}
	}
	targets := make([]stopTarget, 0, len(m.workflows))
	for _, e := range m.workflows {
		if e == nil || e.cancel == nil || e.done == nil {
			continue
		}
		targets = append(targets, stopTarget{cancel: e.cancel, done: e.done})
	}
	m.mu.Unlock()

	for _, t := range targets {
		t.cancel()
	}
	for _, t := range targets {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.done:
		}
	}
	return nil
}

func (m *Manager) startAll(ctx context.Context) error {
	m.mu.Lock()
	ids := append([]string(nil), m.order...)
	firstID := ""
	if len(ids) > 0 {
		firstID = ids[0]
	}
	// If multiple workflows are configured and a default debug port is set, only
	// keep it enabled for the first workflow by default to avoid collisions.
	if m.defaultHTTPPort != nil && len(ids) > 1 {
		for i, id := range ids {
			if i == 0 {
				continue
			}
			entry := m.workflows[id]
			if entry == nil || !entry.configured {
				continue
			}
			if entry.httpPort == nil {
				entry.httpPort = intPtr(-1)
			}
		}
	}
	m.mu.Unlock()

	if len(ids) == 0 {
		return errors.New("no workflows configured")
	}
	for _, id := range ids {
		if err := m.Start(ctx, id, "", nil); err != nil {
			return err
		}
	}
	if firstID != "" {
		m.mu.Lock()
		m.activeID = firstID
		m.mu.Unlock()
	}
	return nil
}

func (m *Manager) resolveWorkflowLocked(workflowID string, workflowPath string) (*workflowEntry, error) {
	if workflowID != "" {
		entry := m.workflows[workflowID]
		if entry == nil {
			return nil, errors.New("unknown workflow_id")
		}
		if workflowPath != "" && normalizeWorkflowPath(workflowPath) != entry.workflowPath {
			return nil, errors.New("workflow_id does not match workflow_path")
		}
		return entry, nil
	}

	if workflowPath == "" {
		return nil, errors.New("workflow_id or workflow_path is required")
	}
	workflowPath = normalizeWorkflowPath(workflowPath)
	id := workflowIDFromPath(workflowPath)
	entry := m.workflows[id]
	if entry == nil {
		entry = &workflowEntry{
			id:           id,
			workflowPath: workflowPath,
			startedAt:    time.Now(),
		}
		m.workflows[id] = entry
		m.order = append(m.order, id)
	}
	if entry.stats == nil {
		// Dynamically added workflows run without DB-backed persistence by default.
		entry.stats = nil
	}
	return entry, nil
}

func normalizeWorkflowPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if !filepath.IsAbs(p) {
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		}
	}
	return filepath.Clean(p)
}

func workflowIDFromPath(absPath string) string {
	absPath = normalizeWorkflowPath(absPath)
	base := strings.TrimSpace(filepath.Base(absPath))
	if base == "" {
		base = "workflow"
	}

	sum := sha1.Sum([]byte(absPath))
	// 8 hex chars is enough for uniqueness within one process.
	hash := fmt.Sprintf("%x", sum[:4])

	slug := slugify(base)
	if slug == "" {
		slug = "workflow"
	}
	return fmt.Sprintf("%s-%s", slug, hash)
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	b.Grow(len(s))
	lastDash := false
	for _, r := range s {
		isAlnum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlnum {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	return out
}

func intPtr(v int) *int { return &v }

func safeWorkflowPath(e *workflowEntry) string {
	if e == nil {
		return ""
	}
	return e.workflowPath
}

func safeLastErr(e *workflowEntry) error {
	if e == nil {
		return nil
	}
	return e.lastErr
}
