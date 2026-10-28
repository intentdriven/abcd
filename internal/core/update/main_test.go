package update

import (
	"os"
	"testing"
)

// TestMain stands in a release claim every fixture passes: the package's
// release binaries are synthetic bytes with no Go build metadata, so none
// claims a release. The binding's own tests put a claim back
// (release_claim_test.go), and ahoy holds the reader itself to real bytes.
func TestMain(m *testing.M) {
	checkReleaseClaim = func([]byte, string) error { return nil }
	os.Exit(m.Run())
}
