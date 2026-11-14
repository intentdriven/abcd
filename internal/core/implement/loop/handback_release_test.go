package loop

import (
	"path/filepath"
	"strings"
	"testing"

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

// TestALaneHandBackSaveRefusedLeavesTheReservationHeld: the lane's own
// receipt hands it back (handIn); its save refused, the reservation stays
// held; handed in again and saved, it is released.
func TestALaneHandBackSaveRefusedLeavesTheReservationHeld(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	res, err := startedOnce(t, repo, secondIssue)
	if err != nil {
		t.Fatal(err)
	}
	id := res.RunID
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
	path := writeReceipt(t, dir, rc)
	if tomb := originReservation(t, bare, claim.RefPrefix+secondIssue); tomb.commit == "" || tomb.released {
		t.Fatalf("premise: the run holds its reservation: %+v", tomb)
	}

	restore := refuseHandingBack(t)
	_, err = Receipt(repo.Root(), id, path, DefaultStages(), Options{})
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
