package claim

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// A declared holder id is held to the account its declaration pairs it with
// (iss-2610290809526033): a commit naming it is that account's only when the
// forge's record of pushes says that account pushed it, and an id declared
// with no login beside it is held to the reader's own login, so a commit
// naming it that another login pushed ends nothing here and no withdrawal of
// it another login recorded is acted on.

// unpairedD is an id octo declares with no login beside it.
var unpairedD = "h_" + strings.Repeat("d", 32)

// mayaB is the holder id kept by maya, one person's second account.
var mayaB = "h_" + strings.Repeat("b", 32)

// endedOverOctosCutNaming is octo's live cut of rel-v0.16.0, which withdrew
// #11, #12 and #13, with a reservation naming holder and a tombstone over it
// that the forge's record says pusher pushed.
func endedOverOctosCutNaming(t *testing.T, holder, pusher string) (*world, *fakeForge, *participant) {
	t.Helper()
	w := newWorld(t)
	f := armedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	over := message{Record: m.Record, Holder: holder, Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: w.clock(), ExpiresAt: w.clock().Add(time.Hour), Withdrawn: ws, AssistedBy: m.AssistedBy}
	w.pushAs(f, "rel-v0.16.0", pusher, over.text())
	tomb := over
	tomb.ReleasedAt, tomb.ExpiresAt = w.clock(), w.clock()
	w.pushAs(f, "rel-v0.16.0", pusher, tomb.text())
	w.set(t0.Add(time.Minute))
	f.reset()
	return w, f, octo
}

// TestAnUnpairedDeclaredIDsReservationAndTombstoneEndNothing: over octo's
// live cut, mal pushes a reservation and a tombstone naming an id octo
// declares with no login beside it. Held to octo's own login, which did not
// push them, the listing, the launch reading and the act's settle each read
// the cut as not ended, and nothing is re-armed.
func TestAnUnpairedDeclaredIDsReservationAndTombstoneEndNothing(t *testing.T) {
	w, f, octo := endedOverOctosCutNaming(t, unpairedD, "mal")
	declareHolderIDs(t, octo.home, unpairedD+"\n")
	rw := listingOf(t, w, octo, "rel-v0.16.0")
	if rw.Ended || len(rw.Due) != 0 || !slices.Equal(numbers(rw.Held), []string{"#11", "#12", "#13"}) {
		t.Fatalf("octo's listing = %+v; want the cut not ended, holding #11-#13, nothing due", rw)
	}
	a, err := ArmedMergesForCut(octo.dir, octo.opts())
	if err != nil {
		t.Fatal(err)
	}
	if a.Release != "rel-v0.16.0" {
		t.Fatalf("octo's launch reading names %q as octo's cut; want rel-v0.16.0, still live", a.Release)
	}
	res, err := octo.reserve("iss-8", "o.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range res.Settles {
		if len(s.Settled) != 0 || s.Commit != "" {
			t.Fatalf("settles = %+v; want nothing ended by a tombstone naming an unpaired id", res.Settles)
		}
	}
	if got := f.mutations(); len(got) != 0 {
		t.Fatalf("mutations %v; want nothing re-armed", got)
	}
}

// TestAPairedDeclaredIDsTombstoneEndsTheCutOnlyFromItsPairedAccount: the same
// reservation and tombstone, naming an id octo pairs with maya's login, end
// nothing when mal pushed them, and end the cut when maya did.
func TestAPairedDeclaredIDsTombstoneEndsTheCutOnlyFromItsPairedAccount(t *testing.T) {
	for _, c := range []struct {
		pusher string
		ended  bool
	}{{"mal", false}, {"maya", true}} {
		t.Run(c.pusher, func(t *testing.T) {
			w, _, octo := endedOverOctosCutNaming(t, mayaB, c.pusher)
			declareHolderIDs(t, octo.home, mayaB+" maya\n")
			if rw := listingOf(t, w, octo, "rel-v0.16.0"); rw.Ended != c.ended {
				t.Fatalf("pushed by %s, octo's listing = %+v; want ended %v", c.pusher, rw, c.ended)
			}
		})
	}
}

// fourArmedForge is commentedForge with #14 open and armed for nothing:
// #11 (squash), #12 (queue) and #13 (merge) are armed, and every comment is
// kept.
func fourArmedForge(t *testing.T) *fakeForge {
	t.Helper()
	f := commentedForge(t)
	f.mu.Lock()
	f.prs[14] = &fakePR{id: "PR_fourteen", head: headA}
	f.mu.Unlock()
	return f
}

