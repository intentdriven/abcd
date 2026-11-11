package loop

// remoteid_limits_test.go holds the limits of what a drive holds of the
// checkout's hooks directory, and its refusals to start and to remove a lane
// worktree it did not add as judged, each as commands/implement.md states it.
// Each test here pins what already held and passes without a production
// change.

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
)

// TestAHooksDirectoryPastTheCapIsHeldByItsCountAndFirstEntries: past
// maxHookEntries entries the hooks directory is recorded by its count and its
// first maxHookEntries entries in name order. A change to an entry past the
// cap is not seen; one to the first entry is, and so is one more entry, by the
// count. Watched failing with the cap's count left out of hashDir.
func TestAHooksDirectoryPastTheCapIsHeldByItsCountAndFirstEntries(t *testing.T) {
	dir := t.TempDir()
	entry := func(i int) string { return filepath.Join(dir, fmt.Sprintf("h%05d", i)) }
	for i := 0; i <= maxHookEntries; i++ {
		writeMode(t, entry(i), "#!/bin/sh\n", 0o755)
	}
	read := func() map[string]string {
		m := map[string]string{}
		hashDir(m, "salt", dir)
		return m
	}
	was := read()
	if _, ok := was[dir]; !ok || len(was) != maxHookEntries+1 {
		t.Fatalf("the count and the first %d entries are recorded: %d, count %v", maxHookEntries, len(was), ok)
	}
	if _, ok := was[entry(maxHookEntries)]; ok {
		t.Fatal("the entry past the cap is not recorded")
	}
	writeMode(t, entry(maxHookEntries), "#!/bin/sh\ntouch x\n", 0o755)
	if !maps.Equal(read(), was) {
		t.Fatal("a change to the entry past the cap is not seen, the limit the docs state")
	}
	writeMode(t, entry(0), "#!/bin/sh\ntouch x\n", 0o755)
	changed := read()
	if maps.Equal(changed, was) {
		t.Fatal("a change to the first entry is seen")
	}
	writeMode(t, entry(maxHookEntries+1), "#!/bin/sh\n", 0o755)
	if now := read(); now[dir] == changed[dir] {
		t.Fatal("one more entry past the cap changes the count")
	}
}

// TestASymlinkedHookIsHeldByItsTargetTextOnly: a hook that is a symlink is
// held by its target's text. Its target's bytes changed elsewhere are not
// seen, the limit the docs state, and the lane opens; the link pointed at
// another file refuses it naming the hook. Watched failing with a symlink
// recorded by its kind alone.
func TestASymlinkedHookIsHeldByItsTargetTextOnly(t *testing.T) {
	repo, _, _ := hookedOrigin(t)
	outside := t.TempDir()
	target := filepath.Join(outside, "hook.sh")
	writeMode(t, target, "#!/bin/sh\nexit 0\n", 0o755)
	link := filepath.Join(repo.Root(), ".githooks", "post-checkout")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	id, err := readRemoteIdentity(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	c := Context{RepoRoot: repo.Root(), remote: id}
	writeMode(t, target, "#!/bin/sh\ntouch x\n", 0o755)
	if err := checkoutHooksUnchanged(c, "lane-2"); err != nil {
		t.Fatalf("the target's bytes are not held: %v", err)
	}
	other := filepath.Join(outside, "other.sh")
	writeMode(t, other, "#!/bin/sh\nexit 0\n", 0o755)
	removeAll(t, link)
	if err := os.Symlink(other, link); err != nil {
		t.Fatal(err)
	}
	worktreeRefused(t, checkoutHooksUnchanged(c, "lane-2"), "post-checkout")
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
