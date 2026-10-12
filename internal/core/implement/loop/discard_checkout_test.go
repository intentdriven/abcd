package loop

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAHeldDiscardLeavesABranchCheckedOutElsewhere: `implement step --discard`
// deleted a held lane's branch with `git update-ref -d`, which does not ask, as
// `git branch -d` does, whether a checkout stands on it, so a second worktree a
// person added on the lane's branch was left with a HEAD naming nothing
// (iss-2610120137339391). The discard refuses, naming that checkout, before it
// removes anything.
func TestAHeldDiscardLeavesABranchCheckedOutElsewhere(t *testing.T) {
	f := armedSibling(t, false)
	f.handedBack(t)
	f.step(t)
	l2 := f.lane(t, "lane-2")
	if l2.Stage != StageHeld || l2.Branch == "" || l2.Worktree == "" {
		t.Fatalf("premise: lane 2 is held with its worktree and branch: %+v", l2)
	}
	tip := strings.TrimSpace(f.repo.Git("rev-parse", "refs/heads/"+l2.Branch))
	other := filepath.Join(t.TempDir(), "look")
	f.repo.Git("worktree", "add", "-q", "-f", other, l2.Branch)

	before := stateBytes(t, f.repo.Root(), f.runID)
	_, err := Discard(f.repo.Root(), f.runID, "lane-2", f.opts())
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "checked out") || !strings.Contains(r.Reason, filepath.Base(other)) {
		t.Fatalf("the discard refuses, naming the checkout that stands on the branch: %+v", r)
	}
	if got := strings.TrimSpace(f.repo.Git("rev-parse", "--verify", "--quiet", "refs/heads/"+l2.Branch)); got != tip {
		t.Fatalf("the branch checked out elsewhere is not deleted: %q, want %q", got, tip)
	}
	if head := strings.TrimSpace(f.repo.Git("-C", other, "rev-parse", "--verify", "HEAD")); head != tip {
		t.Fatalf("that checkout's HEAD still names the lane's tip: %q", head)
	}
	if _, err := os.Stat(l2.Worktree); err != nil {
		t.Fatalf("a refused discard removes nothing, the lane's worktree included: %v", err)
	}
	if !bytes.Equal(before, stateBytes(t, f.repo.Root(), f.runID)) {
		t.Fatal("a refused discard leaves the lane held")
	}
	if n := strings.Count(f.ghLog(t), "pr close"); n != 0 {
		t.Fatalf("a refused discard closes no pull request: %d", n)
	}
}

// TestAHandBackDiscardLeavesABranchCheckedOutElsewhere: a receipt carrying a
// hand-back discards the lane's work through the same `git update-ref -d`, so
// it refuses the same way, before the lane's worktree is removed, and the
// receipt is not taken (iss-2610120137339391).
func TestAHandBackDiscardLeavesABranchCheckedOutElsewhere(t *testing.T) {
	repo, runID, l, dir := issueAwaiting(t)
	sha := laneCommit(t, repo, l, "partial.txt")
	rc := goodReceipt(t, runID, l, dir, sha)
	rc.HandBack = &LaneHandBack{Kind: HandBackUserVisible, Reason: "the fix changes what the status board shows"}
	path := writeReceipt(t, dir, rc)
	other := filepath.Join(t.TempDir(), "look")
	repo.Git("worktree", "add", "-q", "-f", other, l.Branch)

	before := stateBytes(t, repo.Root(), runID)
	_, err := Receipt(repo.Root(), runID, path, DefaultStages(), Options{})
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "checked out") || !strings.Contains(r.Reason, filepath.Base(other)) {
		t.Fatalf("the hand-back refuses, naming the checkout that stands on the branch: %+v", r)
	}
	if got := strings.TrimSpace(repo.Git("rev-parse", "--verify", "--quiet", "refs/heads/"+l.Branch)); got != sha {
		t.Fatalf("the branch checked out elsewhere is not deleted: %q, want %q", got, sha)
	}
	if head := strings.TrimSpace(repo.Git("-C", other, "rev-parse", "--verify", "HEAD")); head != sha {
		t.Fatalf("that checkout's HEAD still names the lane's tip: %q", head)
	}
	if _, err := os.Stat(l.Worktree); err != nil {
		t.Fatalf("a refused hand-back removes nothing, the lane's worktree included: %v", err)
	}
	if !bytes.Equal(before, stateBytes(t, repo.Root(), runID)) {
		t.Fatal("a refused hand-back leaves the lane where it was")
	}
}