// heldWithDeclaredEntry is rel-v0.16.0 reserved, live, under holder, with
// #14 recorded withdrawn by the same holder, in one commit the forge's record
// says pusher pushed; #14 is not armed on the forge.
func heldWithDeclaredEntry(t *testing.T, w *world, f *fakeForge, holder, pusher string) {
	t.Helper()
	wd := []Withdrawal{{Number: 14, ID: "PR_fourteen", Form: FormAutoSquash, Head: headA, By: holder, State: WithdrawnPending}}
	m := message{Record: "rel-v0.16.0", Holder: holder, Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: w.clock(),
		ExpiresAt: w.clock().Add(time.Hour), Withdrawn: wd, AssistedBy: "abcd:0.0.0"}
	f.pusherOf[w.pushTipAt("rel-v0.16.0", m.text(), w.clock())] = pusher
}

// TestATakeOverCarriesADeclaredIDsEntriesButActsOnlyOnItsPairedAccounts: a
// release octo never pushed to holds #14 withdrawn under an id octo
// declares. octo's take-over carries it, so nothing is lost; octo's
// --release re-arms it, and comments on it, only when octo pairs the id
// with a login and that login pushed the commit recording it; otherwise it
// stays withdrawn and is listed stranded.
func TestATakeOverCarriesADeclaredIDsEntriesButActsOnlyOnItsPairedAccounts(t *testing.T) {
	for _, c := range []struct {
		name, declared, pusher string
		rearmed                bool
	}{
		{"unpaired", unpairedD + "\n", "mal", false},
		{"paired, pushed by another login", unpairedD + " maya\n", "mal", false},
		{"paired, pushed by its account", unpairedD + " maya\n", "maya", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := newWorld(t)
			f := fourArmedForge(t)
			heldWithDeclaredEntry(t, w, f, unpairedD, c.pusher)
			octo := w.clone("1234+octo@users.noreply.github.com", "")
			declareHolderIDs(t, octo.home, c.declared)
			if _, err := octo.do(Request{Record: "rel-v0.16.0", TakeOver: true}); err != nil {
				t.Fatalf("octo's take-over: %v", err)
			}
			if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 14) != WithdrawnPending {
				t.Fatalf("the take-over carries %+v; want #14 still withdrawn", ws)
			}
			f.reset()
			res, err := octo.do(Request{Record: "rel-v0.16.0", Release: true})
			if err != nil {
				t.Fatalf("octo's release: %v", err)
			}
			rearmed := slices.Contains(f.mutations(), "enablePullRequestAutoMerge #14")
			commented := slices.ContainsFunc(f.posted(), func(s string) bool { return strings.HasPrefix(s, "#14 ") })
			stranded := slices.ContainsFunc(res.Stranded, func(x Withdrawal) bool { return x.Number == 14 })
			if rearmed != c.rearmed || commented != c.rearmed || stranded == c.rearmed {
				t.Fatalf("#14 re-armed %v, commented %v, stranded %v (%v, %v); want re-armed and commented %v",
					rearmed, commented, stranded, f.mutations(), res.Stranded, c.rearmed)
			}
			if !allRearmed(f) {
				t.Fatalf("octo's own withdrawals were not re-armed: %v", f.mutations())
			}
			if wantUnpaired := c.name == "unpaired"; slices.Equal(res.Unpaired, []string{unpairedD}) != wantUnpaired || !wantUnpaired && len(res.Unpaired) != 0 {
				t.Fatalf("unpaired = %v; want %s named only when it is unpaired", res.Unpaired, unpairedD)
			}
		})
	}
}

