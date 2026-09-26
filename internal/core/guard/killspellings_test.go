package guard

import (
	"strings"
	"testing"
)

// TestKillFedByAProcessSearchIsBlocked — iss-2609251640452031, first half. A
// `kill` handed the pids a name search prints is a kill by name: `kill $(pgrep
// -f make)` and `pgrep -f make | xargs kill` signal every process `pkill -f
// make` does. Read alone, each is a bare `kill` of an unknown pid list and a
// harmless `pgrep`, so the kill entries never saw the pattern. The registry now
// names the search a kill's arguments come from (args_from), and the tokenizer
// records where a word's substitution and a command's piped input come from.
// The search alone, a kill of a literal or recorded pid, and a search that
// names no pattern (its own group) stay allowed.
func TestKillFedByAProcessSearchIsBlocked(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		// A command substitution in the kill's words.
		{`kill $(pgrep -f 'make preflight')`, VerdictBlock, "kill-by-search"},
		{`kill -9 $(pgrep make)`, VerdictBlock, "kill-by-search"},
		{`kill "$(pgrep -f node)"`, VerdictBlock, "kill-by-search"},
		{"kill `pgrep make`", VerdictBlock, "kill-by-search"},
		{`kill -s TERM $(pgrep -x make)`, VerdictBlock, "kill-by-search"},
		{`kill -- $(pgrep make)`, VerdictBlock, "kill-by-search"},
		{`sudo kill $(pgrep make)`, VerdictBlock, "kill-by-search"},
		{`kill $(pgrep -f make | head -1)`, VerdictBlock, "kill-by-search"},
		{`kill $(echo $(pgrep make))`, VerdictBlock, "kill-by-search"},
		{`kill $(pidof make)`, VerdictBlock, "kill-by-search"},
		{`kill $(pgrep -u bob)`, VerdictBlock, "kill-by-search"},
		{`kill $(pgrep make) 2>/dev/null`, VerdictBlock, "kill-by-search"},
		{`cd /tmp && kill $(pgrep make)`, VerdictBlock, "kill-by-search"},
		{`sh -c 'kill $(pgrep make)'`, VerdictBlock, "kill-by-search"},
		{`kill $(sudo pgrep -f make)`, VerdictBlock, "kill-by-search"},
		{`kill $(( $(pgrep make) ))`, VerdictBlock, "kill-by-search"},
		{`kill {$(pgrep make),}`, VerdictBlock, "kill-by-search"},
		{`PGREP make | XARGS KILL`, VerdictBlock, "kill-by-search"},
		{`/usr/bin/pgrep make | /usr/bin/xargs /bin/kill`, VerdictBlock, "kill-by-search"},
		{`pgre? make | xargs kill`, VerdictBlock, "kill-by-search"},
		// The search piped into xargs, which hands its output to kill.
		{`pgrep -f 'make preflight' | xargs kill`, VerdictBlock, "kill-by-search"},
		{`pgrep make | xargs kill -9`, VerdictBlock, "kill-by-search"},
		{`(pgrep make) | xargs kill`, VerdictBlock, "kill-by-search"},
		{`pgrep make | tee /dev/null | xargs -I{} kill {}`, VerdictBlock, "kill-by-search"},
		{`xargs -a <(pgrep make) kill`, VerdictBlock, "kill-by-search"},
		{`pgrep -f node | "$(true)"xargs kill`, VerdictBlock, "kill-by-search"},
		// Behind a launcher the guard does not know, the fail-safe warns.
		{`myrunner kill $(pgrep make)`, VerdictWarn, speculativeEntryID},
		{`pgrep make | myrunner xargs kill`, VerdictWarn, speculativeEntryID},
		{`pgrep -f node | xargs -n1 kill`, VerdictBlock, "kill-by-search"},
		{`pgrep make | sort | xargs kill`, VerdictBlock, "kill-by-search"},
		{`pgrep make | xargs sudo kill`, VerdictBlock, "kill-by-search"},
		{`pgrep make |& xargs kill`, VerdictBlock, "kill-by-search"},
		{`pidof make | xargs kill`, VerdictBlock, "kill-by-search"},
		{`pgrep -f x | xargs -r kill -TERM`, VerdictBlock, "kill-by-search"},
		{`echo $(pgrep make) | xargs kill`, VerdictBlock, "kill-by-search"},
		{"pgrep make |\nxargs kill", VerdictBlock, "kill-by-search"},

		// Near misses: the search alone, a literal or recorded pid, a search
		// with no pattern, and a pipe that does not reach kill.
		{`pgrep -l make`, VerdictAllow, ""},
		{`pgrep -f 'make preflight'`, VerdictAllow, ""},
		{`pidof make`, VerdictAllow, ""},
		{`kill 4242`, VerdictAllow, ""},
		{`kill -- -4242`, VerdictAllow, ""},
		{`kill $(cat pidfile)`, VerdictAllow, ""},
		{`kill $(pgrep -g 4242)`, VerdictAllow, ""},
		{`kill $(pgrep -P $$)`, VerdictAllow, ""},
		{`echo $(pgrep make)`, VerdictAllow, ""},
		{`pgrep make | wc -l`, VerdictAllow, ""},
		{`pgrep make | xargs echo`, VerdictAllow, ""},
		{`pgrep make; kill 4242`, VerdictAllow, ""},
		{`pgrep make && kill 4242`, VerdictAllow, ""},
		{`pgrep make; echo 4242 | xargs kill`, VerdictAllow, ""},
		{"pgrep make\necho 4242 | xargs kill", VerdictAllow, ""},
		{`cat pidfile | xargs kill`, VerdictAllow, ""},
		{`xargs kill < pidfile`, VerdictAllow, ""},
		{`pgrep make | kill 4242`, VerdictAllow, ""},
		{`kill 4242; echo $(pgrep make)`, VerdictAllow, ""},
	})
}

