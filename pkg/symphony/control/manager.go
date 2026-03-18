package control

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"

	anclaxapp "github.com/cloudcarver/anclax/pkg/app"
	"github.com/wibus-wee/synclax/pkg/config"
	"github.com/wibus-wee/synclax/pkg/symphony/orchestrator"
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

	orch   *orchestrator.Orchestrator
	cancel context.CancelFunc
	done   chan struct{}

	lastErr error
}

func NewManager(anclaxApp *anclaxapp.Application, cfg *config.Config) (*Manager, error) {
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

	m := &Manager{
		workflowPath: workflowPath,
		httpPort:     cfg.Symphony.HTTPPort,
	}

	if anclaxApp != nil {
		anclaxApp.GetCloserManager().Register(func(ctx context.Context) error {
			return m.Stop(ctx)
		})
	}

	return m, nil
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
	m.mu.Unlock()
	if orch == nil {
		return map[string]any{
			"running":      []any{},
			"retrying":     []any{},
			"completed":    []any{},
			"codex_totals": map[string]any{"input_tokens": 0, "output_tokens": 0, "total_tokens": 0, "seconds_running": 0.0},
			"rate_limits":  map[string]any{},
		}
	}
	return orch.Snapshot()
}
