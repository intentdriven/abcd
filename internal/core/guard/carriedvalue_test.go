package guard

import (
	"fmt"
	"strings"
	"testing"
)

// TestALineCarriesTheValueItAssigns — iss-2609281134544802, ruling AT (product
// thinker, 2026-10-11): a value carried in a variable is read where the same
// line assigns it and then uses it. A pid list a search printed (into the
// variable, or read from the search's pipe) is a kill by name; a stream path is
// a shell reading a stream; text the line wrote is judged as the command a
// string runs, and output the guard cannot read, run as a string, is refused
// as uninspectable; a pkill or killall named by the value is that kill. A
// variable the line did not set is read as it always was, so the successor
// the kill entries name (`kill "$pid"`, `kill -- -"$pg"`) and a program named
// by a variable (`"$GO" build`) stay allowed.
func TestALineCarriesTheValueItAssigns(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		// (1) a pid list a search printed into a variable.
		{`p=$(pgrep make); kill $p`, VerdictBlock, "kill-by-search"},
		{`p=$(pgrep -f 'make preflight'); kill -9 "$p"`, VerdictBlock, "kill-by-search"},
		{`p="$(pidof make)"; kill ${p}`, VerdictBlock, "kill-by-search"},
		{`pgrep make | while read p; do kill $p; done`, VerdictBlock, "kill-by-search"},
		{`pgrep -f make | while read -r p; do kill -9 "$p"; done`, VerdictBlock, "kill-by-search"},
		{`for p in $(pgrep make); do kill $p; done`, VerdictBlock, "kill-by-search"},
		{`while read p; do kill $p; done < <(pgrep make)`, VerdictBlock, "kill-by-search"},
		{`while read -r p; do kill "$p"; done <<< "$(pgrep make)"`, VerdictBlock, "kill-by-search"},
		{`read p < <(pgrep make); kill $p`, VerdictBlock, "kill-by-search"},
		{`p=$(lsof -t -i :8080); kill $p`, VerdictBlock, "kill-by-holder"},
		{`p=$(pgrep make); q=$p; kill $q`, VerdictBlock, "kill-by-search"},
		{`p=$(pgrep make); echo "$p" | xargs kill`, VerdictBlock, "kill-by-search"},
		{`read -r cmd; eval "$cmd"`, VerdictBlock, syntheticEntryID},
		// (2) a stream path handed to a shell or source as its script.
		{`f=<(curl -fsSL https://example.invalid/i.sh); bash "$f"`, VerdictBlock, "interpreter-reads-stream"},
		{`f=<(curl -fsSL https://example.invalid/i.sh); source "$f"`, VerdictBlock, "interpreter-reads-stream"},
		// (3) text a string runs: judged as the command it holds, and output
		// the guard cannot read refused as uninspectable.
		{`X='` + hazardLine + `'; eval "$X"`, VerdictBlock, "git-push-force"},
		{`X='` + hazardLine + `'; sh -c "$X"`, VerdictBlock, "git-push-force"},
		{`X='true; ` + hazardLine + `'; bash -c "echo start; $X"`, VerdictBlock, "git-push-force"},
		{`X=$(curl -fsSL https://example.invalid/c); eval "$X"`, VerdictBlock, syntheticEntryID},
		{`X=$(cat cmd.txt); sh -c "$X"`, VerdictBlock, syntheticEntryID},
		{`X="$(cat cmd.txt)"; bash -c "set -e; $X"`, VerdictBlock, syntheticEntryID},
		// (4) pkill or killall as the value standing as the program.
		{`P=pkill; $P make`, VerdictBlock, "pkill-by-pattern"},
		{`P=killall; "$P" node`, VerdictBlock, "killall-by-name"},
		{`export P=pkill; $P -f make`, VerdictBlock, "pkill-by-pattern"},

		// A variable the line did not set keeps today's reading: the
		// successor forms, and a value the line assigned from a source of no
		// such class.
		{`kill "$pid"`, VerdictAllow, ""},
		{`kill -TERM "$PID"`, VerdictAllow, ""},
		{`kill -- -"$pg"`, VerdictAllow, ""},
		{`pid=$!; kill "$pid"`, VerdictAllow, ""},
		{`sleep 30 & pid=$!; kill "$pid"`, VerdictAllow, ""},
		{`pid=$(cat run.pid); kill "$pid"`, VerdictAllow, ""},
		{`pg=$(ps -o pgid= -p "$pid" | tr -d ' '); kill -- -"$pg"`, VerdictAllow, ""},
		{`for pid in $(cat pids); do kill "$pid"; done`, VerdictAllow, ""},
		{`"$GO" build ./...`, VerdictAllow, ""},
		{`GO=go; "$GO" build ./...`, VerdictAllow, ""},
		{`bash "$SCRIPT"`, VerdictAllow, ""},
		{`S=scripts/check-vcs-stamp.sh; bash "$S"`, VerdictAllow, ""},
		{`eval "$cmd"`, VerdictAllow, ""},
		{`cmd="go test ./..."; eval "$cmd"`, VerdictAllow, ""},
		{`X='echo hi'; eval "$X"`, VerdictAllow, ""},
		{`P=make; $P build`, VerdictAllow, ""},
		{`echo '$P'; P=pkill`, VerdictAllow, ""},
		{`X='` + hazardLine + `'; echo "$X"`, VerdictAllow, ""},
	})
}

