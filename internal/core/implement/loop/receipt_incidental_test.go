package loop

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// A lane resolves the captures it fixed (commands/implement.md: "an optional
// `resolves` list naming each capture the lane fixed"), its own issue and any
// other it fixed on the way. One that is not open at the lane's head, or that
// another holder reserves, is not the lane's to resolve: the receipt is
// refused naming it, and the landing refuses to resolve one a peer reserved
// since.

// singleOnOrigin is a run of the bundle fixture's lead alone, reserved on its
// origin, driven to its implement await; the other two bundle issues are open
// in its ledger and no issue of the run.
func singleOnOrigin(t *testing.T) (repoRoot, bare, runID string, l Lane, dir string) {
	t.Helper()
	repo, bare := bundleRepo(t)
	start, err := Start(repo.Root(), bundleLead, Options{Now: func() time.Time { return time.Now().UTC().Truncate(time.Second) }})
	if err != nil {
		t.Fatal(err)
	}
	advanceTo(t, repo, start.RunID, StageImplement)
	if res, err := advance(repo.Root(), start.RunID, DefaultStages(), Options{}); err != nil || res.Awaiting == nil {
		t.Fatalf("the implement stage awaits an implementer: %+v %v", res, err)
	}
	l = currentLane(t, repo, start.RunID)
	return repo.Root(), bare, start.RunID, l, filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), start.RunID, l.ID)
}

// TestALaneResolvesOnlyAnIncidentalCaptureThatIsOpenAndNotAnothers: beside
// its own issue, a single lane's receipt may resolve another open capture it
// fixed; one that is not open at the lane's head, or that another holder
// reserves, is refused naming it.
func TestALaneResolvesOnlyAnIncidentalCaptureThatIsOpenAndNotAnothers(t *testing.T) {
	root, bare, runID, l, dir := singleOnOrigin(t)
	repo, _ := bundleRepo(t)
	a := bundleFix(t, repo, l, bundleLead)

	const absent = "iss-2611201500000009"
	rc := goodReceipt(t, runID, l, dir, a)
	rc.Resolves = []Resolution{resolving(bundleLead, a), resolving(absent, a)}
	_, err := Receipt(root, runID, writeReceipt(t, dir, rc), DefaultStages(), Options{})
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, absent+", an open issue at the lane's head") {
		t.Fatalf("an incidental id with no open record is refused naming it: %s", r.Reason)
	}

	q := otherCopy(t, bare, quinnEmail)
	if _, err := claim.Reserve(q, claim.Request{Record: bundleTwo, Footprint: []string{bundleFiles[bundleTwo]}}, as(t, q, claim.Options{})); err != nil {
		t.Fatal(err)
	}
	fetchedBy(t, root)
	rc.Resolves = []Resolution{resolving(bundleLead, a), resolving(bundleTwo, a)}
	_, err = Receipt(root, runID, writeReceipt(t, dir, rc), DefaultStages(), Options{})
	r := mustRefusal(t, err)
	if !strings.Contains(r.Reason, bundleTwo+", which another holder reserves") || !strings.Contains(r.Reason, "reserved by "+heldBy(t, quinnEmail)) {
		t.Fatalf("an incidental id another holder reserves is refused naming it and the holder: %s", r.Reason)
	}

	rc.Resolves = []Resolution{resolving(bundleLead, a), resolving(bundleThree, a)}
	if _, err := Receipt(root, runID, writeReceipt(t, dir, rc), DefaultStages(), Options{}); err != nil {
		t.Fatalf("an open capture no one else reserves is resolved beside the lane's issue: %v", err)
	}
}

// TestTheLandingResolvesNoIncidentalCaptureAnotherHolderReservedSince: a
// capture the receipt resolved beside the lane's issue that another holder
// reserves by the landing is refused there, naming it, and the lane's own
// issue alone is not.
func TestTheLandingResolvesNoIncidentalCaptureAnotherHolderReservedSince(t *testing.T) {
	root, bare, runID, l, _ := singleOnOrigin(t)
	st, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	sha := strings.Repeat("c", 40)
	l.Resolves = []Resolution{resolving(bundleLead, sha), resolving(bundleTwo, sha)}
	c := Context{RepoRoot: root, State: st, Now: time.Now()}
	if err := incidentalHeldAtLanding(c, l); err != nil {
		t.Fatalf("a capture no one else reserves lands: %v", err)
	}
	q := otherCopy(t, bare, quinnEmail)
	if _, err := claim.Reserve(q, claim.Request{Record: bundleTwo, Footprint: []string{bundleFiles[bundleTwo]}}, as(t, q, claim.Options{})); err != nil {
		t.Fatal(err)
	}
	fetchedBy(t, root)
	r := mustRefusal(t, incidentalHeldAtLanding(c, l))
	if r.Stage != string(StageLand) || !strings.Contains(r.Reason, bundleTwo) || !strings.Contains(r.Reason, "another holder reserves") {
		t.Fatalf("the landing refuses naming the capture another holder reserves: %+v", r)
	}
	l.Resolves = l.Resolves[:1]
	if err := incidentalHeldAtLanding(c, l); err != nil {
		t.Fatalf("the lane's own issue alone lands: %v", err)
	}
}
