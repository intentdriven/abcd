package claim

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/lease"
)

// The release hold, round 5, the reading of an end (iss-2610222014550692,
// dcn-2610230328220298): every reading of whether a release has ended holds
// a tombstone to the rule settleTip holds it to. It ends the release only
// when the forge's record of pushes says its holder pushed it, or when the
// tag is on origin; one another login pushed leaves the release still being
// cut, for the second-release refusal, the launch reading and peers.

// tombstonedByMal reserves rel-v0.16.0 as octo, withdrawing #11-#13, and puts
// a tombstone over it that the record of pushes says mal pushed.
func tombstonedByMal(t *testing.T) (*world, *fakeForge, *participant) {
	t.Helper()
	w := newWorld(t)
	f := armedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	tm := message{Record: m.Record, Holder: m.Holder, Copy: m.Copy, ClaimedAt: w.clock(), ExpiresAt: w.clock(), ReleasedAt: w.clock(), Withdrawn: ws, AssistedBy: m.AssistedBy}
	planted := w.pushChild("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), tm.text(), w.clock())
	f.pusherOf[planted] = "mal"
	w.set(t0.Add(time.Minute))
	return w, f, octo
}

// TestAnotherLoginsTombstoneOverALiveReleaseBlocksASecondRelease: a second
// release is refused while the first is live beneath a tombstone its holder
// did not push; once the holder's own --release ends it, the second goes
// through.
func TestAnotherLoginsTombstoneOverALiveReleaseBlocksASecondRelease(t *testing.T) {
	w, f, octo := tombstonedByMal(t)
	quinn := w.clone("99+quinn@users.noreply.github.com", "")
	f.reset()
	_, err := quinn.do(Request{Record: "rel-v0.17.0"})
	if !errors.Is(err, ErrBackOff) || !strings.Contains(err.Error(), "rel-v0.16.0 is being cut by "+octo.holderID()) || len(f.mutations()) != 0 {
		t.Fatalf("quinn's reserve = %v, mutations %v; want a back-off naming octo's cut", err, f.mutations())
	}
	if _, err := octo.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := quinn.do(Request{Record: "rel-v0.17.0"}); err != nil {
		t.Fatalf("quinn's reserve after octo's --release = %v; want it through", err)
	}
}

// TestALapsedReleaseBeneathAnotherLoginsTombstoneBlocksASecondRelease: the
// lease beneath that tombstone lapsing ends nothing either, while it holds
// what it withdrew.
func TestALapsedReleaseBeneathAnotherLoginsTombstoneBlocksASecondRelease(t *testing.T) {
	w, f, octo := tombstonedByMal(t)
	w.set(t0.Add(lease.Max + TakeoverMargin + time.Hour))
	quinn := w.clone("99+quinn@users.noreply.github.com", "")
	f.reset()
	_, err := quinn.do(Request{Record: "rel-v0.17.0"})
	if !errors.Is(err, ErrBackOff) || !strings.Contains(err.Error(), "rel-v0.16.0 is still being cut by "+octo.holderID()) || len(f.mutations()) != 0 {
		t.Fatalf("quinn's reserve = %v, mutations %v; want a back-off naming octo's lapsed cut", err, f.mutations())
	}
}

// TestATombstoneTheRecordDoesNotPlaceBlocksASecondReleaseNamingWhy: when the
// record of pushes cannot be read, the tombstone is read as ending nothing,
// and the refusal says why.
func TestATombstoneTheRecordDoesNotPlaceBlocksASecondReleaseNamingWhy(t *testing.T) {
	w, f, _ := tombstonedByMal(t)
	f.mu.Lock()
	f.pushersErr = errors.New("HTTP 502: the record is not answering")
	f.mu.Unlock()
	quinn := w.clone("99+quinn@users.noreply.github.com", "")
	f.reset()
	_, err := quinn.do(Request{Record: "rel-v0.17.0"})
	if !errors.Is(err, ErrBackOff) || !strings.Contains(err.Error(), "rel-v0.16.0") || !strings.Contains(err.Error(), "not answering") || len(f.mutations()) != 0 {
		t.Fatalf("quinn's reserve = %v, mutations %v; want a back-off naming why the tombstone is not settled", err, f.mutations())
	}
}

// TestASecondReleaseIsRefusedWhenTheReservationsDoNotRead: a view of the
// reservations that fails refuses the second release, naming the failure,
// rather than skipping the check.
func TestASecondReleaseIsRefusedWhenTheReservationsDoNotRead(t *testing.T) {
	err := otherLiveRelease(t.TempDir(), "rel-v0.17.0", t0, Self{})
	if err == nil || !strings.Contains(err.Error(), "reading the reservations") {
		t.Fatalf("otherLiveRelease over no checkout = %v; want a refusal naming the failed reading", err)
	}
}

// TestTheLaunchReadingKeepsAReleaseAnotherLoginsTombstoneDidNotEnd: the
// cutter's launch reading still names the release as theirs while it is
// live beneath the tombstone, and as lapsed once its lease lapses.
func TestTheLaunchReadingKeepsAReleaseAnotherLoginsTombstoneDidNotEnd(t *testing.T) {
	w, _, octo := tombstonedByMal(t)
	if _, err := Fetch(octo.dir, octo.opts()); err != nil {
		t.Fatal(err)
	}
	a, err := ArmedMergesForCut(octo.dir, octo.opts())
	if err != nil {
		t.Fatal(err)
	}
	if a.Release != "rel-v0.16.0" {
		t.Fatalf("release = %q, lapsed = %q; want rel-v0.16.0 still octo's cut", a.Release, a.Lapsed)
	}
	w.set(t0.Add(lease.Max + TakeoverMargin + time.Hour))
	if a, err = ArmedMergesForCut(octo.dir, octo.opts()); err != nil {
		t.Fatal(err)
	}
	if a.Release != "" || a.Lapsed != "rel-v0.16.0" {
		t.Fatalf("release = %q, lapsed = %q; want rel-v0.16.0 named lapsed", a.Release, a.Lapsed)
	}
}

// TestPeersReportsALapseBeneathAnotherLoginsTombstone: peers lists the
// release as not ended, holding what octo withdrew, and as lapsed once its
// lease lapses.
func TestPeersReportsALapseBeneathAnotherLoginsTombstone(t *testing.T) {
	w, _, octo := tombstonedByMal(t)
	w.set(t0.Add(lease.Max + TakeoverMargin + time.Hour))
	if _, err := Fetch(octo.dir, octo.opts()); err != nil {
		t.Fatal(err)
	}
	snap, err := View(octo.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	got := snap.ReleaseWithdrawals(octo.dir, octo.self(), w.clock())
	if len(got) != 1 || got[0].Ended || !got[0].Lapsed || got[0].NotEnded != octo.holderID() || len(got[0].Held) != 3 || len(got[0].Due) != 0 {
		t.Fatalf("listing = %+v; want rel-v0.16.0 not ended, lapsed, holding #11-#13", got)
	}
}
