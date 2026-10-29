package worktree_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	t.Run("copied, its copy entry since expired", func(t *testing.T) {
		// The copy and the reset expire as unreachable; main's inherited
		// commit entry stays, with nothing left to say where the branch began.
		repo, _ := repoWithCommit(t)
		repo.Write("m2.txt", "m2\n")
		repo.Commit("m2")
		repo.Git("branch", "-c", "main", "cp")
		repo.Git("branch", "-f", "cp", "main~1")
		repo.Git("reflog", "expire", "--expire-unreachable=now", "--all")
		if _, err := worktree.Add(repo.Root(), worktree.AddOptions{Name: "cp", Branch: "cp", Existing: true, Git: git}); err != nil {
			t.Fatal(err)
		}
		if r := judged(t, repo, unreachable, "cp"); r.Merged != "unmerged" || r.JudgedBy != "" {
			t.Fatalf("a reflog that no longer records the branch's creation proves no commit of its own: %+v", r)
		}
	})
	t.Run("reflog gone after a real commit, later merged", func(t *testing.T) {
		repo, _ := repoWithCommit(t)
		p := addToLane(t, repo, "gone")
		commitIn(t, repo, p, "g.txt", "g\n")
		repo.Git("merge", "-q", "--ff-only", "gone")
		repo.Git("reflog", "delete", "--rewrite", "refs/heads/gone@{0}", "refs/heads/gone@{1}")
		if r := judged(t, repo, unreachable, "gone"); r.Merged != "unmerged" {
			t.Fatalf("a branch with no reflog to prove a commit of its own is judged by content alone: %+v", r)
		}
	})
	for name, create := range map[string][]string{
		"git branch":      {"branch", "made"},
		"git checkout -b": {"checkout", "-q", "-b", "made"},
		"git switch -c":   {"switch", "-q", "-c", "made"},
	} {
		t.Run("created by "+name+", a real commit, later merged", func(t *testing.T) {
			repo, _ := repoWithCommit(t)
			repo.Git(create...)
			repo.Git("checkout", "-q", "main")
			got, err := worktree.Add(repo.Root(), worktree.AddOptions{Name: "made", Branch: "made", Existing: true, Git: git})
			if err != nil {
				t.Fatal(err)
			}
			commitIn(t, repo, got.Path, "m.txt", "m\n")
			repo.Git("merge", "-q", "--ff-only", "made")
			if r := judged(t, repo, unreachable, "made"); r.Merged != "merged" || r.JudgedBy != "ancestry" {
				t.Fatalf("%s: %+v", name, r)
			}
		})
	}
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

