package guard

import (
	"strings"
	"testing"
)

// TestDoubleQuotedSubstitutionIsFollowed — iss-2609251144159533. A command
// substitution inside double quotes runs exactly as an unquoted one does, and
// double-quoting it is the idiomatic spelling, but the double-quote branch of
// the tokenizer read the whole string as one literal word: `echo "$(gh repo
// delete owner/repo)"` answered allow while its unquoted twin blocked. The
// substitution's command is now its own segment; the quoted word keeps its
// place, and its text, in the enclosing command.
func TestDoubleQuotedSubstitutionIsFollowed(t *testing.T) {
	cases := []struct {
		cmd  string
		want Verdict
	}{
		{`echo "$(gh repo delete owner/repo)"`, VerdictBlock},
		{`x="$(git push --force origin main)"`, VerdictBlock},
		{"echo \"before `gh repo delete owner/repo` after\"", VerdictBlock},
		{`echo "$(echo "$(gh repo delete owner/repo)")"`, VerdictBlock},
		{`cd s && echo "$(rm -rf *)"`, VerdictBlock},
		{`echo "a ) b $(gh repo delete owner/repo) c"`, VerdictBlock},

		{`git commit -m "$(date)" --allow-empty`, VerdictAllow},
		{`echo "$(echo ")")"`, VerdictAllow},
		{`echo "\$(gh repo delete owner/repo)"`, VerdictAllow},
		{`echo "the text $(gh repo list) mentions gh repo delete"`, VerdictAllow},
		{`echo '$(gh repo delete owner/repo)'`, VerdictAllow},
	}
	for _, tc := range cases {
		t.Run(tc.cmd, func(t *testing.T) {
			d, err := Defaults().Check(tc.cmd)
			if err != nil {
				t.Fatalf("Check(%q): %v", tc.cmd, err)
			}
			if d.Verdict != tc.want {
				t.Errorf("Check(%q) = %q via %q, want %q", tc.cmd, d.Verdict, d.EntryID, tc.want)
			}
		})
	}
}

// TestDoubleQuotedSubstitutionKeepsTheWord pins the tokenizer shape: the inner
// command is emitted first, in the enclosing command's chain, and the quoted
// word stays one argument of the enclosing command, holding unknownMark where
// the substitution's output goes (unknown.go) — so its known text is the vanish
// reading (iss-2609251640353993) and its dash-led spelling an unknown flag. An
// unterminated substitution inside the quotes is left as the literal text it
// was.
func TestDoubleQuotedSubstitutionKeepsTheWord(t *testing.T) {
	cases := []struct {
		line string
		want []string
	}{
		{`git commit -m "at $(date) ok"`, []string{"0:date", "0:git|commit|-m|at \x00 ok"}},
		{`echo "$(a)" b`, []string{"0:a", "0:echo|\x00|b"}},
		{"cd s && rm \"$(a)\"-rf x\necho \"`b`\"", []string{"0:cd|s", "0:a", "0:rm|\x00-rf|x", "1:b", "1:echo|\x00"}},
		{`echo "$(( (1+2) * 3 ))"`, []string{"0:echo|0"}},
		{`echo "$(unterminated"`, []string{"0:echo|$(unterminated"}},
	}
	for _, tc := range cases {
		segs, err := tokenize(tc.line)
		if err != nil {
			t.Fatalf("tokenize(%q): %v", tc.line, err)
		}
		got := render(segs)
		if len(got) != len(tc.want) {
			t.Errorf("tokenize(%q) = %q, want %q", tc.line, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("tokenize(%q) = %q, want %q", tc.line, got, tc.want)
				break
			}
		}
	}
}