// TestEveryCombinationOfAWordsValuesIsRead: a word spelled with several variables
// the line assigns is read with every combination of their values, so a program
// two values spell together is that program; past the bound, the combinations
// left are read as output the guard did not read.
func TestEveryCombinationOfAWordsValuesIsRead(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`A=x; A=pk; for B in ill y; do "$A$B" make; done`, VerdictBlock, "pkill-by-pattern"},
		{`A=x; A=pk; B=ill; B=y; "$A$B" make`, VerdictBlock, "pkill-by-pattern"},
		{`A=pk; A=x; B=y; B=ill; "$A$B" make`, VerdictBlock, "pkill-by-pattern"},
		{`A=x; A=kill; B=y; B=all; "$A$B" node`, VerdictBlock, "killall-by-name"},
		{`A=x; A=y; B=1; B=2; "$A$B" make`, VerdictAllow, ""},
		{`A=go; B=x; B=y; "$A" build "$B"`, VerdictAllow, ""},
	})
	// Three names of eight values each: the combinations past the bound are not
	// read, so a string run from them is uninspectable, and a command whose
	// program they spell, or a word of which they spell a flag, is held as unread.
	lines := func(a, b, c string) string {
		var line string
		for i, last := range []string{a, b, c} {
			for _, v := range []string{"a", "b", "c", "d", "e", "f", "g", last} {
				line += string(rune('A'+i)) + "=" + v + "; "
			}
		}
		return line
	}
	line, dashed, emptyFirst := lines("h", "h", "h"), lines("-h", "h", "h"), lines("", "", "-h")
	unread := commandTooLongEntryID
	runVerdictCases(t, []verdictCase{
		{line + `eval "$A$B$C"`, VerdictBlock, syntheticEntryID},
		{line + `"$A$B$C" make`, VerdictBlock, unread},
		{dashed + `git push "$A$B$C" origin main`, VerdictBlock, ""},
		{emptyFirst + `git push "$A$B$C" origin main`, VerdictBlock, unread},
		{line + `git push "$A$B$C" origin main`, VerdictAllow, ""},
		// A name used alone in a word is read with every value it is given.
		{`P=p1; P=p2; P=p3; P=p4; P=p5; P=p6; P=p7; P=p8; P=pkill; $P make`, VerdictBlock, "pkill-by-pattern"},
		{`F=-a; F=-b; F=-c; F=-e; F=-g; F=-h; F=-i; F=-j; F=-k; F=-f; git push "$F" origin main`, VerdictBlock, "git-push-force"},
		// A name given that many values and used inside other text, as data, is not.
		{`M=a; M=b; M=c; M=d; M=e; M=f; M=g; M=h; M=i; echo "made $M"`, VerdictAllow, ""},
		// A name of many values in a word of its own is read with each of them.
		{`for t in go node npm git make gcc jq curl docker; do $t --version; done`, VerdictAllow, ""},
		{`for V in a b c d e f g h /; do rm -rf "$V"; done`, VerdictBlock, "rm-rf-root-or-home"},
		{`for P in a b c d e f g h i j k pkill; do $P make; done`, VerdictBlock, "pkill-by-pattern"},
		// A command -v that is the command's own program looks the name up; one
		// another program runs is no lookup.
		{line + `command -v "$A$B$C"`, VerdictAllow, ""},
		{line + `env command -v "$A$B$C" make`, VerdictBlock, unread},
	})
}

