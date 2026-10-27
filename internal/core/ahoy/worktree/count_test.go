package worktree_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestCountReclaimsOnlyWhatPruneWould is the board's worktrees line
// (spc-2609301811532881, criterion 17) held to prune's own judgement with no
// forge: a fresh worktree with no commit of its own is held and not
// reclaimable, though git calls its branch merged by ancestry; a squash-merged
// clean one is reclaimable; a merged one that is dirty or locked is not; and
// one outside the lane is not held. The count is the dry run's.
func TestCountReclaimsOnlyWhatPruneWould(t *testing.T) {
	repo, home := pruneFixture(t)
	addToLane(t, repo, "fresh")
	sq := addToLane(t, repo, "sq")
	commitIn(t, repo, sq, "s1.txt", "one\n")
	commitIn(t, repo, sq, "s2.txt", "two\n")
	repo.Write("other.txt", "other\n")
	repo.Commit("other")
	repo.Git("merge", "-q", "--squash", "sq")
	repo.Git("commit", "-q", "-m", "squashed")
	dirty := mergedWorktree(t, repo, "dirty")
	if err := os.WriteFile(filepath.Join(dirty, "wip.txt"), []byte("wip\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	locked := mergedWorktree(t, repo, "locked")
	repo.Git("worktree", "lock", "--reason", "in use", locked)
	outside := filepath.Join(home, "outside")
	repo.Git("worktree", "add", "-q", "-b", "outside", outside, "main")

	listed, err := gitutil.ListWorktrees(repo.Root(), gitutil.MaxWorktreeListing)
	if err != nil {
		t.Fatal(err)
	}
	got, err := worktree.Count(repo.Root(), listed, worktree.CountOptions{Held: noneHeld})
	if err != nil {
		t.Fatal(err)
	}
	if got.Held != 4 || got.Reclaimable != 1 {
		t.Fatalf("Count = %+v, want 4 held (fresh, sq, dirty, locked) and 1 reclaimable (sq)", got)
	}
	dry := prune(t, repo, worktree.PruneOptions{DryRun: true})
	if dry.Reclaimed() != got.Reclaimable {
		t.Fatalf("Count says %d reclaimable, prune's dry run %d: %+v", got.Reclaimable, dry.Reclaimed(), dry.Rows)
	}
}
