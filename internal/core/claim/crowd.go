package claim

import (
	"cmp"
	"fmt"
	"path"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/layered"
)

// MaxLive is the most live reservations one person may hold unless their
// machine's reservation.max_live says otherwise (maxlive.go), counted across
// every record in the view by whose holds are the person's own (Self.Owns):
// their holder id, the ids their home declares, and their login on a hold
// made before the id. A release subject is not counted.
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
	// Copy names the copy the other reservation was made from
	// (iss-2610290507526253), as yourCopyAt words it: "your worktree <path>
	// on this machine" or "your copy on another computer", or "your other
	// copy" when the worktrees could not be listed and it is in none listed,
	// with CopyUnread saying why.
	Copy       string
	CopyUnread string
}

// ownCopyIn is how an own overlap names the copy holding key, among the
// sibling worktrees sibs, whose listing failed for unread when it is not "":
// that copy's words, and why the worktrees could not be listed when that
// leaves the copy unknown.
func ownCopyIn(sibs map[string]siblingAt, unread, key string) (copy, whyUnread string) {
	if at := sibs[key].Path; at != "" || unread == "" {
		return yourCopyAt(at), ""
	}
	return "your other copy", unread
}

// GeneratedShare is a path a reservation shares with the live reservation of
// another record that is no overlap because origin's default branch declares
// it generated (iss-2610282340087862): the act is made, and whoever lands
// second regenerates the file after merging.
type GeneratedShare struct {
	// Path is the shared path, the deeper of the two (sharedPath).
	Path string
	// Record is the other reservation's record.
	Record string
}

// Line is how a person is told their reservation shares a generated file,
// the same words wherever it is shown.
func (g GeneratedShare) Line() string {
	return fmt.Sprintf("%s is also in the footprint of %s, but origin's default branch declares it generated, so it is no overlap; whichever lands second regenerates it after merging",
		echo(g.Path), g.Record)
}

