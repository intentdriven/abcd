//go:build !race

package mdrecord

// raceEnabled reports whether the race detector is instrumenting this build.
// TestReadPageReadsARecordAtTheSizeLimitPromptly asserts a wall-clock bound
// only without it: the instrumented run multiplies the cost many times over.
const raceEnabled = false
