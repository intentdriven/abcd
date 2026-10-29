package claim

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/lease"
	"github.com/intentdriven/abcd/internal/core/peers"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/localtier"
	"github.com/intentdriven/abcd/internal/termsafe"
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
	Copy string
	// key is the copy key the tip names, for finding it among this
	// checkout's other worktrees (siblingCopy); never shown.
	key string
	// Sibling is the name of the other worktree of this repository, on this
	// machine, whose copy made the tip (iss-2610231004140527): the fewest
	// trailing folders of its path no other worktree's path ends in, never
	// its key; "" for this copy, for a copy elsewhere, and until
	// NameSiblings fills it. It is display only: no judgement reads it.
	Sibling   string
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
	// ReleasedAt is a tombstone's own time.
	ReleasedAt time.Time
	// HandedBack is a tombstone's hand-back kind, "" for a plain one, and
	// RecordChanged is true when origin's default branch as last fetched
	// holds the record at another blob than the one the tombstone names: a
	// person has changed it since, so the hand-back no longer stands.
	HandedBack    string
	RecordChanged bool
	// Stage is the drain lane's stage the tip's renewal names, one of
	// LaneStages; "" when the tip names none.
	Stage string
}

// Snapshot is the reservations as this checkout last fetched them.
type Snapshot struct {
	// FetchedAt is when abcd last fetched the reservations here; zero when
	// nothing here stamped a fetch, so the view's age is unknown.
	FetchedAt    time.Time
	Reservations []Reservation
	// Generated is the files origin's default branch, as last fetched,
	// declares generated (GeneratedFiles), which no footprint overlap is
	// judged on. It is read only when some live reservation has a footprint,
	// and is empty when the view does not derive (an act whose fetch could
	// not read origin's default branch).
	Generated Generated
	// GeneratedUnread is why generated files were not recognised, though a
	// live reservation has a footprint: the declaration could not be read, or
	// the view does not trust origin's default branch as this clone holds
	// it; "" otherwise.
	GeneratedUnread string
	// SiblingsUnread is why NameSiblings could not list this repository's
	// worktrees, "" otherwise.
	SiblingsUnread string
	siblingsNamed  bool
}

