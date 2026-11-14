package loop

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/gittest"
)

// A hand-in that hands a lane back releases the run's reservation on the
// remote only once the save that hands it back has returned: a save the
// models check refuses leaves the lane where it stood and its reservation
// held, and the same hand-in saved releases it as before.

// refuseHandingBack stands in for the models check so that a save leaving a
// lane handed back is refused, as modelsSavable refuses a save, and every
// other save is judged as it is. The returned func puts the check back.
func refuseHandingBack(t *testing.T) func() {
	t.Helper()
	old := modelsCheck
	restore := func() { modelsCheck = old }
	t.Cleanup(restore)
	modelsCheck = func(m State) error {
		for _, l := range m.Lanes {
			if l.HandBack != nil {
				return refuse("state", "", "", StateRelPath(m.RunID)+" was not saved: the stand-in models check refuses a lane handed back",
					"this is a fault in abcd, so report it")
			}
		}
		return old(m)
	}
	return restore
}

// heldAfterRefusal checks a refused hand-back save released nothing: the
// stand-in remote shows no release, the record logs none, and the lane is not
// handed back.
func heldAfterRefusal(t *testing.T, repo *gittest.Repo, bare, runID string, err error) {
	t.Helper()
	if r := mustRefusal(t, err); r.Stage != "state" || !strings.Contains(r.Reason, "was not saved") {
		t.Fatalf("the hand-in returns the save's refusal: %+v", r)
	}
	if tomb := originReservation(t, bare, claim.RefPrefix+secondIssue); tomb.released {
		t.Fatalf("a refused save leaves the reservation held: %+v", tomb)
	}
	if note := recordNote(t, repo, runID, StageReservationReleased); note != "" {
		t.Fatalf("a refused save logs no release: %q", note)
	}
	if l := currentLane(t, repo, runID); l.HandBack != nil {
		t.Fatalf("a refused save leaves the lane not handed back: %+v", l.HandBack)
	}
}

// releasedAfterSave checks a saved hand-back released the reservation and
// logged it, as before.
func releasedAfterSave(t *testing.T, repo *gittest.Repo, bare, runID string) {
	t.Helper()
	if tomb := originReservation(t, bare, claim.RefPrefix+secondIssue); !tomb.released || tomb.holder != heldBy(t, patEmail) {
		t.Fatalf("a saved hand-back releases the reservation: %+v", tomb)
	}
	if note := recordNote(t, repo, runID, StageReservationReleased); !strings.Contains(note, "released the reservation on "+secondIssue) {
		t.Fatalf("the run record says the reservation was released: %q", note)
	}
	if l := currentLane(t, repo, runID); l.HandBack == nil {
		t.Fatalf("the saved hand-in hands the lane back: %+v", l)
	}
}

// handingBackAtImplementer is a reserving issue run whose lane awaits its
// implementer, with a receipt written at path that hands the lane back.
func handingBackAtImplementer(t *testing.T) (repo *gittest.Repo, bare, id, path string) {
	t.Helper()
	repo, bare = reservingDrainRepo(t, "commands/drain.md")
	res, err := startedOnce(t, repo, secondIssue)
	if err != nil {
		t.Fatal(err)
	}
	id = res.RunID
	for range len(Sequence) {
		if currentLane(t, repo, id).awaiting() != nil {
			break
		}
		if _, err := advance(repo.Root(), id, DefaultStages(), Options{}); err != nil {
			t.Fatal(err)
		}
	}
	l := currentLane(t, repo, id)
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), id, l.ID)
	rc := goodReceipt(t, id, l, dir)
	rc.HandBack = &LaneHandBack{Kind: HandBackDesignFinding, Reason: "the page needs a decision", Home: "the brief"}
	path = writeReceipt(t, dir, rc)
	if tomb := originReservation(t, bare, claim.RefPrefix+secondIssue); tomb.commit == "" || tomb.released {
		t.Fatalf("premise: the run holds its reservation: %+v", tomb)
	}
	return repo, bare, id, path
}

// TestALaneHandBackSaveRefusedLeavesTheReservationHeld: the lane's own
// receipt hands it back (handIn); its save refused, the reservation stays
// held; handed in again and saved, it is released.
func TestALaneHandBackSaveRefusedLeavesTheReservationHeld(t *testing.T) {
	repo, bare, id, path := handingBackAtImplementer(t)

	restore := refuseHandingBack(t)
	_, err := Receipt(repo.Root(), id, path, DefaultStages(), Options{})
	heldAfterRefusal(t, repo, bare, id, err)

	restore()
	if _, err := Receipt(repo.Root(), id, path, DefaultStages(), Options{}); err != nil {
		t.Fatalf("the hand-back saved: %v", err)
	}
	releasedAfterSave(t, repo, bare, id)
}

