package claim

// withdrawact.go is what the reservation acts do with the release hold's
// withdrawals (iss-2610222014550692):
//
//   - a release reserved anew (not a renewal of this copy's live one)
//     withdraws every merge armed or queued on origin's default branch, after
//     its own push has put the hold in place, and records each on a renewal
//     of that reservation (dcn-2610222219289525);
//   - --admit <pr> from the cutting copy re-arms one withdrawn pull request
//     in its recorded form and records it admitted;
//   - --release from the cutting copy re-arms every withdrawn, unadmitted
//     one on the tombstone that ends the reservation, and any act of a copy
//     of the same person, the cutting copy's included, does the same for a
//     release it sees ended (its tag on origin, its tombstone, or its lease
//     lapsed) with withdrawals of that person's not yet re-armed
//     (dcn-2610222219297651);
//   - every re-arm is in the form the pull request was taken from, pinned to
//     the head it had, and one whose head moved, or that was closed or
//     merged, is skipped and named (dcn-2610222219297487).
//
// A copy of another person never re-arms: it carries the withdrawals forward
// untouched and names them, with whose they are.

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/lease"
)

// ActAdmitted is an --admit that re-armed one withdrawn pull request and
// recorded it admitted.
const ActAdmitted Act = "admitted"

// The forge acts a ForgeFailure names.
const (
	ForgeWithdraw = "withdraw"
	ForgeRearm    = "re-arm"
	ForgeRecord   = "record"
)

// ForgeFailure is one pull request the forge, or the record of it, failed
// on: the act (ForgeWithdraw, ForgeRearm, ForgeRecord) and the reason.
type ForgeFailure struct {
	Number int    `json:"number"`
	Act    string `json:"act"`
	Reason string `json:"reason"`
}

// Line is the failure as a surface names it.
func (f ForgeFailure) Line() string {
	switch f.Act {
	case ForgeWithdraw:
		return fmt.Sprintf("#%d was not withdrawn: %s; it may still merge, and the launch check refuses the cut while it is armed", f.Number, f.Reason)
	case ForgeRecord:
		return fmt.Sprintf("#%d: %s", f.Number, f.Reason)
	}
	return fmt.Sprintf("#%d was not re-armed: %s; it stays withdrawn, and the next act of a copy of its holder tries again", f.Number, f.Reason)
}

// Settle is what an act did for one release's withdrawals that were due:
// those it re-armed or skipped (Settled, each in its new state), the
// failures, and the tombstone that recorded it (Commit), or why none was
// pushed (Failed).
type Settle struct {
	Subject  string         `json:"subject"`
	Commit   string         `json:"commit,omitempty"`
	Settled  []Withdrawal   `json:"settled"`
	Failures []ForgeFailure `json:"failures,omitempty"`
	Failed   string         `json:"failed,omitempty"`
}

// SettledLine is the sentence naming one settled withdrawal.
func SettledLine(subject string, w Withdrawal) string {
	switch w.State {
	case WithdrawnRearmed:
		return fmt.Sprintf("#%d re-armed (%s, at %s) after %s", w.Number, w.Form, w.Head[:12], subject)
	case WithdrawnFoundArmed:
		return fmt.Sprintf("#%d was already armed or queued again after %s, which counts as re-armed", w.Number, subject)
	case WithdrawnMoved:
		return fmt.Sprintf("#%d was not re-armed after %s: its head moved since it was withdrawn at %s, so its owner re-arms it", w.Number, subject, w.Head[:12])
	case WithdrawnClosed:
		return fmt.Sprintf("#%d was not re-armed after %s: it was closed", w.Number, subject)
	case WithdrawnMerged:
		return fmt.Sprintf("#%d was not re-armed after %s: it was merged", w.Number, subject)
	case WithdrawnAdmitted:
		return fmt.Sprintf("#%d admitted into %s", w.Number, subject)
	}
	return fmt.Sprintf("#%d withdrawn for %s", w.Number, subject)
}

