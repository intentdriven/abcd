//go:build !race

package reading

// raceEnabled reports whether the race detector is instrumenting this build.
// The linearity guards read it and assert no wall-clock bound under -race: the
// instrumented run multiplies the cost many times over (23.7 s against 1.1 s for
// the size-cap case), so a bound set for the plain lane fails there on linear
// code, and a bound loose enough for both would not catch the quadratic shape.
// The verdicts are still asserted. A build tag answers at compile time, the
// mechanism core/adapter/scanner uses for its cost guards.
const raceEnabled = false
