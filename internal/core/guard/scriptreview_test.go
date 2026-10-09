package guard

import (
	"path/filepath"
	"strings"
	"testing"
)

// Review findings on the script reading (2026-10-09).
func TestScriptReadingReviewFindings(t *testing.T) {
	dir := scriptTree(t, map[string]string{
		"e":    hazardLine,
		"s.sh": hazardLine,
		"a.sh": "bash b.sh",
		"b.sh": "bash c.sh",
		"c.sh": "echo deep",
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
