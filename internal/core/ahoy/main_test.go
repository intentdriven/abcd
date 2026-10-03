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
func TestMain(m *testing.M) {
	currentVintage = func() vintage.Current {
		return vintage.Current{Revision: "testvintage", Known: true}
	}
	// No test runs the machine's own agent tool to read its version; the
	// version test puts the real reading back over a fake command.
	NoHostVersionForTest()
	os.Exit(m.Run())
}
