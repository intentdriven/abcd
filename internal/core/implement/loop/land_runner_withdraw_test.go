package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// The withdrawal of an arming a late runner's report leaves on a handed-back
// lane (land_runner_wait.go): made once the hand-back is written, tried again
// by each later step until the forge reports no arming, and the issue run's
// reservation released only then.

// classReviewer is the model a reviewer of l's last round ran on a class.
func classReviewer(t *testing.T, l Lane) string {
	t.Helper()
	reviewer := ""
	for _, v := range l.Validation[len(l.Validation)-1].Validators {
		if v.Class != "" && v.ReviewerModel != "" {
			reviewer = v.ReviewerModel
		}
	}
	if reviewer == "" {
		t.Fatalf("premise: a reviewer ran on a class: %+v", l.Validation)
	}
	return reviewer
}

// lateReportArmed walks a drain's issue lane to its armed pull request, the
// run holding its reservation, makes its implementer's receipt one a step
// collected before its runner reported, and returns the fixture, the
// receipt's path and a late report naming a reviewer's own model, which hands
// the lane back.
func lateReportArmed(t *testing.T) (*landFixture, string, *runner.RouteRecord) {
	t.Helper()
	f := drainLandFixture(t)
	first := f.firstMove(t)
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	f.runID, f.issue = first.Start.RunID, secondIssue
	l := f.armIssueLaneOnce(t)
	reviewer := classReviewer(t, l)
	path := collectedImplementer(t, f.repo.Root(), f.runID)
	if tomb := originReservation(t, f.bare, claim.RefPrefix+secondIssue); tomb.commit == "" || tomb.released {
		t.Fatalf("premise: the run holds its reservation: %+v", tomb)
	}
	return f, path, &runner.RouteRecord{Asked: runner.Claude, Ran: runner.Claude, Model: reviewer}
}

// handInLate hands the late report in and checks it hands the lane back.
func (f *landFixture) handInLate(t *testing.T, path string, report *runner.RouteRecord) StepResult {
	t.Helper()
	res, err := receipt(f.repo.Root(), f.runID, path, f.stages, Options{}, report)
	if err != nil {
		t.Fatalf("the runner's report is answered: %v", err)
	}
	if l := currentLane(t, f.repo, f.runID); l.Stage != StageHandedBack || l.HandBack == nil || l.HandBack.Kind != HandBackStuck {
		t.Fatalf("premise: the late report hands the lane back: %+v", l)
	}
	return res
}

// reservationReleased reports whether origin holds the issue's reservation
// released.
func (f *landFixture) reservationReleased(t *testing.T) bool {
	t.Helper()
	return originReservation(t, f.bare, claim.RefPrefix+secondIssue).released
}

