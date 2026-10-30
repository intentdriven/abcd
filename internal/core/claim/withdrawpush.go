package claim

// withdrawpush.go holds a release branch to the forge's own record of who
// pushed each commit (iss-2610222014550692, dcn-2610231033516293), the record
// the status branches are already held to (C17, checkSenders). A commit's
// trailers are whatever its writer put there, so a person's record on a
// release is the newest commit the forge's record of pushes shows that person
// pushed: their withdrawals, admissions and states, and whether their holding
// ended, are read from that commit alone. A commit above it that another login
// pushed is not read; it is counted, by pusher, and named once for each tip.
// When that newest push cannot be found or placed (the record does not read,
// was not read in full, does not reach the tip, has a push missing, an app
// pushed a commit above it, or it lies past the walk), abcd acts on nothing of
// that person's and names why. A release ends when its tag is on origin, or
// when the newest push of its holder (the person whose own push made the
// newest reservation on the branch) is a tombstone of theirs.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/github"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// reservationPushers is the forge's record of pushes to one reservation
// branch, asked by its full ref (refs/heads/abcd-reserved/<id>): the same
// activity record `abcd status` asks for the status branches (github.Pushers),
// answered with ErrPartialRecord beside the pushes when its page came back
// full.
func reservationPushers(ctx context.Context, root, ref string) ([]Push, error) {
	pushes, partial, err := github.Pushers(ctx, root, ref)
	if err != nil {
		return nil, err
	}
	out := make([]Push, 0, len(pushes))
	for _, p := range pushes {
		out = append(out, Push{Before: p.Before, After: p.After, Pusher: p.Login, App: p.App})
	}
	if partial {
		return out, ErrPartialRecord
	}
	return out, nil
}

// declaredTipNote is what the other-copy back-off adds when j's holder is
// an id the person only declares: whether the declaration pairs it with a
// login, and, when it does, whether the forge's record of pushes says that
// login pushed j's tip, so the person can tell a reservation that is not
// their other account's. doubtful is true when nothing ties the reservation
// to an account of theirs (the id is unpaired, or its paired account did not
// push it), and the back-off then says plainly what a take-over would do. It
// names neither the paired login nor the pusher: it is shown on this
// terminal alone, and says only whether they match.
func declaredTipNote(p *prepared, j judged) (note string, doubtful bool) {
	h := j.Holder
	if !p.self.declaredOnly(h) {
		return "", false
	}
	login, paired := p.self.pairedLogin(h)
	if !paired {
		return fmt.Sprintf(" (%s is an id %s declares as yours with no login beside it, so which account pushed this reservation is not checked)",
			h, DeclaredHolderIDsFile), true
	}
	pusher, why := tipPusher(p.root, p.id, j.Commit)
	switch {
	case why != "":
		return fmt.Sprintf(" (%s is an id %s pairs with an account's login, but whether that account pushed this reservation is not known: %s)",
			h, DeclaredHolderIDsFile, why), false
	case sameHolder(pusher, login):
		return fmt.Sprintf(" (%s is an id %s pairs with an account's login, and the forge's record of pushes says the account it is paired with pushed this reservation)",
			h, DeclaredHolderIDsFile), false
	}
	return fmt.Sprintf(" (%s is an id %s pairs with an account's login, but the forge's record of pushes says an account other than the one it is paired with pushed this reservation, so it is not your other account's)",
		h, DeclaredHolderIDsFile), true
}

// takeOverDoubtfulTail is how the other-copy back-off over a reservation no
// account of the person's is tied to ends: what --take-over would do, and
// why it is allowed; for a release (release true), what becomes of its hold
// and its withdrawals too.
func takeOverDoubtfulTail(id string, release bool) string {
	does := "it replaces that reservation with yours"
	if release {
		does += " and ends its hold, while its withdrawals are carried untouched and none is re-armed from here"
	}
	return fmt.Sprintf("--take-over is still allowed here, only because %s lists %s as yours: %s; if the id is not yours, remove it from %s and leave the reservation be",
		DeclaredHolderIDsFile, id, does, DeclaredHolderIDsFile)
}