// crowding refuses an act the rest of the view forbids, judged on the view
// just fetched: a footprint that shares a path with another live reservation
// that is not this copy's own (the back-off), and a reservation that would take its holder past the cap
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
// as it backs off another person's, except a renewal reviving a lapsed tip
// that names the files its work changed (Request.OwnOverlapOn;
// iss-2610261837062463, dcn-2610291352512058): it backs off that copy only
// where a path they share lies in one of those files, and goes ahead over
// any other, named in res as the warned act's are. res also names each
// generated file it shares, with the record it is shared with
// (Result.GeneratedShared, iss-2610282340087862), and why no generated file
// was recognised, when the overlap was judged without them
// (Result.GeneratedUnread).
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
	// person's cap nor is refused by it (release.go).
	if !isRelease(p.id) && addsOne(j) {
		limit, note, err := maxLive(p.home)
		if err != nil {
			return err
		}
		var mine []string
		for _, r := range others {
			if p.self.Owns(r.Holder) && !isRelease(r.Record) {
				mine = append(mine, r.Record)
			}
		}
		if len(mine) >= limit {
			return capRefusalIn(&snap, p.root, p.self, mine, limit, note)
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
	// The worktrees are listed only for an own overlap's words, once.
	var sibs map[string]siblingAt
	var sibsUnread string
	sibsListed := false
	ownCopy := func(r Reservation) (string, string) {
		if !sibsListed {
			sibs, sibsUnread = siblingCopies(p.root, ownCopyKey(p.root))
			sibsListed = true
		}
		return ownCopyIn(sibs, sibsUnread, r.key)
	}
	for _, r := range others {
		mine := p.self.Owns(r.Holder)
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
					if g, gen := generatedShare(want, held, snap.Generated); gen {
						s := GeneratedShare{Path: g, Record: r.Record}
						if !slices.Contains(res.GeneratedShared, s) {
							res.GeneratedShared = append(res.GeneratedShared, s)
						}
					}
				case mine && (p.req.WarnOwnOverlap || (p.req.Renew && p.req.OwnOverlapOn != nil && !touches(p.req.OwnOverlapOn, shared))):
					// This person's other copy: their own act, asked by hand,
					// goes ahead and is told; an automated one (a build's, a
					// drain's) backs off below, unless it revives a lapsed
					// reservation and the shared path is no file its work
					// changed, which that copy may go on holding.
					o := OwnOverlap{Path: shared, Record: r.Record, Expiry: r.EffectiveExpiry}
					o.Copy, o.CopyUnread = ownCopy(r)
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
	slices.SortFunc(res.GeneratedShared, func(a, b GeneratedShare) int {
		return cmp.Or(strings.Compare(a.Path, b.Path), strings.Compare(a.Record, b.Record))
	})
	return nil
}

// touches reports whether a path the work changed lies in shared, the path
// an overlap names, or shared lies in it: two automated lanes must never edit
// the same files.
func touches(changed []string, shared string) bool {
	for _, c := range changed {
		if _, ok := sharedPath(c, shared); ok {
			return true
		}
	}
	return false
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

// generatedShare is the path two footprint paths share that overlap lets
// through because it is a generated file, the act's to name
// (Result.GeneratedShared).
func generatedShare(want, held string, g Generated) (string, bool) {
	shared, ok := sharedPath(want, held)
	return shared, ok && g.Is(shared)
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
// own from another copy, naming that copy, the same words wherever it is
// shown.
func (o OwnOverlap) Warning() string {
	where := o.Copy
	if where == "" {
		where = "your other copy"
	}
	line := fmt.Sprintf("%s is also in the footprint of %s, which you hold from %s until %s; reserved anyway, since both are yours",
		echo(o.Path), o.Record, where, stamp(o.Expiry))
	if o.CopyUnread != "" {
		line += " (" + o.CopyUnread + ")"
	}
	return line
}

// addsOne reports whether an act on the judged tip adds a live reservation to
// its holder's count: everything but a renewal or a take-over of a tip this
// person already holds, readable, unreleased and live. One's own lapsed tip is
// not among the live ones the cap counts, so reviving it adds one.
func addsOne(j judged) bool {
	return !j.present || j.released() || j.Unreadable != "" || !j.mine || !j.verdict.Live
}

// CapError is the cap's refusal (spc-2610161044461448 step 6): an act that
// would take its holder past their cap (MaxLive, or reservation.max_live), naming the holder's live reservations
// as the view lists them. It is an ErrRefused refusal, so `abcd reserve`
// exits 2 on it as before; a drain reads it by its type and waits instead.
// Its advice is to wait for a pull request to merge: most of a drain's live
// reservations are lanes whose pull requests wait on one, and the merge ends
// the reservation, where releasing one would free an issue whose work is done.
type CapError struct {
	// Held are the holder's live reservations, the record ids sorted.
	Held []string
	// Where names the copy each held record was made from (copyPlace), so
	// the person can tell this machine's holds from the rest
	// (iss-2610231004140527).
	Where map[string]string
	// Limit is the cap they reached, the person's reservation.max_live
	// (maxlive.go); zero means MaxLive.
	Limit int
	// Note says why the machine's configuration was read past for the cap,
	// and why the worktrees could not be listed to name the copies; "" when
	// neither.
	Note string
}

// Error names the cap, the reservations and the advice, with how to raise
// the cap below its ceiling.
func (e *CapError) Error() string {
	limit := e.Limit
	if limit == 0 {
		limit = MaxLive
	}
	ids := strings.Join(e.Held, ", ")
	if len(e.Where) > 0 {
		ids = heldByPlace(e.Held, e.Where)
	}
	advice := "wait for one of their pull requests to merge, which ends its reservation, or release one whose work is abandoned (abcd reserve <id> --release)"
	if limit < MaxLiveCeiling {
		advice += fmt.Sprintf(", or raise %s in %s (a whole number from 1 to %d, %d when unset)",
			MaxLiveKey, layered.Config.MachineOrigin(), MaxLiveCeiling, MaxLive)
	}
	if e.Note != "" {
		advice += "; " + e.Note
	}
	if len(e.Held) == limit {
		return fmt.Sprintf("you hold %d live reservations, the most one person may hold: %s; %s; %s", len(e.Held), ids, advice, nothingPushed)
	}
	return fmt.Sprintf("you hold %d live reservations, past the %d one person may hold: %s; %s; %s", len(e.Held), limit, ids, advice, nothingPushed)
}

// Is makes the cap an ErrRefused refusal.
func (e *CapError) Is(target error) bool { return target == ErrRefused }

// heldByPlace is the held ids grouped by the copy each was made from: this
// copy first, then each sibling worktree by name, then elsewhere, each group's
// ids in the order held lists them.
func heldByPlace(held []string, where map[string]string) string {
	groups := map[string][]string{}
	for _, id := range held {
		w := where[id]
		if w == "" {
			w = "elsewhere"
		}
		groups[w] = append(groups[w], id)
	}
	places := make([]string, 0, len(groups))
	for w := range groups {
		places = append(places, w)
	}
	rank := func(w string) int {
		switch w {
		case "this copy":
			return 0
		case "elsewhere":
			return 2
		}
		return 1
	}
	sort.Slice(places, func(i, j int) bool {
		if ri, rj := rank(places[i]), rank(places[j]); ri != rj {
			return ri < rj
		}
		return places[i] < places[j]
	})
	parts := make([]string, len(places))
	for i, w := range places {
		parts[i] = w + ": " + strings.Join(groups[w], ", ")
	}
	return strings.Join(parts, "; ")
}

// capRefusalIn is the cap's refusal for self in the view s, naming the copy
// each of mine was made from by self's own live tip, never another
// holder's tip on the same record, and why the worktrees could not be listed
// when they could not (iss-2610272352179242). The worktrees are listed only
// now, for the refusal's words.
func capRefusalIn(s *Snapshot, root string, self Self, mine []string, limit int, note string) error {
	s.NameSiblings(root, self)
	where := map[string]string{}
	for _, r := range s.Reservations {
		if r.State == StateLive && self.Owns(r.Holder) && slices.Contains(mine, r.Record) {
			where[r.Record] = copyPlace(r)
		}
	}
	return capRefusal(mine, where, limit, joinNote(note, s.SiblingsUnread))
}

// capRefusal names the cap and the holder's live reservations, as the view
// lists them, the cap they reached, and note: why the machine's
// configuration was read past for it, or the worktrees could not be listed.
func capRefusal(mine []string, where map[string]string, limit int, note string) error {
	return &CapError{Held: slices.Clone(mine), Where: where, Limit: limit, Note: note}
}

// LiveHeldBy is the records self holds live in the view (Self.Owns), from any copy,
// sorted, less except and less any release hold, which the cap does not count: the count the cap judges a fresh reservation of except
// by (crowding), read from the view alone so a caller counts with no fetch.
func (s Snapshot) LiveHeldBy(self Self, except string) []string {
	var out []string
	for _, r := range s.Reservations {
		if r.State == StateLive && r.Record != except && self.Owns(r.Holder) && !isRelease(r.Record) {
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

// OtherCopyOverlap is the first path of footprint that holder's live
// reservation of a record other than id, made from another copy, shares in
// the view s, as crowding judges an overlap (a generated file is shared), and
// that reservation; ok is false when none does. It reads the view alone. A
// run whose lane leaves its landing for a fresh round judges it
// (iss-2610261837062463, dcn-2610300422481486): a lapsed revival may have
// gone ahead past that copy on a file no lane had changed
// (Request.OwnOverlapOn), which the fresh round's implementer may now edit,
// and two automated lanes must never edit one file.
func (s Snapshot) OtherCopyOverlap(id string, footprint []string, holder string) (OwnOverlap, bool) {
	for _, r := range s.Reservations {
		if r.State != StateLive || r.Record == id || r.Copy == CopyThis || !sameHolder(r.Holder, holder) {
			continue
		}
		for _, want := range footprint {
			for _, held := range r.Footprint {
				if shared, ok := overlap(want, held, s.Generated); ok {
					return OwnOverlap{Path: shared, Record: r.Record, Expiry: r.EffectiveExpiry}, true
				}
			}
		}
	}
	return OwnOverlap{}, false
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
//
// takeOver is whose holds are the person's own for an act with --take-over,
// the zero Self for any other act: that person's own live reservation of id
// from another copy does not hold it off, since the act takes it over (spc-2610161044461448 step 15).
func (s Snapshot) Contention(id string, footprint []string, now time.Time, takeOver Self) string {
	if r, ok := s.Lookup(id); ok && r.Copy != CopyThis && !(r.State == StateLive && takeOver.Owns(r.Holder)) {
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
