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
		// A subscript is read to its matching `]`, and what follows it that
		// is no alternative can leave the value: bash 3.2, the /bin/sh and
		// /bin/bash of macOS, prints the value past `${HOME[0]]}`,
		// `${HOME[0]x}` and `${HOME[0]@Q}`. A subscript whose `]` never
		// comes is spelled as the variable too.
		{`rm -rf ${HOME[x[0]]}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME[x[0]]%/}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME[0]]}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME[a]]}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME[0]x}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME[0]@Q}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME[0}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${HOME[$X]}`, bare | sq, VerdictBlock, home},
		{`rm -rf ${HOME[0]:+/}`, bare | sq | dq, VerdictBlock, home},
		// After a subscript, bash 3.2 takes the first operator byte past any
		// other text: a `+` or `:+` there reads an alternative.
		{`rm -rf ${X[0]]:+$HOME}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${PATH[0]]:+$HOME}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${X[0]x:+$HOME}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${X[0]]]:+$HOME}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${X[0]]+$HOME/}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${X[0]]^+$HOME}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${X[0]]x:+/*}`, bare | sq | dq, VerdictBlock, home},
		// A sequence expression's letters are unquoted name bytes, which a
		// bare name runs on into as it does into a list's.
		{`rm -rf $HO{M..M}E`, bare | sq, VerdictBlock, home},
		{`rm -rf $HOM{E..E}`, bare | sq, VerdictBlock, home},
		{`rm -rf $H{O..O}ME`, bare | sq, VerdictBlock, home},
		{`rm -rf $HO{M..N}E`, bare | sq, VerdictBlock, home},
		{`rm -rf $HO{M..M}E/*`, bare | sq, VerdictBlock, home},
		{`rm -rf $HOM{E..E}/.*`, bare | sq, VerdictBlock, home},
		// An alternative prints its word or nothing, one text, so its word is
		// spelled as written, through its own expansions.
		{`rm -rf ${X:+/}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${X:+/*}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${X:+~}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${X:+~/}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${X:+$HOME/}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${X:+$HOME/*}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${X:+"$HOME"/}`, bare | sq, VerdictBlock, home},
		{`rm -rf ${X:+${HOME%/}/.*}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${X+$HOME/}`, bare | sq | dq, VerdictBlock, home},
		// An unquoted alternative's word is split on whitespace, and a
		// substitution that prints nothing drops out of it.
		{`rm -rf ${X:+$HOME }`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${X:+ $HOME}`, bare | sq | dq, VerdictBlock, home},
		{"rm -rf ${X:+$HOME\tx}", bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${X:+$(true)$HOME}`, bare | sq, VerdictBlock, home},
		{"rm -rf ${X:+`echo`$HOME}", bare | sq, VerdictBlock, home},
		{`rm -rf ${X:+$HOME$(x)}`, bare | sq, VerdictBlock, home},
		{`rm -rf ${X:+a${Y:+ $HOME}}`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${X:+$HOME }x`, bare | sq | dq, VerdictBlock, home},
		{`rm -rf ${X:+x /}`, bare | sq | dq, VerdictBlock, home},
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
		{`rm -rf ${X:+$HOME/x}`, bare | sq, VerdictAllow, ""},
		{`rm -rf ${X:+$HOMEx}`, bare | sq, VerdictAllow, ""},
		{`rm -rf ${X:+$HO}ME`, bare | sq, VerdictAllow, ""},
		{`rm -rf ${X:+/}x`, bare | sq, VerdictAllow, ""},
		{`rm -rf $HOME{1..2}`, bare | sq, VerdictAllow, ""},
		{`rm -rf "$HO"{M..M}E`, bare | sq, VerdictAllow, ""},
		// With X set this prints X's value; with X unset bash 3.2 and
		// /bin/sh print the word, the home. The default's word is the
		// deferred class of iss-2609290426544292, so this allow is a known
		// residual, not a claim that the form stays off the home.
		{`rm -rf ${X[0]]-$HOME}`, bare | sq, VerdictAllow, ""},
		{`rm -rf "${X:+$HOME }"`, bare | sq, VerdictAllow, ""},
		{`rm -rf ${X:+"$HOME "}`, bare | sq, VerdictAllow, ""},
		{`rm -rf ${X:+'x /'}`, bare, VerdictAllow, ""},
		{`rm -rf ${X[0]a-b+$HOME}`, bare | sq, VerdictAllow, ""},
		{`rm -rf ${X[0]]\+$HOME}`, bare | sq, VerdictAllow, ""},
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
		{"nested subscripts", func(n int) string {
			return "rm -rf ${HOME" + strings.Repeat("[x", n/3) + strings.Repeat("]", n/3) + "}"
		}},
		{"alternative words", func(n int) string {
			return "rm -rf " + strings.Repeat(`${X:+"$HOME"/${Y:+~/${Z:+\x$A}}} `, n/32)
		}},
		{"split alternative words", func(n int) string {
			return "rm -rf " + strings.Repeat("${X:+$(x) $HOME `y`\t/ } ", n/28)
		}},
		{"subscript strays", func(n int) string {
			return "rm -rf " + strings.Repeat("${X[0]]"+strings.Repeat("]", 8)+":+$HOME} ", n/24)
		}},
		{"redundant separators", func(n int) string {
			return "rm -rf " + strings.Repeat("/", n/2) + "./" + strings.Repeat("/./", n/6) + "*"
		}},
		{"parent segments", func(n int) string {
			return "rm -rf ~/" + strings.Repeat("x/", n/8) + strings.Repeat("../", n/6) + "* /tmp/" + strings.Repeat("a/../", n/10) + "*"
		}},
		{"glob segments", func(n int) string {
			return "rm -rf ~/" + strings.Repeat(".?/", n/9) + strings.Repeat("**/", n/9) + "* /" + strings.Repeat("x/.*/", n/15) + "*"
		}},
		{"working-directory parents", func(n int) string {
			return "rm -rf $PWD/" + strings.Repeat("x/", n/8) + strings.Repeat("../", n/6) + "* " + strings.Repeat("a/../", n/10) + "../*"
		}},
		{"sequence terms", func(n int) string {
			return "rm -rf " + strings.Repeat("$HO{M..M}E/ ", n/12)
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

// TestRootAndHomeWithRedundantSeparators — iss-2609290625482831. The kernel
// reads a run of slashes as one, a `.` segment as the directory itself and
// the root as its own parent, so an operand written with them names the root
// or the home as its plain spelling does.
func TestRootAndHomeWithRedundantSeparators(t *testing.T) {
	const home, cwd = "rm-rf-root-or-home", "rm-rf-working-directory"
	cases := []struct {
		cmd   string
		want  Verdict
		entry string
	}{
		{`rm -rf //`, VerdictBlock, home},
		{`rm -rf //*`, VerdictBlock, home},
		{`rm -rf ///*`, VerdictBlock, home},
		{`rm -rf /./*`, VerdictBlock, home},
		{`rm -rf /.//./*`, VerdictBlock, home},
		{`rm -rf /../*`, VerdictBlock, home},
		{`rm -rf /../../*`, VerdictBlock, home},
		{`rm -rf $HOME//`, VerdictBlock, home},
		{`rm -rf $HOME//*`, VerdictBlock, home},
		{`rm -rf "$HOME"//.*`, VerdictBlock, home},
		{`rm -rf ~//*`, VerdictBlock, home},
		{`rm -rf ~/./`, VerdictBlock, home},
		{`rm -rf ${HOME}/.//*`, VerdictBlock, home},
		{`rm -rf .//*`, VerdictWarn, cwd},
		{`rm -rf ././*`, VerdictWarn, cwd},
		{`rm -rf //tmp/x`, VerdictAllow, ""},
		{`rm -rf /tmp//x`, VerdictAllow, ""},
		{`rm -rf $HOME//x`, VerdictAllow, ""},
		{`rm -rf ~/./x`, VerdictAllow, ""},
		{`rm -rf /../tmp`, VerdictAllow, ""},
	}
	for _, tc := range cases {
		for _, cmd := range []string{tc.cmd, `bash -c '` + tc.cmd + `'`} {
			t.Run(cmd, func(t *testing.T) {
				d := verdictOf(t, cmd)
				if d.Verdict != tc.want || (cmd == tc.cmd && d.EntryID != tc.entry) || (tc.want == VerdictBlock && d.EntryID != tc.entry) {
					t.Errorf("Check(%q) = %q via %q, want %q via %q", cmd, d.Verdict, d.EntryID, tc.want, tc.entry)
				}
			})
		}
	}
}