// StrandedLine names the withdrawals of another person's that a release
// holds and this copy does not re-arm: "#12, #34 withdrawn by octo for
// rel-v0.16.0 stay withdrawn: only a copy of octo's re-arms them".
func StrandedLine(subject string, ws []Withdrawal) string {
	by := map[string][]Withdrawal{}
	var order []string
	for _, w := range ws {
		k := strings.ToLower(w.By)
		if _, ok := by[k]; !ok {
			order = append(order, k)
		}
		by[k] = append(by[k], w)
	}
	parts := make([]string, 0, len(order))
	for _, k := range order {
		h := by[k][0].By
		parts = append(parts, fmt.Sprintf("%s withdrawn by %s for %s stay withdrawn: only a copy of %s's re-arms them",
			strings.Join(numbers(by[k]), ", "), h, subject, h))
	}
	return strings.Join(parts, "; ")
}

// actor is who an act's commits are written as: the person, their holder,
// this copy's key, the clock and the Assisted-by label.
type actor struct {
	root   string
	who    identity
	holder string
	copy   string
	now    time.Time
	label  string
}

func (p *prepared) actor() actor {
	return actor{root: p.root, who: p.who, holder: p.holder, copy: p.copy, now: p.now, label: p.label}
}

// tombstoneFits reports whether a release's tombstone carrying ws, each in
// the longest state, would be written: under the cap and past the outbound
// policy. A reservation records no withdrawal its own release could not
// carry.
func (p *prepared) tombstoneFits(ws []Withdrawal) bool {
	m := p.tombstone(nil)
	for _, w := range ws {
		w.State = WithdrawnPending
		m.Withdrawn = append(m.Withdrawn, w)
	}
	return checkMessage(p.root, m.text()) == nil
}

// withdrawArmed lists the merges armed or queued on origin's default branch
// and withdraws each one not already among have, while the reservation can
// record it, as by. It returns the withdrawals made (a partial one among
// them), the failures, and why the listing could not be made at all, ""
// when it was.
func (p *prepared) withdrawArmed(have []Withdrawal) (made []Withdrawal, fails []ForgeFailure, unchecked string) {
	owner, name, err := releaseForge.repo(p.root)
	if err != nil {
		return nil, nil, "could not withdraw the merges already armed: " + err.Error()
	}
	base := strings.TrimPrefix(originDefault(p.root), "refs/remotes/"+Remote+"/")
	if base == "" {
		return nil, nil, "could not withdraw the merges already armed: this checkout names no default branch of " + Remote
	}
	armed, err := releaseForge.armedPullRequests(p.root, owner, name, base)
	if err != nil {
		return nil, nil, "could not list the merges armed or queued on " + base + ": " + forgeReason(err) +
			"; none was withdrawn, and the launch check refuses the cut while any is armed"
	}
	all := slices.Clone(have)
	for _, pr := range armed {
		w := Withdrawal{Number: pr.Number, ID: pr.ID, Form: withdrawalFormOf(pr.Queued, pr.Method), Head: pr.Head, By: p.holder, State: WithdrawnPending}
		if !p.tombstoneFits(withWithdrawal(all, w)) {
			fails = append(fails, ForgeFailure{Number: pr.Number, Act: ForgeWithdraw,
				Reason: fmt.Sprintf("the reservation has no room left to record it under the %d-byte cap", MaxMessageBytes)})
			continue
		}
		got, done, err := releaseForge.withdraw(p.root, pr, p.holder)
		if err != nil {
			fails = append(fails, ForgeFailure{Number: pr.Number, Act: ForgeWithdraw, Reason: err.Error()})
		}
		if done {
			made = append(made, got)
			all = withWithdrawal(all, got)
		}
	}
	return made, fails, ""
}

