//go:build !race

package ahoy

// raceEnabled reports whether the race detector is instrumenting this build.
// TestHeadingTextStaysLinear reads it to skip under -race: the detector's
// instrumentation, not the reader, sets the time there, and the plain-only
// guards steps run the test without it.
const raceEnabled = false
