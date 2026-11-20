package loop

import (
	"bytes"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/statusblock"
)

// A build reserves only the step it opens next (iss-2611201418535169): a
// start, or a resume with no lane open on a step still to build, reserves the
// packages of the first step still to build, and each later step's packages
// are reserved as its lane opens, judged for overlap like any widening, so a
// peer's hold on a package only a later step touches never refuses the start
// of step 1. Opening that later step backs off while the hold stands (exit 3),
// with the run kept as it was.

// twoStepSpec is spc-1 with two steps still to build, each in its own
// package, and a Footprint line naming both.
var twoStepSpec = specWithSteps(
	"1. The needs\n   - packages: internal/core/needs\n"+
		"2. The surface\n   - packages: internal/surface/cli\n") +
	"\n## Footprint\n\n- packages: internal/core/needs, internal/surface/cli\n- tests: the needs\n"

// TestAStartReservesOnlyTheStepItOpensNext is the issue's case: Q holds a
// file under internal/surface/cli, which only step 2 names, and the start of
// step 1 is reserved with step 1's package alone rather than refused.
func TestAStartReservesOnlyTheStepItOpensNext(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), twoStepSpec)
	bare := onOrigin(t, repo)
	quinnHolds(t, bare, heldIssue, claim.Options{}, "internal/surface/cli/root.go")

	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatalf("the start of step 1 is refused over a package only step 2 names: %v", err)
	}
	if fp, want := footprintOf(t, bare, "itd-10"), []string{"internal/core/needs"}; !slices.Equal(fp, want) {
		t.Fatalf("the reservation's footprint = %q, want step 1's package %q", fp, want)
	}
	st, err := ReadState(repo.Root(), res.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Lanes) != 1 || st.Lanes[0].SpecStep != 1 {
		t.Fatalf("the start opens a lane for step 1: %+v", st.Lanes)
	}
}

// laneDone drives the current lane of runID through the fake stages to done.
func laneDone(t *testing.T, root, runID string, steps Stages) {
	t.Helper()
	for _, want := range []Stage{StageWorktree, StageBrief} {
		if res, err := advance(root, runID, steps, Options{}); err != nil || res.PerformedStage != want {
			t.Fatalf("performed %q, want %q: %v", res.PerformedStage, want, err)
		}
	}
	res, err := advance(root, runID, steps, Options{})
	if err != nil || res.Awaiting == nil {
		t.Fatalf("the implement stage awaits an implementer: %+v %v", res, err)
	}
	if err := os.WriteFile(res.Awaiting.Receipt, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Receipt(root, runID, res.Awaiting.Receipt, steps, Options{}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []Stage{StageValidate, StageLand} {
		if res, err := advance(root, runID, steps, Options{}); err != nil || res.PerformedStage != want {
			t.Fatalf("performed %q, want %q: %v", res.PerformedStage, want, err)
		}
	}
}

// TestOpeningALaterStepWhoseFootprintAPeerHoldsIsRefusedAndTheRunKept: once
// step 1 lands, the lane for step 2 opens, and its worktree stage widens the
// reservation to step 2's package first. Q's hold on it backs that off as
// contention, with nothing pushed and the run's state as it was; once Q's hold
// ends, the same step widens the reservation and makes the worktree.
func TestOpeningALaterStepWhoseFootprintAPeerHoldsIsRefusedAndTheRunKept(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), twoStepSpec)
	bare := onOrigin(t, repo)
	q := quinnHolds(t, bare, heldIssue, claim.Options{}, "internal/surface/cli/root.go")
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSteps{calls: map[Stage]int{}}
	steps := f.steps()
	laneDone(t, repo.Root(), start.RunID, steps)

	before := stateBytes(t, repo.Root(), start.RunID)
	_, err = advance(repo.Root(), start.RunID, steps, Options{})
	r := mustRefusal(t, err)
	if !r.Contention || r.Stage != string(StageWorktree) || r.Lane != "lane-2" {
		t.Fatalf("opening step 2 backs off as contention at lane-2's worktree: %+v", r)
	}
	for _, want := range []string{"internal/surface/cli", heldIssue} {
		if !strings.Contains(r.Reason, want) {
			t.Fatalf("the refusal names %q: %q", want, r.Reason)
		}
	}
	if !bytes.Equal(before, stateBytes(t, repo.Root(), start.RunID)) {
		t.Fatal("a refused opening of step 2 must leave the run as it was")
	}
	if f.calls[StageWorktree] != 1 {
		t.Fatalf("no worktree is made for step 2 while the hold stands: %d worktree calls", f.calls[StageWorktree])
	}
	if fp, want := footprintOf(t, bare, "itd-10"), []string{"internal/core/needs"}; !slices.Equal(fp, want) {
		t.Fatalf("nothing is pushed: the footprint = %q, want %q", fp, want)
	}

	if _, err := claim.Reserve(q, claim.Request{Record: heldIssue, Release: true}, as(t, q, claim.Options{})); err != nil {
		t.Fatalf("Q releasing %s: %v", heldIssue, err)
	}
	res, err := advance(repo.Root(), start.RunID, steps, Options{})
	if err != nil || res.PerformedStage != StageWorktree {
		t.Fatalf("with the hold ended, step 2's worktree is made: %+v %v", res, err)
	}
	if fp, want := footprintOf(t, bare, "itd-10"), []string{"internal/core/needs", "internal/surface/cli"}; !slices.Equal(fp, want) {
		t.Fatalf("the reservation widens to step 2's package as its lane opens: %q, want %q", fp, want)
	}
	if note := recordNote(t, repo, start.RunID, StageWidenedTo); note != "internal/surface/cli" {
		t.Fatalf("the run record keeps the path widened to: %q", note)
	}
}

