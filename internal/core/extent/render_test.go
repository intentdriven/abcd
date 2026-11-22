package extent

import (
	"strings"
	"testing"
)

// fixtures holds one statement for each verdict and one with no verdict
// member, each with a left-out kind, an exclusion and a listed kind.
func fixtures() map[string]Statement {
	known := []NotLooked{
		leftOut("reservations", CauseNotArmed),
		{ID: "embargo", Label: "embargo store", Source: SourceExclusion},
		{ID: "scratch", Label: "scratch folders", Source: SourceListed},
	}
	return map[string]Statement{
		"clear": {
			LookedAt:    []Looked{looked("worktrees", 0), looked("local_branches", 0)},
			NotLookedAt: known,
		},
		"clear_except_gaps": {
			LookedAt:      []Looked{looked("worktrees", 0), looked("local_branches", 0)},
			NotLookedAt:   known,
			CouldNotCheck: []Gap{gap("process check", "worktrees", ReasonToolMissing)},
		},
		"found": {
			LookedAt:    []Looked{looked("worktrees", 1), looked("local_branches", 0)},
			NotLookedAt: known,
		},
		"none": {
			NotLookedAt: append([]NotLooked{leftOut("worktrees", CauseNothingToRead)}, known...),
		},
		"no verdict member": {
			NoClearState: true,
			LookedAt:     []Looked{looked("worktrees", 1)},
			NotLookedAt:  known,
		},
	}
}

func render(s Statement) string { return strings.Join(s.Lines(RenderOptions{}), "\n") }

func lineWith(lines []string, prefix string) (string, bool) {
	for _, l := range lines {
		if strings.HasPrefix(l, prefix) {
			return l, true
		}
	}
	return "", false
}

// TestLookedAtLineOnEveryVerdict: every result names what was looked at, and
// one that looked at no kind says nothing.
func TestLookedAtLineOnEveryVerdict(t *testing.T) {
	want := map[string]string{
		"clear":             "Looked at: worktrees, local branches",
		"clear_except_gaps": "Looked at: worktrees, local branches",
		"found":             "Looked at: worktrees, local branches",
		"none":              "Looked at: nothing",
		"no verdict member": "Looked at: worktrees",
	}
	for name, s := range fixtures() {
		got, ok := lineWith(s.Lines(RenderOptions{}), "Looked at:")
		if !ok || got != want[name] {
			t.Errorf("%s: Looked at line = %q, want %q\n%s", name, got, want[name], render(s))
		}
	}
}

// TestLeftOutLineCarriesEachCause: what this run left out of the registry is
// shown on every result that has such an entry, each with its cause in words.
func TestLeftOutLineCarriesEachCause(t *testing.T) {
	for name, s := range fixtures() {
		got, ok := lineWith(s.Lines(RenderOptions{}), "Left out in this run:")
		if !ok || !strings.Contains(got, "reservations (not armed)") {
			t.Errorf("%s: Left out line = %q\n%s", name, got, render(s))
		}
	}
	s := Statement{LookedAt: []Looked{looked("rows", 0)}}
	s.LeaveOut("receipts", "receipts", CauseHostRun)
	s.LeaveOut("deep_smoke", "deep smoke", CauseOtherRun)
	s.LeaveOut("citations", "citations", CauseNotArmed)
	s.LeaveOut("parity", "payload parity", CauseNothingToRead)
	got, _ := lineWith(s.Lines(RenderOptions{}), "Left out in this run:")
	want := "Left out in this run: receipts (run by the host), deep smoke (left to another way of running it), citations (not armed), payload parity (nothing to read)"
	if got != want {
		t.Errorf("Left out line = %q\nwant            %q", got, want)
	}
	if _, ok := lineWith(Statement{LookedAt: []Looked{looked("rows", 0)}}.Lines(RenderOptions{}), "Left out"); ok {
		t.Error("a result with nothing left out shows a Left out line")
	}
}

