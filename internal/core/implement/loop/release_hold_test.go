package loop

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/lease"
)

// The release hold at the land fence (itd-2609151838312703 C7,
// spc-2610150728396006 step 4): before the landing arms a merge, the fence
// fetches the reservations and refuses, arming nothing, while a release
// reservation another session holds is live. The cutting session's own copy
// is not held: under the concurrent-sessions rule a session cutting a release
// has the final say on what merges before its tag, so its own landing is its
// ruling. Every other copy is held, the same person's other copy included,
// and its pull request stays open for the cutting session to merge or hold.

// reserveRelease reserves the release subject from the working copy dir at
// at, with the default release lease.
func reserveRelease(t *testing.T, dir, subject string, at time.Time) {
	t.Helper()
	if res, err := claim.Reserve(dir, claim.Request{Record: subject}, clockAt(at)); err != nil || res.Act != claim.ActReserved {
		t.Fatalf("reserving %s: %+v %v", subject, res, err)
	}
}

// TestTheFenceHoldsWhileAnotherPersonCutsARelease is C7: Q reserves v0.16.0
// for a cut after this checkout last fetched; P's lane reaches the arming,
// and the fence fetches, names Q and the release, and arms nothing, whether
// the run holds a reservation of its own or not. The run's own reservation,
// renewed by the fence's fetch, still holds.
func TestTheFenceHoldsWhileAnotherPersonCutsARelease(t *testing.T) {
	for _, reserved := range []bool{true, false} {
		name := map[bool]string{true: "a run holding its reservation", false: "a run holding none"}[reserved]
		t.Run(name, func(t *testing.T) {
			f := newLandFixture(t, queueRuleset("MERGE"))
			f.toArm(t)
			now := time.Now().UTC().Truncate(time.Second)
			if reserved {
				reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-30*time.Minute), "internal/widget")
			}
			q := otherCopy(t, f.bare, quinnEmail)
			reserveRelease(t, q, "rel-v0.16.0", now.Add(-time.Minute))
			// This checkout has not fetched Q's reservation: only the fence's
			// own fetch can see it.
			if holds, _, err := claim.ReleaseHolds(f.repo.Root(), now, claim.Options{}); err != nil || len(holds) != 0 {
				t.Fatalf("the checkout already sees the hold before the fence fetched: %+v %v", holds, err)
			}
			r := f.fenced(t)
			want := "release v0.16.0 is being cut by quinn, until " + now.Add(-time.Minute).Add(24*time.Hour).Format(time.RFC3339) +
				", and a session cutting a release has the final say on what merges before its tag, so this lane does not arm its merge"
			if r.Reason != want {
				t.Fatalf("the fence's reason = %q; want %q", r.Reason, want)
			}
			if !strings.Contains(r.Remedy, "the pull request stays open and unarmed") || !strings.Contains(r.Remedy, "quinn") ||
				!strings.Contains(r.Remedy, "v0.16.0") {
				t.Fatalf("the remedy names the holder and the release: %q", r.Remedy)
			}
		})
	}
}

// TestTheCuttingSessionsOwnLaneArms: the copy cutting the release lands its
// own lane; the cut is its to rule on.
func TestTheCuttingSessionsOwnLaneArms(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.toArm(t)
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-30*time.Minute), "internal/widget")
	reserveRelease(t, f.repo.Root(), "rel-v0.16.0", now.Add(-time.Minute))
	f.step(t)
	if l := currentLane(t, f.repo, f.runID); l.Landing == nil || !l.Landing.Armed {
		t.Fatalf("the cutting copy's own lane is armed: %+v", l.Landing)
	}
}

