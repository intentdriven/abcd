package claim

import (
	"path"
	"strings"
)

// MaxLive is the most live reservations one person may hold, counted by
// username across every record in the view: a laptop and a cloud session that
// resolve to one username are one person.
const MaxLive = 10

// crowding refuses an act the rest of the view forbids, judged on the view
// just fetched: a footprint that shares a path with another live reservation
// (the back-off), and a reservation that would take its holder past MaxLive
// (the cap, checked first, since taking other work does not lift it). A
// renewal, and a take-over of one's own other copy, add no reservation, so the
// cap never refuses them; both still re-check their footprint, so a
// reservation is never widened past another.
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
	for _, r := range others {
		for _, want := range res.Footprint {
			for _, held := range r.Footprint {
				if shared, ok := sharedPath(want, held); ok {
					return backOff("%s is in the footprint of %s, reserved by %s until %s; take other work; %s",
						shared, r.Record, r.Holder, stamp(r.EffectiveExpiry), nothingPushed)
				}
			}
		}
	}
	return nil
}

// addsOne reports whether an act on the judged tip adds a live reservation to
// its holder's count: everything but a renewal or a take-over of a tip this
// person already holds, readable and unreleased.
func addsOne(j judged) bool {
	return !j.present || j.released() || j.Unreadable != "" || !j.mine
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
