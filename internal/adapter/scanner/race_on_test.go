//go:build race

package scanner

// raceEnabled reports whether the race detector is instrumenting this build.
// See the sibling file for why a duration assertion needs it.
const raceEnabled = true