// TestKnownLineOnlyOnTheTwoClearVerdicts: what the command is known not to
// look at is shown, under its label, on clear and clear_except_gaps, and on
// neither found, none nor a result with no verdict member.
func TestKnownLineOnlyOnTheTwoClearVerdicts(t *testing.T) {
	for name, s := range fixtures() {
		got, ok := lineWith(s.Lines(RenderOptions{}), "Known not looked at:")
		switch name {
		case "clear", "clear_except_gaps":
			if got != "Known not looked at: embargo store, scratch folders" {
				t.Errorf("%s: Known line = %q, want both known entries under the label\n%s", name, got, render(s))
			}
		default:
			if ok {
				t.Errorf("%s: shows %q, want no Known line\n%s", name, got, render(s))
			}
		}
	}
}

// TestGapsAreOneCountPerReason: gaps show as one count of checks for each
// reason, naming the flag that lists them, and no line appears without a gap.
func TestGapsAreOneCountPerReason(t *testing.T) {
	s := Statement{
		LookedAt: []Looked{looked("worktrees", 0), looked("branches", 0)},
		CouldNotCheck: []Gap{
			gap("forge", "branches", ReasonUnreachable),
			gap("store", "", ReasonUnreadable),
			gap("runs", "", ReasonUnreadable),
		},
	}
	got, ok := lineWith(s.Lines(RenderOptions{}), "Could not check:")
	want := "Could not check: 2 checks (unreadable), 1 check (unreachable); --gaps lists them"
	if !ok || got != want {
		t.Errorf("Could not check line = %q\nwant                   %q", got, want)
	}
	if strings.Contains(render(s), "did not answer") {
		t.Errorf("the gaps are listed without being asked for:\n%s", render(s))
	}
	if out := render(fixtures()["clear"]); strings.Contains(out, "Could not check") {
		t.Errorf("a result with no gap shows a gap line:\n%s", out)
	}
}

// TestAllGapsNamesNoFlagAndListsEveryGap: where every gap is shown (no one can
// pass a flag, or the person passed it), the line names no flag and every gap
// follows it in full.
func TestAllGapsNamesNoFlagAndListsEveryGap(t *testing.T) {
	s := Statement{
		LookedAt: []Looked{looked("worktrees", 0)},
		CouldNotCheck: []Gap{
			gap("process check", "worktrees", ReasonToolMissing),
			{Check: "forge", Kind: "branches_on_origin", Reason: ReasonTimedOut, Text: "no answer in 10s"},
		},
	}
	lines := s.Lines(RenderOptions{AllGaps: true})
	i := -1
	for n, l := range lines {
		if strings.HasPrefix(l, "Could not check:") {
			i = n
		}
	}
	if i < 0 || i+2 >= len(lines) {
		t.Fatalf("no Could not check line followed by two gaps:\n%s", strings.Join(lines, "\n"))
	}
	if strings.Contains(lines[i], "--gaps") {
		t.Errorf("line %q names a flag", lines[i])
	}
	if lines[i+1] != "  - process check (worktrees): tool missing: process check did not answer" {
		t.Errorf("first gap = %q", lines[i+1])
	}
	if lines[i+2] != "  - forge (branches_on_origin): timed out: no answer in 10s" {
		t.Errorf("second gap = %q", lines[i+2])
	}
}

