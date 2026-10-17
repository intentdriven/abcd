package claim

// handback.go is the hand-back tombstone (spc-2610161044461448 step 3;
// iss-2610161015370582; dcn-2610161040365515): a drain that hands an issue
// back to a person publishes the fact on origin, the one place every account
// shares, as the release of the reservation it held, carrying the hand-back's
// kind and the blob id of the issue's record on origin's default branch. Both
// are fixed-form, a kind from a closed vocabulary and an object name, so no
// free text reaches the remote (dcn-2610150000000001). A reservation a drain
// makes backs off such a tombstone while the record's blob is unchanged, since
// the decision the hand-back names is a person's; a person's own reservation
// goes ahead and is told of it. A changed blob is a person's edit of the
// record, which re-admits the issue. The vocabulary and its fail-closed
// reading are dcn-2610161412287479; when a hand-back stands, and the blob it
// is judged by, dcn-2610161412283450.

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/peers"
)

// The kinds a hand-back tombstone names (Abcd-Handed-Back): the four a lane's
// receipt or the drain's judgement hands an issue back as, a lane stopped
// after its fix rounds, a lane stuck refusing at one stage, and a lane whose
// pull request was closed unmerged. A drain writes stuck for a lane it hands
// back after its bound (spc-2610161044461448 step 5), and closed for a lane
// parked at land whose pull request the forge lists closed (step 8).
const (
	HandBackUserVisible   = "user-visible"
	HandBackTrustRule     = "trust-rule"
	HandBackDesignFinding = "design-finding"
	HandBackSecondPackage = "second-package"
	HandBackUnachievable  = "unachievable"
	HandBackStuck         = "stuck"
	HandBackClosed        = "closed"
)

// HandBackKinds is the closed vocabulary, in the order the brief lists it.
var HandBackKinds = []string{HandBackUserVisible, HandBackTrustRule, HandBackDesignFinding, HandBackSecondPackage,
	HandBackUnachievable, HandBackStuck, HandBackClosed}

// longestBlob stands in for a record blob when a reservation is measured
// against the tombstone that will one day release it: a SHA-256 object name,
// the longer of git's two.
var longestBlob = strings.Repeat("0", 64)

// longestHandBack is the longest kind in the vocabulary.
func longestHandBack() string {
	long := ""
	for _, k := range HandBackKinds {
		if len(k) > len(long) {
			long = k
		}
	}
	return long
}

// HandBack is a hand-back tombstone a person's own reservation was made over,
// its record unchanged on origin's default branch since: the kind, whose it
// was and when it was handed back. Every value is data a writer chose from a
// closed form.
type HandBack struct {
	Kind   string    `json:"kind"`
	Holder string    `json:"holder"`
	At     time.Time `json:"at"`
}

// Note is the sentence a person's own reservation of id over h is told.
func (h HandBack) Note(id string) string {
	return fmt.Sprintf("%s was handed back by %s as %s at %s, and its record on %s's default branch is unchanged since; this reservation is a person's own, so it was made",
		id, h.Holder, h.Kind, stamp(h.At), Remote)
}

// KeptNote is the sentence a plain release of id that kept h standing is told.
func (h HandBack) KeptNote(id string) string {
	return fmt.Sprintf("%s was handed back by %s as %s at %s, and its record on %s's default branch is unchanged since, so the hand-back still stands: "+
		"this release's tombstone keeps it, and no drain takes %s until a person changes the record", id, h.Holder, h.Kind, stamp(h.At), Remote, id)
}

// keptWalk bounds how many commits below a released reservation's tip a
// release reads for the tombstone the reservation was made over: a
// reservation renewed more often than that since keeps no hand-back.
const keptWalk = 256

// keptHandBack is the hand-back a plain release of j, this person's
// reservation from this copy, keeps: the standing hand-back the reservation
// was made over (standingBelow). why is the reason the walk could not be
// read, "" when it was (dcn-2610161806433529).
func keptHandBack(p *prepared, j judged) (kept *tip, why string) {
	kept, err := standingBelow(p, j)
	if err != nil {
		return nil, "could not read whether the reservation on " + p.id + " was made over a hand-back (" + redacted(err) + "), so the tombstone keeps none"
	}
	return kept, ""
}

// standingBelow is the hand-back tombstone the reservation at j's tip was
// made over, when it still stands: the first tombstone on the reservation
// branch's first-parent line below the tip, when it marks a hand-back whose
// record origin's default branch holds unchanged (handBackStands' reading).
// Every reservation commit above it, renewals and take-overs among them, is
// passed over; an unreadable one, a plain tombstone, the branch's root and the
// bound each end the walk with nil. The error is a walk that could not be
// read.
func standingBelow(p *prepared, j judged) (*tip, error) {
	line, err := firstParents(p.root, trackingName(p.id), j.Commit, keptWalk)
	if err != nil {
		return nil, err
	}
	tree, err := emptyTree(p.root)
	if err != nil {
		return nil, err
	}
	for _, c := range line[1:] {
		h, m, err := readCommit(p.root, c, tree, p.now, false, reservationParser(p.id))
		if err != nil {
			return nil, err
		}
		if h.Unreadable != "" {
			return nil, nil
		}
		if !m.released() {
			continue
		}
		if m.HandedBack == "" || changedSince(m.RecordBlob, j.blob) {
			return nil, nil
		}
		return &tip{head: h, message: m}, nil
	}
	return nil, nil
}

