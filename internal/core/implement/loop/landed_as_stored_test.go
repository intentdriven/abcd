package loop

import (
	"context"
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

// TestALandedLaneIsRetiredOnlyAtATipTheMergedCommitStores: the retiring of a
// landed lane removes its branch only when the merged commit stores the
// branch's tip; a replace ref or a graft giving the merged commit that tip as
// a parent leaves the lane refused, its branch kept.
func TestALandedLaneIsRetiredOnlyAtATipTheMergedCommitStores(t *testing.T) {
	for name, graftFile := range map[string]bool{"a replace ref": false, "a graft file": true} {
		t.Run(name, func(t *testing.T) {
			repo, base, tip, merged := threeCommits(t)
			const branch = "build/run-1-lane-1"
			repo.Git("branch", branch, tip)
			standParents(t, repo, graftFile, merged, base, tip)
			if !presentedHolds(repo, tip, merged) {
				t.Fatal("git presents the lane's tip on the merged commit")
			}
			lane := Lane{ID: "lane-1", Branch: branch, Landing: &Landing{Pushed: base}}
			err := retireLanded(context.Background(), Context{RepoRoot: repo.Root()}, lane, merged, string(StageLand), "run it again")
			if r, ok := AsRefusal(err); !ok || !strings.Contains(r.Reason, "past what landed") {
				t.Fatalf("the lane is refused for carrying work that did not land: %v", err)
			}
			if got := repo.Git("rev-parse", "refs/heads/"+branch); got != tip {
				t.Fatalf("the lane's branch is kept at %s, got %s", tip, got)
			}
		})
	}
}

// TestALandedLaneIsRetiredOnlyAtTheTipItsBranchStores: the retiring of a
// landed lane reads the branch's tip as stored, so a refs/replace entry
// standing a tree in for a tip past what landed leaves the lane refused, its
// branch kept.
func TestALandedLaneIsRetiredOnlyAtTheTipItsBranchStores(t *testing.T) {
	repo, base, merged, _ := threeCommits(t)
	tip := repo.Git("commit-tree", base+"^{tree}", "-p", merged, "-m", "past the push")
	const branch = "build/run-1-lane-1"
	repo.Git("branch", branch, tip)
	repo.Git("update-ref", "refs/replace/"+tip, repo.Git("rev-parse", base+"^{tree}"))
	lane := Lane{ID: "lane-1", Branch: branch, Landing: &Landing{Pushed: merged}}
	err := retireLanded(context.Background(), Context{RepoRoot: repo.Root()}, lane, merged, string(StageLand), "run it again")
	if r, ok := AsRefusal(err); !ok || !strings.Contains(r.Reason, "past what landed") {
		t.Fatalf("the lane is refused for carrying work that did not land: %v", err)
	}
	if got := repo.Git("rev-parse", "refs/heads/"+branch); got != tip {
		t.Fatalf("the lane's branch is kept at %s, got %s", tip, got)
	}
}

// TestAParkedLaneIsPlacedOnTheMergedCommitAsStored: a resumed build sees a
// parked lane's pushed head on the fetched default branch only when that
// commit stores it.
func TestAParkedLaneIsPlacedOnTheMergedCommitAsStored(t *testing.T) {
	for name, graftFile := range map[string]bool{"a replace ref": false, "a graft file": true} {
		t.Run(name, func(t *testing.T) {
			repo, base, pushed, merged := threeCommits(t)
			standParents(t, repo, graftFile, merged, base, pushed)
			if !presentedHolds(repo, pushed, merged) {
				t.Fatal("git presents the pushed head on the merged commit")
			}
			if on, err := onCommit(context.Background(), repo.Root(), pushed, merged); err != nil || on {
				t.Fatalf("onCommit = %v, %v; want no from the stored parents", on, err)
			}
			if on, err := onCommit(context.Background(), repo.Root(), base, merged); err != nil || !on {
				t.Fatalf("onCommit(base) = %v, %v; want yes", on, err)
			}
		})
	}
}