// tipPusher is the login the forge's record of pushes to id's branch says
// pushed tip, or why it does not say: no forge, a record that does not read
// or has not caught up with tip, or an app's push, which names no person.
func tipPusher(root, id, tip string) (login, why string) {
	if releaseForge.pushers == nil {
		return "", "no record of who pushed each reservation commit is read on this path"
	}
	if _, _, err := releaseForge.repo(root); err != nil {
		return "", "origin names no repository on the forge whose record of pushes could be asked"
	}
	ctx, cancel := context.WithTimeout(context.Background(), pusherBudget)
	defer cancel()
	pushes, err := releaseForge.pushers(ctx, root, refName(id))
	switch {
	case err != nil && !errors.Is(err, ErrPartialRecord):
		return "", "the forge's record of pushes could not be read (" + termsafe.Sanitize(forgeReason(err)) + ")"
	case len(pushes) == 0 || pushes[0].After != tip:
		return "", "the forge's record of pushes has not caught up with the tip read here"
	case pushes[0].App || !validUsername(pushes[0].Pusher):
		return "", "the forge's record says an app pushed it, which names no person"
	}
	return pushes[0].Pusher, ""
}

// pushRecord is the forge's record of pushes to one release branch, placed
// on this checkout's commits: each commit's pushers, and why the record does
// not settle a commit it places nothing on ("" when it was read in full and
// reaches the tip judged).
type pushRecord struct {
	by  map[string][]Push
	why string
}

// readPushRecord asks the forge (releaseForge.pushers) who pushed each commit
// to id's release branch, and places each push on the commits it carried: the
// first-parent line from its After down to its Before, at most maxChain
// commits. A push whose Before is not on that line (a forced push, or one past
// the walk) is placed on every commit of the line from its After down, never
// on none; one whose After this checkout does not hold to place is placed on
// the whole line the walk read, and leaves the record not settling any commit
// it places nothing else on. tips are the branch's tip as this act judged it,
// and the commit the act itself pushed over it when it pushed one first: the
// record's newest push must be one that moved the branch to one of them, or
// the record has not caught up (or the branch was moved since by a push it
// does not hold yet), and it settles nothing. The pushes must also run
// unbroken, each one's Before the After of the one before it, down to the push
// that made the branch: the forge's activity record passes over an entry it
// holds without a pusher or a commit name, and a record with a push missing
// from it does not say who carried every commit, so it settles nothing either.
func readPushRecord(root, id string, links []chainLink, tips ...string) pushRecord {
	if releaseForge.pushers == nil {
		return pushRecord{why: "no record of who pushed each reservation commit is read on this path"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), pusherBudget)
	defer cancel()
	pushes, err := releaseForge.pushers(ctx, root, refName(id))
	partial := errors.Is(err, ErrPartialRecord)
	if err != nil && !partial {
		return pushRecord{why: "the forge's record of who pushed to " + id + "'s branch could not be read (" + termsafe.Sanitize(forgeReason(err)) + ")"}
	}
	rec := pushRecord{by: map[string][]Push{}}
	switch {
	case partial:
		rec.why = "the forge's record of who pushed to " + id + "'s branch was not read in full"
	case len(pushes) == 0 || !slices.Contains(tips, pushes[0].After):
		rec.why = "the forge's record of who pushed to " + id + "'s branch does not end at the tip read here (it has not caught up yet, or the branch moved since this checkout fetched it)"
	case !unbroken(pushes):
		rec.why = "the forge's record of who pushed to " + id + "'s branch has a push missing from it (one it holds without a pusher or a commit name is passed over), so it does not say who carried every commit"
	}
	line := make([]string, len(links))
	for i, l := range links {
		line[i] = l.commit
	}
	for _, p := range pushes {
		if !gitutil.IsFullSHA(p.After) {
			continue
		}
		carried := line
		if i := slices.Index(line, p.After); i >= 0 {
			carried = line[i:]
		} else {
			held, err := hasCommitCtx(ctx, root, p.After)
			if err != nil || !held {
				// A push this checkout cannot place may have carried any of
				// the line's commits.
				if rec.why == "" {
					rec.why = "a push the forge records to " + id + "'s branch is not held here to place"
				}
				carried = line
			} else if carried, err = firstParentsCtx(ctx, root, refName(id), p.After, maxChain); err != nil {
				return pushRecord{why: "the forge's record of who pushed to " + id + "'s branch was not placed (" + termsafe.Sanitize(redacted(err)) + ")"}
			}
		}
		if p.Before != zeroCommit {
			if i := slices.Index(carried, p.Before); i >= 0 {
				carried = carried[:i]
			}
		}
		for _, c := range carried {
			rec.by[c] = append(rec.by[c], p)
		}
	}
	return rec
}

