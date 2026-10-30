package claim

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// The readings of a release hold a holder id as the acts do
// (iss-2610290809526033): one of the reader's own ids names only the reader's
// login's pushes, an id the reader merely declares (another account of
// theirs) names whoever pushed it, and an id that is not the reader's names
// a holder who is only claimed.

// TestAnIDNamedWithdrawalIsNotDueWhenTheReadersLoginIsUnknown: a reader whose
// id is known but whose login is not cannot have their own newest push read,
// so a withdrawal naming their id on an ended release is not listed as due
// for their next act to re-arm: it is stranded. With the login known it is
// due.
func TestAnIDNamedWithdrawalIsNotDueWhenTheReadersLoginIsUnknown(t *testing.T) {
	w := newWorld(t)
	armedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	w.tagOrigin("v0.16.0")
	if _, err := Fetch(octo.dir, octo.opts()); err != nil {
		t.Fatal(err)
	}
	snap, err := View(octo.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	listing := func(reader Self) ReleaseWithdrawals {
		t.Helper()
		for _, rw := range snap.ReleaseWithdrawals(octo.dir, reader, w.clock()) {
			if rw.Subject == "rel-v0.16.0" {
				return rw
			}
		}
		t.Fatalf("no listing of rel-v0.16.0 for %+v", reader)
		return ReleaseWithdrawals{}
	}
	idOnly := listing(Self{ID: octo.keptHolderID()})
	if !idOnly.Ended || len(idOnly.Due) != 0 || !slices.Equal(numbers(idOnly.Stranded), []string{"#11", "#12", "#13"}) {
		t.Fatalf("with the login unknown, the listing = %+v; want #11-#13 stranded, none due", idOnly)
	}
	if got := listing(octo.self()); !slices.Equal(numbers(got.Due), []string{"#11", "#12", "#13"}) {
		t.Fatalf("with the login known, the listing = %+v; want #11-#13 due", got)
	}
}

// TestAReleaseHeldUnderAnotherHoldersIDIsShownAsClaimed: to a reader whose
// id it is not, a release's holder id names no account the forge's record
// could vouch for, so the listing, the not-ended sentence and the second
// release's back-off name it as claimed; to its holder it is not.
func TestAReleaseHeldUnderAnotherHoldersIDIsShownAsClaimed(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	id := octo.keptHolderID()
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	tomb := message{Record: m.Record, Holder: id, Copy: m.Copy, ClaimedAt: w.clock(), ExpiresAt: w.clock(), ReleasedAt: w.clock(), Withdrawn: ws, AssistedBy: m.AssistedBy}
	w.pushAs(f, "rel-v0.16.0", "mal", tomb.text())
	maya := w.clone("5678+maya@users.noreply.github.com", "")

	theirs := listingOf(t, w, maya, "rel-v0.16.0")
	if theirs.Holder != id || !theirs.HolderClaimed || theirs.NotEnded != id ||
		!strings.Contains(strings.Join(theirs.Lines(), "\n"), "a tombstone "+id+" (claimed) did not push") {
		t.Fatalf("maya's listing = %+v %q; want octo's id named as claimed", theirs, theirs.Lines())
	}
	own := listingOf(t, w, octo, "rel-v0.16.0")
	if own.Holder != id || own.HolderClaimed || strings.Contains(strings.Join(own.Lines(), "\n"), "(claimed)") {
		t.Fatalf("octo's listing = %+v %q; want octo's own id not called claimed", own, own.Lines())
	}
	_, err := maya.do(Request{Record: "rel-v0.17.0"})
	if !errors.Is(err, ErrBackOff) || !strings.Contains(err.Error(), "rel-v0.16.0 is being cut by "+id+" (claimed) until") {
		t.Fatalf("maya's second release = %v; want a back-off naming octo's id as claimed", err)
	}
}

// TestATakeOverBetweenTwoDeclaredAccountsKeepsTheWithdrawals: octo and maya
// are one person's two accounts, each declaring the other's holder id. maya's
// take-over of octo's cut carries every withdrawal octo's cut made, so none
// is lost; once maya's --release ends it, octo's copy reads the release as
// ended and settles what octo withdrew.
func TestATakeOverBetweenTwoDeclaredAccountsKeepsTheWithdrawals(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	a := octo.keptHolderID()
	maya := w.clone("5678+maya@users.noreply.github.com", "")
	b := "h_" + strings.Repeat("b", 32)
	keepHolderID(t, maya.home, b)
	declareHolderIDs(t, maya.home, a+"\n")
	declareHolderIDs(t, octo.home, b+"\n")

	if _, err := maya.do(Request{Record: "rel-v0.16.0", TakeOver: true}); err != nil {
		t.Fatalf("maya's take-over: %v", err)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	if m.Holder != b || !slices.Equal(numbers(ws), []string{"#11", "#12", "#13"}) {
		t.Fatalf("maya's take-over names %q and carries %+v; want maya's id carrying #11-#13", m.Holder, ws)
	}
	for _, x := range ws {
		if x.By != a || !x.Pending() {
			t.Fatalf("the take-over carries %+v; want octo's withdrawal, still withdrawn", x)
		}
	}
	if lost := LostReleaseWithdrawals(maya.dir, w.clock()); len(lost) != 0 {
		t.Fatalf("lost withdrawals after the take-over: %+v", lost)
	}

	if _, err := maya.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatalf("maya's release: %v", err)
	}
	if _, err := Fetch(octo.dir, octo.opts()); err != nil {
		t.Fatal(err)
	}
	snap, err := View(octo.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(snap.Reservations, func(r Reservation) bool { return r.Record == "rel-v0.16.0" })
	if i < 0 {
		t.Fatal("octo's view holds no rel-v0.16.0")
	}
	if _, notEnded, why := asCut(octo.dir, snap.Reservations[i], w.clock(), octo.self()); notEnded || why != "" {
		t.Fatalf("octo reads maya's release as not ended (%q); want it ended by maya's own tombstone", why)
	}
	f.reset()
	res, err := octo.reserve("iss-8", "o.go")
	if err != nil {
		t.Fatal(err)
	}
	i = slices.IndexFunc(res.Settles, func(s Settle) bool { return s.Subject == "rel-v0.16.0" })
	if i < 0 || res.Settles[i].Commit == "" || res.Settles[i].NotEnded != "" {
		t.Fatalf("octo's settles = %+v; want octo's copy to claim the end and settle what octo withdrew", res.Settles)
	}
	if final, _ := w.tipWithdrawals("rel-v0.16.0"); final.Holder != a {
		t.Fatalf("the tip after octo's settle names %q; want octo's tombstone", final.Holder)
	}
}
