package loop

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/lease"
	"github.com/intentdriven/abcd/internal/gittest"
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
	if res, err := claim.Reserve(dir, claim.Request{Record: subject}, as(t, dir, clockAt(at))); err != nil || res.Act != claim.ActReserved {
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
			if holds, err := claim.ReleaseHoldsForTest(f.repo.Root(), now, claim.Options{}); err != nil || len(holds) != 0 {
				t.Fatalf("the checkout already sees the hold before the fence fetched: %+v %v", holds, err)
			}
			r := f.fenced(t)
			want := "release v0.16.0 is being cut by " + heldBy(t, quinnEmail) + " (claimed), until " + now.Add(-time.Minute).Add(24*time.Hour).Format(time.RFC3339) +
				", and a session cutting a release has the final say on what merges before its tag, so this lane does not arm its merge"
			if r.Reason != want {
				t.Fatalf("the fence's reason = %q; want %q", r.Reason, want)
			}
			if !strings.Contains(r.Remedy, "the pull request stays open and unarmed") || !strings.Contains(r.Remedy, "held by "+heldBy(t, quinnEmail)+" from that copy") ||
				strings.Contains(r.Remedy, "ask ") ||
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
			} else if _, err := claim.Reserve(q, claim.Request{Record: "rel-v0.16.0", Release: true}, as(t, q, clockAt(now.Add(-time.Minute)))); err != nil {
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
	if _, _, err := landFence(c, lane, lease.Max); err == nil {
		t.Fatal("with origin there, the fence holds nothing; want Q's release to hold the lane")
	}
	ref := claim.TrackingPrefix + "rel-v0.16.0"
	tip := gitIn(t, root, "rev-parse", ref)
	gitIn(t, root, "remote", "remove", claim.Remote)
	// Removing the remote drops its tracking refs; one put back by hand is
	// the hold a stale copy might still carry.
	gitIn(t, root, "update-ref", ref, tip)
	if exp, _, err := landFence(c, lane, lease.Max); err != nil || !exp.IsZero() {
		t.Fatalf("the fence with no origin = %s, %v; want nothing held and no renewal", exp, err)
	}
}

// pushDatedTip pushes a commit carrying msg on the empty tree, dated at, as
// the reservation branch for subject on origin, from a scratch clone: a
// reservation written on a clock far ahead of this one.
func pushDatedTip(t *testing.T, bare, subject, msg string, at time.Time) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "ahead")
	gitIn(t, "", "clone", "-q", bare, dir)
	path := filepath.Join(t.TempDir(), "msg")
	if err := os.WriteFile(path, []byte(msg), 0o600); err != nil {
		t.Fatal(err)
	}
	tree := gitIn(t, dir, "hash-object", "-w", "-t", "tree", "/dev/null")
	date := "@" + strconv.FormatInt(at.Unix(), 10) + " +0000"
	cmd := exec.Command("git", "-C", dir, "-c", "user.name=Quinn", "-c", "user.email=quinn@example.invalid", "-c", "commit.gpgsign=false",
		"commit-tree", tree, "-F", path)
	cmd.Env = append(gittest.Env(t), "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "push", "-q", "origin", strings.TrimSpace(string(out))+":"+claim.RefPrefix+subject)
}

// TestTheFenceCountsAReleaseDatedAheadFromItsFirstSighting: a lane holding
// its own reservation renews it at the fence, and the renewal's fetch brings a
// release reservation dated five years ahead (readable, or unreadable with
// that commit date). The fence holds the lane from its first sighting of it
// for a day (or the default hold) and then arms, however many times it is
// judged in between; the hold never slides forward with the clock.
func TestTheFenceCountsAReleaseDatedAheadFromItsFirstSighting(t *testing.T) {
	for _, c := range []struct {
		name string
		msg  func(ahead time.Time) string
		hold time.Duration
	}{
		{"readable", func(ahead time.Time) string {
			return "abcd reservation: rel-v0.16.0\n\nAbcd-Reservation: rel-v0.16.0\nAbcd-Holder: quinn\n" +
				"Abcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55\nAbcd-Claimed-At: " + stamp(ahead) +
				"\nAbcd-Expires-At: " + stamp(ahead.Add(lease.Max)) + "\n"
		}, lease.Max},
		{"unreadable", func(time.Time) string { return "abcd reservation: rel-v0.16.0\n\nnot a reservation\n" }, lease.Default},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newLandFixture(t, queueRuleset("MERGE"))
			now := time.Now().UTC().Truncate(time.Second)
			reserveRun(t, f.repo, f.runID, "itd-10", now, "internal/widget")
			ahead := now.Add(5 * 365 * 24 * time.Hour)
			pushDatedTip(t, f.bare, "rel-v0.16.0", c.msg(ahead), ahead)
			lane := Lane{ID: "lane-1"}
			st := State{Key: "itd-10", ReservedAt: &now}
			until := stamp(now.Add(c.hold))
			for _, at := range []time.Duration{0, c.hold / 2, c.hold - time.Minute} {
				_, _, err := landFence(Context{RepoRoot: f.repo.Root(), Now: now.Add(at), State: st}, lane, lease.Max)
				r, ok := AsRefusal(err)
				if !ok || !r.Contention || !strings.Contains(r.Reason, "release v0.16.0") || !strings.Contains(r.Reason, "until "+until) {
					t.Fatalf("at +%s: %v; want the lane held by v0.16.0 until %s", at, err, until)
				}
			}
			exp, _, err := landFence(Context{RepoRoot: f.repo.Root(), Now: now.Add(c.hold), State: st}, lane, lease.Max)
			if err != nil || exp.IsZero() {
				t.Fatalf("at +%s: %s, %v; want the hold ended %s after it was first seen and the reservation renewed", c.hold, exp, err, c.hold)
			}
		})
	}
}

