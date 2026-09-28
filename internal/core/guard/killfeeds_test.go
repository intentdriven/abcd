package guard

import (
	"strings"
	"testing"
)

// TestKillFedThroughAGroupIsBlocked — iss-2609262259360005, brace group. A
// group's output is everything its commands print, so `{ pgrep make; } | xargs
// kill` hands kill what pgrep printed. The separator inside the group started a
// new pipeline in the tokenizer, and the pipe after the group then held only
// the group's closing word. A group now restores, at its close, the pipeline it
// opened in; a separator outside the group still breaks the pipe.
func TestKillFedThroughAGroupIsBlocked(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`{ pgrep make; } | xargs kill`, VerdictBlock, "kill-by-search"},
		{`{ pgrep make; true; } | xargs kill`, VerdictBlock, "kill-by-search"},
		{"{ pgrep make\n} | xargs kill", VerdictBlock, "kill-by-search"},
		{`{ pgrep make; } | sort | xargs kill -9`, VerdictBlock, "kill-by-search"},
		{`(pgrep make; true) | xargs kill`, VerdictBlock, "kill-by-search"},
		{`{ { pgrep make; }; true; } | xargs kill`, VerdictBlock, "kill-by-search"},
		{`echo $({ pgrep make; } | xargs kill)`, VerdictBlock, "kill-by-search"},

		{`{ echo 4242; } | xargs kill`, VerdictAllow, ""},
		{`{ pgrep make; } | wc -l`, VerdictAllow, ""},
		{`{ pgrep make; }; echo 4242 | xargs kill`, VerdictAllow, ""},
		{`(pgrep make; true); echo 4242 | xargs kill`, VerdictAllow, ""},
		{`{ pgrep make; echo 4242 | xargs kill; }`, VerdictAllow, ""},
	})
}

// TestKillFedThroughAStringXargsRunsIsBlocked — iss-2609262259360005, the
// xargs payload and the launcher window. xargs hands what it reads to the
// command it runs, and when that command is a shell the pids reach the string
// it runs: through `{}` replaced into it (`-I{}`), or as the positional
// parameters (`"$@"`). A shell also passes its own standard input to the
// commands of its string, so `pgrep make | sh -c 'xargs kill'` is the same
// kill. And a kill behind xargs and an unknown launcher is still fed by the
// pipe into xargs, so the fail-safe warns on it. Every command of a string
// xargs runs is read as handed xargs's input: the guard does not read which of
// them uses it.
func TestKillFedThroughAStringXargsRunsIsBlocked(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`pgrep make | xargs -I{} sh -c 'kill {}'`, VerdictBlock, "kill-by-search"},
		{`pgrep make | xargs bash -c 'kill "$@"' _`, VerdictBlock, "kill-by-search"},
		{`pgrep -f node | xargs -n1 sh -c 'kill -9 "$1"' _`, VerdictBlock, "kill-by-search"},
		{`pgrep make | xargs sudo sh -c 'kill "$@"' _`, VerdictBlock, "kill-by-search"},
		{`xargs -a <(pgrep make) sh -c 'kill "$@"' _`, VerdictBlock, "kill-by-search"},
		{`pgrep make | xargs sh -c "sh -c 'kill \$1' _ \$1" _`, VerdictBlock, "kill-by-search"},
		{`pgrep make | xargs -I{} eval kill {}`, VerdictBlock, "kill-by-search"},
		{`pgrep make | sh -c 'xargs kill'`, VerdictBlock, "kill-by-search"},
		{`pgrep make | bash -c 'sort | xargs kill'`, VerdictBlock, "kill-by-search"},
		{`pgrep make | xargs myrunner kill`, VerdictWarn, speculativeEntryID},
		{`pgrep make | xargs -n1 myrunner kill -9`, VerdictWarn, speculativeEntryID},
		{`xargs -a <(pgrep make) myrunner kill`, VerdictWarn, speculativeEntryID},

		{`echo 4242 | xargs sh -c 'kill "$@"' _`, VerdictAllow, ""},
		{`xargs -I{} sh -c 'kill {}' < pidfile`, VerdictAllow, ""},
		{`pgrep make | xargs sh -c 'echo "$@"' _`, VerdictAllow, ""},
		{`pgrep make | sh -c 'wc -l'`, VerdictAllow, ""},
		{`echo 4242 | sh -c 'xargs kill'`, VerdictAllow, ""},
		{`pgrep make; sh -c 'xargs kill' < pidfile`, VerdictAllow, ""},
		{`pgrep make | xargs myrunner echo`, VerdictAllow, ""},
		{`echo 4242 | xargs myrunner kill`, VerdictAllow, ""},
	})
}

