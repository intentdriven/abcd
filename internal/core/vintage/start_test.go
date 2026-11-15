package vintage

import (
	"os"
	"testing"
)

// TestTheExecutableAtStartIsTheRunningOne: what the process found at start
// is the executable it runs from.
func TestTheExecutableAtStartIsTheRunningOne(t *testing.T) {
	path, fi := ExecutableAtStart()
	exe, err := os.Executable()
	if err != nil {
		t.Skipf("the executable cannot be named here: %v", err)
	}
	now, err := os.Stat(exe)
	if err != nil {
		t.Fatal(err)
	}
	if path != exe || fi == nil || !os.SameFile(fi, now) {
		t.Fatalf("ExecutableAtStart = %q, %v; want %q as it is now", path, fi, exe)
	}
}