// TestJudgedByLinePerDepthEntry: one Judged by line for each depth entry, with
// not told and none asked said in words, and an entry with no run saying so.
func TestJudgedByLinePerDepthEntry(t *testing.T) {
	s := Statement{
		NoClearState: true,
		LookedAt:     []Looked{looked("open_issues", 2)},
		Depth: []Depth{
			{Role: "validator", Lane: "itd-1",
				Class:  &Told{Values: []string{"strong", "standard"}, From: OriginRunState},
				Runs:   Runs{Count: 3, From: OriginRunState},
				Effort: &Told{Values: []string{"high"}, From: OriginRunState}},
			{Role: "validator", Lane: "itd-2", Runs: Runs{Count: 0, From: OriginRunState}},
			{Role: "drain judge", Issue: "iss-1",
				Class:  &Told{Mark: MarkNoneAsked},
				Runs:   Runs{Count: 1, From: OriginRunState},
				Effort: &Told{Mark: MarkNoneAsked}},
			{Role: "receipts review", Class: &Told{Mark: MarkNotTold}, Runs: Runs{Mark: MarkNotTold}, Effort: &Told{Mark: MarkNotTold}},
			{Role: "host review", Class: &Told{Values: []string{"frontier"}, From: OriginReceipt}, Runs: Runs{Mark: MarkNotTold}, Effort: &Told{Mark: MarkNotTold}},
		},
	}
	var got []string
	for _, l := range s.Lines(RenderOptions{}) {
		if strings.HasPrefix(l, "Judged by:") {
			got = append(got, l)
		}
	}
	want := []string{
		"Judged by: validator, lane itd-1: classes strong, standard, 3 judging runs, effort high",
		"Judged by: validator, lane itd-2: no judging run",
		"Judged by: drain judge, issue iss-1: no class asked, 1 judging run, no effort asked",
		"Judged by: receipts review: abcd was not told the class, the number of runs or the effort",
		"Judged by: host review: class frontier (from a receipt), abcd was not told the number of runs, abcd was not told the effort",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("Judged by lines:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// TestSummaryLineForClearExceptGaps: the summary reads as clear except for the
// gaps, and the two other verdicts with something looked at carry none.
func TestSummaryLineForClearExceptGaps(t *testing.T) {
	f := fixtures()
	lines := f["clear_except_gaps"].Lines(RenderOptions{})
	if last := lines[len(lines)-1]; last != "Clear except for 1 check that could not be made." {
		t.Errorf("summary = %q\n%s", last, strings.Join(lines, "\n"))
	}
	for _, name := range []string{"clear", "found", "no verdict member"} {
		out := render(f[name])
		if strings.Contains(out, "Clear except") || strings.Contains(out, "Nothing was checked") {
			t.Errorf("%s carries a summary line:\n%s", name, out)
		}
	}
}

// TestNothingCheckedSaysWhy: a result that looked at no kind says nothing was
// checked, and why: the causes of what was left out, the reasons nothing could
// be read, or that there was no kind to look at. It holds for the verdict none
// and for a statement with no verdict member.
func TestNothingCheckedSaysWhy(t *testing.T) {
	cases := map[string]struct {
		s    Statement
		want string
	}{
		"none, left out": {fixtures()["none"],
			"Nothing was checked: 2 kinds left out (not armed, nothing to read)."},
		"none, unreadable": {Statement{CouldNotCheck: []Gap{gap("store", "reservations", ReasonUnreadable), gap("forge", "branches", ReasonUnreachable)}},
			"Nothing was checked: 2 checks could not be made (unreadable, unreachable)."},
		"none, both": {Statement{NotLookedAt: []NotLooked{leftOut("rule_a", CauseNotArmed)}, CouldNotCheck: []Gap{gap("store", "rule_b", ReasonUnreadable)}},
			"Nothing was checked: 1 kind left out (not armed); 1 check could not be made (unreadable)."},
		"empty registry": {Statement{NotLookedAt: []NotLooked{{ID: "authors", Label: "authors unless opted in", Source: SourceExclusion}}},
			"Nothing was checked: there was no kind to look at."},
		"no verdict member": {Statement{NoClearState: true, NotLookedAt: []NotLooked{leftOut("lane_changes", CauseNothingToRead)}},
			"Nothing was checked: 1 kind left out (nothing to read)."},
	}
	for name, tc := range cases {
		lines := tc.s.Lines(RenderOptions{})
		if lines[0] != "Looked at: nothing" {
			t.Errorf("%s: first line = %q, want Looked at: nothing", name, lines[0])
		}
		if last := lines[len(lines)-1]; last != tc.want {
			t.Errorf("%s: summary = %q\nwant      %q\n%s", name, last, tc.want, strings.Join(lines, "\n"))
		}
	}
}

// TestRenderMasksControlRunes: a label or a gap's text comes from the
// repository and is shown on one line, so a control rune cannot forge a line.
func TestRenderMasksControlRunes(t *testing.T) {
	s := Statement{
		LookedAt:      []Looked{{ID: "k", Label: "a\nLooked at: forged", Count: 0}},
		CouldNotCheck: []Gap{{Check: "c", Reason: ReasonCheckFailed, Text: "bad\x1b[2Jtext"}},
	}
	for _, l := range s.Lines(RenderOptions{AllGaps: true}) {
		if strings.ContainsAny(l, "\n\x1b") {
			t.Errorf("line %q carries a control rune", l)
		}
	}
}
