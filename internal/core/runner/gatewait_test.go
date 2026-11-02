package runner

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/machineload"
)

// sleepLauncher is a launcher that runs the system's sleep, asking gate
// whether its tree waits at the load gate every few milliseconds.
func sleepLauncher(t *testing.T, gate func(int) bool) (launcher, string) {
	t.Helper()
	bin, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("no sleep on PATH")
	}
	l := defaultLauncher()
	l.gateWaiting, l.gatePoll = gate, 20*time.Millisecond
	return l, bin
}

// TestARunWaitingAtTheLoadGateIsNotKilledForIt: a run whose tree waits at
// the load gate for longer than its bound leaves the gate with its clock
// stopped, and finishes its work rather than being killed at the bound
// (iss-2611021751406429).
func TestARunWaitingAtTheLoadGateIsNotKilledForIt(t *testing.T) {
	start := time.Now()
	l, bin := sleepLauncher(t, func(int) bool { return time.Since(start) < 800*time.Millisecond })
	if _, err := l.run(context.Background(), Claude, bin, []string{"1.5"}, nil, t.TempDir(), time.Second); err != nil {
		t.Fatalf("a run that waited 0.8s at the load gate, then worked 0.7s of its 1s bound, failed: %v", err)
	}
}

// TestARunNotWaitingAtTheGateIsKilledAtItsBound: the clock stops only for
// the gate, and a kill at the bound names its cause, ErrKilled.
func TestARunNotWaitingAtTheGateIsKilledAtItsBound(t *testing.T) {
	var asked atomic.Int32
	l, bin := sleepLauncher(t, func(int) bool { asked.Add(1); return false })
	_, err := l.run(context.Background(), Claude, bin, []string{"5"}, nil, t.TempDir(), 500*time.Millisecond)
	var fl *Failure
	if !errors.As(err, &fl) || !errors.Is(err, ErrKilled) || !strings.Contains(fl.Detail, "time allowed") {
		t.Fatalf("err = %v, want a kill at the bound", err)
	}
	if asked.Load() == 0 {
		t.Error("the run never asked whether its tree waited at the load gate")
	}
}

// TestTheGateCreditIsBounded: a waiter that never opens holds a run open for
// at most gateCredit past its bound, and the kill says how long it waited.
func TestTheGateCreditIsBounded(t *testing.T) {
	l, bin := sleepLauncher(t, func(int) bool { return true })
	start := time.Now()
	_, err := l.run(context.Background(), Claude, bin, []string{"10"}, nil, t.TempDir(), 500*time.Millisecond)
	took := time.Since(start)
	if !errors.Is(err, ErrKilled) {
		t.Fatalf("err = %v, want a kill", err)
	}
	if took > LongestRun(500*time.Millisecond)+2*time.Second {
		t.Errorf("the run lasted %s, past its longest (%s)", took, LongestRun(500*time.Millisecond))
	}
	if !strings.Contains(err.Error(), "load gate") {
		t.Errorf("the kill does not say the run waited at the load gate: %v", err)
	}
}

// TestQueuedAtGateReadsTheHarnessTree: the production question reads the
// process table for the tree the harness leads.
func TestQueuedAtGateReadsTheHarnessTree(t *testing.T) {
	snap := machineload.Snapshot{HasProcs: true, Procs: []machineload.Proc{
		{PID: 10, PPID: 1, Name: "claude"},
		{PID: 11, PPID: 10, Name: "abcd"},
	}}
	args := []string{"implement", "load", "--wait"}
	oldT, oldA := readTable, readArgs
	t.Cleanup(func() { readTable, readArgs = oldT, oldA })
	readTable = func() (machineload.Snapshot, error) { return snap, nil }
	readArgs = func(pids []int) (map[int][]string, error) { return map[int][]string{11: args}, nil }
	if !queuedAtGate(10) {
		t.Error("a queued waiter under the harness was not seen")
	}
	if queuedAtGate(99) {
		t.Error("a waiter outside the harness's tree was counted")
	}
	readTable = func() (machineload.Snapshot, error) { return machineload.Snapshot{}, machineload.ErrUnsupported }
	if queuedAtGate(10) {
		t.Error("an unread table stopped the clock")
	}
}

// TestThePromptSaysTheGateIsNotCounted: the role is told its bound leaves
// out its wait at the load gate, so it does not hurry past a queue.
func TestThePromptSaysTheGateIsNotCounted(t *testing.T) {
	p := prompt(Request{Role: "implementer", Brief: "b", Receipt: "r", Timeout: 3 * time.Hour})
	if !strings.Contains(p, "stopped 3h0m0s after it starts") || !strings.Contains(p, "load gate") {
		t.Fatalf("the prompt names the bound and the load gate's credit:\n%s", p)
	}
}
