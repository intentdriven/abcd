package cli

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// drainSignalChildEnv marks the re-executed test binary as the child that
// takes the drain's signal context and is interrupted twice.
const drainSignalChildEnv = "ABCD_TEST_DRAIN_SIGNAL_CHILD"

// TestDrainSignalContext: the drain's first interrupt ends its context, which
// a routed judge and every move stop on, and the drain's handler is released
// as it ends, so a second interrupt takes the default action and stops the
// process, a move stuck in a network act included, rather than being
// swallowed. The child is this test binary re-executed: it takes the context,
// interrupts itself, waits for the context to end, then interrupts itself
// again for up to two seconds; one still running after that exits 0.
func TestDrainSignalContext(t *testing.T) {
	if os.Getenv(drainSignalChildEnv) != "" {
		ctx, stop := drainSignalContext()
		defer stop()
		_ = syscall.Kill(os.Getpid(), syscall.SIGINT)
		select {
		case <-ctx.Done():
		case <-time.After(5 * time.Second):
			os.Exit(3)
		}
		for range 200 {
			_ = syscall.Kill(os.Getpid(), syscall.SIGINT)
			time.Sleep(10 * time.Millisecond)
		}
		os.Exit(0)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestDrainSignalContext$", "-test.count=1")
	cmd.Env = append(os.Environ(), drainSignalChildEnv+"=1")
	out, err := cmd.CombinedOutput()
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("the child survived a second interrupt (err %v):\n%s", err, out)
	}
	ws, ok := ee.Sys().(syscall.WaitStatus)
	if !ok || !ws.Signaled() || ws.Signal() != syscall.SIGINT {
		t.Fatalf("the second interrupt does not stop the process by its default action: %v\n%s", err, out)
	}
}
