//go:build darwin

package machineload

import (
	"context"
	"os/exec"
	"testing"
	"time"
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

// TestReadContextEndingStopsPS: a context that ends while ps runs stops it,
// and ReadContext answers an error, well before ps's own psTimeout. The seam
// runs a sleep that outlasts psTimeout in ps's place, under ps's command and
// so its context.
func TestReadContextEndingStopsPS(t *testing.T) {
	prev := runPS
	runPS = func(cmd *exec.Cmd) error {
		cmd.Path, cmd.Args = "/bin/sleep", []string{"/bin/sleep", "60"}
		return cmd.Run()
	}
	t.Cleanup(func() { runPS = prev })
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	began := time.Now()
	_, err := ReadContext(ctx)
	took := time.Since(began)
	if err == nil {
		t.Fatal("ReadContext answered no error for a context that ended while ps ran")
	}
	if took >= psTimeout/2 {
		t.Fatalf("ReadContext returned after %s, want the context's end to stop ps, not psTimeout (%s)", took, psTimeout)
	}
}
