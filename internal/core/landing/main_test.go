package landing

import (
	"fmt"
	"os"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestMain keeps the machine's own GitHub login out of reach: a landing
// announces before it pushes and asks the forge whose account it is when git's
// identity does not say, so a test that reached the real login would pass here
// and fail on CI, which has none.
func TestMain(m *testing.M) {
	// A stop on the machine running the tests holds no test here; a test of the stop
	// stands in its own read (Options.StopRead).
	stopReadingOff = true
	// The account's own home is the test's HOME, which each test names: the
	// machine layer and a person's own setting are read for the mode only
	// there (personmode.go).
	AccountHome = os.UserHomeDir
	cleanup, err := gittest.HideGHLogin()
	if err != nil {
		fmt.Fprintln(os.Stderr, "landing tests: cannot hide the machine's gh login:", err)
		os.Exit(1)
	}
	code := m.Run()
	cleanup()
	os.Exit(code)
}