// LiveByHolder counts each person's live reservations of records, as the cap
// counts them: usernames compared without regard to case and named as first
// listed, and a release subject, which sits outside the cap, not counted.
func (s Snapshot) LiveByHolder() map[string]int {
	out := map[string]int{}
	names := map[string]string{}
	for _, r := range s.Reservations {
		if r.State != StateLive || isRelease(r.Record) {
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
	return viewOf(root, now, derive, nil)
}

// viewOf is view over the subjects only admits (every subject when only is
// nil): a reader that needs the releases alone (releaseHolds) reads no other
// tip, and origin's default branch only when a record is there to derive.
func viewOf(root string, now time.Time, derive bool, only func(subject string) bool) (Snapshot, error) {
	now = now.UTC().Truncate(time.Second)
	// The fetch stamp says when this copy last read origin; one git tracks
	// came with a commit and says nothing of this copy. Asked only of a stamp
	// there is.
	if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(reservations.stampRel()))); err == nil {
		if err := localtier.Untracked(root, reservations.stampRel()); err != nil {
			return Snapshot{}, err
		}
	}
	snap := Snapshot{FetchedAt: lastFetched(root, reservations), Reservations: []Reservation{}}
	out, err := localGit(root, 1<<20, "for-each-ref", "--format=%(refname)", TrackingPrefix)
	if err != nil {
		return snap, fmt.Errorf("reading the reservations: %s", redacted(err))
	}
	var refs []string
	for _, ref := range strings.Fields(out) {
		id := strings.TrimPrefix(ref, TrackingPrefix)
		if canon, err := reservableID(id); err == nil && canon == id && (only == nil || only(id)) {
			refs = append(refs, ref)
		}
	}
	if len(refs) == 0 {
		return snap, nil
	}
	tree, err := emptyTreeName(root)
	if err != nil {
		return snap, fmt.Errorf("reading the reservations: %s", redacted(err))
	}
	// The records on origin's default branch as last fetched: a finished one
	// is released whatever its tip says, and a hand-back's blob is compared
	// with the one its tombstone names. A view that does not derive reads
	// neither, so its hand-backs stand; nor does one listing release
	// subjects alone, which no record there ends.
	var records map[string][]peers.RecordAt
	anyRecord := slices.ContainsFunc(refs, func(ref string) bool { return !isRelease(strings.TrimPrefix(ref, TrackingPrefix)) })
	// One name for origin's default branch per view (iss-2610180545134593): a
	// view fetches nothing, so the name cannot move under it.
	defaultRef := originDefault(root)
	if ref := defaultRef; derive && anyRecord && ref != "" {
		if records, err = peers.RecordsAt(root, ref); err != nil {
			return snap, fmt.Errorf("reading %s: %s", gitutil.ShortRef(ref), redacted(err))
		}
	}
	// The release tags are read only when a release subject is there to end.
	var tagged map[string]bool
	if slices.ContainsFunc(refs, func(ref string) bool { return isRelease(strings.TrimPrefix(ref, TrackingPrefix)) }) {
		if tagged, err = releaseTags(root); err != nil {
			return snap, fmt.Errorf("reading the release tags: %s", redacted(err))
		}
	}
	own := ownCopyKey(root)
	dated := snap.FetchedAt
	if dated.IsZero() {
		dated = now
	}
	// The tips in one batch where they read cleanly, each as readSeen reads
	// it (iss-2610180545134593).
	reads := readSeenTips(root, refs, tree, dated, false)
	for i, ref := range refs {
		id := strings.TrimPrefix(ref, TrackingPrefix)
		t, ok, err := reads[i].t, reads[i].ok, reads[i].err
		if err != nil {
			return snap, fmt.Errorf("reading the reservation on %s: %s", id, redacted(err))
		}
		if !ok {
			continue
		}
		by := ""
		switch {
		case isRelease(id) && tagged[releaseTag(id)]:
			by = ReleasedByTag
		case !isRelease(id) && terminal(records[id], id):
			by = ReleasedByDefaultBranch
		}
		r := listed(t, id, by, own, now)
		if r.HandedBack != "" {
			r.RecordChanged = changedSince(t.RecordBlob, blobOf(records[id]))
		}
		snap.Reservations = append(snap.Reservations, r)
	}
	// The generated files are read once per view, and only when a live
	// footprint is there to judge an overlap with; a view that does not trust
	// origin's default branch recognises none.
	if slices.ContainsFunc(snap.Reservations, func(r Reservation) bool { return r.State == StateLive && len(r.Footprint) > 0 }) {
		if derive {
			snap.Generated, snap.GeneratedUnread = generatedFilesAt(root, defaultRef)
		} else {
			snap.GeneratedUnread = "origin's default branch could not be read when the reservations were fetched"
		}
	}
	return snap, nil
}

