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
// commit, so a later fetch does not restart it.

// readSeen is readTip, with the tip's first sighting set when it is dated
// ahead of fetchedAt (seen).
func readSeen(root, record, ref, tree string, fetchedAt time.Time, keep bool) (tip, bool, error) {
	t, ok, err := readTip(root, record, ref, tree, fetchedAt, keep)
	if err != nil || !ok {
		return t, ok, err
	}
	return seen(root, t, fetchedAt, keep), true, nil
}

// seen gives a tip whose own date (its claimed-at, or its commit date when it
// is unreadable) is more than lease.Skew ahead of fetchedAt the time this
// copy first fetched it. A reader at a later now than fetchedAt counts from it
// only a tip still ahead of its own now (lease.Judge); one ahead of fetchedAt
// alone is judged by its own dates.
func seen(root string, t tip, fetchedAt time.Time, keep bool) tip {
	date := t.ClaimedAt
	if t.Unreadable != "" {
		date = t.WrittenAt
	}
	if lease.Ahead(date, fetchedAt) {
		t.FirstSeen = firstFetched(root, t.Commit, fetchedAt, keep)
	}
	return t
}

// keepFirstFetched reads every reservation tip as last fetched with keep, so
// the time this copy first fetched each tip that needs one, dated ahead or
// undated, is recorded. It is best effort: a tip it cannot read is left to
// the reader that reports it.
func keepFirstFetched(root string, now time.Time) {
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
		if canon, err := reservableID(id); err == nil && canon == id {
			_, _, _ = readSeen(root, id, ref, tree, now.UTC().Truncate(time.Second), true)
		}
	}
}
