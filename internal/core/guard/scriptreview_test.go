package guard

import (
	"path/filepath"
	"strings"
	"testing"
)

// Review findings on the script reading (2026-10-09).
func TestScriptReadingReviewFindings(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"e":        hazardLine,
		"s.sh":     hazardLine,
		"a.sh":     "bash b.sh",
		"b.sh":     "bash c.sh",
		"c.sh":     "echo deep",
		"sub/keep": "",
	})
	runScriptCases(t, dir, []scriptCase{
		// declare -x and typeset -x export as export does.
		{`declare -x BASH_ENV={d}/e; bash -c true`, VerdictBlock, scriptHazardEntryID},
		{`typeset -x BASH_ENV={d}/e; bash -c true`, VerdictBlock, scriptHazardEntryID},
		{`declare -gx BASH_ENV={d}/e; bash -c true`, VerdictBlock, scriptHazardEntryID},
		{`BASH_ENV={d}/e; declare -x BASH_ENV; bash -c true`, VerdictBlock, scriptHazardEntryID},
		{`export BASH_ENV={d}/e; export -n BASH_ENV; bash -c true`, VerdictAllow, ""},
		// ln writes its target as cp does.
		{`ln -sf {d}/s.sh {d}/l.sh && bash {d}/l.sh`, VerdictBlock, scriptWrittenEntryID},
		// A single operand links the target's own name into the directory ln
		// runs in, and reading the source is not that write.
		{`cd {d}/sub && ln -s {d}/s.sh && bash s.sh`, VerdictBlock, scriptWrittenEntryID},
		{`cd {d}/sub && ln {d}/s.sh && ./s.sh`, VerdictBlock, scriptWrittenEntryID},
		{`cd {d}/sub && ln -s {d}/s.sh && bash {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		// A chain past the depth names the script and its own fix.
		{`bash a.sh`, VerdictBlock, scriptUnreadEntryID},
	})
	d, err := readingRegistry(dir, "").Check("bash " + filepath.Join(dir, "a.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(d.Successor, "inner script directly") || strings.Contains(d.Successor, "sh -c`/`env -S") {
		t.Errorf("depth successor = %q, want the script's own fix", d.Successor)
	}
}

// A shell under noexec reads its script and runs none of it, so a syntax
// check (`bash -n s.sh`) is no run of the hazard the script holds; an
// interactive shell ignores the option, and whatever turns execution back on
// after it, a `+n`, zsh's and ksh's `--exec`, a `+-` long option, an `exec`
// option by name, leaves a run to judge (iss-2610262219309883).
func TestASyntaxCheckIsNoRunOfTheScript(t *testing.T) {
	dir := scriptTree(t, map[string]string{"s.sh": hazardLine})
	runScriptCases(t, dir, []scriptCase{
		{`bash -n {d}/s.sh`, VerdictAllow, ""},
		{`bash -o noexec {d}/s.sh`, VerdictAllow, ""},
		{`sh -n {d}/s.sh`, VerdictAllow, ""},
		{`zsh -n {d}/s.sh`, VerdictAllow, ""},
		{`bash -n +n {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`bash -n -i {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`zsh -n --exec {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`ksh -n --exec {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`zsh -f -n +-noexec {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`zsh -n -o exec {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`zsh -n -o no_no_exec {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`bash {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
	})
}

// A `+-` long option is one option, not a cluster of letters: its `o` takes
// no word after it, so the script that follows is still read and judged.
func TestAPlusDashLongOptionLeavesTheScriptRead(t *testing.T) {
	dir := scriptTree(t, map[string]string{"s.sh": hazardLine})
	runScriptCases(t, dir, []scriptCase{
		{`zsh +-login {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`zsh -f +-norcs {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
	})
}

// Each shell reads its options its own way, and the script it runs is the one
// judged: a lone `-` ends the options as `--` does; bash reads its long
// options after one dash; zsh and ksh take an -o value from the rest of its
// word, zsh's -O takes none, and only zsh reads `+-` as a long option; ksh93
// runs a word of three dashes as its script. sh is read every way it can be.
func TestEachShellReadsItsOwnOptions(t *testing.T) {
	dir := scriptTree(t, map[string]string{"s.sh": hazardLine, "rc": "echo hi", "---noexec": hazardLine})
	runScriptCases(t, dir, []scriptCase{
		{`bash - {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`sh - {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`dash - {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`zsh - {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`ksh - {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`bash -login {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`bash -noprofile {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`bash -posix {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`bash -verbose {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`sh -login {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`bash -init-file {d}/rc {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`bash -rcfile {d}/s.sh -i -c true`, VerdictBlock, scriptHazardEntryID},
		{`zsh -O {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`zsh -oerrexit {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`ksh -oerrexit {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`sh -oerrexit {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`zsh -n -oexec {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`ksh +-o errexit {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`ksh -n +-o noexec {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`ksh ---noexec`, VerdictBlock, scriptHazardEntryID},
		// The options end at the script: a -n after it is its argument.
		{`ksh ---noexec -n`, VerdictBlock, scriptHazardEntryID},
		// What still reads as a syntax check, every way the name allows.
		{`bash -login -n {d}/s.sh`, VerdictAllow, ""},
		{`zsh -onoexec {d}/s.sh`, VerdictAllow, ""},
		{`sh -n {d}/s.sh`, VerdictAllow, ""},
		{`bash -o errexit {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
	})
}

// bash reads its long options only before its first single-letter option;
// after one, a word such as `-help` is a cluster of letters, and the script
// that follows it still runs. zsh ends its options at a lone `+` as at `-`,
// and bash and dash read a lone `+` as an empty cluster, so the word after
// it is the script in each. dash has no long options, so `--noexec` grants
// it no syntax check.
func TestEachShellEndsItsOptionsWhereItDoes(t *testing.T) {
	dir := scriptTree(t, map[string]string{"s.sh": hazardLine})
	runScriptCases(t, dir, []scriptCase{
		{`bash -x -help {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`bash -o errexit -help {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`zsh + {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`bash + {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`dash + {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`sh + {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		{`dash --noexec {d}/s.sh`, VerdictBlock, scriptHazardEntryID},
		// At the front a long option is still one option.
		{`bash -login -help`, VerdictAllow, ""},
		{`bash -login -n {d}/s.sh`, VerdictAllow, ""},
	})
}

// The site bound holds before any lookup: a line naming thousands of bare
// names makes at most a bounded number of filesystem lookups.
func TestScriptSiteBoundHoldsBeforeLookups(t *testing.T) {
	dir := scriptTree(t, map[string]string{"ok.sh": "echo hi"})
	r := readingRegistry(dir, "")
	var line []string
	for i := 0; i < 2000; i++ {
		line = append(line, ". nf"+itoa(i))
	}
	d, err := r.Check(strings.Join(line, "; "))
	if err != nil {
		t.Fatal(err)
	}
	if d.Verdict != VerdictWarn || d.EntryID != scriptUnreadEntryID {
		t.Fatalf("2000 sources: %s/%s, want the budget's warn", d.Verdict, d.EntryID)
	}
	bound := maxScriptTargets * (len(filepath.SplitList(r.files.path)) + 2)
	if r.files.statCalls > bound {
		t.Fatalf("made %d lookups, bound %d", r.files.statCalls, bound)
	}
}