// unbroken reports whether pushes, newest first and read in full, run without
// a gap: each push's Before is the After of the push before it, and the
// oldest made the branch (its Before is no commit). A push that made the
// branch anew after a deletion starts its run afresh, since the deletion is
// not among them.
func unbroken(pushes []Push) bool {
	for i, p := range pushes {
		switch {
		case p.Before == zeroCommit:
		case i+1 == len(pushes) || pushes[i+1].After != p.Before:
			return false
		}
	}
	return true
}

// reason is why the record leaves a commit not verified.
func (r pushRecord) reason() string {
	if r.why != "" {
		return r.why
	}
	return "the forge's record of pushes does not yet name the push that carried it"
}

// pushKind is what the record places on one commit: no push, pushes of one
// login, an app's push, or pushes of more than one account (a forced push
// placed over commits another pushed).
type pushKind int

const (
	pushedByNone pushKind = iota
	pushedByLogin
	pushedByApp
	pushedByMany
)

// placed is what the record places on commit, and the login when one login
// pushed it.
func (r pushRecord) placed(commit string) (pushKind, string) {
	ps := r.by[commit]
	if len(ps) == 0 {
		return pushedByNone, ""
	}
	if slices.ContainsFunc(ps, func(p Push) bool { return p.App }) {
		return pushedByApp, ""
	}
	login := ps[0].Pusher
	if !validUsername(login) {
		return pushedByNone, ""
	}
	for _, p := range ps[1:] {
		if !sameHolder(p.Pusher, login) {
			return pushedByMany, ""
		}
	}
	return pushedByLogin, login
}

// PushCount is how many of the commits above a person's newest push one
// other login pushed: commits that are not read for that person.
type PushCount struct {
	Pusher  string `json:"pusher"`
	Commits int    `json:"commits"`
}

// ownRecord is one person's record on a release branch: the newest commit the
// forge's record of pushes shows they pushed (found, at index at of the walk,
// with its message), the commits above it by other logins (above), and why it
// could not be found or placed (why, set: nothing of theirs is acted on).
// found false with why empty is a person who pushed nothing to the branch.
type ownRecord struct {
	found  bool
	at     int
	commit string
	m      message
	above  []PushCount
	why    string
	// first reports whether the reservation that first recorded an entry of
	// the push, held under its holder id, was pushed by the record's own
	// login (asker.firstPushedBy, walking down from the push); nil, as for a
	// record no asker read, vouches for none.
	first func(Withdrawal) bool
}

// firstOwn reports whether w, an entry of the record's push naming a holder
// id, was first recorded by a reservation held under that id that the
// record's own login pushed.
func (o ownRecord) firstOwn(w Withdrawal) bool { return o.first != nil && o.first(w) }

// mine is the withdrawals the record holds of self's own: those its own
// newest push carries naming self themselves as their withdrawer. One naming
// their kept holder id is theirs only when the reservation that first
// recorded it, held under that id, was pushed by their own login (firstOwn):
// a later push of theirs that carries it says nothing of who first recorded
// it, so an entry naming the id that another login first recorded, carried
// by a copy of theirs that reads the id as a declared one, is never read as
// theirs. One naming their login, a withdrawal made before the id, is theirs
// as their own newest push carries it, and is named by their kept id
// (Self.asKept), so no push of theirs carries the login. One naming an id
// self only declares is that other account's, and is read from the record
// its own account pushed (asker.vouched), never from self's.
func (o ownRecord) mine(self Self) []Withdrawal {
	if !o.found {
		return nil
	}
	var out []Withdrawal
	for _, w := range o.m.Withdrawn {
		if self.isOwn(w.By) && (!validHolderID(w.By) || o.firstOwn(w)) {
			out = append(out, w)
		}
	}
	return self.asKept(out)
}

