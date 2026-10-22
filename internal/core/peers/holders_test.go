package peers_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/peers"
)

const (
	draftRel   = ".abcd/development/intents/drafts/itd-50-a-drafted-intent.md"
	plannedRel = ".abcd/development/intents/planned/itd-50-a-drafted-intent.md"
	shippedRel = ".abcd/development/intents/shipped/itd-50-a-drafted-intent.md"
)

// staleFixture commits itd-50 to drafts/ on main, runs cut to make a peer
// while the intent is still a draft, and then moves the intent to planned/ on
// main: the peer was cut before the record left drafts/, and holds the default
// branch's older copy unless it changed it itself (iss-2610222008081888).
func staleFixture(t *testing.T, cut func(f *fixture)) *fixture {
	t.Helper()
	f := newFixture(t)
	f.write(f.here(), draftRel, intentRec("itd-50", "A drafted intent"))
	f.git(f.here(), "add", "-A")
	f.git(f.here(), "commit", "-q", "-m", "draft itd-50")
	cut(f)
	if err := os.MkdirAll(filepath.Join(f.here(), filepath.Dir(plannedRel)), 0o755); err != nil {
		t.Fatal(err)
	}
	f.git(f.here(), "mv", draftRel, plannedRel)
	f.git(f.here(), "commit", "-q", "-m", "plan itd-50")
	return f
}

// branchAhead cuts branch from main and gives it a commit of its own that
// touches no record, so it is a live peer (not merged) holding main's copy.
func branchAhead(f *fixture, branch string) {
	f.git(f.here(), "switch", "-q", "-c", branch)
	f.write(f.here(), "note-"+filepath.Base(branch), "ahead\n")
	f.git(f.here(), "add", "-A")
	f.git(f.here(), "commit", "-q", "-m", "work on "+branch)
	f.git(f.here(), "switch", "-q", "main")
}

// worktreeAhead adds a worktree on branch cut from main, with a commit of its
// own that touches no record.
func worktreeAhead(f *fixture, name, branch string) string {
	dir := f.worktree(name, branch)
	f.write(dir, "note", "ahead\n")
	f.git(dir, "add", "note")
	f.git(dir, "commit", "-q", "-m", "work on "+branch)
	return dir
}

func holderBranches(locs []peers.Location) []string {
	var out []string
	for _, l := range locs {
		out = append(out, l.Branch+":"+l.Folder)
	}
	return out
}

// A branch cut while the intent was a draft, which never touched the record,
// holds the default branch's older copy: Locate still names it (the not-found
// paths ask where a record is), and Holders does not (it holds a stale copy,
// not the record).
func TestABranchHoldingTheDefaultBranchsOlderCopyIsNoHolder(t *testing.T) {
	f := staleFixture(t, func(f *fixture) { branchAhead(f, "feat/old") })
	rep := f.read()
	if locs := rep.Locate("itd-50"); len(locs) != 1 || locs[0].Branch != "feat/old" || locs[0].Folder != "drafts" {
		t.Fatalf("Locate(itd-50) = %+v, want feat/old's drafts/ (Locate's meaning is unchanged)", locs)
	}
	if got := rep.Holders("itd-50", "planned"); len(got) != 0 {
		t.Fatalf("Holders(itd-50, planned) = %v, want none: feat/old never changed the record since its merge-base", holderBranches(got))
	}
}