// TestTheFenceArmsPastAReleaseDatedAheadItCannotKeepASightingOf: where this
// copy cannot write a first sighting, a release dated five years ahead would
// hold the lane from every reading, so the fence does not count it: the lane
// arms, and the arming's note names the release that was not counted.
func TestTheFenceArmsPastAReleaseDatedAheadItCannotKeepASightingOf(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.toArm(t)
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-30*time.Minute), "internal/widget")
	undated := filepath.Join(f.repo.Root(), ".abcd", ".work.local", "reservation", "undated")
	if err := os.RemoveAll(undated); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(undated, []byte("not a directory\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ahead := now.Add(5 * 365 * 24 * time.Hour)
	pushDatedTip(t, f.bare, "rel-v0.16.0", "abcd reservation: rel-v0.16.0\n\nAbcd-Reservation: rel-v0.16.0\nAbcd-Holder: quinn\n"+
		"Abcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55\nAbcd-Claimed-At: "+stamp(ahead)+"\nAbcd-Expires-At: "+stamp(ahead.Add(lease.Max))+"\n", ahead)
	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	if l.Landing == nil || !l.Landing.Armed {
		t.Fatalf("the lane is armed past a release it cannot count: %+v", l.Landing)
	}
	want := "release v0.16.0 is reserved by quinn with a date more than 5 minutes ahead of this clock, and this checkout cannot keep when it first saw it"
	if note := recordNote(t, f.repo, f.runID, string(StageLand)); !strings.Contains(note, want) {
		t.Fatalf("the arming's note = %q; want it to name %q", note, want)
	}
}

// toPush takes a validated lane through its records to the push, with its
// preflight receipt minted, stopping before the push fence.
func (f *landFixture) toPush(t *testing.T) Lane {
	t.Helper()
	f.validated(t)
	f.step(t)
	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	preflighted(t, l, l.HeadSHA)
	return l
}

// TestThePushFenceHoldsWhileAnotherPersonCutsARelease is C7 at the push: the
// push fence makes the land fence's release-hold judgement too, as `abcd
// land` judges it before it pushes, so a lane reaching its push while Q cuts
// a release pushes nothing and opens no pull request, whether the run holds a
// reservation of its own or not.
func TestThePushFenceHoldsWhileAnotherPersonCutsARelease(t *testing.T) {
	for _, reserved := range []bool{true, false} {
		name := map[bool]string{true: "a run holding its reservation", false: "a run holding none"}[reserved]
		t.Run(name, func(t *testing.T) {
			f := newLandFixture(t, queueRuleset("MERGE"))
			l := f.toPush(t)
			now := time.Now().UTC().Truncate(time.Second)
			if reserved {
				reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-30*time.Minute), "internal/widget")
			}
			q := otherCopy(t, f.bare, quinnEmail)
			reserveRelease(t, q, "rel-v0.16.0", now.Add(-time.Minute))
			_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
			r, ok := AsRefusal(err)
			if !ok || !r.Contention || r.Stage != string(StageLand) {
				t.Fatalf("the push fence refuses the push as contention: %v", err)
			}
			want := "release v0.16.0 is being cut by " + heldBy(t, quinnEmail) + " (claimed), until " + now.Add(-time.Minute).Add(24*time.Hour).Format(time.RFC3339) +
				", and a session cutting a release has the final say on what merges before its tag, so this lane does not push its branch"
			if r.Reason != want {
				t.Fatalf("the push fence's reason = %q; want %q", r.Reason, want)
			}
			// A release cut is a pause someone chose, so a drain never counts
			// it towards handing the lane back as stuck.
			if !waitByDesign(r) {
				t.Fatalf("a release hold is a wait by design, never a refusal a drain counts: %+v", r)
			}
			if !strings.Contains(r.Remedy, "nothing is pushed and no pull request is opened") || !strings.Contains(r.Remedy, "held by "+heldBy(t, quinnEmail)+" from that copy") ||
				strings.Contains(r.Remedy, "ask ") {
				t.Fatalf("the remedy names what stays and the holder: %q", r.Remedy)
			}
			if got := f.remoteBranch(t, l.Branch); got != "" {
				t.Fatalf("nothing is pushed: origin holds %s at %s", l.Branch, got)
			}
			if l := currentLane(t, f.repo, f.runID); l.Landing == nil || l.Landing.Pushed != "" || l.PR != 0 {
				t.Fatalf("the lane still waits to push: %+v", l)
			}
		})
	}
}

// TestThePushFenceWaitsWhenItCannotReadTheHold: a run holding no reservation
// still fetches the reservations at the push fence, for the hold, and an
// origin it cannot read pushes nothing; like every fence fetch that fails, it
// is a wait by design, never a refusal a drain counts (dcn-2610170116582725).
func TestThePushFenceWaitsWhenItCannotReadTheHold(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.toPush(t)
	if err := os.Rename(f.bare, f.bare+".gone"); err != nil {
		t.Fatal(err)
	}
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	r, ok := AsRefusal(err)
	if !ok || !r.Contention || !waitByDesign(r) {
		t.Fatalf("the push fence waits on an unread origin: %v", err)
	}
	if !strings.HasPrefix(r.Reason, "could not read the reservations from origin: ") ||
		!strings.HasSuffix(r.Reason, ", so no release hold could be judged and nothing is pushed") ||
		r.Remedy != "run abcd implement step again once origin answers" {
		t.Fatalf("the push fence names the failed fetch: %+v", r)
	}
}
