package runner

// gatewait.go is a role's clock at abcd's load gate (iss-2611021751406429). A
// role bounded by its timeout runs its definition of done behind the gate,
// and on a shared machine it can wait there behind other preflights for most
// of its bound: a run queued behind them was killed sooner than its own work
// warranted. So while the process tree a harness leads holds a waiter queued
// at either gate (machineload.QueuedUnder: an `abcd implement load --wait` or
// `--race-gate` running no command yet), the run's clock stands still, and
// its deadline moves by the time the waiter was seen queued.
//
// The tree is read from the process table every gatePollInterval, and a look
// that finds a waiter queued credits the whole interval since the last look.
// The credit is bounded (gateCredit): a waiter that never opens cannot hold a
// run open forever, so a run lasts at most twice its bound (LongestRun), and
// the claim a step takes on its await outlasts that.

import (
	"time"

	"github.com/intentdriven/abcd/internal/core/machineload"
)

// gatePollInterval is how often a run asks whether its tree waits at the
// load gate.
const gatePollInterval = 30 * time.Second

// gateCredit is the most time a run of the given bound is credited for
// waiting at the load gate: the bound itself.
func gateCredit(timeout time.Duration) time.Duration { return timeout }

// LongestRun is the longest a run of the given bound can last: its bound and
// the most it is credited for waiting at the load gate.
func LongestRun(timeout time.Duration) time.Duration { return timeout + gateCredit(timeout) }

// readTable and readArgs read the machine; tests replace them.
var (
	readTable = machineload.Read
	readArgs  = machineload.ReadArgs
)

// queuedAtGate reports whether the tree pid leads holds a waiter queued at
// the load gate. A table or arguments that cannot be read hold none: the
// clock runs on.
func queuedAtGate(pid int) bool {
	snap, err := readTable()
	if err != nil || !snap.HasProcs {
		return false
	}
	cands := machineload.AbcdUnder(snap, pid)
	if len(cands) == 0 {
		return false
	}
	args, err := readArgs(cands)
	if err != nil {
		return false
	}
	return len(machineload.QueuedUnder(snap, pid, args)) > 0
}
