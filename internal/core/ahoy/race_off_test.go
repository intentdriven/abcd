//go:build !race

package ahoy

// raceEnabled reports whether the race detector is instrumenting this build.
// TestHeadingTextStaysLinear reads it to hold its wall-time bound only on the
// uninstrumented run, which the plain-only guards step gives it.
const raceEnabled = false
