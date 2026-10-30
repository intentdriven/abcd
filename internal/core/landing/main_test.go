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
	cleanup, err := gittest.HideGHLogin()
	if err != nil {
		fmt.Fprintln(os.Stderr, "landing tests: cannot hide the machine's gh login:", err)
		os.Exit(1)
	}
	code := m.Run()
	cleanup()
	os.Exit(code)
}
