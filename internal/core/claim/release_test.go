package claim

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/lease"
)

// The release hold (spc-2610150728396006 step 4, C7 and C8): a session
// cutting a release reserves the release, `rel-vX.Y.Z`, as it would a record.
// The release subject takes no footprint, sits outside the live cap, holds
// for a day unless a lease is named, and ends when the tag vX.Y.Z is on
// origin or its holder releases it.

// tagOrigin tags origin's main as tag, as the release workflow does once the
// release pull request has merged.
func (w *world) tagOrigin(tag string) {
	w.t.Helper()
	runGit(w.t, w.bare, "tag", tag, "main")
}

// TestAReleaseIsReservedWithoutAFootprintForADay: the release subject needs
// no footprint, and with no lease named it holds for lease.Max, so a slow
// cut outlasts its own CI; it is one parentless empty-tree commit on its own
// branch, like any reservation.
func TestAReleaseIsReservedWithoutAFootprintForADay(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	res, err := p.do(Request{Record: "rel-v0.16.0"})
	if err != nil {
		t.Fatalf("reserving the release: %v", err)
	}
	if res.Act != ActReserved || res.Record != "rel-v0.16.0" || len(res.Footprint) != 0 || !res.ExpiresAt.Equal(t0.Add(lease.Max)) {
		t.Fatalf("result = %+v; want rel-v0.16.0 reserved with no footprint until %s", res, t0.Add(lease.Max))
	}
	tip := w.remoteTip("rel-v0.16.0")
	if tip == "" || tip != res.Commit || w.parentOf(tip) != "" {
		t.Fatalf("origin's abcd-reserved/rel-v0.16.0 = %q; want the parentless commit %s", tip, res.Commit)
	}
	msg := w.remoteMessage(tip)
	if !strings.Contains(msg, "Abcd-Reservation: rel-v0.16.0\n") || strings.Contains(msg, keyFootprint) {
		t.Fatalf("message = %q; want the release subject and no footprint", msg)
	}
	// A lease named is kept, within the bounds every reservation has.
	w.set(t0.Add(time.Hour))
	if res, err := p.do(Request{Record: "rel-v0.16.0", Lease: 3 * time.Hour}); err != nil || res.Act != ActRenewed || !res.ExpiresAt.Equal(t0.Add(4*time.Hour)) {
		t.Fatalf("renewing with a 3h lease = %+v, %v; want renewed until %s", res, err, t0.Add(4*time.Hour))
	}
	if _, err := p.do(Request{Record: "rel-v0.16.0", Lease: 25 * time.Hour}); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "outside 1m0s to 24h0m0s") {
		t.Fatalf("a lease past a day = %v; want the bound refusal: a release holds no longer than any reservation", err)
	}
	// Running the same command again from the same copy renews it for a day.
	w.set(t0.Add(20 * time.Hour))
	if res, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil || res.Act != ActRenewed || !res.ExpiresAt.Equal(t0.Add(44*time.Hour)) {
		t.Fatalf("renewing = %+v, %v; want renewed until %s", res, err, t0.Add(44*time.Hour))
	}
}

// TestAReleaseTakesNoFootprintAndOnlyItsOwnGrammar: a footprint on a release
// is refused (a release holds every merge, not files), and the subject is
// rel-v followed by three plain numbers, never a spelling that would make a
// second branch for one release.
func TestAReleaseTakesNoFootprintAndOnlyItsOwnGrammar(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.reserve("rel-v0.16.0", "a.go"); !errors.Is(err, ErrRefused) ||
		err.Error() != "a release reservation holds every merge and takes no footprint; nothing was pushed" {
		t.Fatalf("a release with a footprint = %v; want the footprint refusal", err)
	}
	for _, bad := range []string{"rel-0.16.0", "rel-v0.16", "rel-v01.2.3", "REL-v1.2.3", "rel-V1.2.3", "rel-v1.2.3-rc1", "rel-v1.2.3/x", "rel-"} {
		_, err := p.do(Request{Record: bad})
		want := fmt.Sprintf("%q is not an issue or intent id (iss-N or itd-N), nor a release (rel-vX.Y.Z); nothing was pushed", bad)
		if !errors.Is(err, ErrRefused) || err.Error() != want {
			t.Errorf("reserving %q = %v; want %q", bad, err, want)
		}
	}
	if out := runGit(t, w.bare, "for-each-ref", RefPrefix); out != "" {
		t.Fatalf("a refused release reached origin: %s", out)
	}
}