// TestTheCuttingPersonsOtherCopyIsHeld: P cuts the release from another
// computer; P's lane here is another session's, so it is held like anyone's,
// and the refusal says it is P's own other copy.
func TestTheCuttingPersonsOtherCopyIsHeld(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.toArm(t)
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-30*time.Minute), "internal/widget")
	p2 := otherCopy(t, f.bare, patEmail)
	reserveRelease(t, p2, "rel-v0.16.0", now.Add(-time.Minute))
	r := f.fenced(t)
	want := "release v0.16.0 is being cut by your copy on another computer, until " + now.Add(-time.Minute).Add(24*time.Hour).Format(time.RFC3339) +
		", and a session cutting a release has the final say on what merges before its tag, so this lane does not arm its merge"
	if r.Reason != want {
		t.Fatalf("the fence's reason = %q; want %q", r.Reason, want)
	}
}

// TestATaggedOrReleasedCutHoldsNothing: once the release's tag is on origin,
// or its holder released it, the lane arms.
func TestATaggedOrReleasedCutHoldsNothing(t *testing.T) {
	for _, end := range []string{"tagged", "released"} {
		t.Run(end, func(t *testing.T) {
			f := newLandFixture(t, queueRuleset("MERGE"))
			f.toArm(t)
			now := time.Now().UTC().Truncate(time.Second)
			q := otherCopy(t, f.bare, quinnEmail)
			reserveRelease(t, q, "rel-v0.16.0", now.Add(-time.Hour))
			if end == "tagged" {
				gitIn(t, f.bare, "tag", "v0.16.0", "main")
			} else if _, err := claim.Reserve(q, claim.Request{Record: "rel-v0.16.0", Release: true}, clockAt(now.Add(-time.Minute))); err != nil {
				t.Fatal(err)
			}
			f.step(t)
			if l := currentLane(t, f.repo, f.runID); l.Landing == nil || !l.Landing.Armed {
				t.Fatalf("the lane is armed once the cut is %s: %+v", end, l.Landing)
			}
		})
	}
}

// TestTheFenceRefusesWhenItCannotReadTheHold: a run holding no reservation
// still fetches the reservations at the fence, and an origin that cannot be
// read arms nothing, since a hold it cannot see may be there.
func TestTheFenceRefusesWhenItCannotReadTheHold(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.toArm(t)
	if err := os.Rename(f.bare, f.bare+".gone"); err != nil {
		t.Fatal(err)
	}
	r := f.fenced(t)
	if !strings.HasPrefix(r.Reason, "could not read the reservations from origin: ") ||
		!strings.HasSuffix(r.Reason, ", so no release hold could be judged and the merge is not armed") ||
		r.Remedy != "run abcd implement step again once origin answers" {
		t.Fatalf("the fence names the failed fetch: %+v", r)
	}
}

// TestACheckoutWithNoOriginIsNotHeld: a checkout with no remote named origin
// has no reservations to read, so the fence holds nothing, even with a
// release reservation left over from before origin was removed; the same
// leftover with origin there holds the lane.
func TestACheckoutWithNoOriginIsNotHeld(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	now := time.Now().UTC().Truncate(time.Second)
	q := otherCopy(t, f.bare, quinnEmail)
	reserveRelease(t, q, "rel-v0.16.0", now.Add(-time.Minute))
	root := f.repo.Root()
	c := Context{RepoRoot: root, Now: now}
	lane := Lane{ID: "lane-1"}
	if _, err := landFence(c, lane, lease.Max); err == nil {
		t.Fatal("with origin there, the fence holds nothing; want Q's release to hold the lane")
	}
	ref := claim.TrackingPrefix + "rel-v0.16.0"
	tip := gitIn(t, root, "rev-parse", ref)
	gitIn(t, root, "remote", "remove", claim.Remote)
	// Removing the remote drops its tracking refs; one put back by hand is
	// the hold a stale copy might still carry.
	gitIn(t, root, "update-ref", ref, tip)
	if exp, err := landFence(c, lane, lease.Max); err != nil || !exp.IsZero() {
		t.Fatalf("the fence with no origin = %s, %v; want nothing held and no renewal", exp, err)
	}
}