// TestKillFedBySearchInsideAStringIsBlocked — iss-2609262259360005, the
// search inside a shell string. `kill $(sh -c 'pgrep make')` prints what
// pgrep printed, but the string's commands are read after the line is split,
// outside the run the substitution recorded. A command's own string is now
// part of what the command ran, wherever its output is read.
func TestKillFedBySearchInsideAStringIsBlocked(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`kill $(sh -c 'pgrep make')`, VerdictBlock, "kill-by-search"},
		{`kill $(bash -c "pgrep -f node")`, VerdictBlock, "kill-by-search"},
		{`kill $(eval pgrep make)`, VerdictBlock, "kill-by-search"},
		{`kill $(sh -c "sh -c 'pgrep make'")`, VerdictBlock, "kill-by-search"},
		{`sh -c 'pgrep make' | xargs kill`, VerdictBlock, "kill-by-search"},
		{`kill -9 $(sudo sh -c 'pgrep make | head -1')`, VerdictBlock, "kill-by-search"},

		{`kill $(sh -c 'cat pidfile')`, VerdictAllow, ""},
		{`echo $(sh -c 'pgrep make')`, VerdictAllow, ""},
		{`sh -c 'pgrep make'; echo 4242 | xargs kill`, VerdictAllow, ""},
		{`sh -c 'pgrep make' | wc -l`, VerdictAllow, ""},
	})
}

// TestAttachedSelectorsAreRead — iss-2609262259360005, the attached selector
// values. A selector's value glued to its letter is still that selector: the
// first letter after a single dash is always an option, and a byte that is no
// option letter (`/`, `.`) is its value, so `pkill -tpts/3` is `pkill -t
// pts/3`. The upper-case selectors (`-U` real user, `-G` real group) are read
// attached too: a signal name is recognised first, case folded and with or
// without its SIG prefix, as procps-ng and BSD pkill read it, so `-HUP`,
// `-USR1`, `-SEGV` and `-SIGTERM` stay signals, and so do the lower-case
// spellings `-term`, `-hup`, `-int` and `-stop`, which the letter reading had
// taken for a terminal or user selector.
func TestAttachedSelectorsAreRead(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`pkill -tpts/3`, VerdictBlock, "pkill-by-owner"},
		{`pkill -9 -tpts/3`, VerdictBlock, "pkill-by-owner"},
		{`pkill -ubob.smith`, VerdictBlock, "pkill-by-owner"},
		{`pkill -Ubob.smith`, VerdictBlock, "pkill-by-owner"},
		{`killall -ubob.smith`, VerdictBlock, "killall-by-owner"},
		{`kill $(pgrep -ubob.smith)`, VerdictBlock, "kill-by-search"},
		{`pkill -Ubob`, VerdictBlock, "pkill-by-owner"},
		{`pkill -Gstaff`, VerdictBlock, "pkill-by-owner"},
		{`pkill -U 501`, VerdictBlock, "pkill-by-owner"},
		{`pkill -G staff`, VerdictBlock, "pkill-by-owner"},
		{`pkill -HUP -Ubob`, VerdictBlock, "pkill-by-owner"},
		{`kill $(pgrep -Ubob)`, VerdictBlock, "kill-by-search"},
		{`pgrep -Gstaff | xargs kill`, VerdictBlock, "kill-by-search"},
		{`pkill -term -u bob`, VerdictBlock, "pkill-by-owner"},
		{`pkill -term -tpts/3`, VerdictBlock, "pkill-by-owner"},
		{`pkill -hup make`, VerdictBlock, "pkill-by-pattern"},
		{`git commit -nm"fix: the parser"`, VerdictBlock, "git-commit-no-verify"},

		{`pkill -term -g 4242`, VerdictAllow, ""},
		{`pkill -hup -P $$`, VerdictAllow, ""},
		{`pkill -int -g 4242`, VerdictAllow, ""},
		{`pkill -stop -g 4242`, VerdictAllow, ""},
		{`pkill -sigterm -g 4242`, VerdictAllow, ""},
		{`pkill -SIGTERM -P $$`, VerdictAllow, ""},
		{`pkill -SIGUSR1 -g 4242`, VerdictAllow, ""},
		{`pkill -Usr1 -g 4242`, VerdictAllow, ""},
		{`pkill -HUP -g 4242`, VerdictAllow, ""},
		{`pkill -SEGV -P $$`, VerdictAllow, ""},
		{`pkill -kill -g 4242`, VerdictAllow, ""},
		{`pgrep -Ubob`, VerdictAllow, ""},
		{`git commit -m"new feature"`, VerdictAllow, ""},
		{`git push -oci.skip origin main`, VerdictAllow, ""},
	})
}

