package ahoy

import (
	"os"
	"testing"

	"github.com/intentdriven/abcd/internal/core/vintage"
)

// TestMain defaults the build-vintage seam to a fresh, determinable, non-dogfood
// value for the whole package. A go-test binary is itself unstamped, so without
// this every install path would hit the itd-111 unknown-vintage refusal; tests
// that exercise the refusal or the notice override the seam per-test.
// stubCopyRelease is the release every fixture copy claims to be.
const stubCopyRelease = "v0.13.4"

func TestMain(m *testing.M) {
	currentVintage = func() vintage.Current {
		return vintage.Current{Revision: "testvintage", Known: true}
	}
	// No test runs the machine's own agent tool to read its version; the
	// version test puts the real reading back over a fake command.
	NoHostVersionForTest()
	// The fixtures standing for a release binary are shell scripts with no Go
	// build metadata; the copy's claimed release is stood in, and the reader's
	// own tests call it directly (cold_cache_adopt_provenance_test.go).
	copyRelease = func([]byte) (string, error) { return stubCopyRelease, nil }
	os.Exit(m.Run())
}
