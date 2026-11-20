package loop

import (
	"bytes"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

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
// Each path of marks is widened to at the lane's validators, as a real
// landing's fence widens the hold to the spec record its step mark changes.
func laneDone(t *testing.T, root, runID string, steps Stages, marks ...string) {
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
	lane := res.Lane
	if err := os.WriteFile(res.Awaiting.Receipt, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Receipt(root, runID, res.Awaiting.Receipt, steps, Options{}); err != nil {
		t.Fatal(err)
	}
	if len(marks) > 0 {
		if _, err := Widen(root, runID, lane, marks, Options{}); err != nil {
			t.Fatalf("widening to %q: %v", marks, err)
		}
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

// crossedSpec is spc-<n> of itd-<n>, with step 1 on the package first and
// step 2 on second.
func crossedSpec(n, first, second string) string {
	return "---\nid: spc-" + n + "\nslug: i" + n + "\nintent: itd-" + n + "\n---\n# i" + n + "\n\n## Summary\n\nA written design record.\n\n" +
		"## Steps\n\n1. The first\n   - packages: " + first + "\n2. The second\n   - packages: " + second + "\n\n" +
		"## Footprint\n\n- packages: " + first + ", " + second + "\n- tests: the steps\n"
}

// TestTwoRunsWhoseStepOrdersCrossDoNotWaitOnEachOther is the deadlock a
// reservation taken one step at a time risks (dcn-2611012049050703): P's run
// of itd-10 builds internal/core/needs and then internal/surface/cli, and Q's
// run of itd-11, from another copy, the same two the other way round. Both
// starts are reserved, since each reserves its first step alone, and both
// first steps land. Once a step has landed, its run's next renewal narrows the
// reservation to the steps still in flight (and the spec record each landing
// marks), so P's opening of step 2 waits on Q's hold of internal/surface/cli
// holding internal/core/needs no longer; Q's step 2 then opens, and dropping
// its landed step's package lets P's step 2 open in turn.
func TestTwoRunsWhoseStepOrdersCrossDoNotWaitOnEachOther(t *testing.T) {
	const needs, cli = "internal/core/needs", "internal/surface/cli"
	repo := pickRepo(t, map[string][2]string{
		"10": {pickIntent("10", "", settledQuestions, gwt), crossedSpec("10", needs, cli)},
		"11": {pickIntent("11", "", settledQuestions, gwt), crossedSpec("11", cli, needs)},
	})
	bare := onOrigin(t, repo)
	q := otherCopy(t, bare, quinnEmail)
	_, spec10 := pickRel("10")
	_, spec11 := pickRel("11")
	f := &fakeSteps{calls: map[Stage]int{}}
	steps := f.steps()

	x, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatalf("P's start: %v", err)
	}
	var y StartResult
	asHomeOf(t, q, func() { y, err = Start(q, "itd-11", Options{}) })
	if err != nil {
		t.Fatalf("Q's start, on step 1's package alone, overlaps nothing of P's: %v", err)
	}
	laneDone(t, repo.Root(), x.RunID, steps, spec10)
	asHomeOf(t, q, func() { laneDone(t, q, y.RunID, steps, spec11) })

	_, err = advance(repo.Root(), x.RunID, steps, Options{})
	if r := mustRefusal(t, err); !r.Contention || r.Stage != string(StageWorktree) || !strings.Contains(r.Reason, cli) {
		t.Fatalf("P's step 2 waits on Q's hold of %s: %+v", cli, r)
	}
	if fp, want := footprintOf(t, bare, "itd-10"), []string{spec10}; !slices.Equal(fp, want) {
		t.Fatalf("P's run, with step 1 landed and step 2 waiting, holds %q, want %q alone", fp, want)
	}
	var res StepResult
	asHomeOf(t, q, func() { res, err = advance(q, y.RunID, steps, Options{}) })
	if err != nil || res.PerformedStage != StageWorktree {
		t.Fatalf("Q's step 2 opens on %s, which P's landed step no longer holds: %+v %v", needs, res, err)
	}
	if fp, want := footprintOf(t, bare, "itd-11"), []string{spec11, needs}; !slices.Equal(fp, want) {
		t.Fatalf("Q's run holds %q, want %q", fp, want)
	}
	res, err = advance(repo.Root(), x.RunID, steps, Options{})
	if err != nil || res.PerformedStage != StageWorktree {
		t.Fatalf("P's step 2 opens once Q's landed step drops %s: %+v %v", cli, res, err)
	}
	if fp, want := footprintOf(t, bare, "itd-10"), []string{spec10, cli}; !slices.Equal(fp, want) {
		t.Fatalf("P's run holds %q, want %q", fp, want)
	}
}

// threeStepSpec is spc-1 with step 1 on internal/core/needs, step 2 on
// internal/surface/cli, needing step 1 unless needs2 says otherwise, and step
// 3 on internal/other, needing none.
func threeStepSpec(needs2 string) string {
	return specWithSteps(
		"1. The needs\n   - packages: internal/core/needs\n"+
			"2. The surface\n   - packages: internal/surface/cli\n"+needs2+
			"3. The other\n   - packages: internal/other\n   - needs: none\n") +
		"\n## Footprint\n\n- packages: internal/core/needs, internal/surface/cli, internal/other\n- tests: the needs\n"
}

// TestARefusedOpeningLetsALaterStepsLaneOpen: with a slot free beside lane-1's
// implementer, the step opens a lane for step 2, whose package Q holds. That
// opening waits, and the step opens the lane for step 3 in its place rather
// than ending the call on the wait: one step's wait holds back that step
// alone.
func TestARefusedOpeningLetsALaterStepsLaneOpen(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), threeStepSpec("   - needs: none\n"))
	bare := onOrigin(t, repo)
	quinnHolds(t, bare, heldIssue, claim.Options{}, "internal/surface/cli/root.go")
	start, err := Start(repo.Root(), "itd-10", Options{SubAgents: strp("3")})
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSteps{calls: map[Stage]int{}}
	steps := f.steps()
	for _, want := range []Stage{StageWorktree, StageBrief} {
		if res, err := advance(repo.Root(), start.RunID, steps, Options{}); err != nil || res.PerformedStage != want {
			t.Fatalf("performed %q, want %q: %v", res.PerformedStage, want, err)
		}
	}
	if res, err := advance(repo.Root(), start.RunID, steps, Options{}); err != nil || res.Awaiting == nil {
		t.Fatalf("lane-1's implementer is handed out: %+v %v", res, err)
	}

	res, err := advance(repo.Root(), start.RunID, steps, Options{})
	if err != nil || res.PerformedStage != StageWorktree {
		t.Fatalf("step 2's opening waits, and the lane for step 3 opens: %+v %v", res, err)
	}
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Lanes) != 2 || st.Lanes[1].SpecStep != 3 {
		t.Fatalf("the lane opened is step 3's, and none is open for step 2: %+v", st.Lanes)
	}
	if len(res.Blocked) != 1 || res.Blocked[0].Lane != stepKey(2) || !strings.Contains(res.Next, "internal/surface/cli") {
		t.Fatalf("the answer names step 2's wait on Q's hold: %+v, next %q", res.Blocked, res.Next)
	}
	if fp, want := footprintOf(t, bare, "itd-10"), []string{"internal/core/needs", "internal/other"}; !slices.Equal(fp, want) {
		t.Fatalf("the reservation widens to step 3's package alone: %q, want %q", fp, want)
	}
}

