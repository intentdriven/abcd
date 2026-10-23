package claim

// withdrawpush.go holds a release branch's withdrawals to the forge's own
// record of who pushed each commit (iss-2610222014550692), the record the
// status branches are already held to (C17, checkSenders). A commit's
// Abcd-Holder trailer is whatever its writer put there, so before any re-arm
// (the tag sweep, the tombstone sweep, --release, --admit) the commit that
// first recorded a withdrawal must be one the forge's activity record says
// a person pushed, that person being the withdrawal's By; and before the
// launch check exempts an admitted merge, the commits that recorded the
// reservation and its admission must be ones the reader pushed. A record the
// forge has not caught up with, or did not answer in full, decides nothing:
// the withdrawal is reported as not yet verified and a later act asks again.
// An app's push vouches for no one, and is reported for its person to act on
// by hand. Reports (peers, the lost walk) read the chain alone.

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/core/github"
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
// the walk) may have carried any commit the walk read, so its pusher is placed
// on all of them, never on none. tips are the branch's tip as this act
// judged it, and the commit the act itself pushed over it when it pushed one
// first: the record's newest push must be one that moved the branch to one of
// them, or the record has not caught up (or the branch was moved since by a
// push it does not hold yet), and it settles nothing.
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

// vouch is vouching for the withdrawals of ws that keep returns true for whose
// By is holder, over the chain read from the tip and the record of pushes
// placed on it, which record reads, asked only when the chain vouches for
// one and then once.
func vouch(links []chainLink, complete bool, ws []Withdrawal, holder string, record func() pushRecord, keep func(Withdrawal) bool) vouching {
	var v vouching
	var rec *pushRecord
	for _, w := range ws {
		if !keep(w) || !sameHolder(w.By, holder) {
			continue
		}
		first, ok := firstRecord(links, complete, w)
		if !ok {
			v.Unrecorded = append(v.Unrecorded, w)
			continue
		}
		if rec == nil {
			r := record()
			rec = &r
		}
		switch rec.verdict(first, w.By) {
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
	return v
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