// TestARoundTripTakeOverBetweenTwoPairedAccountsLosesNoWithdrawal: octo and
// maya are one person's two accounts, each pairing the other's id with its
// login. maya takes octo's cut over and withdraws #14; octo takes it back.
// Every withdrawal of both is carried, octo's --release re-arms them all,
// and maya's copy then reads the release as ended and settles its own.
func TestARoundTripTakeOverBetweenTwoPairedAccountsLosesNoWithdrawal(t *testing.T) {
	w := newWorld(t)
	f := fourArmedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	a := octo.keptHolderID()
	maya := w.clone("5678+maya@users.noreply.github.com", "")
	keepHolderID(t, maya.home, mayaB)
	declareHolderIDs(t, maya.home, a+" octo\n")
	declareHolderIDs(t, octo.home, mayaB+" maya\n")
	f.mu.Lock()
	f.prs[14].method = "SQUASH"
	f.mu.Unlock()

	if _, err := maya.do(Request{Record: "rel-v0.16.0", TakeOver: true}); err != nil {
		t.Fatalf("maya's take-over: %v", err)
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); !slices.Equal(numbers(ws), []string{"#11", "#12", "#13", "#14"}) {
		t.Fatalf("maya's take-over records %+v; want #11-#13 carried and #14 withdrawn", ws)
	}
	if _, err := octo.do(Request{Record: "rel-v0.16.0", TakeOver: true}); err != nil {
		t.Fatalf("octo's take-back: %v", err)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	if m.Holder != a || stateOf(ws, 14) != WithdrawnPending || len(ws) != 4 {
		t.Fatalf("octo's take-back names %q and records %+v; want octo's id carrying #11-#14", m.Holder, ws)
	}
	if lost := LostReleaseWithdrawals(octo.dir, w.clock()); len(lost) != 0 {
		t.Fatalf("lost withdrawals after the round trip: %+v", lost)
	}
	f.reset()
	if _, err := octo.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatalf("octo's release: %v", err)
	}
	if !allRearmed(f) || f.pr(14).method != "SQUASH" {
		t.Fatalf("after octo's release: %v; want #11-#14 re-armed", f.mutations())
	}
	res, err := maya.reserve("iss-9", "m.go")
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(res.Settles, func(s Settle) bool { return s.Subject == "rel-v0.16.0" })
	if i < 0 || res.Settles[i].Commit == "" || res.Settles[i].NotEnded != "" || res.Settles[i].Unplaced != "" {
		t.Fatalf("maya's settles = %+v; want maya's copy to settle its own withdrawal", res.Settles)
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 14) != WithdrawnFoundArmed {
		t.Fatalf("after maya's settle the tip records %+v; want #14 found armed", ws)
	}
}

// TestAListingSplitsDueFromStrandedByWhoseRecordItIs: on an ended release,
// octo's own withdrawals and those of an id octo pairs with maya that maya's
// own push, holding the cut under that id, first records are due; one of an
// unpaired id, and one naming maya's id that mal pushed, are stranded, and
// the listing says how to pair the unpaired id.
func TestAListingSplitsDueFromStrandedByWhoseRecordItIs(t *testing.T) {
	w := newWorld(t)
	f := armedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	entry := func(n int, by string) Withdrawal {
		return Withdrawal{Number: n, ID: "PR_" + strings.Repeat("x", n), Form: FormAutoSquash, Head: headA, By: by, State: WithdrawnPending}
	}
	// maya's push holds the cut under her own id, as her take-over writes it.
	m.Holder = mayaB
	m.Withdrawn = append(slices.Clone(ws), entry(14, mayaB))
	w.pushAs(f, "rel-v0.16.0", "maya", m.text())
	m.Withdrawn = append(m.Withdrawn, entry(15, unpairedD), entry(16, mayaB))
	w.pushAs(f, "rel-v0.16.0", "mal", m.text())
	w.tagOrigin("v0.16.0")
	declareHolderIDs(t, octo.home, mayaB+" maya\n"+unpairedD+"\n")

	rw := listingOf(t, w, octo, "rel-v0.16.0")
	if !rw.Ended || !slices.Equal(slices.Sorted(slices.Values(numbers(rw.Due))), []string{"#11", "#12", "#13", "#14"}) ||
		!slices.Equal(slices.Sorted(slices.Values(numbers(rw.Stranded))), []string{"#15", "#16"}) {
		t.Fatalf("octo's listing = %+v; want #11-#14 due and #15, #16 stranded", rw)
	}
	i := slices.IndexFunc(rw.Lines(), func(l string) bool {
		return strings.HasPrefix(l, unpairedD+" is an id "+abcdhome.Display("holder-ids")+" declares as yours with no login beside it")
	})
	if i < 0 || strings.Contains(rw.Lines()[i], "maya") || !slices.Equal(rw.Unpaired, []string{unpairedD}) {
		t.Fatalf("lines %q, unpaired %v; want the unpaired id named with how to pair it, naming no login", rw.Lines(), rw.Unpaired)
	}
}

