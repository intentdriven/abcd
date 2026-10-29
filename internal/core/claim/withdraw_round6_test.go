package claim

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// heldOver is octo's live rel-v0.16.0, reserved over commentedForge, so #11
// (squash), #12 (queue) and #13 (merge) are withdrawn and each commented on.
func heldOver(t *testing.T) (*world, *fakeForge, *participant) {
	t.Helper()
	w := newWorld(t)
	f := commentedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	return w, f, octo
}

// pushAs pushes text over id's tip on origin as one commit the forge's record
// of pushes credits to login.
func (w *world) pushAs(f *fakeForge, id, login, text string) string {
	w.t.Helper()
	c := w.pushChild(id, w.remoteTip(id), text, w.clock())
	f.pusherOf[c] = login
	return c
}

// withState is ws with #n's state set to state.
func withState(ws []Withdrawal, n int, state string) []Withdrawal {
	out := slices.Clone(ws)
	for i := range out {
		if out[i].Number == n {
			out[i].State = state
		}
	}
	return out
}

// listingOf is what peers reads of id for reader, fetched first.
func listingOf(t *testing.T, w *world, p *participant, id, reader string) ReleaseWithdrawals {
	t.Helper()
	if _, err := Fetch(p.dir, p.opts()); err != nil {
		t.Fatal(err)
	}
	snap, err := View(p.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	for _, rw := range snap.ReleaseWithdrawals(p.dir, reader, w.clock()) {
		if rw.Subject == id {
			return rw
		}
	}
	return ReleaseWithdrawals{}
}

// rearmedComments is the sentence each of ns posts when it is re-armed.
func rearmedComments(ns ...int) []string {
	var out []string
	for _, n := range ns {
		out = append(out, fmt.Sprintf("#%d abcd release hold: the merge of #%d is re-armed; the hold for release v0.16.0 has ended.", n, n))
	}
	return out
}

// allRearmed reports whether #11, #12 and #13 are armed again as they were.
func allRearmed(f *fakeForge) bool {
	return f.pr(11).method == "SQUASH" && f.pr(12).queued && f.pr(13).method == "MERGE"
}

// TestEachShapeAboveAPersonsNewestPushLeavesTheirRecordAsTheyPushedIt is the
// matrix of what other logins push over octo's live release, ported from the
// round-5 probes: whatever the commits above octo's newest push record, octo's
// --release re-arms all three of octo's withdrawals from octo's own newest
// push, posts one re-armed comment for each, and names the commits above it,
// counted, once; peers lists the same count beforehand, and octo's three as
// held. A shape that buries octo's newest push past the walk acts on
// nothing, and says so, on the act and on peers.
func TestEachShapeAboveAPersonsNewestPushLeavesTheirRecordAsTheyPushedIt(t *testing.T) {
	for name, c := range map[string]struct {
		plant  func(w *world, f *fakeForge)
		above  int  // commits mal pushed above octo's newest push
		buried bool // octo's newest push lies past the walk
	}{
		"one entry dropped, then re-added kept": {plant: func(w *world, f *fakeForge) {
			m, ws := w.tipWithdrawals("rel-v0.16.0")
			m.Withdrawn = withoutWithdrawal(ws, 11)
			w.pushAs(f, "rel-v0.16.0", "mal", m.text())
			m.Withdrawn = withState(ws, 11, WithdrawnKept)
			w.pushAs(f, "rel-v0.16.0", "mal", m.text())
		}, above: 2},
		"one entry dropped, then re-added re-armed": {plant: func(w *world, f *fakeForge) {
			m, ws := w.tipWithdrawals("rel-v0.16.0")
			m.Withdrawn = withoutWithdrawal(ws, 11)
			w.pushAs(f, "rel-v0.16.0", "mal", m.text())
			m.Withdrawn = withState(ws, 11, WithdrawnRearmed)
			w.pushAs(f, "rel-v0.16.0", "mal", m.text())
		}, above: 2},
		"two changes in a row, moved then closed": {plant: func(w *world, f *fakeForge) {
			m, ws := w.tipWithdrawals("rel-v0.16.0")
			m.Withdrawn = withState(ws, 11, WithdrawnMoved)
			w.pushAs(f, "rel-v0.16.0", "mal", m.text())
			m.Withdrawn = withState(ws, 11, WithdrawnClosed)
			w.pushAs(f, "rel-v0.16.0", "mal", m.text())
		}, above: 2},
		"a change beneath an unreadable commit": {plant: func(w *world, f *fakeForge) {
			m, ws := w.tipWithdrawals("rel-v0.16.0")
			m.Withdrawn = withState(ws, 11, WithdrawnKept)
			w.pushAs(f, "rel-v0.16.0", "mal", m.text())
			w.pushAs(f, "rel-v0.16.0", "mal", "garbage\n\nnot a reservation\n")
		}, above: 2},
		"an unreadable commit, then a change": {plant: func(w *world, f *fakeForge) {
			m, ws := w.tipWithdrawals("rel-v0.16.0")
			w.pushAs(f, "rel-v0.16.0", "mal", "garbage\n\nnot a reservation\n")
			m.Withdrawn = withState(ws, 11, WithdrawnKept)
			w.pushAs(f, "rel-v0.16.0", "mal", m.text())
		}, above: 2},
		"a change beneath 65 commits": {plant: func(w *world, f *fakeForge) {
			m, ws := w.tipWithdrawals("rel-v0.16.0")
			m.Withdrawn = withState(ws, 11, WithdrawnKept)
			tip := w.pushChildren("rel-v0.16.0", w.remoteTip("rel-v0.16.0"), m, maxChain+1)
			for _, c := range strings.Fields(runGit(t, w.bare, "rev-list", "-n", "65", tip)) {
				f.pusherOf[c] = "mal"
			}
		}, buried: true},
		"a tombstone dated outside the window": {plant: func(w *world, f *fakeForge) {
			m, ws := w.tipWithdrawals("rel-v0.16.0")
			old := w.clock().Add(-15 * 24 * time.Hour)
			tm := message{Record: m.Record, Holder: "octo", Copy: m.Copy, ClaimedAt: old, ExpiresAt: old, ReleasedAt: old,
				Withdrawn: withState(withState(withState(ws, 11, WithdrawnRearmed), 12, WithdrawnRearmed), 13, WithdrawnRearmed), AssistedBy: m.AssistedBy}
			w.pushAs(f, "rel-v0.16.0", "mal", tm.text())
		}, above: 1},
	} {
		t.Run(name, func(t *testing.T) {
			w, f, octo := heldOver(t)
			c.plant(w, f)
			w.set(t0.Add(time.Minute))
			rw := listingOf(t, w, octo, "rel-v0.16.0", "octo")
			commented, tip := len(f.posted()), w.remoteTip("rel-v0.16.0")
			f.reset()
			res, err := octo.do(Request{Record: "rel-v0.16.0", Release: true})
			if c.buried {
				const why = "64 commits"
				if !strings.Contains(rw.Unplaced, why) || len(rw.Held) != 0 || !strings.Contains(strings.Join(rw.Lines(), "\n"), "abcd acts on none of octo's withdrawals on rel-v0.16.0") {
					t.Fatalf("listing = %+v; want octo's newest push named as not found, and nothing held", rw)
				}
				if err == nil || !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), why) || len(f.mutations()) != 0 ||
					w.remoteTip("rel-v0.16.0") != tip || len(f.posted()) != commented || f.pr(11).method != "" {
					t.Fatalf("release = %v, mutations %v; want it refused with nothing pushed, re-armed or commented on", err, f.mutations())
				}
				return
			}
			want := []PushCount{{Pusher: "mal", Commits: c.above}}
			if !slices.Equal(rw.NotRead, want) || rw.Unplaced != "" || !slices.Equal(numbersOf(rw.Held), []int{11, 12, 13}) {
				t.Fatalf("listing = %+v; want octo's three held and mal's %d counted", rw, c.above)
			}
			if err != nil {
				t.Fatal(err)
			}
			if !allRearmed(f) {
				t.Fatalf("forge = %+v %+v %+v; want all three re-armed in their own forms", f.pr(11), f.pr(12), f.pr(13))
			}
			if got := f.posted()[commented:]; !slices.Equal(got, rearmedComments(11, 12, 13)) {
				t.Fatalf("comments = %q; want one re-armed sentence for each", got)
			}
			noun := "commits"
			if c.above == 1 {
				noun = "commit"
			}
			if notes := settleNotes(res.Settles); !strings.Contains(notes, fmt.Sprintf("%d %s above octo's newest push to rel-v0.16.0's branch, pushed by mal (%d), are not read", c.above, noun, c.above)) {
				t.Fatalf("notes = %q; want mal's commits counted as not read", notes)
			}
			if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnRearmed || stateOf(ws, 12) != WithdrawnRearmed || stateOf(ws, 13) != WithdrawnRearmed {
				t.Fatalf("tombstone records %+v; want all three re-armed", ws)
			}
			if again, err := octo.reserve("iss-8", "o.go"); err != nil || len(again.Settles) != 0 {
				t.Fatalf("next act = %+v, %v; want nothing named again", again.Settles, err)
			}
		})
	}
}

