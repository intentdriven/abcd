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
// Restated is each the tip records settled (kept, re-armed, skipped) where
// the commit that moved it out of withdrawn or admitted was not pushed by its
// withdrawer, as it stands on the tip: the change counts for nothing, and it
// is split as the entry it changed, in restate.
type vouching struct {
	Due, Unrecorded, AppPushed, Unverified []Withdrawal
	Why                                    string
	Restated                               []Withdrawal
	restate                                map[int]Withdrawal
}

// actedOn is ws with each entry Restated put back as it stood before the
// change its withdrawer did not push: what an act settles and records.
func (v vouching) actedOn(ws []Withdrawal) []Withdrawal {
	out := slices.Clone(ws)
	for i, w := range out {
		if e, ok := v.restate[w.Number]; ok {
			out[i] = e
		}
	}
	return out
}

// pushCheck is where vouch asks the forge's record of pushes: the checkout,
// the release, and the tips the record must end at (readPushRecord).
type pushCheck struct {
	root, id string
	tips     []string
}

// asker is the forge's record of pushes to one release branch read from its
// tip (links), asked as c says only when a verdict is wanted and then once,
// beside the verdicts kept for the tip in the local tier (verdicts).
type asker struct {
	c     pushCheck
	links []chainLink
	rec   *pushRecord
	known *verdicts
}