// TestADueLoginNamedWithdrawalWithNoHolderIDSaysWhatSettlesIt: a person with
// withdrawals named by their login on an ended release and no holder id yet
// is told those stay held until their next reservation or announcement.
func TestADueLoginNamedWithdrawalWithNoHolderIDSaysWhatSettlesIt(t *testing.T) {
	w, _, octo := loginNamedEndedRelease(t)
	rw := listingOf(t, w, octo, "rel-v0.16.0")
	lines := strings.Join(rw.Lines(), "\n")
	if !slices.Equal(numbers(rw.Due), []string{"#11"}) ||
		!strings.Contains(lines, "held merges from before the holder id stay held until your next reservation or announcement") ||
		strings.Contains(lines, "your next abcd act that fetches the reservations re-arms them") || strings.Contains(strings.ToLower(lines), "octo") {
		t.Fatalf("octo's listing = %+v %q; want #11 due, held until the next reservation or announcement, and no login", rw, rw.Lines())
	}
}

// loginNamedEndedRelease is rel-v0.16.0 reserved before the holder id, by
// octo's login, with #11 withdrawn by it and disarmed on the forge, and
// v0.16.0 tagged; octo's home keeps no holder id.
func loginNamedEndedRelease(t *testing.T) (*world, *fakeForge, *participant) {
	t.Helper()
	w := newWorld(t)
	f := armedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	wd := []Withdrawal{{Number: 11, ID: "PR_eleven", Form: FormAutoSquash, Head: headA, By: "octo", State: WithdrawnPending}}
	m := message{Record: "rel-v0.16.0", Holder: "octo", Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: w.clock(),
		ExpiresAt: w.clock().Add(time.Hour), Withdrawn: wd, AssistedBy: "abcd:0.0.0"}
	w.pushTipAt("rel-v0.16.0", m.text(), w.clock())
	w.tagOrigin("v0.16.0")
	f.mu.Lock()
	f.prs[11].method = ""
	f.mu.Unlock()
	return w, f, octo
}

// TestAFirstReleaseReservationMintsTheIDAndSettlesLoginNamedWithdrawals: the
// person's first pushing act, a release reservation, mints the holder id,
// settles the ended release's withdrawals named by their login (re-arming
// #11) under it, naming #11 by the id, and leaves the new release's
// reservation, which withdraws what is armed, as the cut the launch check
// reads.
func TestAFirstReleaseReservationMintsTheIDAndSettlesLoginNamedWithdrawals(t *testing.T) {
	w, f, octo := loginNamedEndedRelease(t)
	res, err := octo.do(Request{Record: "rel-v0.17.0"})
	if err != nil {
		t.Fatal(err)
	}
	id := octo.keptHolderID()
	i := slices.IndexFunc(res.Settles, func(s Settle) bool { return s.Subject == "rel-v0.16.0" })
	if i < 0 || !slices.Equal(numbers(res.Settles[i].Settled), []string{"#11"}) {
		t.Fatalf("settles = %+v; want #11 of rel-v0.16.0 settled", res.Settles)
	}
	if got := f.mutations(); len(got) == 0 || got[0] != "enablePullRequestAutoMerge #11" {
		t.Fatalf("mutations = %v; want #11 re-armed first", got)
	}
	old, ows := w.tipWithdrawals("rel-v0.16.0")
	if !old.released() || old.Holder != id || stateOf(ows, 11) != WithdrawnRearmed || len(ows) != 1 || ows[0].By != id {
		t.Fatalf("rel-v0.16.0's tip = %+v; want a tombstone under the new id recording #11 re-armed, withdrawn by the new id", old)
	}
	cut, cws := w.tipWithdrawals("rel-v0.17.0")
	if cut.released() || cut.Holder != id || !slices.Equal(numbers(cws), []string{"#11", "#12", "#13"}) {
		t.Fatalf("rel-v0.17.0's tip = %+v; want the new cut, live under the id, holding #11-#13", cut)
	}
	a, err := ArmedMergesForCut(octo.dir, octo.opts())
	if err != nil {
		t.Fatal(err)
	}
	if a.Release != "rel-v0.17.0" {
		t.Fatalf("the launch reading names %q; want rel-v0.17.0", a.Release)
	}
}
