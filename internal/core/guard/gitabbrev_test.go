package guard

import (
	"strings"
	"testing"
)

// TestGitLongOptionAbbreviationsMatch — review-guard finding 4. git's option
// parser accepts any unambiguous prefix of a long option, so a push or commit
// flag spelled short of its full name runs as the full flag, while the matcher
// compared flags exactly and allowed every such spelling. A long argument now
// matches a blocked long alternative when it is a prefix of it that no other
// option of the same subcommand shares — git's own rule, read fail-closed where
// the prefix is shared only among blocked alternatives (git refuses that one as
// ambiguous, and an older git without one of them runs it).
func TestGitLongOptionAbbreviationsMatch(t *testing.T) {
	cases := []struct {
		cmd   string
		want  Verdict
		entry string
	}{
		{`git commit -m x --no-verif`, VerdictBlock, "git-commit-no-verify"},
		{`git commit --no-veri -m x`, VerdictBlock, "git-commit-no-verify"},
		{`git push --no-veri origin main`, VerdictBlock, "git-push-no-verify"},
		{`git push --force-w origin main`, VerdictBlock, "git-push-force"},
		{`git push --force-with-l origin main`, VerdictBlock, "git-push-force"},
		{`git push --force-w=main:abc origin main`, VerdictBlock, "git-push-force"},
		{`git push --force-i origin main`, VerdictBlock, "git-push-force"},
		{`git push --forc origin main`, VerdictBlock, "git-push-force"},
		{`git -C /repo push --force-w origin main`, VerdictBlock, "git-push-force"},

		{`git push --no-verb origin main`, VerdictAllow, ""},
		{`git push --no-ver origin main`, VerdictAllow, ""},
		{`git push --fo origin main`, VerdictAllow, ""},
		{`git push --follow-t origin main`, VerdictAllow, ""},
		{`git commit --verb -m x`, VerdictAllow, ""},
		{`git commit --no-post -m x`, VerdictAllow, ""},
		{`git push -- --force-w origin main`, VerdictAllow, ""},
		{`git log --no-veri`, VerdictAllow, ""},
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

// TestGitOptionTableHoldsEveryBlockedAlternative pins the table the
// abbreviation rule reads against the registry it serves: a blocked long
// alternative the table does not hold is one whose abbreviations nothing
// resolves, which is the silent gap the rule exists to close.
func TestGitOptionTableHoldsEveryBlockedAlternative(t *testing.T) {
	for id, e := range Defaults().Entries {
		if e.Tier != TierBlocker || !strings.EqualFold(e.Pattern.Command, "git") || len(e.Pattern.Flags) == 0 {
			continue
		}
		opts, ok := gitLongOptions[e.Pattern.Subcommand]
		if !ok {
			t.Errorf("entry %s blocks flags of git %s, which has no option table", id, e.Pattern.Subcommand)
			continue
		}
		for _, group := range e.Pattern.Flags {
			for _, alt := range strings.Split(group, "|") {
				if strings.HasPrefix(alt, "--") && !containsString(opts, alt) {
					t.Errorf("entry %s blocks %s, which the git %s option table does not hold", id, alt, e.Pattern.Subcommand)
				}
			}
		}
	}
}
