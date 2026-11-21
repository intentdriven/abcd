package loop

import (
	"os"
	"strings"
	"testing"
)

// TestTheLandingPlacesItsPushedHeadOnTheDefaultBranchAsStored: the landing's own
// step cleans up a lane only when the default branch stores its pushed head; a
// replace ref giving the default branch's tip that head as a parent leaves the
// lane waiting on its merge, its worktree and branch kept.
func TestTheLandingPlacesItsPushedHeadOnTheDefaultBranchAsStored(t *testing.T) {
	f := drainLandFixture(t)
	first := f.firstMove(t)
	if first.Start == nil {
		t.Fatalf("the drain opens a lane: %+v", first)
	}
	f.runID, f.issue = first.Start.RunID, secondIssue
	l := f.armIssueLaneOnce(t)
	tip := f.repo.Git("rev-parse", "refs/remotes/origin/main")
	parents := strings.Fields(f.repo.Git("rev-list", "--parents", "-n", "1", tip))[1:]
	standParents(t, f.repo, false, tip, append(parents, l.Landing.Pushed)...)
	if !presentedHolds(f.repo, l.Landing.Pushed, tip) {
		t.Fatal("git presents the pushed head on the default branch")
	}
	if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err != nil {
		if r, ok := AsRefusal(err); !ok || strings.Contains(r.Reason, "landed") {
			t.Fatalf("the landing waits on its merge: %v", err)
		}
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Complete() || st.Lanes[0].Stage != StageLand || st.Lanes[0].Landing.Merged != "" {
		t.Fatalf("the lane stays at its landing: %+v", st.Lanes[0])
	}
	if _, err := os.Stat(l.Worktree); err != nil {
		t.Fatalf("the lane keeps its worktree: %v", err)
	}
}