func (k *asker) kept() *verdicts {
	if k.known == nil {
		tip := ""
		if len(k.links) > 0 {
			tip = k.links[0].commit
		}
		k.known = readVerdicts(k.c.root, k.c.id, tip)
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

// verdict is what the record says of commit and login, a verdict kept for
// the tip first. A verdict that never changes once given (another account's
// push, or an app's) is kept, so a withdrawal left that way costs one ask of
// the forge, not one on every act.
func (k *asker) verdict(commit, login string) pushVerdict {
	if v, ok := k.kept().of(commit, login); ok {
		return v
	}
	v := k.record().verdict(commit, login)
	k.kept().keep(commit, login, v)
	return v
}

// change is verdict for a commit that moved a withdrawal of login's out of
// withdrawn or admitted. Its writer's own word decides first: a commit whose
// holder is not login is another person's, whoever pushed it. One the record
// says login pushed is kept as standing for the tip, a verdict that only ever
// leaves the withdrawal as the tip records it.
func (k *asker) change(c chainLink, login string) pushVerdict {
	if !sameHolder(c.m.Holder, login) {
		return pushedByOther
	}
	if k.kept().standing(c.commit, login) {
		return pushedByThem
	}
	v := k.verdict(c.commit, login)
	if v == pushedByThem {
		k.kept().stand(c.commit, login)
	}
	return v
}

func (k *asker) save() {
	if k.known != nil {
		k.known.save()
	}
}

// changeOf finds where the withdrawal w, as the chain's tip records it, was
// moved out of withdrawn or admitted: the commits from the tip down carrying
// it as it stands, field for field and in its state, end at a commit (index
// j) whose entry for its number is still to settle (prev). ok is false when
// w is itself still to settle, or the walk finds no such commit beneath
// (another settled state, no entry, a commit that does not read, or the end
// of the walk). The change is links[j-1]'s.
func changeOf(links []chainLink, w Withdrawal) (j int, prev Withdrawal, ok bool) {
	if w.unsettled() || len(links) == 0 || !links[0].readable {
		return 0, Withdrawal{}, false
	}
	for j = 1; j < len(links); j++ {
		if !links[j].readable {
			return 0, Withdrawal{}, false
		}
		e, found := withdrawalOf(links[j].m.Withdrawn, w.Number)
		if found && sameEntry(e, w) && e.State == w.State {
			continue
		}
		if !found || !e.unsettled() {
			return 0, Withdrawal{}, false
		}
		return j, e, true
	}
	return 0, Withdrawal{}, false
}

// vouch is vouching for the withdrawals of ws that keep returns true for whose
// By is holder, over the chain read from the tip and the record of pushes
// placed on it, asked as c says only when a verdict is wanted and then once.
func vouch(links []chainLink, complete bool, ws []Withdrawal, holder string, c pushCheck, keep func(Withdrawal) bool) vouching {
	k := &asker{c: c, links: links}
	v := k.vouch(complete, ws, holder, keep)
	k.save()
	return v
}

// vouch is vouching for ws over k's chain. A change of state counts only when
// its withdrawer pushed it: an entry the tip records settled, where the
// commit that moved it out of withdrawn or admitted is not one the record
// says its withdrawer pushed, is Restated and judged as the entry it was
// before that commit, so a release re-arms it; one whose change the record
// does not settle yet is left as it stands, and a later act asks again.
func (k *asker) vouch(complete bool, ws []Withdrawal, holder string, keep func(Withdrawal) bool) vouching {
	var v vouching
	for _, w := range ws {
		from := 0
		if j, prev, ok := changeOf(k.links, w); ok && keep(prev) && sameHolder(prev.By, holder) {
			switch k.change(k.links[j-1], prev.By) {
			case pushedByThem:
			case pushNotVerified:
				continue
			default:
				named := w
				named.By = prev.By
				v.Restated = append(v.Restated, named)
				if v.restate == nil {
					v.restate = map[int]Withdrawal{}
				}
				v.restate[w.Number] = prev
				w, from = prev, j
			}
		}
		if !keep(w) || !sameHolder(w.By, holder) {
			continue
		}
		first, ok := firstRecord(k.links[from:], complete, w)
		if !ok {
			v.Unrecorded = append(v.Unrecorded, w)
			continue
		}
		switch k.verdict(first, w.By) {
		case pushedByThem:
			v.Due = append(v.Due, w)
		case pushedByOther:
			v.Unrecorded = append(v.Unrecorded, w)
		case pushedByApp:
			v.AppPushed = append(v.AppPushed, w)
		default:
			v.Unverified = append(v.Unverified, w)
			v.Why = k.record().reason()
		}
	}
	return v
}

// endedBy is whether the tombstones at the top of the chain end the release:
// the lowest of them, written over the reservation it ended, ends it only
// when it is that reservation's holder's and the forge's record says the
// holder pushed it, or when own, the person whose act this is, is that
// holder (their own --release over it). It returns the verdict, the holder,
// and why it is not verified when it is not. A tip that is no tombstone ends
// nothing by this reading.
func (k *asker) endedBy(own string) (verdict pushVerdict, holder, why string) {
	i := 0
	for i < len(k.links) && k.links[i].readable && k.links[i].m.released() {
		i++
	}
	switch {
	case i == 0:
		return pushedByOther, "", ""
	case i == len(k.links) || !k.links[i].readable:
		return pushNotVerified, "", "the reservation its tombstone ended is not within the commits read, or does not read"
	}
	holder = k.links[i].m.Holder
	tomb := k.links[i-1]
	switch {
	case own != "" && sameHolder(own, holder):
		return pushedByThem, holder, ""
	case !sameHolder(tomb.m.Holder, holder):
		return pushedByOther, holder, ""
	}
	verdict = k.verdict(tomb.commit, holder)
	if verdict == pushNotVerified {
		why = k.record().reason()
	}
	return verdict, holder, why
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
	// stands is each change of state (by the commit that made it and the
	// person) the record says its withdrawer pushed: read only for a change,
	// where it leaves the withdrawal as the tip records it, never for a first
	// record, where a verdict acts.
	stands  map[string]bool
	changed bool
}

// verdictWords are the kept verdicts as the file names them.
var verdictWords = map[string]pushVerdict{"other": pushedByOther, "app": pushedByApp}

// readVerdicts is the verdicts kept for id at tip; empty when none is kept,
// they were kept for another tip, or the file does not read.
func readVerdicts(root, id, tip string) *verdicts {
	k := &verdicts{root: root, id: id, tip: tip, by: map[string]pushVerdict{}, stands: map[string]bool{}}
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
		} else if f[2] == standsWord {
			k.stands[f[0]+" "+strings.ToLower(f[1])] = true
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

// standsWord is how the file names a change its withdrawer pushed.
const standsWord = "stands"

// standing reports whether a change commit made for login is kept as theirs.
func (k *verdicts) standing(commit, login string) bool {
	return k.stands[commit+" "+strings.ToLower(login)]
}

// stand keeps commit as a change of login's own.
func (k *verdicts) stand(commit, login string) {
	if !validUsername(login) {
		return
	}
	k.stands[commit+" "+strings.ToLower(login)] = true
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
	stood := make([]string, 0, len(k.stands))
	for key := range k.stands {
		stood = append(stood, key)
	}
	slices.Sort(stood)
	for _, key := range stood {
		b.WriteString(key + " " + standsWord + "\n")
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
