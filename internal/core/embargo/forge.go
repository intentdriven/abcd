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

// UnreadAdvisory is a held advisory whose end on GitHub was not recorded,
// left as the store holds it, under embargo, and why: its state could not be
// told, or GitHub reports a close the store's verdict does not make
// (closedUnjudged).
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
// records the end GitHub reports for each: published_at, or closed_at for an
// advisory whose verdict ends its private track (endsPrivateTrack;
// closedUnjudged says why no other close is recorded), from GitHub's own timestamp: an end GitHub
// gives no timestamp for is left unread, never stamped with the clock's
// time. answers holds each advisory read in a state this abcd
// knows and recorded or left as it is, for the caller to reuse.
//
// GitHub's answer is untrusted input. The request names only an id the
// state holds, already held to the GHSA shape, and getAdvisory holds the
// answer to that id; the state must be one this abcd knows, and a timestamp
// must be RFC 3339 and not later than the clock allows. An answer failing
// any of that, or a read that fails, leaves the advisory as the store holds
// it and is listed in unread, with a reason that repeats nothing GitHub sent.
// Nothing GitHub sends names a path or reaches a command line.
//
// A read gh cannot make at all (gh is not installed) or that GitHub refuses
// for want of a scope or access is no fault of one advisory: every other
// read meets it too. It is returned whole, with its remedy (refusesWhole),
// and the caller refuses, recording nothing.
func readHeld(s Store, st *State, skip map[string]bool, now time.Time) (answers map[string]remoteAdvisory, ended []ForgeEnd, unread []UnreadAdvisory, err error) {
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
		a, rerr := getAdvisory(s.Checkout, st.Repo, h.GHSAID)
		if rerr != nil {
			if refusesWhole(rerr) {
				return nil, nil, nil, rerr
			}
			left("GitHub could not be read for it")
			continue
		}
		if !slices.Contains(ForgeStates, a.State) {
			left("GitHub reports it in a state this abcd does not know")
			continue
		}
		if a.State == "closed" && !endsPrivateTrack(h.Verdict) {
			unread = append(unread, UnreadAdvisory{GHSAID: h.GHSAID, IssueID: h.IssueID, Reason: closedUnjudged(h.Verdict)})
			continue
		}
		if a.State == "published" || a.State == "closed" {
			v := a.endedAt()
			if v == nil || strings.TrimSpace(*v) == "" {
				left("GitHub reports it " + a.State + " but gives no " + a.State + "_at")
				continue
			}
			at, ok := forgeTime(*v, now)
			if !ok {
				left("GitHub reports it " + a.State + " with a " + a.State + "_at that is not a time this abcd reads")
				continue
			}
			recordForgeEnd(h, a.State, at)
			ended = append(ended, ForgeEnd{GHSAID: h.GHSAID, IssueID: h.IssueID, State: a.State, At: at})
		}
		answers[h.GHSAID] = a
	}
	return answers, ended, unread, nil
}

// endsPrivateTrack reports whether the store's verdict (nil when
// untriaged) already ends the advisory's private track, so a close GitHub
// reports for it is consistent with the store's own decision and is
// recorded: a closing verdict (not-abcds, not-a-vulnerability), or the
// public track, whose triage moves the record to wontfix/ and whose title
// the check already treats as public. It is not close's rule: close sends a
// closing text only for a closing verdict (isClosingVerdict).
func endsPrivateTrack(verdict *string) bool {
	return verdict != nil && (isClosingVerdict(*verdict) || *verdict == VerdictPublicTrack)
}

// closedUnjudged is the reason a held advisory GitHub reports closed is
// listed and its close not recorded: the store's verdict for it (verdict,
// nil when untriaged) does not end its private track (endsPrivateTrack).
// Recorded, the close would drop an advisory the store holds as a
// vulnerability, or never judged, from every plan for good, on the word of
// whoever closed it. The advisory stays as the store holds it, under
// embargo and read again by every intake and plan, so a reopen on GitHub is
// seen; a person settles it.
func closedUnjudged(verdict *string) string {
	holds, settle := "untriaged", "record its verdict with `abcd embargo triage`; a closing or public-track verdict records the close"
	if verdict != nil && *verdict == VerdictVulnerability {
		holds, settle = "as a vulnerability", "reopen it on GitHub to publish it, or record a closing verdict with `abcd embargo triage`"
	}
	return "it is closed on GitHub while the store holds it " + holds + ", so its close is not recorded and it stays under embargo; " +
		"a person settles it: " + settle
}

// forgeTime is the time a timestamp GitHub reports names; ok is false for
// one that is not RFC 3339, or lies past the clock by more than forgeSkew.
func forgeTime(v string, now time.Time) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339, v)
	if err != nil || t.After(now.Add(forgeSkew)) {
		return time.Time{}, false
	}
	return t.UTC(), true
}
