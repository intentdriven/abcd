package claim

import (
	"testing"
	"time"
)

// A first read of a status branch past the walk's limit (refining
// dcn-2610161100416718): doubtful only while something beneath what it read
// could still stand.

// TestAFirstReadOfEndedHistoryPastTheLimitIsNotDoubtful: status branches only
// grow, so a new checkout's first read of a long-lived one runs past the
// limit. When the oldest commit it read was sent more than a day and an hour
// ago, every commit beneath it was sent earlier still and has ended, so the
// read is not doubtful.
func TestAFirstReadOfEndedHistoryPastTheLimitIsNotDoubtful(t *testing.T) {
	old := statusWalkLimit
	statusWalkLimit = 3
	t.Cleanup(func() { statusWalkLimit = old })
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	var msgs []string
	for i, item := range []string{"#1", "#2", "#3", "#4", "#5"} {
		msgs = append(msgs, announcementMsg("merged", item, "maya", t0.Add(-30*time.Hour+time.Duration(i)*time.Minute), 30*time.Minute))
	}
	w.pushMany("maya", msgs)
	pat.status(nil)
	if v := pat.read(); len(v.Doubts) != 0 {
		t.Fatalf("doubts = %+v; want none for history that has all ended", v.Doubts)
	}
}

// TestAFirstReadPastTheLimitWhoseOldestCannotBeReadIsDoubtful: when the oldest
// commit the first read took in does not read as an announcement, what lies
// beneath it cannot be dated, and the read is doubtful.
func TestAFirstReadPastTheLimitWhoseOldestCannotBeReadIsDoubtful(t *testing.T) {
	old := statusWalkLimit
	statusWalkLimit = 3
	t.Cleanup(func() { statusWalkLimit = old })
	w := newWorld(t)
	pat := w.clone("9+pat@users.noreply.github.com", "")
	gone := func(item string) string {
		return announcementMsg("merged", item, "maya", t0.Add(-30*time.Hour), 30*time.Minute)
	}
	w.pushMany("maya", []string{gone("#1"), "not an announcement\n", gone("#3"), gone("#4")})
	pat.status(nil)
	if v := pat.read(); len(v.Doubts) != 1 || v.Doubts[0].Reason != DoubtMore {
		t.Fatalf("doubts = %+v; want maya doubtful", v.Doubts)
	}
}
