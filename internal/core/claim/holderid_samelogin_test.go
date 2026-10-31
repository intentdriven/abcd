package claim

import (
	"slices"
	"strings"
	"testing"
)

// One person on two machines under one login, each home minting its own
// holder id (iss-2610290809526033): an id declared with no login beside it
// is held to the reader's own login, so a reservation commit naming it that
// the reader's login pushed is theirs, and a withdrawal under it that a
// reservation held by that id and pushed by the reader's login first
// recorded is acted on and listed due. Pushed by any other login, it is
// still never acted on.

// studioID is the holder id kept by octo's second home.
var studioID = "h_" + strings.Repeat("e", 32)

// twoHomesCut is octo's cut of rel-v0.16.0, made from the laptop's home,
// which withdrew #11, #12 and #13 under the laptop's id, and a copy of
// octo's from a second home, which keeps an id of its own and declares the
// laptop's with no login beside it.
func twoHomesCut(t *testing.T) (*world, *fakeForge, *participant, *participant) {
	t.Helper()
	w := newWorld(t)
	f := fourArmedForge(t)
	laptop := w.clone("1234+octo@users.noreply.github.com", t.TempDir())
	if _, err := laptop.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	studio := w.clone("1234+octo@users.noreply.github.com", t.TempDir())
	keepHolderID(t, studio.home, studioID)
	declareHolderIDs(t, studio.home, laptop.keptHolderID()+"\n")
	f.reset()
	return w, f, laptop, studio
}

func TestAnUnpairedIDOfTheReadersOwnLoginsOtherHomeIsActedOn(t *testing.T) {
	t.Run("take-over and release", func(t *testing.T) {
		_, f, _, studio := twoHomesCut(t)
		if _, err := studio.do(Request{Record: "rel-v0.16.0", TakeOver: true}); err != nil {
			t.Fatalf("the second home's take-over: %v", err)
		}
		f.reset()
		res, err := studio.do(Request{Record: "rel-v0.16.0", Release: true})
		if err != nil {
			t.Fatalf("the second home's release: %v", err)
		}
		if !allRearmed(f) || len(res.Stranded) != 0 || len(res.Unpaired) != 0 {
			t.Fatalf("mutations %v, stranded %+v, unpaired %v; want #11-#13 re-armed and nothing stranded", f.mutations(), res.Stranded, res.Unpaired)
		}
	})
	t.Run("listed due", func(t *testing.T) {
		w, _, laptop, studio := twoHomesCut(t)
		w.tagOrigin("v0.16.0")
		rw := listingOf(t, w, studio, "rel-v0.16.0")
		if !rw.Ended || !slices.Equal(numbers(rw.Due), []string{"#11", "#12", "#13"}) || len(rw.Stranded) != 0 || len(rw.Unpaired) != 0 {
			t.Fatalf("the second home's listing = %+v; want #11-#13 due and nothing stranded", rw)
		}
		if slices.ContainsFunc(rw.Lines(), func(l string) bool { return strings.Contains(l, laptop.keptHolderID()+" is an id ") }) {
			t.Fatalf("lines %q; want no line on how to pair an id whose withdrawals are acted on", rw.Lines())
		}
	})
	t.Run("settled by the next act", func(t *testing.T) {
		w, f, _, studio := twoHomesCut(t)
		w.tagOrigin("v0.16.0")
		res, err := studio.reserve("iss-8", "s.go")
		if err != nil {
			t.Fatal(err)
		}
		i := slices.IndexFunc(res.Settles, func(s Settle) bool { return s.Subject == "rel-v0.16.0" })
		if i < 0 || !allRearmed(f) {
			t.Fatalf("settles %+v, mutations %v; want #11-#13 of rel-v0.16.0 re-armed", res.Settles, f.mutations())
		}
	})
	t.Run("a tombstone the reader's login pushed ends the cut", func(t *testing.T) {
		w, _, octo := endedOverOctosCutNaming(t, unpairedD, "octo")
		declareHolderIDs(t, octo.home, unpairedD+"\n")
		if rw := listingOf(t, w, octo, "rel-v0.16.0"); !rw.Ended {
			t.Fatalf("octo's listing = %+v; want the cut ended", rw)
		}
	})
}

// TestAnUnpairedIDsEntryAnotherLoginRecordedStaysStrandedAfterTheReadersCarry:
// mal records #14 withdrawn under an id octo declares with no login beside
// it, in a reservation held by that id. octo's take-over carries the entry
// into octo's own push, but the reservation that first recorded it was not
// pushed by octo's login, so octo's release leaves it withdrawn and lists it
// stranded, with how to pair the id.
func TestAnUnpairedIDsEntryAnotherLoginRecordedStaysStrandedAfterTheReadersCarry(t *testing.T) {
	w := newWorld(t)
	f := fourArmedForge(t)
	heldWithDeclaredEntry(t, w, f, unpairedD, "mal")
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	declareHolderIDs(t, octo.home, unpairedD+"\n")
	if _, err := octo.do(Request{Record: "rel-v0.16.0", TakeOver: true}); err != nil {
		t.Fatalf("octo's take-over: %v", err)
	}
	w.tagOrigin("v0.16.0")
	rw := listingOf(t, w, octo, "rel-v0.16.0")
	if !slices.Equal(numbers(rw.Stranded), []string{"#14"}) || slices.Contains(numbers(rw.Due), "#14") || !slices.Equal(rw.Unpaired, []string{unpairedD}) {
		t.Fatalf("octo's listing = %+v; want #14 stranded and its id named unpaired", rw)
	}
	f.reset()
	if _, err := octo.reserve("iss-8", "o.go"); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(f.mutations(), "enablePullRequestAutoMerge #14") {
		t.Fatalf("mutations %v; want #14 left withdrawn", f.mutations())
	}
}
