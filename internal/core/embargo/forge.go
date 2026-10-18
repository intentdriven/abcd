package embargo

import (
	"slices"
	"strings"
	"time"
)

// forge.go is what the store learns from GitHub about the advisories it
// holds (iss-2610171250479046): an advisory published or closed on GitHub by
// any hand, not only by `abcd embargo publish` or `abcd embargo close`, is
// recorded so, which releases a published one from the outbound check.

// ForgeEnd is a held advisory GitHub reports published or closed, as the
// store now records it: State is "published" or "closed", At its
// published_at or closed_at.
type ForgeEnd struct {
	GHSAID  string    `json:"ghsa_id"`
	IssueID string    `json:"issue_id"`
	State   string    `json:"state"`
	At      time.Time `json:"at"`
}

// UnreadAdvisory is a held advisory whose state on GitHub could not be
// told, left as the store holds it, under embargo, and why.
type UnreadAdvisory struct {
	GHSAID  string `json:"ghsa_id"`
	IssueID string `json:"issue_id"`
	Reason  string `json:"reason"`
}

// forgeSkew is how far past the clock a timestamp GitHub reports may lie and
// still be read: two clocks disagree by a little, never by days.
const forgeSkew = 5 * time.Minute

// readHeld reads, one advisory at a time, the state GitHub reports for each
// advisory the state holds that is neither published nor closed there, but
// those skip names (intake has just listed them in triage or draft), and
// records the end GitHub reports for each: published_at or closed_at, from
// GitHub's own timestamp, or from now when GitHub gives none. answers holds
// each advisory read in a state this abcd knows, for the caller to reuse.
//
// GitHub's answer is untrusted input. The request names only an id the
// state holds, already held to the GHSA shape, and getAdvisory holds the
// answer to that id; the state must be one this abcd knows, and a timestamp
// must be RFC 3339 and not later than the clock allows. An answer failing
// any of that, or a read that fails, leaves the advisory as the store holds
// it and is listed in unread, with a reason that repeats nothing GitHub sent.
// Nothing GitHub sends names a path or reaches a command line.
func readHeld(s Store, st *State, skip map[string]bool, now time.Time) (answers map[string]remoteAdvisory, ended []ForgeEnd, unread []UnreadAdvisory) {
	answers = map[string]remoteAdvisory{}
	ended, unread = []ForgeEnd{}, []UnreadAdvisory{}
	for i := range st.Advisories {
		h := &st.Advisories[i]
		if h.PublishedAt != nil || h.ClosedAt != nil || skip[h.GHSAID] {
			continue
		}
		left := func(why string) {
			unread = append(unread, UnreadAdvisory{GHSAID: h.GHSAID, IssueID: h.IssueID, Reason: why + "; its state in the store is unchanged"})
		}
		a, err := getAdvisory(s.Checkout, st.Repo, h.GHSAID)
		if err != nil {
			left("GitHub could not be read for it")
			continue
		}
		if !slices.Contains(ForgeStates, a.State) {
			left("GitHub reports it in a state this abcd does not know")
			continue
		}
		if a.State == "published" || a.State == "closed" {
			at, ok := forgeTime(a.endedAt(), now)
			if !ok {
				left("GitHub reports it " + a.State + " with a " + a.State + "_at that is not a time this abcd reads")
				continue
			}
			recordForgeEnd(h, a.State, at)
			ended = append(ended, ForgeEnd{GHSAID: h.GHSAID, IssueID: h.IssueID, State: a.State, At: at})
		}
		answers[h.GHSAID] = a
	}
	return answers, ended, unread
}

// forgeTime is the time a timestamp GitHub reports names, or now when it
// reports none; ok is false for one that is not RFC 3339, or lies past the
// clock by more than forgeSkew.
func forgeTime(v *string, now time.Time) (time.Time, bool) {
	if v == nil || strings.TrimSpace(*v) == "" {
		return now, true
	}
	t, err := time.Parse(time.RFC3339, *v)
	if err != nil || t.After(now.Add(forgeSkew)) {
		return time.Time{}, false
	}
	return t.UTC(), true
}
