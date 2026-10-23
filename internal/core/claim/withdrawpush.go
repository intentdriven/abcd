package claim

// withdrawpush.go holds a release branch's withdrawals to the forge's own
// record of who pushed each commit (iss-2610222014550692), the record the
// status branches are already held to (C17, checkSenders). A commit's
// Abcd-Holder trailer is whatever its writer put there, so before any act on
// a withdrawal (the tag sweep, the tombstone sweep, --release, --admit, a live
// renewal's withdrawal again) the commit that
// first recorded a withdrawal must be one the forge's activity record says
// a person pushed, that person being the withdrawal's By; and before the
// launch check exempts an admitted merge, the commits that recorded the
// reservation and its admission must be ones the reader pushed. A record the
// forge has not caught up with, or did not answer in full, decides nothing:
// the withdrawal is reported as not yet verified and a later act asks again.
// An app's push vouches for no one, and is reported for its person to act on
// by hand. The peers listing splits the reader's own withdrawals the same
// way; the lost walk reads the chain alone.

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"

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

// pushVerdict is what the forge's record says of one commit and one person.
type pushVerdict int

const (
	// pushedByThem: every push the record places on the commit is that
	// person's own.
	pushedByThem pushVerdict = iota
	// pushNotVerified: the record could not be asked, was not read in full,
	// does not yet reach the tip judged, or places no push on the commit.
	pushNotVerified
	// pushedByApp: an app's push carried the commit, which says nothing of
	// which person sent it.
	pushedByApp
	// pushedByOther: another account's push carried the commit.
	pushedByOther
)

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

// verdict is what the record says of commit and login: another account's
// push on it decides, whatever else is unread; then a record that does not
// settle the commit, an app's push, and every push the person's own.
func (r pushRecord) verdict(commit, login string) pushVerdict {
	pushes := r.by[commit]
	if slices.ContainsFunc(pushes, func(p Push) bool { return !p.App && !sameHolder(p.Pusher, login) }) {
		return pushedByOther
	}
	switch {
	case r.why != "" || len(pushes) == 0:
		return pushNotVerified
	case slices.ContainsFunc(pushes, func(p Push) bool { return p.App }):
		return pushedByApp
	}
	return pushedByThem
}

// reason is why the record leaves a commit not verified.
func (r pushRecord) reason() string {
	if r.why != "" {
		return r.why
	}
	return "the forge's record of pushes does not yet name the push that carried it"
}

// vouching is a release tip's withdrawals of one person's split by what the
// branch and the forge's record of pushes say of each: those to act on (Due),
// and those not acted on: no reservation of theirs on the branch recorded it
// first, or another account pushed the commit that did (Unrecorded); an
// app's push carried that commit (AppPushed); or the record does not settle
// it yet (Unverified, with Why), which a later act asks about again.
type vouching struct {
	Due, Unrecorded, AppPushed, Unverified []Withdrawal
	Why                                    string
}

// pushCheck is where vouch asks the forge's record of pushes: the checkout,
// the release, and the tips the record must end at (readPushRecord).
type pushCheck struct {
	root, id string
	tips     []string
}

// vouch is vouching for the withdrawals of ws that keep returns true for whose
// By is holder, over the chain read from the tip and the record of pushes
// placed on it, asked as c says only when the chain vouches for one and then
// once. A verdict that never changes once given (another account's push, or
// an app's, carried the first record) is kept for the tip in the local tier
// (verdicts), so a withdrawal left that way costs one ask of the forge, not
// one on every act.
func vouch(links []chainLink, complete bool, ws []Withdrawal, holder string, c pushCheck, keep func(Withdrawal) bool) vouching {
	var v vouching
	var rec *pushRecord
	var known *verdicts
	for _, w := range ws {
		if !keep(w) || !sameHolder(w.By, holder) {
			continue
		}
		first, ok := firstRecord(links, complete, w)
		if !ok {
			v.Unrecorded = append(v.Unrecorded, w)
			continue
		}
		if known == nil {
			known = readVerdicts(c.root, c.id, links[0].commit)
		}
		verdict, ok := known.of(first, w.By)
		if !ok {
			if rec == nil {
				r := readPushRecord(c.root, c.id, links, c.tips...)
				rec = &r
			}
			verdict = rec.verdict(first, w.By)
			known.keep(first, w.By, verdict)
		}
		switch verdict {
		case pushedByThem:
			v.Due = append(v.Due, w)
		case pushedByOther:
			v.Unrecorded = append(v.Unrecorded, w)
		case pushedByApp:
			v.AppPushed = append(v.AppPushed, w)
		default:
			v.Unverified = append(v.Unverified, w)
			v.Why = rec.reason()
		}
	}
	if known != nil {
		known.save()
	}
	return v
}

