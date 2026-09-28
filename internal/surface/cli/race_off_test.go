//go:build !race

package cli

// raceEnabled reports whether the race detector is instrumenting this build.
// A test whose cost is volume through one goroutine reads it to skip under
// -race (see TestHistoryCaptureAcceptsWhatTheHooksAccept): the detector has
// nothing to watch there, and the instrumented run would assert the same thing
// at many times the cost. A build tag answers at compile time, the same
// mechanism internal/adapter/scanner and internal/core/guard use.
const raceEnabled = false
