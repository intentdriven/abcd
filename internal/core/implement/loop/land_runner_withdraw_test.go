package loop

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

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

// loopReceipt writes the review receipt recordOwnReview would write for the
// pushed head of the run's lane in hand (landing.RecordReview's shape): each
// non-auditor return of the lane's last round, by name, verdict and the sha256
// of its text, read here from the files themselves. recordOwnReview records
// only a round that read the pushed head, as an auto-mode landing's records
// round does, so the lane's last round is first made to have read it. With
// other set, one return's digest is another text's, as a receipt a person
// recorded from other returns would carry. It returns the receipt's path.
func loopReceipt(t *testing.T, root, runID string, other bool) string {
	t.Helper()
	if err := mutate(root, runID, func(_ *os.Root, st *State) (bool, error) {
		i := st.current()
		st.Lanes[i].Validation[len(st.Lanes[i].Validation)-1].HeadSHA = st.Lanes[i].Landing.Pushed
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	l := st.Lanes[st.current()]
	type entry struct {
		Name    string `json:"name"`
		Verdict string `json:"verdict"`
		SHA256  string `json:"sha256"`
	}
	var entries []entry
	for _, v := range l.Validation[len(l.Validation)-1].Validators {
		if v.Role == RoleAuditor {
			continue
		}
		text, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(v.Return)))
		if err != nil {
			t.Fatal(err)
		}
		if other && len(entries) == 0 {
			text = append(text, "a person's own reading\n"...)
		}
		sum := sha256.Sum256(text)
		entries = append(entries, entry{Name: v.Return, Verdict: v.Verdict, SHA256: hex.EncodeToString(sum[:])})
	}
	data, err := json.MarshalIndent(map[string]any{"commit": l.Landing.Pushed, "verdict_rule": 2, "recorded_at": "2026-11-15T09:00:00Z", "reviews": entries}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	at := filepath.Join(root, filepath.FromSlash(landing.ReviewReceiptsRelDir), l.Landing.Pushed)
	if err := os.MkdirAll(filepath.Dir(at), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(at, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return at
}

// TestALateArmingWithdrawnReleasesTheReservation: the withdrawal taken and
// read back as gone clears the arming, says so, and only then releases the
// reservation; the review receipt the loop recorded for the pushed head from
// the lane's round is moved aside, so the release hold's re-arm
// (landing.DecideMerge) no longer counts the pair that judged it.
func TestALateArmingWithdrawnReleasesTheReservation(t *testing.T) {
	f, path, report := lateReportArmed(t)
	receiptAt := loopReceipt(t, f.repo.Root(), f.runID, false)
	f.handInLate(t, path, report)
	if !strings.Contains(f.ghLog(t), "pr merge 7 --disable-auto") {
		t.Fatalf("the forge is asked to withdraw the arming:\n%s", f.ghLog(t))
	}
	if l := currentLane(t, f.repo, f.runID); l.Landing.Armed || l.Landing.Merge != "" {
		t.Fatalf("the withdrawal clears the arming: %+v", l.Landing)
	}
	if !recordSays(t, f.repo.Root(), f.runID, string(StageHandedBack), "withdrew pull request #7's arming", "the review receipt the loop recorded for its head", "moved aside") {
		t.Fatal("the record says the arming was withdrawn and the loop's own review receipt moved aside")
	}
	if !f.reservationReleased(t) {
		t.Fatal("the reservation is released once the arming is gone")
	}
	if _, err := os.Lstat(receiptAt); !os.IsNotExist(err) {
		t.Fatalf("the late writer's hand-back voids the loop's own review receipt: %v", err)
	}
	if _, err := os.Lstat(receiptAt + voidedSuffix); err != nil {
		t.Fatalf("the voided receipt is kept aside, under a name no reader counts: %v", err)
	}
}

// TestALateWritersHandBackLeavesAReceiptTheLoopDidNotWrite: a review receipt
// at the head's path whose returns are not the lane's round (one a person
// recorded) is left where it is, and the record says it stands as its
// recorder's.
func TestALateWritersHandBackLeavesAReceiptTheLoopDidNotWrite(t *testing.T) {
	f, path, report := lateReportArmed(t)
	receiptAt := loopReceipt(t, f.repo.Root(), f.runID, true)
	before, err := os.ReadFile(receiptAt)
	if err != nil {
		t.Fatal(err)
	}
	f.handInLate(t, path, report)
	if after, err := os.ReadFile(receiptAt); err != nil || string(after) != string(before) {
		t.Fatalf("a receipt the loop did not write is left as it is: %v", err)
	}
	if !recordSays(t, f.repo.Root(), f.runID, string(StageHandedBack), "withdrew pull request #7's arming", "is left in place", "stands as its recorder's") ||
		recordSays(t, f.repo.Root(), f.runID, string(StageHandedBack), "moved aside") {
		t.Fatal("the record says the receipt stands as its recorder's, and never that it was moved aside")
	}
}

// TestALateArmingIsRenewedWhileItIsOwed: past half the lease, a step that
// tries the withdrawal again and is refused renews the reservation the
// hand-back keeps, so no peer takes the issue while the pull request can
// still merge.
func TestALateArmingIsRenewedWhileItIsOwed(t *testing.T) {
	f, path, report := lateReportArmed(t)
	f.ghFile(t, "refuse-disarm", "")
	f.handInLate(t, path, report)
	before := originReservation(t, f.bare, claim.RefPrefix+secondIssue)
	// The fixture's lease runs a day: thirteen hours on is past its half.
	later := Options{Now: func() time.Time { return time.Now().Add(13 * time.Hour) }}
	// The run's window has elapsed by then: the first step closes it, and
	// the next, its pause over, takes the move.
	_, err := advance(f.repo.Root(), f.runID, f.stages, later)
	if err == nil {
		_, err = advance(f.repo.Root(), f.runID, f.stages, later)
	}
	if r := mustRefusal(t, err); !r.Contention || !strings.Contains(r.Reason, "the forge refused to withdraw the arming") {
		t.Fatalf("the step tries the withdrawal again and waits on it: %+v", r)
	}
	if after := originReservation(t, f.bare, claim.RefPrefix+secondIssue); after.commit == before.commit || after.released {
		t.Fatalf("the step renews the kept reservation: before %+v, after %+v", before, after)
	}
	if note := recordNote(t, f.repo, f.runID, StageReservationRenewed); !strings.Contains(note, "renewed the reservation on "+secondIssue) {
		t.Fatalf("the run record logs the renewal: %q", note)
	}
}

// TestADrainDrivesALaneOwingItsWithdrawalUntilItClears: a drain lane whose
// late hand-back's withdrawal did not take is not routed by the drain's move:
// it is still in progress, its step is named, and the drive's check reads it
// so; once a step has withdrawn the arming, the next move routes it.
func TestADrainDrivesALaneOwingItsWithdrawalUntilItClears(t *testing.T) {
	f, path, report := lateReportArmed(t)
	f.ghFile(t, "refuse-disarm", "")
	f.handInLate(t, path, report)
	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Routed) != 0 || !strings.Contains(res.Next, "implement step --run "+f.runID) || !strings.Contains(res.Next, "tries the withdrawal again") {
		t.Fatalf("the drain does not route a lane that owes its withdrawal, and names its step: %+v", res)
	}
	st, _, err := readDrain(f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if i := slices.IndexFunc(st.Lanes, func(l DrainLane) bool { return l.RunID == f.runID }); i < 0 || st.Lanes[i].Outcome != DrainLaneInProgress {
		t.Fatalf("the drain's lane is still in progress: %+v", st.Lanes)
	}
	if !(&terminal{repoRoot: f.repo.Root()}).laneInProgress(f.runID) {
		t.Fatal("the drive keeps stepping a lane that owes its withdrawal")
	}
	if f.reservationReleased(t) {
		t.Fatal("the reservation is kept while the pull request is armed")
	}
	f.ghGone(t, "refuse-disarm")
	if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err != nil {
		t.Fatalf("the step that withdraws the arming: %v", err)
	}
	res, err = Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Routed) != 1 || res.Routed[0].Issue != secondIssue {
		t.Fatalf("once the arming is withdrawn the drain routes the hand-back: %+v", res)
	}
	if !f.reservationReleased(t) {
		t.Fatal("the reservation is released once the arming is gone")
	}
}

// TestAnArmedLaneWhoseRunnerHasNotReportedIsWithdrawnAndWaits: an armed lane
// sitting CLEAN outside the merge queue whose implementer's runner has not
// reported is neither queued, toggled nor asked about: its arming is
// withdrawn, and the next step waits at its arming for the report.
func TestAnArmedLaneWhoseRunnerHasNotReportedIsWithdrawnAndWaits(t *testing.T) {
	f, _, rel := armedUnreported(t)
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

// armedUnreported is an armed lane sitting CLEAN outside the merge queue whose
// implementer's runner has not reported, and that receipt's path.
func armedUnreported(t *testing.T) (*landFixture, Lane, string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.landedToArmedOnce(t)
	if !l.Landing.Armed {
		t.Fatalf("premise: the lane is armed: %+v", l.Landing)
	}
	f.queueAnswer(t, stubQueue{head: l.Landing.Pushed, enabledAt: "2026-10-15T07:00:00Z", status: "CLEAN"})
	collectedImplementer(t, f.repo.Root(), f.runID)
	l = currentLane(t, f.repo, f.runID)
	return f, l, l.Receipts[0].Receipt
}

// TestThePreReportWithdrawalVoidsNoReceipt: the landing's withdrawal of an
// armed lane whose runner has not reported leaves the review receipt the loop
// recorded for its head in place: no report has shown its pair to share a
// model with a writer.
func TestThePreReportWithdrawalVoidsNoReceipt(t *testing.T) {
	f, _, _ := armedUnreported(t)
	receiptAt := loopReceipt(t, f.repo.Root(), f.runID, false)
	f.step(t)
	if !strings.Contains(f.ghLog(t), "pr merge 7 --disable-auto") {
		t.Fatalf("premise: its arming is withdrawn:\n%s", f.ghLog(t))
	}
	if _, err := os.Lstat(receiptAt); err != nil {
		t.Fatalf("the pre-report withdrawal leaves the loop's receipt where it is: %v", err)
	}
	if recordSays(t, f.repo.Root(), f.runID, "", "review receipt") {
		t.Fatal("the record says nothing of the review receipt")
	}
}

// TestTheLandingsWithdrawalIsMadeOnceTheStepIsWritten: a landing step whose
// save is refused asks nothing of the forge; one the forge refuses to
// withdraw is the step's refusal, the lane armed at its landing and nothing
// recorded, and the next step tries again until it takes.
func TestTheLandingsWithdrawalIsMadeOnceTheStepIsWritten(t *testing.T) {
	f, _, rel := armedUnreported(t)
	old := modelsCheck
	t.Cleanup(func() { modelsCheck = old })
	modelsCheck = func(m State) error {
		return refuse("state", "", "", StateRelPath(m.RunID)+" was not saved: the stand-in models check refuses it", "this is a fault in abcd, so report it")
	}
	before := f.ghLog(t)
	if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err == nil {
		t.Fatal("premise: the step's save is refused")
	}
	if strings.Contains(strings.TrimPrefix(f.ghLog(t), before), "--disable-auto") {
		t.Fatalf("a refused save asks nothing of the forge:\n%s", f.ghLog(t))
	}
	modelsCheck = old

	f.ghFile(t, "refuse-disarm", "")
	for try := 1; try <= 2; try++ {
		before = f.ghLog(t)
		_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
		if r := mustRefusal(t, err); !r.Contention || !strings.Contains(r.Reason, "the forge refused to withdraw the arming") || !strings.Contains(r.Reason, rel) {
			t.Fatalf("try %d: the forge's refusal is the step's, naming the receipt: %+v", try, r)
		}
		if !strings.Contains(strings.TrimPrefix(f.ghLog(t), before), "--disable-auto") {
			t.Fatalf("try %d: the step asks the forge to withdraw the arming:\n%s", try, f.ghLog(t))
		}
		if l := currentLane(t, f.repo, f.runID); !l.Landing.Armed || l.Stage != StageLand {
			t.Fatalf("try %d: the lane stays armed at its landing: %+v", try, l)
		}
	}
	if recordSays(t, f.repo.Root(), f.runID, string(StageLand), "withdrew pull request #7's arming") {
		t.Fatal("a withdrawal that did not take is not recorded as made")
	}
	f.ghGone(t, "refuse-disarm")
	res := f.step(t)
	if l := currentLane(t, f.repo, f.runID); l.Landing.Armed || l.Stage != StageLand {
		t.Fatalf("the withdrawal that takes clears the arming: %+v", l)
	}
	if !recordSays(t, f.repo.Root(), f.runID, string(StageLand), "withdrew pull request #7's arming", rel) || !strings.Contains(res.Next, "withdrew pull request #7's arming") {
		t.Fatalf("the record and the answer say the arming was withdrawn: %s", res.Next)
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
