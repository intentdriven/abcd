package guard

import "testing"

// TestAStringsQuotingAppliesToTheVariablesValue — review-drainG3 finding 1
// (iss-2609251824244354). A string the guard reads as a payload carries a
// variable the enclosing shell has already expanded: bash hands the inner
// shell `git push '--force'`, and the string's own quote or backslash applies
// to the VALUE. The payload reading spelled the variable back as `$X` text
// and re-read it, so the same quote applied to the NAME there, and `'--$X'`
// read as the literal `--$X`, which names no flag. A variable's value now
// reaches the re-read as its own mark (varMark), which no quote or escape
// turns back into text, as a substitution's output reaches it.
func TestAStringsQuotingAppliesToTheVariablesValue(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`sh -c "git push '--$X' origin main"`, VerdictBlock, "git-push-force"},
		{`sh -c "git push -\\$F origin main"`, VerdictBlock, "git-push-force"},
		{`sh -c "git push --for\\$X origin main"`, VerdictBlock, "git-push-force"},
		{`eval "git push '--$X' origin main"`, VerdictBlock, "git-push-force"},
		{`bash -c "git push '-$F' origin main"`, VerdictBlock, "git-push-force"},
		{`sh -c "git commit -m x '--$X'"`, VerdictBlock, "git-commit-no-verify"},
		{`sh -c "cd s && rm '-$F' *"`, VerdictBlock, "rm-rf-after-cd-chain"},
		{`sh -c "'$GIT' push --force origin main"`, VerdictBlock, ""},
		{`sh -c "git push '--${X}' origin main"`, VerdictBlock, "git-push-force"},
		{`sh -c "git push $'--$X' origin main"`, VerdictBlock, "git-push-force"},
		{`sh -c "sh -c \"git push '--$X' origin main\""`, VerdictBlock, "git-push-force"},
		{`env -S "git push --$X origin main"`, VerdictBlock, ""},

		// What the review's twins already read, and still do.
		{`sh -c "git push --$X origin main"`, VerdictBlock, "git-push-force"},
		{`sh -c "git push \"--$X\" origin main"`, VerdictBlock, "git-push-force"},
		{`sh -c "git push '--$(echo force)' origin main"`, VerdictBlock, "git-push-force"},

		// A word that is wholly a variable is one operand (DECISIONS
		// 2026-09-28, allow (1) of the first 2026-09-25 entry), and a quote or
		// a backslash round it in a string leaves it one: `\$X` there is its
		// bare twin `sh -c "git push $X origin main"`, as `git push $X origin
		// main` is at the top level.
		{`sh -c "git push \\$X origin main"`, VerdictAllow, ""},
		{`sh -c "git push '$X' origin main"`, VerdictAllow, ""},
		{`sh -c "git push $X origin main"`, VerdictAllow, ""},

		// A variable in a string is still read as the carve-outs read it at
		// the top level, and a quoted one in operand position is an operand.
		{`sh -c "git push origin '$BRANCH'"`, VerdictAllow, ""},
		{`sh -c "git push origin \\$BRANCH"`, VerdictAllow, ""},
		{`sh -c "echo '$HOME'"`, VerdictAllow, ""},
		{`bash -c "cd '$DIR' && make test"`, VerdictAllow, ""},
		{`sh -c "curl -o '$OUT' https://example.com/x"`, VerdictAllow, ""},
		{`sh -c "git push origin main" '$X'`, VerdictAllow, ""},
		{`sh -c 'git push "--$X" origin main'`, VerdictBlock, "git-push-force"},

		// An escape that decodes to the mark's byte is that byte, not a
		// variable (readAnsiCQuote).
		{`git push $'--\x01' origin main`, VerdictAllow, ""},
		{`git push $'--\001' origin main`, VerdictAllow, ""},
	})
}
