package guard

import "testing"

// bash imports a function from any environment variable named
// BASH_FUNC_<name>%% whose value starts `() {`, so `env
// 'BASH_FUNC_true%%=() { …; }' bash -c true` runs the body when the -c
// string calls true (iss-2610090925399967). `env` treats every operand that
// carries `=` as an assignment, but the walk stepped only identifier-named
// ones, so it read the function word as the program and judged nothing in
// it. The walk now steps every `=` operand of env (and sudo, which takes
// VAR=value the same way), and an exported function's body is judged with
// the inline rules. Bash itself does not take such a word as a prefix
// assignment (`%` is not a name character), so only the wrapper forms run.
func TestExportedFunctionBodyIsJudged(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{"env 'BASH_FUNC_true%%=() { " + hazardLine + "; }' bash -c true", VerdictBlock, "git-push-force"},
		{"env -i 'BASH_FUNC_ls%%=() { " + hazardLine + "; }' bash -c ls", VerdictBlock, "git-push-force"},
		{"env FOO=1 'BASH_FUNC_f%%=() { " + hazardLine + "; }' bash -c f", VerdictBlock, "git-push-force"},
		{"sudo 'BASH_FUNC_true%%=() { " + hazardLine + "; }' bash -c true", VerdictBlock, "git-push-force"},
		// The program after a non-identifier assignment is the command: a
		// blocker there was read as an argument of the assignment word.
		{"env 'A-B=1' " + hazardLine, VerdictBlock, "git-push-force"},
		{"env 'BASH_FUNC_f%%=() { echo hi; }' bash -c f", VerdictAllow, ""},
		{"env FOO=1 bash -c true", VerdictAllow, ""},
		{"env 'A-B=1' git status", VerdictAllow, ""},
	})
}
