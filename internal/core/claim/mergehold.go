package claim

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// The merge hold (itd-2609151838312703 C7, spc-2610150728396006 step 4): the
// one judgement every verb that lands work makes before it arms or queues a
// merge, the implement loop's land fence and `abcd land` alike. Under the
// concurrent-sessions rule a session cutting a release has the final say on
// what merges before its tag, so the cutting copy's own landing is its ruling,
// and every other session's, the same person's other copy included, waits
// for the cutting session to merge or hold it.
//
// The judgement is: fetch the reservations, which reads origin's release tags
// with them (FetchMergeHold); hold the merge while any release is being cut
// that is not this copy's own, an unreadable release within its default hold
// included (MergeHolds, Snapshot.Holds); and arm nothing when the fetch fails,
// since a hold it cannot see may be there. A checkout with no remote named
// origin has no reservations and is never held. Each caller words the
// refusal's last clause and its next step; the hold's own words are here.

// MergeHold is the releases that hold a merge from this copy: every release
// being cut, as last fetched, but this copy's own.
type MergeHold struct {
	Holds []Hold
}

// Held reports whether any release holds the merge.
func (m MergeHold) Held() bool { return len(m.Holds) > 0 }

// Reason names each release holding the merge, its holder and until when,
// and why it holds: "release v0.16.0 is being cut by maya, until <time>, and
// a session cutting a release has the final say on what merges before its
// tag". A caller adds what it does not do. "" when nothing holds.
func (m MergeHold) Reason() string {
	if !m.Held() {
		return ""
	}
	reasons := make([]string, 0, len(m.Holds))
	for _, h := range m.Holds {
		until := stamp(h.EffectiveExpiry)
		switch {
		case h.Unreadable != "":
			reasons = append(reasons, fmt.Sprintf("release %s is held by an unreadable reservation (%s), until %s",
				h.Release, h.Unreadable, until))
		case h.Mine:
			reasons = append(reasons, fmt.Sprintf("release %s is being cut by your copy on another computer, until %s", h.Release, until))
		default:
			reasons = append(reasons, fmt.Sprintf("release %s is being cut by %s, until %s", h.Release, h.Holder, until))
		}
	}
	return strings.Join(reasons, "; ") + ", and a session cutting a release has the final say on what merges before its tag"
}

// Waits is what ends each hold, for the caller's next step: ask the holder,
// or wait for the tag or the hold's end. "" when nothing holds.
func (m MergeHold) Waits() string {
	waits := make([]string, 0, len(m.Holds))
	for _, h := range m.Holds {
		switch {
		case h.Unreadable != "":
			waits = append(waits, "wait until "+h.Release+" is tagged or the hold ends at "+stamp(h.EffectiveExpiry))
		case h.Mine:
			waits = append(waits, "merge it from the copy cutting "+h.Release+", or wait until "+h.Release+" is tagged or that copy releases "+h.Subject)
		default:
			waits = append(waits, "ask "+h.Holder+" to merge it or to release "+h.Subject+", or wait until "+h.Release+" is tagged")
		}
	}
	return strings.Join(waits, "; ")
}

// MergeHolds is the merge hold as this checkout last fetched the
// reservations, from disk alone (ReleaseHolds without this copy's own): for a
// caller whose own act has just fetched them, as the land fence's renewal
// does.
func MergeHolds(root string, now time.Time, o Options) (MergeHold, error) {
	holds, _, err := ReleaseHolds(root, now, o)
	if err != nil {
		return MergeHold{}, err
	}
	var m MergeHold
	for _, h := range holds {
		if !h.Own {
			m.Holds = append(m.Holds, h)
		}
	}
	return m, nil
}

// FetchMergeHold fetches the reservations and origin's release tags, then
// judges the merge hold (MergeHolds). A checkout with no remote named origin
// is held by nothing and answers no error. A fetch that fails answers its
// *FetchError, and any other failure its error: never an empty hold, so the
// caller arms nothing.
func FetchMergeHold(root string, now time.Time, o Options) (MergeHold, error) {
	if _, err := Fetch(root, o); errors.Is(err, ErrNoOrigin) {
		return MergeHold{}, nil
	} else if err != nil {
		return MergeHold{}, err
	}
	return MergeHolds(root, now, o)
}
