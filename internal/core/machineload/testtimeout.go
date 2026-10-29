package machineload

import (
	"runtime"
	"time"
)

// TestTimeoutMaxFactor is the most GoTestTimeout scales a stage's base package
// budget by. land's bound on a whole preflight run holds both test stages at
// it (internal/core/landing, TestRunBoundHoldsBothStagesAtTheTimeoutCap).
const TestTimeoutMaxFactor = 2

// GoTestTimeout is the -timeout a `go test` stage started now should give each
// package, so a package slowed by the machine's load is not killed with no
// test hung (iss-2610241911052191): base while the one-minute load is at
// most one runnable process per online core, and TestTimeoutMaxFactor times
// base past that, since a package's tests take about as much longer as the
// runnable processes sharing each core. It takes those two values only, not
// one in proportion: go test hashes -timeout into its cache key, so a budget
// that moved with every reading would rerun every cached package on a busy
// machine. An unreadable load leaves the base.
// Like GoTestParallelism it reads the load as the stage starts, not as the
// stage runs.
func GoTestTimeout(snap Snapshot, base time.Duration) time.Duration {
	if !snap.HasLoad {
		return base
	}
	cores := snap.Cores
	if cores < 1 {
		cores = runtime.NumCPU()
	}
	if snap.Load1 <= float64(cores) {
		return base
	}
	return TestTimeoutMaxFactor * base
}
