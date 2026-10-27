package machineload

import (
	"reflect"
	"testing"
	"time"
)

// TestUngatedTestsNamesTheGoTestRunsNoGateAdmitted (iss-2610241130157455): a
// plain go test, by any account and with any flags, that no make preflight and
// no waiter started is outside the gate: it adds to the load the gate waits on
// without ever waiting itself, so the waiter names it. A go test beneath a make
// preflight or beneath a waiter running its command was admitted by a gate and
// is not named, nor is the waiter's own chain, nor a go command that is not a
// test run.
func TestUngatedTestsNamesTheGoTestRunsNoGateAdmitted(t *testing.T) {
	procs, args := waiterChain()
	add := func(pid, ppid int, name string, a []string) {
		procs = append(procs, Proc{PID: pid, PPID: ppid, PGID: pid, UID: otherUID, Age: time.Minute, Name: name})
		if a != nil {
			args[pid] = a
		}
	}
	// Ungated: a plain package run, and one with -race (a race run is also
	// outside the gate when nothing gated it).
	add(70000, 1, "go", []string{"test", "-timeout", "40m", "./internal/surface/cli/"})
	add(70100, 1, "go", []string{"-C", "/x", "test", "-race", "./internal/core/claim/"})
	// Gated: beneath a make preflight, and beneath a peer waiter running it.
	add(71000, 1, "make", []string{"preflight"})
	add(71001, 71000, "go", []string{"test", "-timeout", "20m", "./..."})
	add(72000, 1, "abcd", []string{"implement", "load", "--wait", "--", "go", "test", "./internal/core/spec/"})
	add(72001, 72000, "go", []string{"test", "./internal/core/spec/"})
	// Not a test run.
	add(73000, 1, "go", []string{"build", "./..."})
	snap := Snapshot{HasLoad: true, Cores: 10, Procs: procs, HasProcs: true}
	if got := UngatedTests(snap, waiter, args); !reflect.DeepEqual(got, []int{70000, 70100}) {
		t.Fatalf("ungated test runs = %v, want [70000 70100]", got)
	}
}