// A branch that changed the record since its merge-base holds it: one that
// re-drafted a planned intent, one that shipped it, and one that edited its
// draft in place after the default branch planned it.
func TestABranchThatChangedTheRecordSinceItsMergeBaseIsAHolder(t *testing.T) {
	t.Run("re-drafted", func(t *testing.T) {
		f := staleFixture(t, func(*fixture) {})
		f.git(f.here(), "switch", "-q", "-c", "lane-redraft")
		f.git(f.here(), "mv", plannedRel, draftRel)
		f.git(f.here(), "commit", "-q", "-m", "re-draft itd-50")
		f.git(f.here(), "switch", "-q", "main")
		if got := f.read().Holders("itd-50", "planned"); len(got) != 1 || got[0].Branch != "lane-redraft" || got[0].Folder != "drafts" {
			t.Fatalf("Holders = %v, want lane-redraft:drafts", holderBranches(got))
		}
	})
	t.Run("shipped", func(t *testing.T) {
		f := staleFixture(t, func(*fixture) {})
		f.git(f.here(), "switch", "-q", "-c", "lane-ship")
		if err := os.MkdirAll(filepath.Join(f.here(), filepath.Dir(shippedRel)), 0o755); err != nil {
			t.Fatal(err)
		}
		f.git(f.here(), "mv", plannedRel, shippedRel)
		f.git(f.here(), "commit", "-q", "-m", "ship itd-50")
		f.git(f.here(), "switch", "-q", "main")
		if got := f.read().Holders("itd-50", "planned"); len(got) != 1 || got[0].Branch != "lane-ship" || got[0].Folder != "shipped" {
			t.Fatalf("Holders = %v, want lane-ship:shipped", holderBranches(got))
		}
	})
	t.Run("edited its draft", func(t *testing.T) {
		f := staleFixture(t, func(f *fixture) {
			f.git(f.here(), "switch", "-q", "-c", "lane-edit")
			f.write(f.here(), draftRel, intentRec("itd-50", "A drafted intent, rewritten"))
			f.git(f.here(), "add", "-A")
			f.git(f.here(), "commit", "-q", "-m", "rewrite the draft")
			f.git(f.here(), "switch", "-q", "main")
		})
		if got := f.read().Holders("itd-50", "planned"); len(got) != 1 || got[0].Branch != "lane-edit" || got[0].Folder != "drafts" {
			t.Fatalf("Holders = %v, want lane-edit:drafts", holderBranches(got))
		}
	})
}

// A worktree peer is judged off its disk against the merge-base of its HEAD
// and the default ref: a clean stale copy is no holder, and an uncommitted
// edit or move of the record is one.
func TestAWorktreeIsJudgedOffItsDisk(t *testing.T) {
	t.Run("a clean stale copy", func(t *testing.T) {
		f := staleFixture(t, func(f *fixture) { worktreeAhead(f, "old", "feat/old-wt") })
		if got := f.read().Holders("itd-50", "planned"); len(got) != 0 {
			t.Fatalf("Holders = %v, want none", holderBranches(got))
		}
	})
	t.Run("an uncommitted edit", func(t *testing.T) {
		var dir string
		f := staleFixture(t, func(f *fixture) { dir = worktreeAhead(f, "edit", "feat/edit-wt") })
		f.write(dir, draftRel, intentRec("itd-50", "Edited, not committed"))
		if got := f.read().Holders("itd-50", "planned"); len(got) != 1 || got[0].Branch != "feat/edit-wt" {
			t.Fatalf("Holders = %v, want feat/edit-wt", holderBranches(got))
		}
	})
	t.Run("an uncommitted move", func(t *testing.T) {
		f := staleFixture(t, func(*fixture) {})
		dir := worktreeAhead(f, "move", "feat/move-wt")
		if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(shippedRel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(filepath.Join(dir, plannedRel), filepath.Join(dir, shippedRel)); err != nil {
			t.Fatal(err)
		}
		if got := f.read().Holders("itd-50", "planned"); len(got) != 1 || got[0].Branch != "feat/move-wt" || got[0].Folder != "shipped" {
			t.Fatalf("Holders = %v, want feat/move-wt:shipped", holderBranches(got))
		}
	})
}

// The judgement fails closed: where this checkout's HEAD, the merge-base or
// the peer's copy cannot be read, the location still counts as holding it.
func TestAHoldingWhoseBaseCannotBeReadStillHolds(t *testing.T) {
	t.Run("no merge-base with the default ref", func(t *testing.T) {
		f := staleFixture(t, func(*fixture) {})
		f.git(f.here(), "switch", "-q", "--orphan", "unrelated")
		f.write(f.here(), draftRel, intentRec("itd-50", "A drafted intent"))
		f.git(f.here(), "add", draftRel)
		f.git(f.here(), "commit", "-q", "-m", "an unrelated history")
		f.git(f.here(), "clean", "-fdq")
		f.git(f.here(), "switch", "-q", "-f", "main")
		if got := f.read().Holders("itd-50", "planned"); len(got) != 1 || got[0].Branch != "unrelated" {
			t.Fatalf("Holders = %v, want unrelated (its history shares no base with main)", holderBranches(got))
		}
	})
	t.Run("an unborn HEAD here", func(t *testing.T) {
		f := staleFixture(t, func(f *fixture) { branchAhead(f, "feat/old") })
		f.git(f.here(), "switch", "-q", "--orphan", "nothing-yet")
		if got := f.read().Holders("itd-50", "planned"); len(got) != 1 || got[0].Branch != "feat/old" {
			t.Fatalf("Holders = %v, want feat/old (this checkout has no HEAD to judge it against)", holderBranches(got))
		}
	})
	t.Run("a worktree copy that cannot be read", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("chmod 0 does not deny root")
		}
		var dir string
		f := staleFixture(t, func(f *fixture) { dir = worktreeAhead(f, "shut", "feat/shut-wt") })
		p := filepath.Join(dir, filepath.FromSlash(draftRel))
		if err := os.Chmod(p, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(p, 0o644) })
		if got := f.read().Holders("itd-50", "planned"); len(got) != 1 || got[0].Branch != "feat/shut-wt" {
			t.Fatalf("Holders = %v, want feat/shut-wt (its copy cannot be compared)", holderBranches(got))
		}
	})
}

