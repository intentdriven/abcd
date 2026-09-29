package reading

import (
	"syscall"
	"testing"
	"time"
)

// processCPU is the CPU time this test process has consumed, user and system,
// across every thread. The linearity guards bound it rather than the wall clock
// because it is the cost that tells a linear scan from a quadratic one and the
// one machine load cannot inflate: time spent waiting for a core is not
// charged, so a gate run on a busy machine reads the same figure as an idle one
// (iss-2609240046582859). No test in this package runs in parallel, so the
// figure is the guarded call's, plus the runtime's own collector. getrusage is
// the macOS and Linux call the gates run on.
func processCPU() time.Duration {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		panic("getrusage: " + err.Error())
	}
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano())
}

// TestTheLinearityClockDoesNotChargeWaiting: the clock the linearity guards
// read charges work, not waiting. A wall-clock ceiling charged the time a
// loaded machine kept the scan off a core, so a gate run could fail on linear
// code with nothing wrong (iss-2609240046582859); a sleep stands in for that
// wait here, and the clock must not see it.
func TestTheLinearityClockDoesNotChargeWaiting(t *testing.T) {
	start := processCPU()
	time.Sleep(300 * time.Millisecond)
	if spent := processCPU() - start; spent > 100*time.Millisecond {
		t.Fatalf("the clock charged %s across a 300ms wait; it counts time off the core, which load inflates", spent)
	}

	// And it charges work, or every ceiling read off it would pass a quadratic scan.
	start = processCPU()
	deadline := time.Now().Add(30 * time.Second)
	sink := 0
	for processCPU()-start < 50*time.Millisecond {
		if time.Now().After(deadline) {
			t.Fatalf("30s of spinning charged %s; the clock does not count work", processCPU()-start)
		}
		for i := range 100000 {
			sink += i
		}
	}
	_ = sink
}