// YourCopy names the copy of the person's own that holds r when it is not
// this one: "your worktree <path> on this machine" when a sibling worktree of
// the checkout at root keeps its key, else "your copy on another computer"
// (iss-2610250013552001).
func (r Reservation) YourCopy(root string) string { return yourCopyAt(siblingCopy(root, r.key)) }

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
		r.Stage, r.key = t.Stage, t.Copy
		if own != "" && t.Copy == own {
			r.Copy = CopyThis
		}
		switch {
		case t.message.released():
			r.State, r.ReleasedBy = StateReleased, ReleasedByTombstone
			r.ReleasedAt, r.HandedBack = t.ReleasedAt, t.HandedBack
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

// NameSiblings fills Sibling on self's tips (Self.Owns) made from another worktree of
// the repository at root on this machine, reading each worktree's kept copy
// key once (iss-2610231004140527). It is the only reader of the worktrees, so
// a view that never shows a sibling never lists them; a second call does
// nothing. A tip of anyone else is never named, since a copy key is no secret
// from whoever pushed a tip. When the worktrees cannot be listed, every tip
// reads as made elsewhere and SiblingsUnread says why.
func (s *Snapshot) NameSiblings(root string, self Self) {
	if s.siblingsNamed {
		return
	}
	s.siblingsNamed = true
	var sibs map[string]siblingAt
	for i := range s.Reservations {
		r := &s.Reservations[i]
		if r.Copy == CopyThis || !validCopyKey(r.key) || !self.Owns(r.Holder) {
			continue
		}
		if sibs == nil {
			sibs, s.SiblingsUnread = siblingCopies(root, ownCopyKey(root))
		}
		r.Sibling = sibs[r.key].Name
	}
}

// siblingAt is one sibling worktree as abcd names it: Name, the shortest
// tail of its path that tells it apart (siblingTail), where a listing shows
// it, and Path, home-redacted, where a hold's words name it; both
// terminal-safe. A copied worktree, two holding one key, joins both of each
// with " or ".
type siblingAt struct{ Name, Path string }

// siblingCopies maps the copy key kept in each other worktree of the
// repository at root to that worktree, own's excluded, read without minting a
// key. It is the one reading of the worktrees every surface names a sibling
// from (iss-2610272352153282). unread is why the worktrees could not be
// listed, "" when they were.
func siblingCopies(root, own string) (sibs map[string]siblingAt, unread string) {
	sibs = map[string]siblingAt{}
	wts, err := gitutil.ListWorktrees(root, gitutil.MaxWorktreeListing)
	if err != nil {
		return sibs, "this repository's worktrees could not be listed (" + redacted(err) + "), so holds from them count as made elsewhere"
	}
	var paths []string
	for _, wt := range wts {
		if !wt.Bare {
			paths = append(paths, wt.Path)
		}
	}
	for _, wt := range wts {
		if wt.Bare {
			continue
		}
		key := ownCopyKey(wt.Path)
		if key == "" || key == own {
			continue
		}
		at := siblingAt{Name: termsafe.Sanitize(siblingTail(wt.Path, paths)), Path: termsafe.Sanitize(fsutil.RedactHome(wt.Path))}
		if prev, ok := sibs[key]; ok {
			at = siblingAt{Name: prev.Name + " or " + at.Name, Path: prev.Path + " or " + at.Path}
		}
		sibs[key] = at
	}
	return sibs, ""
}

// siblingTail is the fewest trailing folders of p that no other of paths ends
// in: its directory's name alone unless another worktree shares it
// (iss-2610272352168723), and the whole of p, home-redacted, when every tail
// is shared. Tails are cut from the home-redacted paths, so the folders of
// the home directory itself never show.
func siblingTail(p string, paths []string) string {
	segs := tailSegments(p)
	for k := 1; k < len(segs); k++ {
		tail := strings.Join(segs[len(segs)-k:], "/")
		shared := false
		for _, q := range paths {
			qs := tailSegments(q)
			if q != p && len(qs) >= k && strings.Join(qs[len(qs)-k:], "/") == tail {
				shared = true
				break
			}
		}
		if !shared {
			return tail
		}
	}
	return fsutil.RedactHome(p)
}

// tailSegments is p cleaned and home-redacted, split into its folders.
func tailSegments(p string) []string {
	return strings.Split(filepath.ToSlash(fsutil.RedactHome(filepath.Clean(p))), "/")
}

// ownCopyKey is the copy key the checkout at root keeps, read without minting
// one; "" when it keeps none that is valid. It is the one reader of a kept key,
// this checkout's or a sibling worktree's (siblingCopy, siblingCopies).
func ownCopyKey(root string) string {
	r, err := os.OpenRoot(root)
	if err != nil {
		return ""
	}
	defer r.Close()
	data, err := localtier.ReadInRoot(r, copyKeyRel, 256)
	if err != nil {
		return ""
	}
	if key := strings.TrimSpace(string(data)); validCopyKey(key) {
		return key
	}
	return ""
}