// TestALateWriterHandBackSaveRefusedLeavesTheReservationHeld: a runner's
// report reaching a receipt a step collected, naming a reviewer's own model
// after the round left the validate stage, hands the lane back as stuck
// (answerCollected); its save refused, the reservation stays held; reported
// again and saved, it is released.
func TestALateWriterHandBackSaveRefusedLeavesTheReservationHeld(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	res, err := startedOnce(t, repo, secondIssue)
	if err != nil {
		t.Fatal(err)
	}
	id, stages := res.RunID, DefaultStages()
	stepTo(t, repo, id, stages, StageImplement)
	if r, err := advance(repo.Root(), id, stages, Options{}); err != nil || r.Awaiting == nil {
		t.Fatalf("the implement stage awaits an implementer: %+v %v", r, err)
	}
	l := currentLane(t, repo, id)
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), id, l.ID)
	sha := laneCommit(t, repo, l, "fix.txt")
	rc := goodReceipt(t, id, l, dir, sha)
	rc.Resolves = []Resolution{{Issue: secondIssue, Commit: sha, Note: "the page names the flag", Impact: "fix",
		Grounds: "pursued: the page names the flag; shown wrong if a reader cannot find it"}}
	if _, err := Receipt(repo.Root(), id, writeReceipt(t, dir, rc), stages, Options{}); err != nil {
		t.Fatal(err)
	}
	path := collectedImplementer(t, repo.Root(), id)
	passRound(t, repo, id, stages, RoleRuthless, RoleSecurity)
	for range 3 {
		l := currentLane(t, repo, id)
		if l.Stage != StageValidate {
			break
		}
		if a := l.awaiting(); a != nil && a.Role == RoleAuditor {
			passRound(t, repo, id, stages, RoleAuditor)
			continue
		}
		if _, err := advance(repo.Root(), id, stages, Options{}); err != nil {
			t.Fatal(err)
		}
	}
	l = currentLane(t, repo, id)
	if l.Stage == StageValidate || l.HandBack != nil {
		t.Fatalf("premise: the round passed and the lane left the validate stage: %+v", l)
	}
	model := ""
	for _, v := range l.Validation[len(l.Validation)-1].Validators {
		if v.Class != "" && v.ReviewerModel != "" {
			model = v.ReviewerModel
		}
	}
	if model == "" {
		t.Fatalf("premise: a reviewer ran on a class: %+v", l.Validation)
	}
	report := &runner.RouteRecord{Asked: runner.Claude, Ran: runner.Claude, Model: model}

	restore := refuseHandingBack(t)
	_, err = receipt(repo.Root(), id, path, stages, Options{}, report)
	heldAfterRefusal(t, repo, bare, id, err)

	restore()
	if _, err := receipt(repo.Root(), id, path, stages, Options{}, report); err != nil {
		t.Fatalf("the late report saved: %v", err)
	}
	releasedAfterSave(t, repo, bare, id)
	if hb := currentLane(t, repo, id).HandBack; hb.Kind != HandBackStuck {
		t.Fatalf("the late writer hands the lane back as stuck: %+v", hb)
	}
}

// TestAHandBackWhoseReleaseFailsSaysNotReleased: the hand-back is written and
// origin refuses the tombstone, so the hand-in succeeds, the lane is handed
// back, the reservation is still held, the record logs the release owed, and
// the answer says "not released", why, and when the lease ends. A plain step
// on the run then renews nothing, though the lease is past its half: nothing
// renews a handed-back lane's reservation.
func TestAHandBackWhoseReleaseFailsSaysNotReleased(t *testing.T) {
	repo, bare, id, path := handingBackAtImplementer(t)
	before := originReservation(t, bare, claim.RefPrefix+secondIssue)
	undo := refuseReservationPush(t, bare, secondIssue)

	res, err := Receipt(repo.Root(), id, path, DefaultStages(), Options{})
	if err != nil {
		t.Fatalf("the hand-back is written, so the hand-in succeeds: %v", err)
	}
	if l := currentLane(t, repo, id); l.HandBack == nil {
		t.Fatalf("the lane is handed back: %+v", l)
	}
	if tomb := originReservation(t, bare, claim.RefPrefix+secondIssue); tomb.released || tomb.commit != before.commit {
		t.Fatalf("origin refused the tombstone, so the reservation is still held: %+v", tomb)
	}
	if note := recordNote(t, repo, id, StageReservationReleaseFailed); !strings.Contains(note, "could not push the tombstone") {
		t.Fatalf("the record logs the release owed: %q", note)
	}
	if note := recordNote(t, repo, id, StageReservationReleased); note != "" {
		t.Fatalf("a release origin refused is not logged as made: %q", note)
	}
	for _, want := range []string{"the lane was handed back and its reservation not released", "could not push the tombstone",
		"unless a drain routing the hand-back releases it", "its lease ends at ", "nothing renews a handed-back lane's reservation"} {
		if !strings.Contains(res.Next, want) {
			t.Fatalf("the answer says %q: %s", want, res.Next)
		}
	}
	if strings.Contains(res.Next, "released, not recorded") {
		t.Fatalf("a release not made is not told as made: %s", res.Next)
	}

	undo()
	later := Options{Now: func() time.Time { return time.Now().Add(90 * time.Minute) }}
	_, _ = advance(repo.Root(), id, DefaultStages(), later)
	if tomb := originReservation(t, bare, claim.RefPrefix+secondIssue); tomb.commit != before.commit {
		t.Fatalf("a step pushed to a handed-back lane's reservation: %+v, was %+v", tomb, before)
	}
	if note := recordNote(t, repo, id, StageReservationRenewed); note != "" {
		t.Fatalf("a step renews no handed-back lane's reservation: %q", note)
	}
}