// drainOverHandBack refuses a drain's take-over or renewal of the reservation
// at j's tip when that reservation was made over a hand-back that still
// stands: a person reserved the issue over it by hand, and a reservation of
// theirs, lapsed or not, hands the issue to no drain. A walk that cannot be
// read backs off too, since it cannot say the hand-back does not stand
// (dcn-2610161931122606).
func drainOverHandBack(p *prepared, j judged) error {
	kept, err := standingBelow(p, j)
	if err != nil {
		return backOff("could not read whether the reservation on %s was made over a hand-back (%s), so a drain neither takes it over nor renews it; take other work; %s",
			p.id, redacted(err), nothingPushed)
	}
	if kept == nil {
		return nil
	}
	return backOff("the reservation on %s was made over a hand-back by %s as %s at %s, and its record on %s's default branch is unchanged since: "+
		"a drain neither takes it over nor renews it until a person changes the record; take other work; %s",
		p.id, kept.Holder, kept.HandedBack, stamp(kept.ReleasedAt), Remote, nothingPushed)
}

// firstParent is commit's first parent, "" for a root commit.
func firstParent(root, commit string) (string, error) {
	line, err := firstParents(root, commit, commit, 1)
	if err != nil || len(line) < 2 {
		return "", err
	}
	return line[1], nil
}

// handBackUnfit is why m, a tombstone marking a hand-back, may not be written
// with it: past the message cap, or refused by the outbound policy. "" when it
// may.
func handBackUnfit(root string, m message) string {
	text := m.text()
	if n := len(text); n > MaxMessageBytes {
		return fmt.Sprintf("the tombstone marking the hand-back would be %d bytes, over the %d-byte cap, so a plain tombstone released it", n, MaxMessageBytes)
	}
	if err := checkMessage(root, text); err != nil {
		return "the tombstone marking the hand-back could not be written (" + err.Error() + "), so a plain tombstone released it"
	}
	return ""
}

// validateHandBack reads a message's hand-back trailers into m, which
// validateReservation has filled: none, or exactly one kind from the
// vocabulary and one object name, on a tombstone. It returns reasonHandBack
// for anything else.
func validateHandBack(m *message, t trailers) string {
	kinds, blobs := t.vals[keyHandedBack], t.vals[keyRecordBlob]
	if len(kinds) == 0 && len(blobs) == 0 {
		return ""
	}
	if len(kinds) != 1 || len(blobs) != 1 || !slices.Contains(HandBackKinds, kinds[0]) || !validBlob(blobs[0]) || !m.released() {
		return reasonHandBack
	}
	m.HandedBack, m.RecordBlob = kinds[0], blobs[0]
	return ""
}

// validBlob reports whether s is an object name in full, lowercase hex.
func validBlob(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, r := range s {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

// defaultRecords is every record origin's default branch holds as last
// fetched, by id, each with its folders and blobs; nil when this checkout
// names no default branch of origin.
func defaultRecords(root string) (map[string][]peers.RecordAt, error) {
	ref := originDefault(root)
	if ref == "" {
		return nil, nil
	}
	return peers.RecordsAt(root, ref)
}

// terminal reports whether a record held in recs sits in a terminal folder.
func terminal(recs []peers.RecordAt, id string) bool {
	return slices.ContainsFunc(recs, func(r peers.RecordAt) bool { return slices.Contains(terminalFolders[id[:3]], r.Folder) })
}

// blobOf is the blob of the one file recs holds a record in, "" when it is
// held in none or in several.
func blobOf(recs []peers.RecordAt) string {
	if len(recs) != 1 {
		return ""
	}
	return recs[0].Blob
}

// changedSince reports whether a hand-back tombstone naming blob was made
// over another record than origin's default branch holds now, current: only
// a blob read there and differing from the tombstone's. A record the branch
// holds in no file, or several, or a branch that cannot be read, is not read
// as changed, so the hand-back stands.
func changedSince(blob, current string) bool { return current != "" && current != blob }

// handBackStands reports whether the judged tip is a hand-back tombstone
// whose record is unchanged on origin's default branch.
func (j judged) handBackStands() bool {
	return j.present && j.Unreadable == "" && j.message.released() && j.HandedBack != "" && !changedSince(j.RecordBlob, j.blob)
}

// handBackBackOff is a drain's reservation refused over a standing hand-back.
func (j judged) handBackBackOff(id string) error {
	return backOff("%s was handed back by %s as %s at %s, and its record on %s's default branch is unchanged since: "+
		"a drain takes it again only once a person changes the record; take other work; %s",
		id, j.Holder, j.HandedBack, stamp(j.ReleasedAt), Remote, nothingPushed)
}

// HandBack is why a drain passes over id in the view at now: a hand-back
// tombstone at its tip whose record origin's default branch still holds
// unchanged, whichever copy handed it back. "" when there is none.
func (s Snapshot) HandBack(id string, now time.Time) string {
	r, ok := s.Lookup(id)
	if !ok || r.HandedBack == "" || r.RecordChanged || r.ReleasedBy != ReleasedByTombstone {
		return ""
	}
	return fmt.Sprintf("%s was handed back by %s as %s at %s, and its record on %s's default branch is unchanged since, %s",
		id, r.Holder, r.HandedBack, stamp(r.ReleasedAt), Remote, s.AsOf(now))
}