// recorded is every withdrawal the record's own newest push carries under
// one of self's holder names, a declared id's included: what that push
// itself records, which the launch check's exemption of an admission reads.
// One naming self's kept id counts only as mine counts it.
func (o ownRecord) recorded(self Self) []Withdrawal {
	if !o.found {
		return nil
	}
	var out []Withdrawal
	for _, w := range o.m.Withdrawn {
		if self.Owns(w.By) && (w.By != self.ID || o.firstOwn(w)) {
			out = append(out, w)
		}
	}
	return out
}

// pushCheck is where an asker asks the forge's record of pushes: the
// checkout, the release, and the tips the record must end at (readPushRecord).
type pushCheck struct {
	root, id string
	tips     []string
}

// asker reads one release branch (links, newest first, complete when the walk
// reached its first commit) against the forge's record of pushes, asked once
// and only when a reading needs it.
type asker struct {
	c        pushCheck
	links    []chainLink
	complete bool
	rec      *pushRecord
	known    *tipCache
	// reading is true for an asker that only reads (peers, the launch
	// reading, the second-release refusal): it takes what the record placed
	// on a commit from the cache kept for the tip when it holds it. An act
	// that withdraws, admits or re-arms never does: it asks the record.
	reading bool
	// self is whose holder ids the asker can hold to the forge's record of
	// pushes (namesPusher); the zero Self, for a reading of another
	// person's release, holds none.
	self Self
}

// namesPusher reports whether a commit login pushed names that login's own
// person as its holder. A holder named by a GitHub username, as every hold
// made before the holder id is, names login when it is login. A holder id
// names no account the record of pushes could be matched against, so self's
// own kept id names login only when login is self's; an id self only
// declares (DeclaredHolderIDsFile) names login only when the declaration
// pairs it with login, and one declared with no login beside it only when
// login is self's own, as it is for the same person's other home under one
// login. Any other id is taken as naming whoever pushed it: the record of
// pushes, not the trailer, says who the person is, and a commit someone else
// pushes naming another's id can do no more than one naming themselves
// already could.
func (k *asker) namesPusher(holder, login string) bool {
	switch {
	case !validHolderID(holder):
		return sameHolder(holder, login)
	case holder == k.self.ID:
		return k.self.Login != "" && sameHolder(login, k.self.Login)
	case k.self.declaredOnly(holder):
		if paired, ok := k.self.pairedLogin(holder); ok {
			return sameHolder(login, paired)
		}
		return k.self.Login != "" && sameHolder(login, k.self.Login)
	}
	return true
}

// vouched is the withdrawals of ws still to settle, naming one of self's
// holder ids, that this copy acts on: each only when the reservation that
// first recorded it (firstRecord), held under that id, was pushed by the
// account the id is held to, since a later push that carries it says
// nothing of who first recorded it. One naming self's kept id is among them
// when own, self's own newest push, records it as theirs (ownRecord.mine),
// kept ones included, which a live renewal tries to withdraw again.
// One naming an id self pairs with a login (DeclaredHolderIDsFile) is among
// them when the newest push of that login to the branch, as the forge's
// record of pushes shows it, records it exactly as ws does, and that login
// pushed its first record. One naming an id declared with no login beside it
// is among them when self's own login pushed its first record: the same
// person's other home under one login. An entry whose login's push cannot be
// found or placed, or that its login did not first record, is not among
// them, and nothing acts on it here.
func (k *asker) vouched(ws []Withdrawal, own ownRecord) map[Withdrawal]bool {
	out := map[Withdrawal]bool{}
	recs := map[string]ownRecord{}
	mine := own.mine(k.self)
	for _, w := range ws {
		switch {
		case validHolderID(w.By) && w.By == k.self.ID:
			// A kept one, a withdrawal the forge refused, which a live
			// renewal tries again, is theirs on the same terms.
			if (w.unsettled() || w.State == WithdrawnKept) && slices.Contains(mine, w) {
				out[w] = true
			}
			continue
		case !w.unsettled():
			continue
		case !k.self.declaredOnly(w.By):
			continue
		}
		login, ok := k.self.pairedLogin(w.By)
		if !ok {
			if k.self.Login != "" && k.firstPushedBy(0, w, k.self.Login) {
				out[w] = true
			}
			continue
		}
		key := strings.ToLower(login)
		rec, seen := recs[key]
		if !seen {
			rec = k.newest(login)
			recs[key] = rec
		}
		if rec.found && rec.why == "" && slices.Contains(rec.m.Withdrawn, w) && rec.firstOwn(w) {
			out[w] = true
		}
	}
	return out
}

