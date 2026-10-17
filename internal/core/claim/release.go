package claim

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// The release hold (itd-2609151838312703 C7 and C8, spc-2610150728396006
// step 4, resolving iss-2609091037191879): a session cutting a release
// reserves the release itself, the subject `rel-vX.Y.Z`, on the branch
// `abcd-reserved/rel-vX.Y.Z`, and every other session's land step reads it
// and holds its merge until the cut is done.
//
// A release subject is a reservation like any other, with three differences:
// it takes no footprint, since it holds every merge rather than some files;
// it sits outside the live cap, since a person cutting a release is not
// taking work; and it ends when the tag vX.Y.Z is on origin as last fetched
// (the derived release, as a record finished on the default branch ends a
// record's), or when its holder releases it. With no lease named it holds for
// lease.Max, a day, so a cut that waits overnight on its checks or its
// approval outlasts its own lease; a slower cut renews it by reserving it
// again from the same copy. Without a renewal no reservation holds past
// lease.Max from its claim, or, when it was claimed more than lease.Skew ahead
// of when this copy first fetched it, past lease.Max from that first fetch
// (lease.Judge); an unreadable one holds for lease.Default on the same terms.

// releasePrefix begins every release subject.
const releasePrefix = "rel-"

// releaseSubjectRe is the release subject's one spelling: rel-v and three
// plain numbers, with no leading zero, no pre-release and no build suffix, so
// two spellings of one release never make two branches.
var releaseSubjectRe = regexp.MustCompile(`^rel-v(0|[1-9][0-9]{0,8})\.(0|[1-9][0-9]{0,8})\.(0|[1-9][0-9]{0,8})$`)

// isRelease reports whether subject is a release subject, rel-vX.Y.Z.
func isRelease(subject string) bool { return releaseSubjectRe.MatchString(subject) }

// releaseTag is the tag a release subject names: rel-v0.16.0 names v0.16.0.
// It takes only a subject isRelease admits.
func releaseTag(subject string) string { return strings.TrimPrefix(subject, releasePrefix) }

// ReleasedByTag is what released a release reservation whose tag is on
// origin as last fetched.
const ReleasedByTag = "tag"

// ReleaseTagsTracking is where this checkout keeps what it last fetched of
// origin's release tags: one ref per tag vX.Y.Z, written only by the
// reservations' fetch and pruned with it. It sits outside refs/tags/, so the
// fetch never makes, moves or deletes a tag of the person's own, and outside
// refs/remotes/, so no tag is listed as a remote branch.
const ReleaseTagsTracking = "refs/abcd/release-tags/"

// releaseTagsRefspec is the refspec the reservations' fetch reads origin's
// release tags with.
const releaseTagsRefspec = "+refs/tags/v*:" + ReleaseTagsTracking + "v*"

// taggedOnOrigin reports whether the tag a release subject names is on origin
// as this checkout last fetched it.
func taggedOnOrigin(root, subject string) (bool, error) {
	ref := ReleaseTagsTracking + releaseTag(subject)
	out, err := localGit(root, 64<<10, "for-each-ref", "--format=%(refname)", ref)
	if err != nil {
		return false, err
	}
	for _, l := range strings.Fields(out) {
		if l == ref {
			return true, nil
		}
	}
	return false, nil
}

// Hold is a release being cut, as this checkout last fetched the
// reservations: a live release reservation, or an unreadable one within its
// hold, which holds every merge but its holder's own (C7). Every value in it
// is validated data, never a writer's text.
type Hold struct {
	// Release is the tag being cut, vX.Y.Z; Subject is the reservation's
	// subject, rel-vX.Y.Z.
	Release string `json:"release"`
	Subject string `json:"subject"`
	// Holder is the GitHub username cutting it; "" when the reservation is
	// unreadable.
	Holder string `json:"holder,omitempty"`
	// Copy is CopyThis when the reservation names this checkout's own copy.
	Copy string `json:"copy,omitempty"`
	// Mine is true when the hold is under the reader's own username, from
	// any copy; Own when it is also this copy's: the cutting session itself.
	// Only releaseHolds sets them.
	Mine bool `json:"mine"`
	Own  bool `json:"own"`
	// EffectiveExpiry is when the hold ends unless it is renewed, released or
	// its tag reaches origin first.
	EffectiveExpiry time.Time `json:"effective_expiry"`
	// Unreadable is the fixed reason the reservation cannot be read.
	Unreadable string `json:"unreadable,omitempty"`
	// Unsighted is true when the reservation is dated more than lease.Skew
	// ahead of now, or its date cannot be read (Undated), and this copy keeps
	// no first sighting of it (Reservation.Unsighted).
	Unsighted bool `json:"unsighted,omitempty"`
	Undated   bool `json:"undated,omitempty"`
}

