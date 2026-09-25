package cli

import (
	"os"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/rules"
	"github.com/intentdriven/abcd/internal/core/vintage"
)

// TestMain defaults the ahoy build-vintage seam to a fresh, determinable,
// non-dogfood value for the whole package. runCLI executes the commands
// in-process through this unstamped go-test binary, so without the override
// every `ahoy install` exercised here would hit the itd-111 unknown-vintage
// refusal.
//
// It also keeps the developer's own ~/.abcd/rules.json out of every rules load
// a test did not lay a user layer out for: while HOME is still the process's
// own, the user layer reads as absent, and a test that sets HOME to a fixture
// gets that fixture's user layer (spc-23).
func TestMain(m *testing.M) {
	ahoy.SetCurrentVintageForTest(func() vintage.Current {
		return vintage.Current{Revision: "testvintage", Known: true}
	})
	real := os.Getenv("HOME")
	rules.SwapUserHomeForTest(func() (string, error) {
		if home := os.Getenv("HOME"); home != real {
			return home, nil
		}
		return "", nil
	})
	os.Exit(m.Run())
}
