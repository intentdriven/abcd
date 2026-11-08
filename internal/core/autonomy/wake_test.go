package autonomy

import (
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/lease"
)

// TestDeadlineNeverPast (iss-2611011332417169): once now is within lease.Skew
// of the owed set's WakeBy, WakeBy less the skew lies before now; the deadline
// still comes out after now, no earlier than now plus MinWakeLead, so a wake
// set now for the next minute or two can release the stop, rather than a
// deadline already past naming a time no wake can meet.
func TestDeadlineNeverPast(t *testing.T) {
	loc := london(t)
	now := at(t, "2026-10-31 18:28")
	for _, left := range []time.Duration{0, time.Minute, lease.Skew - time.Second} {
		owed := Owed{Items: []OwedItem{renewalDue}, WakeBy: now.Add(left), NextRenewal: now.Add(-time.Hour)}
		d := wakeDeadline(owed, now)
		if !d.After(now) || d.Before(now.Add(MinWakeLead)) {
			t.Errorf("WakeBy %s after now: deadline %s, want no earlier than %s", left, d, now.Add(MinWakeLead))
		}
		next := []SessionCron{{ID: "once", Schedule: "29 18 31 10 *", Recurring: false}}
		if _, _, ok := qualifyingWake(next, now, d, loc); !ok {
			t.Errorf("WakeBy %s after now: a one-shot wake for the next minute does not qualify against %s", left, d)
		}
	}
	// A WakeBy far enough off keeps its own deadline, less the skew.
	far := Owed{Items: []OwedItem{renewalDue}, WakeBy: now.Add(20 * time.Minute)}
	if got, want := wakeDeadline(far, now), now.Add(15*time.Minute); !got.Equal(want) {
		t.Errorf("WakeBy 20 minutes off: deadline %s, want %s", got, want)
	}
}