// TestARenewalOverAnotherLoginsTombstoneKeepsAdmissions: octo admits #13,
// mal pushes a tombstone over the live release, and octo renews it: the
// renewal is read from octo's own newest push, so #13 stays armed and
// admitted, nothing is withdrawn or commented on twice, and mal's tombstone is
// named, counted, once.
func TestARenewalOverAnotherLoginsTombstoneKeepsAdmissions(t *testing.T) {
	w, f, octo := heldOver(t)
	if _, err := octo.do(Request{Record: "rel-v0.16.0", Admit: 13}); err != nil {
		t.Fatal(err)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	tm := message{Record: m.Record, Holder: "octo", Copy: m.Copy, ClaimedAt: w.clock(), ExpiresAt: w.clock(), ReleasedAt: w.clock(), Withdrawn: ws, AssistedBy: m.AssistedBy}
	w.pushAs(f, "rel-v0.16.0", "mal", tm.text())
	w.set(t0.Add(time.Minute))
	if rw := listingOf(t, w, octo, "rel-v0.16.0", "octo"); !slices.Equal(rw.NotRead, []PushCount{{Pusher: "mal", Commits: 1}}) ||
		!slices.Equal(numbersOf(rw.Admitted), []int{13}) || !slices.Equal(numbersOf(rw.Held), []int{11, 12}) {
		t.Fatalf("listing = %+v; want #13 admitted, #11 and #12 held, mal's tombstone counted", rw)
	}
	commented := len(f.posted())
	f.reset()
	res, err := octo.do(Request{Record: "rel-v0.16.0"})
	if err != nil {
		t.Fatal(err)
	}
	if f.pr(13).method != "MERGE" || f.pr(11).method != "" || f.pr(12).queued || len(f.mutations()) != 0 {
		t.Fatalf("forge = %+v %+v %+v, mutations %v; want #13 armed, #11 and #12 still withdrawn, nothing done", f.pr(11), f.pr(12), f.pr(13), f.mutations())
	}
	if len(f.posted()) != commented {
		t.Fatalf("comments = %q; want none posted by the renewal", f.posted()[commented:])
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 13) != WithdrawnAdmitted || stateOf(ws, 11) != WithdrawnPending || stateOf(ws, 12) != WithdrawnPending {
		t.Fatalf("renewal records %+v; want #13 admitted, #11 and #12 withdrawn", ws)
	}
	if !strings.Contains(settleNotes(res.Settles), "1 commit above octo's newest push to rel-v0.16.0's branch, pushed by mal (1), are not read") {
		t.Fatalf("settles = %+v; want mal's tombstone counted", res.Settles)
	}
	if rw := listingOf(t, w, octo, "rel-v0.16.0", "octo"); len(rw.NotRead) != 0 || rw.Unplaced != "" {
		t.Fatalf("listing after = %+v; want octo's own renewal read, nothing above it", rw)
	}
}

// TestAPersonsNewestPushNotFoundActsOnNothingForThem: a record of pushes that
// cannot be read leaves octo's newest push unknown, so octo's --release is
// refused with nothing pushed, re-armed or commented on, peers names why, and
// the same act once the record reads re-arms all three.
func TestAPersonsNewestPushNotFoundActsOnNothingForThem(t *testing.T) {
	w, f, octo := heldOver(t)
	f.pushersErr = errors.New("HTTP 502")
	rw := listingOf(t, w, octo, "rel-v0.16.0", "octo")
	if !strings.Contains(rw.Unplaced, "HTTP 502") || rw.Reader != "octo" || len(rw.Held) != 0 {
		t.Fatalf("listing = %+v; want why named for octo, and nothing held", rw)
	}
	commented, tip := len(f.posted()), w.remoteTip("rel-v0.16.0")
	f.reset()
	_, err := octo.do(Request{Record: "rel-v0.16.0", Release: true})
	if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "abcd acts on none of octo's withdrawals on rel-v0.16.0") ||
		len(f.mutations()) != 0 || w.remoteTip("rel-v0.16.0") != tip || len(f.posted()) != commented {
		t.Fatalf("release = %v, mutations %v; want it refused with nothing done", err, f.mutations())
	}
	f.pushersErr = nil
	if _, err := octo.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil || !allRearmed(f) {
		t.Fatalf("release = %v, forge %+v %+v %+v; want all three re-armed", err, f.pr(11), f.pr(12), f.pr(13))
	}
}

