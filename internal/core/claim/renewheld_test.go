package claim

import (
	"testing"
	"time"
)

// TestRenewHeldRenewsTheRunsReservationsFromEveryWorktree
// (iss-2611220942554289): the heartbeat's renewal lists the person's own
// reservations held from any worktree of this checkout, renews each half
// through its lease with its own term through the renewal from any worktree,
// and leaves alone one held from another clone, another person's, and a
// release subject. Before half through it renews nothing and lists them all.
func TestRenewHeldRenewsTheRunsReservationsFromEveryWorktree(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	sib := p.worktree("lane-b")
	far := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	for _, c := range []struct {
		who *participant
		id  string
	}{{p, "iss-1"}, {sib, "iss-2"}, {far, "iss-3"}, {q, "iss-4"}} {
		if _, err := c.who.reserve(c.id, c.id+".go"); err != nil {
			t.Fatal(err)
		}
	}

	w.set(t0.Add(30 * time.Minute))
	held, err := RenewHeld(sib.dir, sib.opts())
	if err != nil {
		t.Fatal(err)
	}
	if len(held) != 2 || held[0].Record != "iss-1" || held[1].Record != "iss-2" {
		t.Fatalf("held = %+v; want iss-1 and iss-2, this checkout's", held)
	}
	for _, h := range held {
		if h.Renewed || h.Err != nil || !h.Until.Equal(t0.Add(2*time.Hour)) {
			t.Fatalf("before half through nothing is renewed: %+v", h)
		}
	}
	if held[0].HeldFrom == "" || held[1].HeldFrom != "" {
		t.Fatalf("each names the worktree holding it, this one by none: %+v", held)
	}

	w.set(t0.Add(time.Hour))
	tips := map[string]string{}
	for _, id := range []string{"iss-1", "iss-2", "iss-3", "iss-4"} {
		tips[id] = w.remoteTip(id)
	}
	held, err = RenewHeld(sib.dir, sib.opts())
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range held {
		if !h.Renewed || h.Err != nil || !h.Until.Equal(t0.Add(3*time.Hour)) {
			t.Fatalf("half through, each is renewed with its own term: %+v", h)
		}
	}
	for id, tip := range tips {
		moved := w.remoteTip(id) != tip
		if want := id == "iss-1" || id == "iss-2"; moved != want {
			t.Fatalf("%s moved = %v; want %v", id, moved, want)
		}
	}
	if pushedHere(p.dir, "iss-1") != w.remoteTip("iss-1") {
		t.Fatal("the holding worktree's note does not follow the renewal")
	}
}