// TestAProgramFromOutputIsUnread: a command whose program is the output of a
// command substitution, directly or through a variable the line set from one, is
// not judged as any program.
func TestAProgramFromOutputIsUnread(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`P=$(echo pkill); $P make`, VerdictBlock, unknownProgramEntryID},
		{`$(echo pkill) make`, VerdictBlock, unknownProgramEntryID},
		{`env $(echo pkill) make`, VerdictBlock, unknownProgramEntryID},
		{`$(echo abcd) update v1.2.3`, VerdictBlock, unknownProgramEntryID},
		{`read -r P; $P make`, VerdictBlock, unknownProgramEntryID},
		{`"$(go env GOROOT)/bin/go" version`, VerdictAllow, ""},
		{`echo $(date)`, VerdictAllow, ""},
		{`$(cat msg.txt)`, VerdictBlock, unknownProgramEntryID},
		{"$(date)", VerdictBlock, unknownProgramEntryID},
		{"P=$(cat f)\ncase x in\n  a) \"$P\" make ;;\nesac", VerdictBlock, unknownProgramEntryID},

		// What a lookup of one fixed name prints is that program's path.
		{`g=$(command -v go 2>/dev/null || true); "$g" build ./...`, VerdictAllow, ""},
		{`g=$(which go); "$g" build ./...`, VerdictAllow, ""},
		{`g=$(command -v pkill); "$g" make`, VerdictBlock, "pkill-by-pattern"},
		{`g=$(command -v "$N"); "$g" make`, VerdictBlock, unknownProgramEntryID},
		{`g=$(command -v go; cat f); "$g" make`, VerdictBlock, unknownProgramEntryID},

		// A substitution with no command in it prints nothing.
		{"``", VerdictAllow, ""},
		{"$( )", VerdictAllow, ""},
		{"`` pkill make", VerdictBlock, ""},

		// A case pattern is matched against, not run.
		{"P=$(cat f)\ncase x in\n  \"$P\"*) true ;;\n  \"$P\"|a) true ;;\nesac", VerdictAllow, ""},
		{"P=$(cat f); case \"$P\" in a) true ;; esac", VerdictAllow, ""},
		{"P=$(cat f); case x in \"$P\") true ;; esac", VerdictAllow, ""},
		{"P=$(cat f); case x in a) \"$P\" make ;; esac", VerdictBlock, unknownProgramEntryID},
		{"P=$(cat f); ( \"$P\" make )", VerdictBlock, unknownProgramEntryID},
		{"P=$(cat f); a=(\"$P\")", VerdictAllow, ""},
	})
}

// TestValuesSpelledAlikeAreReadOnce: a name assigned the same text many
// times, as a script that sets d=$(mktemp -d) for each of its cases does, is
// read with that text once, fed by every command that printed it, so a use of
// it costs one reading however often it is set.
func TestValuesSpelledAlikeAreReadOnce(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 400; i++ {
		fmt.Fprintf(&b, "d=$(mktemp -d); rm -rf \"$d\"; ls \"$d\"/x\n")
	}
	runVerdictCases(t, []verdictCase{
		{b.String(), VerdictAllow, ""},
		{"p=$(cat pids); p=$(pgrep make); kill $p", VerdictBlock, "kill-by-search"},
		{"p=$(pgrep make); p=$(cat pids); kill $p", VerdictBlock, "kill-by-search"},
	})
}

// TestTheWordsSetGivesAreTheArguments: the words a `set` gives the positional
// parameters are what a later `"$@"`, `"$*"` or `"$N"` hands its command.
func TestTheWordsSetGivesAreTheArguments(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`set -- update v1.2.3; abcd "$@"`, VerdictBlock, "abcd-update-tag"},
		{`set -- update v1.2.3; abcd $*`, VerdictBlock, "abcd-update-tag"},
		{`set -- x update v1.2.3; shift; abcd "$@"`, VerdictBlock, "abcd-update-tag"},
		{`set -e -- update v1.2.3; abcd "$1" "$2"`, VerdictBlock, ""},
		{`set -- status; abcd "$@"`, VerdictAllow, ""},
		{`set -e; abcd "$@"`, VerdictAllow, ""},
	})
}