// TestTheHonestPathIsUnchanged: octo reserves and releases with no one else
// pushing: peers lists the three held and nothing else, the release re-arms
// all three with one comment each and names nothing, and the next act has
// nothing to say.
func TestTheHonestPathIsUnchanged(t *testing.T) {
	w, f, octo := heldOver(t)
	rw := listingOf(t, w, octo, "rel-v0.16.0", "octo")
	if !slices.Equal(numbersOf(rw.Held), []int{11, 12, 13}) || len(rw.NotRead) != 0 || rw.Unplaced != "" || rw.NotEnded != "" {
		t.Fatalf("listing = %+v; want the three held and nothing else", rw)
	}
	commented := len(f.posted())
	res, err := octo.do(Request{Record: "rel-v0.16.0", Release: true})
	if err != nil {
		t.Fatal(err)
	}
	if !allRearmed(f) || !slices.Equal(f.posted()[commented:], rearmedComments(11, 12, 13)) || settleNotes(res.Settles) != "" {
		t.Fatalf("forge %+v %+v %+v, comments %q, notes %q; want all three re-armed and commented on, nothing named",
			f.pr(11), f.pr(12), f.pr(13), f.posted()[commented:], settleNotes(res.Settles))
	}
	if again, err := octo.reserve("iss-8", "o.go"); err != nil || len(again.Settles) != 0 {
		t.Fatalf("next act = %+v, %v; want nothing", again.Settles, err)
	}
}

