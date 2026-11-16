package landing

import (
	"errors"
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

// writeApprovalAt writes a recorded approval of sha straight into root's local
// tier, as one written there by hand would be: RecordApproval refuses a drive
// lane's worktree.
func writeApprovalAt(t *testing.T, root, sha string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(ApprovalsRelDir), sha)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	body := `{"commit":"` + sha + `","approval":"the technical facilitator","recorded_at":"2026-11-16T00:00:00Z"}`
	if err := os.WriteFile(p, []byte(body+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// A recorded approval in a drive lane's worktree counts for nothing under
// human; one in a person's linked worktree counts, and the reason still names
// the lane's as passed over.
func TestAnApprovalInADriveLanesWorktreeNeverCounts(t *testing.T) {
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": humanConfig})
	f.forge.head = f.head()
	f.forge.reviews = "[]"
	lane := addWorktree(t, f, driveLaneDir)
	writeApprovalAt(t, lane, f.head())
	d := gateAt(t, f)
	if d.Arm || d.ApprovalCounted || !strings.Contains(d.Why, "was passed over") || !strings.Contains(d.Why, driveLaneDir) {
		t.Fatalf("decision %+v: want nothing armed on a lane's approval, naming it", d)
	}
	if strings.Count(d.Why, "recorded approval of") != 1 {
		t.Fatalf("why = %q; want the lane's approval named once", d.Why)
	}
	mine := addWorktree(t, f, "mine")
	if _, err := RecordApproval(mine, f.head()); err != nil {
		t.Fatal(err)
	}
	d = gateAt(t, f)
	if !d.Arm || !d.ApprovalCounted {
		t.Fatalf("decision %+v: want the person's recorded approval to arm", d)
	}
	if !strings.Contains(d.Why, "was passed over") || !strings.Contains(d.Why, driveLaneDir) {
		t.Fatalf("why = %q; want it to name the lane's approval passed over though another counted", d.Why)
	}
}

// gateFrom asks the merge gate about a fixture's head from the checkout at
// root, one of the fixture repository's worktrees.
func gateFrom(t *testing.T, f *fixture, root string) MergeDecision {
	t.Helper()
	d, err := DecideMerge(MergeQuery{RepoRoot: root, Roots: f.options().Roots, Tip: f.repo.Git("rev-parse", "origin/main"), TipName: "origin/main", Default: "main",
		Repo: "o/r", PR: "7", Head: f.head(), Forge: func(args ...string) (string, error) { return f.forge.run("", args...) }})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// Asked from a checkout that is itself a drive lane's worktree, the gate counts
// no review receipt at all, even a full pair in the main checkout, and says so;
// the answer is one it could not read, so nothing is withdrawn on it. Asked
// from the main checkout, the same pair arms.
func TestTheGateAskedFromADriveLanesWorktreeCountsNoReceipt(t *testing.T) {
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	f.forge.head = f.head()
	f.reviewed(t)
	lane := addWorktree(t, f, driveLaneDir)
	d := gateFrom(t, f, lane)
	if d.Arm || !d.Unread {
		t.Fatalf("decision %+v: want nothing armed from a lane's worktree, as an unread answer", d)
	}
	if !strings.Contains(d.Why, "is a drive lane's worktree") || !strings.Contains(d.Why, "main checkout or your own worktree") {
		t.Fatalf("why = %q; want it to say the gate ran from a drive lane's worktree and where to run it", d.Why)
	}
	if vouched, err := headVouched(lane, f.head()); err != nil || vouched {
		t.Fatalf("headVouched from a lane = %v, %v; want nothing vouched", vouched, err)
	}
	if d := gateFrom(t, f, f.repo.Root()); !d.Arm {
		t.Fatalf("decision %+v: want the main checkout's pair to arm", d)
	}
}

// Under human, a recorded approval in the main checkout counts for nothing when
// the gate is asked from a drive lane's worktree, and the reason says so.
func TestTheGateAskedFromADriveLanesWorktreeCountsNoRecordedApproval(t *testing.T) {
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": humanConfig})
	f.forge.head = f.head()
	f.forge.reviews = "[]"
	writeApprovalAt(t, f.repo.Root(), f.head())
	lane := addWorktree(t, f, driveLaneDir)
	d := gateFrom(t, f, lane)
	if d.Arm || d.ApprovalCounted || !d.Unread || !strings.Contains(d.Why, "no recorded approval was read") || !strings.Contains(d.Why, "is a drive lane's worktree") {
		t.Fatalf("decision %+v: want nothing armed from a lane's worktree, saying why", d)
	}
	if d := gateFrom(t, f, f.repo.Root()); !d.Arm || !d.ApprovalCounted {
		t.Fatalf("decision %+v: want the main checkout's approval to arm", d)
	}
}

// Recording a review receipt, a delta's or an approval in a drive lane's
// worktree is refused, naming where to record it, and nothing is written.
func TestRecordingInADriveLanesWorktreeIsRefused(t *testing.T) {
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	lane := addWorktree(t, f, driveLaneDir)
	head := f.head()
	_, rerr := RecordReview(lane, head, fullPair(head))
	_, derr := RecordDeltaReview(lane, head, strings.Repeat("a", 40), fullPair(head))
	_, aerr := RecordApproval(lane, head)
	for name, err := range map[string]error{"review": rerr, "delta": derr, "approval": aerr} {
		if !errors.Is(err, ErrDriveLaneRecord) || !strings.Contains(err.Error(), "main checkout or your own worktree") {
			t.Fatalf("%s in a lane's worktree: %v; want it refused, naming where to record it", name, err)
		}
	}
	for _, rel := range []string{ReviewReceiptsRelDir, ApprovalsRelDir} {
		if _, err := os.Stat(filepath.Join(lane, filepath.FromSlash(rel), head)); !os.IsNotExist(err) {
			t.Fatalf("%s/%s was written in the lane's worktree: %v", rel, head, err)
		}
	}
}
