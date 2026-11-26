package guard

import (
	"strings"
	"testing"
)

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
		{"function git { :; }", VerdictAllow, ""},
		// The word `function` outside command position is an argument.
		{"grep -rn function src", VerdictAllow, ""},
		{"echo function 'f' g {", VerdictAllow, ""},
		{"git log --grep function", VerdictAllow, ""},
	})
}

// TestFunctionInAnUnusualFormIsHeld: the guard reads a function written as
// `NAME() BODY` or `function NAME` with one plain name and an unquoted `{`, or
// a body bash reads (dcn-2611260046000744). Every other use of the keyword at
// command position is held, harmless body or not, so no reading of the shell's
// wider function grammar is needed.
func TestFunctionInAnUnusualFormIsHeld(t *testing.T) {
	const held = "function-form-unread"
	var cases []verdictCase
	for _, line := range []string{
		"function f g { " + hazardLine + "; }; f",
		"function -T f { " + hazardLine + "; }; f",
		"function f g h if true; then echo hi; fi; f",
		"function if g { " + hazardLine + "; }; g",
		"function f if g { " + hazardLine + "; }; f",
		"function ! f { " + hazardLine + "; }; f",
		"function f do g { " + hazardLine + "; }; g",
		"function f while g { " + hazardLine + "; }; g",
		"function f if " + hazardLine + " {; then true; fi; f",
		"function f if echo { ; then echo hi; fi; f",
		"function f g () { echo hi; }; f",
		"function '{' g { " + hazardLine + "; }; g",
		`function "{" g { ` + hazardLine + "; }; g",
		`function \{ g { ` + hazardLine + "; }; g",
		"function f '{' g { " + hazardLine + "; }; g",
		"function f g >/dev/null " + hazardLine + "; f",
		"function f g 2>&1 " + hazardLine + "; f",
		"function 'f' { echo hi; }",
		"function f g { echo hi; }",
		"function",
		"if true; then function f g { echo hi; }; fi",
		"function f { function g h { " + hazardLine + "; }; }; f; g",
		"time function f '{' g { " + hazardLine + "; }; g",
		"bash -c 'function f g { " + hazardLine + "; }; f'",
	} {
		cases = append(cases, verdictCase{line, VerdictBlock, held})
	}
	runVerdictCases(t, cases)
	// Where the body's hazard is a command of its own, the hold stands beside
	// the hazard's entry.
	for _, line := range []string{
		"function f g h if true; then " + hazardLine + "; fi; f",
		"function f if echo { ; then " + hazardLine + "; fi; f",
		"function f g () { " + hazardLine + "; }; f",
	} {
		if d := readingOf(t, line); d.Verdict != VerdictBlock || !containsString(d.Matches, held) {
			t.Errorf("Check(%q) = %q, matches %v, want a block holding %q", line, d.Verdict, d.Matches, held)
		}
	}
}

// TestFunctionFormStaysLinear: a function the guard does not read is held
// where the walk meets it, so a line of them costs one pass.
func TestFunctionFormStaysLinear(t *testing.T) {
	build := func(n int) string { return strings.Repeat("function f if a { ", n) }
	assertWorkGrowth(t, build, 903, "a function in an unusual form is held where the walk meets it, never walked past")
	// The work count above does not see what each place the walk arrives at
	// costs the entries after it; the arrivals are counted here.
	segs, err := tokenize(build(3610))
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range segs {
		if out, _, _ := walkToCommand(s.tokens); len(out) > 1 {
			t.Errorf("the walk arrives at %d places in a %d-word segment, want one at most", len(out), len(s.tokens))
		}
	}
}
