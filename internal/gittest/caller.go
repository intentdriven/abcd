package gittest

import (
	"os"
	"path/filepath"
	"testing"
)

// NeutralAccount is the account name a neutral caller's HOME ends in: a word no
// committed content carries.
const NeutralAccount = "zzneutralcaller"

// NeutralCaller points HOME at a fresh test-owned directory named for
// NeutralAccount and returns it, so a test that scans content judges the
// content and not whoever runs the suite. The identity scanner takes the
// caller's account name from HOME's last segment and flags every word-bounded
// occurrence of it, so an account named for a word the content holds (the
// cloud setup's `abcd`, which is this product's own name) fails a scan of a
// clean tree. Env reuses this HOME, since it is inside the test's temp area.
func NeutralCaller(t *testing.T) string {
	t.Helper()
	home, err := NeutralHomeIn(t.TempDir())
	if err != nil {
		t.Fatalf("gittest: %v", err)
	}
	t.Setenv("HOME", home)
	return home
}

// NeutralHomeIn makes the directory NeutralCaller points HOME at, inside dir,
// for a TestMain that sets HOME for a whole package (it has no t to set it
// through, and removes dir itself). It is the account's folder, which HOME
// names, not abcd's home inside it: the code under test makes that one, at
// abcdhome.DirMode.
func NeutralHomeIn(dir string) (string, error) {
	account := filepath.Join(dir, NeutralAccount)
	if err := os.Mkdir(account, 0o700); err != nil {
		return "", err
	}
	return account, nil
}
