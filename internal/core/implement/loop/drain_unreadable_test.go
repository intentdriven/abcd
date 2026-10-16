package loop

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// One unreadable sibling worktree (spc-2610161044461448 step 9;
// iss-2610161015397586, dcn-2610161040378713): the drain refuses the move
// once, before any reservation or judgement, naming the worktree and its
// repair, rather than reading the worktree as a holder of every issue and
// passing each over as contended. A worktree another tool has locked, and
// that still reads, is no such worktree.

// TestAnUnreadableWorktreeRefusesTheDrainMoveOnce: a sibling worktree whose
// git file points nowhere refuses the move naming it, with its repair; the
// refusal is not contention, asks no judgement, reserves nothing, starts no
// run and writes no drain state; once the worktree is repaired the same
// drain judges its first issue.
func TestAnUnreadableWorktreeRefusesTheDrainMoveOnce(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	broken := filepath.Join(t.TempDir(), "wt-broken")
	repo.Git("worktree", "add", "-q", "-b", "lane-broken", broken)
	if err := os.WriteFile(filepath.Join(broken, ".git"), []byte("gitdir: "+filepath.Join(t.TempDir(), "nowhere")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for move := 1; move <= 2; move++ {
		res, err := Drain(repo.Root(), Options{}, DrainOptions{})
		r := mustRefusal(t, err)
		if r.Stage != StageDrain || r.Check != CheckPeers || r.Contention {
			t.Fatalf("move %d: want the drain's own peers refusal, not contention: %+v", move, r)
		}
		if !strings.Contains(r.Reason, "the worktree at wt-broken (branch lane-broken) could not be read") {
			t.Errorf("move %d: the refusal names the worktree: %q", move, r.Reason)
		}
		if strings.Count(r.Reason, "wt-broken (branch") != 1 || strings.Contains(r.Reason, secondIssue) || strings.Contains(r.Reason, eligibleIssue) {
			t.Errorf("move %d: the worktree is named once, and no issue is: %q", move, r.Reason)
		}
		for _, want := range []string{"git worktree repair wt-broken", "git worktree remove --force wt-broken"} {
			if !strings.Contains(r.Remedy, want) {
				t.Errorf("move %d: the remedy lacks %q: %q", move, want, r.Remedy)
			}
		}
		if res.Judging != nil || len(res.Passed) != 0 {
			t.Fatalf("move %d: no judgement is asked and no issue is passed over: %+v", move, res)
		}
	}
	if _, err := os.Stat(filepath.Join(repo.Root(), filepath.FromSlash(DrainStateRel))); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a refused move writes no drain state: %v", err)
	}
	for _, id := range []string{secondIssue, eligibleIssue} {
		if sha := gitIn(t, bare, "for-each-ref", "--format=%(objectname)", claim.RefPrefix+id); sha != "" {
			t.Fatalf("a refused move reserves nothing: %s holds %s", id, sha)
		}
	}
	if n := runCount(t, repo); n != 0 {
		t.Fatalf("a refused move starts no run: %d", n)
	}

	// The repair the remedy names is the whole fix. git rewrites the broken
	// git file and still exits 1 for having found it broken, so the repair is
	// proved by the worktree answering again, not by its exit status.
	_ = exec.Command("git", "-C", repo.Root(), "worktree", "repair", broken).Run()
	gitIn(t, broken, "rev-parse", "--git-common-dir")
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatalf("a repaired worktree lets the move through: %v", err)
	}
	if res.Judging == nil || res.Judging.Issue != secondIssue {
		t.Fatalf("the repaired drain judges its first issue: %+v", res)
	}
}

// TestALockedReadableWorktreeDoesNotRefuseTheDrain: git's lock on a worktree,
// as another tool takes it, keeps prune and remove away from it and says
// nothing of whether it reads; a locked worktree that reads is an ordinary
// peer, and the drain's move goes on to its judgement.
func TestALockedReadableWorktreeDoesNotRefuseTheDrain(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	locked := filepath.Join(t.TempDir(), "wt-locked")
	repo.Git("worktree", "add", "-q", "-b", "lane-locked", locked)
	repo.Git("worktree", "lock", "--reason", `{"owner":"another-tool"}`, locked)

	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatalf("a locked worktree that reads refuses nothing: %v", err)
	}
	if res.Judging == nil || res.Judging.Issue != secondIssue {
		t.Fatalf("the drain judges its first issue past a locked worktree: %+v", res)
	}
}
