//go:build !race

package ahoy

// raceEnabled reports whether the race detector is instrumenting this build.
// TestHeadingTextStaysLinear reads it to skip under -race, leaving its
// wall-time bound to the uninstrumented run the plain-only guards steps give it.
const raceEnabled = false
