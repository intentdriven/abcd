package gittest

import "os"

// ghTokenVars are the variables gh takes a token from ahead of its own
// configuration.
var ghTokenVars = []string{"GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN"}

// HideGHLogin points gh, for the rest of the process, at an empty
// configuration directory and clears the token variables, so no test reaches
// the machine's own GitHub login. A test that did would pass on a developer's
// machine and fail on CI, which has no login: a landing test read the
// account behind an announcement that way and passed preflight, then failed
// in the merge queue. A package whose tests can reach gh calls it from
// TestMain before m.Run and runs the returned cleanup after; a test that
// wants a login of its own still sets one with t.Setenv.
func HideGHLogin() (cleanup func(), err error) {
	dir, err := os.MkdirTemp("", "abcd-test-gh-config-")
	if err != nil {
		return nil, err
	}
	for _, k := range ghTokenVars {
		if err := os.Unsetenv(k); err != nil {
			_ = os.RemoveAll(dir)
			return nil, err
		}
	}
	if err := os.Setenv("GH_CONFIG_DIR", dir); err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	return func() { _ = os.RemoveAll(dir) }, nil
}
