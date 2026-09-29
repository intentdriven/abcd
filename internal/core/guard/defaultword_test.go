package guard

import (
	"strings"
	"testing"
)

// spellingCase is one line of the written-spelling tables below: the command,
// the shells it is also read through (a bare line, `bash -c '…'`, `sh -c
// "…"`), and the verdict and entry it must get.
type spellingCase struct {
	cmd    string
	shells int
	want   Verdict
	entry  string
}

const (
	shellBare = 1 << iota
	shellSQ
	shellDQ
)

// checkSpellingCases runs each case as a bare line and as the string of each
// shell it names, and holds it to its verdict: a block names its entry at
// every depth, a warn names it on the bare line (a string holding `${` is a
// warn the payload reader may raise itself), and an allow is no rm-target
// verdict and no block anywhere, and an allow on the bare line.
func checkSpellingCases(t *testing.T, cases []spellingCase) {
	t.Helper()
	const home, cwd = "rm-rf-root-or-home", "rm-rf-working-directory"
	for _, tc := range cases {
		var spellings []string
		if tc.shells&shellBare != 0 {
			spellings = append(spellings, tc.cmd)
		}
		if tc.shells&shellSQ != 0 {
			spellings = append(spellings, `bash -c '`+tc.cmd+`'`)
		}
		if tc.shells&shellDQ != 0 {
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

// TestDefaultWordsTheWrittenCompareReads — iss-2609290426544292. A default
// or an assignment (`${DIR:-w}`, `${DIR-w}`, `${DIR:=w}`, `${DIR=w}`) prints
// the variable's value when it is set and its word w when it is not, so its
// written spelling is both texts, and rm-rf-root-or-home reads the word as it
// reads the plain spelling: `rm -rf ${DIR:-$HOME}` deletes the home with DIR
// unset. bash 3.2 and /bin/sh, the shells of macOS, read the same default at
// the first operator after a subscript's `]` (`${X[0]]-$HOME}`), which bash 5
// refuses as a bad substitution.
func TestDefaultWordsTheWrittenCompareReads(t *testing.T) {
	const home, cwd = "rm-rf-root-or-home", "rm-rf-working-directory"
	const all = shellBare | shellSQ | shellDQ
	checkSpellingCases(t, []spellingCase{
		// The home as a default's or an assignment's word.
		{`rm -rf ${DIR:-$HOME}`, all, VerdictBlock, home},
		{`rm -rf ${DIR-$HOME}`, all, VerdictBlock, home},
		{`rm -rf ${DIR:=$HOME}`, all, VerdictBlock, home},
		{`rm -rf ${DIR=$HOME}`, all, VerdictBlock, home},
		{`rm -rf "${DIR:-$HOME}"`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf ${DIR:-${HOME}}`, all, VerdictBlock, home},
		{`rm -rf ${DIR:-"$HOME"}`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf ${DIR:-$HOME/}`, all, VerdictBlock, home},
		{`rm -rf ${DIR:-$HOME}/*`, all, VerdictBlock, home},
		{`rm -rf ${DIR:-~}`, all, VerdictBlock, home},
		{`rm -rf ${DIR:-~/}`, all, VerdictBlock, home},
		{`rm -rf ${A:-${B:-$HOME}}`, all, VerdictBlock, home},
		{`rm -rf ${DIR:-${HOME%/}}`, all, VerdictBlock, home},
		{`rm -rf ${DIR:-x $HOME}`, all, VerdictBlock, home},
		{`rm -rf ${A:-x}${B:-$HOME}`, all, VerdictAllow, ""},
		{`rm -rf ${A:+x}${B:-$HOME}`, all, VerdictBlock, home},
		{`rm -rf ${A:+x}$HOME`, all, VerdictBlock, home},
		{`rm -rf ${A+/tmp}/`, all, VerdictBlock, home},
		// The root as the word.
		{`rm -rf ${DIR:-/}`, all, VerdictBlock, home},
		{`rm -rf ${DIR-/}`, all, VerdictBlock, home},
		{`rm -rf ${DIR:=/}`, all, VerdictBlock, home},
		{`rm -rf ${DIR=/*}`, all, VerdictBlock, home},
		{`rm -rf ${DIR:-/}*`, all, VerdictBlock, home},
		{`rm -rf ${DIR:-'/'}`, shellBare | shellDQ, VerdictBlock, home},
		// A default after a subscript, at the first operator byte past its `]`.
		{`rm -rf ${X[0]]-$HOME}`, all, VerdictBlock, home},
		{`rm -rf ${X[0]]:-$HOME}`, all, VerdictBlock, home},
		{`rm -rf ${X[0]]=$HOME}`, all, VerdictBlock, home},
		{`rm -rf ${X[0]]:=$HOME}`, all, VerdictBlock, home},
		{`rm -rf ${X[0]]x-$HOME}`, all, VerdictBlock, home},
		{`rm -rf ${X[0]]-/}`, all, VerdictBlock, home},
		// The working directory as the word warns, as its plain spelling does.
		{`rm -rf ${DIR:-$PWD}`, shellBare, VerdictWarn, cwd},
		{`rm -rf ${DIR:-.}`, shellBare, VerdictWarn, cwd},
		// A word that names neither: the everyday default.
		{`rm -rf ${DIR:-./build}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf "${DIR:-./build}"`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${DIR:-/tmp/x}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${DIR:-$HOME/.cache/x}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf "${TMPDIR:-/tmp}/abcd-x"`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${DIR:-$HOME}x`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf "${DIR:-$HOME}/build"`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${DIR:-}`, shellBare | shellSQ, VerdictAllow, ""},
		// An error message is not printed to the command's words.
		{`rm -rf ${DIR:?$HOME}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${DIR?/}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${X[0]]?$HOME}`, shellBare | shellSQ, VerdictAllow, ""},
		// A trimmed default is the variable's, not the word's.
		{`rm -rf ${DIR%$HOME}`, shellBare | shellSQ, VerdictAllow, ""},
	})
}

// TestDeepAlternativesAndSubstringsTheWrittenCompareReads —
// iss-2609290426544292. An alternative nested deeper than the spelling used to
// follow (`${X:+${X:+${X:+${X:+$HOME}}}}`) prints its innermost word as a
// shallow one does, and a substring can print the root: `${PWD:0:1}` is the
// `/` every absolute path begins with. A replacement whose pattern is only
// `*` prints its string in place of the whole value (`${X/*/$HOME}`).
func TestDeepAlternativesAndSubstringsTheWrittenCompareReads(t *testing.T) {
	const home = "rm-rf-root-or-home"
	const all = shellBare | shellSQ | shellDQ
	checkSpellingCases(t, []spellingCase{
		{`rm -rf ${X:+${X:+${X:+${X:+$HOME}}}}`, all, VerdictBlock, home},
		{`rm -rf ${X:+${X:+${X:+${X:+${X:+/}}}}}`, all, VerdictBlock, home},
		{`rm -rf ${A:-${B:-${C:-${D:-$HOME}}}}`, all, VerdictBlock, home},
		{`rm -rf ${PWD:0:1}`, all, VerdictBlock, home},
		{`rm -rf ${HOME:0:1}`, all, VerdictBlock, home},
		{`rm -rf ${PWD:0:1}*`, all, VerdictBlock, home},
		{`rm -rf ${X/*/$HOME}`, all, VerdictBlock, home},
		{`rm -rf ${X//*/~}`, all, VerdictBlock, home},
		{`rm -rf ${X/#*/\/}`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf ${X:+${X:+${X:+${X:+./build}}}}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${X/foo/$HOME}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf "${DIR/#\~/$HOME}"`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${X/*/./build}`, shellBare | shellSQ, VerdictAllow, ""},
	})
}

// TestSpellingPastItsBoundRefuses — iss-2609290426544292. A written spelling
// is followed a bounded depth into nested expansions and holds a bounded
// number of texts. Past either bound the word is read as naming every value
// an arg_values entry names, so `rm -r` of it refuses (fail closed), where
// before the bound it was read as naming nothing; a command no arg_values
// entry names reads it as it always did.
func TestSpellingPastItsBoundRefuses(t *testing.T) {
	const home = "rm-rf-root-or-home"
	// Sixteen alternatives deep, past the depth the spelling follows.
	deep := strings.Repeat("${X:+", 16) + "./build" + strings.Repeat("}", 16)
	wide := ""
	for i := 0; i < 6; i++ {
		wide += "${V" + string(rune('A'+i)) + ":-x}"
	}
	checkSpellingCases(t, []spellingCase{
		{"rm -rf " + deep, shellBare | shellSQ | shellDQ, VerdictBlock, home},
		{"rm -rf " + wide, shellBare | shellSQ | shellDQ, VerdictBlock, home},
		{"echo " + deep, shellBare | shellSQ, VerdictAllow, ""},
		{"rm -f " + deep, shellBare | shellSQ, VerdictAllow, ""},
	})
}

// TestSpellingSetsStayLinear holds the spelling sets to the cost bar
// (iss-2609290426544292): a nest of defaults is followed a bounded depth, a
// word of many defaults stops at the bound on its texts rather than
// enumerating every combination, and a string handed to a shell is re-read a
// bounded number of times.
func TestSpellingSetsStayLinear(t *testing.T) {
	shapes := []struct {
		name  string
		build func(int) string
	}{
		{"nested defaults", func(n int) string {
			return "rm -rf " + strings.Repeat("${X:-", n/5) + "$HOME" + strings.Repeat("}", n/5)
		}},
		{"wide defaults", func(n int) string {
			return "rm -rf " + strings.Repeat("${X:-$HOME}", n/11)
		}},
		{"default words", func(n int) string {
			return "rm -rf " + strings.Repeat(`${A:-"$HOME"/${B:-~/${C:-\x$D}}} `, n/32)
		}},
		{"payload defaults", func(n int) string {
			return `sh -c "rm -rf ` + strings.Repeat("${A:-/}${B:-~} x ", n/18) + `"`
		}},
	}
	for _, s := range shapes {
		t.Run(s.name, func(t *testing.T) {
			assertWorkGrowth(t, s.build, 1<<11, "a spelling set is bounded in its depth and its size")
		})
	}
}