// TestAHandBackReleasedButNotRecordedSaysSo: the hand-back is written and the
// release made, and the second write, which records it, is refused: the hand-in
// succeeds, the reservation is gone from origin, the record lacks the line, and
// the answer says "released, not recorded".
func TestAHandBackReleasedButNotRecordedSaysSo(t *testing.T) {
	repo, bare, id, path := handingBackAtImplementer(t)
	old := modelsCheck
	t.Cleanup(func() { modelsCheck = old })
	modelsCheck = func(m State) error {
		for _, e := range m.Record {
			if e.Stage == StageReservationReleased {
				return refuse("state", "", "", StateRelPath(m.RunID)+" was not saved: the stand-in models check refuses the release's line",
					"this is a fault in abcd, so report it")
			}
		}
		return old(m)
	}

	res, err := Receipt(repo.Root(), id, path, DefaultStages(), Options{})
	if err != nil {
		t.Fatalf("the hand-back is written, so the hand-in succeeds: %v", err)
	}
	if l := currentLane(t, repo, id); l.HandBack == nil {
		t.Fatalf("the lane is handed back: %+v", l)
	}
	if tomb := originReservation(t, bare, claim.RefPrefix+secondIssue); !tomb.released {
		t.Fatalf("the release was made: %+v", tomb)
	}
	if note := recordNote(t, repo, id, StageReservationReleased); note != "" {
		t.Fatalf("the refused second write leaves no line: %q", note)
	}
	for _, want := range []string{"the lane was handed back and its reservation released, not recorded", "released the reservation on " + secondIssue,
		"the run record could not be written to say so", "was not saved"} {
		if !strings.Contains(res.Next, want) {
			t.Fatalf("the answer says %q: %s", want, res.Next)
		}
	}
	if strings.Contains(res.Next, "not released") {
		t.Fatalf("a release made is not told as not made: %s", res.Next)
	}
}

// TestTheAfterWriteReleaseReadsTheLaneAndTheRecordAsWritten: the step a
// hand-back registers reads the run as it was written: a lane not handed back
// there is not released for, and a reservation the record already logs
// released since it was made is not released again.
func TestTheAfterWriteReleaseReadsTheLaneAndTheRecordAsWritten(t *testing.T) {
	repo, _, id, path := handingBackAtImplementer(t)
	calls := 0
	was := claimReserve
	t.Cleanup(func() { claimReserve = was })
	counting := func(dir string, req claim.Request, o claim.Options) (claim.Result, error) {
		calls++
		return was(dir, req, o)
	}

	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	lane := st.Lanes[0].ID
	claimReserve = counting
	rel := releaseAfterWrite(repo.Root(), &st, lane, time.Now())
	st.written(&st, func() error { return nil })
	if calls != 0 || rel.made {
		t.Fatalf("a lane not handed back where the state was written is not released for: %d call(s), %+v", calls, rel)
	}

	claimReserve = was
	if _, err := Receipt(repo.Root(), id, path, DefaultStages(), Options{}); err != nil {
		t.Fatal(err)
	}
	if st, err = ReadState(repo.Root(), id); err != nil {
		t.Fatal(err)
	}
	if !releasedThisHold(st) {
		t.Fatalf("premise: the hand-in released and logged it: %+v", st.Record)
	}
	claimReserve = counting
	rel = releaseAfterWrite(repo.Root(), &st, lane, time.Now())
	st.written(&st, func() error { return nil })
	if calls != 0 || rel.made {
		t.Fatalf("a reservation logged released is not released again: %d call(s), %+v", calls, rel)
	}
}