// A holding in this checkout's own folder is a copy, not a holding, whatever
// the peer did to it: Holders never names one.
func TestAHoldingInThisCheckoutsFolderIsNoHolder(t *testing.T) {
	f := staleFixture(t, func(*fixture) {})
	f.git(f.here(), "switch", "-q", "-c", "lane-touch")
	f.write(f.here(), plannedRel, intentRec("itd-50", "Touched in place"))
	f.git(f.here(), "add", "-A")
	f.git(f.here(), "commit", "-q", "-m", "touch the plan")
	f.git(f.here(), "switch", "-q", "main")
	if got := f.read().Holders("itd-50", "planned"); len(got) != 0 {
		t.Fatalf("Holders = %v, want none (same folder)", holderBranches(got))
	}
}

// onMain runs fn in a temporary worktree checked out on main, then removes the
// worktree, so main moves on while this checkout stays where it is.
func (f *fixture) onMain(fn func(dir string)) {
	f.t.Helper()
	dir := filepath.Join(f.home, "wt", "on-main")
	f.git(f.here(), "worktree", "add", "-q", dir, "main")
	fn(dir)
	f.git(f.here(), "worktree", "remove", dir)
}

// commitIn commits everything in dir.
func (f *fixture) commitIn(dir, msg string) {
	f.t.Helper()
	f.git(dir, "add", "-A")
	f.git(dir, "commit", "-q", "-m", msg)
}

// The peer is measured against its merge-base with THIS checkout, not with
// the default ref: when this checkout is behind main, a branch cut after main
// shipped the intent holds it shipped, and a build here must not start it.
func TestABranchCutAfterTheDefaultBranchShippedItIsAHolderOfACheckoutBehind(t *testing.T) {
	f := newFixture(t)
	f.write(f.here(), plannedRel, intentRec("itd-50", "A planned intent"))
	f.commitIn(f.here(), "plan itd-50")
	f.git(f.here(), "switch", "-q", "-c", "behind")
	f.onMain(func(dir string) {
		if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(shippedRel)), 0o755); err != nil {
			t.Fatal(err)
		}
		f.git(dir, "mv", plannedRel, shippedRel)
		f.commitIn(dir, "ship itd-50")
		f.git(dir, "switch", "-q", "-c", "feat/fresh")
		f.write(dir, "note", "ahead\n")
		f.commitIn(dir, "work on feat/fresh")
	})
	if got := f.read().Holders("itd-50", "planned"); len(got) != 1 || got[0].Branch != "feat/fresh" || got[0].Folder != "shipped" {
		t.Fatalf("Holders = %v, want feat/fresh:shipped (main shipped it past this checkout)", holderBranches(got))
	}
}