// firstPushedBy reports whether the reservation commit that first recorded
// w on the branch, as the walk down from links[from] reads it
// (firstRecordAs, with the reader's own login and kept id read alike), is
// one the forge's record of pushes says login pushed: false when the chain
// does not vouch for a first record, or the record places no one login's
// push on it.
func (k *asker) firstPushedBy(from int, w Withdrawal, login string) bool {
	if from < 0 || from >= len(k.links) {
		return false
	}
	commit, ok := firstRecordAs(k.links[from:], k.complete, w, k.self.alike)
	if !ok && w.State == WithdrawnKept {
		commit, ok = firstKeptRecordAs(k.links[from:], k.complete, w, k.self.alike)
	}
	if !ok {
		return false
	}
	i := slices.IndexFunc(k.links, func(l chainLink) bool { return l.commit == commit })
	if i < 0 {
		return false
	}
	kind, who, why := k.place(i)
	return why == "" && kind == pushedByLogin && sameHolder(who, login)
}

// actsOn is the gate every act that re-arms, withdraws again or admits a
// merge passes a withdrawal through: it acts only on one vouched names (each
// naming one of self's holder ids, held to its account's first record), and
// on one naming self's login, a withdrawal made before the id that their own
// newest push carries (merged), which a reader keeping no id yet still
// lists as theirs.
func actsOn(self Self, vouched map[Withdrawal]bool) func(Withdrawal) bool {
	return func(w Withdrawal) bool { return vouched[w] || !validHolderID(w.By) && self.isOwn(w.By) }
}

func (k *asker) cache() *tipCache {
	if k.known == nil {
		tip := ""
		if len(k.links) > 0 {
			tip = k.links[0].commit
		}
		k.known = readTipCache(k.c.root, k.c.id, tip)
	}
	return k.known
}

func (k *asker) record() *pushRecord {
	if k.rec == nil {
		r := readPushRecord(k.c.root, k.c.id, k.links, k.c.tips...)
		k.rec = &r
	}
	return k.rec
}

// place is what the record places on links[i], or why it places nothing it
// can be read by. A reading takes it from the cache first; what a record read
// in full places is kept for the tip.
func (k *asker) place(i int) (pushKind, string, string) {
	c := k.links[i].commit
	if k.reading {
		if kind, login, ok := k.cache().placed(c); ok {
			return kind, login, ""
		}
	}
	r := k.record()
	if r.why != "" {
		return pushedByNone, "", r.why
	}
	kind, login := r.placed(c)
	k.cache().place(c, kind, login)
	return kind, login, ""
}

// newest is login's record on the branch (ownRecord): the walk from the tip
// down passes over each commit one other login pushed, counting it, and stops
// at the first one login pushed. A commit an app pushed, or the record places
// no push or several accounts' pushes on, before it, leaves login's newest
// push not known; so does a walk that ends before finding it.
func (k *asker) newest(login string) (o ownRecord) {
	counts := map[string]int{}
	var order []string
	for i, l := range k.links {
		kind, who, why := k.place(i)
		switch {
		case why != "":
			o.why = why
			return o
		case kind == pushedByApp:
			o.why = "an app pushed commit " + l.commit[:12] + " of " + k.c.id + "'s branch, which says nothing of which person sent it, so " +
				login + "'s newest push there is not known"
			return o
		case kind == pushedByMany:
			o.why = "the forge's record places pushes of more than one account on commit " + l.commit[:12] + " of " + k.c.id + "'s branch, so " +
				login + "'s newest push there is not known"
			return o
		case kind == pushedByNone:
			o.why = "the forge's record of pushes places no push on commit " + l.commit[:12] + " of " + k.c.id + "'s branch, so " +
				login + "'s newest push there is not known"
			return o
		case sameHolder(who, login):
			if !l.readable {
				o.why = login + "'s newest push to " + k.c.id + "'s branch, commit " + l.commit[:12] + ", does not read"
				return o
			}
			o.found, o.at, o.commit, o.m = true, i, l.commit, l.m
			at := i
			o.first = func(w Withdrawal) bool { return k.firstPushedBy(at, w, login) }
			for _, p := range order {
				o.above = append(o.above, PushCount{Pusher: p, Commits: counts[strings.ToLower(p)]})
			}
			return o
		}
		if counts[strings.ToLower(who)] == 0 {
			order = append(order, who)
		}
		counts[strings.ToLower(who)]++
	}
	if !k.complete {
		o.why = login + "'s newest push to " + k.c.id + "'s branch is not within the " + fmt.Sprint(maxChain) + " commits read"
	}
	return o
}

