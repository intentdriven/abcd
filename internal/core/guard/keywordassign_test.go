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
		{`G=echo; while G=git; do break; done; $G push --force origin main`, VerdictBlock, "git-push-force"},
		{`G=echo; until G=git; do :; done; $G push --force origin main`, VerdictBlock, "git-push-force"},
		{`G=echo; time G=git; $G push --force origin main`, VerdictBlock, "git-push-force"},
		{`G=echo; repeat 1 G=git; $G push --force origin main`, VerdictBlock, "git-push-force"},
		{`G=echo; time -p G=git; $G push --force origin main`, VerdictBlock, "git-push-force"},
		{`G=echo; time -p -- G=git; $G push --force origin main`, VerdictBlock, "git-push-force"},
		// A command's output assigned after a keyword is carried as output,
		// as it is standing alone.
		{`P=echo; if true; then P=$(echo pkill); fi; $P node`, VerdictBlock, "program-name-unknown"},
		{`if true; then p=$(pgrep make); fi; kill $p`, VerdictBlock, "kill-by-search"},
		{`if true; then P=$(cat f); fi; eval "$P"`, VerdictBlock, ""},
		// A lookup of a program's path names that program, in a branch too.
		{`x=""; if true; then x=$(command -v abcd); fi; [ -z "$x" ] || "$x" guard hook`, VerdictAllow, ""},
	})
}