// ghFile writes body into the stub forge's file name ("" for an empty file).
func (f *landFixture) ghFile(t *testing.T, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(f.gh, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// ghGone removes the stub forge's file name.
func (f *landFixture) ghGone(t *testing.T, name string) {
	t.Helper()
	if err := os.Remove(filepath.Join(f.gh, name)); err != nil {
		t.Fatal(err)
	}
}

// triedAgain: a withdrawal that does not take (fail makes it so, undo lets
// it take) leaves the handed-back lane armed, the record saying why (want)
// and that it can still merge, and the reservation held; the next step tries
// again and is refused the same way, holding it still; once it takes, the
// step clears the arming, records it and releases the reservation.
func triedAgain(t *testing.T, fail, undo func(f *landFixture), want string) {
	t.Helper()
	f, path, report := lateReportArmed(t)
	fail(f)
	res := f.handInLate(t, path, report)
	if l := currentLane(t, f.repo, f.runID); !l.Landing.Armed {
		t.Fatalf("a withdrawal that did not take leaves the lane armed: %+v", l.Landing)
	}
	if !recordSays(t, f.repo.Root(), f.runID, string(StageHandedBack), want, "until the forge reports no arming it can still merge") {
		t.Fatalf("the record says why the arming stands, and that it can still merge (%q)", want)
	}
	if !strings.Contains(res.Next, want) || !strings.Contains(res.Next, "not released while it is armed") {
		t.Fatalf("the answer says the arming stands and nothing was released: %s", res.Next)
	}
	if f.reservationReleased(t) {
		t.Fatal("the reservation is kept while the pull request is armed")
	}
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	if r := mustRefusal(t, err); !r.Contention || !strings.Contains(r.Reason, want) {
		t.Fatalf("the next step tries the withdrawal again and waits on it: %+v", r)
	}
	if l := currentLane(t, f.repo, f.runID); !l.Landing.Armed || f.reservationReleased(t) {
		t.Fatalf("a retry that did not take changes nothing: %+v", l.Landing)
	}
	undo(f)
	res, err = advance(f.repo.Root(), f.runID, f.stages, Options{})
	if err != nil {
		t.Fatalf("the step that withdraws the arming: %v", err)
	}
	if l := currentLane(t, f.repo, f.runID); l.Landing.Armed || l.Landing.Merge != "" || l.Stage != StageHandedBack {
		t.Fatalf("the withdrawal that takes clears the arming: %+v", l)
	}
	if !recordSays(t, f.repo.Root(), f.runID, string(StageHandedBack), "withdrew pull request #7's arming, since") || !strings.Contains(res.Next, "withdrew pull request #7's arming, since") {
		t.Fatalf("the record and the answer say the arming was withdrawn: %s", res.Next)
	}
	if !f.reservationReleased(t) {
		t.Fatal("the reservation is released once the forge reports no arming")
	}
}

// TestALateArmingWhoseQueueIsUnreadIsTriedAgain: the forge's answer on the
// pull request's place in the queue cannot be read.
func TestALateArmingWhoseQueueIsUnreadIsTriedAgain(t *testing.T) {
	triedAgain(t, func(f *landFixture) {
		f.ghFile(t, "queue.json", `{"errors":[{"message":"the forge is busy"}]}`)
	}, func(f *landFixture) { f.ghGone(t, "queue.json") }, "its arming could not be read")
}

// TestALateArmingTheForgeRefusesToWithdrawIsTriedAgain: the forge refuses the
// withdrawal.
func TestALateArmingTheForgeRefusesToWithdrawIsTriedAgain(t *testing.T) {
	triedAgain(t, func(f *landFixture) { f.ghFile(t, "refuse-disarm", "") },
		func(f *landFixture) { f.ghGone(t, "refuse-disarm") }, "the forge refused to withdraw the arming")
}

// TestALateArmingWhoseReadBackFailsIsTriedAgain: the withdrawal cannot be read
// back.
func TestALateArmingWhoseReadBackFailsIsTriedAgain(t *testing.T) {
	triedAgain(t, func(f *landFixture) { f.ghFile(t, "refuse-armings", "") },
		func(f *landFixture) { f.ghGone(t, "refuse-armings") }, "could not read it back")
}

// TestALateArmingStillReadBackArmedIsTriedAgain: the forge goes on reporting
// the pull request armed after the withdrawal.
func TestALateArmingStillReadBackArmedIsTriedAgain(t *testing.T) {
	triedAgain(t, func(f *landFixture) { stillArmed(t, f) },
		func(f *landFixture) { f.ghGone(t, "armings.json") }, "still reports it armed or queued")
}

// TestALateArmingAlreadyGoneIsSaidAsNone: a pull request the forge reports with
// no auto-merge and no queue entry has no arming to withdraw: nothing is asked
// of the forge, the record says there was none left, the lane's arming is
// cleared and the reservation released.
func TestALateArmingAlreadyGoneIsSaidAsNone(t *testing.T) {
	f, path, report := lateReportArmed(t)
	f.queueAnswer(t, stubQueue{head: currentLane(t, f.repo, f.runID).Landing.Pushed, status: "CLEAN"})
	f.handInLate(t, path, report)
	if strings.Contains(f.ghLog(t), "--disable-auto") || strings.Contains(f.ghLog(t), "dequeuePullRequest") {
		t.Fatalf("nothing is withdrawn where the forge reports no arming:\n%s", f.ghLog(t))
	}
	if l := currentLane(t, f.repo, f.runID); l.Landing.Armed {
		t.Fatalf("the lane's arming is cleared: %+v", l.Landing)
	}
	if !recordSays(t, f.repo.Root(), f.runID, string(StageHandedBack), "reports no arming left on it", "so there was none to withdraw") ||
		recordSays(t, f.repo.Root(), f.runID, string(StageHandedBack), "withdrew pull request #7's arming") {
		t.Fatal("the record says there was no arming left, and never that one was withdrawn")
	}
	if !f.reservationReleased(t) {
		t.Fatal("the reservation is released")
	}
}

// TestALateArmingAlreadyMergedIsSaidAsMerged: a pull request the forge reports
// merged is said as merged, not as withdrawn.
func TestALateArmingAlreadyMergedIsSaidAsMerged(t *testing.T) {
	f, path, report := lateReportArmed(t)
	f.ghFile(t, "queue.json", `{"data":{"repository":{"pullRequest":{"id":"PR_kwDOstub7","state":"MERGED","headRefOid":"","isInMergeQueue":false,"mergeQueueEntry":null,"autoMergeRequest":null,"mergeStateStatus":"UNKNOWN","timelineItems":{"nodes":[]}}}}}`)
	f.handInLate(t, path, report)
	if strings.Contains(f.ghLog(t), "--disable-auto") {
		t.Fatalf("nothing is withdrawn from a merged pull request:\n%s", f.ghLog(t))
	}
	if !recordSays(t, f.repo.Root(), f.runID, string(StageHandedBack), "had already merged", "no arming to withdraw") ||
		recordSays(t, f.repo.Root(), f.runID, string(StageHandedBack), "withdrew pull request #7's arming") {
		t.Fatal("the record says the pull request merged, not that its arming was withdrawn")
	}
	if l := currentLane(t, f.repo, f.runID); l.Landing.Armed {
		t.Fatalf("the lane's arming is cleared: %+v", l.Landing)
	}
}

// TestALateArmingWithdrawnReleasesTheReservation: the withdrawal taken and
// read back as gone clears the arming, says so, and only then releases the
// reservation; the review receipt this checkout holds for the pushed head is
// removed, so the release hold's re-arm (landing.DecideMerge) no longer counts
// the pair that judged it.
func TestALateArmingWithdrawnReleasesTheReservation(t *testing.T) {
	f, path, report := lateReportArmed(t)
	l := currentLane(t, f.repo, f.runID)
	receiptAt := filepath.Join(f.repo.Root(), filepath.FromSlash(landing.ReviewReceiptsRelDir), l.Landing.Pushed)
	if err := os.MkdirAll(filepath.Dir(receiptAt), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receiptAt, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.handInLate(t, path, report)
	if !strings.Contains(f.ghLog(t), "pr merge 7 --disable-auto") {
		t.Fatalf("the forge is asked to withdraw the arming:\n%s", f.ghLog(t))
	}
	if l := currentLane(t, f.repo, f.runID); l.Landing.Armed || l.Landing.Merge != "" {
		t.Fatalf("the withdrawal clears the arming: %+v", l.Landing)
	}
	if !recordSays(t, f.repo.Root(), f.runID, string(StageHandedBack), "withdrew pull request #7's arming", "review receipt of its head") {
		t.Fatal("the record says the arming was withdrawn and the head's review receipt removed")
	}
	if !f.reservationReleased(t) {
		t.Fatal("the reservation is released once the arming is gone")
	}
	if _, err := os.Lstat(receiptAt); !os.IsNotExist(err) {
		t.Fatalf("the hand-back supersedes the head's own review receipt: %v", err)
	}
}

// TestAnArmedLaneWhoseRunnerHasNotReportedIsWithdrawnAndWaits: an armed lane
// sitting CLEAN outside the merge queue whose implementer's runner has not
// reported is neither queued, toggled nor asked about: its arming is
// withdrawn, and the next step waits at its arming for the report.
func TestAnArmedLaneWhoseRunnerHasNotReportedIsWithdrawnAndWaits(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.landedToArmedOnce(t)
	if !l.Landing.Armed {
		t.Fatalf("premise: the lane is armed: %+v", l.Landing)
	}
	f.queueAnswer(t, stubQueue{head: l.Landing.Pushed, enabledAt: "2026-10-15T07:00:00Z", status: "CLEAN"})
	collectedImplementer(t, f.repo.Root(), f.runID)
	rel := currentLane(t, f.repo, f.runID).Receipts[0].Receipt
	before := f.ghLog(t)
	f.step(t)
	log := strings.TrimPrefix(f.ghLog(t), before)
	if strings.Contains(log, "enqueuePullRequest") || strings.Contains(log, "pulls/7/reviews") || strings.Contains(log, "--auto --merge") {
		t.Fatalf("an armed lane whose runner has not reported is not queued, asked about or armed again:\n%s", log)
	}
	if !strings.Contains(log, "pr merge 7 --disable-auto") {
		t.Fatalf("its arming is withdrawn:\n%s", log)
	}
	if l := currentLane(t, f.repo, f.runID); l.Landing.Armed || l.Stage != StageLand {
		t.Fatalf("the lane stays at land, its arming withdrawn: %+v", l)
	}
	if !recordSays(t, f.repo.Root(), f.runID, string(StageLand), "withdrew pull request #7's arming", rel) {
		t.Fatal("the record says the arming was withdrawn, naming the receipt")
	}
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	if r, ok := AsRefusal(err); !ok || !r.Contention || !strings.Contains(r.Reason, rel) || !strings.Contains(r.Reason, "not armed") {
		t.Fatalf("the next step waits at its arming for the runner's report: %v", err)
	}
	if n := runnerWaitLines(t, f, rel); n != 1 {
		t.Fatalf("the record says once what the lane waits for: %d lines", n)
	}
}

// TestALeftOpenLaneWaitsForItsRunnersReport: a lane left open, whose head
// the gate would arm now, is not armed while its implementer's runner has not
// reported: the re-ask waits at its arming as the first ask does.
func TestALeftOpenLaneWaitsForItsRunnersReport(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	f := newLandFixture(t, unreviewedQueueRuleset("MERGE"))
	l := f.landedToArmedOnce(t)
	if l.Landing.Armed || !strings.HasPrefix(l.Landing.Merge, "left open") {
		t.Fatalf("premise: the lane is left open: %+v", l.Landing)
	}
	collectedImplementer(t, f.repo.Root(), f.runID)
	rel := currentLane(t, f.repo, f.runID).Receipts[0].Receipt
	f.approve(t)
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	if r, ok := AsRefusal(err); !ok || !r.Contention || !strings.Contains(r.Reason, rel) || !strings.Contains(r.Reason, "not armed") {
		t.Fatalf("the re-ask waits for the runner's report: %v", err)
	}
	if strings.Contains(f.ghLog(t), "pr merge 7 --auto") {
		t.Fatalf("a left-open lane is not armed while its runner has not reported:\n%s", f.ghLog(t))
	}
	if l := currentLane(t, f.repo, f.runID); l.Landing.Armed {
		t.Fatalf("the lane stays unarmed: %+v", l.Landing)
	}
}
