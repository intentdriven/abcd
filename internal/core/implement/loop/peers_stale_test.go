package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

const draftedRel = ".abcd/development/intents/drafts/itd-10-alpha.md"

// draftedThenPlanned stands up the READY repository with itd-10 committed in
// drafts/ first, runs cut while it is a draft, and then plans it on main, so
// whatever cut made was cut before the intent left drafts/.
func draftedThenPlanned(t *testing.T, cut func(repo *gittest.Repo)) *gittest.Repo {
	t.Helper()
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	if err := os.MkdirAll(filepath.Join(repo.Root(), filepath.Dir(draftedRel)), 0o755); err != nil {
		t.Fatal(err)
	}
	repo.Git("mv", plannedRel, draftedRel)
	repo.Git("commit", "-q", "-m", "itd-10 is a draft")
	cut(repo)
	repo.Git("mv", draftedRel, plannedRel)
	repo.Git("commit", "-q", "-m", "plan itd-10")
	return repo
}

// staleBranch cuts branch from main with a commit of its own that touches no
// record, the shape of a long-lived branch left behind.
func staleBranch(repo *gittest.Repo, branch string) {
	repo.Git("checkout", "-q", "-b", branch)
	repo.Write("note-"+branch, "ahead\n")
	repo.Commit("work on " + branch)
	repo.Git("checkout", "-q", "main")
}

func peersRow(t *testing.T, repo *gittest.Repo) CheckRow {
	t.Helper()
	chk, err := Check(repo.Root(), "itd-10")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range chk.Checks {
		if row.Name == CheckPeers {
			return row
		}
	}
	t.Fatalf("no peers row in %+v", chk.Checks)
	return CheckRow{}
}

// TestThePeersCheckPassesOverAStaleCopy is iss-2610222008081888: branches
// cut before the intent left drafts/, which never changed the record since
// their merge-base with the default branch, hold the default branch's older
// copy, not the intent, and the build's peers check passes over them.
func TestThePeersCheckPassesOverAStaleCopy(t *testing.T) {
	repo := draftedThenPlanned(t, func(repo *gittest.Repo) {
		staleBranch(repo, "feat/hosted-sites-design")
		staleBranch(repo, "fix/fence-day-lease")
	})
	if row := peersRow(t, repo); !row.OK {
		t.Fatalf("two stale branches holding main's older draft refuse the build: %+v", row)
	}
	if _, err := Start(repo.Root(), "itd-10", Options{}); err != nil {
		t.Fatalf("the start is refused past a stale copy: %v", err)
	}
}

// TestThePeersCheckStillRefusesAPeerThatChangedTheRecord: a branch that
// re-drafted the intent since its merge-base, one cut while it was a draft
// that rewrote the draft, and a worktree with an uncommitted move each hold
// it; and so does a branch cut after main shipped it, when this checkout is
// behind main (the peer is measured against this checkout, not main).
func TestThePeersCheckStillRefusesAPeerThatChangedTheRecord(t *testing.T) {
	t.Run("a branch that re-drafted it", func(t *testing.T) {
		repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
		repo.Git("checkout", "-q", "-b", "lane-redraft")
		if err := os.MkdirAll(filepath.Join(repo.Root(), filepath.Dir(draftedRel)), 0o755); err != nil {
			t.Fatal(err)
		}
		repo.Git("mv", plannedRel, draftedRel)
		repo.Git("commit", "-q", "-m", "re-draft")
		repo.Git("checkout", "-q", "main")
		if row := peersRow(t, repo); row.OK || !strings.Contains(row.Detail, "branch lane-redraft holds it in drafts/") {
			t.Fatalf("want lane-redraft named as holding it in drafts/: %+v", row)
		}
	})
	t.Run("a stale branch that rewrote its draft", func(t *testing.T) {
		repo := draftedThenPlanned(t, func(repo *gittest.Repo) {
			repo.Git("checkout", "-q", "-b", "lane-edit")
			repo.Write(draftedRel, readyIntent("", settledQuestions)+"\nRewritten.\n")
			repo.Commit("rewrite the draft")
			repo.Git("checkout", "-q", "main")
		})
		if row := peersRow(t, repo); row.OK || !strings.Contains(row.Detail, "branch lane-edit holds it in drafts/") {
			t.Fatalf("want lane-edit named as holding it in drafts/: %+v", row)
		}
	})
	t.Run("a worktree with an uncommitted move", func(t *testing.T) {
		repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
		wt := filepath.Join(t.TempDir(), "wt-move")
		repo.Git("worktree", "add", "-q", "-b", "lane-move", wt)
		if err := os.WriteFile(filepath.Join(wt, "note"), []byte("ahead\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		repo.Git("-C", wt, "add", "note")
		repo.Git("-C", wt, "commit", "-q", "-m", "ahead of main")
		if err := os.MkdirAll(filepath.Join(wt, filepath.Dir(draftedRel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(filepath.Join(wt, plannedRel), filepath.Join(wt, draftedRel)); err != nil {
			t.Fatal(err)
		}
		if row := peersRow(t, repo); row.OK || !strings.Contains(row.Detail, "(branch lane-move) holds it in drafts/") {
			t.Fatalf("want the worktree named as holding it in drafts/: %+v", row)
		}
	})
	t.Run("a branch cut after main shipped it, past this checkout", func(t *testing.T) {
		repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
		repo.Git("checkout", "-q", "-b", "behind")
		tmp := filepath.Join(t.TempDir(), "on-main")
		repo.Git("worktree", "add", "-q", tmp, "main")
		shipped := ".abcd/development/intents/shipped/itd-10-alpha.md"
		if err := os.MkdirAll(filepath.Join(tmp, filepath.Dir(shipped)), 0o755); err != nil {
			t.Fatal(err)
		}
		repo.Git("-C", tmp, "mv", plannedRel, shipped)
		repo.Git("-C", tmp, "commit", "-q", "-m", "ship alpha")
		repo.Git("-C", tmp, "checkout", "-q", "-b", "lane-fresh")
		if err := os.WriteFile(filepath.Join(tmp, "note"), []byte("ahead\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		repo.Git("-C", tmp, "add", "note")
		repo.Git("-C", tmp, "commit", "-q", "-m", "ahead of main")
		repo.Git("worktree", "remove", tmp)
		if row := peersRow(t, repo); row.OK || !strings.Contains(row.Detail, "branch lane-fresh holds it in shipped/") {
			t.Fatalf("an intent main shipped past this checkout must not start here: %+v", row)
		}
	})
}
