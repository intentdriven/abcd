package dashboard

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"
)

// TestReadProcessTakesATimedOutPsForAnError holds a ps killed by its own
// deadline to a real error. The kill makes ps exit with "signal: killed", an
// exec.ExitError like the one a missing pid gives, and reading it as
// errProcessGone would have Stop take a live server on a stalled machine for
// stale, remove its run file and signal nothing (iss-2610112255507078).
func TestReadProcessTakesATimedOutPsForAnError(t *testing.T) {
	timeout, command := psTimeout, psCommand
	t.Cleanup(func() { psTimeout, psCommand = timeout, command })
	psTimeout = 50 * time.Millisecond
	psCommand = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "/bin/sleep", "30")
	}
	_, err := readProcess(os.Getpid())
	if err == nil || errors.Is(err, errProcessGone) {
		t.Fatalf("readProcess with ps killed by its deadline = %v, want an error other than errProcessGone", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("readProcess with ps killed by its deadline = %v, want it to wrap context.DeadlineExceeded", err)
	}
}
