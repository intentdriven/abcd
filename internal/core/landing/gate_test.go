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

	"github.com/intentdriven/abcd/internal/core/machineload"
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
// and only then runs its stages, whose race stage may wait at the race gate
// for up to that gate's own --timeout. Land's bound on the preflight must
// outlast both waits and a run's length, or land kills a preflight that is
// still waiting its turn or partway through it. Both waits are read from the
// Makefile, so they cannot drift.
func TestGateTimeoutOutlastsTheWholeRunWait(t *testing.T) {
	wait := makefileGateTimeout(t, "preflight", "implement load --wait")
	if wait != preflightGateWait {
		t.Errorf("the Makefile's preflight gate waits up to %s, and preflightGateWait says %s", wait, preflightGateWait)
	}
	race := makefileGateTimeout(t, "preflight-stages", "implement load --race-gate")
	if race != preflightRaceGateWait {
		t.Errorf("the Makefile's race gate waits up to %s, and preflightRaceGateWait says %s", race, preflightRaceGateWait)
	}
	if want := wait + race + minPreflightRun; gateTimeout < want {
		t.Fatalf("gateTimeout is %s; the Makefile's gate waits up to %s and its race gate up to %s around a run of at least %s, so want at least %s",
			gateTimeout, wait, race, minPreflightRun, want)
	}
}

// makefileGateTimeout is the --timeout of the load gate that the recipe of the
// Makefile's target runs, found by marker, its spelling of the gate.
func makefileGateTimeout(t *testing.T, target, marker string) time.Duration {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	in := false
	for _, l := range strings.Split(string(raw), "\n") {
		if l == target+":" || strings.HasPrefix(l, target+": ") {
			in = true
			continue
		}
		if in && !strings.HasPrefix(l, "\t") {
			break
		}
		if !in || !strings.Contains(l, marker) {
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
	t.Fatalf("the Makefile's %s recipe runs no `%s --timeout <d>`", target, marker)
	return 0
}

// TestRunBoundHoldsBothStagesAtTheTimeoutCap: each of preflight's two test
// stages takes its base package budget scaled with the load as it starts, up
// to machineload.TestTimeoutMaxFactor times the base (iss-2610241911052191),
// so land's bound on the run admits both stages at that cap.
func TestRunBoundHoldsBothStagesAtTheTimeoutCap(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	var stages []time.Duration
	for _, l := range strings.Split(string(raw), "\n") {
		f := strings.Fields(l)
		for i := 0; i+1 < len(f); i++ {
			if f[i] == "./cmd/test-parallelism" && f[i+1] == "-timeout" && i+2 < len(f) {
				d, err := time.ParseDuration(strings.TrimSuffix(f[i+2], ")"))
				if err != nil {
					t.Fatalf("the Makefile's stage base %q: %v", f[i+2], err)
				}
				stages = append(stages, d)
			}
		}
	}
	if len(stages) != 2 {
		t.Fatalf("the Makefile derives %d stage timeouts through cmd/test-parallelism -timeout, want 2 (the unit and race stages)", len(stages))
	}
	var need time.Duration
	for _, d := range stages {
		need += time.Duration(machineload.TestTimeoutMaxFactor) * d
	}
	if preflightRunBound < need {
		t.Fatalf("preflightRunBound is %s; two stages at %d times their bases %v need %s", preflightRunBound, machineload.TestTimeoutMaxFactor, stages, need)
	}
}