// TestARefusedOpenLaneLetsTheOtherLanesMove: lane-2, for step 3, is at work
// when lane-1 lands and the lane for step 2 opens, saved at its worktree
// stage. Q holds step 2's package, so its worktree waits; the step goes on to
// lane-2's validators rather than ending the call on the wait.
func TestARefusedOpenLaneLetsTheOtherLanesMove(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), threeStepSpec(""))
	bare := onOrigin(t, repo)
	quinnHolds(t, bare, heldIssue, claim.Options{}, "internal/surface/cli/root.go")
	start, err := Start(repo.Root(), "itd-10", Options{SubAgents: strp("2")})
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSteps{calls: map[Stage]int{}}
	steps := f.steps()
	step := func(want Stage) StepResult {
		t.Helper()
		res, err := advance(repo.Root(), start.RunID, steps, Options{})
		if err != nil || (want != "" && res.PerformedStage != want) || (want == "" && res.Awaiting == nil) {
			t.Fatalf("performed %q, want %q (\"\" for an await): %+v %v", res.PerformedStage, want, res, err)
		}
		return res
	}
	hand := func(res StepResult) {
		t.Helper()
		if err := os.WriteFile(res.Awaiting.Receipt, []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Receipt(repo.Root(), start.RunID, res.Awaiting.Receipt, steps, Options{}); err != nil {
			t.Fatal(err)
		}
	}
	step(StageWorktree)
	step(StageBrief)
	one := step("")
	step(StageWorktree)
	step(StageBrief)
	two := step("")
	hand(one)
	step(StageValidate)
	step(StageLand)
	hand(two)

	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Lanes) != 3 || st.Lanes[1].SpecStep != 3 || st.Lanes[2].SpecStep != 2 || st.Lanes[2].Stage != StageWorktree {
		t.Fatalf("lane-3 is open for step 2 at its worktree, beside lane-2 for step 3: %+v", st.Lanes)
	}
	res := step(StageValidate)
	if res.Lane != "lane-2" {
		t.Fatalf("lane-2's validators run while lane-3 waits: %+v", res)
	}
	if len(res.Blocked) != 1 || res.Blocked[0].Lane != "lane-3" || !strings.Contains(res.Next, "internal/surface/cli") {
		t.Fatalf("the answer names lane-3's wait on Q's hold: %+v, next %q", res.Blocked, res.Next)
	}
}