// stepPickSpec is spc-10 with two steps, each in its own package.
const stepPickSpec = "---\nid: spc-10\nslug: i10\nintent: itd-10\n---\n# i10\n\n## Summary\n\nA written design record.\n\n" +
	"## Steps\n\n1. The needs\n   - packages: internal/core/needs\n2. The surface\n   - packages: internal/surface/cli\n\n" +
	"## Footprint\n\n- packages: internal/core/needs, internal/surface/cli\n- tests: the needs\n"

// TestBuildNextAndTheBoardJudgeTheStepTheStartReserves: Q's hold on step 2's
// package alone passes nothing over, as the start reserves step 1 alone, so
// build next takes itd-10 and the board names it next up; a hold on step 1's
// package is the start's overlap, and both pass it over.
func TestBuildNextAndTheBoardJudgeTheStepTheStartReserves(t *testing.T) {
	for name, held := range map[string]string{
		"step 2 held": "internal/surface/cli/root.go",
		"step 1 held": "internal/core/needs/needs.go",
	} {
		t.Run(name, func(t *testing.T) {
			repo := pickRepo(t, map[string][2]string{
				"10": {pickIntent("10", "", settledQuestions, gwt), stepPickSpec},
				"11": {pickIntent("11", "", settledQuestions, "- it works\n"), pickSpec("11", "- packages: internal/other\n- tests: the other\n")},
			})
			bare := onOrigin(t, repo)
			quinnHolds(t, bare, heldIssue, claim.Options{}, held)
			fetchedBy(t, repo.Root())

			want := "itd-10"
			if name == "step 1 held" {
				want = "itd-11"
			}
			set, err := candidates(repo.Root(), "")
			if err != nil {
				t.Fatal(err)
			}
			e, excluded := excludedAt(set, "itd-10")
			if excluded != (want == "itd-11") || (excluded && e.Check != CheckOverlap) {
				t.Fatalf("build next excludes itd-10 only over step 1's package: %+v", set.Excluded)
			}
			b, err := statusblock.Read(repo.Root(), StatusLanes, StatusPeers)
			if err != nil {
				t.Fatal(err)
			}
			var head string
			for _, r := range b.Now {
				if r.NextUp {
					head = r.ID
				}
			}
			if head != want {
				t.Fatalf("the board's head is %s, the intent build next starts: %q in %+v", want, head, b.Now)
			}
			res, err := Next(repo.Root(), Options{}, NextOptions{})
			if err != nil || res.Pick == nil || res.Pick.Chosen.ID != want {
				t.Fatalf("build next starts %s: %+v %v", want, res.Pick, err)
			}
		})
	}
}
