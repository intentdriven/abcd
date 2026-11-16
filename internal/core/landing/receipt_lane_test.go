package landing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// driveLaneDir is a worktree path named as the implement loop names a lane's.
const driveLaneDir = "run-0000000000000001-lane-1"

// addWorktree adds a detached worktree of f's repository at a fresh directory
// named name, with its local tier made, and returns its path.
func addWorktree(t *testing.T, f *fixture, name string) string {
	t.Helper()
	wt := filepath.Join(t.TempDir(), name)
	f.repo.Git("worktree", "add", "-q", "--detach", wt, "HEAD")
	if err := os.MkdirAll(filepath.Join(wt, filepath.FromSlash(TierRelDir)), 0o700); err != nil {
		t.Fatal(err)
	}
	return wt
}

func TestADriveLaneWorktreeIsToldByItsName(t *testing.T) {
	for path, want := range map[string]bool{
		"/home/u/.abcd.noindex/worktrees/abc/run-0000000000000001-lane-1":   true,
		"/home/u/.abcd.noindex/worktrees/abc/run-1234567890123456-lane-42/": true,
		"/elsewhere/run-0000000000000001-lane-9999":                         true,
		"/home/u/.abcd.noindex/worktrees/abc/receipt-scope":                 false,
		"/home/u/.abcd.noindex/worktrees/abc/run-0000000000000001-lane-0":   false,
		"/home/u/.abcd.noindex/worktrees/abc/run-000000000000001-lane-1":    false,
		"/home/u/.abcd.noindex/worktrees/abc/run-0000000000000001-lane-1x":  false,
		"/home/u/.abcd.noindex/worktrees/abc/x-run-0000000000000001-lane-1": false,
		"/home/u/.abcd.noindex/worktrees/abc/run-0000000000000001-lane-1/x": false,
		"/home/u/projects/repo": false,
	} {
		if got := DriveLaneWorktree(path); got != want {
			t.Errorf("DriveLaneWorktree(%q) = %v, want %v", path, got, want)
		}
	}
}

// A full pair in a drive lane's worktree counts for nothing under auto: the
// gate arms nothing on it and its reason names the receipt it passed over;
// the same pair in a person's linked worktree arms.
func TestAReceiptInADriveLanesWorktreeNeverArms(t *testing.T) {
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	f.forge.head = f.head()
	lane := addWorktree(t, f, driveLaneDir)
	writeReceiptAt(t, lane, f.head(), fullPairBody(f.head()), 0o600)

	d := gateAt(t, f)
	if d.Arm || d.Unread {
		t.Fatalf("decision %+v: want nothing armed on a lane's receipt, and no unread answer", d)
	}
	if !strings.Contains(d.Why, "was passed over") || !strings.Contains(d.Why, driveLaneDir) || !strings.Contains(d.Why, "no review receipt names") {
		t.Fatalf("why = %q; want it to name the receipt passed over and the head as unreviewed", d.Why)
	}
	if proof := readReviewProof(f.repo.Root(), f.head(), ""); proof.reviewed || proof.passed == "" {
		t.Fatalf("proof = %+v; want no review and the lane's receipt named", proof)
	}
	if p, err := reviewReceipt(f.repo.Root(), f.head()); err != nil || p != "" {
		t.Fatalf("reviewReceipt = %q, %v; want the lane's receipt passed over", p, err)
	}

	// The same pair recorded in a person's own linked worktree counts, and the
	// reason still says the lane's was passed over.
	mine := addWorktree(t, f, "mine")
	if _, err := RecordReview(mine, f.head(), fullPair(f.head())); err != nil {
		t.Fatal(err)
	}
	d = gateAt(t, f)
	if !d.Arm {
		t.Fatalf("decision %+v: want the person's pair to arm", d)
	}
	if !strings.Contains(d.Why, "was passed over") {
		t.Fatalf("why = %q; want it to name the lane's receipt passed over", d.Why)
	}
}

// A delta cannot rest on a full pair recorded in a drive lane's worktree.
func TestADeltaCannotRestOnALanesFullPair(t *testing.T) {
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	base := f.head()
	lane := addWorktree(t, f, driveLaneDir)
	writeReceiptAt(t, lane, base, fullPairBody(base), 0o600)
	err := hasFullPair(f.repo.Root(), base)
	if err == nil || !strings.Contains(err.Error(), "was passed over") {
		t.Fatalf("hasFullPair = %v; want no full pair, naming the lane's receipt", err)
	}
}

// The release hold's re-arm asks the same gate, so a lane's receipt arms
// nothing there either.
func TestARearmCountsNoReceiptInADriveLanesWorktree(t *testing.T) {
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	f.forge.head = f.head()
	lane := addWorktree(t, f, driveLaneDir)
	writeReceiptAt(t, lane, f.head(), fullPairBody(f.head()), 0o600)
	ask := func(args ...string) (string, error) { return f.forge.run(f.repo.Root(), args...) }
	arm, why, err := rearmGate(ask, f.repo.Root(), "o/r", "7", f.head())
	if err != nil || arm || !strings.Contains(why, "was passed over") {
		t.Fatalf("arm %v, why %q, err %v: want no arming on a lane's receipt, naming it", arm, why, err)
	}
	if vouched, err := headVouched(f.repo.Root(), f.head()); err != nil || vouched {
		t.Fatalf("headVouched = %v, %v; want a lane's receipt to vouch for nothing", vouched, err)
	}
	f.reviewed(t)
	if arm, why, err := rearmGate(ask, f.repo.Root(), "o/r", "7", f.head()); err != nil || !arm {
		t.Fatalf("arm %v, why %q, err %v: want the checkout's own receipt to arm", arm, why, err)
	}
}

// A recorded approval in a drive lane's worktree counts for nothing under
// human; one in a person's linked worktree counts.
func TestAnApprovalInADriveLanesWorktreeNeverCounts(t *testing.T) {
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": humanConfig})
	f.forge.head = f.head()
	f.forge.reviews = "[]"
	lane := addWorktree(t, f, driveLaneDir)
	if _, err := RecordApproval(lane, f.head()); err != nil {
		t.Fatal(err)
	}
	d := gateAt(t, f)
	if d.Arm || d.ApprovalCounted || !strings.Contains(d.Why, "was passed over") || !strings.Contains(d.Why, driveLaneDir) {
		t.Fatalf("decision %+v: want nothing armed on a lane's approval, naming it", d)
	}
	mine := addWorktree(t, f, "mine")
	if _, err := RecordApproval(mine, f.head()); err != nil {
		t.Fatal(err)
	}
	if d := gateAt(t, f); !d.Arm || !d.ApprovalCounted {
		t.Fatalf("decision %+v: want the person's recorded approval to arm", d)
	}
}