// TestAResumeReservesNoPathOnlyALandedStepWidenedTo: lane-1 widened its hold
// to internal/extra beside step 1's package and the spec record, and landed;
// the next step narrowed the hold to what step 2's lane needs and opened it.
// Q then reserves a file under internal/extra, which only the landed step
// needed. A resume reserves what the lanes in flight need (landedNarrowing's
// rule), so Q's hold refuses nothing, and the hold stays without the path.
func TestAResumeReservesNoPathOnlyALandedStepWidenedTo(t *testing.T) {
	const extra = "internal/extra"
	repo := loopRepo(t, readyIntent("", settledQuestions), twoStepSpec)
	bare := onOrigin(t, repo)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSteps{calls: map[Stage]int{}}
	steps := f.steps()
	laneDone(t, repo.Root(), start.RunID, steps, specRel, extra)
	if res, err := advance(repo.Root(), start.RunID, steps, Options{}); err != nil || res.PerformedStage != StageWorktree {
		t.Fatalf("step 2's lane opens: %+v %v", res, err)
	}
	want := []string{"internal/surface/cli", specRel}
	slices.Sort(want)
	fp := footprintOf(t, bare, "itd-10")
	slices.Sort(fp)
	if !slices.Equal(fp, want) {
		t.Fatalf("with step 1 landed and step 2 open, the hold is %q, want %q", fp, want)
	}
	quinnHolds(t, bare, heldIssue, claim.Options{}, extra+"/x.go")

	if _, err := Start(repo.Root(), "itd-10", Options{}); err != nil {
		t.Fatalf("the resume is refused over %s, which only the landed step widened to: %v", extra, err)
	}
	fp = footprintOf(t, bare, "itd-10")
	slices.Sort(fp)
	if slices.ContainsFunc(fp, func(p string) bool { return strings.HasPrefix(p, extra) }) || !slices.Equal(fp, want) {
		t.Fatalf("the resumed hold is %q, want %q", fp, want)
	}
}

// TestAResumeOverALapsedTipReservesNoPathOnlyALandedStepNeeded: lane-1
// widened its hold to internal/extra and landed, and the run stopped there,
// before the renewal that narrows the hold, which then lapsed still naming the
// path. Q reserves a file under it. The resume revives the tip with what the
// lanes in flight need (inFlightFootprint) rather than the tip as it stood, so
// Q's hold refuses nothing.
func TestAResumeOverALapsedTipReservesNoPathOnlyALandedStepNeeded(t *testing.T) {
	const extra = "internal/extra"
	repo := loopRepo(t, readyIntent("", settledQuestions), twoStepSpec)
	bare := onOrigin(t, repo)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSteps{calls: map[Stage]int{}}
	laneDone(t, repo.Root(), start.RunID, f.steps(), specRel, extra)
	if fp := footprintOf(t, bare, "itd-10"); !slices.Contains(fp, extra) {
		t.Fatalf("the landed lane's hold still names %s: %q", extra, fp)
	}
	later := time.Now().Add(6 * time.Hour)
	clock := func() time.Time { return later }
	quinnHolds(t, bare, heldIssue, claim.Options{Now: clock}, extra+"/x.go")

	if _, err := Start(repo.Root(), "itd-10", Options{Now: clock}); err != nil {
		t.Fatalf("the resume is refused over %s, which only the landed step needed: %v", extra, err)
	}
	if fp := footprintOf(t, bare, "itd-10"); slices.ContainsFunc(fp, func(p string) bool { return strings.HasPrefix(p, extra) }) {
		t.Fatalf("the revived hold names %s again: %q", extra, fp)
	}
}
