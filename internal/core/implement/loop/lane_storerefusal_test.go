package loop

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
)

// TestALaneWhoseHookAndFinishBothFailNamesTheFinishingFailure
// (iss-2610231125096250): when git makes the lane's worktree and exits with
// the checkout hook's failure, and the store's Add then cannot lay the
// worktree's local tier (a file stands where it goes), the worktree stage's
// refusal names both failures, the hook's exit status and the tier, with the
// store's remedy. It is not reported as git's failure alone, since a rerun
// adopts the standing worktree and never reports the tier again.
//
// The hook's failure is stood in for by the runner: it runs the loop's own
// git add, then plants the file and returns a real exit status 3.
func TestALaneWhoseHookAndFinishBothFailNamesTheFinishingFailure(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	start, err := Start(repo.Root(), "itd-10", Options{noReserve: true})
	if err != nil {
		t.Fatal(err)
	}
	lw, err := laneWorktree(repo.Root(), start.RunID, "lane-1")
	if err != nil {
		t.Fatal(err)
	}
	was := addWorktree
	t.Cleanup(func() { addWorktree = was })
	addWorktree = func(root string, o worktree.AddOptions) (worktree.Added, error) {
		run := o.Git
		o.Git = func(dir string, args ...string) (string, error) {
			out, err := run(dir, args...)
			if err != nil || len(args) < 2 || args[0] != "worktree" || args[1] != "add" {
				return out, err
			}
			if err := os.MkdirAll(filepath.Join(lw.Path, ".abcd"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(lw.Path, ".abcd", ".work.local"), []byte("not a directory\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			hookErr := exec.Command("sh", "-c", "exit 3").Run()
			if hookErr == nil {
				t.Fatal("the stand-in hook failure did not fail")
			}
			return out, hookErr
		}
		return was(root, o)
	}
	_, err = advance(repo.Root(), start.RunID, DefaultStages(), Options{})
	r := worktreeRefused(t, err, "exit status 3")
	if !strings.Contains(r.Reason, ".abcd/.work.local") {
		t.Fatalf("the refusal names the local tier the store could not lay: %+v", r)
	}
	if strings.Contains(r.Reason, "git could not add the lane's worktree") {
		t.Fatalf("the refusal is not reported as git's failure alone: %+v", r)
	}
	if !strings.Contains(r.Remedy, "move aside what the branch carries there") {
		t.Fatalf("the refusal carries the store's remedy for the tier: %+v", r)
	}
}