// rearmMine re-arms each withdrawal in ws still withdrawn whose holder is
// a's, and returns ws with their new states, the ones it settled and the
// failures. why is set, and nothing is re-armed, when the forge cannot be
// named.
func rearmMine(a actor, ws []Withdrawal) (out, settled []Withdrawal, fails []ForgeFailure, why string) {
	out = slices.Clone(ws)
	due := slices.ContainsFunc(ws, func(w Withdrawal) bool { return w.Pending() && sameHolder(w.By, a.holder) })
	if !due {
		return out, nil, nil, ""
	}
	owner, name, err := releaseForge.repo(a.root)
	if err != nil {
		return out, nil, nil, "could not re-arm the withdrawn merges: " + err.Error()
	}
	for i, w := range out {
		if !w.Pending() || !sameHolder(w.By, a.holder) {
			continue
		}
		state, err := releaseForge.rearm(a.root, owner, name, w)
		if err != nil {
			fails = append(fails, ForgeFailure{Number: w.Number, Act: ForgeRearm, Reason: err.Error()})
			continue
		}
		out[i].State = state
		settled = append(settled, out[i])
	}
	return out, settled, fails, ""
}

// stranded is the withdrawals in ws still withdrawn that are not holder's.
func stranded(ws []Withdrawal, holder string) []Withdrawal {
	var out []Withdrawal
	for _, w := range ws {
		if w.Pending() && !sameHolder(w.By, holder) {
			out = append(out, w)
		}
	}
	return out
}

// releaseEnded reports whether a release tip holds nothing any more: its tag
// on origin, a tombstone, or its lease lapsed past the take-over margin.
func releaseEnded(t tip, tagged bool, now time.Time) bool {
	if tagged || t.message.released() {
		return true
	}
	v := lease.Judge(lease.Stamp{ClaimedAt: t.ClaimedAt, ExpiresAt: t.ExpiresAt, FirstSeen: t.FirstSeen}, lease.Default, now)
	return !now.Before(v.EffectiveExpiry.Add(TakeoverMargin))
}

// settleEnded settles id's withdrawals of a's holder when its release has
// ended: re-arms each, then pushes a tombstone over the tip recording every
// withdrawal in its new state, so no copy acts twice. ok is false when there
// was nothing due. A push that fails leaves what was re-armed armed: the next
// act finds it armed and counts it done.
func settleEnded(a actor, tree, id string) (s Settle, ok bool) {
	t, present, err := readSeen(a.root, id, trackingName(id), tree, a.now, true)
	if err != nil || !present || t.Unreadable != "" {
		return Settle{}, false
	}
	if !slices.ContainsFunc(t.Withdrawn, func(w Withdrawal) bool { return w.Pending() && sameHolder(w.By, a.holder) }) {
		return Settle{}, false
	}
	tagged, err := taggedOnOrigin(a.root, id)
	if err != nil || !releaseEnded(t, tagged, a.now) {
		return Settle{}, false
	}
	s = Settle{Subject: id, Settled: []Withdrawal{}}
	ws, settled, fails, why := rearmMine(a, t.Withdrawn)
	s.Settled, s.Failures, s.Failed = append(s.Settled, settled...), fails, why
	if len(settled) == 0 {
		return s, true
	}
	m := message{Record: id, Holder: a.holder, Copy: a.copy, ClaimedAt: a.now, ExpiresAt: a.now, ReleasedAt: a.now,
		Withdrawn: ws, AssistedBy: a.label}
	sha, err := writeEmptyCommit(a.root, a.who, a.now, m, tree, t.Commit)
	if err == nil {
		err = push(a.root, pushArgs(sha, id))
	}
	if err != nil {
		s.Failed = fmt.Sprintf("re-armed %s, but the tombstone recording it was not pushed (%s); the next act finds them armed and records them",
			strings.Join(numbers(settled), ", "), redacted(err))
		return s, true
	}
	s.Commit = sha
	return s, true
}

// sweepEnded settles every ended release subject but skip whose withdrawals
// a's holder has not yet re-armed (settleEnded), as the reservations were
// just fetched. pushed reports whether a tombstone reached origin, so the
// caller fetches again before judging.
func sweepEnded(a actor, tree, skip string) (out []Settle, pushed bool) {
	refs, err := localGit(a.root, 1<<20, "for-each-ref", "--format=%(refname)", TrackingPrefix+releasePrefix+"*")
	if err != nil {
		return nil, false
	}
	for _, ref := range strings.Fields(refs) {
		id := strings.TrimPrefix(ref, TrackingPrefix)
		if !isRelease(id) || id == skip {
			continue
		}
		if s, ok := settleEnded(a, tree, id); ok {
			out = append(out, s)
			pushed = pushed || s.Commit != ""
		}
	}
	return out, pushed
}

