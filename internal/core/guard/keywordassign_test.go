package guard

import "testing"

// TestAnAssignmentAfterAKeywordIsCarried: an assignment that opens a branch, a
// loop body, a group or a negation reaches every later use, as the same
// assignment standing alone does, so the hazard its value names is judged.
func TestAnAssignmentAfterAKeywordIsCarried(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`G=echo; G=git; $G push --force origin main`, VerdictBlock, "git-push-force"},
		{`G=echo; if true; then G=git; fi; $G push --force origin main`, VerdictBlock, "git-push-force"},
		{`G=echo; if false; then :; else G=git; fi; $G push --force origin main`, VerdictBlock, "git-push-force"},
		{`G=echo; { G=git; }; $G push --force origin main`, VerdictBlock, "git-push-force"},
		{`G=echo; for x in 1; do G=git; done; $G push --force origin main`, VerdictBlock, "git-push-force"},
		{`G=echo; ! G=git; $G push --force origin main`, VerdictBlock, "git-push-force"},
		{`P=echo; if true; then P=pkill; fi; $P node`, VerdictBlock, ""},
		// A command's output assigned after a keyword is not carried: the
		// name keeps the reading it had, so abcd's own hook scripts, which
		// pick their binary in a branch, are not blocked.
		{`x=""; if true; then x=$(command -v abcd); fi; [ -z "$x" ] || "$x" guard hook`, VerdictAllow, ""},
	})
}
