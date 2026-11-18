//go:build !race

package ahoy

// raceEnabled reports whether the race detector is instrumenting this build.
// TestHeadingTextStaysLinear reads it to drop its wall-clock bound under
// -race: the detector's instrumentation, not the reader, sets the time there,
// and the plain-only guards step runs the test without it.
const raceEnabled = false