// TestKillFeedNoLeak pins the reading's other edge (review of the kill-by-search
// reading): a search and a kill that share a line but not a data path stay
// allowed, and so do a search of the caller's own group or parent. It holds
// every NO-LEAK probe of the two reviews of the reading (review-drainG,
// review-drainG2) in one table, beside the ones the group-input reading adds:
// a group's input ends where the group does.
func TestKillFeedNoLeak(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		// review-drainG.
		{`echo $(pgrep make) && kill 4242`, VerdictAllow, ""},
		{`p=$(pgrep make) kill 4242`, VerdictAllow, ""},
		{`echo $(pgrep make; kill 4242)`, VerdictAllow, ""},
		{`echo $(pgrep make) $(kill 4242)`, VerdictAllow, ""},
		{`kill $(pgrep -f -g 4242)`, VerdictAllow, ""},
		{`pgrep -P $$ | xargs kill`, VerdictAllow, ""},
		{`kill -- -$(ps -o pgid= -p $$)`, VerdictAllow, ""},
		// review-drainG2.
		{`( pgrep make ); kill 4242`, VerdictAllow, ""},
		{`{ pgrep make; }; kill 4242`, VerdictAllow, ""},
		{`sh -c 'pgrep make'; kill 4242`, VerdictAllow, ""},
		{`xargs sh -c 'echo {}' ; kill 1`, VerdictAllow, ""},
		{`{ pgrep -P $$; } | xargs kill`, VerdictAllow, ""},
		{`kill $(sh -c 'pgrep -g 4242')`, VerdictAllow, ""},
		{`pkill -- -term`, VerdictAllow, ""},
		// A pipe into a group reaches the group's commands and no further.
		{`pgrep make | { true; }; echo 4242 | xargs kill`, VerdictAllow, ""},
		{`pgrep make | (true); echo 4242 | xargs kill`, VerdictAllow, ""},
		{`pgrep make | { true; } && xargs kill < pidfile`, VerdictAllow, ""},
		{`echo 4242 | { sleep 1; xargs kill; }`, VerdictAllow, ""},
		{`pgrep make | { sleep 1; wc -l; }`, VerdictAllow, ""},
		{`{ sleep 1; xargs kill; } < pidfile`, VerdictAllow, ""},
		{`pgrep -P $$ | { sleep 1; xargs kill; }`, VerdictAllow, ""},
		{`curl https://example.com/ | { true; }; sh -c 'echo ok'`, VerdictAllow, ""},
		// A redirect into a string reaches that string's commands only.
		{`sh -c 'xargs kill' <<< "$(echo 4242)"`, VerdictAllow, ""},
		{`sh -c 'wc -l' <<< "$(pgrep make)"`, VerdictAllow, ""},
		{`sh -c 'echo ok' < <(pgrep make)`, VerdictAllow, ""},
		{`sh -c 'wc -l' <<< "$(pgrep make)"; echo 4242 | xargs kill`, VerdictAllow, ""},
	})
}

