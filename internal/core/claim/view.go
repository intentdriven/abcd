package claim

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/lease"
	"github.com/intentdriven/abcd/internal/core/peers"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The states a listing shows a reservation in.
const (
	StateLive       = "live"
	StateExpired    = "expired"
	StateReleased   = "released"
	StateUnreadable = "unreadable"
)

// What released a reservation: its own tombstone, the record finished on
// origin's default branch (the derived release), or for a release subject its
// tag on origin (ReleasedByTag, release.go).
const (
	ReleasedByTombstone     = "tombstone"
	ReleasedByDefaultBranch = "default-branch"
)

// ErrNoOrigin is Fetch's answer for a checkout with no remote named origin:
// there are no reservations to read, which is not a failed fetch.
var ErrNoOrigin = errors.New("this checkout has no remote named origin")

// Reservation is one reservation as the view holds it. Every value in it is
// data a writer chose: a surface quotes it and never treats it as instruction.
type Reservation struct {
	Record string
	// Holder is the GitHub username the tip names; "" when it is unreadable.
	Holder string
	// Copy is CopyThis when the tip names this checkout's own copy, else "".
	Copy      string
	Footprint []string
	ClaimedAt time.Time
	// EffectiveExpiry is when it stops holding: the lease clamped to a day,
	// or an unreadable tip's hold. A tip dated more than lease.Skew ahead of
	// the reader counts from when this copy first fetched it (lease.Judge).
	EffectiveExpiry time.Time
	// State is StateLive, StateExpired, StateReleased or StateUnreadable.
	State string
	// ReleasedBy says which release, when State is StateReleased.
	ReleasedBy string
	// Unreadable is the fixed reason the tip cannot be read.
	Unreadable string
	// HeldFromFetch is true when an unreadable tip's hold is counted from
	// when this copy first fetched it, its commit date being unreadable.
	HeldFromFetch bool
	// Unsighted is true when the tip needs a first sighting and none is kept
	// here: its own date (its claimed-at, or its commit date when it is
	// unreadable) is more than lease.Skew ahead of now, or that date cannot
	// be read. lease.Judge then counts it from now, so at every reading it
	// holds as if first seen then.
	Unsighted bool
	Commit    string
}

// Snapshot is the reservations as this checkout last fetched them.
type Snapshot struct {
	// FetchedAt is when abcd last fetched the reservations here; zero when
	// nothing here stamped a fetch, so the view's age is unknown.
	FetchedAt    time.Time
	Reservations []Reservation
}

// LiveByHolder counts each person's live reservations of records, as the cap
// counts them: usernames compared without regard to case and named as first
// listed, and a release subject, which sits outside the cap, not counted.
func (s Snapshot) LiveByHolder() map[string]int {
	out := map[string]int{}
	names := map[string]string{}
	for _, r := range s.Reservations {
		if r.State != StateLive || IsRelease(r.Record) {
			continue
		}
		key := strings.ToLower(r.Holder)
		if _, ok := names[key]; !ok {
			names[key] = r.Holder
		}
		out[names[key]]++
	}
	return out
}

// Fetch reads every reservation branch from origin, with pruning, and the
// default branch, and stamps when it did, as every act does before it judges;
// it is the listing's one network act. gap is why the default branch could
// not be read, when it could not. A checkout with no remote named origin
// answers ErrNoOrigin. After a fetch it keeps, for each tip dated more than
// lease.Skew ahead (and each whose date cannot be read), when this copy first
// fetched it, so the view, which writes nothing, counts it from then.
func Fetch(root string, o Options) (gap string, err error) {
	defer func() {
		if err == nil {
			keepFirstFetched(root, o.now(), nil)
		}
	}()
	out, err := localGit(root, 64<<10, "remote")
	if err != nil {
		return "", refuse("could not list this checkout's remotes: %s; nothing was fetched", redacted(err))
	}
	if !slices.Contains(strings.Fields(out), Remote) {
		return "", ErrNoOrigin
	}
	return fetch(root, o.now(), noReservation, reservations)
}

// View reads the reservations from this checkout's disk only, as last
// fetched: no fetch and no push. Each is judged at now by lease.Judge, after
// the derived release on origin's default branch as last fetched; nothing is
// written. It is the "as of last fetch" reader every render shares.
func View(root string, now time.Time) (Snapshot, error) {
	return view(root, now, true)
}

