package claim

import (
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/lease"
)

// A tip dated more than lease.Skew ahead of the reader's clock is counted
// from when this copy first fetched it (lease.Judge's FirstSeen), so a tip
// whose writer's clock runs far ahead lapses a lease after it was first seen
// here instead of holding a lease from every reading. The first time is kept
// where an undated tip's is (firstFetched, undatedRelDir), one file per
// commit, so a later fetch does not restart it, and once kept it is the
// tip's FirstSeen at every reading.

// readSeen is readTip, with the tip's first sighting set (seen).
func readSeen(root, record, ref, tree string, fetchedAt time.Time, keep bool) (tip, bool, error) {
	t, ok, err := readTip(root, record, ref, tree, fetchedAt, keep)
	if err != nil || !ok {
		return t, ok, err
	}
	return seen(root, t, fetchedAt, keep), true, nil
}

// seen gives a tip the time this copy first fetched it whenever one is kept,
// whatever the tip's date is now, so lease.Judge counts it the same way at
// every reading. With keep, a tip whose own date (its claimed-at, or its
// commit date when it is unreadable) is more than lease.Skew ahead of
// fetchedAt and has no time kept is kept as first fetched at fetchedAt.
// Without keep (the view) nothing is kept, and a tip with no time kept has no
// FirstSeen: lease.Judge then counts one dated ahead from the reader's now, as
// the act that would keep it does, never from an earlier fetch that may not
// have brought it.
func seen(root string, t tip, fetchedAt time.Time, keep bool) tip {
	if at, ok := keptFetch(root, t.Commit, fetchedAt, keep); ok {
		t.FirstSeen = at
		return t
	}
	date := t.ClaimedAt
	if t.Unreadable != "" {
		date = t.WrittenAt
	}
	if keep && lease.Ahead(date, fetchedAt) {
		t.FirstSeen = firstFetched(root, t.Commit, fetchedAt, true)
	}
	return t
}

// keepFirstFetched reads every reservation tip as last fetched whose subject
// only admits (every tip when only is nil) with keep, so the time this copy
// first fetched each tip that needs one, dated ahead or undated, is recorded.
// It is best effort: a tip it cannot read is left to the reader that reports
// it, and a time it cannot keep (no local tier, or one it cannot write) is
// left unrecorded, for the reader to see that none is kept.
func keepFirstFetched(root string, now time.Time, only func(subject string) bool) {
	out, err := localGit(root, 1<<20, "for-each-ref", "--format=%(refname)", TrackingPrefix)
	if err != nil {
		return
	}
	tree, err := emptyTree(root)
	if err != nil {
		return
	}
	for _, ref := range strings.Fields(out) {
		id := strings.TrimPrefix(ref, TrackingPrefix)
		if canon, err := reservableID(id); err == nil && canon == id && (only == nil || only(id)) {
			_, _, _ = readSeen(root, id, ref, tree, now.UTC().Truncate(time.Second), true)
		}
	}
}