// ended is whether the release has ended by its holder's own tombstone: the
// holder is the person whose own push made the newest reservation on the
// branch (a commit that reads, is not a tombstone, and names as its holder
// the person the record says pushed it, namesPusher), and the release has
// ended when that person's newest push is a tombstone naming them. It returns
// the holder as that reservation names them (a holder id, or the username a
// hold made before the id names) and their record, and why it could not tell (an app's push, or one the record
// does not place, above that reservation; the record not read; the walk ending
// first), which reads as not ended. A branch holding no reservation its holder
// pushed has not ended, and names no holder.
func (k *asker) ended() (ended bool, holder string, rec ownRecord, why string) {
	for i, l := range k.links {
		kind, who, why := k.place(i)
		switch {
		case why != "":
			return false, "", ownRecord{}, why
		case kind == pushedByApp:
			return false, "", ownRecord{}, "an app pushed commit " + l.commit[:12] + " of " + k.c.id + "'s branch, which says nothing of which person sent it, so whether the release has ended is not known"
		case kind != pushedByLogin:
			return false, "", ownRecord{}, "the forge's record of pushes does not place one account's push on commit " + l.commit[:12] + " of " + k.c.id + "'s branch, so whether the release has ended is not known"
		}
		if !l.readable || l.m.released() || !k.namesPusher(l.m.Holder, who) {
			continue
		}
		rec = k.newest(who)
		if rec.why != "" {
			return false, l.m.Holder, rec, rec.why
		}
		return rec.m.released() && k.namesPusher(rec.m.Holder, who), l.m.Holder, rec, ""
	}
	if !k.complete {
		return false, "", ownRecord{}, "no reservation its holder pushed is within the " + fmt.Sprint(maxChain) + " commits of " + k.c.id + "'s branch read, so whether the release has ended is not known"
	}
	return false, "", ownRecord{}, ""
}

func (k *asker) save() {
	if k.known != nil {
		k.known.save()
	}
}

// tipCacheDirRel is where what the record of pushes placed on each release
// tip's commits is kept, in the local tier, for the one tip it was read at,
// with the persons this copy found nothing more to do for there.
const tipCacheDirRel = stateRelDir + "/release-pushes"

// maxTipCacheBytes bounds what readTipCache reads.
const maxTipCacheBytes = 16 << 10

// tipCache is what this copy kept for one release tip: what a record read in
// full placed on each commit beneath it (read only by a reading, never by an
// act), and each person whose record there held nothing for this copy to do
// (done, which only ever lets an act do nothing). A lost or stale file costs
// an ask of the forge, never an act.
type tipCache struct {
	root, id, tip string
	kinds         map[string]pushKind
	logins        map[string]string
	done          map[string]bool
	failed        map[string]time.Time
	// endFailed is when this copy first failed to read whether the release
	// at the tip has ended (asCut).
	endFailed time.Time
	changed   bool
}

// tipCacheWords name the kinds the file keeps.
var tipCacheWords = map[string]pushKind{"app": pushedByApp, "many": pushedByMany}

