package control

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"time"

	anclaxapp "github.com/cloudcarver/anclax/pkg/app"
	"github.com/wibus-wee/synclax/pkg/config"
	"github.com/wibus-wee/synclax/pkg/symphony/orchestrator"
	"github.com/wibus-wee/synclax/pkg/zcore/model"
)

type Health struct {
	Running      bool
	WorkflowPath string
	LastError    error
	HTTPPort     *int
}

type Manager struct {
	mu sync.Mutex

	workflowPath string
	httpPort     *int

	startedAt time.Time
	stats     orchestrator.StatsStore

	orch   *orchestrator.Orchestrator
	cancel context.CancelFunc
	done   chan struct{}

	lastErr error
}

func NewManager(anclaxApp *anclaxapp.Application, cfg *config.Config, mdl model.ModelInterface) (*Manager, error) {
	if cfg == nil {
		return nil, errors.New("nil config")
	}

	workflowPath := strings.TrimSpace(cfg.Symphony.WorkflowPath)
	if workflowPath == "" {
		workflowPath = "WORKFLOW.md"
	}
	if !filepath.IsAbs(workflowPath) {
		if abs, err := filepath.Abs(workflowPath); err == nil {
			workflowPath = abs
		}
	}

	mgr := &Manager{
		workflowPath: workflowPath,
		httpPort:     cfg.Symphony.HTTPPort,
		startedAt:    time.Now(),
	}
	if mdl != nil {
		// best-effort DB-backed stats persistence
		mgr.stats = NewDBStatsStore(mdl)
	}

	if anclaxApp != nil {
		anclaxApp.GetCloserManager().Register(func(ctx context.Context) error {
			return mgr.Stop(ctx)
		})
	}

	return mgr, nil
}

func (m *Manager) Health() Health {
	m.mu.Lock()
	defer m.mu.Unlock()
	running := m.orch != nil && m.cancel != nil && m.done != nil
	var port *int
	if running && m.httpPort != nil && *m.httpPort >= 0 {
		port = m.httpPort
	}
	return Health{
		Running:      running,
		WorkflowPath: m.workflowPath,
		LastError:    m.lastErr,
		HTTPPort:     port,
	}
}

func (m *Manager) Start(ctx context.Context, workflowPath string, httpPort *int) error {
	if ctx == nil {
		ctx = context.Background()
	}
	workflowPath = strings.TrimSpace(workflowPath)
	if workflowPath != "" && !filepath.IsAbs(workflowPath) {
		if abs, err := filepath.Abs(workflowPath); err == nil {
			workflowPath = abs
		}
	}

	m.mu.Lock()
	if m.orch != nil {
		// Idempotent start; allow setting params only when unchanged.
		if workflowPath != "" && workflowPath != m.workflowPath {
			m.mu.Unlock()
			return errors.New("symphony already running with a different workflow_path")
		}
		if httpPort != nil && (m.httpPort == nil || *httpPort != *m.httpPort) {
			m.mu.Unlock()
			return errors.New("symphony already running with a different http_port")
		}
		m.mu.Unlock()
		return nil
	}

	if workflowPath != "" {
		m.workflowPath = workflowPath
	}
	if httpPort != nil {
		m.httpPort = httpPort
	}

	orch, err := orchestrator.New(orchestrator.Options{
		WorkflowPath: m.workflowPath,
		StatsStore:   m.stats,
	})
	if err != nil {
		m.mu.Unlock()
		return err
	}

	runCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	m.orch = orch
	m.cancel = cancel
	m.done = done
	m.lastErr = nil
	m.mu.Unlock()

	go func() {
		err := orch.Run(runCtx, m.httpPort)

		m.mu.Lock()
		m.lastErr = err
		m.orch = nil
		m.cancel = nil
		close(done)
		m.done = nil
		m.mu.Unlock()
	}()

	return nil
}

func (m *Manager) Stop(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	m.mu.Lock()
	cancel := m.cancel
	done := m.done
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

func (m *Manager) Snapshot() map[string]any {
	m.mu.Lock()
	orch := m.orch
	startedAt := m.startedAt
	stats := m.stats
	m.mu.Unlock()

	uptime := 0.0
	if !startedAt.IsZero() {
		uptime = time.Since(startedAt).Seconds()
	}

	if orch == nil {
		totals := orchestrator.CodexTotals{SecondsRunning: uptime}
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
			"codex_totals": totals,
			"rate_limits":  rateLimits,
		}
	}

	snap := orch.Snapshot()
	if v, ok := snap["codex_totals"].(orchestrator.CodexTotals); ok {
		v.SecondsRunning = uptime
		snap["codex_totals"] = v
	} else if v, ok := snap["codex_totals"].(*orchestrator.CodexTotals); ok && v != nil {
		v.SecondsRunning = uptime
	}
	return snap
}