// view is View, applying the derived release only when derive is true: an act
// whose fetch could not read origin's default branch does not trust the copy
// this clone holds of it.
func view(root string, now time.Time, derive bool) (Snapshot, error) {
	now = now.UTC().Truncate(time.Second)
	snap := Snapshot{FetchedAt: lastFetched(root, reservations), Reservations: []Reservation{}}
	out, err := localGit(root, 1<<20, "for-each-ref", "--format=%(refname)", TrackingPrefix)
	if err != nil {
		return snap, fmt.Errorf("reading the reservations: %s", redacted(err))
	}
	var refs []string
	for _, ref := range strings.Fields(out) {
		id := strings.TrimPrefix(ref, TrackingPrefix)
		if canon, err := reservableID(id); err == nil && canon == id {
			refs = append(refs, ref)
		}
	}
	if len(refs) == 0 {
		return snap, nil
	}
	tree, err := emptyTree(root)
	if err != nil {
		return snap, fmt.Errorf("reading the reservations: %s", redacted(err))
	}
	var finished map[string][]string
	if ref := originDefault(root); derive && ref != "" {
		if finished, err = peers.FoldersAt(root, ref); err != nil {
			return snap, fmt.Errorf("reading %s: %s", gitutil.ShortRef(ref), redacted(err))
		}
	}
	// The release tags are read only when a release subject is there to end.
	var tagged map[string]bool
	if slices.ContainsFunc(refs, func(ref string) bool { return IsRelease(strings.TrimPrefix(ref, TrackingPrefix)) }) {
		if tagged, err = releaseTags(root); err != nil {
			return snap, fmt.Errorf("reading the release tags: %s", redacted(err))
		}
	}
	own := ownCopyKey(root)
	dated := snap.FetchedAt
	if dated.IsZero() {
		dated = now
	}
	for _, ref := range refs {
		id := strings.TrimPrefix(ref, TrackingPrefix)
		t, ok, err := readSeen(root, id, ref, tree, dated, false)
		if err != nil {
			return snap, fmt.Errorf("reading the reservation on %s: %s", id, redacted(err))
		}
		if !ok {
			continue
		}
		by := ""
		switch {
		case IsRelease(id) && tagged[releaseTag(id)]:
			by = ReleasedByTag
		case !IsRelease(id) && terminal(finished, id):
			by = ReleasedByDefaultBranch
		}
		snap.Reservations = append(snap.Reservations, listed(t, id, by, own, now))
	}
	return snap, nil
}

// listed is one tip as the view shows it, judged at now. derivedBy is what
// released it outside its own tip (ReleasedByDefaultBranch or ReleasedByTag),
// "" when nothing did.
func listed(t tip, id, derivedBy, own string, now time.Time) Reservation {
	r := Reservation{Record: id, Commit: t.Commit, Unreadable: t.Unreadable, HeldFromFetch: t.DatedFromFetch}
	date := t.ClaimedAt
	if t.Unreadable != "" {
		date = t.WrittenAt
	}
	r.Unsighted = t.FirstSeen.IsZero() && (t.DatedFromFetch || lease.Ahead(date, now))
	if t.Unreadable != "" {
		r.EffectiveExpiry = lease.Judge(lease.Stamp{Unreadable: true, WrittenAt: t.WrittenAt, FirstSeen: t.FirstSeen}, lease.Default, now).EffectiveExpiry
		r.State = StateUnreadable
	} else {
		v := lease.Judge(lease.Stamp{ClaimedAt: t.ClaimedAt, ExpiresAt: t.ExpiresAt, FirstSeen: t.FirstSeen}, lease.Default, now)
		r.Holder, r.Footprint, r.ClaimedAt, r.EffectiveExpiry = t.Holder, t.Footprint, t.ClaimedAt, v.EffectiveExpiry
		if own != "" && t.Copy == own {
			r.Copy = CopyThis
		}
		switch {
		case t.message.released():
			r.State, r.ReleasedBy = StateReleased, ReleasedByTombstone
		case v.Live:
			r.State = StateLive
		default:
			r.State = StateExpired
		}
	}
	if derivedBy != "" && r.ReleasedBy == "" {
		r.State, r.ReleasedBy = StateReleased, derivedBy
	}
	return r
}

// releaseTags is the set of release tags on origin as last fetched, by tag.
func releaseTags(root string) (map[string]bool, error) {
	out, err := localGit(root, 1<<20, "for-each-ref", "--format=%(refname)", ReleaseTagsTracking)
	if err != nil {
		return nil, err
	}
	tags := map[string]bool{}
	for _, ref := range strings.Fields(out) {
		tags[strings.TrimPrefix(ref, ReleaseTagsTracking)] = true
	}
	return tags, nil
}

// terminal reports whether id sits in a terminal folder in finished, the
// default branch's record folders.
func terminal(finished map[string][]string, id string) bool {
	for _, f := range finished[id] {
		if slices.Contains(terminalFolders[id[:3]], f) {
			return true
		}
	}
	return false
}

// lastFetched is the time this checkout's last fetch of ns stamped; zero when
// no stamp reads.
func lastFetched(root string, ns namespace) time.Time {
	r, err := os.OpenRoot(root)
	if err != nil {
		return time.Time{}
	}
	defer r.Close()
	data, err := fsutil.ReadGuardedInRoot(r, ns.stampRel(), 64)
	if err != nil {
		return time.Time{}
	}
	at, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
	if err != nil {
		return time.Time{}
	}
	return at.UTC()
}

// ownCopyKey is this checkout's copy key when one is kept, read without
// minting one; "" otherwise.
func ownCopyKey(root string) string {
	r, err := os.OpenRoot(root)
	if err != nil {
		return ""
	}
	defer r.Close()
	data, err := fsutil.ReadGuardedInRoot(r, copyKeyRel, 256)
	if err != nil {
		return ""
	}
	if key := strings.TrimSpace(string(data)); validCopyKey(key) {
		return key
	}
	return ""
}
