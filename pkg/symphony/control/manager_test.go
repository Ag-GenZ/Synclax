package control

import (
	"testing"
	"time"

	"github.com/wibus-wee/synclax/pkg/symphony/orchestrator"
)

func TestSnapshotSecondsRunningIncreases(t *testing.T) {
	m := &Manager{startedAt: time.Now().Add(-500 * time.Millisecond)}

	s1 := m.Snapshot()
	sec1 := snapshotSecondsRunning(t, s1)

	time.Sleep(25 * time.Millisecond)

	s2 := m.Snapshot()
	sec2 := snapshotSecondsRunning(t, s2)

	if sec2 <= sec1 {
		t.Fatalf("expected seconds_running to increase, got sec1=%v sec2=%v", sec1, sec2)
	}
}

func snapshotSecondsRunning(t *testing.T, snap map[string]any) float64 {
	t.Helper()
	v, ok := snap["codex_totals"]
	if !ok {
		t.Fatalf("missing codex_totals in snapshot")
	}
	switch tt := v.(type) {
	case orchestrator.CodexTotals:
		return tt.SecondsRunning
	case *orchestrator.CodexTotals:
		if tt == nil {
			t.Fatalf("nil codex_totals")
		}
		return tt.SecondsRunning
	case map[string]any:
		if f, ok := tt["seconds_running"].(float64); ok {
			return f
		}
		t.Fatalf("unexpected codex_totals map shape: %T", tt["seconds_running"])
	default:
		t.Fatalf("unexpected codex_totals type: %T", v)
	}
	return 0
}
