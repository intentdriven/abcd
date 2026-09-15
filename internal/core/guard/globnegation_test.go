package guard

import "testing"

// TestBashNegatedClassGlobMatches pins the one dialect difference between bash's
// patterns and path.Match's that changes an answer: bash negates a character
// class with `[!…]`, path.Match with `[^…]`, and path.Match reads bash's `!` as
// an ordinary member of the set. So `git clea[!x] -fd` — which bash expands to
// `git clean -fd` whenever a file called `clean` is there — was compared as "clea
// followed by `!` or `x`", matched nothing, and allowed.
//
// The last two rows are what stops the translation over-reaching: a class that
// genuinely excludes the literal must still not match, and an unconstrained
// position is never compared at all.
func TestBashNegatedClassGlobMatches(t *testing.T) {
	cases := []struct {
		line    string
		verdict Verdict
	}{
		{"git clea[!x] -fd", VerdictWarn},
		{"git clean -f[!x]", VerdictWarn},
		{"git push --forc[!x] origin main", VerdictBlock},
		{"git clea[!n] -fd", VerdictAllow},
		{"git add [!x].md", VerdictAllow},
	}
	for _, tc := range cases {
		d, err := Defaults().Check(tc.line)
		if err != nil {
			t.Fatalf("Check(%q): %v", tc.line, err)
		}
		if d.Verdict != tc.verdict {
			t.Errorf("Check(%q) = %q via %q, want %q", tc.line, d.Verdict, d.EntryID, tc.verdict)
		}
	}
}

// TestGlobSpelledGitReachesTheAliasPrePass pins that a glob-spelled `git`
// reaches the in-process alias pre-pass, so the pre-pass and matchSegment's
// command compare cannot disagree about what the word is. They disagreed on the
// allow side: matchSegment read `g?t` as the pattern it is, while the pre-pass
// compared it with `git` literally and built no rewrite, so the alias the
// command line declared went unread.
func TestGlobSpelledGitReachesTheAliasPrePass(t *testing.T) {
	const line = "g?t -c alias.p='push --force' p origin main"
	d, err := Defaults().Check(line)
	if err != nil {
		t.Fatalf("Check(%q): %v", line, err)
	}
	if d.Verdict != VerdictBlock || d.EntryID != "git-push-force" {
		t.Fatalf("Check(%q) = %q via %q, want block via git-push-force", line, d.Verdict, d.EntryID)
	}
	const benign = "noglob g?t -c alias.st=status st"
	if d, err := Defaults().Check(benign); err != nil || d.Verdict != VerdictAllow {
		t.Fatalf("Check(%q) = %q (err %v), want allow — noglob withholds the expansion", benign, d.Verdict, err)
	}
}
