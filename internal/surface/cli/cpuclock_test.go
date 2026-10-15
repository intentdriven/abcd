package cli

import (
	"syscall"
	"time"
)

// processCPU is the CPU time this test process has consumed, user and system,
// across every thread — internal/core/reading's linearity clock, kept here for
// this package's own guards. A linearity guard bounds it rather than the wall
// clock because it tells a linear scan from a quadratic one and machine load
// cannot inflate it: time spent waiting for a core is not charged
// (iss-2609240046582859). No test in this package runs in parallel, so the
// figure is the guarded call's, plus the runtime's own collector.
func processCPU() time.Duration {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		panic("getrusage: " + err.Error())
	}
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano())
}
