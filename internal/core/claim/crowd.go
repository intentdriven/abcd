package claim

import (
	"fmt"
	"path"
	"slices"
	"strings"
	"time"
)

// MaxLive is the most live reservations one person may hold, counted by
// username across every record in the view: a laptop and a cloud session that
// resolve to one username are one person.
const MaxLive = 10

// crowding refuses an act the rest of the view forbids, judged on the view
// just fetched: a footprint that shares a path with another live reservation
// that is not this copy's own (the back-off), and a reservation that would take its holder past MaxLive
// (the cap, checked first, since taking other work does not lift it). A
// renewal, and a take-over of one's own other copy, add no reservation, so the
// cap never refuses them. A reserve naming its footprint re-checks it, so a
// reservation is never widened past another; a renew-only act over a live
// tip, which keeps the tip's footprint, is not checked for overlap.
//
// Both are judged on the view just fetched, not compared-and-swapped: each
// reservation is its own ref, so two people racing can each pass the cap or
// the overlap against a view without the other's. They are best-effort under
// a race; only a collision on one record is atomic (the plain push).
func crowding(p *prepared, j judged, res Result) error {
	snap, err := view(p.root, p.now, p.derivedGap == "")
	if err != nil {
		return refuse("could not read the reservations to count and compare: %s; %s", err, noReservation)
	}
	others := make([]Reservation, 0, len(snap.Reservations))
	for _, r := range snap.Reservations {
		if r.State == StateLive && r.Record != p.id {
			others = append(others, r)
		}
	}
	if addsOne(j) {
		var mine []string
		for _, r := range others {
			if sameHolder(r.Holder, p.holder) {
				mine = append(mine, r.Record)
			}
		}
		if len(mine) >= MaxLive {
			return capRefusal(mine)
		}
	}
	// A renew-only act over a live tip keeps the tip's own footprint, so it
	// widens nothing, and is not judged for overlap: an overlap a race let in
	// would otherwise stop both reservations renewing, and both lanes
	// landing, until one lapsed. One over a lapsed tip revives a reservation
	// that held nothing while it lapsed, so it is judged like a new one.
	if p.req.Renew && j.verdict.Live {
		return nil
	}
	for _, r := range others {
		// This copy's own reservations on other records are no overlap: the
		// work they name is this copy's, as a drain's next issue beside its
		// own open lane. This person's other copy still backs off.
		if r.Copy == CopyThis && sameHolder(r.Holder, p.holder) {
			continue
		}
		for _, want := range res.Footprint {
			for _, held := range r.Footprint {
				if shared, ok := sharedPath(want, held); ok {
					return backOff("%s is in the footprint of %s, reserved by %s until %s; take other work; %s",
						echo(shared), r.Record, r.Holder, stamp(r.EffectiveExpiry), nothingPushed)
				}
			}
		}
	}
	return nil
}

// addsOne reports whether an act on the judged tip adds a live reservation to
// its holder's count: everything but a renewal or a take-over of a tip this
// person already holds, readable, unreleased and live. One's own lapsed tip is
// not among the live ones the cap counts, so reviving it adds one.
func addsOne(j judged) bool {
	return !j.present || j.released() || j.Unreadable != "" || !j.mine || !j.verdict.Live
}

// CapError is the cap's refusal (spc-2610161044461448 step 6): an act that
// would take its holder past MaxLive, naming the holder's live reservations
// as the view lists them. It is an ErrRefused refusal, so `abcd reserve`
// exits 2 on it as before; a drain reads it by its type and waits instead.
// Its advice is to wait for a pull request to merge: most of a drain's live
// reservations are lanes whose pull requests wait on one, and the merge ends
// the reservation, where releasing one would free an issue whose work is done.
type CapError struct {
	// Held are the holder's live reservations, the record ids sorted.
	Held []string
}

// Error names the cap, the reservations and the advice.
func (e *CapError) Error() string {
	ids := strings.Join(e.Held, ", ")
	advice := "wait for one of their pull requests to merge, which ends its reservation, or release one whose work is abandoned (abcd reserve <id> --release)"
	if len(e.Held) == MaxLive {
		return fmt.Sprintf("you hold %d live reservations, the most one person may hold: %s; %s; %s", len(e.Held), ids, advice, nothingPushed)
	}
	return fmt.Sprintf("you hold %d live reservations, past the %d one person may hold: %s; %s; %s", len(e.Held), MaxLive, ids, advice, nothingPushed)
}

// Is makes the cap an ErrRefused refusal.
func (e *CapError) Is(target error) bool { return target == ErrRefused }

// capRefusal names the cap and the holder's live reservations, as the view
// lists them.
func capRefusal(mine []string) error {
	return &CapError{Held: slices.Clone(mine)}
}

// LiveHeldBy is the records holder holds live in the view, from any copy,
// sorted, less except: the count the cap judges a fresh reservation of except
// by (crowding), read from the view alone so a caller counts with no fetch.
func (s Snapshot) LiveHeldBy(holder, except string) []string {
	var out []string
	for _, r := range s.Reservations {
		if r.State == StateLive && r.Record != except && sameHolder(r.Holder, holder) {
			out = append(out, r.Record)
		}
	}
	slices.Sort(out)
	return out
}

// sharedPath reports whether two footprint paths are shared, equal or one a
// parent directory of the other, segment by segment, and names the deeper of
// the two, the path that lies in both footprints.
func sharedPath(a, b string) (string, bool) {
	a, b = path.Clean(a), path.Clean(b)
	if len(a) < len(b) {
		a, b = b, a
	}
	if a == b || strings.HasPrefix(a, b+"/") {
		return a, true
	}
	return "", false
}

// Contention is why a reservation of id with footprint would back off on the
// view s at now, as crowding and the record's own tip would judge it: id
// reserved by another copy (live, or unreadable within its hold), or a path of
// footprint shared with another copy's live reservation of another record. ""
// when nothing in the view holds it off; this copy's own reservations never
// do. It reads the view alone, so a caller passes over held work with no
// fetch and no push; the reservation act itself, whose push is the
// compare-and-swap, still judges what origin holds by then.
func (s Snapshot) Contention(id string, footprint []string, now time.Time) string {
	if r, ok := s.Lookup(id); ok && r.Copy != CopyThis {
		switch {
		case r.State == StateLive:
			return fmt.Sprintf("%s is reserved by %s until %s, %s", id, r.Holder, stamp(r.EffectiveExpiry), s.AsOf(now))
		case r.State == StateUnreadable && now.Before(r.EffectiveExpiry):
			return fmt.Sprintf("%s has an unreadable reservation holding it until %s, %s", id, stamp(r.EffectiveExpiry), s.AsOf(now))
		}
	}
	for _, r := range s.Reservations {
		if r.State != StateLive || r.Record == id || r.Copy == CopyThis {
			continue
		}
		for _, want := range footprint {
			for _, held := range r.Footprint {
				if shared, ok := sharedPath(want, held); ok {
					return fmt.Sprintf("%s is in the footprint of %s, reserved by %s until %s, %s",
						echo(shared), r.Record, r.Holder, stamp(r.EffectiveExpiry), s.AsOf(now))
				}
			}
		}
	}
	return ""
}
