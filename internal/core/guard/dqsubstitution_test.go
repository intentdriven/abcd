package guard

import "testing"

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
// word stays one argument of the enclosing command, its text unchanged. An unterminated
// substitution inside the quotes is left as the literal text it was.
func TestDoubleQuotedSubstitutionKeepsTheWord(t *testing.T) {
	cases := []struct {
		line string
		want []string
	}{
		{`git commit -m "at $(date) ok"`, []string{"0:date", "0:git|commit|-m|at $(date) ok"}},
		{`echo "$(a)" b`, []string{"0:a", "0:echo|$(a)|b"}},
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
