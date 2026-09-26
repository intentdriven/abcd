package guard

import "testing"

// TestHooksPathOverrideIsNoVerify — review-guard finding 6. Pointing
// core.hooksPath somewhere else for one git command skips the repository's
// hooks exactly as --no-verify does, and the matcher stepped the `-c` value
// over unread, so the no-verify entries never saw it. A commit or push that
// sets the key in its own command line — through `-c`, `--config-env`, or the
// GIT_CONFIG_* environment — is now read as carrying --no-verify, and blocks
// under the same entries. Any value counts: the guard cannot tell a directory
// of real hooks from an empty one, and the repository's own hooks are the ones
// the entry protects.
func TestHooksPathOverrideIsNoVerify(t *testing.T) {
	cases := []struct {
		cmd   string
		want  Verdict
		entry string
	}{
		{`git -c core.hooksPath=/dev/null commit -m x`, VerdictBlock, "git-commit-no-verify"},
		{`git -c core.hooksPath=/dev/null push origin main`, VerdictBlock, "git-push-no-verify"},
		{`git -c CORE.HOOKSPATH=/tmp/none commit -m x`, VerdictBlock, "git-commit-no-verify"},
		{`git -C /repo -c core.hooksPath= commit -m x -- a.txt`, VerdictBlock, "git-commit-no-verify"},
		{`GIT_CONFIG_PARAMETERS="'core.hooksPath=/dev/null'" git commit -m x`, VerdictBlock, "git-commit-no-verify"},
		{`GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=core.hooksPath GIT_CONFIG_VALUE_0=/dev/null git commit -m x`, VerdictBlock, "git-commit-no-verify"},
		{`git --config-env=core.hooksPath=HOOKS commit -m x`, VerdictBlock, "git-commit-no-verify"},
		{`git -c core.hooksPath=/dev/null -c alias.c=commit c -m x`, VerdictBlock, "git-commit-no-verify"},
		{`sh -c 'git -c core.hooksPath=/dev/null commit -m x'`, VerdictBlock, "git-commit-no-verify"},

		{`git -c core.hooksPath=/dev/null status`, VerdictAllow, ""},
		{`git -c core.hooksPath=/dev/null log --oneline`, VerdictAllow, ""},
		{`git -c user.name=x commit -m x`, VerdictAllow, ""},
		{`git commit -m "set core.hooksPath=/dev/null"`, VerdictAllow, ""},
		{`git config core.hooksPath .githooks`, VerdictAllow, ""},
	}
	for _, tc := range cases {
		t.Run(tc.cmd, func(t *testing.T) {
			d := verdictOf(t, tc.cmd)
			if d.Verdict != tc.want || d.EntryID != tc.entry {
				t.Errorf("verdict = %q via %q, want %q via %q", d.Verdict, d.EntryID, tc.want, tc.entry)
			}
		})
	}
}