// A checkout left in drafts/ while main planned the intent: a lane cut from
// main's tip, which never touched the record, holds it planned, which this
// checkout has not seen, and is a holder.
func TestAFreshLaneIsAHolderOfAStaleCheckout(t *testing.T) {
	f := newFixture(t)
	f.write(f.here(), draftRel, intentRec("itd-50", "A drafted intent"))
	f.commitIn(f.here(), "draft itd-50")
	f.git(f.here(), "switch", "-q", "-c", "stale")
	f.onMain(func(dir string) {
		if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(plannedRel)), 0o755); err != nil {
			t.Fatal(err)
		}
		f.git(dir, "mv", draftRel, plannedRel)
		f.commitIn(dir, "plan itd-50")
		f.git(dir, "switch", "-q", "-c", "lane-fresh")
		f.write(dir, "note", "ahead\n")
		f.commitIn(dir, "work on lane-fresh")
	})
	if got := f.read().Holders("itd-50", "drafts"); len(got) != 1 || got[0].Branch != "lane-fresh" || got[0].Folder != "planned" {
		t.Fatalf("Holders = %v, want lane-fresh:planned", holderBranches(got))
	}
}

// In a criss-cross history the peer and this checkout have two merge-bases,
// and a single `git merge-base` names whichever its dates favour. The peer
// is unchanged only when its copy matches the copy in EVERY merge-base: here
// the peer's draft matches one base's and not the other's, whichever side
// rewrote it, so it holds the record either way.
func TestACrissCrossPeerMatchingOneMergeBaseOnlyIsAHolder(t *testing.T) {
	for _, side := range []string{"main", "y"} {
		t.Run("rewritten on "+side, func(t *testing.T) {
			f := newFixture(t)
			f.write(f.here(), draftRel, intentRec("itd-50", "A drafted intent"))
			f.commitIn(f.here(), "draft itd-50")
			f.git(f.here(), "branch", "y")
			step := func(branch string) string {
				f.git(f.here(), "switch", "-q", branch)
				if branch == side {
					f.write(f.here(), draftRel, intentRec("itd-50", "A drafted intent, rewritten"))
				} else {
					f.write(f.here(), "note-"+branch, "side\n")
				}
				f.commitIn(f.here(), "one side on "+branch)
				return f.git(f.here(), "rev-parse", "HEAD")
			}
			x1, y1 := step("main"), step("y")
			f.git(f.here(), "switch", "-q", "main")
			f.git(f.here(), "merge", "-q", "--no-ff", "-m", "x2", y1)
			f.git(f.here(), "switch", "-q", "y")
			f.git(f.here(), "merge", "-q", "--no-ff", "-m", "y2", x1)
			f.git(f.here(), "switch", "-q", "main")
			if err := os.MkdirAll(filepath.Join(f.here(), filepath.Dir(plannedRel)), 0o755); err != nil {
				t.Fatal(err)
			}
			f.git(f.here(), "mv", draftRel, plannedRel)
			f.commitIn(f.here(), "plan itd-50")
			if bases := strings.Fields(f.git(f.here(), "merge-base", "--all", "y", "HEAD")); len(bases) != 2 {
				t.Fatalf("the fixture is not criss-cross: merge-bases %v", bases)
			}
			if got := f.read().Holders("itd-50", "planned"); len(got) != 1 || got[0].Branch != "y" || got[0].Folder != "drafts" {
				t.Fatalf("Holders = %v, want y:drafts", holderBranches(got))
			}
		})
	}
}
