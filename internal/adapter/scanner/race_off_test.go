//go:build !race

package scanner

// raceEnabled reports whether the race detector is instrumenting this build.
// The cost guards read it to skip under -race (see assertCostGrowth): their
// counts are deterministic, so the instrumented run would assert the same
// numbers at many times the cost. A build tag answers at compile time, which is
// why it is the package's one race-detection mechanism.
const raceEnabled = false
