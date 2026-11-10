//go:build darwin

package machineload

import (
	"context"
	"os/exec"
	"testing"
)

// TestReadContextSpentRunsNoPS: on macOS the process table is a ps run, and
// a context already ended starts none (the hook's deadline, not ps's own
// twenty seconds, decides).
func TestReadContextSpentRunsNoPS(t *testing.T) {
	runs := 0
	prev := runPS
	runPS = func(cmd *exec.Cmd) error { runs++; return prev(cmd) }
	t.Cleanup(func() { runPS = prev })
	spent, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ReadContext(spent); err == nil {
		t.Fatal("ReadContext on a spent context answered no error")
	}
	if runs != 0 {
		t.Fatalf("ReadContext on a spent context started ps %d time(s)", runs)
	}
	if _, err := ReadContext(context.Background()); err != nil {
		t.Logf("a live read failed (%v); the count below still holds", err)
	}
	if runs != 1 {
		t.Fatalf("ReadContext on a live context started ps %d time(s), want 1: the seam counts nothing", runs)
	}
}
