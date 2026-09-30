package guard

import "testing"

// TestQuotedDefaultWordsTheWrittenCompareReads — review-guardSet MAJOR-1. A
// default's or an alternative's word written as an ANSI-C string (`$'/'`,
// `$'\x2f'`, `$'\57'`, `$'/'`) or a locale string (`$"/"`) prints what
// it decodes to: bash 3.2, /bin/sh and bash 5.3 print `/` for
// `${X:-$'/'}` and `${X:-$"/"}` with X unset, as they do for the bare
// `rm -rf $'/'`. A `$` that opens nothing is the `$` it is (`${X:-$/}`
// prints `$/`), and `$!` can print nothing, which leaves the text beside it
// (`${X:-$!/}` is `/` with no background job).
func TestQuotedDefaultWordsTheWrittenCompareReads(t *testing.T) {
	const home = "rm-rf-root-or-home"
	const all = shellBare | shellSQ | shellDQ
	checkSpellingCases(t, []spellingCase{
		{`rm -rf ${X:-$'/'}`, shellBare, VerdictBlock, home},
		{`rm -rf "${X:-$'/'}"`, shellBare, VerdictBlock, home},
		{`rm -rf ${X-$'/'}`, shellBare, VerdictBlock, home},
		{`rm -rf ${X:=$'/'}`, shellBare, VerdictBlock, home},
		{`rm -rf ${X:+$'/'}`, shellBare, VerdictBlock, home},
		{`rm -rf ${X:-$'\x2f'}`, shellBare, VerdictBlock, home},
		{`rm -rf ${X:-$'\57'}`, shellBare, VerdictBlock, home},
		{`rm -rf ${X:-$'/'}`, shellBare, VerdictBlock, home},
		{`rm -rf ${X:-$'\x2f\x00zz'}`, shellBare, VerdictBlock, home},
		{`rm -rf ${X:-"$HOME"$'/'}`, shellBare, VerdictBlock, home},
		{`rm -rf ${X:-$"/"}`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf ${X:+$"/"}`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf ${X:-$"$HOME"}`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf ${X:-$!/}`, all, VerdictBlock, home},
		// The look-alikes: a word that decodes to a path of its own, a
		// quoted `$'` that stays text, and a `$` that opens nothing.
		{`rm -rf ${X:-$'./build'}`, shellBare, VerdictAllow, ""},
		{`rm -rf ${X:-"$'/'"}`, shellBare, VerdictAllow, ""},
		{`rm -rf ${X:-$}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${X:-$/}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${X:-$$}`, shellBare | shellSQ, VerdictAllow, ""},
	})
}

// TestDefaultWordsSplitOnANamedIFSTheWrittenCompareReads — review-guardSet
// MAJOR-2. An unquoted default's or alternative's word is split into fields
// on the IFS the shell holds when it expands it, and an assignment in a
// command of its own changes that IFS: `IFS=x; rm -rf ${U:-x/x}` hands rm
// `""` and `/` on bash 3.2, /bin/sh, dash and bash 5.3. So is an unquoted
// `$HOME` (`IFS=Uv; rm -rf $HOME/x` hands rm `/` with HOME=/Users/dev). On a
// line that names IFS such a word refuses: the guard reads the split on the
// default IFS only. A quoted word is not split, and a variable of unknown
// value splits into text no more known than its value, so `while IFS= read
// -r d; do rm -rf $d; done` stays allowed.
func TestDefaultWordsSplitOnANamedIFSTheWrittenCompareReads(t *testing.T) {
	const home = "rm-rf-root-or-home"
	checkSpellingCases(t, []spellingCase{
		{`IFS=x; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictBlock, home},
		{`IFS=x; rm -rf ${U:+x/x}`, shellBare | shellSQ, VerdictBlock, home},
		{`IFS=x; rm -rf ${U-x/x}`, shellBare | shellSQ, VerdictBlock, home},
		{`export IFS=x; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictBlock, home},
		{`read IFS; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictBlock, home},
		{`IFS=x; rm -rf ${U:-${V:-x/x}}`, shellBare | shellSQ, VerdictBlock, home},
		{`IFS=x; eval rm -rf ${U:-x/x}`, shellBare, VerdictBlock, home},
		{`IFS=x; eval 'rm -rf ${U:-x/x}'`, shellBare, VerdictBlock, home},
		{`IFS=Uv; rm -rf $HOME/x`, shellBare | shellSQ, VerdictBlock, home},
		{`IFS=Uv; rm -rf ${HOME%/}/x`, shellBare | shellSQ, VerdictBlock, home},
		// The look-alikes: a quoted word, a variable of unknown value, and a
		// line that names no IFS.
		{`IFS=x; rm -rf "${U:-x/x}"`, shellBare, VerdictAllow, ""},
		{`while IFS= read -r d; do rm -rf "$d"; done`, shellBare | shellSQ, VerdictAllow, ""},
		{`while IFS= read -r d; do rm -rf $d; done`, shellBare | shellSQ, VerdictAllow, ""},
		{`while IFS= read -r d; do rm -rf ${d%/}; done`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${TMPDIR:-/tmp}/abcd-x`, shellBare | shellSQ, VerdictAllow, ""},
	})
}

// TestQuotedSlashReplacementsTheWrittenCompareReads — review-guardSet
// MAJOR-3. bash 5 reads a quoted `/` as part of a replacement's pattern,
// where bash 3.2 ends the pattern there: `${X/"/"*/$HOME}` prints the home on
// bash 5.3 and the value on bash 3.2. Both readings are the expansion's.
// A `$""` or `$"…"` in a pattern is the quoted text it holds
// (`${X%%$""*}/` is `/`).
func TestQuotedSlashReplacementsTheWrittenCompareReads(t *testing.T) {
	const home = "rm-rf-root-or-home"
	checkSpellingCases(t, []spellingCase{
		{`rm -rf ${X/"/"*/$HOME}`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf ${X//"/"*/$HOME}`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf "${X/"/"*/$HOME}"`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf ${X/'/'*/$HOME}`, shellBare, VerdictBlock, home},
		{`rm -rf ${X/$"/"*/$HOME}`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf ${X/$""*/$HOME}`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf ${X%%$""*}/`, shellBare | shellSQ, VerdictBlock, home},
		// The look-alikes: a quoted `/` that replaces one byte, and a bracket.
		{`rm -rf ./${X/"/"/_}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${X/[/]*/$HOME}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf "./${X//"/"/_}"`, shellBare | shellSQ, VerdictAllow, ""},
	})
}

// TestIndirectAndSpecialDefaultsTheWrittenCompareReads — review-guardSet
// MAJOR-4. bash 3.2 and /bin/sh read `${!X:-/}` as X's indirection with a
// default, and print `/` with X unset. The value an indirection names is not
// in the line, so its spelling past an operator reads as every value. A
// positional or special parameter takes the same operators (`${1:-/}`,
// `${@:-/}`, `${!:-/}` and `${#:+/}` print `/` on bash 3.2, /bin/sh, dash and
// bash 5.3), and its value is the parameter as written.
func TestIndirectAndSpecialDefaultsTheWrittenCompareReads(t *testing.T) {
	const home = "rm-rf-root-or-home"
	const all = shellBare | shellSQ | shellDQ
	checkSpellingCases(t, []spellingCase{
		{`rm -rf ${!X:-/}`, all, VerdictBlock, home},
		{`rm -rf ${!X-/}`, all, VerdictBlock, home},
		{`rm -rf ${!X:-$HOME}`, all, VerdictBlock, home},
		{`rm -rf ${!X:+/}`, all, VerdictBlock, home},
		{`rm -rf ${1:-/}`, all, VerdictBlock, home},
		{`rm -rf ${1-$HOME}`, all, VerdictBlock, home},
		{`rm -rf ${10:-/}`, all, VerdictBlock, home},
		{`rm -rf ${@:-/}`, all, VerdictBlock, home},
		{`rm -rf ${*:-/}`, all, VerdictBlock, home},
		{`rm -rf ${!:-/}`, all, VerdictBlock, home},
		{`rm -rf ${#:+/}`, all, VerdictBlock, home},
		{`rm -rf ${?:+/}`, all, VerdictBlock, home},
		// The look-alikes: an indirection alone, a name list, a length, and
		// a positional default naming a path of its own.
		{`rm -rf ${!X}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${!X*}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${#X}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${1:-dist}`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf "${1:-./build}"`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${1%/}`, shellBare | shellSQ, VerdictAllow, ""},
	})
}

// TestEverydayExpansionsStayAllowed pins the everyday idioms review-guardSet
// held allowed at the base and the head.
func TestEverydayExpansionsStayAllowed(t *testing.T) {
	cases := []string{
		`rm -rf "${TMPDIR:-/tmp}/abcd-x"`,
		`rm -rf "${XDG_CACHE_HOME:-$HOME/.cache}/abcd"`,
		`rm -rf ${DIR:-./build}`,
		`rm -rf ${f%.*}`,
		`rm -rf ${p##*/}`,
		`rm -rf "${DIR/#\~/$HOME}"`,
		`rm -rf ${name//[^a-z]/}`,
		`rm -rf ${1:-dist}`,
		`rm -rf "./${X//\//_}"`,
		`rm -rf ${X/foo/$HOME}`,
		`rm -rf "${DIR%/}/build"`,
		`rm -rf "$HOME/${d##*/}"`,
	}
	var sc []spellingCase
	for _, c := range cases {
		sc = append(sc, spellingCase{c, shellBare | shellSQ, VerdictAllow, ""})
	}
	checkSpellingCases(t, sc)
}

// TestSeparatorsBeforeTheHomeTheWrittenCompareReads — iss-2609300057462186.
// The home and the working directory are absolute paths, and a run of `/`
// written before one names the same directory: `rm -rf /$HOME` deletes the
// home. A separator after the name is a path beneath it.
func TestSeparatorsBeforeTheHomeTheWrittenCompareReads(t *testing.T) {
	const home = "rm-rf-root-or-home"
	checkSpellingCases(t, []spellingCase{
		{`rm -rf /$HOME`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf //${HOME}/`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf "/$HOME"/*`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf /$HOME/build`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf /$HOMEDIR`, shellBare | shellSQ, VerdictAllow, ""},
	})
}

// TestParametersThatPrintNothingTheWrittenCompareReads — iss-2609300057467536.
// A parameter that can print nothing at the top of a fresh shell leaves the
// text beside it: bash 3.2, /bin/sh, dash and bash 5.3 print `/` for
// `$!/` (no background job ran), `$@/`, `$*/` and `$1/` (no argument),
// `$_/` after `x=` or `true ""`, and dash for `$-/` (no option letter),
// braced or not, quoted or not. A number that is never empty (`$$`, `$?`,
// `$#`) and the shell's name (`$0`) name no path, and the job's number
// stays the operand `kill` and `wait` take.
func TestParametersThatPrintNothingTheWrittenCompareReads(t *testing.T) {
	const home = "rm-rf-root-or-home"
	const all = shellBare | shellSQ | shellDQ
	checkSpellingCases(t, []spellingCase{
		{`rm -rf $!/`, all, VerdictBlock, home},
		{`rm -rf "$!"/`, all, VerdictBlock, home},
		{`rm -rf "$!/"`, all, VerdictBlock, home},
		{`rm -rf /$!`, all, VerdictBlock, home},
		{`rm -rf ~$!`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf $!/*`, all, VerdictBlock, home},
		{`rm -rf $(true)$!/`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf {$!,x}/`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf ${!}/`, all, VerdictBlock, home},
		{`rm -rf "${!}"/`, all, VerdictBlock, home},
		{`rm -rf $-/`, all, VerdictBlock, home},
		{`rm -rf "${-}/"`, all, VerdictBlock, home},
		{`rm -rf $_/`, all, VerdictBlock, home},
		{`rm -rf "${_}"/`, all, VerdictBlock, home},
		{`rm -rf $@/`, all, VerdictBlock, home},
		{`rm -rf "$@"/`, all, VerdictBlock, home},
		{`rm -rf "${@}/"`, all, VerdictBlock, home},
		{`rm -rf $*/`, all, VerdictBlock, home},
		{`rm -rf "$*/"`, all, VerdictBlock, home},
		{`rm -rf ${*}/`, all, VerdictBlock, home},
		{`rm -rf $1/`, all, VerdictBlock, home},
		{`rm -rf "${1}"/`, all, VerdictBlock, home},
		{`rm -rf ${10}/`, all, VerdictBlock, home},
		{`rm -rf ${X:-$@/}`, all, VerdictBlock, home},
		{`rm -rf ${X:-$_/}`, all, VerdictBlock, home},
		{`rm -rf ${@%x}/`, all, VerdictBlock, home},
		// In a pattern too: `$!*` can be `*`, which takes all of PWD.
		{`rm -rf ${PWD%%$!*}/`, all, VerdictBlock, home},
		{`rm -rf "${X%%"$!"*}"/`, shellBare | shellSQ, VerdictBlock, home},
		// The look-alikes: a parameter that is never empty, a name that runs
		// on past the `_`, and the job's number where it names no path.
		{`rm -rf $$/`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf $?/`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf $#/`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf $0/`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${0}/`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf $_x/`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf $!`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf "$1"`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -f "$tmp.$!"`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf "$d/$!"`, shellBare | shellSQ, VerdictAllow, ""},
		{`kill $!`, shellBare | shellSQ, VerdictAllow, ""},
		{`wait $!`, shellBare | shellSQ, VerdictAllow, ""},
		{`echo $!`, shellBare | shellSQ, VerdictAllow, ""},
		// The empty reading splits into no field: a line that names IFS
		// reads these as it did before it (ifsSplits).
		{`IFS=, ; rm -rf $1`, shellBare | shellSQ, VerdictAllow, ""},
		{`IFS=, ; rm -rf ${1}`, shellBare | shellSQ, VerdictAllow, ""},
		{`IFS=, ; rm -rf ${1%/}`, shellBare | shellSQ, VerdictAllow, ""},
		{`IFS=, ; rm -rf $_`, shellBare | shellSQ, VerdictAllow, ""},
	})
}

// TestIFSNamedThroughAMarkTheWrittenCompareReads — reverify-guardSet finding
// 1. An IFS can be named through a word that holds an expansion:
// `export ${I}FS=x`, `declare I${F}FS=x`, `read -r ${I}FS`,
// `printf -v ${I}FS x`, and `eval "I${F:-F}S=x"`, whose string the eval
// runs as an assignment. With I=I and F=F each gives IFS the value x, and
// `rm -rf ${U:-x/x}` then hands rm `""` and `/` on bash 3.2, /bin/sh, dash
// and bash 5.3. The guard does not spell the name, so a declaration's word,
// a `read` or `printf -v` target, and an assignment word whose name holds an
// expansion count as naming IFS, and so does an arithmetic expression that
// names IFS or assigns through an expansion (`: $((IFS=1))`).
func TestIFSNamedThroughAMarkTheWrittenCompareReads(t *testing.T) {
	const home = "rm-rf-root-or-home"
	checkSpellingCases(t, []spellingCase{
		{`I=I; export ${I}FS=x; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictBlock, home},
		{`export I${F}FS=x; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictBlock, home},
		{`export "${I}FS"=x; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictBlock, home},
		{`declare ${I}FS=x; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictBlock, home},
		{`declare -x ${I}FS=x; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictBlock, home},
		{`typeset ${I}FS=x; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictBlock, home},
		{`readonly ${I}FS=x; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictBlock, home},
		{`local ${I}FS=x; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictBlock, home},
		{`declare $(echo I)FS=x; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictBlock, home},
		{`read -r ${I}FS <<< x; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictBlock, home},
		{`printf -v ${I}FS x; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictBlock, home},
		{`printf -v "$n" x; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictBlock, home},
		{`eval "I${F:-F}S=x"; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictBlock, home},
		{`eval I${F}FS=x; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictBlock, home},
		{`I${F:-F}S=x; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictBlock, home},
		{`eval "${I}FS=x"; eval rm -rf ${U:-x/x}`, shellBare, VerdictBlock, home},
		{`let ${I}FS=1; rm -rf ${U:-1/1}`, shellBare | shellSQ, VerdictBlock, home},
		{`getopts a ${I}FS; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictBlock, home},
		{`mapfile -t ${I}FS < f; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictBlock, home},
		{`wait -p ${I}FS; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictBlock, home},
		// An arithmetic assignment leaves no word to read the name in.
		{`: $((IFS=1)); rm -rf ${U:-1/1}`, shellBare | shellSQ, VerdictBlock, home},
		{`(( IFS=1 )); rm -rf ${U:-1/1}`, shellBare | shellSQ, VerdictBlock, home},
		{`: $((${I}FS=1)); rm -rf ${U:-1/1}`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf ${U:-1/1}; echo "$((IFS=1))"`, shellBare | shellSQ, VerdictBlock, home},
		{`: $((${I}FS<<=1)); rm -rf ${U:-1/1}`, shellBare | shellSQ, VerdictBlock, home},
		// The look-alikes: a declaration, a read and a printf whose names are
		// written, an assignment whose value (not its name) holds an
		// expansion, and the everyday reads with a quoted operand.
		{`export PATH=$HOME/bin:$PATH; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictAllow, ""},
		{`declare -a files; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictAllow, ""},
		{`read -r f; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictAllow, ""},
		{`printf '%s\n' "$x"; rm -rf ${U:-x/x}`, shellBare, VerdictAllow, ""},
		{`printf -v out '%s' "$x"; rm -rf ${U:-x/x}`, shellBare, VerdictAllow, ""},
		{`OUT=$x; rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictAllow, ""},
		{`n=$((n+1)); rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictAllow, ""},
		{`(( $n == 1 )) && rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictAllow, ""},
		{`: $(($n <= 1)); rm -rf ${U:-x/x}`, shellBare | shellSQ, VerdictAllow, ""},
		{`IFS= read -r f; rm -rf "$f"`, shellBare | shellSQ, VerdictAllow, ""},
		{`IFS=, read -ra arr <<< "$x"; rm -rf "${arr[0]}"`, shellBare | shellSQ, VerdictAllow, ""},
	})
}

// TestColonDefaultsOfAnEmptyParameterTheWrittenCompareReads —
// reverify-guardSet finding 2. With the colon, a default, an assignment and
// an error message treat an empty parameter as unset, so `${1:-dist}` with
// no argument, or an empty one, prints `dist`, never nothing: bash 3.2,
// /bin/sh, dash and bash 5.3 hand rm `dist/` for `${1:-dist}/`. Without the
// colon a set but empty parameter prints its value, nothing (`${1-dist}/` is
// `/` after `set -- ""`).
func TestColonDefaultsOfAnEmptyParameterTheWrittenCompareReads(t *testing.T) {
	const home = "rm-rf-root-or-home"
	const all = shellBare | shellSQ | shellDQ
	checkSpellingCases(t, []spellingCase{
		{`rm -rf "${1:-build}"/*`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${1:-dist}/`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf "${1:-dist}/"*`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${@:-x}/`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${1:=dist}/`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${1:?}/`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ./${1:-dist}`, shellBare | shellSQ, VerdictAllow, ""},
		// In a string the outer shell expands, `${1:-dist}` hands the inner
		// shell `${1}`'s text, which it reads as a parameter of its own that
		// can print nothing: `sh -c "rm -rf ${1:-dist}/"` refuses, a
		// fail-closed over-read.
		// The block forms: a default that is the root, an empty or
		// colonless default, and an error message without the colon.
		{`rm -rf ${1:-/}`, all, VerdictBlock, home},
		{`rm -rf ${1-}/`, all, VerdictBlock, home},
		{`rm -rf ${1-dist}/`, all, VerdictBlock, home},
		{`rm -rf ${1=dist}/`, all, VerdictBlock, home},
		{`rm -rf ${@-x}/`, all, VerdictBlock, home},
		{`rm -rf ${1?}/`, all, VerdictBlock, home},
		// An indirection past a colon default still reads as every value.
		{`rm -rf ${!X:-dist}/`, all, VerdictBlock, home},
		{`rm -rf ${!X:?}/`, all, VerdictBlock, home},
	})
}

// TestExpansionsThatCanPrintNothingTheWrittenCompareReads —
// reverify-guardSet finding 5. An empty default's word prints the empty
// text (`${X:-}/` and `${X-}/` are `/` with X unset), a subscript can name
// an element that is not set (`${A[0]}/`, `${A[@]}/` with A unset or a
// scalar's `${A[1]}`), and a case change or a transform prints nothing for
// a value it maps to nothing (`${X^}/`, `${X@P}/` on bash 5.3 with X
// empty). bash 3.2, /bin/sh and bash 5.3 hand rm `/` for each.
func TestExpansionsThatCanPrintNothingTheWrittenCompareReads(t *testing.T) {
	const home = "rm-rf-root-or-home"
	const all = shellBare | shellSQ | shellDQ
	checkSpellingCases(t, []spellingCase{
		{`rm -rf ${X:-}/`, all, VerdictBlock, home},
		{`rm -rf ${X-}/`, all, VerdictBlock, home},
		{`rm -rf ${X:=}/`, all, VerdictBlock, home},
		{`rm -rf ${X:+}/`, all, VerdictBlock, home},
		{`rm -rf ${X:-""}/`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf ${X:-''}/`, shellBare, VerdictBlock, home},
		{`rm -rf $HOME${X:-}`, all, VerdictBlock, home},
		{`rm -rf ${A[0]}/`, all, VerdictBlock, home},
		{`rm -rf ${A[@]}/`, all, VerdictBlock, home},
		{`rm -rf "${A[1]}"/*`, shellBare | shellSQ, VerdictBlock, home},
		{`rm -rf ${X^}/`, all, VerdictBlock, home},
		{`rm -rf ${X^^}/`, all, VerdictBlock, home},
		{`rm -rf ${X,,}/`, all, VerdictBlock, home},
		{`rm -rf ${X@P}/`, all, VerdictBlock, home},
		{`rm -rf ${X@U}/`, all, VerdictBlock, home},
		// The look-alikes: text after the empty text, a quoted array, and
		// an error message.
		{`rm -rf "${TMPDIR:-}/abcd-x"`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf "${A[0]}/build"`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf "${files[@]}"`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf ${X^^}.txt`, shellBare | shellSQ, VerdictAllow, ""},
		{`rm -rf "${BUILD_DIR:?}/"*`, shellBare | shellSQ, VerdictAllow, ""},
		{`while IFS= read -r f; do rm -rf ${f:-}; done`, shellBare | shellSQ, VerdictAllow, ""},
		{`IFS=, ; rm -rf ${A[0]}`, shellBare | shellSQ, VerdictAllow, ""},
	})
}

// TestPositionalSlicesReadAsTheParameters — reverify-guardSet finding 3.
// `${@:2}` and `${*:2}` are the arguments from the second on, and
// `${1:2}` a part of the first: each prints what the parameters hold, as
// `"$2"` and `"$1"` do, or nothing, which leaves the text beside it
// (`"${@:2}"/` is `/` with no argument).
func TestPositionalSlicesReadAsTheParameters(t *testing.T) {
	const home = "rm-rf-root-or-home"
	const all = shellBare | shellSQ | shellDQ
	checkSpellingCases(t, []spellingCase{
		{`rm -rf "${@:2}"`, all, VerdictAllow, ""},
		{`rm -rf "${@:1}"`, all, VerdictAllow, ""},
		{`rm -rf ${@:2}`, all, VerdictAllow, ""},
		{`rm -rf "${*:2}"`, all, VerdictAllow, ""},
		{`rm -rf "${1:2}"`, all, VerdictAllow, ""},
		{`rm -rf "${@:2}"/`, all, VerdictBlock, home},
		{`rm -rf ${1:2}/`, all, VerdictBlock, home},
		{`rm -rf ${X:0:1}/`, all, VerdictBlock, home},
	})
}
