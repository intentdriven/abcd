package rules

import (
	"os"
	"testing"
)

// TestMain keeps the developer's own ~/.abcd/rules.json out of every test that
// did not lay one out: while HOME is still the process's own, the user layer
// reads as absent, and a test that sets HOME to a fixture gets that fixture's
// user layer (spc-23). Without it, a machine carrying a user layer would change
// what every Load in this package returns.
func TestMain(m *testing.M) {
	real := os.Getenv("HOME")
	restore := SwapUserHomeForTest(func() (string, error) {
		if home := os.Getenv("HOME"); home != real {
			return home, nil
		}
		return "", nil
	})
	code := m.Run()
	restore()
	os.Exit(code)
}