// TestPkillAndKillallByOwnerAreBlocked — iss-2609251640452031, second half. A
// kill selecting by user or terminal carries no pattern operand: the selector
// was listed as a value flag, which consumed it, so the count read zero and
// `pkill -u bob` — every process that user owns, every session of theirs —
// answered allow. The population selectors (user, terminal, group) are no
// longer stepped over, so the selector counts as what the kill is by, and the
// attached spellings (`-ubob`, `--euid=bob`) that stand as one flag word are
// read by their own entries. The own-group and own-parent routes stay
// allowed, a signal name that happens to hold a selector's letter included.
func TestPkillAndKillallByOwnerAreBlocked(t *testing.T) {
	runVerdictCases(t, []verdictCase{
		{`pkill -u bob`, VerdictBlock, "pkill-by-owner"},
		{`pkill -u $(whoami)`, VerdictBlock, "pkill-by-owner"},
		{`pkill -ubob`, VerdictBlock, "pkill-by-owner"},
		{`pkill --euid=bob`, VerdictBlock, "pkill-by-owner"},
		{`pkill --euid bob`, VerdictBlock, "pkill-by-owner"},
		{`pkill --uid=501`, VerdictBlock, "pkill-by-owner"},
		{`pkill -t pts/3`, VerdictBlock, "pkill-by-owner"},
		{`pkill -tttys001`, VerdictBlock, "pkill-by-owner"},
		{`pkill --terminal=ttys001`, VerdictBlock, "pkill-by-owner"},
		{`pkill -9 -t ttys001`, VerdictBlock, "pkill-by-owner"},
		{`pkill --group=staff`, VerdictBlock, "pkill-by-owner"},
		{`sudo pkill -u bob`, VerdictBlock, "pkill-by-owner"},
		{`pkill -U 501`, VerdictBlock, "pkill-by-pattern"},
		{`pkill -G staff`, VerdictBlock, "pkill-by-pattern"},
		{`killall -u bob`, VerdictBlock, "killall-by-name"},
		{`killall -ubob`, VerdictBlock, "killall-by-owner"},
		{`killall --user=bob`, VerdictBlock, "killall-by-owner"},
		{`killall -t ttys001`, VerdictBlock, "killall-by-name"},
		{`killall -c make`, VerdictBlock, "killall-by-name"},

		{`pkill -g 4242`, VerdictAllow, ""},
		{`pkill -P $$`, VerdictAllow, ""},
		{`pkill -g $(cat pgid)`, VerdictAllow, ""},
		{`pkill -HUP -P $$`, VerdictAllow, ""},
		{`pkill -USR1 -g 4242`, VerdictAllow, ""},
		{`pkill -QUIT -g 4242`, VerdictAllow, ""},
		{`pkill -SEGV -g 4242`, VerdictAllow, ""},
		{`pgrep -u bob`, VerdictAllow, ""},
		{`killall -l`, VerdictAllow, ""},
	})
}

