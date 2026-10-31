package claim

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

// A holder id names no account the forge's record of pushes could be matched
// against, so a release's reading holds one of the reader's own ids to the
// reader's login (namesPusher): a commit another account pushed naming the
// reader's id is not the reader's push, as a commit naming their login is not
// (iss-2610290809526033).

// rewrittenOverNaming is rewrittenOver with the commits naming holder, as holder
// and as the withdrawer of #11.
func (w *world) rewrittenOverNaming(id, holder string) (record, tomb string) {
	w.t.Helper()
	wd := []Withdrawal{{Number: 11, ID: "PR_eleven", Form: FormAutoSquash, Head: headA, By: holder, State: WithdrawnPending}}
	rec := message{Record: id, Holder: holder, Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: t0, ExpiresAt: t0.Add(time.Hour), Withdrawn: wd}
	record = w.pushChild(id, w.remoteTip(id), rec.text(), t0)
	ts := rec
	ts.ReleasedAt, ts.ExpiresAt = t0, t0
	tomb = w.pushChild(id, record, ts.text(), t0)
	return record, tomb
}

// TestACommitAnotherLoginPushedNamingTheReadersHolderIDIsNeverReArmed: commits
// naming octo's holder id that the record says mal pushed lie above octo's
// newest push, so octo's next act reads nothing from them, re-arms nothing and
// counts them; the same commits pushed by octo are octo's record, and are
// re-armed.
func TestACommitAnotherLoginPushedNamingTheReadersHolderIDIsNeverReArmed(t *testing.T) {
	w, f, octo := releasedWithNothingArmed(t)
	record, tomb := w.rewrittenOverNaming("rel-v0.16.0", octo.keptHolderID())
	f.pusherOf[record], f.pusherOf[tomb] = "mal", "mal"
	res, err := octo.reserve("iss-8", "o.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.mutations()) != 0 || f.pr(11).method != "" {
		t.Fatalf("mutations %v, #11 %+v; want nothing re-armed for a record mal pushed naming octo's id", f.mutations(), f.pr(11))
	}
	if len(res.Settles) != 1 || !slices.Equal(res.Settles[0].NotRead, []PushCount{{Pusher: "mal", Commits: 2}}) || res.Settles[0].Commit != "" {
		t.Fatalf("settles = %+v; want mal's two commits counted and nothing pushed", res.Settles)
	}
	delete(f.pusherOf, record)
	delete(f.pusherOf, tomb)
	forgetVerdicts(t, octo.dir)
	if _, err := octo.reserve("iss-9", "p.go"); err != nil {
		t.Fatal(err)
	}
	if got := f.mutations(); !slices.Equal(got, []string{"enablePullRequestAutoMerge #11"}) {
		t.Fatalf("mutations = %v; want #11 re-armed once octo's own push recorded it", got)
	}
}

// malEndedOctosCutNamingOctosID is octo's live cut of rel-v0.16.0, which
// withdrew #11, #12 and #13, with a reservation naming octo's holder id and a
// tombstone over it that the forge's record says mal pushed.
func malEndedOctosCutNamingOctosID(t *testing.T) (*world, *fakeForge, *participant) {
	t.Helper()
	w := newWorld(t)
	f := armedForge(t)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	m, ws := w.tipWithdrawals("rel-v0.16.0")
	if m.Holder != octo.keptHolderID() {
		t.Fatalf("octo's cut names %q; want octo's holder id", m.Holder)
	}
	rival := message{Record: m.Record, Holder: m.Holder, Copy: "9f2c41d07be35a6c80e1f4b2a7d39c55", ClaimedAt: w.clock(), ExpiresAt: w.clock().Add(time.Hour), Withdrawn: ws, AssistedBy: m.AssistedBy}
	w.pushAs(f, "rel-v0.16.0", "mal", rival.text())
	tomb := rival
	tomb.ReleasedAt, tomb.ExpiresAt = w.clock(), w.clock()
	w.pushAs(f, "rel-v0.16.0", "mal", tomb.text())
	w.set(t0.Add(time.Minute))
	f.reset()
	return w, f, octo
}

// TestAReservationAndTombstoneAnotherLoginPushedNamingTheHoldersIDEndNothingForThem:
// over octo's live cut, mal pushes a reservation naming octo's holder id and
// a tombstone over it. To octo, neither is octo's push, so the release is not
// ended for octo and nothing octo withdrew is re-armed; octo's own --release
// still ends it.
func TestAReservationAndTombstoneAnotherLoginPushedNamingTheHoldersIDEndNothingForThem(t *testing.T) {
	_, f, octo := malEndedOctosCutNamingOctosID(t)
	res, err := octo.reserve("iss-8", "o.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range res.Settles {
		if len(s.Settled) != 0 || s.Commit != "" {
			t.Fatalf("settles = %+v; want nothing ended by mal's pushes naming octo's id", res.Settles)
		}
	}
	if got := f.mutations(); len(got) != 0 {
		t.Fatalf("mutations %v; want nothing re-armed by mal's pushes naming octo's id", got)
	}
	if _, err := octo.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	if got := f.mutations(); len(got) == 0 {
		t.Fatal("octo's own --release re-armed nothing")
	}
}

// TestOctosReadingsOfACutMalEndedNamingOctosIDReadItLive: the readings hold
// octo's id to octo's login as the acts do. Over mal's reservation and
// tombstone naming octo's id, octo's listing shows the cut not ended, holding
// #11-#13 and nothing due; octo's launch reading names it as octo's live cut;
// and a second release octo reserves backs off while it is cut.
func TestOctosReadingsOfACutMalEndedNamingOctosIDReadItLive(t *testing.T) {
	w, f, octo := malEndedOctosCutNamingOctosID(t)
	id := octo.keptHolderID()
	rw := listingOf(t, w, octo, "rel-v0.16.0")
	if rw.Ended || len(rw.Due) != 0 || !slices.Equal(numbers(rw.Held), []string{"#11", "#12", "#13"}) || rw.NotEnded != id {
		t.Fatalf("octo's listing = %+v; want the cut not ended, holding #11-#13, nothing due, its tombstone named as not octo's", rw)
	}
	a, err := ArmedMergesForCut(octo.dir, octo.opts())
	if err != nil {
		t.Fatal(err)
	}
	if a.Release != "rel-v0.16.0" {
		t.Fatalf("octo's launch reading names %q as octo's cut; want rel-v0.16.0, still live", a.Release)
	}
	tip := w.remoteTip("rel-v0.17.0")
	_, err = octo.do(Request{Record: "rel-v0.17.0"})
	if !errors.Is(err, ErrBackOff) || !strings.Contains(err.Error(), "rel-v0.16.0 is being cut by "+id) {
		t.Fatalf("octo's second release = %v; want a back-off naming the live cut", err)
	}
	if w.remoteTip("rel-v0.17.0") != tip || len(f.mutations()) != 0 {
		t.Fatalf("the second release moved its branch or the forge: %v", f.mutations())
	}
}
