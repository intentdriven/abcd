package loop

// remoteid_limits_test.go holds the bound on what a drive holds of the
// checkout's hooks directory, and its refusals to start and to remove a lane
// worktree it did not add as judged, each as commands/implement.md states it.
// Each test here but the bound's pins what already held and passes without a
// production change.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
)

// TestAHooksTreePastTheCapIsMarkedAndItsFirstEntriesHeld: past
// maxHookEntries entries, at every depth, the walk stops and says so, with
// the first maxHookEntries entries in walk order and a marker at the
// directory's path recorded; the drive refuses to start on one
// (TestADriveDoesNotStartPastTheHooksBound). Watched failing on the default
// branch, where only the top level was counted.
func TestAHooksTreePastTheCapIsMarkedAndItsFirstEntriesHeld(t *testing.T) {
	dir := t.TempDir()
	mkdirs(t, filepath.Join(dir, "sub"))
	for i := 0; i < maxHookEntries; i++ {
		writeMode(t, filepath.Join(dir, "sub", fmt.Sprintf("h%05d", i)), "#!/bin/sh\n", 0o755)
	}
	m := map[string]string{}
	if _, over := hashDir(m, "salt", dir); !over {
		t.Fatal("the subdirectory and its entries are past the bound")
	}
	if _, ok := m[dir]; !ok || len(m) != maxHookEntries+1 {
		t.Fatalf("the marker and the first %d entries are recorded: %d, marker %v", maxHookEntries, len(m), ok)
	}
	if _, ok := m[filepath.Join(dir, "sub", fmt.Sprintf("h%05d", maxHookEntries-1))]; ok {
		t.Fatal("the entry past the bound is not recorded")
	}
	// The marker carries the tree's count, so one more entry past the bound
	// changes the record (watched failing in fix round 1, where the marker
	// was the same past any count).
	writeMode(t, filepath.Join(dir, "sub", "one-more"), "#!/bin/sh\n", 0o755)
	more := map[string]string{}
	hashDir(more, "salt", dir)
	if more[dir] == m[dir] {
		t.Fatal("one more entry past the bound changes the marker")
	}
	removeAll(t, filepath.Join(dir, "sub", "one-more"))
	under := map[string]string{}
	removeAll(t, filepath.Join(dir, "sub", fmt.Sprintf("h%05d", maxHookEntries-1)))
	if _, over := hashDir(under, "salt", dir); over || len(under) != maxHookEntries {
		t.Fatalf("at the bound the tree is held whole: %d", len(under))
	}
}

// TestADriveWhoseSettingsCannotBeListedDoesNotStart: settings git cannot
// list (here a bad line in the repository's own settings) stop the drive
// before its first move, saying it cannot start, with nothing held. Watched
// failing with readRemoteIdentity's error ignored.
func TestADriveWhoseSettingsCannotBeListedDoesNotStart(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	cfg := filepath.Join(repo.Root(), ".git", "config")
	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	writeMode(t, cfg, string(data)+"[core\n", 0o644)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tm := newTerminal(repo.Root(), TerminalOptions{Stages: DefaultStages(), Say: func(string) {}})
	_, err = tm.run(ctx)
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "the drive cannot start") {
		t.Fatalf("the drive refuses to start: %+v", r)
	}
	if tm.o.Options.remote != nil {
		t.Fatal("nothing is held")
	}
}

// TestALaneWorktreeNotAsAddedIsLeftWhenTheBranchMoved: when the branch moved
// during the add and what stands at the lane's path is not the worktree the
// add made as git lists it (a symlink put there, or the worktree switched to
// another branch), the stage removes nothing and refuses, naming the failed
// removal; what stands there is left. The other-branch case was watched
// failing with RemoveAdded's own checks dropped; the symlink case was not, as
// git's own removal refuses that path here too, and holds the stage's outcome
// only. RemoveAdded's real-directory check on a symlink is held directly by
// TestRemoveAddedRemovesOnlyTheWorktreeAddMade in internal/core/ahoy/worktree.
func TestALaneWorktreeNotAsAddedIsLeftWhenTheBranchMoved(t *testing.T) {
	for name, after := range map[string]func(t *testing.T, added worktree.Added){
		"a symlink at the lane's path": func(t *testing.T, added worktree.Added) {
			if err := os.Rename(added.Path, added.Path+"-real"); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(added.Path+"-real", added.Path); err != nil {
				t.Fatal(err)
			}
		},
		"the worktree on another branch": func(t *testing.T, added worktree.Added) {
			gitIn(t, added.Path, "switch", "-q", "-c", "elsewhere")
		},
	} {
		t.Run(name, func(t *testing.T) {
			repo, _, tip := hookedOrigin(t)
			repo.Write("notes.txt", "moved\n")
			repo.Commit("moved")
			moved := repo.Git("rev-parse", "HEAD")
			repo.Git("reset", "-q", "--hard", tip)
			start, err := Start(repo.Root(), "itd-10", Options{noReserve: true})
			if err != nil {
				t.Fatal(err)
			}
			lw, err := laneWorktree(repo.Root(), start.RunID, "lane-1")
			if err != nil {
				t.Fatal(err)
			}
			repo.Git("branch", lw.Branch, tip)
			was := addWorktree
			t.Cleanup(func() { addWorktree = was })
			addWorktree = func(root string, o worktree.AddOptions) (worktree.Added, error) {
				repo.Git("update-ref", "refs/heads/"+lw.Branch, moved)
				added, err := was(root, o)
				if err == nil {
					after(t, added)
				}
				return added, err
			}
			_, err = advance(repo.Root(), start.RunID, DefaultStages(), Options{})
			worktreeRefused(t, err, "could not be removed")
			if _, err := os.Lstat(lw.Path); err != nil {
				t.Fatalf("what stands at the lane's path is left: %v", err)
			}
		})
	}
}