// verdictsDirRel is where the verdicts that never change are kept for each
// release, in the local tier, for the one tip they were given at.
const verdictsDirRel = stateRelDir + "/withdrawal-verdicts"

// maxVerdictsBytes bounds what readVerdicts reads.
const maxVerdictsBytes = 16 << 10

// verdicts is the kept verdicts for one release tip: by the commit judged and
// the person, lower-cased, only those that never change (pushedByOther,
// pushedByApp). Only a verdict that leaves a withdrawal alone is kept, so a
// lost or stale file can cost an ask of the forge, and never an act.
type verdicts struct {
	root, id, tip string
	by            map[string]pushVerdict
	changed       bool
}

// verdictWords are the kept verdicts as the file names them.
var verdictWords = map[string]pushVerdict{"other": pushedByOther, "app": pushedByApp}

// readVerdicts is the verdicts kept for id at tip; empty when none is kept,
// they were kept for another tip, or the file does not read.
func readVerdicts(root, id, tip string) *verdicts {
	k := &verdicts{root: root, id: id, tip: tip, by: map[string]pushVerdict{}}
	if !isRelease(id) {
		return k
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return k
	}
	defer r.Close()
	data, err := fsutil.ReadGuardedInRoot(r, verdictsDirRel+"/"+id, maxVerdictsBytes)
	if err != nil {
		return k
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || lines[0] != tip {
		return k
	}
	for _, l := range lines[1:] {
		f := strings.Fields(l)
		if len(f) != 3 || !gitutil.IsFullSHA(f[0]) || !validUsername(f[1]) {
			continue
		}
		if v, ok := verdictWords[f[2]]; ok {
			k.by[f[0]+" "+strings.ToLower(f[1])] = v
		}
	}
	return k
}

// of is the verdict kept for commit and login.
func (k *verdicts) of(commit, login string) (pushVerdict, bool) {
	v, ok := k.by[commit+" "+strings.ToLower(login)]
	return v, ok
}

// keep keeps v for commit and login when it is one that never changes.
func (k *verdicts) keep(commit, login string, v pushVerdict) {
	if v != pushedByOther && v != pushedByApp || !validUsername(login) {
		return
	}
	k.by[commit+" "+strings.ToLower(login)] = v
	k.changed = true
}

// save writes the kept verdicts, best effort: a file lost costs only an ask.
func (k *verdicts) save() {
	if !k.changed || !isRelease(k.id) || !gitutil.IsFullSHA(k.tip) {
		return
	}
	keys := make([]string, 0, len(k.by))
	for key := range k.by {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	var b strings.Builder
	b.WriteString(k.tip + "\n")
	for _, key := range keys {
		for w, v := range verdictWords {
			if v == k.by[key] {
				b.WriteString(key + " " + w + "\n")
			}
		}
	}
	_ = writeUnflushedIn(k.root, verdictsDirRel, k.id, []byte(b.String()))
}

// appPushedLine is the sentence naming withdrawals an app's push recorded.
func appPushedLine(subject string, ws []Withdrawal) string {
	if len(ws) == 0 {
		return ""
	}
	return strings.Join(numbers(ws), ", ") + " on " + subject + " recorded as withdrawn by " + ws[0].By +
		", but an app pushed the commit that recorded them, and an app's push says nothing of which person sent it, so none of them is re-armed: " +
		ws[0].By + " re-arms each by hand"
}

// unverifiedLine is the sentence naming withdrawals the forge's record of
// pushes does not settle yet.
func unverifiedLine(subject string, ws []Withdrawal, why string) string {
	if len(ws) == 0 {
		return ""
	}
	return strings.Join(numbers(ws), ", ") + " on " + subject + " not yet verified: " + why +
		", and abcd re-arms a withdrawal only once the forge's record shows its withdrawer pushed the commit that recorded it; a later act asks again"
}
