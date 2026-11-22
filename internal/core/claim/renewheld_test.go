package claim

import (
	"strings"
	"testing"
	"time"
)

// liveOn is a LiveWork that finds live work behind the branches named alone,
// and records each branch it was asked about.
func liveOn(asked *[]string, branches ...string) LiveWork {
	return func(branch string) (bool, string) {
		*asked = append(*asked, branch)
		for _, b := range branches {
			if b == branch {
				return true, ""
			}
		}
		return false, "no pull request from its branch " + branch
	}
}

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
	sib.git("checkout", "-q", "-b", "lane-b")
	def := p.git("symbolic-ref", "--short", "HEAD")
	far := w.clone("1234+octo@users.noreply.github.com", "")
	q := w.clone("99+quinn@users.noreply.github.com", "")
	var asked []string
	live := liveOn(&asked, def, "lane-b")
	for _, c := range []struct {
		who *participant
		id  string
	}{{p, "iss-1"}, {sib, "iss-2"}, {far, "iss-3"}, {q, "iss-4"}} {
		if _, err := c.who.reserve(c.id, c.id+".go"); err != nil {
			t.Fatal(err)
		}
	}

	w.set(t0.Add(30 * time.Minute))
	held, err := RenewHeld(sib.dir, live, sib.opts())
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
	held, err = RenewHeld(sib.dir, live, sib.opts())
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

// TestRenewHeldRenewsOnlyAReservationLiveWorkStandsBehind (the review of
// iss-2611220942554289): the heartbeat renews a reservation only when the
// branch of the worktree holding it has a pull request open, armed or queued.
// One whose worktree still exists with no pull request (abandoned, or a lane
// mid-work with commits not yet pushed) and one on no branch are listed held
// and not renewed, each with why, so each lapses on its lease; the lane's own
// loop renews its reservations elsewhere. With no LiveWork nothing is renewed.
func TestRenewHeldRenewsOnlyAReservationLiveWorkStandsBehind(t *testing.T) {
	w := newWorld(t)
	p := w.clone("1234+octo@users.noreply.github.com", "")
	queued := p.worktree("queued")
	queued.git("checkout", "-q", "-b", "fix-queued")
	abandoned := p.worktree("abandoned")
	abandoned.git("checkout", "-q", "-b", "fix-abandoned")
	midwork := p.worktree("midwork")
	midwork.git("checkout", "-q", "-b", "lane-midwork")
	detached := p.worktree("detached")
	for _, c := range []struct {
		who *participant
		id  string
	}{{queued, "iss-1"}, {abandoned, "iss-2"}, {midwork, "iss-3"}, {detached, "iss-4"}} {
		if _, err := c.who.reserve(c.id, c.id+".go"); err != nil {
			t.Fatal(err)
		}
	}
	midwork.git("commit", "-q", "--allow-empty", "-m", "work not yet pushed")

	w.set(t0.Add(time.Hour))
	var asked []string
	held, err := RenewHeld(p.dir, liveOn(&asked, "fix-queued"), p.opts())
	if err != nil {
		t.Fatal(err)
	}
	if len(held) != 4 {
		t.Fatalf("held = %+v; want all four listed", held)
	}
	for _, h := range held {
		backed := h.Record == "iss-1"
		if h.Renewed != backed || h.Err != nil || (h.Unbacked == "") != backed {
			t.Fatalf("%s: renewed %v, unbacked %q, err %v; want renewed only behind the queued pull request", h.Record, h.Renewed, h.Unbacked, h.Err)
		}
	}
	if !strings.Contains(held[1].Unbacked, "fix-abandoned") || !strings.Contains(held[2].Unbacked, "lane-midwork") ||
		!strings.Contains(held[3].Unbacked, "no branch") {
		t.Fatalf("each unrenewed one says why: %+v", held)
	}
	if !held[0].Until.Equal(t0.Add(3*time.Hour)) || !held[1].Until.Equal(t0.Add(2*time.Hour)) {
		t.Fatalf("the backed one's lease moves on, the others' stay: %+v", held)
	}

	held, err = RenewHeld(p.dir, nil, p.opts())
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range held {
		if h.Renewed || h.Unbacked == "" {
			t.Fatalf("with no LiveWork nothing is renewed: %+v", h)
		}
	}
}
