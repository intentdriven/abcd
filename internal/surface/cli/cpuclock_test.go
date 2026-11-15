package cli

import (
	"syscall"
	"time"
)

// processCPU is the CPU time this test process has consumed, user and system,
// across every thread — internal/core/reading's linearity clock, kept here for
// this package's own guards. Its one caller, scanGrowth, compares two of its
// readings in one run, a read of an input and of one four times larger, rather
// than bounding either: time spent waiting for a core is not charged, but a
// slower core is, and it slows both readings alike (iss-2609240046582859,
// iss-2611121820123919). No test in this package runs in parallel, so the
// figure is the measured call's, plus the runtime's own collector.
func processCPU() time.Duration {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		panic("getrusage: " + err.Error())
	}
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano())
}
