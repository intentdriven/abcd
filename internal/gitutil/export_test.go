package gitutil

import "testing"

// SetGitVersionSource replaces the `git version` probe the lazy-fetch floor
// reads for the length of t, clearing the cached answer on both sides.
func SetGitVersionSource(t *testing.T, src func() (string, error)) {
	t.Helper()
	gitVersion.set(src)
	t.Cleanup(func() { gitVersion.set(probeGitVersion) })
}

// SetAccountPasswdForTest makes the account-home lookup HomeInsideWorkingTree
// exempts read the passwd-format file at path, the route every platform but
// darwin takes, for the length of t.
func SetAccountPasswdForTest(t *testing.T, path string) {
	t.Helper()
	prev := accountSource
	accountSource.osUser = false
	accountSource.passwd = path
	t.Cleanup(func() { accountSource = prev })
}