// TestAnAdmitAfterATakeOverNamesTheRemedyAndItWorks: ann withdraws #11 on
// her release, bob takes it over after it lapses, and ann's --admit 11 is
// refused naming the remedy: ann takes the release back with --take-over and
// admits it. Doing so admits #11, arms it under ann's account, and posts the
// admitted sentence naming ann.
func TestAnAdmitAfterATakeOverNamesTheRemedyAndItWorks(t *testing.T) {
	w := newWorld(t)
	f := commentedForge(t)
	ann := w.clone("1111+ann@users.noreply.github.com", "")
	if _, err := ann.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	w.set(w.clock().Add(25 * time.Hour))
	bob := w.clone("2222+bob@users.noreply.github.com", "")
	if _, err := bob.do(Request{Record: "rel-v0.16.0", TakeOver: true}); err != nil {
		t.Fatalf("bob's take-over: %v", err)
	}
	commented, tip := len(f.posted()), w.remoteTip("rel-v0.16.0")
	f.reset()
	_, err := ann.do(Request{Record: "rel-v0.16.0", Admit: 11})
	if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), "ann takes the release back with --take-over and admits it") {
		t.Fatalf("ann's admit = %v; want it refused, naming the remedy", err)
	}
	if len(f.mutations()) != 0 || len(f.posted()) != commented || w.remoteTip("rel-v0.16.0") != tip {
		t.Fatalf("mutations %v, comments %q; want nothing done", f.mutations(), f.posted()[commented:])
	}
	w.set(w.clock().Add(25 * time.Hour))
	if _, err := ann.do(Request{Record: "rel-v0.16.0", TakeOver: true}); err != nil {
		t.Fatalf("ann's take-over: %v", err)
	}
	if _, err := ann.do(Request{Record: "rel-v0.16.0", Admit: 11}); err != nil {
		t.Fatalf("ann's admit after taking it back: %v", err)
	}
	if f.pr(11).method != "SQUASH" {
		t.Fatalf("#11 = %+v; want it armed again", f.pr(11))
	}
	const admitted = "#11 abcd release hold: the merge of #11 is admitted into release v0.16.0 by ann."
	if !slices.Contains(f.posted(), admitted) {
		t.Fatalf("comments = %q; want %q", f.posted(), admitted)
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 11) != WithdrawnAdmitted {
		t.Fatalf("tip records %+v; want #11 admitted", ws)
	}
}

