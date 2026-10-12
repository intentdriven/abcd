//go:build unix

package cli

import (
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/implement"
)

// TestImplementLoadWaitForwardsSignals: a SIGTERM to the waiter reaches the
// command it runs, so the command is never orphaned (an orphan's hold reads as
// a dead holder's, and the next waiter would be admitted beside it); the waiter
// then exits with the command's code and releases its hold. The child traps
// TERM and otherwise sleeps briefly, so the test never loads the machine.
func TestImplementLoadWaitForwardsSignals(t *testing.T) {
	waitFixture(t, machine(3, 3, 3))
	ready := sigtermWhenReady(t)
	script := `trap 'echo child-got-term; exit 7' TERM; : > "$1"; i=0; while [ $i -lt 100 ]; do sleep 0.1; i=$((i+1)); done; echo child-not-signalled`
	code, out, errOut := implementCLI(t, "implement", "load", "--wait", "--", "sh", "-c", script, "sh", ready)
	if !strings.Contains(out, "child-got-term") || strings.Contains(out, "child-not-signalled") {
		t.Fatalf("the SIGTERM was not forwarded to the command (exit %d):\n%s%s", code, out, errOut)
	}
	if code != 7 {
		t.Fatalf("exit %d, want the command's 7\n%s%s", code, out, errOut)
	}
	home, _ := os.UserHomeDir()
	if _, err := os.Stat(abcdhome.Path(home, implement.WaitHoldFileName)); !os.IsNotExist(err) {
		t.Fatalf("the hold outlived its signalled command: %v", err)
	}
}

// TestImplementLoadWaitExitsLikeAShellOnASignalledCommand: a command killed by
// signal N makes the waiter exit 128+N, as a shell reports it, so a preflight
// stopped by SIGTERM reads 143 and never 1, which reads as a failed gate. The
// child does not trap TERM, so the forwarded signal kills it.
func TestImplementLoadWaitExitsLikeAShellOnASignalledCommand(t *testing.T) {
	waitFixture(t, machine(3, 3, 3))
	ready := sigtermWhenReady(t)
	script := `: > "$1"; i=0; while [ $i -lt 100 ]; do sleep 0.1; i=$((i+1)); done; echo child-not-signalled`
	code, out, errOut := implementCLI(t, "implement", "load", "--wait", "--", "sh", "-c", script, "sh", ready)
	if strings.Contains(out, "child-not-signalled") {
		t.Fatalf("the SIGTERM was not forwarded to the command (exit %d):\n%s%s", code, out, errOut)
	}
	if want := 128 + int(syscall.SIGTERM); code != want {
		t.Fatalf("exit %d, want %d for a command killed by SIGTERM\n%s%s", code, want, out, errOut)
	}
}

// sigtermWhenReady returns a path the gated command creates once it is
// running, and sends the test process a SIGTERM as soon as that path exists.
// The test process takes SIGTERM itself too, so a waiter that forwards nothing
// fails the caller's assertions instead of killing the test binary.
func sigtermWhenReady(t *testing.T) string {
	t.Helper()
	own := make(chan os.Signal, 1)
	signal.Notify(own, syscall.SIGTERM)
	t.Cleanup(func() { signal.Stop(own) })
	ready := filepath.Join(t.TempDir(), "started")
	go func() {
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(ready); err == nil {
				_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
	}()
	return ready
}
