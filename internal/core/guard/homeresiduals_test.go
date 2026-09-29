package guard

import (
	"strings"
	"testing"
)

// TestHomeSpellingsTheWrittenCompareReads — iss-2609290419119456. Three ways
// of writing the home that bash reads as the home, or as a path that can be
// it, reached rm-rf-root-or-home's arg_values compare as no word it names:
//
//   - a backslash-newline inside the variable's name, which bash drops before
//     it reads the name (`$HO\⏎ME` is `$HOME`);
//   - a variable inside a brace expansion, which bash expands before it reads
//     the variable (`{$HOME,x}` is `$HOME` and `x`; `$HO{ME,}` is `$HOME` and
//     `$HO`);
//   - a parameter expansion of HOME whose operator can leave the value as it
//     is: a default, an assignment or an error message (the home is set), a
//     trimmed prefix or suffix and a pattern replacement (the pattern need not
//     match), a substring (its offset can be 0), a case change (a
//     case-insensitive disk), a subscript, and an alternative whose word is
//     the home.
//
// Each line is also read as the string of `sh -c "…"` and of `bash -c '…'`
// where that shell reads it the same way (shells lists which).
func TestHomeSpellingsTheWrittenCompareReads(t *testing.T) {
	const home, cwd = "rm-rf-root-or-home", "rm-rf-working-directory"
	const bare, sq, dq = 1, 2, 4
	cases := []struct {
		cmd    string
		shells int
		want   Verdict
		entry  string
	}{
		// A backslash-newline inside the name.
		{"rm -rf $HO\\\nME", bare | sq | dq, VerdictBlock, home},
		{"rm -rf \"$HO\\\nME\"", bare | sq, VerdictBlock, home},
		{"rm -rf $H\\\nO\\\n\\\nME/*", bare | sq | dq, VerdictBlock, home},
		{"rm -rf ${HO\\\nME}", bare | sq | dq, VerdictBlock, home},
		{"rm -rf $PW\\\nD", bare | sq, VerdictWarn, cwd},
		// A variable inside a brace expansion.
		{`rm -rf {$HOME,x}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf {x,${HOME}}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf $HOME/{.*,}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf {"$HOME",/tmp/x}`, bare | sq, VerdictBlock, home},
		{`rm -rf $HO{ME,}`, bare | sq, VerdictBlock, home},
		{`rm -rf {$HO,x}ME`, bare | sq, VerdictBlock, home},
		{`rm -rf {$PWD,x}`, bare | sq, VerdictWarn, cwd},
		// A parameter expansion of HOME with an operator.
		{`rm -rf ${HOME%/}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME%%/}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME%/*}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME#}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME##x}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME:-x}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME:-/}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME-x}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME:=x}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME:?x}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME/x/x}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME//x/y}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME:0}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME^^}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME,,}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME[0]}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME[@]%/}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME@P}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME:+$HOME}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${X:+"$HOME"}`, bare | sq, VerdictBlock, home},
		{`rm -rf ${X:+${HOME%/}}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME%/}/*`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf "${HOME%/}"/.*`, bare | sq, VerdictBlock, home},
		{`rm -rf ${PWD%/}`, bare | sq, VerdictWarn, cwd},
		// What stays off the home: a suffix glued on, a quoted value, a
		// length, an indirection, an alternative that is not the home, and
		// quoting that ends the name before the brace.
		{`rm -rf ${HOME%/}x`, bare | sq, VerdictAllow, ""},
		{`rm -rf ${HOME@Q}`, bare | sq, VerdictAllow, ""},
		{`rm -rf ${#HOME}`, bare | sq, VerdictAllow, ""},
		{`rm -rf ${!HOME}`, bare | sq, VerdictAllow, ""},
		{`rm -rf ${X:+x}`, bare | sq, VerdictAllow, ""},
		{`rm -rf "$HO"{ME,}`, bare | sq, VerdictAllow, ""},
		{`rm -rf ${HO}{ME,}`, bare | sq, VerdictAllow, ""},
		{"rm -rf $HO\\ME", bare | sq, VerdictAllow, ""},
		{`rm -rf {$OUT,x}/`, bare | sq, VerdictAllow, ""},
	}
	for _, tc := range cases {
		var spellings []string
		if tc.shells&bare != 0 {
			spellings = append(spellings, tc.cmd)
		}
		if tc.shells&sq != 0 {
			spellings = append(spellings, `bash -c '`+tc.cmd+`'`)
		}
		if tc.shells&dq != 0 {
			spellings = append(spellings, `sh -c "`+strings.ReplaceAll(tc.cmd, `"`, `\"`)+`"`)
		}
		for n, cmd := range spellings {
			t.Run(cmd, func(t *testing.T) {
				d := verdictOf(t, cmd)
				switch tc.want {
				case VerdictBlock:
					if d.Verdict != VerdictBlock || d.EntryID != tc.entry {
						t.Errorf("Check(%q) = %q via %q, want block via %q", cmd, d.Verdict, d.EntryID, tc.entry)
					}
				case VerdictWarn:
					// A string holding `${` is a warn the payload reader
					// raises itself, so only the line names the entry.
					if d.Verdict != VerdictWarn || (n == 0 && d.EntryID != tc.entry) {
						t.Errorf("Check(%q) = %q via %q, want warn via %q", cmd, d.Verdict, d.EntryID, tc.entry)
					}
				default:
					if d.EntryID == home || d.EntryID == cwd || d.Verdict == VerdictBlock || (n == 0 && d.Verdict != VerdictAllow) {
						t.Errorf("Check(%q) = %q via %q, want no rm-target verdict", cmd, d.Verdict, d.EntryID)
					}
				}
			})
		}
	}
}

// TestHomeSpellingsStayLinear holds the spellings above to the cost bar
// (iss-2609290419119456): an alternative's word is followed at most
// spellAlternativeDepth deep, a brace group's words carry their variables by
// index, and a name read across backslash-newlines is read once.
func TestHomeSpellingsStayLinear(t *testing.T) {
	shapes := []struct {
		name  string
		build func(int) string
	}{
		{"alternatives", func(n int) string {
			return "rm -rf " + strings.Repeat("${X:+${Y:+${Z:+${HOME%/}}}} ", n/28)
		}},
		{"nested alternatives", func(n int) string {
			return "rm -rf " + strings.Repeat("${X:+", n/5) + "$HOME" + strings.Repeat("}", n/5)
		}},
		{"brace groups", func(n int) string {
			return "rm -rf " + strings.Repeat("{$A,$HO}{ME,x}/ ", n/16)
		}},
		{"continued names", func(n int) string {
			return "rm -rf $H" + strings.Repeat("\\\nO", n/3)
		}},
	}
	for _, s := range shapes {
		t.Run(s.name, func(t *testing.T) {
			assertWorkGrowth(t, s.build, 1<<11, "a spelling reads each byte a bounded number of times")
		})
	}
}
