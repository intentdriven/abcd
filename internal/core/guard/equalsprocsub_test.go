package guard

import "testing"

// TestZshEqualsProcessSubstitutionIsAStream — iss-2610111559070779. zsh's
// `=(…)` is process substitution through a temporary file: a word-initial
// `=(` runs its command and hands the command the file's name, so `source`
// or a shell pointed at one runs the substitution's output as a script, the
// same stream `<(…)` is. The tokenizer read the `=` as a plain word and the
// line was an allow.
func TestZshEqualsProcessSubstitutionIsAStream(t *testing.T) {
	dir := scriptTree(t, map[string]string{"evil.sh": hazardLine})
	runScriptCases(t, dir, []scriptCase{
		// Both zsh spellings, the stream's text inside a command that is not
		// itself in the registry.
		{`zsh -c 'source =(echo "` + hazardLine + `")'`, VerdictBlock, interpreterStreamEntryID},
		{`zsh -c 'source =(cat {d}/evil.sh)'`, VerdictBlock, interpreterStreamEntryID},
		{`source =(echo "` + hazardLine + `")`, VerdictBlock, interpreterStreamEntryID},
		{`. =(cat {d}/evil.sh)`, VerdictBlock, interpreterStreamEntryID},
		{`zsh =(cat {d}/evil.sh)`, VerdictBlock, interpreterStreamEntryID},
		// The `<(…)` control stays a block.
		{`source <(echo "` + hazardLine + `")`, VerdictBlock, interpreterStreamEntryID},
		// The substitution's own command is still read at command position.
		{`source =(` + hazardLine + `)`, VerdictBlock, "git-push-force"},
		// An array assignment's `=(` is not word-initial: no substitution.
		{`a=(1 2)`, VerdictAllow, ""},
		{`a=(1 2); echo "${a[@]}"`, VerdictAllow, ""},
		{`zsh -c 'fpath=(/usr/share/zsh/functions); echo ok'`, VerdictAllow, ""},
		// A substitution handed to a command that is not a shell is a file it
		// reads, as `diff <(…) <(…)` is.
		{`diff =(sort a) =(sort b)`, VerdictAllow, ""},
	})
}
