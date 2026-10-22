package peers_test

import (
	"os"
	"path/filepath"
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

// The judgement fails closed: where the default ref, the merge-base or the
// peer's copy cannot be read, the location still counts as holding it.
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
	t.Run("no default ref", func(t *testing.T) {
		f := staleFixture(t, func(f *fixture) { branchAhead(f, "feat/old") })
		f.git(f.here(), "branch", "-q", "-m", "main", "work")
		rep := f.read()
		if rep.DefaultRef != "" {
			t.Fatalf("the fixture still resolves a default ref %q", rep.DefaultRef)
		}
		if got := rep.Holders("itd-50", "planned"); len(got) != 1 || got[0].Branch != "feat/old" {
			t.Fatalf("Holders = %v, want feat/old (no default ref to judge it against)", holderBranches(got))
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
