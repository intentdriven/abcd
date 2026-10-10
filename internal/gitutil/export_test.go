package gitutil

import "testing"

// SetGitVersionSource replaces the `git version` probe the lazy-fetch floor
// reads for the length of t, clearing the cached answer on both sides.
func SetGitVersionSource(t *testing.T, src func() (string, error)) {
	t.Helper()
	gitVersion.set(src)
	t.Cleanup(func() { gitVersion.set(probeGitVersion) })
}