// TestParentSegmentsThatReachTheRootOrTheHome — iss-2609290745243990. A `..`
// after a named directory is folded as the path reads lexically where the
// operand begins at the root or the home: `/tmp/../*` is `/*`, and `~/../*`
// globs the home's parent, which holds the home, so it deletes the home as
// `~` does. The kernel reads `..` differently only through a symlink, and
// the lexical reading is the one that blocks. A trailing `..` is folded too,
// though rm refuses it, since what it names is the root or holds the home.
func TestParentSegmentsThatReachTheRootOrTheHome(t *testing.T) {
	const home, cwd = "rm-rf-root-or-home", "rm-rf-working-directory"
	cases := []struct {
		cmd   string
		want  Verdict
		entry string
	}{
		{`rm -rf ~/../*`, VerdictBlock, home},
		{`rm -rf ~/../../*`, VerdictBlock, home},
		{`rm -rf $HOME/../*`, VerdictBlock, home},
		{`rm -rf $HOME/../../*`, VerdictBlock, home},
		{`rm -rf ${HOME}/../*`, VerdictBlock, home},
		{`rm -rf "$HOME"/../*`, VerdictBlock, home},
		{`rm -rf "$HOME/../"*`, VerdictBlock, home},
		{`rm -rf ~/..`, VerdictBlock, home},
		{`rm -rf ~/../`, VerdictBlock, home},
		{`rm -rf ~/../..`, VerdictBlock, home},
		{`rm -rf ~/..//*`, VerdictBlock, home},
		{`rm -rf ~/.././*`, VerdictBlock, home},
		{`rm -rf ~/x/../*`, VerdictBlock, home},
		{`rm -rf ~/x/..`, VerdictBlock, home},
		{`rm -rf ~/x/../.*`, VerdictBlock, home},
		{`rm -rf ~/x/../../*`, VerdictBlock, home},
		{`rm -rf $HOME/x/y/../../*`, VerdictBlock, home},
		{`rm -rf ~/../*/*`, VerdictBlock, home},
		{`rm -rf ~/../*/.*`, VerdictBlock, home},
		{`rm -rf ~/../*/`, VerdictBlock, home},
		{`rm -rf ~/../../*/*`, VerdictBlock, home},
		{`rm -rf /tmp/../*`, VerdictBlock, home},
		{`rm -rf /etc/../*`, VerdictBlock, home},
		{`rm -rf /tmp/..`, VerdictBlock, home},
		{`rm -rf /tmp/x/../..`, VerdictBlock, home},
		{`rm -rf /tmp/x/../../*`, VerdictBlock, home},
		{`rm -rf /a/b/../../../*`, VerdictBlock, home},
		{`rm -rf /tmp/./../*`, VerdictBlock, home},
		{`rm -rf /tmp//../*`, VerdictBlock, home},
		{`rm -rf /tmp/*/../../*`, VerdictBlock, home},
		{`rm -rf /../tmp/../*`, VerdictBlock, home},
		{`rm -rf "/tmp/.."/*`, VerdictBlock, home},
		{`rm -rf ../*`, VerdictWarn, cwd},
		{`rm -rf ~/../x`, VerdictAllow, ""},
		{`rm -rf ~/../x/*`, VerdictAllow, ""},
		{`rm -rf ~/x/../y`, VerdictAllow, ""},
		{`rm -rf ~/../.*`, VerdictAllow, ""},
		{`rm -rf ~/../x/../y`, VerdictAllow, ""},
		{`rm -rf /tmp/../tmp/x`, VerdictAllow, ""},
		{`rm -rf /tmp/x/..`, VerdictAllow, ""},
		{`rm -rf /tmp/x/../*`, VerdictAllow, ""},
		{`rm -rf /usr/../usr/local/../*`, VerdictAllow, ""},
		{`rm -rf x/../y`, VerdictAllow, ""},
	}
	for _, tc := range cases {
		spellings := []string{tc.cmd, `bash -c '` + tc.cmd + `'`, `sh -c "` + strings.ReplaceAll(tc.cmd, `"`, `\"`) + `"`}
		for n, cmd := range spellings {
			t.Run(cmd, func(t *testing.T) {
				d := verdictOf(t, cmd)
				switch tc.want {
				case VerdictBlock:
					if d.Verdict != VerdictBlock || d.EntryID != tc.entry {
						t.Errorf("Check(%q) = %q via %q, want block via %q", cmd, d.Verdict, d.EntryID, tc.entry)
					}
				case VerdictWarn:
					if d.Verdict != VerdictWarn || (n == 0 && d.EntryID != tc.entry) {
						t.Errorf("Check(%q) = %q via %q, want warn via %q", cmd, d.Verdict, d.EntryID, tc.entry)
					}
				default:
					if d.EntryID == home || d.Verdict == VerdictBlock || (n == 0 && d.Verdict != VerdictAllow) {
						t.Errorf("Check(%q) = %q via %q, want no root-or-home verdict", cmd, d.Verdict, d.EntryID)
					}
				}
			})
		}
	}
}
