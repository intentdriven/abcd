package claim

import (
	"testing"
	"time"
)

// A first read of a status branch past the walk's limit (dcn-2610161737571276,
// refining dcn-2610161320457650 and dcn-2610161100416718): within the bounded
// walk no signal settles whether a line older than what the read took in can
// still stand, so the read is always doubtful.

// TestAFirstReadPastTheLimitIsDoubtfulWhateverItsTimes: a first read of a
// branch longer than the walk's limit is doubtful whatever times its commits
// carry: all ended long ago, sent far in the past, or sent ahead of now.
func TestAFirstReadPastTheLimitIsDoubtfulWhateverItsTimes(t *testing.T) {
	old := statusWalkLimit
	statusWalkLimit = 3
	t.Cleanup(func() { statusWalkLimit = old })
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	branches := map[string]time.Time{
		"ended":  t0.Add(-30 * time.Hour),
		"ages":   t0.Add(-1000 * 24 * time.Hour),
		"recent": t0.Add(-time.Minute),
		"ahead":  t0.Add(48 * time.Hour),
	}
	for b, sent := range branches {
		var msgs []string
		for i, item := range []string{"#1", "#2", "#3", "#4", "#5"} {
			msgs = append(msgs, announcementMsg("merged", item, b, sent.Add(time.Duration(i)*time.Minute), 30*time.Minute))
		}
		w.pushMany(b, msgs)
	}
	pat.status(nil)
	v := pat.read()
	got := map[string]string{}
	for _, d := range v.Doubts {
		got[d.Branch] = d.Reason
	}
	for b := range branches {
		if got[b] != DoubtMore {
			t.Errorf("a first read of %s past the walk's limit: doubt %q; want %q", b, got[b], DoubtMore)
		}
	}
	if len(got) != len(branches) {
		t.Errorf("doubts = %+v; want exactly the %d branches read past the limit", v.Doubts, len(branches))
	}
}

// TestAFirstReadWithinTheLimitIsNotDoubtful: a first read that reaches the
// branch's root within the walk's limit reads all of it, and is not doubtful.
func TestAFirstReadWithinTheLimitIsNotDoubtful(t *testing.T) {
	old := statusWalkLimit
	statusWalkLimit = 3
	t.Cleanup(func() { statusWalkLimit = old })
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	w.pushMany("maya", []string{
		announcementMsg("merged", "#1", "maya", t0.Add(-30*time.Hour), 30*time.Minute),
		announcementMsg("merged", "#2", "maya", t0.Add(-30*time.Hour), 30*time.Minute),
		announcementMsg("merged", "#3", "maya", t0.Add(-30*time.Hour), 30*time.Minute),
	})
	pat.status(nil)
	if v := pat.read(); len(v.Doubts) != 0 {
		t.Fatalf("doubts = %+v; want none for a branch read to its root", v.Doubts)
	}
}