// startRecord is where add keeps the start of the worktree name's branch.
func startRecord(t *testing.T, repo *gittest.Repo, name string) string {
	t.Helper()
	l, err := worktree.LaneFor(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, ok := worktree.StartForTest(l, name); !ok {
		t.Fatalf("the store recorded no start for %s", name)
	}
	return filepath.Join(l.Dir, ".starts", name)
}

// copiedAndExpired is iss-2610231011465325's branch: a copy of a branch that
// was itself created and committed to, merged, then force-reset and left
// until its unreachable reflog expired. Its reflog keeps the source's
// creation entry and a commit entry its tip still holds, though the copy made
// no commit of its own.
func copiedAndExpired(t *testing.T, repo *gittest.Repo) {
	t.Helper()
	repo.Git("branch", "src")
	repo.Git("checkout", "-q", "src")
	repo.Write("s1.txt", "s1\n")
	repo.Commit("s1")
	repo.Write("s2.txt", "s2\n")
	repo.Commit("s2")
	repo.Git("checkout", "-q", "main")
	repo.Git("merge", "-q", "--ff-only", "src")
	repo.Write("m2.txt", "m2\n")
	repo.Commit("m2")
	repo.Git("branch", "-c", "src", "cp")
	repo.Git("branch", "-f", "cp", "src~1")
	repo.Git("reflog", "expire", "--expire-unreachable=now", "--all")
}

// TestMergedJudgementReadsAStoreBranchFromItsStart: a worktree the store
// added has its branch's own commits read from the start add recorded, so a
// reflog carrying another branch's history proves nothing for it, while one
// real commit merged any way still reads as merged.
func TestMergedJudgementReadsAStoreBranchFromItsStart(t *testing.T) {
	t.Run("a copy of a created branch, force-reset, its reflog expired", func(t *testing.T) {
		repo, _ := repoWithCommit(t)
		copiedAndExpired(t, repo)
		if _, err := worktree.Add(repo.Root(), worktree.AddOptions{Name: "cp", Branch: "cp", Existing: true, Git: git}); err != nil {
			t.Fatal(err)
		}
		startRecord(t, repo, "cp")
		if r := judged(t, repo, unreachable, "cp"); r.Merged != "unmerged" || r.JudgedBy != "" {
			t.Fatalf("a branch whose only commits came before the store added it is not merged: %+v", r)
		}
	})
	t.Run("one real commit, merged by fast-forward", func(t *testing.T) {
		repo, _ := repoWithCommit(t)
		p := addToLane(t, repo, "ff")
		startRecord(t, repo, "ff")
		commitIn(t, repo, p, "f.txt", "f\n")
		repo.Git("merge", "-q", "--ff-only", "ff")
		if r := judged(t, repo, unreachable, "ff"); r.Merged != "merged" || r.JudgedBy != "ancestry" {
			t.Fatalf("a fast-forward merge: %+v", r)
		}
	})
	t.Run("one real commit, merged by squash", func(t *testing.T) {
		repo, _ := repoWithCommit(t)
		p := addToLane(t, repo, "sq")
		startRecord(t, repo, "sq")
		commitIn(t, repo, p, "s.txt", "s\n")
		repo.Write("other.txt", "other\n")
		repo.Commit("other")
		repo.Git("merge", "-q", "--squash", "sq")
		repo.Git("commit", "-q", "-m", "squashed")
		if r := judged(t, repo, unreachable, "sq"); r.Merged != "merged" || r.JudgedBy != "content" {
			t.Fatalf("a squash merge: %+v", r)
		}
	})
	t.Run("one real commit, merged by rebase", func(t *testing.T) {
		repo, _ := repoWithCommit(t)
		p := addToLane(t, repo, "rb")
		startRecord(t, repo, "rb")
		c := commitIn(t, repo, p, "r.txt", "r\n")
		repo.Write("other.txt", "other\n")
		repo.Commit("other")
		repo.Git("cherry-pick", c)
		if r := judged(t, repo, unreachable, "rb"); r.Merged != "merged" || r.JudgedBy != "content" {
			t.Fatalf("a rebase merge: %+v", r)
		}
	})
	t.Run("an existing branch committed to after add, merged by fast-forward", func(t *testing.T) {
		repo, _ := repoWithCommit(t)
		repo.Git("branch", "topic")
		got, err := worktree.Add(repo.Root(), worktree.AddOptions{Name: "topic", Branch: "topic", Existing: true, Git: git})
		if err != nil {
			t.Fatal(err)
		}
		startRecord(t, repo, "topic")
		commitIn(t, repo, got.Path, "t.txt", "t\n")
		repo.Git("merge", "-q", "--ff-only", "topic")
		if r := judged(t, repo, unreachable, "topic"); r.Merged != "merged" || r.JudgedBy != "ancestry" {
			t.Fatalf("an existing branch's commit made in the store: %+v", r)
		}
	})
}

// TestMergedJudgementFallsBackWithoutAStartRecord: a start record that is
// missing, unreadable, not a regular file, or names another branch is no
// record, and the branch is judged by the reflog rule as before: a real
// commit merged by fast-forward is merged, a branch never committed to is not.
func TestMergedJudgementFallsBackWithoutAStartRecord(t *testing.T) {
	for name, spoil := range map[string]func(t *testing.T, repo *gittest.Repo, record string){
		"missing": func(t *testing.T, _ *gittest.Repo, record string) {
			if err := os.Remove(record); err != nil {
				t.Fatal(err)
			}
		},
		"corrupted": func(t *testing.T, _ *gittest.Repo, record string) {
			if err := os.WriteFile(record, []byte("{\"start\": \"not a commit\n\x00"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"a symlink": func(t *testing.T, _ *gittest.Repo, record string) {
			moved := filepath.Join(t.TempDir(), "record")
			if err := os.Rename(record, moved); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(moved, record); err != nil {
				t.Fatal(err)
			}
		},
		"naming another branch": func(t *testing.T, repo *gittest.Repo, record string) {
			data, err := os.ReadFile(record)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(record, []byte(strings.Replace(string(data), "refs/heads/", "refs/heads/other-", 1)), 0o600); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			repo, _ := repoWithCommit(t)
			p := addToLane(t, repo, "real")
			addToLane(t, repo, "fresh")
			for _, wt := range []string{"real", "fresh"} {
				spoil(t, repo, startRecord(t, repo, wt))
			}
			commitIn(t, repo, p, "r.txt", "r\n")
			repo.Git("merge", "-q", "--ff-only", "real")
			if r := judged(t, repo, unreachable, "real"); r.Merged != "merged" || r.JudgedBy != "ancestry" {
				t.Fatalf("a real commit merged is merged by the reflog rule: %+v", r)
			}
			if r := judged(t, repo, unreachable, "fresh"); r.Merged != "unmerged" || r.JudgedBy != "" {
				t.Fatalf("a branch never committed to stays unmerged: %+v", r)
			}
		})
	}
}
