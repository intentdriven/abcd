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
// allowed, and so do a search of the caller's own group or parent.
func TestKillFeedNoLeak(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`echo $(pgrep make) && kill 4242`, VerdictAllow, ""},
		{`p=$(pgrep make) kill 4242`, VerdictAllow, ""},
		{`echo $(pgrep make; kill 4242)`, VerdictAllow, ""},
		{`echo $(pgrep make) $(kill 4242)`, VerdictAllow, ""},
		{`kill $(pgrep -f -g 4242)`, VerdictAllow, ""},
		{`pgrep -P $$ | xargs kill`, VerdictAllow, ""},
		{`kill -- -$(ps -o pgid= -p $$)`, VerdictAllow, ""},
		{`{ pgrep -P $$; } | xargs kill`, VerdictAllow, ""},
		{`kill $(sh -c 'pgrep -g 4242')`, VerdictAllow, ""},
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
	}
	for name, build := range shapes {
		build := build
		t.Run(name, func(t *testing.T) {
			assertWorkGrowth(t, build, 1<<8, "a kill's feeds are counted once per list")
		})
	}
}
