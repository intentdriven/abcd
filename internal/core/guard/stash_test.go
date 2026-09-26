package guard

import (
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestSharedStashWarnsInAMultiWorktreeClone — iss-2609190338340796. git keeps
// ONE stash stack per repository, not per worktree, so in a clone with several
// worktrees a bare `git stash` then `git stash pop` in one lane can pop another
// lane's entry into the wrong tree, with no message either lane can act on. The
// guard warns on the bare forms there, naming the shared stack; a stash with a
// message and a pop by entry stay allowed, and a single-worktree clone keeps
// allow for everything.
func TestSharedStashWarnsInAMultiWorktreeClone(t *testing.T) {
	cases := []struct {
		cmd  string
		warn bool
	}{
		{"git stash", true},
		{"git stash pop", true},
		{"git stash apply", true},
		{"git stash push", true},
		{"git stash --include-untracked", true},
		{"git stash push -- internal/core", true},
		{"git -C ../lane stash pop", true},
		{"git stash save", true},

		{"git stash push -m 'lane-a: probe'", false},
		{"git stash -m 'lane-a: probe'", false},
		{"git stash push --message=lane-a", false},
		{"git stash save 'lane-a: probe'", false},
		{"git stash pop stash@{1}", false},
		{"git stash apply stash@{0}", false},
		{"git stash list", false},
		{"git stash show -p stash@{0}", false},
		{"git status", false},
	}
	multi := Defaults()
	multi.worktrees = func() int { return 3 }
	single := Defaults()
	single.worktrees = func() int { return 1 }
	for _, tc := range cases {
		t.Run(tc.cmd, func(t *testing.T) {
			d, err := multi.Check(tc.cmd)
			if err != nil {
				t.Fatal(err)
			}
			if tc.warn {
				if d.Verdict != VerdictWarn || d.EntryID != stashEntryID {
					t.Errorf("multi-worktree Check(%q) = %q via %q, want warn via %q", tc.cmd, d.Verdict, d.EntryID, stashEntryID)
				}
			} else if d.EntryID == stashEntryID || contains(d.Matches, stashEntryID) {
				t.Errorf("multi-worktree Check(%q) warned on a stash that names its entry: %+v", tc.cmd, d)
			}
			if sd, _ := single.Check(tc.cmd); sd.Verdict != VerdictAllow && contains(sd.Matches, stashEntryID) {
				t.Errorf("single-worktree Check(%q) = %q via %q, want no stash warning", tc.cmd, sd.Verdict, sd.EntryID)
			}
		})
	}
	if contains(reservedEntryIDs, stashEntryID) == false {
		t.Errorf("the stash id must be reserved so no repo entry can claim the guard's own voice")
	}
}

// TestSharedStashCountsRealWorktrees wires the count to git: a registry loaded
// for a repository reads its worktree list, and only when a stash segment is
// present.
func TestSharedStashCountsRealWorktrees(t *testing.T) {
	repo := newStashRepo(t)
	r, err := Load(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if d, _ := r.Check("git stash"); d.EntryID == stashEntryID {
		t.Fatalf("a single-worktree clone warned: %+v", d)
	}
	repo.Git("worktree", "add", "-q", "-b", "lane", t.TempDir()+"/lane")
	r, err = Load(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if d, _ := r.Check("git stash pop"); d.Verdict != VerdictWarn || d.EntryID != stashEntryID {
		t.Fatalf("a two-worktree clone did not warn on a bare pop: %+v", d)
	}
}

// newStashRepo is a repository with one commit, which `git worktree add` needs.
func newStashRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	repo := gittest.NewRepo(t)
	repo.Write("README", "x\n")
	repo.Commit("seed")
	return repo
}