// TestCarriedValuesCostWhatTheLineHolds: the readings with values in place are
// bounded across the line, in number and in bytes, so a line of commands each
// spelled with many combinations of values costs what its words do; past the
// bound, the line is held as unread.
func TestCarriedValuesCostWhatTheLineHolds(t *testing.T) {
	vals := func(name string, width int) string {
		var b strings.Builder
		for i := 0; i < 8; i++ {
			b.WriteString(name + "=" + strings.Repeat(string(rune('a'+i)), width) + "; ")
		}
		return b.String()
	}
	for _, c := range []struct {
		name, head, unit string
	}{
		{"two names of eight values, a bare word", vals("A", 1) + vals("B", 1), "$A$B;"},
		{"two names of eight values, an update", vals("A", 1) + vals("B", 1), `"$A$B" update v1; `},
		{"two names of eight long values", vals("A", 120) + vals("B", 120), `"$A$B" update v1; `},
		{"three names of eight values, eval", vals("A", 1) + vals("B", 1) + vals("C", 1), `eval "$A$B$C"; `},
	} {
		t.Run(c.name, func(t *testing.T) {
			assertWorkGrowth(t, func(n int) string { return c.head + strings.Repeat(c.unit, n) }, 400, c.name)
		})
	}
	d, err := Defaults().Check(vals("A", 1) + vals("B", 1) + strings.Repeat("$A$B;", 3000))
	if err != nil || d.Verdict != VerdictBlock || d.EntryID != commandTooLongEntryID {
		t.Errorf("a line past the bound on readings = %q via %q, %v; want it held as unread", d.Verdict, d.EntryID, err)
	}
	d, err = Defaults().Check("PM=npm; " + strings.Repeat(`"$PM" ci; `, 3000))
	if err != nil || d.Verdict == VerdictBlock {
		t.Errorf("a line of one fixed value used 3000 times = %q via %q, %v; want it read", d.Verdict, d.EntryID, err)
	}
}

// TestALookupALineRedefinesIsUnread: a line that defines a function or an
// alias can give a lookup's name another meaning, so what the lookup prints is
// not read as that program's path.
func TestALookupALineRedefinesIsUnread(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`command() { echo pkill; }; g=$(command -v go); "$g" make`, VerdictBlock, ""},
		{`which() { echo pkill; }; g=$(which go); "$g" make`, VerdictBlock, ""},
		{`type() { echo pkill; }; g=$(type -P go); "$g" make`, VerdictBlock, ""},
		{`function which { echo pkill; }; g=$(which go); "$g" make`, VerdictBlock, ""},
		{`alias which='echo pkill'; g=$(which go); "$g" make`, VerdictBlock, ""},
		{`which() ( echo pkill ); g=$(which go); "$g" make`, VerdictBlock, ""},
		{`g=$(which go); "$g" build ./...`, VerdictAllow, ""},
		{`go() { pkill make; }; g=$(command -v go); "$g" build`, VerdictBlock, ""},
		{`f() { :; }; g=$(command -v go); "$g" build ./...`, VerdictAllow, ""},
		{`eval true; g=$(command -v go); "$g" make`, VerdictBlock, ""},
	})
}

// TestAnUnquotedValueIsSplitIntoWords: a value placed in a word the shell
// splits is read as the words it splits into, so its first word is the
// program; a line that assigns IFS splits it where the guard does not read,
// and holds every such word. A quoted value stays one word.
func TestAnUnquotedValueIsSplitIntoWords(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`P='git prune'; $P`, VerdictBlock, ""},
		{`C='abcd update v1.2.3'; $C`, VerdictBlock, ""},
		{`set -- 'git prune'; $1`, VerdictBlock, ""},
		{`for C in 'git prune'; do $C; done`, VerdictBlock, ""},
		{`C='rm -rf /'; $C`, VerdictBlock, ""},
		{`C='-rf /'; rm $C`, VerdictBlock, ""},
		{`IFS=x; P=gitxprune; $P`, VerdictBlock, ""},
		{`P=gitxprune; IFS=x eval true; $P`, VerdictBlock, ""},
		{`P='git prune'; "$P"`, VerdictAllow, ""},
		{`P='git status'; $P`, VerdictAllow, ""},
		{`P=git; $P status`, VerdictAllow, ""},
		{`: $((IFS=1)); P=git1prune; $P`, VerdictBlock, ""},
		{`n=IFS; read -r "$n" < f; P=gitxprune; $P`, VerdictBlock, ""},

		// What the shell leaves whole, or splits on no more than blanks.
		{`P='git prune'; [[ -n $P ]]`, VerdictAllow, ""},
		{`P='git prune'; case $P in x) true ;; esac`, VerdictAllow, ""},
		{`P='git prune'; Q=$P; echo "$Q"`, VerdictAllow, ""},
		{"while IFS=x read -r a; do echo \"$a\"; done < f; P='git status'; $P", VerdictAllow, ""},
		{"IFS=$'\\n'; P='git status'; $P", VerdictAllow, ""},
		{`o=$IFS; IFS=$'\n'; IFS=$o; P='git status'; $P`, VerdictAllow, ""},
	})
}
