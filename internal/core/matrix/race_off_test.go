//go:build !race

package matrix

// raceEnabled reports whether the race detector is instrumenting this build.
// The scrub's linearity guard reads it and asserts no wall-clock bound under
// -race: the instrumented run multiplies the cost many times over, so a bound
// set for the plain lane would fail there on linear code, and one loose enough
// for both would not catch the quadratic shape.
const raceEnabled = false
