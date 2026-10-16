package claim

import (
	"errors"
	"strings"
	"testing"
)

// A record finished on origin's default branch is not reserved (spc-2610161044461448
// step 1, iss-2610160957404827): any reservation act other than a release or a
// renewal backs off, whether or not a reservation tip exists for it, and nothing
// is pushed. The listing still counts such a tip released (C15).

// finishOnMain moves id's record into folder on origin's default branch.
func (w *world) finishOnMain(id, folder string) {
	w.t.Helper()
	w.seed.Write(".abcd/work/issues/"+folder+"/"+id+"-the-widget.md", "---\nid: "+id+"\n---\n# The widget\n")
	w.seed.Commit(folder + " " + id)
	w.seed.Git("push", "-q", "origin", "main")
}

// wantFinishedBackOff fails unless err is the back-off naming id as finished on
// origin's default branch, and origin's tip on id is still before.
func wantFinishedBackOff(t *testing.T, w *world, id string, err error, before string) {
	t.Helper()
	if !errors.Is(err, ErrBackOff) {
		t.Fatalf("reserving %s, finished on origin's default branch: %v; want the back-off", id, err)
	}
	if want := id + " is finished on origin's default branch"; !strings.Contains(err.Error(), want) {
		t.Fatalf("the back-off = %q; want it to say %q", err.Error(), want)
	}
	if !strings.HasSuffix(err.Error(), "; "+nothingPushed) {
		t.Fatalf("the back-off = %q; want it to end saying nothing was pushed", err.Error())
	}
	if got := w.remoteTip(id); got != before {
		t.Fatalf("origin's tip on %s moved from %q to %q; want nothing pushed", id, before, got)
	}
}

func TestAReservationOfARecordFinishedOnTheDefaultBranchBacksOff(t *testing.T) {
	t.Run("no tip", func(t *testing.T) {
		w := newWorld(t)
		q := w.clone("99+quinn@users.noreply.github.com", "")
		w.finishOnMain("iss-5", "resolved")
		_, err := q.reserve("iss-5", "b.go")
		wantFinishedBackOff(t, w, "iss-5", err, "")
		w.finishOnMain("iss-6", "wontfix")
		_, err = q.do(Request{Record: "iss-6", Footprint: []string{"b.go"}, TakeOver: true})
		wantFinishedBackOff(t, w, "iss-6", err, "")
	})
	t.Run("live tip", func(t *testing.T) {
		w := newWorld(t)
		p := w.clone("1234+octo@users.noreply.github.com", "")
		q := w.clone("99+quinn@users.noreply.github.com", "")
		first, err := p.reserve("iss-5", "a.go")
		if err != nil {
			t.Fatal(err)
		}
		w.finishOnMain("iss-5", "resolved")
		_, err = q.reserve("iss-5", "b.go")
		wantFinishedBackOff(t, w, "iss-5", err, first.Commit)
		// The holder's own copy reserving again is no renewal either.
		_, err = p.reserve("iss-5", "a.go")
		wantFinishedBackOff(t, w, "iss-5", err, first.Commit)
		// The listing still counts the tip released by the default branch.
		snap, err := View(q.dir, w.clock())
		if err != nil || len(snap.Reservations) != 1 || snap.Reservations[0].ReleasedBy != ReleasedByDefaultBranch {
			t.Fatalf("the view = %+v, %v; want the tip released by the default branch", snap.Reservations, err)
		}
	})
	t.Run("tombstone", func(t *testing.T) {
		w := newWorld(t)
		p := w.clone("1234+octo@users.noreply.github.com", "")
		q := w.clone("99+quinn@users.noreply.github.com", "")
		if _, err := p.reserve("iss-5", "a.go"); err != nil {
			t.Fatal(err)
		}
		rel, err := p.do(Request{Record: "iss-5", Release: true})
		if err != nil || rel.Act != ActReleased {
			t.Fatalf("P's release = %+v, %v", rel, err)
		}
		w.finishOnMain("iss-5", "resolved")
		_, err = q.reserve("iss-5", "b.go")
		wantFinishedBackOff(t, w, "iss-5", err, rel.Commit)
	})
	t.Run("lost race", func(t *testing.T) {
		w := newWorld(t)
		p := w.clone("1234+octo@users.noreply.github.com", "")
		q := w.clone("99+quinn@users.noreply.github.com", "")
		var ptip string
		beforePush = func(root string) {
			if root != q.dir {
				return
			}
			beforePush = nil
			res, err := p.reserve("iss-5", "a.go")
			if err != nil {
				t.Errorf("P's reserve: %v", err)
			}
			ptip = res.Commit
			w.finishOnMain("iss-5", "resolved")
		}
		t.Cleanup(func() { beforePush = nil })
		_, err := q.reserve("iss-5", "b.go")
		wantFinishedBackOff(t, w, "iss-5", err, ptip)
	})
}