// handingBackStages is DefaultStages with the brief stage's body replaced by one
// that hands the lane back, as a stage's Outcome does (performMove).
func handingBackStages() Stages {
	stages := DefaultStages()
	for i := range stages {
		if stages[i].Name == StageBrief {
			stages[i].Run = func(Context, *Lane) (Outcome, error) {
				return Outcome{HandBack: &HandBack{Kind: HandBackStuck, Reason: "the brief cannot be written", Home: "a person"}}, nil
			}
		}
	}
	return stages
}

// TestAStepHandBackSaveRefusedLeavesTheReservationHeld: a stage that hands the
// lane back (performMove); the step's save refused, the reservation stays held;
// stepped again and saved, it is released.
func TestAStepHandBackSaveRefusedLeavesTheReservationHeld(t *testing.T) {
	c := &clock{t: time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)}
	o := Options{Now: c.now}
	repo, bare, id := stuckDrain(t, c)
	stages := handingBackStages()

	restore := refuseHandingBack(t)
	_, err := advance(repo.Root(), id, stages, o)
	heldAfterRefusal(t, repo, bare, id, err)

	restore()
	if _, err := advance(repo.Root(), id, stages, o); err != nil {
		t.Fatalf("the step's hand-back saved: %v", err)
	}
	releasedAfterSave(t, repo, bare, id)
}

// TestAStuckHandBackSaveRefusedLeavesTheReservationHeld: a drain hands a lane
// back as stuck (handBackStuck); its save refused, the reservation stays held;
// the next move, saved, releases it.
func TestAStuckHandBackSaveRefusedLeavesTheReservationHeld(t *testing.T) {
	c := &clock{t: time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)}
	o := Options{Now: c.now}
	repo, bare, id := stuckDrain(t, c)
	stages := briefRefusing(sameRefusal)
	for range 3 {
		refuseStep(t, repo, id, stages, o)
	}

	restore := refuseHandingBack(t)
	_, _ = Drain(repo.Root(), o, DrainOptions{})
	if tomb := originReservation(t, bare, claim.RefPrefix+secondIssue); tomb.released {
		t.Fatalf("a refused save leaves the reservation held: %+v", tomb)
	}
	if l := currentLane(t, repo, id); l.HandBack != nil {
		t.Fatalf("a refused save leaves the lane not handed back: %+v", l.HandBack)
	}

	restore()
	if _, err := Drain(repo.Root(), o, DrainOptions{}); err != nil {
		t.Fatal(err)
	}
	releasedAfterSave(t, repo, bare, id)
	if hb := currentLane(t, repo, id).HandBack; hb.Kind != claim.HandBackStuck {
		t.Fatalf("the lane is handed back as stuck: %+v", hb)
	}
}

// TestAClosedHandBackSaveRefusedLeavesTheReservationHeld: a parked lane whose
// pull request was closed is handed back as closed (handBackClosed); its save
// refused, the reservation stays held; the next move, saved, releases it.
func TestAClosedHandBackSaveRefusedLeavesTheReservationHeld(t *testing.T) {
	f, l := parkDrainLane(t)
	f.listPRs(t, stubPR{Number: 7, HeadRefName: l.Branch, State: "CLOSED", Mergeable: "UNKNOWN", MergeStateStatus: "UNKNOWN"})

	restore := refuseHandingBack(t)
	_, _ = Drain(f.repo.Root(), Options{}, DrainOptions{})
	if tomb := originReservation(t, f.bare, claim.RefPrefix+secondIssue); tomb.released {
		t.Fatalf("a refused save leaves the reservation held: %+v", tomb)
	}
	if l := currentLane(t, f.repo, f.runID); l.HandBack != nil {
		t.Fatalf("a refused save leaves the lane not handed back: %+v", l.HandBack)
	}

	restore()
	if _, err := Drain(f.repo.Root(), Options{}, DrainOptions{}); err != nil {
		t.Fatal(err)
	}
	if tomb := originReservation(t, f.bare, claim.RefPrefix+secondIssue); !tomb.released {
		t.Fatalf("the saved hand-back releases the reservation: %+v", tomb)
	}
	if hb := currentLane(t, f.repo, f.runID).HandBack; hb == nil || hb.Kind != claim.HandBackClosed {
		t.Fatalf("the lane is handed back as closed: %+v", hb)
	}
	if note := recordNote(t, f.repo, f.runID, StageReservationReleased); !strings.Contains(note, "released the reservation on "+secondIssue) {
		t.Fatalf("the run record says the reservation was released: %q", note)
	}
}
