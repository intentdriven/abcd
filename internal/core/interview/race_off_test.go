//go:build !race

package interview

// raceEnabled reports whether the race detector is instrumenting this build.
// stubBinary reads it: under -race the stub runner is this package's test
// binary built without the detector, since every start of an instrumented
// binary pays the detector's set-up (about a second on a developer Mac) and
// the stub is the harness's stand-in, not the code under test.
const raceEnabled = false