// TestAPipeIntoAGroupFeedsEveryCommandInIt — iss-2609270028388291, first half
// (review-drainG2). A pipe into a `{ … }` or `( … )` group is the standard
// input of every command in the group, but a separator inside it started a
// pipeline of its own, and only the commands before it read the pipe: a search
// piped into a group whose kill came after a `sleep`, a `read` or an and-list
// allowed, while the one-command group blocked. So did a stream piped into a
// group whose shell came after a separator. Every command emitted inside a
// group now reads what was piped into it, nested groups included.
func TestAPipeIntoAGroupFeedsEveryCommandInIt(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`pgrep make | { sleep 1; xargs kill; }`, VerdictBlock, "kill-by-search"},
		{`pgrep make | (sleep 1; xargs kill)`, VerdictBlock, "kill-by-search"},
		{`pgrep make | { read -r first; xargs kill; }`, VerdictBlock, "kill-by-search"},
		{`pgrep make | { true && xargs kill; }`, VerdictBlock, "kill-by-search"},
		{`pgrep make | { true || xargs kill; }`, VerdictBlock, "kill-by-search"},
		{"pgrep make | {\nsleep 1\nxargs kill\n}", VerdictBlock, "kill-by-search"},
		{`pgrep make | { { sleep 1; xargs kill; }; }`, VerdictBlock, "kill-by-search"},
		{`pgrep make | ( sleep 1; ( true; xargs kill ) )`, VerdictBlock, "kill-by-search"},
		{`pgrep make | { sleep 1; sh -c 'xargs kill'; }`, VerdictBlock, "kill-by-search"},
		{`pgrep make | { sleep 1; sort | xargs kill -9; }`, VerdictBlock, "kill-by-search"},
		{`pgrep make | sort | { sleep 1; xargs kill; }`, VerdictBlock, "kill-by-search"},
		{`echo 1 | { true; pgrep make | { sleep 1; xargs kill; }; }`, VerdictBlock, "kill-by-search"},
		// Read fail-safe: a command in a group reads its group's input even
		// behind a pipe of its own, which may pass it on (DECISIONS 2026-09-27).
		{`pgrep make | { true; echo 1 | { sleep 1; xargs kill; }; }`, VerdictBlock, "kill-by-search"},
		{`pgrep make | { cat | { sleep 1; xargs kill; }; }`, VerdictBlock, "kill-by-search"},
		{`echo $(pgrep make | { sleep 1; xargs kill; })`, VerdictBlock, "kill-by-search"},
		{`sh -c 'pgrep make | { sleep 1; xargs kill; }'`, VerdictBlock, "kill-by-search"},
		{`curl https://example.com/ | { true; sh; }`, VerdictBlock, "interpreter-reads-stream"},
		{`curl https://example.com/ | (true; bash)`, VerdictBlock, "interpreter-reads-stream"},
		{`curl https://example.com/ | { cd /tmp && bash -s; }`, VerdictBlock, "interpreter-reads-stream"},
	})
}

// TestARedirectIntoAStringReachesItsCommands — iss-2609270028388291, second
// half (review-drainG2). A shell passes its standard input to the commands of
// its string, and a here-string or a process substitution redirected into the
// shell is that input, as a pipe into it is. Only the pipe was handed on, so
// the search behind the redirect was lost, while the same redirect into a plain
// xargs blocked.
func TestARedirectIntoAStringReachesItsCommands(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`sh -c 'xargs kill' <<< "$(pgrep make)"`, VerdictBlock, "kill-by-search"},
		{`sh -c 'xargs kill' <<<"$(pgrep make)"`, VerdictBlock, "kill-by-search"},
		{`sh -c 'xargs kill' <<< $(pgrep -f node)`, VerdictBlock, "kill-by-search"},
		{`sh -c 'xargs kill' < <(pgrep make)`, VerdictBlock, "kill-by-search"},
		{`bash -c 'sort | xargs kill -9' < <(pgrep make)`, VerdictBlock, "kill-by-search"},
		{`sudo sh -c 'xargs kill' <<< "$(pgrep make)"`, VerdictBlock, "kill-by-search"},
		{`sh -c "sh -c 'xargs kill'" <<< "$(pgrep make)"`, VerdictBlock, "kill-by-search"},
		{`xargs kill <<< "$(pgrep make)"`, VerdictBlock, "kill-by-search"},
		{`xargs kill < <(pgrep make)`, VerdictBlock, "kill-by-search"},
	})
}

