package workflow

import (
	"context"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Manager struct {
	path string

	mu       sync.RWMutex
	def      *Definition
	lastErr  error
	revision int64
}

func NewManager(path string) *Manager {
	return &Manager{path: path}
}

func (m *Manager) Path() string {
	return m.path
}

func (m *Manager) Get() (*Definition, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.def == nil {
		return nil, m.revision, m.lastErr
	}
	return m.def, m.revision, m.lastErr
}

func (m *Manager) LoadOnce() (*Definition, error) {
	def, err := Load(m.path)
	m.mu.Lock()
	defer m.mu.Unlock()
	if err != nil {
		m.lastErr = err
		return nil, err
	}
	m.def = def
	m.lastErr = nil
	m.revision++
	return def, nil
}

func (m *Manager) Watch(ctx context.Context) error {
	if _, err := m.LoadOnce(); err != nil {
		return err
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()

	dir := filepath.Dir(m.path)
	if err := w.Add(dir); err != nil {
		return err
	}

	// Coalesce bursty file events.
	var (
		pending bool
		timer   *time.Timer
	)
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	reload := func() {
		if _, err := m.LoadOnce(); err != nil {
			log.Printf("symphony workflow_reload status=failed error=%v", err)
			return
		}
		log.Printf("symphony workflow_reload status=ok path=%s", m.path)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-w.Errors:
			if err != nil {
				log.Printf("symphony workflow_watch error=%v", err)
			}
		case ev := <-w.Events:
			if ev.Name != m.path {
				// Some editors do atomic rename; ensure we reload when file base name matches.
				if filepath.Base(ev.Name) != filepath.Base(m.path) {
					continue
				}
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			if timer == nil {
				timer = time.NewTimer(150 * time.Millisecond)
				pending = true
				continue
			}
			if !pending {
				timer.Reset(150 * time.Millisecond)
				pending = true
			}
		case <-func() <-chan time.Time {
			if timer == nil {
				return nil
			}
			return timer.C
		}():
			if timer != nil {
				pending = false
				reload()
			}
		}
	}
}
