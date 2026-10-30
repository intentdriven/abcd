package claim

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// Every holder id is held to the account that first recorded a withdrawal
// under it (iss-2610290809526033): an entry naming one is acted on only when
// the reservation that first recorded it, held under that id, was pushed by
// the account the id is held to, the reader's own kept id included. A later
// push that carries the entry says nothing of who first recorded it, so an
// entry another login recorded under a copy's kept id, carried by a second
// copy of the same person that reads the id as a declared one, is never read
// as the first copy's own.

// copyAID is the holder id kept by the first of two copies of octo's, each
// in a home of its own; the second keeps studioID.
var copyAID = "h_" + strings.Repeat("7", 32)

// twoCopiesOverMalsEntry is rel-v0.16.0 held under studioID with #14
// recorded withdrawn under it, in one commit the forge's record says mal
// pushed, lapsed past the take-over margin; a copy of octo's keeping copyAID
// that declares studioID as declared says, and a second copy of octo's that
// keeps studioID itself.
func twoCopiesOverMalsEntry(t *testing.T, declared string) (*world, *fakeForge, *participant, *participant) {
	t.Helper()
	w := newWorld(t)
	f := fourArmedForge(t)
	heldWithDeclaredEntry(t, w, f, studioID, "mal")
	w.set(w.clock().Add(2 * time.Hour))
	a := w.clone("1234+octo@users.noreply.github.com", t.TempDir())
	keepHolderID(t, a.home, copyAID)
	declareHolderIDs(t, a.home, declared)
	b := w.clone("1234+octo@users.noreply.github.com", t.TempDir())
	keepHolderID(t, b.home, studioID)
	f.reset()
	return w, f, a, b
}

// actedOn14 is each call that re-armed #14 or commented on it.
func actedOn14(f *fakeForge) []string {
	var out []string
	for _, m := range f.mutations() {
		if strings.HasSuffix(m, " #14") {
			out = append(out, m)
		}
	}
	for _, c := range f.posted() {
		if strings.HasPrefix(c, "#14 ") {
			out = append(out, c)
		}
	}
	return out
}

func TestAnEntryUnderACopysKeptIDAnotherLoginFirstRecordedIsNeverThatCopysOwn(t *testing.T) {
	for _, c := range []struct{ name, declared string }{
		{"declared with no login beside it", studioID + "\n"},
		{"paired with the copies' own login", studioID + " octo\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			w, f, a, b := twoCopiesOverMalsEntry(t, c.declared)
			if _, err := a.do(Request{Record: "rel-v0.16.0", TakeOver: true}); err != nil {
				t.Fatalf("the first copy's take-over: %v", err)
			}
			res, err := a.do(Request{Record: "rel-v0.16.0", Release: true})
			if err != nil {
				t.Fatalf("the first copy's release: %v", err)
			}
			if got := actedOn14(f); len(got) != 0 {
				t.Fatalf("the first copy's release acted on #14: %v; want it left withdrawn", got)
			}
			if !slices.Equal(numbers(res.Stranded), []string{"#14"}) {
				t.Fatalf("the first copy's release stranded %+v; want #14", res.Stranded)
			}
			if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 14) != WithdrawnPending {
				t.Fatalf("rel-v0.16.0's tip records %+v; want #14 carried, still withdrawn", ws)
			}
			rw := listingOf(t, w, b, "rel-v0.16.0")
			if !rw.Ended || slices.Contains(numbers(rw.Due), "#14") || !slices.Contains(numbers(rw.Stranded), "#14") {
				t.Fatalf("the second copy's listing = %+v; want #14 stranded, never due", rw)
			}
			if l := strings.Join(rw.Lines(), "\n"); strings.Contains(l, "only a copy of "+studioID) || !strings.Contains(l, "#14 named by your holder id on rel-v0.16.0 stay withdrawn") {
				t.Fatalf("the second copy's lines = %q; want #14 named as under its own id, never as another's", rw.Lines())
			}
			if _, err := SettleEndedReleases(b.dir, b.opts()); err != nil {
				t.Fatal(err)
			}
			if _, err := b.reserve("iss-8", "o.go"); err != nil {
				t.Fatal(err)
			}
			if got := actedOn14(f); len(got) != 0 {
				t.Fatalf("the second copy acted on #14: %v; want it left withdrawn", got)
			}
			if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 14) != WithdrawnPending {
				t.Fatalf("rel-v0.16.0's tip records %+v; want #14 still withdrawn", ws)
			}
		})
	}
}

// TestACutTakenOverByASecondCopyAndBackIsSettledFromEither: octo's cut from
// one home, taken over by a second home that declares the first's id, and
// taken back by the first, which declares the second's, re-arms every withdrawal on the first home's release; released
// from the second home instead, it does too.
func TestACutTakenOverByASecondCopyAndBackIsSettledFromEither(t *testing.T) {
	t.Run("taken back and released by the copy that cut it", func(t *testing.T) {
		_, f, laptop, studio := twoHomesCut(t)
		declareHolderIDs(t, laptop.home, studioID+"\n")
		if _, err := studio.do(Request{Record: "rel-v0.16.0", TakeOver: true}); err != nil {
			t.Fatalf("the second home's take-over: %v", err)
		}
		if _, err := laptop.do(Request{Record: "rel-v0.16.0", TakeOver: true}); err != nil {
			t.Fatalf("the first home's take-over back: %v", err)
		}
		f.reset()
		res, err := laptop.do(Request{Record: "rel-v0.16.0", Release: true})
		if err != nil {
			t.Fatalf("the first home's release: %v", err)
		}
		if !allRearmed(f) || len(res.Stranded) != 0 {
			t.Fatalf("mutations %v, stranded %+v; want #11-#13 re-armed and nothing stranded", f.mutations(), res.Stranded)
		}
	})
	t.Run("released by the second copy", func(t *testing.T) {
		_, f, _, studio := twoHomesCut(t)
		if _, err := studio.do(Request{Record: "rel-v0.16.0", TakeOver: true}); err != nil {
			t.Fatalf("the second home's take-over: %v", err)
		}
		f.reset()
		res, err := studio.do(Request{Record: "rel-v0.16.0", Release: true})
		if err != nil {
			t.Fatalf("the second home's release: %v", err)
		}
		if !allRearmed(f) || len(res.Stranded) != 0 {
			t.Fatalf("mutations %v, stranded %+v; want #11-#13 re-armed and nothing stranded", f.mutations(), res.Stranded)
		}
	})
}
