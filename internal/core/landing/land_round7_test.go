package landing

import (
	"strings"
	"testing"
	"time"
)

// Round 7: the forge lags land's own push (iss-2610151855227496). Twice on
// 2026-10-15/16 a real run pushed feat/abcd-land and, in the same run, read
// pull request #1020 at the old head; a re-run 12 and 14 seconds later saw
// the new one. After a push that moves the remote head, land re-reads the
// pull request's head until the forge reports the pushed commit, within a
// bound.

// openListings are the open pull-request listings the landing made.
func (f *fixture) openListings() int {
	return f.forge.called("pr", "list", "--head=feat/x", "--state=open")
}

func TestALaggingForgeHeadIsWaitedForThenArmed(t *testing.T) {
	f := newFixture(t, queueRuleset)
	old := f.prFor(strings.Repeat("a", 40))
	f.forge.prsSeq = []string{old, old, f.prFor(f.head())}
	f.forge.views = []string{viewBlocked, viewQueued}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if s := step(t, res, "enqueue"); s.Status != StatusDone {
		t.Fatalf("enqueue step %+v, want armed once the forge showed the push", s)
	}
	if n := f.openListings(); n != 3 {
		t.Fatalf("%d open listings, want exactly 3 (old, old, new): %v", n, f.forge.calls)
	}
	if f.forge.called("pr", "merge", "7", "--auto", "--merge", "--match-head-commit="+f.head()) != 1 {
		t.Fatalf("forge calls %v, want one arm pinned to the head", f.forge.calls)
	}
	if len(f.pushes()) != 1 || len(f.gates) != 1 {
		t.Fatalf("pushes %v, gates %v: the wait repeats only the listing", f.pushes(), f.gates)
	}
	if len(f.slept) != 2 || f.slept[0] != headWaitInterval {
		t.Fatalf("slept %v, want two pauses of %v", f.slept, headWaitInterval)
	}
}

func TestAForgeThatNeverShowsThePushStopsAfterTheBoundedWait(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(strings.Repeat("a", 40))
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	s := step(t, res, "enqueue")
	if s.Status != StatusStopped || f.forge.called("pr", "merge") != 0 {
		t.Fatalf("enqueue step %+v, calls %v: want stopped and unarmed", s, f.forge.calls)
	}
	if !strings.Contains(s.Note, "if it is older, wait for the forge to see the push, then run abcd land again") || !strings.Contains(s.Note, "land waited 30 s") {
		t.Fatalf("enqueue note %q: want today's wording and the 30 s land waited", s.Note)
	}
	if n := f.openListings(); n < 2 || n > 16 {
		t.Fatalf("%d open listings, want a wait of at most 16", n)
	}
	var total time.Duration
	for _, d := range f.slept {
		total += d
	}
	if total > headWaitCap || headWaitInterval <= 0 {
		t.Fatalf("slept %v in all, over the %v cap", total, headWaitCap)
	}
	if len(f.pushes()) != 1 || len(f.gates) != 1 {
		t.Fatalf("pushes %v, gates %v: the wait repeats only the listing", f.pushes(), f.gates)
	}
}

func TestAPushThatChangedNothingDoesNotWait(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.repo.Git("push", "-q", "origin", "feat/x")
	f.forge.prs = f.prFor(strings.Repeat("a", 40))
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if s := step(t, res, "push"); !strings.Contains(s.Note, "changed nothing") {
		t.Fatalf("push step %+v, want a push that changed nothing", s)
	}
	if s := step(t, res, "enqueue"); s.Status != StatusStopped || strings.Contains(s.Note, "waited") {
		t.Fatalf("enqueue step %+v, want today's stop with no wait", s)
	}
	if n := f.openListings(); n != 1 || len(f.slept) != 0 {
		t.Fatalf("%d open listings, slept %v: want one listing and no pause", n, f.slept)
	}
}

func TestAForgeErrorDuringTheWaitRefusesUnarmed(t *testing.T) {
	f := newFixture(t, queueRuleset)
	f.forge.prs = f.prFor(strings.Repeat("a", 40))
	f.forge.prsErrAt = 2
	_, err := f.land(f.options())
	wantRefusal(t, err, "the listing failed")
	if f.forge.called("pr", "merge") != 0 {
		t.Fatalf("armed: %v", f.forge.calls)
	}
	if n := f.openListings(); n != 2 {
		t.Fatalf("%d open listings, want the wait to end at the failing second", n)
	}
}