// SettleEndedReleases is the fallback re-arm a listing makes (`abcd peers`),
// after its own fetch: as this checkout's person, every ended release's
// withdrawals of theirs not yet re-armed are re-armed and recorded on a
// tombstone (dcn-2610222219297651). It asks the forge nothing and pushes
// nothing when no such withdrawal is there, and acts only when this copy's
// person and key are known without asking the network.
func SettleEndedReleases(root string, o Options) ([]Settle, error) {
	refs, err := localGit(root, 1<<20, "for-each-ref", "--format=%(refname)", TrackingPrefix+releasePrefix+"*")
	if err != nil || strings.TrimSpace(refs) == "" {
		return nil, nil
	}
	who, err := gitIdentity(root)
	if err != nil {
		return nil, nil
	}
	h, err := resolveHolder(root, who, o, false)
	if err != nil || h.Login == "" {
		return nil, nil
	}
	if !who.person() {
		if h.NoReply == "" {
			return nil, nil
		}
		who = identity{Name: h.Login, Email: h.NoReply}
	}
	key := ownCopyKey(root)
	if key == "" {
		return nil, nil
	}
	tree, err := emptyTree(root)
	if err != nil {
		return nil, err
	}
	settles, _ := sweepEnded(actor{root: root, who: who, holder: h.Login, copy: key, now: o.now(), label: o.assistedBy()}, tree, "")
	return settles, nil
}

// admitOne re-arms the withdrawal for pull request n on m, the renewal an
// --admit pushes, and marks it admitted there; the refusal says why it was
// not, and nothing is pushed then.
func (p *prepared) admitOne(m *message, n int) error {
	i := slices.IndexFunc(m.Withdrawn, func(w Withdrawal) bool { return w.Number == n })
	w := m.Withdrawn[i]
	owner, name, err := releaseForge.repo(p.root)
	if err != nil {
		return refuse("#%d was not admitted: %s; %s", n, err.Error(), nothingPushed)
	}
	state, err := releaseForge.rearm(p.root, owner, name, w)
	switch {
	case err != nil:
		return refuse("#%d was not admitted: the forge refused its re-arm (%s), so it stays withdrawn; %s", n, err.Error(), nothingPushed)
	case state == WithdrawnMoved:
		return refuse("#%d was not admitted: its head moved since it was withdrawn at %s, and an admission re-arms only the commits withdrawn; "+
			"it stays withdrawn, for its owner to re-arm after the release; %s", n, w.Head[:12], nothingPushed)
	case state == WithdrawnClosed || state == WithdrawnMerged:
		return refuse("#%d was not admitted: it was %s since it was withdrawn; %s", n, state, nothingPushed)
	}
	m.Withdrawn[i].State = WithdrawnAdmitted
	return nil
}

// recordWithdrawals pushes, over sha (the reservation just pushed), the
// renewal recording made beside the reservation's own withdrawals. When the
// push fails, every one made is re-armed again, so no merge is left
// withdrawn with nothing pointing at it; failures name what was not.
func (p *prepared) recordWithdrawals(tree, sha string, m message, made []Withdrawal) (string, []ForgeFailure) {
	for _, w := range made {
		m.Withdrawn = withWithdrawal(m.Withdrawn, w)
	}
	next, err := writeCommit(p, m, tree, sha)
	if err == nil {
		err = push(p.root, pushArgs(next, p.id))
	}
	if err == nil {
		return next, nil
	}
	reason := redacted(err)
	var fails []ForgeFailure
	_, settled, rfails, why := rearmMine(p.actor(), made)
	for _, w := range settled {
		fails = append(fails, ForgeFailure{Number: w.Number, Act: ForgeRecord,
			Reason: "withdrawn, but the reservation recording it was not pushed (" + reason + "), so it was put back as it was"})
	}
	for _, f := range rfails {
		f.Act, f.Reason = ForgeRecord, "withdrawn, but the reservation recording it was not pushed ("+reason+"), and putting it back failed ("+f.Reason+"): re-arm it by hand"
		fails = append(fails, f)
	}
	if why != "" {
		for _, w := range made {
			fails = append(fails, ForgeFailure{Number: w.Number, Act: ForgeRecord,
				Reason: "withdrawn, but the reservation recording it was not pushed (" + reason + "), and it could not be put back (" + why + "): re-arm it by hand"})
		}
	}
	return sha, fails
}