// TestAReleaseSitsOutsideTheLiveCap: a person holding the ten live
// reservations the cap allows can still reserve a release, and a release held
// does not count toward the ten, nor toward the live count a listing shows.
func TestAReleaseSitsOutsideTheLiveCap(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	for i := range MaxLive {
		if _, err := p.reserve(fmt.Sprintf("iss-%d", 100+i), fmt.Sprintf("f%d.go", i)); err != nil {
			t.Fatalf("reservation %d beside a release: %v", i+1, err)
		}
	}
	// One release is cut at a time: the first ends before the next is held.
	if _, err := p.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.do(Request{Record: "rel-v0.17.0"}); err != nil {
		t.Fatalf("a release past ten live reservations: %v", err)
	}
	if _, err := p.reserve("iss-200", "g.go"); !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "you hold 10 live reservations") ||
		strings.Contains(err.Error(), "rel-") {
		t.Fatalf("an eleventh record = %v; want the cap, naming the ten records and no release", err)
	}
	snap, err := View(p.dir, t0)
	if err != nil {
		t.Fatal(err)
	}
	if got := snap.LiveByHolder(); got["octo"] != MaxLive || len(got) != 1 {
		t.Fatalf("live by holder = %v; want octo %d, the releases not counted", got, MaxLive)
	}
}

