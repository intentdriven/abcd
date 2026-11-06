package loop

import (
	"testing"
)

// TestADriveTakesOnlyAReceiptItsOwnPreflightMinted: a receipt a lane or anything
// else left is treated as none: the lane waits for the driver's own preflight.
func TestADriveTakesOnlyAReceiptItsOwnPreflightMinted(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.validatedOnce(t)
	for range 2 { // the landing's records are committed, which moves the head
		f.step(t)
	}
	l := currentLane(t, f.repo, f.runID)
	preflighted(t, l, l.HeadSHA)
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{ReceiptOwn: func(string) bool { return false }})
	r, ok := AsRefusal(err)
	if !ok || r.checkWait.IsZero() {
		t.Fatalf("a receipt not its own is no receipt: %v", err)
	}
	if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{ReceiptOwn: func(string) bool { return true }}); err != nil {
		t.Fatalf("its own receipt lands the lane: %v", err)
	}
	if got := f.remoteBranch(t, l.Branch); got != l.HeadSHA {
		t.Fatalf("the head is pushed on its own receipt: %q, want %s", got, l.HeadSHA)
	}
}
