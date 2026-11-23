package landing

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAPreflightReceiptInADriveLanesWorktreeDoesNotSkipTheGate: a receipt for
// the head found only in a drive lane's worktree, which the lane's stages can
// write, does not stand for the head, so the gate runs; the receipt the gate
// mints counts, even when it mints it in that lane.
func TestAPreflightReceiptInADriveLanesWorktreeDoesNotSkipTheGate(t *testing.T) {
	f := newFixture(t, "")
	lane := addWorktree(t, f, driveLaneDir)
	if err := f.mint(lane); err != nil {
		t.Fatal(err)
	}
	if p, err := ReceiptOutsideLanes(f.repo.Root(), f.head()); err != nil || p != "" {
		t.Fatalf("ReceiptOutsideLanes = %q, %v; want the lane's receipt passed over", p, err)
	}
	if p, err := Receipt(f.repo.Root(), f.head()); err != nil || p == "" {
		t.Fatalf("Receipt = %q, %v; want the lane's receipt found", p, err)
	}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if len(f.gates) != 1 {
		t.Fatalf("the gate ran in %v, want it run once", f.gates)
	}
	if s := step(t, res, "gate"); s.Status != StatusDone {
		t.Fatalf("gate step %+v, want done", s)
	}
}

// TestAPreflightReceiptInsideALaneCountsForNothingFromIt: asked from a lane's
// own worktree, no receipt the listing names stands for the head.
func TestAPreflightReceiptInsideALaneCountsForNothingFromIt(t *testing.T) {
	f := newFixture(t, "")
	lane := addWorktree(t, f, driveLaneDir)
	if err := f.mint(f.repo.Root()); err != nil {
		t.Fatal(err)
	}
	if p, err := ReceiptOutsideLanes(lane, f.head()); err != nil || p != "" {
		t.Fatalf("ReceiptOutsideLanes from the lane = %q, %v; want none", p, err)
	}
	if p, err := ReceiptOutsideLanes(f.repo.Root(), f.head()); err != nil || p == "" {
		t.Fatalf("ReceiptOutsideLanes = %q, %v; want the main checkout's receipt", p, err)
	}
	if _, err := os.Stat(filepath.Join(f.repo.Root(), filepath.FromSlash(ReceiptsRelDir), f.head())); err != nil {
		t.Fatal(err)
	}
}