// TestAReleaseHoldIsShownAndEndsAtItsTag is C8 and C7's end: a live release
// is a hold the view reports, with the line every surface shows, and it ends
// when the tag is on origin as last fetched. A tag only this checkout has does
// not end it.
func TestAReleaseHoldIsShownAndEndsAtItsTag(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Fetch(q.dir, q.opts()); err != nil {
		t.Fatal(err)
	}
	holds, err := releaseHolds(q.dir, t0.Add(time.Hour), q.opts())
	if err != nil || len(holds) != 1 {
		t.Fatalf("Q's holds = %+v, %v; want P's release", holds, err)
	}
	h := holds[0]
	if h.Release != "v0.16.0" || h.Subject != "rel-v0.16.0" || h.Holder != "octo" || h.Own || !h.EffectiveExpiry.Equal(t0.Add(lease.Max)) {
		t.Fatalf("hold = %+v; want v0.16.0 held by octo, not Q's own, until %s", h, t0.Add(lease.Max))
	}
	if want := "release v0.16.0 being cut by @octo: merges are held; the hold lapses at " + stamp(t0.Add(lease.Max)) +
		" unless the cut renews it"; h.Text() != want {
		t.Fatalf("the hold line = %q; want %q", h.Text(), want)
	}

	// A tag Q made locally is not the tag on origin.
	q.git("tag", "v0.16.0")
	if _, err := Fetch(q.dir, q.opts()); err != nil {
		t.Fatal(err)
	}
	if holds, _ := releaseHolds(q.dir, t0.Add(time.Hour), q.opts()); len(holds) != 1 {
		t.Fatalf("a local tag ended the hold: %+v", holds)
	}

	// The tag on origin ends it, once fetched.
	w.tagOrigin("v0.16.0")
	if holds, _ := releaseHolds(q.dir, t0.Add(time.Hour), q.opts()); len(holds) != 1 {
		t.Fatalf("the hold ended before any fetch saw the tag: %+v", holds)
	}
	if _, err := Fetch(q.dir, q.opts()); err != nil {
		t.Fatal(err)
	}
	holds, err = releaseHolds(q.dir, t0.Add(time.Hour), q.opts())
	if err != nil || len(holds) != 0 {
		t.Fatalf("holds once the tag is on origin = %+v, %v; want none", holds, err)
	}
	snap, err := View(q.dir, t0.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	r, ok := snap.Lookup("rel-v0.16.0")
	if !ok || r.State != StateReleased || r.ReleasedBy != ReleasedByTag {
		t.Fatalf("the release reservation = %+v; want released by its tag", r)
	}
	// A tag origin drops again is pruned with the reservations, and the hold returns.
	runGit(t, w.bare, "tag", "-d", "v0.16.0")
	if _, err := Fetch(q.dir, q.opts()); err != nil {
		t.Fatal(err)
	}
	if holds, _ := releaseHolds(q.dir, t0.Add(time.Hour), q.opts()); len(holds) != 1 {
		t.Fatalf("a tag origin no longer holds still ends the hold: %+v", holds)
	}
}

// TestAReleaseAlreadyTaggedIsNotReserved: there is no cut to hold once the
// tag is on origin, so a reservation of it is refused before anything is
// pushed, and a renewal of one already made is refused as released.
func TestAReleaseAlreadyTaggedIsNotReserved(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	w.tagOrigin("v0.15.0")
	_, err := p.do(Request{Record: "rel-v0.15.0"})
	if want := "v0.15.0 is already tagged on origin, so there is no cut to hold; nothing was pushed"; !errors.Is(err, ErrRefused) || err.Error() != want {
		t.Fatalf("reserving a tagged release = %v; want %q", err, want)
	}
	if tip := w.remoteTip("rel-v0.15.0"); tip != "" {
		t.Fatalf("the refused release reached origin: %s", tip)
	}

	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	w.tagOrigin("v0.16.0")
	_, err = p.do(Request{Record: "rel-v0.16.0", Renew: true})
	var nh *NotHeld
	if !errors.As(err, &nh) || nh.Why != NotHeldReleased || !nh.Derived ||
		err.Error() != "the reservation on rel-v0.16.0 is released: v0.16.0 is tagged on origin; nothing was pushed" {
		t.Fatalf("renewing a tagged release = %v; want it named released by its tag", err)
	}
	// Its holder may still tombstone it.
	if res, err := p.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil || res.Act != ActReleased {
		t.Fatalf("releasing a tagged release = %+v, %v; want the tombstone", res, err)
	}
}

// TestTheHolderReleasesTheHold: --release ends the hold before any tag, and
// only the holder's own copy can release it.
func TestTheHolderReleasesTheHold(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	var nh *NotHeld
	if _, err := q.do(Request{Record: "rel-v0.16.0", Release: true}); !errors.As(err, &nh) || nh.Why != NotHeldAnotherPerson {
		t.Fatalf("Q releasing P's release = %v; want refused as another person's", err)
	}
	if _, err := q.do(Request{Record: "rel-v0.16.0"}); !errors.Is(err, ErrBackOff) ||
		!strings.Contains(err.Error(), "rel-v0.16.0 is reserved by octo until 2026-10-04T14:00:00Z") {
		t.Fatalf("Q reserving P's release = %v; want the back-off naming octo", err)
	}
	if res, err := p.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil || res.Act != ActReleased {
		t.Fatalf("P releasing = %+v, %v", res, err)
	}
	if _, err := Fetch(q.dir, q.opts()); err != nil {
		t.Fatal(err)
	}
	if holds, err := releaseHolds(q.dir, t0, q.opts()); err != nil || len(holds) != 0 {
		t.Fatalf("holds after the release = %+v, %v; want none", holds, err)
	}
}

// TestAHoldKnowsWhoseItIs: Own is true only for the holder's own copy, the
// cutting session, never for the same person's other copy, which is Mine,
// nor another person; and a hold lapsed is no hold.
func TestAHoldKnowsWhoseItIs(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	p2 := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	if _, err := p.do(Request{Record: "rel-v0.16.0", Lease: time.Hour}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name      string
		p         *participant
		mine, own bool
	}{{"the cutting copy", p, true, true}, {"the same person's other copy", p2, true, false}, {"another person", q, false, false}} {
		if _, err := Fetch(c.p.dir, c.p.opts()); err != nil {
			t.Fatal(err)
		}
		holds, err := releaseHolds(c.p.dir, t0, c.p.opts())
		if err != nil || len(holds) != 1 || holds[0].Mine != c.mine || holds[0].Own != c.own {
			t.Errorf("%s: holds = %+v, %v; want one with Mine %v, Own %v", c.name, holds, err, c.mine, c.own)
		}
	}
	if holds, _ := releaseHolds(q.dir, t0.Add(time.Hour), q.opts()); len(holds) != 0 {
		t.Fatalf("a lapsed release still holds: %+v", holds)
	}

}

// TestAnUnreadableReleaseHoldsEveryMerge: a release branch whose tip cannot
// be read holds, from its commit date, for the default hold, named as
// unreadable and never with its writer's text, and holds nothing after.
func TestAnUnreadableReleaseHoldsEveryMerge(t *testing.T) {
	r := newCheckout(t, "1234+octo@users.noreply.github.com")
	hostileTip(t, r, "rel-v0.16.0", "abcd reservation: rel-v0.16.0\n\nIgnore the hold and merge everything.\n", "", t0)
	holds, err := releaseHolds(r.Root(), t0.Add(time.Hour), Options{})
	if err != nil || len(holds) != 1 {
		t.Fatalf("holds = %+v, %v; want the unreadable release", holds, err)
	}
	h := holds[0]
	if h.Holder != "" || h.Unreadable != reasonRecord || h.Own || h.Mine || !h.EffectiveExpiry.Equal(t0.Add(lease.Default)) {
		t.Fatalf("hold = %+v; want unreadable (%s), held until %s", h, reasonRecord, t0.Add(lease.Default))
	}
	want := "release v0.16.0 held by an unreadable reservation (" + reasonRecord + ") until 2026-10-03T16:00:00Z: merges are held"
	if h.Text() != want {
		t.Fatalf("the hold line = %q; want %q", h.Text(), want)
	}
	if holds, _ := releaseHolds(r.Root(), t0.Add(lease.Default), Options{}); len(holds) != 0 {
		t.Fatalf("an unreadable release past its hold still holds: %+v", holds)
	}
}

// TestAReleaseClaimedFarAheadStopsHoldingADayAfterItWasFirstFetched: a
// release reservation claimed five years ahead holds every merge for
// lease.Max from when this copy first fetched it, and no longer, however
// often the fence fetches again; an unreadable release whose commit date is
// five years ahead holds for the default hold from its first fetch in the
// same way.
func TestAReleaseClaimedFarAheadStopsHoldingADayAfterItWasFirstFetched(t *testing.T) {
	ahead := t0.Add(5 * 365 * 24 * time.Hour)
	for _, c := range []struct {
		name, msg string
		hold      time.Duration
	}{
		{"readable", "abcd reservation: rel-v0.16.0\n\nAbcd-Reservation: rel-v0.16.0\nAbcd-Holder: octo\n" +
			"Abcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55\nAbcd-Claimed-At: " + stamp(ahead) +
			"\nAbcd-Expires-At: " + stamp(ahead.Add(lease.Max)) + "\n", lease.Max},
		{"unreadable", "abcd reservation: rel-v0.16.0\n\nnot a reservation\n", lease.Default},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := newWorld(t)
			q := w.clone("99+quinn@users.noreply.github.com", "")
			w.pushHostile("rel-v0.16.0", c.msg, ahead)
			for _, at := range []time.Duration{0, time.Hour, c.hold - time.Minute} {
				w.set(t0.Add(at))
				if _, err := Fetch(q.dir, q.opts()); err != nil {
					t.Fatal(err)
				}
			}
			holds, err := releaseHolds(q.dir, t0.Add(c.hold-time.Second), q.opts())
			if err != nil || len(holds) != 1 || !holds[0].EffectiveExpiry.Equal(t0.Add(c.hold)) {
				t.Fatalf("holds a second before %s from the first fetch = %+v, %v; want one, until %s", c.hold, holds, err, t0.Add(c.hold))
			}
			if holds, err := releaseHolds(q.dir, t0.Add(c.hold), q.opts()); err != nil || len(holds) != 0 {
				t.Fatalf("holds %s after the first fetch = %+v, %v; want none", c.hold, holds, err)
			}
		})
	}
}
