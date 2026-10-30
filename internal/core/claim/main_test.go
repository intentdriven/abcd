package claim

import (
	"fmt"
	"os"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/testshard"
)

// TestMain runs the package as several test processes when make preflight
// asks: one is too long for its package budget on a loaded machine
// (iss-2610251004378066). Each keeps the machine's own GitHub login out of
// reach: an announcement asks the forge whose account it is, and a test that
// reached the real login would pass here and fail on CI, which has none.
func TestMain(m *testing.M) {
	if code, ok := testshard.Shard(); ok {
		os.Exit(code)
	}
	cleanup, err := gittest.HideGHLogin()
	if err != nil {
		fmt.Fprintln(os.Stderr, "claim tests: cannot hide the machine's gh login:", err)
		os.Exit(1)
	}
	code := m.Run()
	cleanup()
	os.Exit(code)
}
