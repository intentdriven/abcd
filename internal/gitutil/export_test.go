package gitutil

import "testing"

// SetGitVersionSource replaces the `git version` probe (the lazy-fetch floor
// and the merge driver's options both read it) for the length of t, clearing
// the cached answer on both sides.
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

// RemoteHeadFrom is RemoteHeadBranch's reading of an ls-remote answer.
var RemoteHeadFrom = remoteHeadFrom

// NetEnvFor is netEnv with the test-binary question answered by underTest.
var NetEnvFor = netEnvFor

// AllowNetTransportsForTest switches the test-binary transport guard off for
// the length of t, for a test that drives a transport through a fake.
func AllowNetTransportsForTest(t *testing.T) {
	t.Helper()
	netGuardOff = true
	t.Cleanup(func() { netGuardOff = false })
}

// ColdGitVersionForTest clears the cached `git version` answer and puts the
// real probe back, so the next guard asks git again, for the length of t.
func ColdGitVersionForTest(t *testing.T) {
	t.Helper()
	gitVersion.set(probeGitVersion)
	t.Cleanup(func() { gitVersion.set(probeGitVersion) })
}
