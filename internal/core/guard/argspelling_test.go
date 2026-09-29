package guard

import (
	"strings"
	"testing"
)

// TestArgValuesReadAVariableAsWritten — iss-2609290321312087. The tokenizer
// writes a parameter expansion as the unknown word's mark (unknown.go), so an
// operand's known text no longer holds `$HOME` or `$PWD`: `rm -rf $HOME` read
// as an empty operand and allowed, and `rm -rf $OUT/` read as `/` and blocked
// as a delete of the root. An entry's arg_values are compared with the word's
// written spelling of its variables instead, and a substitution's output is
// dropped from it as before. Each line is also read as the string of `sh -c
// "…"`, whose variables the enclosing shell expands, and of `bash -c '…'`,
// whose variables the string's own shell expands.
func TestArgValuesReadAVariableAsWritten(t *testing.T) {
	const home, cwd = "rm-rf-root-or-home", "rm-rf-working-directory"
	cases := []struct {
		cmd   string
		want  Verdict
		entry string
	}{
		{`rm -rf $HOME`, VerdictBlock, home},
		{`rm -rf "$HOME"`, VerdictBlock, home},
		{`rm -rf ${HOME}`, VerdictBlock, home},
		{`rm -rf $HOME/.*`, VerdictBlock, home},
		{`rm -rf "$PWD"`, VerdictWarn, cwd},
		{`rm -rf $PWD`, VerdictWarn, cwd},
		{`rm -rf ${PWD}/*`, VerdictWarn, cwd},
		{`rm -rf "$BUILD_DIR"/*`, VerdictAllow, ""},
		{`rm -rf $OUT/`, VerdictAllow, ""},
		{`rm -rf /`, VerdictBlock, home},
		{`rm -rf ~`, VerdictBlock, home},
		{`rm -rf ~/.*`, VerdictBlock, home},
		{`rm -rf *`, VerdictWarn, cwd},
	}
	for _, tc := range cases {
		spellings := []string{
			tc.cmd,
			`sh -c "` + strings.ReplaceAll(tc.cmd, `"`, `\"`) + `"`,
			`bash -c '` + tc.cmd + `'`,
		}
		for n, cmd := range spellings {
			t.Run(cmd, func(t *testing.T) {
				d := verdictOf(t, cmd)
				switch {
				case tc.want == VerdictBlock:
					if d.Verdict != VerdictBlock || d.EntryID != tc.entry {
						t.Errorf("Check(%q) = %q via %q, want block via %q", cmd, d.Verdict, d.EntryID, tc.entry)
					}
				case tc.want == VerdictWarn:
					// A string holding `${` is itself a warn the payload
					// reader raises, so only the top-level line names the
					// entry; the strings must warn and never block.
					if d.Verdict != VerdictWarn || (n == 0 && d.EntryID != tc.entry) {
						t.Errorf("Check(%q) = %q via %q, want warn via %q", cmd, d.Verdict, d.EntryID, tc.entry)
					}
				default:
					// Never a delete of the root or the working directory.
					if d.EntryID == home || d.EntryID == cwd || d.Verdict == VerdictBlock || (n == 0 && d.Verdict != VerdictAllow) {
						t.Errorf("Check(%q) = %q via %q, want no rm-target verdict", cmd, d.Verdict, d.EntryID)
					}
				}
			})
		}
	}
}

// TestArgValuesWrittenSpellingEdges pins how the written spelling reads the
// shapes around the table above (iss-2609290321312087): a substitution glued
// to the variable is dropped as its output is; a simple name the next byte
// would extend is braced, so `"$HOM"E` is not `$HOME`; the enclosing shell's
// variable inside the string's own single quotes is still that variable's
// value; a nested string carries the name down; and a mark whose name the
// string does not hold — a raw 0x01 byte — names nothing, where reading it as
// empty text read `\x01/` as the root. A brace expansion's words and a
// default (`${HOME:-/}`) are the recorded residual: no written spelling.
func TestArgValuesWrittenSpellingEdges(t *testing.T) {
	const home = "rm-rf-root-or-home"
	runVerdictCases(t, []verdictCase{
		{`rm -rf "$(true)"$HOME`, VerdictBlock, home},
		{`rm -rf $HOME"$(true)"`, VerdictBlock, home},
		{`rm -rf "$(true)"$HOME/.*`, VerdictBlock, home},
		{`sh -c "rm -rf '$HOME'"`, VerdictBlock, home},
		{`sh -c 'rm -rf "$HOME"'`, VerdictBlock, home},
		{`sh -c "sh -c \"rm -rf $HOME\""`, VerdictBlock, home},
		{`sh -c "cd /tmp && rm -r $HOME"`, VerdictBlock, home},
		{`rm -rf "$HOM"E`, VerdictAllow, ""},
		{`rm -rf "$HOME"x`, VerdictAllow, ""},
		{`rm -rf $HOMEx`, VerdictAllow, ""},
		{`sh -c "rm -rf \"$HOM\"E"`, VerdictAllow, ""},
		{"rm -rf \x01/", VerdictAllow, ""},
		{"rm -rf \"\x01\"/", VerdictAllow, ""},
		{`rm -rf ${HOME:-/}`, VerdictAllow, ""},
		{`rm -rf {$HOME,x}`, VerdictAllow, ""},
	})
}
