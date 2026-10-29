package landing

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// m7: a preflight past its bound is stopped with every process it started,
// its whole process group, not make alone.
func TestAPreflightPastItsBoundIsKilledWithItsGroup(t *testing.T) {
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make is not installed")
	}
	dir := t.TempDir()
	mk := "preflight:\n\t@sleep 20 & echo $$! > child.pid; wait\n"
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(mk), 0o644); err != nil {
		t.Fatal(err)
	}
	prev := gateTimeout
	gateTimeout = 2 * time.Second
	t.Cleanup(func() { gateTimeout = prev })

	var out bytes.Buffer
	start := time.Now()
	err := makePreflight(&out)(dir)
	if err == nil {
		t.Fatal("a preflight past its bound passed")
	}
	if took := time.Since(start); took > 10*time.Second {
		t.Fatalf("the gate returned after %s: the bound did not stop what the preflight started", took)
	}
	raw, rerr := os.ReadFile(filepath.Join(dir, "child.pid"))
	if rerr != nil {
		t.Fatal(rerr)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(raw)))
	if pid <= 0 {
		t.Fatalf("child.pid %q", raw)
	}
	deadline := time.Now().Add(3 * time.Second)
	for syscall.Kill(pid, 0) == nil {
		if time.Now().After(deadline) {
			_ = syscall.Kill(pid, syscall.SIGKILL)
			t.Fatalf("the preflight's child %d outlived the bound", pid)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// minPreflightRun is the least a whole preflight run takes once the load gate
// admits it, on a loaded machine: its unit and race stages each run under a
// 40-minute package budget, after the gates, the eval lanes, build and vet.
const minPreflightRun = 4 * time.Hour

// TestGateTimeoutOutlastsTheWholeRunWait: `make preflight` waits at the load
// gate as one whole run, for up to the --timeout its recipe gives the gate,
// and only then runs its stages. Land's bound on the preflight must outlast
// that wait and a run's length, or land kills a preflight that is still
// waiting its turn or partway through it. The wait is read from the Makefile,
// so the two cannot drift.
func TestGateTimeoutOutlastsTheWholeRunWait(t *testing.T) {
	wait := makefileGateWait(t)
	if wait != preflightGateWait {
		t.Errorf("the Makefile's preflight gate waits up to %s, and preflightGateWait says %s", wait, preflightGateWait)
	}
	if gateTimeout < wait+minPreflightRun {
		t.Fatalf("gateTimeout is %s; the Makefile's gate waits up to %s before a run of at least %s, so want at least %s",
			gateTimeout, wait, minPreflightRun, wait+minPreflightRun)
	}
}

// makefileGateWait is the --timeout of the load gate the Makefile's preflight
// recipe runs its whole run behind.
func makefileGateWait(t *testing.T) time.Duration {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	in := false
	for _, l := range strings.Split(string(raw), "\n") {
		if l == "preflight:" {
			in = true
			continue
		}
		if in && !strings.HasPrefix(l, "\t") {
			break
		}
		if !in || !strings.Contains(l, "implement load --wait") {
			continue
		}
		f := strings.Fields(l)
		for i := 0; i+1 < len(f) && f[i] != "--"; i++ {
			if f[i] == "--timeout" {
				d, err := time.ParseDuration(f[i+1])
				if err != nil {
					t.Fatalf("the preflight gate's --timeout %q: %v", f[i+1], err)
				}
				return d
			}
		}
	}
	t.Fatal("the Makefile's preflight recipe runs no `implement load --wait --timeout <d>`")
	return 0
}
