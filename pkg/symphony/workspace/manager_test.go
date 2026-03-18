package workspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCreateForIssue_CreatesAndRunsAfterCreate(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()

	m, err := NewManager(root, HookScripts{
		AfterCreate: `echo "ok" > after_create.txt`,
		Timeout:     5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewManager error: %v", err)
	}

	ws, err := m.CreateForIssue(ctx, "ABC-123")
	if err != nil {
		t.Fatalf("CreateForIssue error: %v", err)
	}
	if !ws.CreatedNow {
		t.Fatal("expected CreatedNow")
	}
	if _, err := os.Stat(filepath.Join(ws.Path, "after_create.txt")); err != nil {
		t.Fatalf("expected after_create.txt, err=%v", err)
	}

	// Second call should reuse and not re-run after_create.
	_ = os.Remove(filepath.Join(ws.Path, "after_create.txt"))
	ws2, err := m.CreateForIssue(ctx, "ABC-123")
	if err != nil {
		t.Fatalf("CreateForIssue (second) error: %v", err)
	}
	if ws2.CreatedNow {
		t.Fatal("expected reuse workspace")
	}
	if _, err := os.Stat(filepath.Join(ws2.Path, "after_create.txt")); err == nil {
		t.Fatal("expected after_create not to run on reuse")
	}
}

func TestBeforeRun_FailureIsFatal(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()

	m, err := NewManager(root, HookScripts{
		BeforeRun: `exit 2`,
		Timeout:   5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewManager error: %v", err)
	}
	ws, err := m.CreateForIssue(ctx, "ABC-123")
	if err != nil {
		t.Fatalf("CreateForIssue error: %v", err)
	}
	if err := m.BeforeRun(ctx, ws); err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateForIssue_RejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	ctx := context.Background()

	m, err := NewManager(root, HookScripts{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewManager error: %v", err)
	}

	// Create a symlink workspace inside the root that points outside the root.
	linkPath := filepath.Join(root, "MT-1000")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Fatalf("Symlink error: %v", err)
	}

	_, err = m.CreateForIssue(ctx, "MT-1000")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrInvalidWorkspaceCwd) {
		t.Fatalf("expected ErrInvalidWorkspaceCwd, got %v", err)
	}
}