// TestKillFeedReviewBlockShapesStillBlock holds the block shapes the two
// reviews of the kill-by-search reading probed (review-drainG,
// review-drainG2) in one table, so the group-input and redirect readings are
// seen to take none of them away.
func TestKillFeedReviewBlockShapesStillBlock(t *testing.T) {
	var cases []verdictCase
	for _, cmd := range []string{
		`kill $( $(pgrep make) )`,
		`kill "$(pgrep make)"`,
		"kill `pgrep make`",
		`kill -9 $(pgrep -f make) 4242`,
		`pgrep make | tee /dev/null | xargs kill`,
		`pgrep make | xargs -I{} kill {}`,
		`pgrep make | xargs -0 kill`,
		`pgrep make | xargs -d , kill`,
		`pgrep make | xargs -n 1 kill`,
		`pgrep make | xargs -P 4 kill`,
		`pgrep make | xargs -- kill`,
		`pgrep make | xargs -t kill`,
		`pgrep make | xargs -L 1 kill`,
		`pgrep make | xargs -s 1024 kill`,
		`pgrep make | xargs -E end kill`,
		`pgrep make | xargs -i kill {}`,
		`pgrep make | xargs -I% kill %`,
		`kill $(cat <(pgrep make))`,
		`xargs kill <<< "$(pgrep make)"`,
		`kill ${pids:-$(pgrep make)}`,
		`kill $(( $(pgrep make) + 0 ))`,
		`pgrep make | { xargs kill; }`,
		`pgrep make | (xargs kill)`,
		`pgrep make | xargs env kill`,
		`pgrep make | xargs nice kill`,
		`pgrep make | xargs sudo kill`,
		`pgrep make | xargs timeout 5 kill`,
		`pgrep make | xargs busybox kill`,
		`pgrep make | xargs /bin/kill`,
		`echo $({ pgrep make; } | xargs kill)`,
		`pgrep make 2>&1 | xargs kill`,
		`xargs -a <(pgrep make) kill`,
		`sh -c 'exec kill $(pgrep make)'`,
		`sh -c 'command kill $(pgrep make)'`,
		`sh -c 'builtin kill $(pgrep make)'`,
		`sh -c "sh -c 'kill \$(pgrep make)'"`,
		`kill $(sh -c 'kill $(pgrep make)')`,
		`pgrep make | sh -c 'sleep 1; xargs kill'`,
		`( pgrep make ) | xargs kill`,
		`{ pgrep make; } | xargs kill`,
	} {
		cases = append(cases, verdictCase{cmd, VerdictBlock, "kill-by-search"})
	}
	runVerdictCases(t, cases)
}

// TestBSDXargsValueFlagsAreStepped — iss-2609270028432249. The xargs of macOS
// and the BSDs takes a value after `-J` (the replacement string), `-R` (the
// most replacements) and `-S` (the replacement size). The walk did not know
// them, so it read the value as the command xargs runs and warned on an
// unrecognised launcher instead of reading the kill behind it.
func TestBSDXargsValueFlagsAreStepped(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`pgrep make | xargs -J % kill %`, VerdictBlock, "kill-by-search"},
		{`pgrep make | xargs -R 5 -I{} kill {}`, VerdictBlock, "kill-by-search"},
		{`pgrep make | xargs -S 1024 -I{} kill {}`, VerdictBlock, "kill-by-search"},
		{`pgrep make | xargs -J % -R 2 kill -9 %`, VerdictBlock, "kill-by-search"},

		{`echo 4242 | xargs -J % kill %`, VerdictAllow, ""},
		{`ls | xargs -J % cp % /tmp/`, VerdictAllow, ""},
	})
}

