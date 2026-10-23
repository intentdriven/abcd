package claim

// withdrawchain.go reads a release branch's ancestry for what its tip alone
// cannot say about the withdrawals it records (iss-2610222014550692):
//
//   - whose a withdrawal is, for the report of lost withdrawals alone, which
//     reads the chain without the forge: one is counted only when the
//     reservation commit that first recorded it was itself held by that
//     person (vouched). An act never reads a withdrawal this way: it reads a
//     person's withdrawals only from their own newest push (asker.newest).
//   - which withdrawals a later commit lost. An older abcd ignores the
//     Abcd-Withdrawn trailer, so its renewal, take-over or tombstone rewrites
//     the tip without them; the walk finds each one still withdrawn on an
//     ancestor that no commit since carries, and names it lost
//     (dcn-2610230328242929).
//
// The walk is bounded: the first-parent line from the tip, at most maxChain
// commits, read in one rev-list and one batch of bodies. A withdrawal whose
// first record lies past the bound is not vouched for.

import (
	"strings"
	"time"
)

// maxChain bounds the walk of one release branch's ancestry. A cut writes a
// handful of commits (the reservation, its withdrawal record, an admission
// each, renewals, the tombstone and its settle), so a line this long is a
// writer's doing.
const maxChain = 64

// chainLink is one commit of a release branch's first-parent line, newest
// first: its name, and its message when it reads.
type chainLink struct {
	commit   string
	m        message
	readable bool
}

// releaseChain reads the first-parent line of id's release branch from the
// tip commit, newest first, at most maxChain commits. complete is false when
// the line runs past the bound. A commit that does not read is kept, with
// readable false: nothing past it is trusted.
func releaseChain(root, id, tip, tree string) (links []chainLink, complete bool, err error) {
	line, err := firstParents(root, refName(id), tip, maxChain)
	if err != nil {
		return nil, false, err
	}
	complete = len(line) <= maxChain
	if !complete {
		line = line[:maxChain]
	}
	bodies := commitBodies(root, line, batchBudget)
	for _, c := range line {
		b, ok := bodies[c]
		if !ok {
			links = append(links, chainLink{commit: c})
			continue
		}
		h, m := parseTip(c, []byte(strings.TrimRight(b, " \t\r\n")), tree, reservationParser(id))
		links = append(links, chainLink{commit: c, m: m, readable: h.Unreadable == ""})
	}
	return links, complete, nil
}

// sameEntry reports whether a and b record the same withdrawal: the same
// pull request, node id, form and head, withdrawn by the same person. Every
// field a forge call reads from the entry is compared, so a commit that
// carries an entry with any of them changed is that entry's own first
// record, held to the record of pushes as any first record is. Only the
// state is not compared, since a settle changes it. abcd's own correction
// after a partial withdrawal changes the form, and is then the corrected
// entry's first record: a reservation of the withdrawer's, pushed by them.
func sameEntry(a, b Withdrawal) bool {
	return a.Number == b.Number && a.ID == b.ID && a.Form == b.Form && a.Head == b.Head && sameHolder(a.By, b.By)
}

func entryIn(ws []Withdrawal, w Withdrawal) (Withdrawal, bool) {
	for _, x := range ws {
		if sameEntry(x, w) {
			return x, true
		}
	}
	return Withdrawal{}, false
}

// vouched reports whether w, as the chain's tip records it, was first
// recorded by a reservation its own By held (firstRecord), by the chain
// alone: what the report of lost withdrawals reads. An act reads a person's
// withdrawals from their own newest push instead (asker.newest).
func vouched(links []chainLink, complete bool, w Withdrawal) bool {
	_, ok := firstRecord(links, complete, w)
	return ok
}

// firstRecord is the commit that first recorded w, as the chain's tip records
// it, and whether the chain vouches for it: walking down from the tip while
// each commit reads and carries it still to settle (withdrawn or admitted),
// the last one that does must be a reservation (not a tombstone) whose holder
// is w.By, recording it withdrawn. A commit that carries it settled (re-armed,
// skipped, kept) ends the walk, since nothing settled is ever put back to
// withdrawn, so a commit over it carrying it withdrawn again is its own first
// record. A first record past the walk's bound, or beneath a commit that does
// not read, is not vouched for. A withdrawal the tip records kept (one the
// forge refused, which a live renewal tries again) is vouched for by the
// withdrawn record beneath the commits that carry it kept: the walk passes
// over those first, and nothing else.
func firstRecord(links []chainLink, complete bool, w Withdrawal) (string, bool) {
	first := -1
	kept := w.State == WithdrawnKept
	for i, l := range links {
		if !l.readable {
			return "", false
		}
		e, ok := entryIn(l.m.Withdrawn, w)
		if ok && kept {
			if e.State == WithdrawnKept {
				continue
			}
			kept = false
		}
		if !ok || !e.unsettled() {
			break
		}
		first = i
	}
	if first < 0 || first == len(links)-1 && !complete {
		return "", false
	}
	l := links[first]
	e, _ := entryIn(l.m.Withdrawn, w)
	if l.m.released() || !sameHolder(l.m.Holder, w.By) || e.State != WithdrawnPending {
		return "", false
	}
	return l.commit, true
}

