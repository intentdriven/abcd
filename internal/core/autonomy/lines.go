package autonomy

import (
	"fmt"
	"strings"
)

// PersonLine is the one line the prompt hook shows the person on every prompt
// while the record holds: until when, set by whom and from which working
// copy, the counts of queued steps and handed-back items, and the literal
// `abcd mode back` as the way to end it (dcn-2610181035355741).
func (r *Record) PersonLine() string {
	until := showTime(r.Ends())
	if r.OpenEnded {
		until += " (open-ended)"
	}
	hold := ""
	if r.HoldPushes {
		hold = "; pushes wait for you"
	}
	return fmt.Sprintf("abcd: away until %s, set by %s from %s%s; %s queued, %s handed back. `abcd mode back` ends it.",
		until, showWho(r.SetBy), showCopy(r.SetFrom), hold,
		count(len(r.Queue), "step", "steps"), count(len(r.HandedBack), "item", "items"))
}

// LapsedLine is the one line the prompt hook shows the person on every prompt
// while the record has lapsed unclosed holding a queued step or a recorded
// stop, and "" otherwise (dcn-2610181003474765).
func (r *Record) LapsedLine() string {
	if len(r.Queue) == 0 && len(r.Stops) == 0 {
		return ""
	}
	var parts []string
	if n := len(r.Queue); n > 0 {
		parts = append(parts, count(n, "step", "steps")+" queued")
	}
	if n := len(r.Stops); n > 0 {
		parts = append(parts, count(n, "stop", "stops")+" recorded")
	}
	return fmt.Sprintf("abcd: the away window ended at %s with %s, not yet read. `abcd mode back` prints them and closes it.",
		showTime(r.Ends()), strings.Join(parts, " and "))
}

func count(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}
