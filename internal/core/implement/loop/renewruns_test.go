package loop

import (
	"os"
	"slices"
	"strings"
	"testing"
	"time"
)

// A run's reservation is renewed wherever its agent waits
// (iss-2610261837062463, dcn-2610291352512058): the gated waiter an agent's
// go test and make preflight run under, and implement status, renew a due
// reservation of a run in progress through the loop's own renewal, so one
// whose implementer holds a stage for hours does not lapse for want of a
// step.

// TestTheWaiterRenewsARunWhoseLaneIsItsCheckout: RenewRuns, from a lane's
// worktree, renews the reservation of the run that lane belongs to, though
// the run lives in another checkout, once it is half through its lease, and
// logs the renewal in the run record; under half its lease nothing is
// pushed. RenewRunsIn, implement status's, renews only the runs that live in
// the checkout it is given.
func TestTheWaiterRenewsARunWhoseLaneIsItsCheckout(t *testing.T) {
	f := newNarrowFixture(t)
	f.awaitImplementer(t)
	l := currentLane(t, f.repo, f.runID)
	before := f.tip(t)

	if got := RenewRuns(l.Worktree, before.claimed.Add(30*time.Minute)); len(got) != 0 {
		t.Fatalf("a renewal under half the lease = %+v; want none", got)
	}
	at := before.claimed.Add(61 * time.Minute)
	if got := RenewRunsIn(l.Worktree, "", at); len(got) != 0 {
		t.Fatalf("RenewRunsIn the lane's worktree, which holds no run, = %+v; want none", got)
	}
	if tip := f.tip(t); tip.commit != before.commit {
		t.Fatalf("nothing due pushed %+v over %+v", tip, before)
	}

	got := RenewRuns(l.Worktree, at)
	if len(got) != 1 || !got[0].Renewed || got[0].RunID != f.runID || got[0].Key != secondIssue ||
		!strings.Contains(got[0].Note, "renewed the reservation on "+secondIssue) {
		t.Fatalf("RenewRuns from the lane's worktree = %+v; want the run's reservation renewed", got)
	}
	renewed := f.renewedAt(t, before, at)
	if note := recordNote(t, f.repo, f.runID, StageReservationRenewed); !strings.Contains(note, stamp(renewed.expires)) {
		t.Fatalf("the run record logs the renewal with its new expiry: %q", note)
	}

	later := at.Add(61 * time.Minute)
	got = RenewRunsIn(f.repo.Root(), "", later)
	if len(got) != 1 || !got[0].Renewed || got[0].RunID != f.runID {
		t.Fatalf("RenewRunsIn the run's checkout = %+v; want its reservation renewed", got)
	}
	f.renewedAt(t, renewed, later)
}

// TestTheWaiterPassesOverARunWhoseLockIsHeld: a run whose lock another
// invocation holds is a step mid-act, which renews the reservation itself,
// so RenewRuns passes over it silently, pushing and logging nothing.
func TestTheWaiterPassesOverARunWhoseLockIsHeld(t *testing.T) {
	shortLockTimeout(t)
	f := newNarrowFixture(t)
	f.awaitImplementer(t)
	before := f.tip(t)
	logged := recordStages(t, f.repo, f.runID)
	held, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- withLock(f.repo.Root(), func(*os.Root) error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	got := RenewRuns(f.repo.Root(), before.claimed.Add(61*time.Minute))
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("RenewRuns over a held lock = %+v; want the run passed over silently", got)
	}
	if tip := f.tip(t); tip.commit != before.commit {
		t.Fatalf("a renewal was pushed under another invocation's lock: %+v", tip)
	}
	if after := recordStages(t, f.repo, f.runID); !slices.Equal(after, logged) {
		t.Fatalf("the run record changed under another invocation's lock: %v, was %v", after, logged)
	}
}

// TestALapsedRevivalIsJudgedOnTheLanesChangedFiles: the run's reservation
// of the directory docs lapsed while its implementer worked, and the same
// person's other copy reserved a file in it since. A revival backs off that
// copy's reservation of a file the lane's branch changed, logging the failed
// renewal; once that copy holds only a file the lane did not change, the
// revival goes ahead, keeps the whole directory, and its note says it went
// ahead past that copy.
func TestALapsedRevivalIsJudgedOnTheLanesChangedFiles(t *testing.T) {
	f := newNarrowFixture(t)
	f.awaitImplementer(t)
	commitIn(t, f.repo, currentLane(t, f.repo, f.runID), "docs/page.md")
	before := f.tip(t)
	lapsed := before.expires.Add(time.Minute)

	f.reserveAs(t, "iss-77", "octo", "docs/page.md", lapsed)
	got := RenewRuns(f.repo.Root(), lapsed.Add(time.Minute))
	if len(got) != 1 || got[0].Renewed || !strings.Contains(got[0].Note, "could not renew the reservation on "+secondIssue) ||
		!strings.Contains(got[0].Note, "docs/page.md") {
		t.Fatalf("a revival over one's other copy's hold of a changed file = %+v; want it refused, naming the file", got)
	}
	if tip := f.tip(t); tip.commit != before.commit {
		t.Fatalf("the refused revival moved the reservation: %+v", tip)
	}

	// The other copy's reservation of the changed file lapses, and it
	// reserves a file of the directory the lane did not change.
	free := lapsed.Add(5*time.Hour + time.Minute)
	f.reserveAs(t, "iss-78", "octo", "docs/guide.md", free)
	at := free.Add(time.Minute)
	got = RenewRuns(f.repo.Root(), at)
	if len(got) != 1 || !got[0].Renewed {
		t.Fatalf("a revival past one's other copy's hold of an unchanged file = %+v; want it renewed", got)
	}
	f.renewedAt(t, before, at)
	if fp := f.footprint(t); !slices.Equal(fp, []string{"docs"}) {
		t.Fatalf("the revival keeps its whole footprint: %q", fp)
	}
	if note := recordNote(t, f.repo, f.runID, StageReservationRenewed); !strings.Contains(note, "docs/guide.md") ||
		!strings.Contains(note, "iss-78") || !strings.Contains(note, "other copy") {
		t.Fatalf("the renewal's note says it went ahead past the other copy: %q", note)
	}
}
