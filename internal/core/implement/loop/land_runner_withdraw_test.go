package loop

import (
	"os"
	"path/filepath"
	"regexp"
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

// loopReceipt records the review receipt for the pushed head of the run's lane
// in hand the way recordOwnReview does: through landing.RecordReview, with each
// non-auditor return of the lane's last round by its name and text. A return
// names the head it read on its first line, and recordOwnReview records only a
// round that read the pushed head, as an auto-mode landing's records round
// does, so the lane's last round is first made to have read it, each return
// naming that head where it named the one the round read. With other set, one return's text handed in is another
// text than its file's, as a receipt a person recorded from other returns
// carries. It returns the receipt's path.
func loopReceipt(t *testing.T, root, runID string, other bool) string {
	t.Helper()
	read := ""
	if err := mutate(root, runID, func(_ *os.Root, st *State) (bool, error) {
		i := st.current()
		read = st.Lanes[i].Validation[len(st.Lanes[i].Validation)-1].HeadSHA
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
	head := l.Landing.Pushed
	var returns []landing.ReviewReturn
	written := map[string]string{}
	for _, v := range l.Validation[len(l.Validation)-1].Validators {
		if v.Role == RoleAuditor {
			continue
		}
		at := filepath.Join(root, filepath.FromSlash(v.Return))
		text, err := os.ReadFile(at)
		if err != nil {
			t.Fatal(err)
		}
		if read != "" && read != head {
			text = []byte(strings.ReplaceAll(string(text), read, head))
		}
		if landing.StatesFullPair(string(text), head) != nil {
			if returnHeadLine.Match(text) {
				text = []byte(returnHeadLine.ReplaceAllString(string(text), "Reviewed head: "+head))
			} else {
				text = append([]byte("Reviewed head: "+head+"\n\n"), text...)
			}
		}
		// The fixture's reviewers of one class return one text; each reviewer
		// writes its own, and a receipt counts no text twice.
		if !strings.Contains(string(text), "### Findings") {
			t.Fatalf("premise: the return has its findings section: %s", text)
		}
		text = []byte(strings.Replace(string(text), "### Findings", "Read as "+v.Return+".\n\n### Findings", 1))
		if err := os.WriteFile(at, text, 0o600); err != nil {
			t.Fatal(err)
		}
		written[v.Return] = answerDigest(text)
		if other && len(returns) == 0 {
			text = append(text, "a person's own reading\n"...)
		}
		returns = append(returns, landing.ReviewReturn{Name: v.Return, Text: string(text)})
	}
	// The round read the returns as now written, so their digests are the
	// ones its verdicts were recorded with.
	if err := mutate(root, runID, func(_ *os.Root, st *State) (bool, error) {
		i := st.current()
		r := &st.Lanes[i].Validation[len(st.Lanes[i].Validation)-1]
		for k := range r.Validators {
			if d, ok := written[r.Validators[k].Return]; ok && r.Validators[k].ReturnSHA256 != "" {
				r.Validators[k].ReturnSHA256 = d
			}
		}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	at, err := landing.RecordReview(root, head, returns)
	if err != nil {
		t.Fatalf("the round's returns are recorded as recordOwnReview records them: %v", err)
	}
	if !filepath.IsAbs(at) {
		at = filepath.Join(root, filepath.FromSlash(at))
	}
	return at
}

// returnHeadLine is a return's line naming the head it read.
var returnHeadLine = regexp.MustCompile(`(?m)^Reviewed head: [0-9a-f]+`)

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

// TestADrainLaneReadAtItsPullRequestThenOwingItsArmingIsDrivenUntilItClears:
// the common case. The drain has already read the lane at its pull request when
// a late report hands its run back with the pull request still armed and the
// withdrawal does not take: the next move takes the lane back to in progress
// and names its step, routing nothing and keeping the reservation; once a step
// has withdrawn the arming, the next move routes the hand-back.
func TestADrainLaneReadAtItsPullRequestThenOwingItsArmingIsDrivenUntilItClears(t *testing.T) {
	f, path, report := lateReportArmed(t)
	root := f.repo.Root()
	if _, err := Drain(root, Options{}, DrainOptions{}); err != nil {
		t.Fatal(err)
	}
	if got := drainLaneOutcome(t, root, f.runID); got != DrainLanePullRequest {
		t.Fatalf("premise: the drain reads the armed lane at its pull request: %s", got)
	}
	f.ghFile(t, "refuse-disarm", "")
	f.handInLate(t, path, report)
	res, err := Drain(root, Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got := drainLaneOutcome(t, root, f.runID); got != DrainLaneInProgress {
		t.Fatalf("the lane owing its withdrawal is in progress again: %s", got)
	}
	if len(res.Routed) != 0 || !strings.Contains(res.Next, "implement step --run "+f.runID) || !strings.Contains(res.Next, "tries the withdrawal again") {
		t.Fatalf("the drain routes nothing and names the lane's step: %+v", res)
	}
	if f.reservationReleased(t) {
		t.Fatal("the reservation is kept while the pull request is armed")
	}
	if _, err := advance(root, f.runID, f.stages, Options{}); mustRefusal(t, err).Contention == false {
		t.Fatalf("the step tries the withdrawal again and waits on it: %v", err)
	}
	f.ghGone(t, "refuse-disarm")
	if _, err := advance(root, f.runID, f.stages, Options{}); err != nil {
		t.Fatalf("the step that withdraws the arming: %v", err)
	}
	res, err = Drain(root, Options{}, DrainOptions{})
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

// drainLaneOutcome is the outcome the drain's state stores for runID's lane.
func drainLaneOutcome(t *testing.T, root, runID string) string {
	t.Helper()
	st, _, err := readDrain(root)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(st.Lanes, func(l DrainLane) bool { return l.RunID == runID })
	if i < 0 {
		t.Fatalf("the drain has no lane for %s: %+v", runID, st.Lanes)
	}
	return st.Lanes[i].Outcome
}

// TestADrainDrivesANewerLaneWhileAnOwedWithdrawalKeepsFailing: a lane in
// progress only for the withdrawal its hand-back owes holds up no newer lane
// the drain has in progress: the move names the newer lane's step and hands
// a drive that lane, and the owed lane stays in progress, its step still trying the withdrawal, its
// reservation kept.
func TestADrainDrivesANewerLaneWhileAnOwedWithdrawalKeepsFailing(t *testing.T) {
	f, path, report := lateReportArmed(t)
	root := f.repo.Root()
	newer := drainToLane(t, f.repo, Options{}, DrainOptions{})
	if newer.Start == nil || newer.Lane == nil || newer.Lane.Issue != eligibleIssue {
		t.Fatalf("premise: with the armed lane at its pull request, the drain opens the next issue's lane: %+v", newer)
	}
	f.ghFile(t, "refuse-disarm", "")
	f.handInLate(t, path, report)
	res, err := Drain(root, Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Next, "implement step --run "+newer.Start.RunID) || strings.Contains(res.Next, f.runID) {
		t.Fatalf("the drain names the newer lane's step, not the owed one's: %s", res.Next)
	}
	if res.Lane == nil || res.Lane.RunID != newer.Start.RunID {
		t.Fatalf("the lane a drive takes up is the newer one: %+v", res.Lane)
	}
	if got := drainLaneOutcome(t, root, f.runID); got != DrainLaneInProgress {
		t.Fatalf("the owed lane stays in progress: %s", got)
	}
	_, err = advance(root, f.runID, f.stages, Options{})
	if r := mustRefusal(t, err); !r.Contention || !strings.Contains(r.Reason, "the forge refused to withdraw the arming") {
		t.Fatalf("the owed lane's step still tries the withdrawal: %+v", r)
	}
	if f.reservationReleased(t) {
		t.Fatal("the owed lane's reservation is kept")
	}
}

// TestAnOwedWithdrawalPastTheBoundIsAPersonsToWithdraw: once the stuck bound's
// minutes have passed since the lane's first line of a withdrawal that did not
// take, the drain's next and the step's refusal say a person withdraws the
// arming or decides the lane, naming the pull request, and the reservation is
// kept.
func TestAnOwedWithdrawalPastTheBoundIsAPersonsToWithdraw(t *testing.T) {
	f, path, report := lateReportArmed(t)
	root := f.repo.Root()
	f.ghFile(t, "refuse-disarm", "")
	f.handInLate(t, path, report)
	res, err := Drain(root, Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(res.Next, "needs a person") {
		t.Fatalf("premise: inside the bound the drain names the step alone: %s", res.Next)
	}
	// The hand-back and its lines two hours back: past the bundled sixty minutes.
	back := 2 * time.Hour
	if err := mutate(root, f.runID, func(_ *os.Root, st *State) (bool, error) {
		i := st.current()
		hb := *st.Lanes[i].HandBack
		hb.At = hb.At.Add(-back)
		st.Lanes[i].HandBack = &hb
		for k, e := range st.Record {
			if e.Lane == st.Lanes[i].ID && e.Stage == string(StageHandedBack) {
				st.Record[k].At = e.At.Add(-back)
			}
		}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	if !recordSays(t, root, f.runID, string(StageHandedBack), "pull request #7 is armed and", "the forge refused to withdraw the arming") {
		t.Fatal("premise: the record holds the first try that did not take")
	}
	res, err = Drain(root, Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"needs a person", "pull request #7", "minutes of tries", "a person withdraws it with `gh pr merge 7 --disable-auto`", "or decides the lane", "past the bound of 60", "kept and renewed"} {
		if !strings.Contains(res.Next, want) {
			t.Fatalf("past the bound the drain's next names the person's act (%q): %s", want, res.Next)
		}
	}
	_, err = advance(root, f.runID, f.stages, Options{})
	r := mustRefusal(t, err)
	if !r.Contention || !strings.Contains(r.Reason, "pull request #7 is armed") || !strings.Contains(r.Remedy, "a person withdraws it with `gh pr merge 7 --disable-auto`") || !strings.Contains(r.Remedy, "or decides the lane") {
		t.Fatalf("past the bound the step's refusal names the person's act: %+v", r)
	}
	if f.reservationReleased(t) {
		t.Fatal("the reservation is kept past the bound")
	}
	if got := drainLaneOutcome(t, root, f.runID); got != DrainLaneInProgress {
		t.Fatalf("the lane stays in progress past the bound: %s", got)
	}
}

// TestAnOwedWithdrawalWithNoTryLineCountsFromTheHandBack: a lane whose record
// holds no line of a try that did not take, as one an older build handed back,
// counts its bound from the hand-back, and past it the drain's next and the
// step's refusal say the minutes since it was handed back, not minutes of
// tries.
func TestAnOwedWithdrawalWithNoTryLineCountsFromTheHandBack(t *testing.T) {
	f, path, report := lateReportArmed(t)
	root := f.repo.Root()
	f.ghFile(t, "refuse-disarm", "")
	f.handInLate(t, path, report)
	back := 2 * time.Hour
	if err := mutate(root, f.runID, func(_ *os.Root, st *State) (bool, error) {
		i := st.current()
		lane := st.Lanes[i]
		hb := *lane.HandBack
		hb.At = hb.At.Add(-back)
		st.Lanes[i].HandBack = &hb
		prefix := owedRetryPrefix(lane)
		st.Record = slices.DeleteFunc(st.Record, func(e Entry) bool {
			return e.Lane == lane.ID && strings.HasPrefix(e.Note, prefix)
		})
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	if recordSays(t, root, f.runID, string(StageHandedBack), "pull request #7 is armed and") {
		t.Fatal("premise: the record holds no line of a try that did not take")
	}
	const since = "minutes since it was handed back, past the bound of 60"
	res, err := Drain(root, Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Next, "needs a person") || !strings.Contains(res.Next, since) || strings.Contains(res.Next, "minutes of tries") {
		t.Fatalf("past the bound the drain's next counts the minutes since the hand-back: %s", res.Next)
	}
	_, err = advance(root, f.runID, f.stages, Options{})
	r := mustRefusal(t, err)
	if !r.Contention || !strings.Contains(r.Remedy, since) || strings.Contains(r.Remedy, "minutes of tries") {
		t.Fatalf("past the bound the step's refusal counts the minutes since the hand-back: %+v", r)
	}
}

// TestEachLineOfAWithdrawalNamesTheReceiptsMoveAlike: a late writer's
// hand-back whose first withdrawal does not take moves the loop's own review
// receipt aside and its line says so; the line of the later try that takes
// says it in the same words.
func TestEachLineOfAWithdrawalNamesTheReceiptsMoveAlike(t *testing.T) {
	f, path, report := lateReportArmed(t)
	root := f.repo.Root()
	receiptAt := loopReceipt(t, root, f.runID, false)
	head := currentLane(t, f.repo, f.runID).Landing.Pushed
	moved := "the review receipt the loop recorded for its head " + shortSHA(head) + " from the lane's round is moved aside to " + head + voidedSuffix
	f.ghFile(t, "refuse-disarm", "")
	f.handInLate(t, path, report)
	if !recordSays(t, root, f.runID, string(StageHandedBack), "the forge refused to withdraw the arming", moved) {
		t.Fatal("the first try's line says the receipt is moved aside")
	}
	if _, err := os.Lstat(receiptAt + voidedSuffix); err != nil {
		t.Fatalf("premise: the receipt is moved aside: %v", err)
	}
	f.ghGone(t, "refuse-disarm")
	res, err := advance(root, f.runID, f.stages, Options{})
	if err != nil {
		t.Fatalf("the step that withdraws the arming: %v", err)
	}
	if !recordSays(t, root, f.runID, string(StageHandedBack), "withdrew pull request #7's arming, since", moved) || !strings.Contains(res.Next, moved) {
		t.Fatalf("the line of the try that takes names the move as the first did: %s", res.Next)
	}
}

// TestAStopSaysALaneOwingItsWithdrawalIsStillArmed: a person's stop of a run
// whose lane owes its arming's withdrawal releases what it released before;
// its record line says the pull request is still armed and that a person
// withdraws it, and its answer and its page name the armed pull request and
// its disarm once each.
func TestAStopSaysALaneOwingItsWithdrawalIsStillArmed(t *testing.T) {
	f, path, report := lateReportArmed(t)
	root := f.repo.Root()
	f.ghFile(t, "refuse-disarm", "")
	f.handInLate(t, path, report)
	o, _, folder := stopped(t)
	at := time.Now()
	o.Now = func() time.Time { return at }
	r := wantStopRefusal(t, checkOnce(root, f.runID, o))
	c := controlOf(t, root, f.runID)
	if c.Stop == nil || c.Stop.HaltedAt == nil {
		t.Fatalf("premise: the stop halts the run: %+v", c.Stop)
	}
	const armed, person = "pull request #7 is still armed", "a person withdraws it with `gh pr merge 7 --disable-auto`"
	if !recordSays(t, root, f.runID, StageStop, armed, person) {
		t.Fatal("the stop's record line says the pull request is still armed and a person withdraws it")
	}
	const named, disarm = "pull request #7", "gh pr merge 7 --disable-auto"
	if strings.Count(r.Reason, named) != 1 || strings.Count(r.Reason, disarm) != 1 || !strings.Contains(r.Reason, "can still merge") {
		t.Fatalf("the stop's answer names the armed pull request and its disarm once: %s", r.Reason)
	}
	page := pageOf(t, folder, at)
	if strings.Count(page, named) != 1 || strings.Count(page, disarm) != 1 || !strings.Contains(page, "the withdrawal of its arming not yet taken") {
		t.Fatalf("the stop's page names the lane's state, and the armed pull request and its disarm once:\n%s", page)
	}
	i := slices.IndexFunc(c.Stop.Left, func(l StopLeft) bool { return l.ArmedPR == 7 && l.Stage == StageHandedBack })
	if i < 0 || !c.Stop.Left[i].Settled || strings.Contains(c.Stop.Left[i].Note, named) {
		t.Fatalf("what the stop left names the armed pull request through ArmedPR alone, settled: %+v", c.Stop.Left)
	}
	if !f.reservationReleased(t) {
		t.Fatal("the stop releases the reservation as it did")
	}
	if wantStopRefusal(t, checkOnce(root, f.runID, o)); countLines(t, root, f.runID, StageStop, armed) != 1 {
		t.Fatal("a later check says it no second time")
	}
}

// countLines counts the record lines at stage that say want.
func countLines(t *testing.T, root, runID, stage, want string) int {
	t.Helper()
	st, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range st.Record {
		if e.Stage == stage && strings.Contains(e.Note, want) {
			n++
		}
	}
	return n
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
