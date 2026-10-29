package claim

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestACopyKeyACommitCarriesIsRefused: the copy key names this working copy
// alone, so a key git tracks came with a commit and every clone would share
// it: it is refused as this copy's, and no reservation verb reads it.
func TestACopyKeyACommitCarriesIsRefused(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write(".gitignore", ".abcd/.work.local/\n")
	repo.Write(tierRelDir+"/.keep", "")
	repo.Commit("seed")
	key, err := copyKey(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if got := ownCopyKey(repo.Root()); got != key {
		t.Fatalf("ownCopyKey = %q; want this copy's own %q", got, key)
	}
	repo.Git("add", "-f", "--", copyKeyRel)
	repo.Git("commit", "-q", "-m", "an unrelated change")
	if got := ownCopyKey(repo.Root()); got != "" {
		t.Fatalf("ownCopyKey over a tracked key = %q; want none", got)
	}
	if _, err := copyKey(repo.Root()); err == nil || !strings.Contains(err.Error(), "git tracks "+copyKeyRel) {
		t.Fatalf("copyKey over a tracked key = %v; want it refused as tracked", err)
	}
}

// TestStatusStateACommitCarriesIsRefused: what this copy kept of the status
// branches (each announcement's first fetch, the forge's verdict on who
// pushed it) is its own, so state git tracks there came with a commit and the
// status is refused rather than judged by it.
func TestStatusStateACommitCarriesIsRefused(t *testing.T) {
	w := newWorld(t)
	maya := w.clone("5+maya@users.noreply.github.com", "")
	pat := w.clone("9+pat@users.noreply.github.com", "")
	maya.announceAs("merging", "#1012")
	pat.status(nil)
	pat.read()
	runGit(t, pat.dir, "add", "-f", "--", statusBranches.stateDir)
	if _, err := ReadStatus(context.Background(), pat.dir, w.clock(), pat.opts()); err == nil || !strings.Contains(err.Error(), "git tracks "+statusBranches.stateDir) {
		t.Fatalf("ReadStatus over tracked status state = %v; want it refused as tracked", err)
	}
	if _, err := FetchStatus(pat.dir, nil, pat.opts()); err == nil || !strings.Contains(err.Error(), "git tracks "+statusBranches.stateDir) {
		t.Fatalf("FetchStatus over tracked status state = %v; want it refused as tracked", err)
	}
}

// TestAFetchStampACommitCarriesIsRefused: the reservations' fetch stamp says
// when this copy last read origin, so one git tracks is refused rather than
// taken as this copy's fetch.
func TestAFetchStampACommitCarriesIsRefused(t *testing.T) {
	repo := gittest.NewRepo(t)
	repo.Write(".gitignore", ".abcd/.work.local/\n")
	repo.Write(reservations.stampRel(), time.Now().UTC().Format(time.RFC3339)+"\n")
	repo.Commit("seed")
	if _, err := View(repo.Root(), time.Now()); err != nil {
		t.Fatalf("View over this copy's own stamp: %v", err)
	}
	if LastFetched(repo.Root()).IsZero() {
		t.Fatal("LastFetched read no time from this copy's own stamp")
	}
	repo.Git("add", "-f", "--", reservations.stampRel())
	if at := LastFetched(repo.Root()); !at.IsZero() {
		t.Errorf("LastFetched over a tracked stamp = %s; want none", at)
	}
	if _, err := View(repo.Root(), time.Now()); err == nil || !strings.Contains(err.Error(), "git tracks "+reservations.stampRel()) {
		t.Fatalf("View over a tracked stamp = %v; want it refused as tracked", err)
	}
}
