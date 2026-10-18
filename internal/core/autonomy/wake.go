package autonomy

import (
	"time"

	"github.com/intentdriven/abcd/internal/core/lease"
)

const (
	// OwedCeiling is the furthest a qualifying wake may fire while work is
	// owed: a session with owed work waits half an hour at most.
	OwedCeiling = 30 * time.Minute
	// IdleCeiling is the furthest a qualifying wake may fire while nothing
	// is owed.
	IdleCeiling = time.Hour
	// MaxLateness is the most the host runs a recurring task late: half the
	// gap to its following run, at most this.
	MaxLateness = 30 * time.Minute
)

// SessionCron is one of the host's scheduled tasks for the session, as the
// Stop payload's session_crons carries it.
type SessionCron struct {
	ID        string `json:"id"`
	Schedule  string `json:"schedule"`
	Recurring bool   `json:"recurring"`
	Prompt    string `json:"prompt"`
}

// BackgroundTask is one in-flight task the Stop payload's background_tasks
// carries. A background task is never a wake on its own while away: a
// `sleep 36000` or a `tail -f` would otherwise hold a session idle.
type BackgroundTask struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Description string `json:"description,omitempty"`
	Command     string `json:"command,omitempty"`
}

// Deadline is the latest a wake may fire for the stop to wait on it: the
// earlier of the earliest renewal due among the live reservations under this
// copy's key less lease.Skew, and now plus a ceiling, OwedCeiling while work
// is owed and IdleCeiling when none is (dcn-2610181003455731).
func Deadline(owed Owed, now time.Time) time.Time {
	ceiling := IdleCeiling
	if !owed.Empty() {
		ceiling = OwedCeiling
	}
	d := now.Add(ceiling)
	if !owed.NextRenewal.IsZero() {
		if r := owed.NextRenewal.Add(-lease.Skew); r.Before(d) {
			d = r
		}
	}
	return d
}

// LatestFire is a task's next match after now, no later than deadline, and
// the latest the host may run it: for a recurring task the match plus half
// the gap to the following match, at most MaxLateness, and nothing for
// exactly `*/5 * * * *`; a one-shot task is taken as matched, since the host
// only ever runs one early. ok is false for an expression abcd cannot parse
// or one with no match by the deadline.
func LatestFire(c SessionCron, now, deadline time.Time, loc *time.Location) (match, latest time.Time, ok bool) {
	cr, err := ParseCron(c.Schedule)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	match, ok = cr.Next(now, deadline, loc)
	if !ok {
		return time.Time{}, time.Time{}, false
	}
	if !c.Recurring || cr.everyFive {
		return match, match, true
	}
	late := MaxLateness
	if following, found := cr.Next(match, deadline.Add(time.Hour), loc); found {
		if half := following.Sub(match) / 2; half < late {
			late = half
		}
	}
	return match, match.Add(late), true
}

// QualifyingWake is the task whose latest fire time is no later than
// deadline, the earliest-matching of them, and its next match, the session's
// next look. ok is false when none qualifies.
func QualifyingWake(crons []SessionCron, now, deadline time.Time, loc *time.Location) (SessionCron, time.Time, bool) {
	var best SessionCron
	var look time.Time
	found := false
	for _, c := range crons {
		match, latest, ok := LatestFire(c, now, deadline, loc)
		if !ok || latest.After(deadline) {
			continue
		}
		if !found || match.Before(look) {
			best, look, found = c, match, true
		}
	}
	return best, look, found
}
