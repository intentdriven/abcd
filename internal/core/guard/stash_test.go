package guard

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestStashWarnsWhereWorktreesShareTheList — iss-2609190338340796 and
// iss-2611220942432452. git keeps ONE stash list per repository, shared by
// every linked worktree, so in a clone with several worktrees a stash in one
// lane and a pop in another can hand the second lane the first lane's work. A
// stash that names its entry is no safer: `stash@{0}` is whatever the newest
// entry is when the command runs, and another lane may have pushed since. The
// guard warns on every stash that changes the list there, through a registry
// entry the SHELL rules teach; `git stash list` and `git stash show` only read
// it and stay allowed, and a single-worktree clone keeps allow for everything.
func TestStashWarnsWhereWorktreesShareTheList(t *testing.T) {
	cases := []struct {
		cmd  string
		warn bool
	}{
		{"git stash", true},
		{"git stash pop", true},
		{"git stash apply", true},
		{"git stash push", true},
		{"git stash drop", true},
		{"git stash clear", true},
		{"git stash --include-untracked", true},
		{"git stash push -- internal/core", true},
		{"git -C ../lane stash pop", true},
		{"git stash save", true},
		{"git stash push -m 'lane-a: probe'", true},
		{"git stash -m 'lane-a: probe'", true},
		{"git stash -m list", true},
		{"git stash push --message=lane-a", true},
		{"git stash save 'lane-a: probe'", true},
		{"git stash pop stash@{1}", true},
		{"git stash apply stash@{0}", true},
		{"git stash drop stash@{0}", true},
		{"git stash branch lane-b stash@{0}", true},
		{"git stash $(echo pop)", true},
		{"sh -c 'git stash pop'", true},

		{"git stash list", false},
		{"git stash show -p stash@{0}", false},
		{"git -C ../lane stash list", false},
		{"git status", false},
	}
	multi := Defaults()
	multi.worktrees = func() int { return 3 }
	single := Defaults()
	single.worktrees = func() int { return 1 }
	unknown := Defaults()
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
				t.Errorf("multi-worktree Check(%q) warned on a stash that only reads the list: %+v", tc.cmd, d)
			}
			if sd, _ := single.Check(tc.cmd); contains(sd.Matches, stashEntryID) {
				t.Errorf("single-worktree Check(%q) = %q via %q, want no stash warning", tc.cmd, sd.Verdict, sd.EntryID)
			}
			// With no repository to count, the guard cannot tell, and warns.
			if ud, _ := unknown.Check(tc.cmd); contains(ud.Matches, stashEntryID) != tc.warn {
				t.Errorf("Check(%q) with no repository behind it matched %v, want the stash warning %v", tc.cmd, ud.Matches, tc.warn)
			}
		})
	}
}

// TestStashEntryTeachesTheSharedList: the warning is a registry entry, so the
// SHELL rules teach it before a stash runs, and its words name the shared list
// and the two ways round it.
func TestStashEntryTeachesTheSharedList(t *testing.T) {
	r := Defaults()
	e, ok := r.Entries[stashEntryID]
	if !ok {
		t.Fatalf("the bundled registry has no %s entry", stashEntryID)
	}
	if e.Tier != TierWarn {
		t.Errorf("%s tier = %q, want warn", stashEntryID, e.Tier)
	}
	if !strings.Contains(e.Why, "one stash list") || !strings.Contains(e.Why, "worktree") {
		t.Errorf("%s why does not name the shared list: %q", stashEntryID, e.Why)
	}
	for _, want := range []string{"commit", "git diff > "} {
		if !strings.Contains(e.Successor, want) {
			t.Errorf("%s successor lacks %q: %q", stashEntryID, want, e.Successor)
		}
	}
	if contains(reservedEntryIDs, stashEntryID) {
		t.Errorf("%s is reserved, so the bundled entry could never load", stashEntryID)
	}
	warned, _ := r.SplitLessons(r)
	found := false
	for _, l := range warned {
		found = found || strings.Contains(l, "("+stashEntryID+")")
	}
	if !found {
		t.Errorf("the SHELL lessons sent first do not teach %s", stashEntryID)
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
	if d, _ := r.Check("git stash"); contains(d.Matches, stashEntryID) {
		t.Fatalf("a single-worktree clone warned: %+v", d)
	}
	repo.Git("worktree", "add", "-q", "-b", "lane", t.TempDir()+"/lane")
	r, err = Load(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	for _, cmd := range []string{"git stash pop", "git stash pop stash@{0}"} {
		if d, _ := r.Check(cmd); d.Verdict != VerdictWarn || d.EntryID != stashEntryID {
			t.Fatalf("a two-worktree clone did not warn on %q: %+v", cmd, d)
		}
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
