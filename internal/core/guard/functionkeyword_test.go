package guard

import "testing"

// `function NAME { …; }` defines a function exactly as `NAME() { …; }` does,
// and the body runs when the line calls it (iss-2610090821313095). The
// keyword form kept `function` and the name in front of the body in one
// segment, so the walk read `function` as the program and the body's
// blocker as its arguments: an unrecognised-launcher warn, which the hook
// lets run, where the POSIX form's body blocked. The walk now steps over the
// keyword and the name, as it steps over `coproc NAME`, and the body is
// judged like the other function forms.
func TestFunctionKeywordBodyIsJudged(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{"function f { " + hazardLine + "; }; f", VerdictBlock, "git-push-force"},
		{"function f {\n" + hazardLine + "\n}\nf", VerdictBlock, "git-push-force"},
		{"function f\n{\n" + hazardLine + "\n}\nf", VerdictBlock, "git-push-force"},
		{"function f() { " + hazardLine + "; }; f", VerdictBlock, "git-push-force"},
		{"function f () { " + hazardLine + "; }; f", VerdictBlock, "git-push-force"},
		{"function f ( " + hazardLine + " ); f", VerdictBlock, "git-push-force"},
		{"function f { if true; then " + hazardLine + "; fi; }; f", VerdictBlock, "git-push-force"},
		{"function f if true; then " + hazardLine + "; fi; f", VerdictBlock, "git-push-force"},
		{"f() { " + hazardLine + "; }; f", VerdictBlock, "git-push-force"},
		{"bash -c 'function f { " + hazardLine + "; }; f'", VerdictBlock, "git-push-force"},
		// zsh's `repeat COUNT` takes one word before the command, as
		// `function` takes its NAME (sibling sweep).
		{"repeat 3 " + hazardLine, VerdictBlock, "git-push-force"},
		{"zsh -c 'repeat 3 { " + hazardLine + "; }'", VerdictBlock, "git-push-force"},
		{"repeat 3 echo hi", VerdictAllow, ""},
		{"function f { git status; }; f", VerdictAllow, ""},
		{"function f { echo hi; }", VerdictAllow, ""},
	})
}
