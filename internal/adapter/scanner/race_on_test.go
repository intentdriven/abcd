//go:build race

package scanner

// raceEnabled reports whether the race detector is instrumenting this build.
// See the sibling file for what reads it.
const raceEnabled = true
