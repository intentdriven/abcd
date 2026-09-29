package fsutil

import (
	"os"
	"testing"
)

// SkipFlushEnv names the opt-in a test run sets to skip the flush to stable
// storage. The Makefile's test and preflight targets and the test steps of
// ci.yml's check job set it to "1".
//
// It is honoured only inside a test binary. On macOS, File.Sync is F_FULLFSYNC,
// which costs about 8.8 ms per synced write on a developer disk, and the tests
// write thousands of files while asserting nothing a flush makes true: a flush
// is what survives a power cut, and no test cuts the power. A shipped binary
// flushes whatever its environment says, because testing.Testing reports false
// in every binary the go command builds other than a test binary, so neither
// half of the gate alone disables durability.
const SkipFlushEnv = "ABCD_TEST_SKIP_FLUSH"

// syncFile is the flush itself: fsync, and F_FULLFSYNC on macOS. It is a
// variable so the durability test can count the flushes a write makes.
var syncFile = (*os.File).Sync

// Flush makes what was written to f durable — its content for a file, its
// entries for a directory — and returns the flush's error. It is the one call
// every durable write in this module flushes through
// (TestEveryFlushGoesThroughTheGate), so the test-only skip lives in one place.
func Flush(f *os.File) error {
	if skipFlush(testing.Testing(), os.Getenv(SkipFlushEnv)) {
		return nil
	}
	return syncFile(f)
}

// skipFlush is the gate: a flush is skipped only in a test binary whose
// environment opted in with exactly "1". Any other value, or any binary that
// is not a test binary, flushes.
func skipFlush(testBinary bool, optIn string) bool {
	return testBinary && optIn == "1"
}
