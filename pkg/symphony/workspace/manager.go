package workspace

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wibus-wee/synclax/pkg/symphony/ssh"
)

type Workspace struct {
	Path         string
	WorkspaceKey string
	CreatedNow   bool
	WorkerHost   *string // nil = local, non-nil = SSH target
}

type HookScripts struct {
	AfterCreate  string
	BeforeRun    string
	AfterRun     string
	BeforeRemove string
	Timeout      time.Duration
}

type Manager struct {
	rootAbs   string // expanded absolute path for local operations
	rootRaw   string // original value from config, used for remote paths (preserves ~)
	rootCanon string
	hooks     HookScripts

	mu sync.Mutex
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
	return &Manager{rootAbs: rootAbs, rootRaw: root, rootCanon: "", hooks: hooks}, nil
}

func (m *Manager) Root() string { return m.rootAbs }

func (m *Manager) CreateForIssue(ctx context.Context, issueIdentifier string, workerHost *string) (Workspace, error) {
	key := sanitizeWorkspaceKey(issueIdentifier)
	workspacePath := filepath.Join(m.rootAbs, key)

	if workerHost != nil && strings.TrimSpace(*workerHost) != "" {
		// Use the raw (unexpanded) root so that ~ is expanded by the remote shell,
		// not by the local process (which may be running in a container with a
		// different home directory than the SSH target).
		remotePath := m.rootRaw + "/" + key
		return m.createRemote(ctx, key, remotePath, workerHost)
	}

	if err := m.ensureRootReady(); err != nil {
		return Workspace{}, err
	}
	if err := ensureInsideRoot(m.rootAbs, workspacePath); err != nil {
		return Workspace{}, err
	}

	createdNow := false
	stat, err := os.Lstat(workspacePath)
	switch {
	case err == nil && stat.Mode()&os.ModeSymlink != 0:
		return Workspace{}, fmt.Errorf("%w: workspace path is a symlink (path=%s)", ErrInvalidWorkspaceCwd, workspacePath)
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

	canonicalPath, err := m.canonicalizeSafeWorkspacePath(workspacePath)
	if err != nil {
		return Workspace{}, err
	}

	ws := Workspace{Path: workspacePath, WorkspaceKey: key, CreatedNow: createdNow}
	ws.Path = canonicalPath
	ws.WorkerHost = nil
	if err := m.prepareWorkspace(ws.Path); err != nil {
		return Workspace{}, err
	}

	if ws.CreatedNow && strings.TrimSpace(m.hooks.AfterCreate) != "" {
		if err := runHookLocal(ctx, ws.Path, "after_create", m.hooks.AfterCreate, m.hooks.Timeout, true); err != nil {
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
	return m.runHook(ctx, ws, "before_run", m.hooks.BeforeRun, true)
}

func (m *Manager) AfterRunBestEffort(ctx context.Context, ws Workspace) {
	if strings.TrimSpace(m.hooks.AfterRun) == "" {
		return
	}
	if err := m.runHook(ctx, ws, "after_run", m.hooks.AfterRun, false); err != nil {
		log.Printf("symphony hook status=ignored name=after_run error=%v", err)
	}
}

func (m *Manager) RemoveBestEffort(ctx context.Context, issueIdentifier string, workerHost *string) {
	key := sanitizeWorkspaceKey(issueIdentifier)
	workspacePath := m.rootRaw + "/" + key

	if workerHost != nil && strings.TrimSpace(*workerHost) != "" {
		target := ssh.ParseTarget(*workerHost)
		_, _, err := ssh.Run(ctx, target, "rm -rf "+bashSingleQuote(workspacePath))
		if err != nil {
			log.Printf("symphony workspace_remove status=failed worker_host=%s path=%s error=%v", *workerHost, workspacePath, err)
			return
		}
		log.Printf("symphony workspace_remove status=ok worker_host=%s path=%s", *workerHost, workspacePath)
		return
	}

	if err := m.ensureRootReady(); err != nil {
		log.Printf("symphony workspace_remove status=failed error=%v", err)
		return
	}
	if err := ensureInsideRoot(m.rootAbs, workspacePath); err != nil {
		log.Printf("symphony workspace_remove status=failed error=%v", err)
		return
	}

	stat, err := os.Lstat(workspacePath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Printf("symphony workspace_remove status=failed path=%s error=%v", workspacePath, err)
		return
	}
	if stat.Mode()&os.ModeSymlink != 0 {
		log.Printf("symphony workspace_remove status=failed path=%s error=symlink", workspacePath)
		return
	}
	if !stat.IsDir() {
		log.Printf("symphony workspace_remove status=failed path=%s error=not a directory", workspacePath)
		return
	}

	if strings.TrimSpace(m.hooks.BeforeRemove) != "" {
		if err := runHookLocal(ctx, workspacePath, "before_remove", m.hooks.BeforeRemove, m.hooks.Timeout, false); err != nil {
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
	key := b.String()
	switch key {
	case "", ".", "..":
		return "_"
	}
	return key
}

func runHookLocal(ctx context.Context, cwd, name, script string, timeout time.Duration, fatal bool) error {
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

func (m *Manager) runHook(ctx context.Context, ws Workspace, name, script string, fatal bool) error {
	if ws.WorkerHost == nil || strings.TrimSpace(*ws.WorkerHost) == "" {
		return runHookLocal(ctx, ws.Path, name, script, m.hooks.Timeout, fatal)
	}

	timeout := m.hooks.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	hctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	target := ssh.ParseTarget(*ws.WorkerHost)
	log.Printf("symphony hook status=starting name=%s cwd=%s worker_host=%s", name, ws.Path, *ws.WorkerHost)
	out, _, err := ssh.Run(hctx, target, "cd "+bashSingleQuote(ws.Path)+" && "+script)
	if hctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("hook timeout name=%s", name)
	}
	if err != nil {
		msg := strings.TrimSpace(out)
		if fatal {
			return fmt.Errorf("hook failed name=%s error=%w output=%s", name, err, msg)
		}
		return fmt.Errorf("hook failed name=%s error=%w output=%s", name, err, msg)
	}
	log.Printf("symphony hook status=ok name=%s worker_host=%s", name, *ws.WorkerHost)
	return nil
}

func (m *Manager) createRemote(ctx context.Context, key string, workspacePath string, workerHost *string) (Workspace, error) {
	target := ssh.ParseTarget(*workerHost)

	script := strings.Join([]string{
		"set -e",
		`dir=` + bashSingleQuote(workspacePath),
		"created=0",
		`if [ ! -d "$dir" ]; then mkdir -p "$dir"; created=1; fi`,
		`echo "__SYMPHONY_WORKSPACE__	$created	$dir"`,
	}, "\n")

	out, _, err := ssh.Run(ctx, target, script)
	if err != nil {
		return Workspace{}, fmt.Errorf("remote workspace create failed (worker_host=%s path=%s): %w\noutput: %s", *workerHost, workspacePath, err, strings.TrimSpace(out))
	}

	path, created, ok := parseWorkspaceMarker(out)
	if !ok {
		return Workspace{}, fmt.Errorf("remote workspace create: marker missing (worker_host=%s output=%s)", *workerHost, strings.TrimSpace(out))
	}

	ws := Workspace{
		Path:         path,
		WorkspaceKey: key,
		CreatedNow:   created,
		WorkerHost:   workerHost,
	}

	if ws.CreatedNow && strings.TrimSpace(m.hooks.AfterCreate) != "" {
		if err := m.runHook(ctx, ws, "after_create", m.hooks.AfterCreate, true); err != nil {
			_, _, _ = ssh.Run(ctx, target, "rm -rf "+bashSingleQuote(ws.Path))
			return Workspace{}, err
		}
	}
	return ws, nil
}

func parseWorkspaceMarker(output string) (path string, created bool, ok bool) {
	sc := bufio.NewScanner(strings.NewReader(output))
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "__SYMPHONY_WORKSPACE__\t") {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			return "", false, false
		}
		n, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return "", false, false
		}
		p := strings.TrimSpace(parts[2])
		if p == "" {
			return "", false, false
		}
		return p, n != 0, true
	}
	return "", false, false
}

func bashSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func (m *Manager) ensureRootReady() error {
	if m == nil {
		return errors.New("nil workspace manager")
	}
	if strings.TrimSpace(m.rootAbs) == "" {
		return errors.New("workspace root is required")
	}
	if err := os.MkdirAll(m.rootAbs, 0o755); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if strings.TrimSpace(m.rootCanon) != "" {
		return nil
	}
	canon, err := filepath.EvalSymlinks(m.rootAbs)
	if err != nil {
		return err
	}
	canon, err = filepath.Abs(canon)
	if err != nil {
		return err
	}
	m.rootCanon = canon
	return nil
}

func (m *Manager) canonicalizeSafeWorkspacePath(workspacePath string) (string, error) {
	if err := m.ensureRootReady(); err != nil {
		return "", err
	}

	m.mu.Lock()
	rootCanon := strings.TrimSpace(m.rootCanon)
	m.mu.Unlock()
	if rootCanon == "" {
		return "", errors.New("workspace root is not canonicalized")
	}

	wsCanon, err := filepath.EvalSymlinks(workspacePath)
	if err != nil {
		return "", fmt.Errorf("%w: workspace path unreadable (path=%s err=%v)", ErrInvalidWorkspaceCwd, workspacePath, err)
	}
	wsCanon, err = filepath.Abs(wsCanon)
	if err != nil {
		return "", err
	}

	sep := string(filepath.Separator)
	rootPrefix := rootCanon
	if !strings.HasSuffix(rootPrefix, sep) {
		rootPrefix += sep
	}

	if wsCanon == rootCanon {
		return "", fmt.Errorf("%w: workspace path resolves to workspace root (path=%s)", ErrInvalidWorkspaceCwd, workspacePath)
	}
	if !strings.HasPrefix(wsCanon+sep, rootPrefix) {
		// This catches symlink escapes where the lexical path is under the root but the
		// canonical target is outside.
		return "", fmt.Errorf("%w: workspace path escapes root via symlink (path=%s resolved=%s root=%s)", ErrInvalidWorkspaceCwd, workspacePath, wsCanon, rootCanon)
	}
	return wsCanon, nil
}
