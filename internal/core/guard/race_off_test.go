//go:build !race

package guard

// raceEnabled reports whether the race detector is instrumenting this build.
// The cost guards read it to skip under -race (see assertWorkGrowth): their
// counts are deterministic, so the instrumented run would assert the same
// numbers at many times the cost.
const raceEnabled = false
