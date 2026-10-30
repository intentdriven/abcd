package loop

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
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
// so RenewRuns, and RenewRunsIn (implement status's) alike, passes over it
// silently once the lock's bounded wait (lockTimeout) is up, pushing and
// logging nothing, and never waits on the lock past that bound.
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
	due := before.claimed.Add(61 * time.Minute)
	for name, renew := range map[string]func() []RunRenewal{
		"RenewRuns":   func() []RunRenewal { return RenewRuns(f.repo.Root(), due) },
		"RenewRunsIn": func() []RunRenewal { return RenewRunsIn(f.repo.Root(), "", due) },
	} {
		begin := time.Now()
		got := renew()
		if took := time.Since(begin); took > 10*lockTimeout+5*time.Second {
			t.Errorf("%s waited %s on the held lock; want it bounded by the lock's wait", name, took)
		}
		if len(got) != 0 {
			t.Errorf("%s over a held lock = %+v; want the run passed over silently", name, got)
		}
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if tip := f.tip(t); tip.commit != before.commit {
		t.Fatalf("a renewal was pushed under another invocation's lock: %+v", tip)
	}
	if after := recordStages(t, f.repo, f.runID); !slices.Equal(after, logged) {
		t.Fatalf("the run record changed under another invocation's lock: %v, was %v", after, logged)
	}
}

