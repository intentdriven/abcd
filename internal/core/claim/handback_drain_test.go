package claim

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// A hand-back stands until a person changes the record, and a person's own
// reservation over it is a person's: one that lapses hands the issue to no
// drain. So a drain that would take over, or renew, a reservation made over a
// standing hand-back reads the reservation's line down to the tombstone it
// was made over, and backs off when that tombstone still stands.

// handBackUnderQuinn is P's drain handing iss-5 back as user-visible, then
// Q's own reservation over it, at t0: the issue waits on a person, and Q
// holds it by hand.
func handBackUnderQuinn(t *testing.T) (w *world, q, r *participant) {
	t.Helper()
	w = newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	q = w.clone("99+quinn@users.noreply.github.com", "")
	r = w.clone("77+rosa@users.noreply.github.com", "")
	w.fileOnMain("iss-5", "The widget breaks.")
	if _, err := p.do(Request{Record: "iss-5", Footprint: []string{"a.go"}, Drain: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.do(Request{Record: "iss-5", Release: true, HandBack: HandBackUserVisible}); err != nil {
		t.Fatal(err)
	}
	if res, err := q.reserve("iss-5", "b.go"); err != nil || res.HandedBack == nil {
		t.Fatalf("Q's reserve over the hand-back = %+v, %v; want it made with the note", res, err)
	}
	return w, q, r
}

// wantDrainBackOff is a drain's act over iss-5 refused as a back-off naming
// the standing hand-back below the reservation, with nothing pushed.
func wantDrainBackOff(t *testing.T, w *world, who *participant, what string) {
	t.Helper()
	before := w.remoteTip("iss-5")
	res, err := who.do(Request{Record: "iss-5", Footprint: []string{"c.go"}, Drain: true})
	if !errors.Is(err, ErrBackOff) || !strings.Contains(err.Error(), "was made over a hand-back by octo as user-visible at 2026-10-03T14:00:00Z") {
		t.Fatalf("%s over a reservation made over a standing hand-back = %+v, %v; want the hand-back's back-off", what, res, err)
	}
	if w.remoteTip("iss-5") != before {
		t.Fatalf("%s's back-off pushed", what)
	}
}

// TestADrainDoesNotTakeOverALapsedReservationMadeOverAHandBack: Q's own
// reservation over P's hand-back lapses; R's drain backs off it rather than
// take it over, while the record is unchanged, and takes it over once a
// person has changed the record. R's own reservation takes it over either
// way: a person settles hand-backs.
func TestADrainDoesNotTakeOverALapsedReservationMadeOverAHandBack(t *testing.T) {
	w, _, r := handBackUnderQuinn(t)
	w.set(t0.Add(3 * time.Hour))
	wantDrainBackOff(t, w, r, "R's drain")

	w.fileOnMain("iss-5", "The widget breaks; a person settled it: keep the page as it is.")
	res, err := r.do(Request{Record: "iss-5", Footprint: []string{"c.go"}, Drain: true})
	if err != nil || res.Act != ActTakenOver {
		t.Fatalf("R's drain once the record changed = %+v, %v; want it taken over", res, err)
	}
}

// TestAPersonsTakeOverOfALapsedReservationOverAHandBackGoesAhead: the walk
// is a drain's alone; R's own reservation takes Q's lapsed one over.
func TestAPersonsTakeOverOfALapsedReservationOverAHandBackGoesAhead(t *testing.T) {
	w, _, r := handBackUnderQuinn(t)
	w.set(t0.Add(3 * time.Hour))
	if res, err := r.reserve("iss-5", "c.go"); err != nil || res.Act != ActTakenOver {
		t.Fatalf("R's own reserve over Q's lapsed reservation = %+v, %v; want it taken over", res, err)
	}
}

// TestADrainDoesNotRenewAReservationMadeOverAHandBack: Q's own reservation
// over P's hand-back, renewed by Q's own drain from the same copy, lapsed or
// still live, is backed off: a renewal by a drain makes the hold the drain's.
func TestADrainDoesNotRenewAReservationMadeOverAHandBack(t *testing.T) {
	for name, at := range map[string]time.Duration{"lapsed": 3 * time.Hour, "live": 30 * time.Minute} {
		t.Run(name, func(t *testing.T) {
			w, q, _ := handBackUnderQuinn(t)
			w.set(t0.Add(at))
			wantDrainBackOff(t, w, q, "Q's drain over Q's "+name+" reservation")

			w.fileOnMain("iss-5", "The widget breaks; a person settled it: keep the page as it is.")
			res, err := q.do(Request{Record: "iss-5", Footprint: []string{"c.go"}, Drain: true})
			if err != nil || res.Act != ActRenewed {
				t.Fatalf("Q's drain once the record changed = %+v, %v; want it renewed", res, err)
			}
		})
	}
}

// TestADrainRenewsItsOwnReservationMadeOverNoHandBack: a drain's reservation
// over a plain history is renewed as before.
func TestADrainRenewsItsOwnReservationMadeOverNoHandBack(t *testing.T) {
	w := newWorld(t)
	q := w.clone("99+quinn@users.noreply.github.com", "")
	w.fileOnMain("iss-5", "The widget breaks.")
	if _, err := q.reserve("iss-5", "b.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := q.do(Request{Record: "iss-5", Release: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := q.reserve("iss-5", "b.go"); err != nil {
		t.Fatal(err)
	}
	w.set(t0.Add(3 * time.Hour))
	res, err := q.do(Request{Record: "iss-5", Footprint: []string{"b.go"}, Drain: true})
	if err != nil || res.Act != ActRenewed {
		t.Fatalf("Q's drain over its own lapsed reservation = %+v, %v; want it renewed", res, err)
	}
}

// TestADrainBacksOffAReservationLineHoldingAMerge: a reservation branch is
// a single line of commits, and abcd never writes a merge on one, so a line
// holding a merge is one abcd cannot judge, and a drain backs off it.
func TestADrainBacksOffAReservationLineHoldingAMerge(t *testing.T) {
	w, _, r := handBackUnderQuinn(t)
	w.set(t0.Add(3 * time.Hour))
	tip := w.remoteTip("iss-5")
	msg := runGit(t, w.bare, "log", "-1", "--format=%B", tip)
	empty := "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
	id := []string{"-c", "user.name=Test", "-c", "user.email=test@example.com"}
	root := runGit(t, w.bare, append(id, "commit-tree", empty, "-m", msg)...)
	merge := runGit(t, w.bare, append(id, "commit-tree", empty, "-p", root, "-p", tip, "-m", msg)...)
	runGit(t, w.bare, "update-ref", RefPrefix+"iss-5", merge, tip)
	before := w.remoteTip("iss-5")
	res, err := r.do(Request{Record: "iss-5", Footprint: []string{"c.go"}, Drain: true})
	if !errors.Is(err, ErrBackOff) || !strings.Contains(err.Error(), "more than one parent") {
		t.Fatalf("R's drain over a reservation line holding a merge = %+v, %v; want a back-off naming the merge", res, err)
	}
	if w.remoteTip("iss-5") != before {
		t.Fatal("the back-off pushed")
	}
}
