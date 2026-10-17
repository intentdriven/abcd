package loop

import (
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// The key a drain draws its own order from (spc-2610161044461448 step 2):
// this copy's holder, read with no network act and lower-cased, and its copy
// key; the copy key alone when no holder can be told; and nothing when
// neither is there. A move mints the copy's key before it plans, so the order
// it asks a judgement in is the one the next move keeps.

// TestDrainSpreadIsTheHolderAndTheCopyKey: a GitHub no-reply address names
// the holder; an address that names none leaves the copy key alone to tell
// copies apart; with neither the spread is empty.
func TestDrainSpreadIsTheHolderAndTheCopyKey(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	key := strings.Repeat("a", 32)
	repo.Git("config", "user.email", "1234+Octo@users.noreply.github.com")
	if got := drainSpread(repo.Root(), key); got != "octo\x00"+key {
		t.Fatalf("drainSpread with a no-reply address = %q; want the holder, lower-cased, and the key", got)
	}
	repo.Git("config", "user.email", "pat@example.com")
	if got := drainSpread(repo.Root(), key); got != "\x00"+key {
		t.Fatalf("drainSpread with no holder to tell = %q; want the copy key alone", got)
	}
	if got := drainSpread(repo.Root(), ""); got != "" {
		t.Fatalf("drainSpread with neither = %q; want none", got)
	}
}

// TestTheFirstMoveMintsTheKeyBeforeItPlans: on a fresh copy the first move
// mints the copy's key and plans the order from it, so a dry run after it
// shows that order, and the next move, unanswered, asks over the same issue.
func TestTheFirstMoveMintsTheKeyBeforeItPlans(t *testing.T) {
	repo, _ := groupDrainRepo(t)
	if claim.OwnCopyKey(repo.Root()) != "" {
		t.Fatal("the fixture's copy already has a key")
	}
	first, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(copyKeyPath(repo)); err != nil || claim.OwnCopyKey(repo.Root()) == "" {
		t.Fatalf("the first move minted no key: %v", err)
	}
	pv, err := DrainDryRun(repo.Root(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(eligibleIDs(pv.Dispositions), eligibleIDs(first.Dispositions)) {
		t.Fatalf("the first move's order = %v; drawn from its key = %v", eligibleIDs(first.Dispositions), eligibleIDs(pv.Dispositions))
	}
	next, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if first.Judging == nil || next.Judging == nil || next.Judging.Issue != first.Judging.Issue {
		t.Fatalf("the next move asks over %+v; the first asked over %+v", next.Judging, first.Judging)
	}
}