// TestAnEndThatCannotBeSettledNeverReArms: octo's own tombstone sits over a
// take-over an app pushed, so who holds the release, and whether it ended, is
// not known: octo's next act re-arms nothing, comments on nothing, and names
// why.
func TestAnEndThatCannotBeSettledNeverReArms(t *testing.T) {
	w, f, octo := heldOver(t)
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	bob := message{Record: m.Record, Holder: "bob", Copy: "0123456789abcdef0123456789abcdef", ClaimedAt: w.clock(), ExpiresAt: w.clock().Add(time.Hour), Withdrawn: ws, AssistedBy: m.AssistedBy}
	w.pushAs(f, "rel-v0.16.0", "app", bob.text())
	tm := message{Record: m.Record, Holder: "octo", Copy: m.Copy, ClaimedAt: w.clock(), ExpiresAt: w.clock(), ReleasedAt: w.clock(), Withdrawn: ws, AssistedBy: m.AssistedBy}
	w.pushAs(f, "rel-v0.16.0", "octo", tm.text())
	w.set(t0.Add(time.Minute))
	commented := len(f.posted())
	f.reset()
	s := settleAfter(t, octo)
	if !strings.Contains(s.Unplaced, "whether the release has ended is not known") || len(f.mutations()) != 0 || len(f.posted()) != commented || f.pr(11).method != "" {
		t.Fatalf("settle = %+v, mutations %v; want nothing re-armed and why named", s, f.mutations())
	}
}

// TestAnActNeverTakesAnEndFromTheKeptPlacements: what a reading kept for a
// tip is never what an act reads an end by: with the placements kept saying
// octo pushed the tombstone mal pushed, octo's next act asks the record
// again, re-arms nothing, and names the tombstone as ending nothing.
func TestAnActNeverTakesAnEndFromTheKeptPlacements(t *testing.T) {
	w, f, octo := heldOver(t)
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	tm := message{Record: m.Record, Holder: "octo", Copy: m.Copy, ClaimedAt: w.clock(), ExpiresAt: w.clock(), ReleasedAt: w.clock(), Withdrawn: ws, AssistedBy: m.AssistedBy}
	tomb := w.pushAs(f, "rel-v0.16.0", "mal", tm.text())
	if _, err := Fetch(octo.dir, octo.opts()); err != nil {
		t.Fatal(err)
	}
	c := readTipCache(octo.dir, "rel-v0.16.0", tomb)
	c.place(tomb, pushedByLogin, "octo")
	c.place(strings.TrimSpace(runGit(t, w.bare, "rev-parse", tomb+"^")), pushedByLogin, "octo")
	c.save()
	w.set(t0.Add(time.Minute))
	asks := f.pushAsks
	f.reset()
	s := settleAfter(t, octo)
	if f.pushAsks == asks || f.pr(11).method != "" || len(f.mutations()) != 0 || s.NotEnded != "octo" {
		t.Fatalf("settle = %+v, asks %d then %d, mutations %v; want the record asked, nothing re-armed", s, asks, f.pushAsks, f.mutations())
	}
}

// TestASecondReleaseReadsTheFirstsEndFromTheRecord: a tombstone mal pushed
// over octo's live release ends nothing, so bob's rel-v0.17.0 backs off
// naming it; once octo's own --release ends it, bob reserves.
func TestASecondReleaseReadsTheFirstsEndFromTheRecord(t *testing.T) {
	w, f, octo := heldOver(t)
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	tm := message{Record: m.Record, Holder: "octo", Copy: m.Copy, ClaimedAt: w.clock(), ExpiresAt: w.clock(), ReleasedAt: w.clock(), Withdrawn: ws, AssistedBy: m.AssistedBy}
	w.pushAs(f, "rel-v0.16.0", "mal", tm.text())
	bob := w.clone("2222+bob@users.noreply.github.com", "")
	_, err := bob.do(Request{Record: "rel-v0.17.0"})
	if err == nil || !strings.Contains(err.Error(), "rel-v0.16.0 is being cut by octo") || !strings.Contains(err.Error(), "its tip is a tombstone octo did not push") {
		t.Fatalf("bob's reserve = %v; want it backing off, the tombstone named as ending nothing", err)
	}
	if _, err := octo.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := bob.do(Request{Record: "rel-v0.17.0"}); err != nil {
		t.Fatalf("bob's reserve after octo's --release = %v; want it reserved", err)
	}
}

