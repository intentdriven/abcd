package claim

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestRenewDueRenewsOnlyThisCopysOwnLiveHoldsPastHalf is the waiter's renewal
// (iss-2610170926295770): a reservation made by hand is renewed while its
// holder waits at the load gate once half its lease has passed, and nothing
// else is: not one under half, not another person's, not this person's from
// another copy. Nothing new is ever reserved.
func TestRenewDueRenewsOnlyThisCopysOwnLiveHoldsPastHalf(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	p2 := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	for _, r := range []struct {
		who *participant
		id  string
		fp  string
	}{{p, "iss-5", "a.go"}, {p, "iss-6", "b.go"}, {p2, "iss-7", "c.go"}, {q, "iss-8", "d.go"}} {
		if _, err := r.who.reserve(r.id, r.fp); err != nil {
			t.Fatalf("%s reserves: %v", r.id, err)
		}
	}
	if _, err := Fetch(p.dir, p.opts()); err != nil {
		t.Fatal(err)
	}
	tips := map[string]string{}
	for _, id := range []string{"iss-5", "iss-6", "iss-7", "iss-8"} {
		tips[id] = w.remoteTip(id)
	}

	w.set(t0.Add(59 * time.Minute))
	got, err := RenewDue(p.dir, p.opts())
	if err != nil || len(got) != 0 {
		t.Fatalf("under half its lease: renewals %+v, err %v; want none", got, err)
	}
	for id, tip := range tips {
		if w.remoteTip(id) != tip {
			t.Fatalf("a renewal under half the lease pushed %s", id)
		}
	}

	at := t0.Add(61 * time.Minute)
	w.set(at)
	got, err = RenewDue(p.dir, p.opts())
	if err != nil {
		t.Fatal(err)
	}
	renewed := map[string]Renewal{}
	for _, r := range got {
		renewed[r.Record] = r
	}
	if len(got) != 2 {
		t.Fatalf("renewals %+v; want iss-5 and iss-6 alone", got)
	}
	for _, id := range []string{"iss-5", "iss-6"} {
		r, ok := renewed[id]
		if !ok || r.Err != nil || !r.Until.Equal(at.Add(2*time.Hour)) {
			t.Errorf("%s: %+v; want renewed until %s", id, r, at.Add(2*time.Hour))
		}
		if w.remoteTip(id) == tips[id] {
			t.Errorf("%s: the renewal pushed nothing", id)
		}
	}
	for _, id := range []string{"iss-7", "iss-8"} {
		if w.remoteTip(id) != tips[id] {
			t.Errorf("%s, not this copy's own, was moved", id)
		}
	}

	// Renewed, each is under half its new lease, so the next look pushes
	// nothing.
	w.set(at.Add(time.Minute))
	if got, err := RenewDue(p.dir, p.opts()); err != nil || len(got) != 0 {
		t.Fatalf("just after a renewal: renewals %+v, err %v; want none", got, err)
	}
}

// A copy holding no reservation renews nothing and pushes nothing.
func TestRenewDueWithNothingHeldDoesNothing(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	w.set(t0.Add(3 * time.Hour))
	got, err := RenewDue(p.dir, p.opts())
	if err != nil || len(got) != 0 {
		t.Fatalf("renewals %+v, err %v; want none", got, err)
	}
}

// TestRenewDueRenewsOnlyWhatThisCopyPushed: the renewal renews the
// reservations this copy itself pushed, and leaves a tip it did not push as
// it is, whatever the tip reads as.
func TestRenewDueRenewsOnlyWhatThisCopyPushed(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	own := w.remoteMessage(w.remoteTip("iss-5"))
	other := w.pushHostile("iss-9", strings.ReplaceAll(own, "iss-5", "iss-9"), t0)
	if _, err := Fetch(p.dir, p.opts()); err != nil {
		t.Fatal(err)
	}
	snap, err := View(p.dir, w.clock())
	if err != nil {
		t.Fatal(err)
	}
	read := false
	for _, r := range snap.Reservations {
		read = read || r.Record == "iss-9" && r.Copy == CopyThis && r.State == StateLive
	}
	if !read {
		t.Fatalf("iss-9 does not read as this copy's live hold: %+v", snap.Reservations)
	}

	w.set(t0.Add(61 * time.Minute))
	got, err := RenewDue(p.dir, p.opts())
	if err != nil {
		t.Fatal(err)
	}
	byRecord := map[string]Renewal{}
	for _, r := range got {
		byRecord[r.Record] = r
	}
	if len(got) != 2 || byRecord["iss-5"].Err != nil || byRecord["iss-5"].Until.IsZero() ||
		!errors.Is(byRecord["iss-9"].Err, errNotPushedHere) {
		t.Fatalf("renewals %+v; want iss-5 renewed and iss-9 answered as not pushed here", got)
	}
	if w.remoteTip("iss-9") != other {
		t.Fatalf("iss-9, whose tip this copy did not push, was renewed from it")
	}

	// The renewal this copy pushed is its own in turn, so it is renewed again
	// once half its new lease has passed.
	w.set(t0.Add(61*time.Minute + 61*time.Minute))
	got, err = RenewDue(p.dir, p.opts())
	if err != nil {
		t.Fatal(err)
	}
	renewedAgain := false
	for _, r := range got {
		renewedAgain = renewedAgain || r.Record == "iss-5" && r.Err == nil
	}
	if !renewedAgain || w.remoteTip("iss-9") != other {
		t.Fatalf("second renewals %+v; want iss-5 renewed again and iss-9 left as it is", got)
	}
}

// TestTwoWaitersInOneCopyTakeTurnsToRenew is iss-2610270659564789: two
// waiters in one copy renewed the same reservations at once, each judging from
// the tips it had read, and the one whose push lost reported the loss as a
// fault. Each pass now holds the copy's renewal lock, so the second waits for
// the first to finish, reads the tip it pushed, and finds nothing due.
func TestTwoWaitersInOneCopyTakeTurnsToRenew(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := p.reserve("iss-5", "a.go"); err != nil {
		t.Fatal(err)
	}
	if _, err := Fetch(p.dir, p.opts()); err != nil {
		t.Fatal(err)
	}
	w.set(t0.Add(61 * time.Minute))

	// At its push, the first pass starts a sibling pass in the same copy and
	// gives it time to reach its own push before going on, the window two
	// unlocked waiters raced in.
	type pass struct {
		got []Renewal
		err error
	}
	var pushes atomic.Int32
	reached := make(chan struct{})
	sibling := make(chan pass, 1)
	beforePush = func(string) {
		switch pushes.Add(1) {
		case 1:
			go func() {
				got, err := RenewDue(p.dir, p.opts())
				sibling <- pass{got, err}
			}()
			select {
			case <-reached:
			case <-time.After(3 * time.Second):
			}
		case 2:
			close(reached)
		}
	}
	t.Cleanup(func() { beforePush = nil })

	first, err := RenewDue(p.dir, p.opts())
	second := <-sibling
	if err != nil || second.err != nil {
		t.Fatalf("passes failed: first %v, second %v", err, second.err)
	}
	if len(first) != 1 || first[0].Record != "iss-5" || first[0].Err != nil {
		t.Errorf("the first pass: %+v; want iss-5 renewed", first)
	}
	for _, r := range second.got {
		if r.Err != nil {
			t.Errorf("the second pass reported %s as a fault: %v", r.Record, r.Err)
		}
	}
	if len(second.got) != 0 {
		t.Errorf("the second pass renewed %+v; the first had just renewed it, so nothing was due", second.got)
	}
}
