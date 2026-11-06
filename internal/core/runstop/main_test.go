package runstop

import (
	"os"
	"testing"
)

// readNoPin is the read of a move with no run behind it.
func readNoPin(home string) Reading { return ReadPinned(home, nil) }

// TestMain keeps the machine's own shared folder out of every fixture: a test of
// the well-known folder names its own.
func TestMain(m *testing.M) {
	WellKnownFolder = func() string { return "" }
	os.Exit(m.Run())
}
