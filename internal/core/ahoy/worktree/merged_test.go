package worktree_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/gittest"
)

// unreachable is a forge that never answers, as gh does with no GitHub remote.
func unreachable(string, ...string) (string, error) {
	return "", errors.New("no git remotes found")
}

// forgeSaying is a forge whose one listing holds a pull request from branch,
// merged into main with head as its head commit.
func forgeSaying(branch, head string) worktree.GH {
	return func(_ string, args ...string) (string, error) {
		if len(args) < 2 || args[0] != "pr" || args[1] != "list" {
			return "", fmt.Errorf("unexpected forge call %v", args)
		}
		return fmt.Sprintf(`[{"headRefName":%q,"headRefOid":%q,"baseRefName":"main","isCrossRepository":false,"state":"MERGED"}]`, branch, head), nil
	}
}

func judged(t *testing.T, repo *gittest.Repo, forge worktree.GH, name string) worktree.Row {
	t.Helper()
	got, err := worktree.List(repo.Root(), worktree.ListOptions{Forge: forge})
	if err != nil {
		t.Fatal(err)
	}
	return rowNamed(t, got.Worktrees, name)
}

// TestMergedJudgement is decision 2's ladder with the forge unreachable,
// then a forge answer whose head is or is not the branch's tip.
func TestMergedJudgement(t *testing.T) {
	t.Run("ancestry", func(t *testing.T) {
		repo, _ := repoWithCommit(t)
		p := addToLane(t, repo, "ff")
		commitIn(t, repo, p, "f.txt", "f\n")
		repo.Git("merge", "-q", "--ff-only", "ff")
		if r := judged(t, repo, unreachable, "ff"); r.Merged != "merged" || r.JudgedBy != "ancestry" {
			t.Fatalf("a fast-forwarded branch: %+v", r)
		}
	})
	t.Run("never committed to", func(t *testing.T) {
		// The default branch moved on past the branch's creation point, so
		// the tip is its ancestor; the branch holds no commit of its own.
		repo, _ := repoWithCommit(t)
		addToLane(t, repo, "fresh")
		repo.Write("later.txt", "later\n")
		repo.Commit("later")
		if r := judged(t, repo, unreachable, "fresh"); r.Merged != "unmerged" || r.JudgedBy != "" {
			t.Fatalf("a branch never committed to is not merged: %+v", r)
		}
	})
	t.Run("fast-forwarded with no commit of its own", func(t *testing.T) {
		// The branch moved past its creation point by a fast-forward to the
		// default branch, so its tip is a later ancestor of it; it still
		// holds no work of its own.
		repo, _ := repoWithCommit(t)
		p := addToLane(t, repo, "fresh")
		repo.Write("later.txt", "later\n")
		repo.Commit("later")
		repo.Git("-C", p, "merge", "-q", "--ff-only", "main")
		if r := judged(t, repo, unreachable, "fresh"); r.Merged != "unmerged" || r.JudgedBy != "" {
			t.Fatalf("a fast-forwarded branch with no commit of its own is not merged: %+v", r)
		}
	})
	t.Run("checked out on the default branch itself", func(t *testing.T) {
		repo, _ := repoWithCommit(t)
		repo.Git("checkout", "-q", "-b", "side")
		repo.Write("later.txt", "later\n")
		repo.Commit("later")
		repo.Git("checkout", "-q", "main")
		repo.Git("merge", "-q", "--ff-only", "side")
		repo.Git("checkout", "-q", "side")
		if _, err := worktree.Add(repo.Root(), worktree.AddOptions{Name: "onmain", Branch: "main", Existing: true, Git: git}); err != nil {
			t.Fatal(err)
		}
		if r := judged(t, repo, unreachable, "onmain"); r.Merged != "unmerged" || r.JudgedBy != "" {
			t.Fatalf("the default branch is never merged into itself: %+v", r)
		}
	})
	t.Run("one real commit, later fast-forwarded", func(t *testing.T) {
		repo, _ := repoWithCommit(t)
		p := addToLane(t, repo, "real")
		repo.Write("later.txt", "later\n")
		repo.Commit("later")
		repo.Git("-C", p, "merge", "-q", "--ff-only", "main")
		commitIn(t, repo, p, "r.txt", "r\n")
		repo.Git("merge", "-q", "--ff-only", "real")
		if r := judged(t, repo, unreachable, "real"); r.Merged != "merged" || r.JudgedBy != "ancestry" {
			t.Fatalf("a branch with a commit of its own, merged by fast-forward: %+v", r)
		}
	})
	t.Run("copied with no commit of its own", func(t *testing.T) {
		// git copies main's reflog with the branch, commits and all.
		repo, _ := repoWithCommit(t)
		repo.Git("branch", "-c", "main", "copied")
		if _, err := worktree.Add(repo.Root(), worktree.AddOptions{Name: "copied", Branch: "copied", Existing: true, Git: git}); err != nil {
			t.Fatal(err)
		}
		if r := judged(t, repo, unreachable, "copied"); r.Merged != "unmerged" || r.JudgedBy != "" {
			t.Fatalf("a copied branch with no commit of its own is not merged: %+v", r)
		}
	})
	t.Run("copied, one real commit, later merged", func(t *testing.T) {
		repo, _ := repoWithCommit(t)
		repo.Git("branch", "-c", "main", "copied")
		got, err := worktree.Add(repo.Root(), worktree.AddOptions{Name: "copied", Branch: "copied", Existing: true, Git: git})
		if err != nil {
			t.Fatal(err)
		}
		commitIn(t, repo, got.Path, "c.txt", "c\n")
		repo.Git("merge", "-q", "--ff-only", "copied")
		if r := judged(t, repo, unreachable, "copied"); r.Merged != "merged" || r.JudgedBy != "ancestry" {
			t.Fatalf("a copied branch with a commit of its own, merged: %+v", r)
		}
	})
	t.Run("renamed after a real commit, later merged", func(t *testing.T) {
		// A rename carries the branch's own reflog forward.
		repo, _ := repoWithCommit(t)
		p := addToLane(t, repo, "before")
		commitIn(t, repo, p, "r.txt", "r\n")
		repo.Git("branch", "-m", "before", "after")
		repo.Git("merge", "-q", "--ff-only", "after")
		if r := judged(t, repo, unreachable, "before"); r.Branch != "after" || r.Merged != "merged" || r.JudgedBy != "ancestry" {
			t.Fatalf("a renamed branch keeps its own commits: %+v", r)
		}
	})
	t.Run("rebase merge", func(t *testing.T) {
		repo, _ := repoWithCommit(t)
		p := addToLane(t, repo, "rb")
		c := commitIn(t, repo, p, "r.txt", "r\n")
		repo.Write("other.txt", "other\n")
		repo.Commit("other")
		repo.Git("cherry-pick", c)
		if r := judged(t, repo, unreachable, "rb"); r.Merged != "merged" || r.JudgedBy != "content" {
			t.Fatalf("a rebase-merged branch: %+v", r)
		}
	})
	t.Run("squash merge", func(t *testing.T) {
		repo, _ := repoWithCommit(t)
		p := addToLane(t, repo, "sq")
		commitIn(t, repo, p, "s1.txt", "one\n")
		commitIn(t, repo, p, "s2.txt", "two\n")
		repo.Write("other.txt", "other\n")
		repo.Commit("other")
		repo.Git("merge", "-q", "--squash", "sq")
		repo.Git("commit", "-q", "-m", "squashed")
		got, err := worktree.List(repo.Root(), worktree.ListOptions{Forge: unreachable})
		if err != nil {
			t.Fatal(err)
		}
		if r := rowNamed(t, got.Worktrees, "sq"); r.Merged != "merged" || r.JudgedBy != "content" {
			t.Fatalf("a squash-merged branch: %+v", r)
		}
		if !strings.Contains(got.ForgeNote, "no forge answered") {
			t.Fatalf("the run says once that the forge did not answer: %q", got.ForgeNote)
		}
	})
	t.Run("unmerged work", func(t *testing.T) {
		repo, _ := repoWithCommit(t)
		p := addToLane(t, repo, "wip")
		commitIn(t, repo, p, "w.txt", "w\n")
		if r := judged(t, repo, unreachable, "wip"); r.Merged != "unmerged" {
			t.Fatalf("a branch with work of its own: %+v", r)
		}
	})
	t.Run("forge head is the tip", func(t *testing.T) {
		repo, _ := repoWithCommit(t)
		p := addToLane(t, repo, "pr")
		tip := commitIn(t, repo, p, "p.txt", "p\n")
		got, err := worktree.List(repo.Root(), worktree.ListOptions{Forge: forgeSaying("pr", tip)})
		if err != nil {
			t.Fatal(err)
		}
		if r := rowNamed(t, got.Worktrees, "pr"); r.Merged != "merged" || r.JudgedBy != "forge" {
			t.Fatalf("a pull request merged at the tip: %+v", r)
		}
		if got.ForgeNote != "" {
			t.Fatalf("a forge that answered needs no note: %q", got.ForgeNote)
		}
	})
	t.Run("forge head is not the tip", func(t *testing.T) {
		repo, _ := repoWithCommit(t)
		p := addToLane(t, repo, "pr")
		old := commitIn(t, repo, p, "p.txt", "p\n")
		commitIn(t, repo, p, "q.txt", "more after the merge\n")
		if r := judged(t, repo, forgeSaying("pr", old), "pr"); r.Merged != "unmerged" {
			t.Fatalf("a branch that gained commits after its merge: %+v", r)
		}
	})
}
