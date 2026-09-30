package guard

import (
	"strings"
	"testing"
)

// TestTrimsThatCanLeaveTheRootTheWrittenCompareReads — iss-2609292320015665.
// A trim takes a prefix or a suffix off the value, and what it leaves depends
// on the shape of its pattern. One whose pattern can take a remainder of any
// length the line does not spell, from the end the trim anchors at, can leave
// only the `/` an absolute path begins with, or the one a directory's value
// ends with: with X=/a/b and T=/tmp/x/, bash 3.2, /bin/sh and dash print `/`
// for `${X%${X#?}}`, `${X%%[!/]*}`, `${T#${T%?}}` and `${T##*[!/]}`. The
// everyday trims, whose pattern is literal text or a glob that meets literal
// text at that end, stay allowed: `${DIR%/}`, `${f%.txt}`, `${p##*/}` and
// `${p%/*}` never leave the root on their own.
func TestTrimsThatCanLeaveTheRootTheWrittenCompareReads(t *testing.T) {
	const home = "rm-rf-root-or-home"
	const all = shellBare | shellSQ | shellDQ
	checkSpellingCases(t, []spellingCase{
		// A suffix trim whose pattern begins with unknown text.
		{`rm -rf ${X%${X#?}}`, all, VerdictBlock, home},
		{`rm -rf ${X%%${X#?}}`, all, VerdictBlock, home},
		{`rm -rf "${X%${X#?}}"`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf ${X%"${X#?}"}`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf ${X%$Y}`, all, VerdictBlock, home},
		{`rm -rf ${X%%$Y*}`, all, VerdictBlock, home},
		{`rm -rf ${X%*${X#?}}`, all, VerdictBlock, home},
		{`rm -rf ${X%$(echo a/b)}`, shellBare | shellSQ, VerdictBlock, home},
		{"rm -rf ${X%`echo a/b`}", shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf ${DIR%$HOME}`, all, VerdictBlock, home},
		// A suffix trim whose pattern begins with a glob and can take any length.
		{`rm -rf ${X%%[!/]*}`, all, VerdictBlock, home},
		{`rm -rf ${X%%""[!/]*}`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf ${X%%?*}/`, all, VerdictBlock, home},
		// A prefix trim whose pattern ends with unknown text or such a glob.
		{`rm -rf ${T#${T%?}}`, all, VerdictBlock, home},
		{`rm -rf ${T##*[!/]}`, all, VerdictBlock, home},
		{`rm -rf ${T##$Y}`, all, VerdictBlock, home},
		// The root it leaves, with the text around it.
		{`rm -rf ${X%${X#?}}*`, all, VerdictBlock, home},
		{`rm -rf ${X%%[!/]*}*/`, all, VerdictBlock, home},
		{`rm -rf ${HOME%${HOME#?}}`, all, VerdictBlock, home},
		{`rm -rf $HOME${X%${X#?}}`, all, VerdictBlock, home},
		// An element of an array, trimmed the same way.
		{`rm -rf ${A[1]%${A[1]#?}}`, all, VerdictBlock, home},
		// The everyday trims: literal text, or a glob that meets literal text
		// at the end the trim anchors at, or one of a fixed width.
		{`rm -rf ${DIR%/}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf "${DIR%/}"`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf "${DIR%/}/build"`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${f%.txt}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${f%.*}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${f%%.*}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${p##*/}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf "$HOME/${d##*/}"`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${p%/*}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf "${p%/*}/build"`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${p#*/}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${p##*.}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf "${p#$HOME/}"`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf "$HOME/${p#$HOME/}"`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${X%?}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${X#?}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${X%\*}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${X%"*"}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${X%*}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${X%x$Y}`, shellBare | shellSQ, VerdictAllow, ""},
	})
}

// TestExpansionsThatPrintNothingTheWrittenCompareReads — iss-2609300009506126.
// Some expansions print nothing whatever the value is, and the text around
// them is then the whole word: with X=/a/b, `${X%%*}`, `${X##*}`, `${X%%/*}`
// and `${X:0:0}` print nothing on bash 3.2, /bin/sh and dash, so
// `rm -rf ${X%%*}/` deletes the root. bash 3.2, the /bin/bash and /bin/sh of
// macOS, prints nothing too for a trim, a replacement or a substring after a
// scalar's subscript (`${X[0]%zzz}`). An operand the expansion is the whole
// of stays allowed: a word that prints nothing is no operand.
func TestExpansionsThatPrintNothingTheWrittenCompareReads(t *testing.T) {
	const home = "rm-rf-root-or-home"
	const all = shellBare | shellSQ | shellDQ
	checkSpellingCases(t, []spellingCase{
		{`rm -rf ${X%%*}/`, all, VerdictBlock, home},
		{`rm -rf ${X##*}/*`, all, VerdictBlock, home},
		{`rm -rf $HOME/${X%%*}`, all, VerdictBlock, home},
		{`rm -rf ~/${X%%/*}`, all, VerdictBlock, home},
		{`rm -rf ${X##/*}/`, all, VerdictBlock, home},
		{`rm -rf ${X%${X}}/`, all, VerdictBlock, home},
		{`rm -rf ${X:0:0}/`, all, VerdictBlock, home},
		{`rm -rf $HOME${X:9}`, all, VerdictBlock, home},
		{`rm -rf ${X[0]%zzz}/`, all, VerdictBlock, home},
		{`rm -rf ${X[0]#zzz}/*`, all, VerdictBlock, home},
		{`rm -rf ${X[0]/a/b}/`, all, VerdictBlock, home},
		{`rm -rf ${X[0]:1}/`, all, VerdictBlock, home},
		// The expansion alone, and one beside a path that is not the root.
		{`rm -rf ${X%%*}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${X%%/*}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${X%%*}/build`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf "${p%%/*}/build"`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${X[0]%zzz}/build`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${X%%.*}/`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf "${X%/*}/"`, shellBare | shellSQ, VerdictAllow, ""},
	})
}

// TestReplacementsThatCanTakeTheWholeValueTheWrittenCompareReads —
// iss-2609300009581165. A replacement whose pattern can match the whole of an
// absolute path, whatever it holds, prints its string in its place, and one
// whose pattern can match all of it after the leading `/` prints `/` and its
// string: with X=/a/b, `${X/\/*/$HOME}` and `${X/?*/$HOME}` print the home,
// and `${X/${X#?}}` and `${X//[!\/]*/}` print `/`. The everyday replacements,
// whose pattern is literal text at its start or matches a fixed width, stay
// allowed.
func TestReplacementsThatCanTakeTheWholeValueTheWrittenCompareReads(t *testing.T) {
	const home = "rm-rf-root-or-home"
	const all = shellBare | shellSQ | shellDQ
	checkSpellingCases(t, []spellingCase{
		{`rm -rf ${X/${X#?}}`, all, VerdictBlock, home},
		{`rm -rf ${X/%${X#?}}`, all, VerdictBlock, home},
		{`rm -rf ${X//[!\/]*/}`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf ${X/\/*/$HOME}`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf ${X/?*/$HOME}`, all, VerdictBlock, home},
		{`rm -rf ${X/$Y/~}`, all, VerdictBlock, home},
		{`rm -rf ${X/${X#?}/*}`, all, VerdictBlock, home},
		{`rm -rf ${X/*/}/`, all, VerdictBlock, home},
		{`rm -rf ${X/foo/$HOME}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf "${DIR/#\~/$HOME}"`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${X/*/./build}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${name//[^a-z]/}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf "./${X//\//_}"`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${X/#$HOME/~}/build`, shellBare | shellSQ, VerdictAllow, ""},
	})
}

// TestPatternShapesStayLinear holds the pattern reading to the cost bar: a
// trim's or a replacement's pattern is read once, its nested expansions
// stepped over rather than read again, so a long or deeply nested pattern
// costs what its length does.
func TestPatternShapesStayLinear(t *testing.T) {
	shapes := []struct {
		name  string
		build func(int) string
	}{
		{"long trim pattern", func(n int) string {
			return "rm -rf ${X%" + strings.Repeat("[!/]*", n/5) + "}/"
		}},
		{"nested trims", func(n int) string {
			return "rm -rf " + strings.Repeat("${X%", n/4) + "x" + strings.Repeat("}", n/4)
		}},
		{"many trims", func(n int) string {
			return "rm -rf " + strings.Repeat("${X%${Y#?}}", n/11)
		}},
		{"replacement patterns", func(n int) string {
			return "rm -rf " + strings.Repeat(`${X/\/*/$HOME}`, n/14)
		}},
	}
	for _, s := range shapes {
		t.Run(s.name, func(t *testing.T) {
			assertWorkGrowth(t, s.build, 1<<11, "a pattern is read once")
		})
	}
}