// lostWithdrawals is each withdrawal still withdrawn on an ancestor of the
// chain's tip that the commit above it no longer carries for its number, in
// any state, and that the tip does not carry either: what a commit dropped
// without settling, as an older binary's renewal does. abcd's own correction
// after a refused withdrawal keeps the entry as kept, so it is not among them.
// Each is vouched for as vouched does, from the ancestor that held it; one
// that is not is left out, since nothing then says it was ever withdrawn.
func lostWithdrawals(links []chainLink, complete bool) []Withdrawal {
	if len(links) == 0 || !links[0].readable {
		return nil
	}
	var out []Withdrawal
	for i := 1; i < len(links); i++ {
		if !links[i].readable || !links[i-1].readable {
			break
		}
		for _, w := range links[i].m.Withdrawn {
			if !w.Pending() {
				continue
			}
			// Carried in any form counts: a re-withdrawal at a new head
			// replaces the entry for its number.
			if _, ok := withdrawalOf(links[i-1].m.Withdrawn, w.Number); ok {
				continue
			}
			if _, ok := withdrawalOf(links[0].m.Withdrawn, w.Number); ok {
				continue
			}
			if _, ok := entryIn(out, w); ok {
				continue
			}
			if vouched(links[i:], complete, w) {
				out = append(out, w)
			}
		}
	}
	return out
}

// lostWindow is how far back a listing looks for lost withdrawals: release
// tips claimed within it. Older ones are not walked, so a repository that
// keeps a release branch for every cut pays for the recent ones alone.
const lostWindow = 14 * 24 * time.Hour

// LostWithdrawals is one release's withdrawals that a later commit on its
// branch dropped (lostWithdrawals), as a listing and the launch check name
// them.
type LostWithdrawals struct {
	Subject string       `json:"subject"`
	Lost    []Withdrawal `json:"lost"`
}

// Line is the sentence naming them: "#11, #12 withdrawn for rel-v0.16.0 by
// octo are lost: a later commit on its branch dropped the record without
// settling them (...)", ending with whose hand re-arms them.
func (l LostWithdrawals) Line() string {
	by := map[string][]Withdrawal{}
	var order []string
	for _, w := range l.Lost {
		k := strings.ToLower(w.By)
		if _, ok := by[k]; !ok {
			order = append(order, k)
		}
		by[k] = append(by[k], w)
	}
	parts := make([]string, 0, len(order))
	for _, k := range order {
		h := by[k][0].By
		parts = append(parts, strings.Join(numbers(by[k]), ", ")+" withdrawn for "+l.Subject+" by "+h+
			" are lost: a later commit on its branch dropped the record without settling them (an older abcd's renewal does that), so nothing re-arms them; "+h+" re-arms them by hand")
	}
	return strings.Join(parts, "; ")
}

// lostOnReleases walks each release branch this checkout last fetched whose
// tip was claimed within lostWindow of now, and returns those with lost
// withdrawals, in ref order. A branch that cannot be walked is skipped.
func lostOnReleases(root, tree string, now time.Time) []LostWithdrawals {
	refs, err := localGit(root, 1<<20, "for-each-ref", "--format=%(refname)", TrackingPrefix+releasePrefix+"*")
	if err != nil {
		return nil
	}
	var listed []string
	for _, ref := range strings.Fields(refs) {
		if isRelease(strings.TrimPrefix(ref, TrackingPrefix)) {
			listed = append(listed, ref)
		}
	}
	reads := readSeenTips(root, listed, tree, now, false)
	var out []LostWithdrawals
	for i, ref := range listed {
		r := reads[i]
		if r.err != nil || !r.ok || r.t.Unreadable != "" || now.Sub(r.t.ClaimedAt) > lostWindow {
			continue
		}
		id := strings.TrimPrefix(ref, TrackingPrefix)
		links, complete, err := releaseChain(root, id, r.t.Commit, tree)
		if err != nil {
			continue
		}
		if lost := lostWithdrawals(links, complete); len(lost) > 0 {
			out = append(out, LostWithdrawals{Subject: id, Lost: lost})
		}
	}
	return out
}

// LostReleaseWithdrawals is lostOnReleases for the checkout at root, as last
// fetched: it fetches nothing and writes nothing.
func LostReleaseWithdrawals(root string, now time.Time) []LostWithdrawals {
	tree, err := emptyTree(root)
	if err != nil {
		return nil
	}
	return lostOnReleases(root, tree, now.UTC().Truncate(time.Second))
}
