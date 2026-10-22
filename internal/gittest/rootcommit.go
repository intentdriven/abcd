package gittest

import (
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// RootCommit is gitutil.RootCommit for a fixture: the root commit of the
// repository at dir, "" for none, and a failed test when git did not answer.
// A fixture's identity is never in doubt, so a lookup that errors is the test
// machine's fault and is said as such rather than read as "no root commit".
func RootCommit(t *testing.T, dir string) string {
	t.Helper()
	sha, err := gitutil.RootCommit(dir)
	if err != nil {
		t.Fatalf("gittest: the root commit of %s: %v", dir, err)
	}
	return sha
}
