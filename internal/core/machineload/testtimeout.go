package machineload

import (
	"math"
	"runtime"
	"time"
)

// TestTimeoutMaxFactor is the most GoTestTimeout scales a stage's base package
// budget by. land's bound on a whole preflight run holds both test stages at
// it (internal/core/landing, TestRunBoundHoldsBothStagesAtTheTimeoutCap).
const TestTimeoutMaxFactor = 2

// GoTestTimeout is the -timeout a `go test` stage started now should give each
// package, so a package slowed by the machine's load is not killed with no
// test hung (iss-2610241911052191): base scaled by the one-minute load per
// online core, never less than base and never more than TestTimeoutMaxFactor
// times it, rounded up to a whole minute. The load per core is how many
// runnable processes share each core, so a package's tests take about that
// much longer than on a quiet machine. An unreadable load leaves the base.
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
	factor := min(max(snap.Load1/float64(cores), 1), TestTimeoutMaxFactor)
	return time.Duration(math.Ceil(float64(base)*factor/float64(time.Minute))) * time.Minute
}
