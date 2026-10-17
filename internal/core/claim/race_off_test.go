//go:build !race

package claim

// raceEnabled reports whether the race detector is instrumenting this build.
// The wall-clock bounds of the generated-file reader's budget tests read it
// to hold only uninstrumented: the budget they stand in for counts steps, so
// the instrumented run asserts the same stop at many times the cost, and the
// uninstrumented run asserts its time.
const raceEnabled = false