// validPRNumber reports whether n is a pull request number a trailer may
// carry.
func validPRNumber(n int) bool { return n > 0 && n <= 999999999 }

// ReleaseWithdrawals is one release's withdrawals as a listing names them to
// a reader: on a release still held, those it holds withdrawn and those it
// admitted; on one ended, those still to re-arm, split into the reader's own
// (Due, which the reader's next act re-arms) and another person's
// (Stranded, which only that person re-arms), and those its last tombstone
// skipped for their owners (moved, closed or merged).
type ReleaseWithdrawals struct {
	Subject  string       `json:"subject"`
	Ended    bool         `json:"ended"`
	Held     []Withdrawal `json:"held,omitempty"`
	Admitted []Withdrawal `json:"admitted,omitempty"`
	Due      []Withdrawal `json:"due,omitempty"`
	Stranded []Withdrawal `json:"stranded,omitempty"`
	Skipped  []Withdrawal `json:"skipped,omitempty"`
}

// Lines is each sentence the listing shows for the release's withdrawals.
func (r ReleaseWithdrawals) Lines() []string {
	var out []string
	if len(r.Held) > 0 {
		out = append(out, fmt.Sprintf("%s holds %s withdrawn until it ends: the cutting copy admits one with --admit <pr>",
			r.Subject, strings.Join(numbers(r.Held), ", ")))
	}
	if len(r.Admitted) > 0 {
		out = append(out, fmt.Sprintf("%s admitted %s", r.Subject, strings.Join(numbers(r.Admitted), ", ")))
	}
	if len(r.Due) > 0 {
		out = append(out, fmt.Sprintf("%s has ended, and %s you withdrew for it are not re-armed yet: your next abcd act that fetches the reservations re-arms them",
			r.Subject, strings.Join(numbers(r.Due), ", ")))
	}
	if len(r.Stranded) > 0 {
		out = append(out, StrandedLine(r.Subject, r.Stranded))
	}
	for _, w := range r.Skipped {
		out = append(out, SettledLine(r.Subject, w))
	}
	return out
}

// ReleaseWithdrawals is every release in the view at now whose tip records a
// withdrawal there is something to say about, for reader (the listing's own
// GitHub username, "" when it is not known, which owns none).
func (s Snapshot) ReleaseWithdrawals(reader string, now time.Time) []ReleaseWithdrawals {
	var out []ReleaseWithdrawals
	for _, r := range s.Reservations {
		if !isRelease(r.Record) || len(r.Withdrawn) == 0 || r.State == StateUnreadable {
			continue
		}
		rw := ReleaseWithdrawals{Subject: r.Record,
			Ended: r.State == StateReleased || r.State == StateExpired && !now.Before(r.EffectiveExpiry.Add(TakeoverMargin))}
		for _, w := range r.Withdrawn {
			switch {
			case w.State == WithdrawnMoved || w.State == WithdrawnClosed || w.State == WithdrawnMerged:
				rw.Skipped = append(rw.Skipped, w)
			case w.State == WithdrawnAdmitted && !rw.Ended:
				rw.Admitted = append(rw.Admitted, w)
			case !w.Pending():
			case !rw.Ended:
				rw.Held = append(rw.Held, w)
			case reader != "" && sameHolder(w.By, reader):
				rw.Due = append(rw.Due, w)
			default:
				rw.Stranded = append(rw.Stranded, w)
			}
		}
		if len(rw.Held)+len(rw.Admitted)+len(rw.Due)+len(rw.Stranded)+len(rw.Skipped) > 0 {
			out = append(out, rw)
		}
	}
	return out
}