// TestSignalTableHoldsOnlySignals — review-drainG2 (INFO). The signal names
// pkill's first `-NAME` word is read by are the ones a pkill accepts: a word no
// pkill takes for a signal is a cluster of options, so `pkill -null` is `-n -u
// ll` and BSD's `pkill -unused` is `-u nused`, both by owner.
func TestSignalTableHoldsOnlySignals(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`pkill -null -g 1`, VerdictBlock, "pkill-by-owner"},
		{`pkill -unused -g 1`, VerdictBlock, "pkill-by-owner"},
		{`pkill -term -g 1`, VerdictAllow, ""},
		{`pkill -SIGINFO -g 1`, VerdictAllow, ""},
	})
}

// TestKillFeedGroupsAndStringsStayLinear holds the group and string readings
// to the cost class the rest of the guard keeps (work_test.go): nested groups,
// a pipeline of groups, strings in the substitutions a kill reads, a
// launcher's windows behind one xargs, and a pipeline of strings xargs runs.
func TestKillFeedGroupsAndStringsStayLinear(t *testing.T) {
	shapes := map[string]func(int) string{
		"nested groups": func(n int) string {
			return strings.Repeat("{ ", n) + "pgrep make" + strings.Repeat("; }", n) + " | xargs kill"
		},
		"a pipeline of groups": func(n int) string {
			return "pgrep make" + strings.Repeat(" | { xargs kill; }", n)
		},
		"a kill of many strings": func(n int) string {
			return "kill" + strings.Repeat(" $(sh -c 'pgrep make')", n)
		},
		"kills behind xargs and a launcher": func(n int) string {
			return "echo 4242 | xargs myrunner" + strings.Repeat(" kill $(echo 1)", n)
		},
		"a pipeline of xargs strings": func(n int) string {
			return "pgrep make" + strings.Repeat(` | xargs sh -c 'kill "$@"' _`, n)
		},
		"a pipe into nested groups": func(n int) string {
			return "pgrep make" + strings.Repeat(" | { true; ", n) + "xargs kill" + strings.Repeat("; }", n)
		},
		"pipes into disjoint nested groups": func(n int) string {
			return strings.Repeat("echo 1 | { true; ", n) + "xargs kill" + strings.Repeat("; }", n)
		},
		"strings under here-strings": func(n int) string {
			return strings.Repeat(`sh -c 'xargs kill' <<< "$(echo 1)"; `, n)
		},
	}
	for name, build := range shapes {
		build := build
		t.Run(name, func(t *testing.T) {
			assertWorkGrowth(t, build, 1<<8, "a kill's feeds are counted once per list")
		})
	}
}

// TestArgsReaderReadsEachWordOnce pins the argsReader's contract directly
// (review-drainG2, finding 5): the words after an xargs are read once however
// many places are asked about, left to right. The growth shapes above do not
// catch a reader that re-reads the words per place, because the speculation
// caps turn that re-read into a constant factor; this counts the reader alone.
func TestArgsReaderReadsEachWordOnce(t *testing.T) {
	segs, err := tokenize("echo 4242 | xargs myrunner" + strings.Repeat(" kill $(echo 1)", 300))
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	var s segment
	found := false
	for _, seg := range segs {
		if len(seg.tokens) > 0 && seg.tokens[0] == "xargs" {
			s, found = seg, true
			break
		}
	}
	if !found {
		t.Fatal("no xargs segment in the shape")
	}
	n := 0
	workTally = &n
	defer func() { workTally = nil }()
	r := newArgsReader(s)
	for i := 0; i <= len(s.tokens); i++ {
		r.before(i)
	}
	if n > 2*len(s.tokens) {
		t.Errorf("asking every place of a %d-word segment counted %d units, want at most %d: the words are read once", len(s.tokens), n, 2*len(s.tokens))
	}
}