// Text is the line every surface shows a hold by (C8): "release v0.16.0 being
// cut by @maya: merges are held".
func (h Hold) Text() string {
	if h.Unreadable != "" {
		return fmt.Sprintf("release %s held by an unreadable reservation (%s) until %s: merges are held",
			h.Release, h.Unreadable, stamp(h.EffectiveExpiry))
	}
	return fmt.Sprintf("release %s being cut by @%s: merges are held", h.Release, h.Holder)
}

// Line is the line a listing shows a judged hold by (JudgedHolds), by the
// rule the merge hold counts it with: the cutting copy's own release as the
// cut it is making ("you are cutting release v0.16.0 from this copy: every
// other session's merges are held"), a release dated ahead whose first
// sighting this copy cannot keep as not counted, in the words `abcd land`'s
// notice uses, and any other by Text.
func (h Hold) Line() string {
	switch {
	case h.Own:
		return fmt.Sprintf("you are cutting release %s from this copy: every other session's merges are held", h.Release)
	case h.Unsighted:
		return h.notCounted() + " merges"
	}
	return h.Text()
}

// Holds is every release the view shows being cut at now: each live release
// reservation, and each unreadable one within its hold, in the view's order.
// It reads nothing; Own is left false.
func (s Snapshot) Holds(now time.Time) []Hold {
	var out []Hold
	for _, r := range s.Reservations {
		if !isRelease(r.Record) {
			continue
		}
		switch {
		case r.State == StateLive:
		case r.State == StateUnreadable && now.Before(r.EffectiveExpiry):
		default:
			continue
		}
		out = append(out, Hold{Release: releaseTag(r.Record), Subject: r.Record, Holder: r.Holder, Copy: r.Copy,
			EffectiveExpiry: r.EffectiveExpiry, Unreadable: r.Unreadable, Unsighted: r.Unsighted, Undated: r.HeldFromFetch})
	}
	return out
}

// releaseHolds is every release being cut as this checkout last fetched the
// reservations, from disk alone: no fetch, no push and nothing written
// (invariant 7). Each hold's Mine says whether it is under the username a
// reservation verb here would resolve, and Own whether it is also from this
// copy. JudgedHolds reads it, for MergeHolds, which judges a merge from it for
// the land fence and `abcd land`, and for `abcd peers`, which shows each hold
// by the same rule (Hold.Line), as `abcd status` and the prompt hook do. It
// reads the release reservations' tips alone.
func releaseHolds(root string, now time.Time, o Options) ([]Hold, error) {
	now = now.UTC().Truncate(time.Second)
	snap, err := viewOf(root, now, true, isRelease)
	if err != nil {
		return nil, err
	}
	holds := snap.Holds(now)
	if len(holds) == 0 {
		return nil, nil
	}
	reader := readerHolder(root, o)
	for i, h := range holds {
		holds[i].Mine = h.Unreadable == "" && reader != "" && sameHolder(h.Holder, reader)
		holds[i].Own = holds[i].Mine && h.Copy == CopyThis
	}
	return holds, nil
}

// ReleaseHoldsForTest is releaseHolds, for the implement loop's tests of the
// release hold, which read it from outside this package.
func ReleaseHoldsForTest(root string, now time.Time, o Options) ([]Hold, error) {
	return releaseHolds(root, now, o)
}
