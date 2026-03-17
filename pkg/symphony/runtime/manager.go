package runtime

import (
	"context"
	"log"
	"sync"

	"github.com/wibus-wee/synclax/pkg/symphony/workflow"
)

type Manager struct {
	workflow *workflow.Manager

	mu       sync.RWMutex
	lastGood *EffectiveRuntime
	revision int64
}

func NewManager(workflowPath string) *Manager {
	return &Manager{workflow: workflow.NewManager(workflowPath)}
}

func (m *Manager) WorkflowPath() string { return m.workflow.Path() }

func (m *Manager) Start(ctx context.Context) error {
	if _, err := m.workflow.LoadOnce(); err != nil {
		return err
	}
	if err := m.refreshLocked(); err != nil {
		return err
	}

	go func() {
		if err := m.workflow.Watch(ctx); err != nil {
			log.Printf("symphony workflow_watch status=exited error=%v", err)
		}
	}()
	return nil
}

func (m *Manager) Get() (*EffectiveRuntime, int64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastGood, m.revision
}

func (m *Manager) RefreshIfNeeded() {
	def, rev, lastErr := m.workflow.Get()
	if lastErr != nil {
		log.Printf("symphony workflow_reload status=failed error=%v", lastErr)
	}
	if def == nil {
		return
	}

	m.mu.RLock()
	curRev := m.revision
	m.mu.RUnlock()
	if rev == curRev {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	// Double-check under lock.
	if rev == m.revision {
		return
	}
	if err := m.refreshLocked(); err != nil {
		log.Printf("symphony runtime_refresh status=failed error=%v", err)
		return
	}
}

func (m *Manager) refreshLocked() error {
	def, rev, err := m.workflow.Get()
	if err != nil {
		// Keep last good runtime on reload error, but fail startup when there is no runtime.
		if m.lastGood == nil {
			return err
		}
		return nil
	}
	if def == nil {
		return ErrWorkflowInvalid
	}
	rt, err := Build(def)
	if err != nil {
		// Keep last good runtime on invalid workflow, but fail startup when there is no runtime.
		if m.lastGood == nil {
			return err
		}
		return nil
	}
	m.lastGood = rt
	m.revision = rev
	return nil
}