// TestFollowedQuotedSubstitutionGluesNoText — review-guard finding 1. bash
// joins a double-quoted substitution's output onto the text beside it in the
// same word, and an empty output leaves exactly that text: a flag glued after,
// or split around, an empty quoted substitution is the flag. The quoted word
// kept the substitution's literal text instead, so the flag compare saw
// `$(true)--force` and every blocker allowed, while the unquoted twin and an
// empty single-quoted pair blocked. The word holds unknownMark where the
// output goes (unknown.go), and its known text is that vanish reading.
func TestFollowedQuotedSubstitutionGluesNoText(t *testing.T) {
	cases := []struct {
		cmd   string
		want  Verdict
		entry string
	}{
		{`git push "$(true)"--force origin main`, VerdictBlock, "git-push-force"},
		{`git push --for"$(:)"ce origin main`, VerdictBlock, "git-push-force"},
		{`git push "$(true)"-f`, VerdictBlock, "git-push-force"},
		{`git push "$(true)--force" origin main`, VerdictBlock, "git-push-force"},
		{"git push \"`true`\"--force origin main", VerdictBlock, "git-push-force"},
		{`gh repo "$(true)"delete o/r`, VerdictBlock, "gh-repo-delete"},
		{`cd s && rm "$(true)"-rf *`, VerdictBlock, "rm-rf-after-cd-chain"},
		{`git commit -m x "$(true)"--no-verify`, VerdictBlock, "git-commit-no-verify"},
		{`git push origin "$(true)"+main:main`, VerdictBlock, "git-push-force-refspec"},
		{`sh -c "echo $(date); git push --force origin main"`, VerdictBlock, "git-push-force"},
		{`sh -c "$(echo hi)"`, VerdictWarn, syntheticEntryID},

		{`git commit -m "at $(date) ok"`, VerdictAllow, ""},
		{`echo "$(date)"--force`, VerdictAllow, ""},
		{`git push origin "$(git branch --show-current)"`, VerdictAllow, ""},
		{`git push origin "$(git branch --show-current)":main`, VerdictAllow, ""},
	}
	for _, tc := range cases {
		t.Run(tc.cmd, func(t *testing.T) {
			d := verdictOf(t, tc.cmd)
			if d.Verdict != tc.want || (tc.entry != "" && d.EntryID != tc.entry) {
				t.Errorf("verdict = %q via %q, want %q via %q", d.Verdict, d.EntryID, tc.want, tc.entry)
			}
		})
	}
}

// TestQuotedSubstitutionStaysLinear pins the cost of following quoted
// substitutions: each double-quote level re-reads only its own text, so a line
// of many quoted substitutions, each nested to the depth budget, still costs
// work linear in its length.
func TestQuotedSubstitutionStaysLinear(t *testing.T) {
	build := func(n int) string {
		return strings.Repeat(`x "$(a)"b `+nestQuoted("y", maxQuotedSubstitutionDepth)+"; ", n)
	}
	assertWorkGrowth(t, build, 1<<9, "each quoted level reads its own text, never the line once per substitution")
}

// nestQuoted wraps inner in n levels of `echo "$( … )"`.
func nestQuoted(inner string, n int) string {
	return strings.Repeat(`echo "$(`, n) + inner + strings.Repeat(`)"`, n)
}

// TestQuotedSubstitutionPastTheDepthFailsClosed — review-guard finding 2.
// Past maxQuotedSubstitutionDepth the tokenizer stops following substitutions
// nested inside double quotes, and the text it stopped at was left literal:
// nine nested levels around a force push allowed while eight blocked, and bash
// runs the innermost command either way. What the guard stops reading is now a
// fail-closed block under a reserved id, the brace-group and here-document
// precedent; within the depth nothing changes.
func TestQuotedSubstitutionPastTheDepthFailsClosed(t *testing.T) {
	hazard := `git push --force origin main`
	for _, n := range []int{maxQuotedSubstitutionDepth + 1, maxQuotedSubstitutionDepth + 4} {
		d := verdictOf(t, nestQuoted(hazard, n))
		if d.Verdict != VerdictBlock {
			t.Errorf("%d nested levels around a force push: verdict %q via %q, want block", n, d.Verdict, d.EntryID)
		}
		d = verdictOf(t, nestQuoted("echo hi", n))
		if d.Verdict != VerdictBlock || d.EntryID != substitutionEntryID {
			t.Errorf("%d nested levels: verdict %q via %q, want the fail-closed block via %q", n, d.Verdict, d.EntryID, substitutionEntryID)
		}
	}
	if d := verdictOf(t, nestQuoted(hazard, maxQuotedSubstitutionDepth)); d.Verdict != VerdictBlock || d.EntryID != "git-push-force" {
		t.Errorf("at the depth: verdict %q via %q, want block via git-push-force", d.Verdict, d.EntryID)
	}
	if d := verdictOf(t, nestQuoted("echo hi", maxQuotedSubstitutionDepth)); d.Verdict != VerdictAllow {
		t.Errorf("a harmless nest within the depth: verdict %q via %q, want allow", d.Verdict, d.EntryID)
	}
	if _, isEntry := Defaults().Entries[substitutionEntryID]; isEntry {
		t.Errorf("the reserved id %q must never be a registry entry", substitutionEntryID)
	}
}