// TestACommitAnotherLoginPushedNamingThemselvesHolderLeavesTheRecordAsPushed:
// mal pushes a reservation naming mal as holder over octo's live release,
// changing octo's entries, then mal's own tombstone: the release ends with
// mal's tombstone, and octo's next act re-arms all three from octo's own
// newest push.
func TestACommitAnotherLoginPushedNamingThemselvesHolderLeavesTheRecordAsPushed(t *testing.T) {
	w, f, octo := heldOver(t)
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	changed := withState(withoutWithdrawal(ws, 13), 11, WithdrawnKept)
	held := message{Record: m.Record, Holder: "mal", Copy: "0123456789abcdef0123456789abcdef", ClaimedAt: w.clock(), ExpiresAt: w.clock().Add(time.Hour), Withdrawn: changed, AssistedBy: m.AssistedBy}
	w.pushAs(f, "rel-v0.16.0", "mal", held.text())
	held.ReleasedAt, held.ExpiresAt = w.clock(), w.clock()
	w.pushAs(f, "rel-v0.16.0", "mal", held.text())
	w.set(t0.Add(time.Minute))
	commented := len(f.posted())
	f.reset()
	s := settleAfter(t, octo)
	if !allRearmed(f) || !slices.Equal(f.posted()[commented:], rearmedComments(11, 12, 13)) {
		t.Fatalf("settle = %+v, forge %+v %+v %+v; want all three re-armed from octo's own push", s, f.pr(11), f.pr(12), f.pr(13))
	}
	if !slices.Equal(s.NotRead, []PushCount{{Pusher: "mal", Commits: 2}}) {
		t.Fatalf("settle = %+v; want mal's two commits counted", s)
	}
}

// TestAnAppPushedTombstoneOfTheHolderNamesTheApp: a tombstone naming octo
// that an app pushed leaves octo's newest push unknown, and the line says an
// app pushed it.
func TestAnAppPushedTombstoneOfTheHolderNamesTheApp(t *testing.T) {
	w, f, octo := heldOver(t)
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	tm := message{Record: m.Record, Holder: "octo", Copy: m.Copy, ClaimedAt: w.clock(), ExpiresAt: w.clock(), ReleasedAt: w.clock(), Withdrawn: ws, AssistedBy: m.AssistedBy}
	tomb := w.pushAs(f, "rel-v0.16.0", "app", tm.text())
	w.set(t0.Add(time.Minute))
	f.reset()
	s := settleAfter(t, octo)
	if !strings.Contains(s.Unplaced, "an app pushed commit "+tomb[:12]+" of rel-v0.16.0's branch, which says nothing of which person sent it") || len(f.mutations()) != 0 {
		t.Fatalf("settle = %+v, mutations %v; want the app named and nothing done", s, f.mutations())
	}
}

// TestTheLaunchReadingAsksTheRecordOnlyForTheReadersReleases: the cut's
// reading of armed merges reads the record of pushes only for a release the
// reader holds, or a tombstone, never for another person's live release.
func TestTheLaunchReadingAsksTheRecordOnlyForTheReadersReleases(t *testing.T) {
	w := newWorld(t)
	f := commentedForge(t)
	bob := w.clone("2222+bob@users.noreply.github.com", "")
	if _, err := bob.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := Fetch(octo.dir, octo.opts()); err != nil {
		t.Fatal(err)
	}
	asks := f.pushAsks
	if _, err := ArmedMergesForCut(octo.dir, octo.opts()); err != nil {
		t.Fatal(err)
	}
	if f.pushAsks != asks {
		t.Fatalf("the reading asked the record %d times for bob's release; want none", f.pushAsks-asks)
	}
}

