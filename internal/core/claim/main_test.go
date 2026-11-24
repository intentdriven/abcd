package claim

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/testshard"
)

// TestMain runs the package as several test processes when make preflight
// asks: one is too long for its package budget on a loaded machine
// (iss-2610251004378066). Each keeps the machine's own GitHub login out of
// reach: an announcement asks the forge whose account it is, and a test that
// reached the real login would pass here and fail on CI, which has none.
//
// It also widens pusherBudget for the whole process. The budget bounds the
// git walks that place each push, one per push, and the walk tests build
// hundreds of renewals, one push per commit: on a slow runner placing them
// outran the production ten seconds and they came back unplaced, failing the
// macOS suite on three nightlies and a release gate. That per-push cost is
// iss-2611240757104266; here the walk cases judge placement, not the runner's
// speed. The tests of the budget itself set and restore their own value.
func TestMain(m *testing.M) {
	pusherBudget = testPusherBudget
	if code, ok := testshard.Shard(); ok {
		os.Exit(code)
	}
	cleanup, err := gittest.HideGHLogin()
	if err != nil {
		fmt.Fprintln(os.Stderr, "claim tests: cannot hide the machine's gh login:", err)
		os.Exit(1)
	}
	code := m.Run()
	removeWorldTemplates()
	cleanup()
	os.Exit(code)
}

// testPusherBudget is the claim tests' pusherBudget: wide enough for the
// slowest runner's walk, so only a test that sets its own budget judges one.
const testPusherBudget = 2 * time.Minute
