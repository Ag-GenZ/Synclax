package orchestrator

import (
	"testing"
	"time"
)

func TestBackoffDelay_CappedExponential(t *testing.T) {
	cap := 45 * time.Second
	if got := backoffDelay(1, cap); got != 10*time.Second {
		t.Fatalf("attempt 1: expected 10s, got %v", got)
	}
	if got := backoffDelay(2, cap); got != 20*time.Second {
		t.Fatalf("attempt 2: expected 20s, got %v", got)
	}
	if got := backoffDelay(3, cap); got != 40*time.Second {
		t.Fatalf("attempt 3: expected 40s, got %v", got)
	}
	if got := backoffDelay(4, cap); got != cap {
		t.Fatalf("attempt 4: expected cap %v, got %v", cap, got)
	}
}
