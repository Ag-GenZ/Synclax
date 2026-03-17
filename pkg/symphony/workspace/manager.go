package workspace

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Workspace struct {
	Path         string
	WorkspaceKey string
	CreatedNow   bool
}

type HookScripts struct {
	AfterCreate  string
	BeforeRun    string
	AfterRun     string
	BeforeRemove string
	Timeout      time.Duration
}

type Manager struct {
	rootAbs string
	hooks   HookScripts
}

var (
	ErrInvalidWorkspaceCwd = errors.New("invalid_workspace_cwd")
)

func NewManager(root string, hooks HookScripts) (*Manager, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("workspace root is required")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &Manager{rootAbs: rootAbs, hooks: hooks}, nil
}

func (m *Manager) Root() string { return m.rootAbs }

func (m *Manager) CreateForIssue(ctx context.Context, issueIdentifier string) (Workspace, error) {
	key := sanitizeWorkspaceKey(issueIdentifier)
	workspacePath := filepath.Join(m.rootAbs, key)

	if err := ensureInsideRoot(m.rootAbs, workspacePath); err != nil {
		return Workspace{}, err
	}

	if err := os.MkdirAll(m.rootAbs, 0o755); err != nil {
		return Workspace{}, err
	}

	createdNow := false
	stat, err := os.Stat(workspacePath)
	switch {
	case err == nil && stat.IsDir():
		createdNow = false
	case err == nil && !stat.IsDir():
		return Workspace{}, fmt.Errorf("workspace path exists and is not a directory: %s", workspacePath)
	case os.IsNotExist(err):
		if err := os.MkdirAll(workspacePath, 0o755); err != nil {
			return Workspace{}, err
		}
		createdNow = true
	default:
		return Workspace{}, err
	}

	ws := Workspace{Path: workspacePath, WorkspaceKey: key, CreatedNow: createdNow}
	if err := m.prepareWorkspace(ws.Path); err != nil {
		return Workspace{}, err
	}

	if ws.CreatedNow && strings.TrimSpace(m.hooks.AfterCreate) != "" {
		if err := runHook(ctx, ws.Path, "after_create", m.hooks.AfterCreate, m.hooks.Timeout, true); err != nil {
			_ = os.RemoveAll(ws.Path)
			return Workspace{}, err
		}
	}
	return ws, nil
}

func (m *Manager) BeforeRun(ctx context.Context, ws Workspace) error {
	if strings.TrimSpace(m.hooks.BeforeRun) == "" {
		return nil
	}
	return runHook(ctx, ws.Path, "before_run", m.hooks.BeforeRun, m.hooks.Timeout, true)
}

func (m *Manager) AfterRunBestEffort(ctx context.Context, ws Workspace) {
	if strings.TrimSpace(m.hooks.AfterRun) == "" {
		return
	}
	if err := runHook(ctx, ws.Path, "after_run", m.hooks.AfterRun, m.hooks.Timeout, false); err != nil {
		log.Printf("symphony hook status=ignored name=after_run error=%v", err)
	}
}

func (m *Manager) RemoveBestEffort(ctx context.Context, issueIdentifier string) {
	key := sanitizeWorkspaceKey(issueIdentifier)
	workspacePath := filepath.Join(m.rootAbs, key)
	if err := ensureInsideRoot(m.rootAbs, workspacePath); err != nil {
		log.Printf("symphony workspace_remove status=failed error=%v", err)
		return
	}

	stat, err := os.Stat(workspacePath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Printf("symphony workspace_remove status=failed path=%s error=%v", workspacePath, err)
		return
	}
	if !stat.IsDir() {
		log.Printf("symphony workspace_remove status=failed path=%s error=not a directory", workspacePath)
		return
	}

	if strings.TrimSpace(m.hooks.BeforeRemove) != "" {
		if err := runHook(ctx, workspacePath, "before_remove", m.hooks.BeforeRemove, m.hooks.Timeout, false); err != nil {
			log.Printf("symphony hook status=ignored name=before_remove error=%v", err)
		}
	}

	if err := os.RemoveAll(workspacePath); err != nil {
		log.Printf("symphony workspace_remove status=failed path=%s error=%v", workspacePath, err)
		return
	}
	log.Printf("symphony workspace_remove status=ok path=%s", workspacePath)
}

func (m *Manager) prepareWorkspace(path string) error {
	// Remove common temp artifacts per spec test matrix.
	for _, name := range []string{"tmp", ".elixir_ls"} {
		p := filepath.Join(path, name)
		if _, err := os.Stat(p); err == nil {
			_ = os.RemoveAll(p)
		}
	}
	return nil
}

func ensureInsideRoot(rootAbs, workspacePath string) error {
	abs, err := filepath.Abs(workspacePath)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(rootAbs, abs)
	if err != nil {
		return err
	}
	if rel == "." {
		return nil
	}
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return ErrInvalidWorkspaceCwd
	}
	return nil
}

func sanitizeWorkspaceKey(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return "_"
	}
	var b strings.Builder
	b.Grow(len(identifier))
	for _, r := range identifier {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.' || r == '_' || r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

func runHook(ctx context.Context, cwd, name, script string, timeout time.Duration, fatal bool) error {
	script = strings.TrimSpace(script)
	if script == "" {
		return nil
	}
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	hctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	log.Printf("symphony hook status=starting name=%s cwd=%s", name, cwd)
	cmd := exec.CommandContext(hctx, "bash", "-lc", script)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if hctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("hook timeout name=%s", name)
	}
	if err != nil {
		if fatal {
			return fmt.Errorf("hook failed name=%s error=%w output=%s", name, err, strings.TrimSpace(string(out)))
		}
		return fmt.Errorf("hook failed name=%s error=%w output=%s", name, err, strings.TrimSpace(string(out)))
	}
	log.Printf("symphony hook status=ok name=%s", name)
	return nil
}