// TestTheHoldReadsAnEndItCannotSettleAsNotEnded: each way the reading of a
// release's end can fail reads as not ended, with why, and the reading of
// an untagged release whose end this copy has failed to settle for longer
// than lostWindow reads it as ended.
func TestTheHoldReadsAnEndItCannotSettleAsNotEnded(t *testing.T) {
	w, f, octo := heldOver(t)
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	tm := message{Record: m.Record, Holder: "octo", Copy: m.Copy, ClaimedAt: w.clock(), ExpiresAt: w.clock(), ReleasedAt: w.clock(), Withdrawn: ws, AssistedBy: m.AssistedBy}
	tomb := w.pushAs(f, "rel-v0.16.0", "octo", tm.text())
	if _, err := Fetch(octo.dir, octo.opts()); err != nil {
		t.Fatal(err)
	}
	snap, err := View(octo.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	var r Reservation
	for _, x := range snap.Reservations {
		if x.Record == "rel-v0.16.0" {
			r = x
		}
	}
	if r.Commit != tomb || r.ReleasedBy != ReleasedByTombstone {
		t.Fatalf("reservation = %+v; want octo's tombstone", r)
	}
	if _, notEnded, why := asCut(octo.dir, r, w.clock()); notEnded || why != "" {
		t.Fatalf("octo's own tombstone: notEnded %v, why %q; want ended", notEnded, why)
	}
	nowhere := filepath.Join(t.TempDir(), "none")
	if err := os.MkdirAll(nowhere, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, notEnded, why := asCut(nowhere, r, w.clock()); !notEnded || !strings.Contains(why, "whether v0.16.0 is on origin could not be read") {
		t.Fatalf("tags not read: notEnded %v, why %q; want not ended, why named", notEnded, why)
	}
	bogus := r
	bogus.Commit = strings.Repeat("e", 40)
	if _, notEnded, why := asCut(octo.dir, bogus, w.clock()); !notEnded || !strings.Contains(why, "the history of its branch could not be read") {
		t.Fatalf("branch not read: notEnded %v, why %q; want not ended, why named", notEnded, why)
	}
	for _, gate := range []Reservation{func() Reservation { x := r; x.Record = "iss-8"; return x }(), func() Reservation { x := r; x.State = StateLive; return x }()} {
		if got, notEnded, why := asCut(octo.dir, gate, w.clock()); notEnded || why != "" || got.Commit != gate.Commit {
			t.Fatalf("gate %+v: notEnded %v, why %q; want it read as listed", gate, notEnded, why)
		}
	}
	f.pushersErr = errors.New("HTTP 502")
	_ = os.RemoveAll(filepath.Join(octo.dir, filepath.FromSlash(tipCacheDirRel)))
	if _, notEnded, why := asCut(octo.dir, r, w.clock()); !notEnded || !strings.Contains(why, "HTTP 502") {
		t.Fatalf("record not read: notEnded %v, why %q; want not ended, why named", notEnded, why)
	}
	if _, notEnded, why := asCut(octo.dir, r, w.clock().Add(lostWindow+time.Hour)); notEnded || why != "" {
		t.Fatalf("record not read for longer than lostWindow: notEnded %v, why %q; want it read as ended", notEnded, why)
	}
}

// TestATaggedReleaseUnderAnUnreadableTipReArmsFromTheHoldersOwnPush: an
// unreadable commit another login pushed over octo's live release does not
// keep the tag from ending it: octo's next act re-arms all three from octo's
// own newest push.
func TestATaggedReleaseUnderAnUnreadableTipReArmsFromTheHoldersOwnPush(t *testing.T) {
	w, f, octo := heldOver(t)
	w.pushAs(f, "rel-v0.16.0", "mal", "garbage\n\nnot a reservation\n")
	w.tagOrigin("v0.16.0")
	w.set(t0.Add(time.Minute))
	commented := len(f.posted())
	f.reset()
	s := settleAfter(t, octo)
	if !allRearmed(f) || !slices.Equal(f.posted()[commented:], rearmedComments(11, 12, 13)) || !slices.Equal(s.NotRead, []PushCount{{Pusher: "mal", Commits: 1}}) {
		t.Fatalf("settle = %+v, forge %+v %+v %+v; want all three re-armed and mal's commit counted", s, f.pr(11), f.pr(12), f.pr(13))
	}
}
