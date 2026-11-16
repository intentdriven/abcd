package loop

import (
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/runner"
)

// runnerWaitLines counts the record's lines that began a wait at the arming
// for the runner's report on receipt.
func runnerWaitLines(t *testing.T, f *landFixture, receipt string) int {
	t.Helper()
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range st.Record {
		if e.Stage == string(StageLand) && strings.HasPrefix(e.Note, "waiting for the runner's report on "+receipt) {
			n++
		}
	}
	return n
}

// TestALaneIsNotArmedUntilItsRunnersReportComes: a lane whose implementer's
// receipt a step collected before the runner's report came waits at its
// arming, the record saying once what it waits for, and nothing is asked of
// the forge's merge; once the report comes, the next step arms it.
func TestALaneIsNotArmedUntilItsRunnersReportComes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.toArmOnce(t)
	path := collectedImplementer(t, f.repo.Root(), f.runID)
	rel := currentLane(t, f.repo, f.runID).Receipts[0].Receipt
	for range 2 {
		_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
		r, ok := AsRefusal(err)
		if !ok || !r.Contention || !strings.Contains(r.Reason, rel) || !strings.Contains(r.Reason, "not armed") {
			t.Fatalf("the step waits for the runner's report, naming the receipt: %v", err)
		}
	}
	if l := currentLane(t, f.repo, f.runID); l.Landing.Armed || l.Landing.Merge != "" || l.Stage != StageLand {
		t.Fatalf("the lane is not armed while the report has not come: %+v", l.Landing)
	}
	if strings.Contains(f.ghLog(t), "pr merge") {
		t.Fatalf("nothing is armed:\n%s", f.ghLog(t))
	}
	if n := runnerWaitLines(t, f, rel); n != 1 {
		t.Fatalf("the record says once what the lane waits for: %d lines", n)
	}
	if st, err := ReadState(f.repo.Root(), f.runID); err != nil || st.Lanes[0].Refused != nil {
		t.Fatalf("the wait is no refusal: %+v %v", st.Lanes[0].Refused, err)
	}
	report := &runner.RouteRecord{Asked: runner.Claude, Ran: runner.Claude, Model: f.model}
	if _, err := receipt(f.repo.Root(), f.runID, path, f.stages, Options{}, report); err != nil {
		t.Fatalf("the runner's report is answered: %v", err)
	}
	f.step(t)
	if l := currentLane(t, f.repo, f.runID); !l.Landing.Armed || strings.Count(f.ghLog(t), "pr merge 7 --auto") != 1 {
		t.Fatalf("once the report has come the lane is armed: %+v\n%s", l.Landing, f.ghLog(t))
	}
}

// TestALaneWhoseRunnerNeverReportsIsHandedBackAsStuck: past the stuck bound's
// minutes since the wait began, the lane is handed back as stuck, naming the
// receipt, and still nothing is armed.
func TestALaneWhoseRunnerNeverReportsIsHandedBackAsStuck(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.toArmOnce(t)
	collectedImplementer(t, f.repo.Root(), f.runID)
	rel := currentLane(t, f.repo, f.runID).Receipts[0].Receipt
	began := time.Now()
	if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{Now: func() time.Time { return began }}); err == nil {
		t.Fatal("premise: the step waits for the runner's report")
	}
	within := began.Add(time.Duration(BundledStuckMinutes-1) * time.Minute)
	if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{Now: func() time.Time { return within }}); err == nil {
		t.Fatal("within the bound the lane still waits")
	}
	past := began.Add(time.Duration(BundledStuckMinutes+1) * time.Minute)
	res, err := advance(f.repo.Root(), f.runID, f.stages, Options{Now: func() time.Time { return past }})
	if err != nil {
		t.Fatalf("past the bound the step hands the lane back: %v", err)
	}
	if hb := res.HandBack; hb == nil || hb.Kind != HandBackStuck || !strings.Contains(hb.Reason, rel) || !strings.Contains(hb.Reason, "past the bound") {
		t.Fatalf("the lane is handed back as stuck, naming the receipt: %+v", res)
	}
	if l := currentLane(t, f.repo, f.runID); l.Stage != StageHandedBack || l.Landing.Armed {
		t.Fatalf("the handed-back lane is not armed: %+v", l)
	}
	if strings.Contains(f.ghLog(t), "pr merge") {
		t.Fatalf("nothing is armed:\n%s", f.ghLog(t))
	}
}

// TestALateReportHandingBackAnArmedLaneWithdrawsTheArming: a run whose lane
// was armed before the arming waited for its runners' reports, met by a late
// report naming a reviewer's own model, is handed back and its arming
// withdrawn through the forge, the record saying so.
func TestALateReportHandingBackAnArmedLaneWithdrawsTheArming(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.landedToArmedOnce(t)
	if !l.Landing.Armed {
		t.Fatalf("premise: the lane is armed: %+v", l.Landing)
	}
	reviewer := ""
	for _, v := range l.Validation[len(l.Validation)-1].Validators {
		if v.Class != "" && v.ReviewerModel != "" {
			reviewer = v.ReviewerModel
		}
	}
	if reviewer == "" {
		t.Fatalf("premise: a reviewer ran on a class: %+v", l.Validation)
	}
	path := collectedImplementer(t, f.repo.Root(), f.runID)
	report := &runner.RouteRecord{Asked: runner.Claude, Ran: runner.Claude, Model: reviewer}
	if _, err := receipt(f.repo.Root(), f.runID, path, f.stages, Options{}, report); err != nil {
		t.Fatalf("the runner's report is answered: %v", err)
	}
	l = currentLane(t, f.repo, f.runID)
	if l.Stage != StageHandedBack || l.HandBack == nil || l.HandBack.Kind != HandBackStuck {
		t.Fatalf("premise: the late report hands the lane back: %+v", l)
	}
	if l.Landing.Armed || l.Landing.Merge != "" {
		t.Fatalf("the hand-back withdraws the arming: %+v", l.Landing)
	}
	if !strings.Contains(f.ghLog(t), "pr merge 7 --disable-auto") {
		t.Fatalf("the forge is asked to withdraw the arming:\n%s", f.ghLog(t))
	}
	if !recordSays(t, f.repo.Root(), f.runID, string(StageHandedBack), "withdrew pull request #7's arming") {
		t.Fatal("the record says the arming was withdrawn")
	}
}
