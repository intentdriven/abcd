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
// resolve to one username are one person. A release subject is not counted.
const MaxLive = 10

// OwnOverlap is a path a person's own reservation shares with the live
// reservation of another record they hold from another copy.
type OwnOverlap struct {
	// Path is the shared path, the deeper of the two (sharedPath).
	Path string
	// Record is the other reservation's record.
	Record string
	// Expiry is when the other reservation stops holding.
	Expiry time.Time
}

// crowding refuses an act the rest of the view forbids, judged on the view
// just fetched: a footprint that shares a path with another live reservation
// that is not this copy's own (the back-off), and a reservation that would take its holder past MaxLive
// (the cap, checked first, since taking other work does not lift it). A
// renewal, and a take-over of one's own other copy, add no reservation, so the
// cap never refuses them. A reserve naming its footprint re-checks it, so a
// reservation is never widened past another; a renew-only act over a live
// tip, which keeps the tip's footprint, is not checked for overlap.
//
// A path is shared as overlap names it: a file origin's default branch
// declares generated is no overlap (spc-2610161044461448 step 11). An act
// that asks for it (Request.WarnOwnOverlap, the reserve verb's) and shares a
// path with the same person's reservation from another copy goes ahead, and
// res names each such path (Result.OwnOverlap); every other act backs off it,
// as it backs off another person's. res also carries
// why no generated file was recognised, when the overlap was judged without
// them (Result.GeneratedUnread).
//
// Both are judged on the view just fetched, not compared-and-swapped: each
// reservation is its own ref, so two people racing can each pass the cap or
// the overlap against a view without the other's. They are best-effort under
// a race; only a collision on one record is atomic (the plain push).
func crowding(p *prepared, j judged, res *Result) error {
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
	// A release subject sits outside the cap: it neither counts toward a
	// person's ten nor is refused by them (release.go).
	if !isRelease(p.id) && addsOne(j) {
		var mine []string
		for _, r := range others {
			if sameHolder(r.Holder, p.holder) && !isRelease(r.Record) {
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
	res.GeneratedUnread = snap.GeneratedUnread
	var own []OwnOverlap
	for _, r := range others {
		mine := sameHolder(r.Holder, p.holder)
		// This copy's own reservations on other records are no overlap: the
		// work they name is this copy's, as a drain's next issue beside its
		// own open lane.
		if r.Copy == CopyThis && mine {
			continue
		}
		for _, want := range res.Footprint {
			for _, held := range r.Footprint {
				shared, ok := overlap(want, held, snap.Generated)
				switch {
				case !ok:
				case mine && p.req.WarnOwnOverlap:
					// This person's other copy: their own act, asked by hand,
					// goes ahead and is told; an automated one (a build's, a
					// drain's) backs off below.
					o := OwnOverlap{Path: shared, Record: r.Record, Expiry: r.EffectiveExpiry}
					if !slices.Contains(own, o) {
						own = append(own, o)
					}
				default:
					return backOff("%s is in the footprint of %s, reserved by %s until %s%s; take other work; %s",
						echo(shared), r.Record, r.Holder, stamp(r.EffectiveExpiry), notRecognised(snap.GeneratedUnread), nothingPushed)
				}
			}
		}
	}
	res.OwnOverlap = own
	return nil
}

// overlap is the path two footprint paths share as an overlap: sharedPath's,
// unless that path, the deeper of the two, is a generated file. A directory
// is never generated, so two directory footprints that share still overlap.
// It is the one judgement crowding and Contention share.
func overlap(want, held string, g Generated) (string, bool) {
	shared, ok := sharedPath(want, held)
	if !ok || g.Is(shared) {
		return "", false
	}
	return shared, true
}

// notRecognised is the words a back-off adds when generated files were not
// recognised, "" when they were (or none are declared).
func notRecognised(why string) string {
	if why == "" {
		return ""
	}
	return " (generated files were not recognised: " + why + ")"
}

// GeneratedUnreadLine is how an act tells a person that no generated file was
// recognised, and why (Result.GeneratedUnread), the same words wherever it is
// shown.
func GeneratedUnreadLine(why string) string {
	return "not recognised: generated files (" + why + "), so no shared path was let through as one"
}

// Warning is how a person is told their reservation shares a path with their
// own from another copy, the same words wherever it is shown.
func (o OwnOverlap) Warning() string {
	return fmt.Sprintf("%s is also in the footprint of %s, which you hold from your other copy until %s; reserved anyway, since both are yours",
		echo(o.Path), o.Record, stamp(o.Expiry))
}

// addsOne reports whether an act on the judged tip adds a live reservation to
// its holder's count: everything but a renewal or a take-over of a tip this
// person already holds, readable, unreleased and live. One's own lapsed tip is
// not among the live ones the cap counts, so reviving it adds one.
func addsOne(j judged) bool {
	return !j.present || j.released() || j.Unreadable != "" || !j.mine || !j.verdict.Live
}

// capRefusal names the cap and the holder's live reservations, as the view
// lists them.
func capRefusal(mine []string) error {
	ids := strings.Join(mine, ", ")
	if len(mine) == MaxLive {
		return refuse("you hold %d live reservations, the most one person may hold: %s; release one (abcd reserve <id> --release) first; %s",
			len(mine), ids, nothingPushed)
	}
	return refuse("you hold %d live reservations, past the %d one person may hold: %s; release one (abcd reserve <id> --release) first; %s",
		len(mine), MaxLive, ids, nothingPushed)
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
// footprint shared with another copy's live reservation of another record
// (overlap: a generated file is shared, as crowding shares it), this person's
// other copy included, as a drain's reservation judges it. "" when nothing in
// the view holds it off; this copy's own reservations never do. It reads the view alone, so a caller passes over held work with no
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
				if shared, ok := overlap(want, held, s.Generated); ok {
					return fmt.Sprintf("%s is in the footprint of %s, reserved by %s until %s, %s%s",
						echo(shared), r.Record, r.Holder, stamp(r.EffectiveExpiry), s.AsOf(now), notRecognised(s.GeneratedUnread))
				}
			}
		}
	}
	return ""
}
