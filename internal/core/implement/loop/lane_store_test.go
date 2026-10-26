package loop

import (
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
)

// TestTheLaneGoesThroughTheStorePackage is criterion 5 of
// itd-2609091014076309: the build loop's lane is made by the worktree store's
// own primitive, not a copy of it. The loop's call to worktree.Add is watched:
// the worktree stage makes the lane through it exactly once, with the lane's
// name and branch, at the path the store derives for the repository; the
// restarted stage adopts what it made (worktree.Adopt) and makes nothing.
func TestTheLaneGoesThroughTheStorePackage(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	var calls []worktree.AddOptions
	was := addWorktree
	addWorktree = func(root string, o worktree.AddOptions) (worktree.Added, error) {
		calls = append(calls, o)
		return was(root, o)
	}
	t.Cleanup(func() { addWorktree = was })

	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := advance(repo.Root(), start.RunID, DefaultStages(), Options{}); err != nil {
		t.Fatal(err)
	}
	name := start.RunID + "-lane-1"
	if len(calls) != 1 {
		t.Fatalf("the lane is made through worktree.Add once, got %d calls", len(calls))
	}
	if c := calls[0]; c.Name != name || c.Branch != BranchPrefix+name || c.Existing || c.Git == nil || c.Base == "" {
		t.Fatalf("worktree.Add is asked for the lane: %+v", c)
	}
	lane, err := worktree.LaneFor(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Lanes[0].Worktree != lane.Path(name) {
		t.Fatalf("the lane lives where the store says: %q, want %q", st.Lanes[0].Worktree, lane.Path(name))
	}

	again := st.Lanes[0]
	c := Context{RepoRoot: repo.Root(), RunDir: runRel(st.RunID), State: st}
	if _, err := worktreeStage(c, &again); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 {
		t.Fatalf("an adopted lane is not made again: %d calls", len(calls))
	}
}