// TestKillFeedReadingStaysLinear holds the new reading to the cost class the
// rest of the guard keeps (work_test.go). A feed names a run of the tokenizer's
// output rather than copying it, and each list counts its matching commands
// once per entry, so a kill nested in a kill, a pipeline of kills, a kill with
// many substitutions, and a launcher's windows over them each cost what the
// line's bytes do.
func TestKillFeedReadingStaysLinear(t *testing.T) {
	shapes := map[string]func(int) string{
		"nested kills": func(n int) string {
			return strings.Repeat("kill $(", n) + "pgrep make" + strings.Repeat(")", n)
		},
		"a pipeline of kills": func(n int) string {
			return "pgrep make" + strings.Repeat(" | xargs kill", n)
		},
		"a kill of many searches": func(n int) string {
			return "kill" + strings.Repeat(" $(pgrep make)", n)
		},
		"kills behind a launcher": func(n int) string {
			return "myrunner" + strings.Repeat(" kill $(echo 1)", n)
		},
		"quoted kills": func(n int) string {
			return strings.Repeat(`kill "$(pgrep make)"; `, n)
		},
	}
	for name, build := range shapes {
		build := build
		t.Run(name, func(t *testing.T) {
			assertWorkGrowth(t, build, 1<<8, "a kill's feeds are counted once per list")
		})
	}
}

// TestArgsFromConstraint pins the pattern field kill-by-search uses: a source
// is a plain pattern held to the same load-time checks, a repo override
// replaces the list whole, a change to a blocker's sources is a weakening that
// waits for HEAD, and a cloned registry shares no source with the bundled one.
func TestArgsFromConstraint(t *testing.T) {
	entry := func(p Pattern) Registry {
		return Registry{SchemaVersion: 1, Entries: map[string]Entry{"x": {
			Tier: TierBlocker, Why: "w", Successor: "s", Pattern: p,
		}}}
	}
	yes := true
	for name, p := range map[string]Pattern{
		"a source with no command":  {Command: "kill", ArgsFrom: []Pattern{{}}},
		"a source naming a path":    {Command: "kill", ArgsFrom: []Pattern{{Command: "/usr/bin/pgrep"}}},
		"a source with sources":     {Command: "kill", ArgsFrom: []Pattern{{Command: "pgrep", ArgsFrom: []Pattern{{Command: "ps"}}}}},
		"a source after a cd":       {Command: "kill", ArgsFrom: []Pattern{{Command: "pgrep", AfterCD: &yes}}},
		"a source with a bad count": {Command: "kill", ArgsFrom: []Pattern{{Command: "pgrep", MinOperands: -1}}},
	} {
		if err := Validate(entry(p)); err == nil {
			t.Errorf("%s: Validate accepted it; a source nothing can match must be refused at load", name)
		}
	}
	if err := Validate(entry(Pattern{Command: "kill", ArgsFrom: []Pattern{{Command: "pgrep", MinOperands: 1}}})); err != nil {
		t.Errorf("a plain source was refused: %v", err)
	}

	over := Registry{SchemaVersion: 1, Entries: map[string]Entry{"kill-by-search": {
		Pattern: Pattern{ArgsFrom: []Pattern{{Command: "pidof", MinOperands: 1}}},
	}}}
	merged := Merge(Defaults(), over)
	if got := merged.Entries["kill-by-search"].Pattern.ArgsFrom; len(got) != 1 || got[0].Command != "pidof" {
		t.Errorf("an override's args_from replaces the list whole; got %+v", got)
	}
	if what := weakening(Defaults(), merged); !strings.Contains(what, "kill-by-search") {
		t.Errorf("narrowing a blocker's sources is a weakening; weakening() = %q", what)
	}

	r := Defaults()
	r.Entries["kill-by-search"].Pattern.ArgsFrom[0].ValueFlags[0] = "mutated"
	if Defaults().Entries["kill-by-search"].Pattern.ArgsFrom[0].ValueFlags[0] == "mutated" {
		t.Error("Defaults() shares a source's slices with the bundled registry")
	}
}