// readTipCache is what was kept for id at tip; empty when nothing was, it was
// kept for another tip, or the file does not read.
func readTipCache(root, id, tip string) *tipCache {
	k := &tipCache{root: root, id: id, tip: tip, kinds: map[string]pushKind{}, logins: map[string]string{}, done: map[string]bool{}, failed: map[string]time.Time{}}
	if !isRelease(id) {
		return k
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return k
	}
	defer r.Close()
	data, err := fsutil.ReadGuardedInRoot(r, tipCacheDirRel+"/"+id, maxTipCacheBytes)
	if err != nil {
		return k
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || lines[0] != tip {
		return k
	}
	for _, l := range lines[1:] {
		f := strings.Fields(l)
		switch {
		case len(f) == 2 && f[0] == "done" && validHolder(f[1]):
			k.done[doneEntry(f[1], "")] = true
		case len(f) == 3 && f[0] == "done" && validHolder(f[1]) && doneScopeRe.MatchString(f[2]):
			k.done[doneEntry(f[1], f[2])] = true
		case len(f) == 2 && f[0] == "end-failed":
			if at, err := time.Parse(time.RFC3339, f[1]); err == nil {
				k.endFailed = at
			}
		case len(f) == 3 && f[0] == "failed" && validHolder(f[1]):
			if at, err := time.Parse(time.RFC3339, f[2]); err == nil {
				k.failed[strings.ToLower(f[1])] = at
			}
		case len(f) == 2 && gitutil.IsFullSHA(f[0]):
			if kind, ok := tipCacheWords[f[1]]; ok {
				k.kinds[f[0]] = kind
			} else if validUsername(f[1]) {
				k.kinds[f[0]], k.logins[f[0]] = pushedByLogin, f[1]
			}
		}
	}
	return k
}

// placed is what was kept as placed on commit.
func (k *tipCache) placed(commit string) (pushKind, string, bool) {
	kind, ok := k.kinds[commit]
	return kind, k.logins[commit], ok
}

// place keeps what a record read in full placed on commit.
func (k *tipCache) place(commit string, kind pushKind, login string) {
	if kind == pushedByNone || !gitutil.IsFullSHA(commit) {
		return
	}
	if old, ok := k.kinds[commit]; ok && old == kind && k.logins[commit] == login {
		return
	}
	k.kinds[commit], k.changed = kind, true
	if kind == pushedByLogin {
		k.logins[commit] = login
	}
}

// isDone reports whether holder's record at the tip was found to hold nothing
// more for this copy to do, read under scope (doneScope).
func (k *tipCache) isDone(holder, scope string) bool { return k.done[doneEntry(holder, scope)] }

// markDone keeps that holder's record at the tip holds nothing more to do,
// read under scope (doneScope).
func (k *tipCache) markDone(holder, scope string) {
	if !validHolder(holder) || scope != "" && !doneScopeRe.MatchString(scope) || k.isDone(holder, scope) {
		return
	}
	k.done[doneEntry(holder, scope)], k.changed = true, true
}

// doneEntry is how the cache keeps holder's done mark under scope.
func doneEntry(holder, scope string) string {
	return strings.TrimSpace(strings.ToLower(holder) + " " + scope)
}

// doneScopeRe is a scope as the cache keeps it.
var doneScopeRe = regexp.MustCompile(`^[0-9a-f]{16}$`)

// doneScope is what, beside the holder id, decides which withdrawals self
// acts on at a tip: their login, and the ids they declare with the login
// paired with each (DeclaredHolderIDsFile), as a digest, so a done mark kept
// before the person changed their declaration (paired an id, say) is not
// read after it, and the tip is read afresh. It is kept in the local tier
// alone and names no login.
func doneScope(self Self) string {
	ids := slices.Clone(self.Declared)
	slices.Sort(ids)
	h := sha256.New()
	fmt.Fprintf(h, "login %s\n", strings.ToLower(self.Login))
	for _, id := range ids {
		fmt.Fprintf(h, "%s %s\n", id, strings.ToLower(self.paired[id]))
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// markFailed keeps when this copy first failed to read holder's record at the
// tip.
func (k *tipCache) markFailed(holder string, now time.Time) {
	if _, ok := k.failed[strings.ToLower(holder)]; ok || !validHolder(holder) {
		return
	}
	k.failed[strings.ToLower(holder)], k.changed = now.UTC().Truncate(time.Second), true
}

// gaveUp reports whether this copy has failed to read holder's record at the
// tip for longer than lostWindow: an act then stops asking about it.
func (k *tipCache) gaveUp(holder string, now time.Time) bool {
	at, ok := k.failed[strings.ToLower(holder)]
	return ok && now.Sub(at) > lostWindow
}

// markEndFailed keeps when this copy first failed to read whether the
// release at the tip has ended.
func (k *tipCache) markEndFailed(now time.Time) {
	if !k.endFailed.IsZero() {
		return
	}
	k.endFailed, k.changed = now.UTC().Truncate(time.Second), true
}

// endGaveUp reports whether this copy has failed to read whether the release
// at the tip has ended for longer than lostWindow.
func (k *tipCache) endGaveUp(now time.Time) bool {
	return !k.endFailed.IsZero() && now.Sub(k.endFailed) > lostWindow
}

// save writes what was kept, best effort: a file lost costs only an ask.
func (k *tipCache) save() {
	if !k.changed || !isRelease(k.id) || !gitutil.IsFullSHA(k.tip) {
		return
	}
	var b strings.Builder
	b.WriteString(k.tip + "\n")
	commits := make([]string, 0, len(k.kinds))
	for c := range k.kinds {
		commits = append(commits, c)
	}
	slices.Sort(commits)
	for _, c := range commits {
		word := k.logins[c]
		for w, kind := range tipCacheWords {
			if kind == k.kinds[c] {
				word = w
			}
		}
		b.WriteString(c + " " + word + "\n")
	}
	logins := make([]string, 0, len(k.done))
	for l := range k.done {
		logins = append(logins, l)
	}
	slices.Sort(logins)
	for _, l := range logins {
		b.WriteString("done " + l + "\n")
	}
	failed := make([]string, 0, len(k.failed))
	for l := range k.failed {
		failed = append(failed, l)
	}
	slices.Sort(failed)
	for _, l := range failed {
		b.WriteString("failed " + l + " " + k.failed[l].Format(time.RFC3339) + "\n")
	}
	if !k.endFailed.IsZero() {
		b.WriteString("end-failed " + k.endFailed.Format(time.RFC3339) + "\n")
	}
	_ = writeUnflushedIn(k.root, tipCacheDirRel, k.id, []byte(b.String()))
}

// merged is the withdrawals a push of self's carries over base, the
// withdrawals of the record it is written over: every entry of another
// person's, every entry naming an id self only declares (another account or
// home of theirs, whose own push recorded it), and every entry naming self's
// kept id that self's own newest push does not record as theirs
// (ownRecord.mine: one another login first recorded), as base records it,
// each carried as another's, which nothing here acts on unless it is
// vouched; and self's own only as their own newest push (own) records them,
// named by their kept id (Self.asKept), so nothing another login pushed is
// carried as theirs and no push carries their login. An entry of base for a
// pull request self's own record also names is dropped for self's. An entry
// naming self's login with no push of self's login is not theirs to carry,
// and is dropped.
func merged(base []Withdrawal, own ownRecord, self Self) []Withdrawal {
	mine := own.mine(self)
	var out []Withdrawal
	for _, w := range base {
		switch {
		case !self.Owns(w.By):
			out = append(out, w)
		case self.declaredOnly(w.By) || validHolderID(w.By):
			if _, dup := withdrawalOf(mine, w.Number); !dup {
				out = append(out, w)
			}
		}
	}
	return append(out, mine...)
}

// unplacedLine is the sentence saying nothing of login's on subject was acted
// on, and why.
func unplacedLine(subject, login, why string) string {
	if why == "" {
		return ""
	}
	return "abcd acts on none of " + login + "'s withdrawals on " + subject + ": " + why +
		"; a person's record on a release is their newest push, as the forge's record of pushes shows it, and a later act asks again"
}

// notReadLine is the sentence naming the commits above login's newest push on
// subject that other logins pushed, which are not read.
func notReadLine(subject, login string, above []PushCount) string {
	if len(above) == 0 {
		return ""
	}
	n := 0
	parts := make([]string, 0, len(above))
	for _, a := range above {
		n += a.Commits
		parts = append(parts, fmt.Sprintf("%s (%d)", a.Pusher, a.Commits))
	}
	noun := "commits"
	if n == 1 {
		noun = "commit"
	}
	return fmt.Sprintf("%d %s above %s's newest push to %s's branch, pushed by %s, are not read: abcd reads %s's withdrawals only from %s's own newest push",
		n, noun, login, subject, strings.Join(parts, ", "), login, login)
}
