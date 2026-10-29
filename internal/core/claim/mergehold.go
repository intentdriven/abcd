package claim

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/lease"
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
// with them (FetchMergeHold); keep this copy's first sighting of each release
// reservation dated more than lease.Skew ahead, whichever fetch brought it, so
// it holds a day (an unreadable one its default hold) from that sighting and
// no longer; hold the merge while any release is being cut that is not this
// copy's own, an unreadable release within its default hold included
// (MergeHolds, Snapshot.Holds); and arm nothing when the fetch fails, since a
// hold it cannot see may be there. A checkout with no remote named origin has
// no reservations and is never held. Each caller words the refusal's last
// clause and its next step; the hold's own words are here.
//
// A release dated more than lease.Skew ahead whose first sighting cannot be
// kept, in a checkout with no local tier or one whose tier cannot be written,
// would be counted from every reading and hold for ever, so it is not counted
// as a hold: it is returned in Ahead, and the caller names it (Notice). The
// local tier is never made for it. An honest cut is reserved on a clock within
// lease.Skew of the reader's, so this lets no merge past one, with one limit:
// a cutting copy whose clock runs more than lease.Skew fast is not counted by
// a checkout that cannot keep a sighting.

// MergeHold is the releases that hold a merge from this copy: every release
// being cut, as last fetched, but this copy's own.
type MergeHold struct {
	Holds []Hold
	// Ahead is every release being cut, but this copy's own, that is dated
	// more than lease.Skew ahead (or not dated) with no first sighting kept
	// here: dated ahead and not counted as a hold.
	Ahead []Hold
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
			reasons = append(reasons, fmt.Sprintf("release %s is being cut by %s, until %s", h.Release, h.yourCopy(), until))
		default:
			reasons = append(reasons, fmt.Sprintf("release %s is being cut by %s, until %s", h.Release, h.Holder, until))
		}
	}
	return strings.Join(reasons, "; ") + ", and a session cutting a release has the final say on what merges before its tag"
}

// yourCopy names the copy of the reader's own that holds a release
// (yourCopyAt).
func (h Hold) yourCopy() string { return yourCopyAt(h.Sibling) }

// Waits is what ends each hold, for the caller's next step: the copy cutting
// the release merges it, or that copy's holder releases the hold or stops
// renewing it, or the tag reaches origin. Another person's hold is named by
// its subject and its holder id, which names no one a reader could ask, so
// the words say what ends it rather than whom to ask. "" when nothing holds.
func (m MergeHold) Waits() string {
	waits := make([]string, 0, len(m.Holds))
	for _, h := range m.Holds {
		switch {
		case h.Unreadable != "":
			waits = append(waits, "wait until "+h.Release+" is tagged or the hold ends at "+stamp(h.EffectiveExpiry))
		case h.Mine && h.Sibling != "":
			waits = append(waits, "merge it from "+h.Sibling+", the worktree cutting "+h.Release+", or wait until "+h.Release+" is tagged or that copy releases "+h.Subject)
		case h.Mine:
			waits = append(waits, "merge it from the copy cutting "+h.Release+", or wait until "+h.Release+" is tagged or that copy releases "+h.Subject)
		default:
			waits = append(waits, "wait until "+h.Release+" is tagged, or until "+h.Subject+", held by "+h.Holder+" from the copy cutting it, ends: "+
				"its holder releases it, or stops renewing it and it ends at "+stamp(h.EffectiveExpiry)+"; the copy cutting it can also merge this itself")
		}
	}
	return strings.Join(waits, "; ")
}

// Notice is the one line that names each release dated ahead and not
// counted (Ahead), for the caller to show beside a merge it lets through: "release
// v0.16.0 is reserved by quinn with a date more than 5 minutes ahead of this
// clock, and this checkout cannot keep when it first saw it (...), so it is
// not counted as holding this merge". "" when there is none.
func (m MergeHold) Notice() string {
	if len(m.Ahead) == 0 {
		return ""
	}
	lines := make([]string, 0, len(m.Ahead))
	for _, h := range m.Ahead {
		lines = append(lines, h.notCounted()+" this merge")
	}
	return strings.Join(lines, "; ")
}

// notCounted names a release dated ahead whose first sighting this copy
// cannot keep, up to what it is not counted as holding: "release v0.16.0 is
// reserved by quinn with a date more than 5 minutes ahead of this clock, and
// this checkout cannot keep when it first saw it (...), so it is not counted
// as holding". The caller names what.
func (h Hold) notCounted() string {
	ahead := fmt.Sprintf("dated more than %d minutes ahead of this clock", lease.Skew/time.Minute)
	var what string
	switch {
	case h.Unreadable != "" && h.Undated:
		what = fmt.Sprintf("release %s is held by an unreadable reservation (%s) whose date cannot be read", h.Release, h.Unreadable)
	case h.Unreadable != "":
		what = fmt.Sprintf("release %s is held by an unreadable reservation (%s) %s", h.Release, h.Unreadable, ahead)
	case h.Mine:
		what = fmt.Sprintf("release %s is reserved by %s with a date more than %d minutes ahead of this clock",
			h.Release, h.yourCopy(), lease.Skew/time.Minute)
	default:
		what = fmt.Sprintf("release %s is reserved by %s with a date more than %d minutes ahead of this clock",
			h.Release, h.Holder, lease.Skew/time.Minute)
	}
	return what + ", and this checkout cannot keep when it first saw it (" + tierRelDir +
		"/ is missing or cannot be written), so it is not counted as holding"
}

// MergeHolds is the merge hold as this checkout last fetched the
// reservations (JudgedHolds without this copy's own): for a caller whose own
// act has just fetched them, as the land fence's renewal does. It fetches
// nothing and pushes nothing. A release that needs a first sighting and has
// none kept is returned in Ahead, not Holds.
func MergeHolds(root string, now time.Time, o Options) (MergeHold, error) {
	holds, err := JudgedHolds(root, now, o)
	if err != nil {
		return MergeHold{}, err
	}
	var m MergeHold
	for _, h := range holds {
		switch {
		case h.Own:
		case h.Unsighted:
			m.Ahead = append(m.Ahead, h)
		default:
			m.Holds = append(m.Holds, h)
		}
	}
	return m, nil
}

// JudgedHolds is every release being cut as this checkout last fetched the
// reservations (releaseHolds), judged by the merge hold's rule: it first
// keeps, in the local tier, this copy's first sighting of each release
// reservation that needs one, so every path that judges a merge counts a
// release dated ahead from the same sighting, and each hold's Own and
// Unsighted then say whether MergeHolds counts it. MergeHolds counts from
// it; `abcd peers` shows it, each hold by its Line. It fetches nothing and
// pushes nothing.
func JudgedHolds(root string, now time.Time, o Options) ([]Hold, error) {
	now = now.UTC().Truncate(time.Second)
	keepFirstFetched(root, now, isRelease)
	return releaseHolds(root, now, o)
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
