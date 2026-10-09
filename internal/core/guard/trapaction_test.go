package guard

import "testing"

// `trap ACTION SIGNAL…` stores ACTION and the shell runs it as a command line
// when the signal arrives, EXIT included, which every shell raises when it
// ends (iss-2610090821476887). The guard read ACTION as a plain argument, so a
// blocker in it ran at exit although `eval` of the same text blocked. ACTION
// is now read the way eval's arguments are: a readable string is judged, a
// substitution the guard cannot read keeps eval's verdict, and the forms that
// only reset or list (`trap - SIG`, `trap SIG`, `trap 0 1`, `trap -p`) carry
// no command. `mapfile -C CALLBACK` (and `readarray`) evaluates CALLBACK the
// same way every few lines, so it is read the same way (sibling sweep).
func TestTrapActionIsJudged(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{"trap '" + hazardLine + "' EXIT", VerdictBlock, "git-push-force"},
		{"trap \"" + hazardLine + "\" 0", VerdictBlock, "git-push-force"},
		{"trap -- '" + hazardLine + "' INT TERM", VerdictBlock, "git-push-force"},
		{"trap '" + hazardLine + "' EXIT; echo done", VerdictBlock, "git-push-force"},
		{"bash -c \"trap '" + hazardLine + "' EXIT; true\"", VerdictBlock, "git-push-force"},
		{"builtin trap '" + hazardLine + "' EXIT", VerdictBlock, "git-push-force"},
		{"mapfile -C '" + hazardLine + "' -c 1 arr < /dev/null", VerdictBlock, "git-push-force"},
		{"readarray -C'" + hazardLine + "' arr < /dev/null", VerdictBlock, "git-push-force"},
		{"mapfile -$(echo C) '" + hazardLine + "' arr < /dev/null", VerdictBlock, "git-push-force"},
		{"trap 'echo bye' EXIT", VerdictAllow, ""},
		{"trap - EXIT", VerdictAllow, ""},
		{"trap EXIT", VerdictAllow, ""},
		{"trap 0 1 2", VerdictAllow, ""},
		{"trap '' INT", VerdictAllow, ""},
		{"trap -p", VerdictAllow, ""},
		{"trap -l", VerdictAllow, ""},
		{"mapfile -t arr < /dev/null", VerdictAllow, ""},
		{"mapfile -C 'echo row' -c 1 arr < /dev/null", VerdictAllow, ""},
	})
}

// A trap action the guard cannot read keeps the verdict eval gives the same
// operand, never a quieter one (coordinator ruling: match eval, not block).
func TestTrapActionMatchesEvalOnUnreadableText(t *testing.T) {
	for _, op := range []string{`"$(cat f)"`, `"$ACTION"`} {
		e := verdictOf(t, "eval "+op)
		tr := verdictOf(t, "trap "+op+" EXIT")
		if tr.Verdict != e.Verdict {
			t.Errorf("trap %s EXIT = %q, eval %s = %q; want the same verdict", op, tr.Verdict, op, e.Verdict)
		}
	}
}
