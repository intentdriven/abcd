package gittest

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNeutralCallerEndsHOMEInTheNeutralAccount: the HOME a neutral caller
// gets ends in NeutralAccount, the segment the identity scanner reads as the
// account name, and is the test's own, so Env keeps it rather than minting
// another.
func TestNeutralCallerEndsHOMEInTheNeutralAccount(t *testing.T) {
	home := NeutralCaller(t)
	if got := os.Getenv("HOME"); got != home {
		t.Fatalf("HOME = %q, want the neutral home %q", got, home)
	}
	if filepath.Base(home) != NeutralAccount {
		t.Fatalf("the neutral home %q does not end in %q", home, NeutralAccount)
	}
	if fi, err := os.Stat(home); err != nil || !fi.IsDir() {
		t.Fatalf("the neutral home is not a directory: %v", err)
	}
	Env(t)
	if got := os.Getenv("HOME"); got != home {
		t.Fatalf("Env replaced the neutral home %q with %q", home, got)
	}
}
