package guard

import "testing"

// TestTokenizeRecordsUnquotedGlobs pins the tokenizer half of
// GHSA-3w99-pgv4-8g55: a token carrying an UNQUOTED, unescaped `*`, `?` or `[`
// is a word bash rewrites against the working directory before exec, and the
// matcher has to know which tokens those are. Quotes strip as they are read, so
// without the record `'*'`, `"*"`, `\*` and a bare `*` all tokenised to the same
// bytes and the distinction bash acts on was gone.
func TestTokenizeRecordsUnquotedGlobs(t *testing.T) {
	cases := []struct {
		line string
		want []bool // per token; nil when no token is globbed
	}{
		{"git pus? --force origin main", []bool{false, true, false, false, false}},
		{"git push --forc? origin main", []bool{false, false, true, false, false}},
		{"git push --for[c]e origin main", []bool{false, false, true, false, false}},
		{"ls *", []bool{false, true}},
		{"git push '--forc?' origin main", nil},
		{`git push "--forc?" origin main`, nil},
		{`git push --forc\? origin main`, nil},
		{"git push $'--forc?' origin main", nil},
		// A partly quoted word is still expanded by bash: the unquoted `?` makes
		// the whole word a pattern.
		{"git push '--forc'? origin main", []bool{false, false, true, false, false}},
		{"echo plain words", nil},
	}
	for _, tc := range cases {
		seg := firstSegment(t, tc.line)
		if tc.want == nil {
			if seg.globbed != nil {
				t.Errorf("tokenize(%q).globbed = %v, want nil: nothing here is an unquoted glob", tc.line, seg.globbed)
			}
			continue
		}
		if len(seg.globbed) != len(tc.want) {
			t.Fatalf("tokenize(%q).globbed = %v, want %v", tc.line, seg.globbed, tc.want)
		}
		for i := range tc.want {
			if seg.globbed[i] != tc.want[i] {
				t.Errorf("tokenize(%q).globbed[%d] = %v, want %v (token %q)", tc.line, i, seg.globbed[i], tc.want[i], seg.tokens[i])
			}
		}
	}
}

// TestGlobSpelledHazardsMatchTheirExpansion — GHSA-3w99-pgv4-8g55. bash expands
// an unquoted glob against the working directory before the command runs, so
// `git pus? --force origin main` IS `git push --force origin main` whenever a
// file named `push` exists — and a file named for a hazard is the attacker's
// cheapest condition, in a directory the guard cannot see (the hook's cwd is
// the session's, not the post-`cd` one). The matcher compared the pattern text
// literally, so every spelling below was an allow. A glob denotes a SET of
// words, and whether an entry's literal is in that set is path.Match, decidable
// with no filesystem read: at the positions an entry constrains, a literal the
// glob CAN produce is treated as produced. Everywhere else the token is
// unconstrained, so `ls *` and `git add *.md` do not change.
func TestGlobSpelledHazardsMatchTheirExpansion(t *testing.T) {
	blocked := []struct {
		line  string
		entry string
	}{
		{"git pus? --force origin main", "git-push-force"},
		{"git p?sh --force origin main", "git-push-force"},
		{"g?t push --force origin main", "git-push-force"},
		{"git push --forc? origin main", "git-push-force"},
		{"git push --for[c]e origin main", "git-push-force"},
		// `--force*` expands to --force-with-lease, a member of the entry's own
		// flag group; the spelled-out form blocks, so this one must too.
		{"git push --force* origin main", "git-push-force"},
		{"git push '--forc'? origin main", "git-push-force"},
		{"sh -c 'git pus? --force origin main'", "git-push-force"},
		{"eval 'git push --forc? origin main'", "git-push-force"},
		// A globbed INTERPRETER name: the one lookup where a miss was silent,
		// because the payload behind it is a token nothing else opens.
		{"s? -c 'git push --force origin main'", "git-push-force"},
		{"ev?l 'git push --force origin main'", "git-push-force"},
		// The other constrained positions: subcommand2 and a flag VALUE.
		{"gh repo delet? owner/repo", "gh-repo-delete"},
		{"gh api -X DELET? repos/owner/repo", "gh-api-repo-delete"},
		{"cd scratch && rm -r? *", "rm-rf-after-cd-chain"},
	}
	for _, tc := range blocked {
		t.Run(tc.line, func(t *testing.T) {
			d := checkGuard(t, tc.line)
			if d.Verdict != VerdictBlock || d.EntryID != tc.entry {
				t.Fatalf("Check(%q) = %q via %q, want %q via %q: bash can expand this to the hazard", tc.line, d.Verdict, d.EntryID, VerdictBlock, tc.entry)
			}
		})
	}

	allowed := []string{
		"ls *",
		"git add *.md",
		"git push '--forc?' origin main",
		`git push "--forc?" origin main`,
		`git push --forc\? origin main`,
		"git push $'--forc?' origin main",
		// An unmatched `[` is not a pattern in bash: the word stays literal, and
		// git rejects it as an unknown option.
		"git push --for[ce origin main",
		// zsh's noglob turns expansion off for the command it precedes, so the
		// pattern text is what git sees, and git sees no --force in it.
		"noglob git push --forc? origin main",
		"git status --porcelain",
		// An OPERAND glob is not a flag. A flag constraint is not positional —
		// every argument is offered to it — so a bare `*` compared as a pattern
		// satisfied `--recursive`, `--force`, `--no-verify` and `--hard` all at
		// once, and five of the commonest lines anyone types became blocks.
		"cd build && rm *",
		"cd dist && rm -- *",
		"git commit -m 'fix typo' *",
		"git push origin *",
		"git reset HEAD *",
		// `--` ends the options: what follows is an operand however it is spelled.
		"git push -- --force origin main",
	}
	for _, line := range allowed {
		t.Run(line, func(t *testing.T) {
			if d := checkGuard(t, line); d.Verdict != VerdictAllow {
				t.Fatalf("Check(%q) = %q via %q, want %q: nothing here expands to a hazard", line, d.Verdict, d.EntryID, VerdictAllow)
			}
		})
	}
}