// TestALapsedRevivalAtValidateAwaitingItsReviewersIsJudgedWhole
// (dcn-2610300422481486): the run's reservation of the directory docs lapsed
// while its reviewers judged the lane, its implementer done and its work
// committed. One verdict from them hands a fix implementer any file of the
// found_at, so the lane is not past editing: the revival is judged on the
// whole footprint and backs off the same person's other copy's reservation
// of docs/guide.md, a file the lane's branch did not change.
func TestALapsedRevivalAtValidateAwaitingItsReviewersIsJudgedWhole(t *testing.T) {
	f := newNarrowFixture(t)
	f.implement(t, Options{}, "docs/page.md")
	if l := f.answerUntil(t, nil, func(l Lane) bool { return l.awaitsRole(RoleRuthless) }); l.Stage != StageValidate || len(l.Awaits) != 1 {
		t.Fatalf("the lane awaits its reviewer alone: stage %s, awaits %+v", l.Stage, l.Awaits)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if got := revivalChangedFiles(f.repo.Root(), st); got != nil {
		t.Fatalf("a revival with a lane at validate is judged on %q; want the whole footprint", got)
	}
	f.revivalBacksOffOnGuide(t, "at validate awaiting its reviewers")
}

// TestALapsedRevivalAtLandIsJudgedOnTheLanesChangedFiles: the run's
// reservation of the directory docs lapsed with its lane at its landing, its
// validators passed and no sync pending, and the same person's other copy
// reserved a file in it since. The revival is judged on the files the lane's
// branch changed: it backs off that copy's reservation of a changed file,
// logging the failed renewal, and once that copy holds only a file the lane
// did not change, it goes ahead, narrowed to the lane's changed file as the
// landing's renewal narrows (narrowedFootprint), so that file is outside it.
func TestALapsedRevivalAtLandIsJudgedOnTheLanesChangedFiles(t *testing.T) {
	f := newNarrowFixture(t)
	f.implement(t, Options{}, "docs/page.md")
	f.toLand(t)
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if got := revivalChangedFiles(f.repo.Root(), st); !slices.Equal(got, []string{"docs/page.md"}) {
		t.Fatalf("a revival with its lane at land is judged on %q; want the lane's changed docs/page.md", got)
	}
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
	// An issue run's revival at its landing narrows as the landing's renewal
	// does (dcn-2610300615451659), so the other copy's file is outside it and
	// nothing was gone ahead past.
	if fp := f.footprint(t); !slices.Equal(fp, []string{"docs/page.md"}) {
		t.Fatalf("the revival narrows the reservation to the lane's changed file: %q", fp)
	}
	note := recordNote(t, f.repo, f.runID, StageReservationRenewed)
	if !strings.Contains(note, "its footprint narrowed from docs to the files its branch changed: docs/page.md") ||
		strings.Contains(note, "revived past") {
		t.Fatalf("the run record logs the narrowing and no other copy gone past: %q", note)
	}
}

// TestALapsedIntentRevivalAtLandGoesPastItsOtherCopyOnAnUnchangedFile
// (dcn-2610292238191496, dcn-2610300615451659): an intent run's reservation of
// the directory docs and one.txt lapsed with its one lane at its landing,
// having changed one.txt alone, and the same person's other copy reserved
// docs/guide.md since. An intent run never narrows, so the revival is judged
// against that copy on the lane's changed files only: it goes ahead past
// docs/guide.md, keeps its whole footprint, and the run record names the
// reservation it went past. Judged on the whole footprint, it would back off.
func TestALapsedIntentRevivalAtLandGoesPastItsOtherCopyOnAnUnchangedFile(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.validated(t)
	reserveRun(t, f.repo, f.runID, "itd-10", time.Now().UTC().Truncate(time.Second), "docs", "one.txt")
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if got := revivalChangedFiles(f.repo.Root(), st); !slices.Equal(got, []string{"one.txt"}) {
		t.Fatalf("a revival with its lane at land is judged on %q; want the lane's changed one.txt", got)
	}
	ref := claim.RefPrefix + "itd-10"
	before := originReservation(t, f.bare, ref)
	lapsed := before.expires.Add(time.Minute)
	(&narrowFixture{bare: f.bare}).reserveAs(t, "iss-77", "octo", "docs/guide.md", lapsed)

	at := lapsed.Add(time.Minute)
	got := RenewRuns(f.repo.Root(), at)
	if len(got) != 1 || !got[0].Renewed {
		t.Fatalf("an intent run's revival past its other copy's hold of an unchanged file = %+v; want it renewed", got)
	}
	if tip := originReservation(t, f.bare, ref); tip.parent != before.commit || !tip.claimed.Equal(at) {
		t.Fatalf("the revival is pushed over the lapsed tip at %s: before %+v, after %+v", stamp(at), before, tip)
	}
	if fp := originFootprint(t, f.bare, "itd-10"); !slices.Equal(fp, []string{"docs", "one.txt"}) {
		t.Fatalf("an intent run's revival keeps its whole footprint: %q", fp)
	}
	if note := recordNote(t, f.repo, f.runID, StageReservationRenewed); !strings.Contains(note, "revived past your other copy's reservation of docs/guide.md (iss-77)") {
		t.Fatalf("the run record names the other copy's reservation the revival went past: %q", note)
	}
}

// TestALapsedRevivalWhileTheImplementerWorksIsJudgedWhole: the run's
// reservation of the directory docs lapsed while its implementer worked, its
// edit of docs/page.md not yet committed, so its branch's tip is still its
// base and names no changed file; the same person's other copy reserved that
// file since. The revival is judged on the whole footprint, not on the
// branch's empty list, and backs off, since two automated lanes must never
// edit one file (dcn-2610291925508286).
func TestALapsedRevivalWhileTheImplementerWorksIsJudgedWhole(t *testing.T) {
	f := newNarrowFixture(t)
	f.awaitImplementer(t)
	l := currentLane(t, f.repo, f.runID)
	if err := os.WriteFile(filepath.Join(l.Worktree, "docs", "page.md"), []byte("the page, half edited\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	before := f.tip(t)
	lapsed := before.expires.Add(time.Minute)

	f.reserveAs(t, "iss-77", "octo", "docs/page.md", lapsed)
	got := RenewRuns(f.repo.Root(), lapsed.Add(time.Minute))
	if len(got) != 1 || got[0].Renewed || !strings.Contains(got[0].Note, "could not renew the reservation on "+secondIssue) ||
		!strings.Contains(got[0].Note, "docs/page.md") {
		t.Fatalf("a revival while the implementer's edit is uncommitted = %+v; want it refused on the other copy's file", got)
	}
	if tip := f.tip(t); tip.commit != before.commit {
		t.Fatalf("the refused revival moved the reservation: %+v", tip)
	}
}

// TestTheWaiterLeavesARunParkedAtLandToTheDrain: a drain lane parked at
// land, waiting on its pull request, is renewed by the drain's move, which
// reads that pull request first (drain_finish.go); neither the waiter
// (RenewRuns, from the run's checkout or the lane's worktree) nor implement
// status (RenewRunsIn) renews it, though it is half through its lease, and
// nothing is pushed or logged.
func TestTheWaiterLeavesARunParkedAtLandToTheDrain(t *testing.T) {
	f, l := parkDrainLane(t)
	ref := claim.RefPrefix + secondIssue
	fenced := originReservation(t, f.bare, ref)
	at := fenced.claimed.Add(12*time.Hour + 30*time.Minute)
	logged := recordStages(t, f.repo, f.runID)

	for name, got := range map[string][]RunRenewal{
		"RenewRuns from the run's checkout":    RenewRuns(f.repo.Root(), at),
		"RenewRuns from the lane's worktree":   RenewRuns(l.Worktree, at),
		"RenewRunsIn the run's checkout":       RenewRunsIn(f.repo.Root(), "", at),
		"RenewRunsIn the run's checkout by id": RenewRunsIn(f.repo.Root(), f.runID, at),
	} {
		if len(got) != 0 {
			t.Errorf("%s = %+v; want a run parked at land left to the drain", name, got)
		}
	}
	if tip := originReservation(t, f.bare, ref); tip.commit != fenced.commit {
		t.Fatalf("a run parked at land was renewed outside the drain: fenced %+v, now %+v", fenced, tip)
	}
	if after := recordStages(t, f.repo, f.runID); !slices.Equal(after, logged) {
		t.Fatalf("the parked run's record changed: %v, was %v", after, logged)
	}
}

// TestALapseIsRevivedOnceThoughTheViewWasReadBeforeAnotherRevival: two
// waiters read the reservation view, both finding the run's reservation
// lapsed, and the first revives it while the second waits for the run's
// lock. The second re-reads the view under the lock, finds the reservation
// just revived, and pushes and logs nothing.
func TestALapseIsRevivedOnceThoughTheViewWasReadBeforeAnotherRevival(t *testing.T) {
	f := newNarrowFixture(t)
	f.awaitImplementer(t)
	before := f.tip(t)
	at := before.expires.Add(time.Minute)

	first := true
	runsView = func(root string, now time.Time) (claim.Snapshot, error) {
		snap, err := claim.View(root, now)
		if first {
			first = false
			if got := RenewRunsIn(f.repo.Root(), "", now); len(got) != 1 || !got[0].Renewed {
				t.Errorf("the first waiter's revival = %+v; want it renewed", got)
			}
		}
		return snap, err
	}
	t.Cleanup(func() { runsView = claim.View })

	if got := RenewRunsIn(f.repo.Root(), "", at); len(got) != 0 {
		t.Fatalf("the second waiter, its view read before the first revival = %+v; want nothing renewed", got)
	}
	f.renewedAt(t, before, at)
	n := 0
	for _, s := range recordStages(t, f.repo, f.runID) {
		if s == StageReservationRenewed {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("the run record logs %d renewals; want one", n)
	}
}

// answerUntil steps the run, answering each agent it hands out whose role
// returns names with that return, until stop holds of the current lane, which
// it returns.
func (f *narrowFixture) answerUntil(t *testing.T, returns map[string]string, stop func(Lane) bool) Lane {
	t.Helper()
	for range 4 * len(Sequence) {
		l := currentLane(t, f.repo, f.runID)
		if stop(l) {
			return l
		}
		if a := l.awaiting(); a != nil && returns[a.Role] != "" {
			path := filepath.Join(f.repo.Root(), filepath.FromSlash(a.Receipt))
			if err := os.WriteFile(path, []byte(returns[a.Role]), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Receipt(f.repo.Root(), f.runID, path, f.stages, Options{}); err != nil {
				t.Fatalf("the %s's return: %v", a.Role, err)
			}
			continue
		}
		if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err != nil {
			t.Fatalf("stepping the run: %v", err)
		}
	}
	t.Fatal("the lane never reached the state the test waits for")
	return Lane{}
}

// revivalBacksOffOnGuide reports, failing t otherwise, that the lapsed
// revival of the run's reservation is refused over the same person's other
// copy's reservation of docs/guide.md, a file the lane's branch does not
// change, and that the reservation does not move: the whole footprint was
// judged, not the branch's changed files (dcn-2610292238191496).
func (f *narrowFixture) revivalBacksOffOnGuide(t *testing.T, what string) {
	t.Helper()
	before := f.tip(t)
	lapsed := before.expires.Add(time.Minute)
	f.reserveAs(t, "iss-77", "octo", "docs/guide.md", lapsed)
	got := RenewRuns(f.repo.Root(), lapsed.Add(time.Minute))
	if len(got) != 1 || got[0].Renewed || !strings.Contains(got[0].Note, "could not renew the reservation on "+secondIssue) ||
		!strings.Contains(got[0].Note, "docs/guide.md") {
		t.Fatalf("a revival %s = %+v; want it refused on the other copy's docs/guide.md", what, got)
	}
	if tip := f.tip(t); tip.commit != before.commit {
		t.Fatalf("the refused revival moved the reservation: %+v", tip)
	}
}

// TestALapsedRevivalBeforeAFixRoundIsJudgedWhole: the lane's reviewer asked
// for a fix, so it waits at validate for a fresh implementer, whose edits stay
// uncommitted until its receipt and may reach any file of the found_at. A
// lapsed revival, whether the fix round's implementer is out, editing
// docs/guide.md, or its findings are yet to be handed out, is judged on the
// whole footprint and backs off the same person's other copy's reservation of
// docs/guide.md, though the branch changed only docs/page.md.
func TestALapsedRevivalBeforeAFixRoundIsJudgedWhole(t *testing.T) {
	fixFirst := map[string]string{RoleRuthless: reviewerReturn("FIX FIRST"), RoleSecurity: reviewerReturn("APPROVE")}
	for name, stop := range map[string]func(Lane) bool{
		"while the fix round's implementer works": func(l Lane) bool { return l.awaitsRole(RoleImplementer) },
		"before the fix round's findings are handed out": func(l Lane) bool {
			n := len(l.Validation)
			if n == 0 || len(l.Awaits) > 0 {
				return false
			}
			for _, v := range l.Validation[n-1].Validators {
				if v.Verdict == "" {
					return false
				}
			}
			return true
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newNarrowFixture(t)
			f.implement(t, Options{}, "docs/page.md")
			l := f.answerUntil(t, fixFirst, stop)
			if l.Stage != StageValidate {
				t.Fatalf("the fix round waits at validate: %s", l.Stage)
			}
			if err := os.WriteFile(filepath.Join(l.Worktree, "docs", "guide.md"), []byte("the guide, half fixed\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			f.revivalBacksOffOnGuide(t, name)
		})
	}
}

// TestALapsedRevivalWithASyncConflictPendingIsJudgedWhole: the lane's
// landing synced with the default branch and conflicted, which sends it back
// to validate for a fresh implementer to resolve; that implementer's edits
// stay uncommitted until its receipt. A lapsed revival then is judged on the
// whole footprint and backs off.
func TestALapsedRevivalWithASyncConflictPendingIsJudgedWhole(t *testing.T) {
	f := newNarrowFixture(t)
	f.implement(t, Options{}, "docs/page.md")
	l := currentLane(t, f.repo, f.runID)
	editState(t, f.repo.Root(), f.runID, func(doc map[string]any) {
		lane := doc["lanes"].([]any)[0].(map[string]any)
		lane["syncs"] = []any{map[string]any{
			"at": stamp(time.Now().UTC()), "siblings": []any{}, "merged": l.HeadSHA, "conflicted": true,
			"paths": []any{"docs/guide.md"}, "brief": "sync-1/brief.md", "receipt": "sync-1/receipt.json",
		}}
	})
	if l := currentLane(t, f.repo, f.runID); l.Stage != StageValidate || l.pendingSync() == nil || len(l.Awaits) != 0 {
		t.Fatalf("the lane waits at validate on its sync conflict, its implementer not yet out: %+v", l)
	}
	f.revivalBacksOffOnGuide(t, "with a sync conflict pending")
}

// TestALapsedRevivalWithAHeldLaneIsJudgedWhole: a lane in a state the
// allow-list does not name, held here, counts as one that may yet edit, so a
// lapsed revival is judged on the whole footprint and backs off.
func TestALapsedRevivalWithAHeldLaneIsJudgedWhole(t *testing.T) {
	f := newNarrowFixture(t)
	f.implement(t, Options{}, "docs/page.md")
	editState(t, f.repo.Root(), f.runID, func(doc map[string]any) {
		doc["lanes"].([]any)[0].(map[string]any)["stage"] = string(StageHeld)
	})
	if l := currentLane(t, f.repo, f.runID); l.Stage != StageHeld {
		t.Fatalf("the lane is held: %s", l.Stage)
	}
	f.revivalBacksOffOnGuide(t, "with a held lane")
}
