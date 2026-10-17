package claim

// recenthandbacks.go is each holder's recent hand-backs, as a listing shows
// them (spc-2610161044461448 step 18; iss-2610161015434746;
// dcn-2610161040394970): the released tips that mark a hand-back, from the
// view as last fetched, so another account sees what a drain handed back of
// late without a network act of its own.

import (
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/lease"
)

// HandBackWindow is how far back a listing reads hand-backs: the last day.
const HandBackWindow = 24 * time.Hour

// HandBacksShown is how many of one holder's recent hand-backs a listing
// shows, the newest; the rest are counted, never dropped silently. Ten keeps
// one holder's block to a screenful while a session reading it before a pick
// still sees the latest of a busy day (dcn-2610170757251547).
const HandBacksShown = 10

// RecentHandBack is one hand-back tombstone a listing shows: the record, the
// kind, one of HandBackKinds, and the tombstone's Abcd-Released-At.
type RecentHandBack struct {
	Record     string
	Kind       string
	ReleasedAt time.Time
}

// HolderHandBacks is one holder's recent hand-backs: the newest
// HandBacksShown, newest first, and how many more the window holds.
type HolderHandBacks struct {
	Holder    string
	HandBacks []RecentHandBack
	More      int
}

// RecentHandBacks is every holder's hand-back tombstones released within
// HandBackWindow of now, by holder (usernames compared without regard to case
// and named as first listed, ordered by name), each holder's newest first and
// bounded by HandBacksShown. Only a reservation's tip is read, so a hand-back
// since reserved again is not one. A released-at more than lease.Skew ahead of
// now is never trusted: such a tombstone is not listed, so a clock running
// ahead cannot keep a hand-back on show past its day (dcn-2610170757256374).
// It reads the snapshot alone and writes nothing.
func (s Snapshot) RecentHandBacks(now time.Time) []HolderHandBacks {
	now = now.UTC().Truncate(time.Second)
	since := now.Add(-HandBackWindow)
	byKey := map[string]*HolderHandBacks{}
	var keys []string
	for _, r := range s.Reservations {
		if r.State != StateReleased || r.ReleasedBy != ReleasedByTombstone || r.HandedBack == "" {
			continue
		}
		if r.ReleasedAt.Before(since) || lease.Ahead(r.ReleasedAt, now) {
			continue
		}
		key := strings.ToLower(r.Holder)
		h, ok := byKey[key]
		if !ok {
			h = &HolderHandBacks{Holder: r.Holder}
			byKey[key] = h
			keys = append(keys, key)
		}
		h.HandBacks = append(h.HandBacks, RecentHandBack{Record: r.Record, Kind: r.HandedBack, ReleasedAt: r.ReleasedAt})
	}
	sort.Strings(keys)
	out := make([]HolderHandBacks, 0, len(keys))
	for _, k := range keys {
		h := byKey[k]
		sort.Slice(h.HandBacks, func(i, j int) bool {
			a, b := h.HandBacks[i], h.HandBacks[j]
			if !a.ReleasedAt.Equal(b.ReleasedAt) {
				return a.ReleasedAt.After(b.ReleasedAt)
			}
			return a.Record < b.Record
		})
		if n := len(h.HandBacks); n > HandBacksShown {
			h.More, h.HandBacks = n-HandBacksShown, h.HandBacks[:HandBacksShown]
		}
		out = append(out, *h)
	}
	return out
}
