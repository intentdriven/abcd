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
//     one after the tombstone that ends the reservation, and checks every
//     admitted one on the forge, arming it when it is not armed; any act of
//     a copy of the same person, the cutting copy's included, does the same
//     for a release it sees ended (its tag on origin, or a tombstone the
//     forge's record of pushes says its holder pushed) with withdrawals of
//     that person's not yet settled (dcn-2610222219297651,
//     dcn-2610230833367984).
//     Each is acted on only when the commit that first recorded it was
//     pushed, as the forge's record of pushes says, by its withdrawer
//     (withdrawpush.go).
//     A release lease that lapsed with no tag on origin ends the hold once
//     the take-over margin has passed, for its own holder alone: the next
//     act of a copy of that person ends it with a tombstone and re-arms what
//     the cut recorded withdrawn, while a cut still going on renews its lease
//     (dcn-2610271926369425, superseding Q1 of dcn-2610230328220298);
//   - every re-arm is in the form the pull request was taken from, pinned to
//     the head it had, and one whose head moved, or that was closed or
//     merged, is skipped and named (dcn-2610222219297487).
//
// A copy of another person never re-arms: it carries the withdrawals forward
// untouched and names them, with whose they are.

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/lease"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// ActAdmitted is an --admit that re-armed one withdrawn pull request and
// recorded it admitted.
const ActAdmitted Act = "admitted"

// The forge acts a ForgeFailure names.
const (
	ForgeWithdraw = "withdraw"
	ForgeRearm    = "re-arm"
	ForgeRecord   = "record"
	ForgeComment  = "comment"
)

// ForgeFailure is one pull request the forge, or the record of it, failed
// on: the act (ForgeWithdraw, ForgeRearm, ForgeRecord, ForgeComment) and the reason.
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
	case ForgeRecord, ForgeComment:
		return fmt.Sprintf("#%d: %s", f.Number, f.Reason)
	}
	return fmt.Sprintf("#%d was not re-armed: %s; it stays withdrawn, and the next act of a copy of its holder tries again", f.Number, f.Reason)
}

// Settle is what an act did for one release's withdrawals that were due:
// those it re-armed or skipped (Settled, each in its new state), the
// failures, and the tombstone that recorded it (Commit), or why none was
// pushed (Failed). Every one it acted on is the act's person's own, read from
// their newest push to the branch, as the forge's record of pushes shows it
// (dcn-2610231033516293). NotRead counts the commits above that push that
// other logins pushed, which are not read; Unplaced, when set, is why that
// newest push could not be found or placed, and nothing of theirs was acted
// on; NotEnded, when set, is the holder of a release whose tip is a tombstone
// that holder's own newest push is not, which ends nothing. NotRead, Unplaced
// and NotEnded are named once for each tip. ByHand is each entry under
// preIDMarker the act settled as left to be re-armed by hand when it ended
// the release (settledByHand): no copy re-arms one, and nothing is posted on
// it.
type Settle struct {
	Subject  string         `json:"subject"`
	Commit   string         `json:"commit,omitempty"`
	Settled  []Withdrawal   `json:"settled"`
	Failures []ForgeFailure `json:"failures,omitempty"`
	Failed   string         `json:"failed,omitempty"`
	// Person is whose record NotRead and Unplaced speak of.
	Person   string      `json:"person,omitempty"`
	NotRead  []PushCount `json:"not_read,omitempty"`
	Unplaced string      `json:"unplaced,omitempty"`
	NotEnded string      `json:"not_ended,omitempty"`
	// NotEndedClaimed is true when NotEnded is a holder id that is not the
	// person's own, which the reading only has the trailer's word for
	// (holderClaimed).
	NotEndedClaimed bool `json:"not_ended_claimed,omitempty"`
	// LapsedAt, when set, is when the release's lease lapsed: the settle
	// ended a cut that lapsed with no tag on origin (dcn-2610271926369425).
	LapsedAt *time.Time   `json:"lapsed_at,omitempty"`
	ByHand   []Withdrawal `json:"by_hand,omitempty"`
}

// Notes is each sentence naming what the settle did not act on, the entries
// it left to be re-armed by hand, and the lapse it ended a cut on.
func (s Settle) Notes() []string {
	var out []string
	for _, l := range []string{lapsedEndLine(s.Subject, s.LapsedAt), notEndedLine(s.Subject, s.NotEnded, s.NotEndedClaimed), unplacedLine(s.Subject, s.Person, s.Unplaced),
		notReadLine(s.Subject, s.Person, s.NotRead), ByHandLine(s.Subject, s.ByHand)} {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

// said reports whether s says anything.
func (s Settle) said() bool {
	return s.Commit != "" || len(s.Settled)+len(s.Failures)+len(s.NotRead)+len(s.ByHand) > 0 || s.Failed != "" || s.Unplaced != "" || s.NotEnded != ""
}

// lapsedEndLine is the sentence naming a cut a settle ended because its lease
// lapsed with no tag on origin; "" when at is nil.
func lapsedEndLine(subject string, at *time.Time) string {
	if at == nil {
		return ""
	}
	return fmt.Sprintf("%s's lease lapsed at %s with no tag on %s, so its hold has ended and what it withdrew of yours is re-armed",
		subject, stamp(*at), Remote)
}

// notEndedLine is the sentence naming a release whose tip is a tombstone its
// holder's own newest push is not; claimed marks a holder the reader has only
// the trailer's word for (holderClaimed).
func notEndedLine(subject, holder string, claimed bool) string {
	if holder == "" {
		return ""
	}
	name := claimedName(holder, claimed)
	return fmt.Sprintf("%s's tip is a tombstone %s did not push, as the forge's record of pushes says, over %s's reservation, so it ends nothing "+
		"and no withdrawal is re-armed: the release ends with %s's --release, or when its tag is on %s", subject, name, name, name, Remote)
}

// holderClaimed reports whether holder, a release's holder as a reading of
// reader's names it, is only claimed to reader: a holder id that is not
// theirs names no account the forge's record of pushes could vouch for, so
// whose it is only the trailer says, as `abcd status` shows a sender a
// branch named by a holder id names (SenderClaimed). Local renders only.
func holderClaimed(holder string, reader Self) bool {
	return validHolderID(holder) && !reader.Owns(holder)
}

// claimedName is holder as a render names it: marked claimed when claimed.
func claimedName(holder string, claimed bool) string {
	if claimed {
		return holder + " (" + SenderClaimed + ")"
	}
	return holder
}

// ByHandLine is the sentence naming the entries under preIDMarker on
// subject, which no copy re-arms: their authors re-arm them by hand once the
// release ends. It names the pull requests alone, never a holder; "" when ws
// is empty.
func ByHandLine(subject string, ws []Withdrawal) string {
	switch len(ws) {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf("%s was withdrawn for %s by a holder from before the holder id, whom nothing abcd pushes may name, so no copy re-arms it: "+
			"once the release ends, its author re-arms it by hand", numbers(ws)[0], subject)
	}
	return fmt.Sprintf("%s were withdrawn for %s by holders from before the holder id, whom nothing abcd pushes may name, so no copy re-arms them: "+
		"once the release ends, each one's author re-arms it by hand", strings.Join(numbers(ws), ", "), subject)
}

// SettledLine is the sentence naming one settled withdrawal.
func SettledLine(subject string, w Withdrawal) string {
	if w.preID() && w.State == WithdrawnManual {
		return ByHandLine(subject, []Withdrawal{w})
	}
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
	case WithdrawnMismatch:
		return fmt.Sprintf("#%d was not re-armed after %s: the node id recorded for it is not the forge's for #%d, so the record was not trusted to name it", w.Number, subject, w.Number)
	case WithdrawnManual:
		return fmt.Sprintf("#%d was not re-armed after %s: it can merge now and there is no merge queue to put it in, so it needs a manual merge by its owner", w.Number, subject)
	case WithdrawnRefused:
		return fmt.Sprintf("#%d was not re-armed after %s: the forge refused its re-arm %d times, so no copy tries again; its owner re-arms it", w.Number, subject, maxRearmRefusals)
	case WithdrawnKept:
		return fmt.Sprintf("#%d was not withdrawn for %s: the forge refused it, so it was left armed as it was", w.Number, subject)
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

// StrandedNote names every withdrawal of ws, the stranded ones on subject,
// to a reader whose kept holder id is ownID: another person's by whose they
// are (StrandedLine), those naming the reader's own id by why no copy of
// theirs acts on them (ownStrandedLine), never calling the reader's own id
// another's, and those under preIDMarker as left to be re-armed by hand
// (ByHandLine).
func StrandedNote(subject string, ws []Withdrawal, ownID string, loginKnown bool) string {
	var others, own, byHand []Withdrawal
	for _, w := range ws {
		switch {
		case w.preID():
			byHand = append(byHand, w)
		case ownID != "" && w.By == ownID:
			own = append(own, w)
		default:
			others = append(others, w)
		}
	}
	var parts []string
	if len(byHand) > 0 {
		parts = append(parts, ByHandLine(subject, byHand))
	}
	if len(others) > 0 {
		parts = append(parts, StrandedLine(subject, others))
	}
	if len(own) > 0 {
		parts = append(parts, ownStrandedLine(subject, own, loginKnown))
	}
	return strings.Join(parts, "; ")
}

// ownStrandedLine names the reader's own stranded withdrawals on subject:
// with their login not known here, their own newest push cannot be read, and
// a reservation act, which asks the forge, or a login named locally settles
// them; with it known, the reservation that first recorded them under the
// reader's id was not pushed by the reader's own account, so no copy of
// theirs acts on them. It names no login.
func ownStrandedLine(subject string, ws []Withdrawal, loginKnown bool) string {
	if !loginKnown {
		return fmt.Sprintf("%s withdrawn under your holder id for %s are not re-armed from here: which GitHub account you are is not known in this session, "+
			"so your own newest push to its branch cannot be read; a reservation act, which asks the forge, settles them, "+
			"or set git's user.email to your GitHub no-reply address, or reservation.github_username in ~/.abcd.noindex/config.json",
			strings.Join(numbers(ws), ", "), subject)
	}
	return fmt.Sprintf("%s named by your holder id on %s stay withdrawn: the reservation that first recorded them under your id was not pushed by your own GitHub account, "+
		"so no copy of yours acts on them, and each pull request's author re-arms it by hand", strings.Join(numbers(ws), ", "), subject)
}

// actor is who an act's commits are written as: the person, their holder,
// this copy's key, the clock and the Assisted-by label; subject is the
// release whose withdrawals it settles, which a comment names.
type actor struct {
	root string
	who  identity
	// holder is the holder id every message the actor writes names, and self
	// whose withdrawals and pushes are the actor's own (Self): the login in
	// it is what the forge's record of pushes and gh name the person by.
	holder  string
	self    Self
	copy    string
	now     time.Time
	label   string
	subject string
}

func (p *prepared) actor() actor {
	return actor{root: p.root, who: p.who, holder: p.holder, self: p.self, copy: p.copy, now: p.now, label: p.label, subject: p.id}
}

// longestLogin and longestLabel are the room a tombstone leaves for whoever
// writes it: the longest holder name, a GitHub username of 39 characters,
// which an old withdrawal may still name and which is longer than any holder
// id, and an Assisted-by label longer
// than any abcd composes. Whichever copy of whichever person settles a
// release writes its tombstone, so its fit is not measured by the cutter's
// own login and label.
var (
	longestLogin = strings.Repeat("x", 39)
	longestLabel = "abcd:" + strings.Repeat("9", 59)
)

// longestState is the longest word in WithdrawalStates.
func longestState() string {
	l := ""
	for _, s := range WithdrawalStates {
		if len(s) > len(l) {
			l = s
		}
	}
	return l
}

// tombstoneFits reports whether a release's tombstone carrying ws, each in
// the longest state, would be written by any holder: under the cap and past
// the outbound policy. A reservation records no withdrawal its own release
// could not carry.
func (p *prepared) tombstoneFits(ws []Withdrawal) bool {
	m := p.tombstone(nil)
	m.Holder = longestLogin
	if len(m.AssistedBy) < len(longestLabel) {
		m.AssistedBy = longestLabel
	}
	st := longestState()
	for _, w := range ws {
		w.State = st
		m.Withdrawn = append(m.Withdrawn, w)
	}
	return checkMessage(p.root, m.text()) == nil
}

// thisBranch is the branch the checkout at root is on, "" when detached or
// unreadable: a pull request from it is the release pull request itself, which
// the cut opened, and the hold neither withdraws nor judges it.
func thisBranch(root string) string {
	out, err := localGit(root, 4096, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// loginMismatch compares the account the session's gh token acts as, which
// every withdrawal and re-arm is made under, with self's login, the account
// the act's holder resolver named. It returns "" when they are the same
// person; otherwise why nothing may be touched, and definite when gh named
// another account (a lookup that failed is not definite). The login is only
// compared and named here, on this machine; nothing pushed carries it.
func loginMismatch(root string, self Self) (why string, definite bool) {
	if releaseForge.login == nil {
		return "", false
	}
	if _, _, err := releaseForge.repo(root); err != nil {
		return "", false // no forge to act on; the act names that itself
	}
	login, err := releaseForge.login(root)
	switch {
	case err != nil:
		return "gh could not say which GitHub account it acts as, so no merge was withdrawn or re-armed", false
	case !validUsername(login):
		return "gh acts as an account that is not a GitHub username, so no merge was withdrawn or re-armed", true
	case self.Login == "":
		return "which GitHub account you reserve as could not be told, so gh's account cannot be checked against it and no merge was withdrawn or re-armed", false
	case !sameHolder(login, self.Login):
		return fmt.Sprintf("gh acts as %s, but you reserve as %s, and every withdrawal and re-arm is made under gh's account: "+
			"none was made; switch gh to %s (gh auth switch) and run it again", login, self.Login, self.Login), true
	}
	return "", false
}

// pushRecord writes m, a renewal of the reservation, over parent as p and
// pushes it. It returns the commit pushed, or why it was not.
func (p *prepared) pushRecord(tree, parent string, m message) (string, error) {
	sha, err := writeCommit(p, m, tree, parent)
	if err == nil {
		err = push(p.root, pushArgs(sha, p.id))
	}
	if err != nil {
		return "", err
	}
	return sha, nil
}

// withdrawArmed lists the merges armed or queued on origin's default branch
// and withdraws them for the release reservation m, just pushed as sha. The
// record comes first: a renewal naming every
// one about to be withdrawn, still withdrawn, is pushed over sha before the
// forge is touched, so a process killed mid-way leaves a record of every
// merge it may have withdrawn, and one it never reached reads as armed when
// the re-arm comes to it, which counts as done. When the forge then refuses
// one, or withdraws only half of one, a second renewal records what actually
// happened. It skips the release pull request itself (a same-repository pull
// request from this checkout's branch), one already admitted, and one past
// what a tombstone can carry. It returns the commit last pushed (sha when
// none was), what that commit records, the withdrawals made, the failures,
// and why the merges could not be listed or touched at all, "" when they
// could.
//
// again is a live renewal's: it withdraws again only a merge m already
// records as this person's, withdrawn or kept, that the forge lists armed now
// (a process cut off between its record and the forge, an arming made since
// by hand, or a withdrawal the forge refused before). m is read from this
// person's own newest push (dcn-2610231033516293), so each such entry is
// theirs; it lists nothing when m records none such, and touches nothing
// else.
func (p *prepared) withdrawArmed(tree, sha string, m message, again bool) (commit string, recorded message, made []Withdrawal, fails []ForgeFailure, unchecked string) {
	commit, recorded = sha, m
	retryable := func(w Withdrawal) bool {
		return (w.Pending() || w.State == WithdrawnKept) && p.acts(w)
	}
	if again && !slices.ContainsFunc(m.Withdrawn, retryable) {
		return commit, recorded, nil, nil, ""
	}
	owner, name, err := releaseForge.repo(p.root)
	if err != nil {
		return commit, recorded, nil, nil, "could not withdraw the merges already armed: " + err.Error()
	}
	base := strings.TrimPrefix(originDefault(p.root), "refs/remotes/"+Remote+"/")
	if base == "" {
		return commit, recorded, nil, nil, "could not withdraw the merges already armed: this checkout names no default branch of " + Remote
	}
	armed, err := releaseForge.armedPullRequests(p.root, owner, name, base)
	if err != nil {
		return commit, recorded, nil, nil, "could not list the merges armed or queued on " + base + ": " + forgeReason(err) +
			"; none was withdrawn, and the launch check refuses the cut while any is armed"
	}
	branch := thisBranch(p.root)
	ours := func(pr ArmedPR) bool { return !pr.CrossRepository && branch != "" && pr.HeadRef == branch }
	all := slices.Clone(m.Withdrawn)
	var due []ArmedPR
	for _, pr := range armed {
		if ours(pr) {
			continue // the release pull request itself
		}
		if have, ok := withdrawalOf(all, pr.Number); ok && have.State == WithdrawnAdmitted {
			continue // ruled into the cut already
		}
		if again {
			if w, ok := withdrawalOf(all, pr.Number); !ok || !retryable(w) {
				continue
			}
		}
		w := Withdrawal{Number: pr.Number, ID: pr.ID, Form: withdrawalFormOf(pr.Queued, pr.Method), Head: pr.Head, By: p.holder, State: WithdrawnPending}
		next := withWithdrawal(all, w)
		if len(next) > maxWithdrawals || !p.tombstoneFits(next) {
			fails = append(fails, ForgeFailure{Number: pr.Number, Act: ForgeWithdraw,
				Reason: fmt.Sprintf("the reservation has no room left to record it (at most %d pull requests, under the %d-byte cap)", maxWithdrawals, MaxMessageBytes)})
			continue
		}
		all, due = next, append(due, pr)
	}
	if len(due) == 0 {
		return commit, recorded, nil, fails, ""
	}
	if why, _ := loginMismatch(p.root, p.self); why != "" {
		return commit, recorded, nil, fails, why + "; the launch check refuses the cut while any is armed"
	}
	// The record first: every one about to be withdrawn, as withdrawn. A
	// renewal that withdraws again what it already records as withdrawn, at
	// the same heads, has nothing new to record.
	first := m
	first.Withdrawn = all
	rec, err := sha, error(nil)
	if !slices.Equal(all, m.Withdrawn) {
		rec, err = p.pushRecord(tree, sha, first)
	}
	if err != nil {
		reason := redacted(err)
		for _, pr := range due {
			fails = append(fails, ForgeFailure{Number: pr.Number, Act: ForgeRecord,
				Reason: "was not withdrawn: the reservation recording it first was not pushed (" + reason + "), so it was left armed as it was; the launch check refuses the cut while it is armed"})
		}
		return commit, recorded, nil, fails, ""
	}
	commit, recorded = rec, first
	// Then the forge, and a correction for whatever did not go as recorded.
	actual := slices.Clone(all)
	var changed []int
	for _, pr := range due {
		got, done, err := releaseForge.withdraw(p.root, pr, p.holder)
		if err != nil {
			fails = append(fails, ForgeFailure{Number: pr.Number, Act: ForgeWithdraw, Reason: err.Error()})
		}
		switch {
		case !done:
			// Kept, not dropped: a dropped entry reads as one a later
			// commit lost (lostWithdrawals).
			if w, ok := withdrawalOf(actual, pr.Number); ok {
				w.State = WithdrawnKept
				actual = withWithdrawal(actual, w)
			}
			changed = append(changed, pr.Number)
		case got.Form != withdrawalFormOf(pr.Queued, pr.Method):
			actual = withWithdrawal(actual, got)
			changed = append(changed, pr.Number)
			made = append(made, got)
		default:
			made = append(made, got)
		}
		// Every withdrawal made whole is a real disarm, of a pull request
		// the forge lists as armed now, so each is commented on: a first
		// withdrawal made on a retry, and one made again after its owner
		// re-armed it by hand, included. A refused one, and a partial one
		// (auto-merge off, still queued), posts nothing: the failure line
		// names it to the cutter.
		if done && err == nil {
			if f := releaseForge.comment(p.root, p.id, got, p.holder, false); f != nil {
				fails = append(fails, *f)
			}
		}
	}
	if len(changed) == 0 {
		return commit, recorded, made, fails, ""
	}
	fixed := m
	fixed.Withdrawn = actual
	next, err := p.pushRecord(tree, rec, fixed)
	if err != nil {
		reason := redacted(err)
		for _, n := range changed {
			fails = append(fails, ForgeFailure{Number: n, Act: ForgeRecord,
				Reason: "the reservation records it as withdrawn in the form it was armed in, but the correction saying what the forge did was not pushed (" + reason +
					"); its re-arm reads it on the forge first, and finds it armed when it still is"})
		}
		return commit, recorded, made, fails, ""
	}
	return next, fixed, made, fails, ""
}

// ownRelease reads a release's tip as this act judged it (j) against the
// forge's record of pushes (dcn-2610231033516293), before the act decides
// anything: the release is judged as its holder's own newest push records it
// (asker.ended), so a commit another login pushed over it, a tombstone among
// them, neither ends nor renews it, and this person's withdrawals are carried
// only as their own newest push records them (merged), so nothing another
// login pushed is carried as theirs. The settle it returns counts the commits
// above that push other logins pushed, which are not read. When either
// cannot be found or placed, the act is refused and nothing is pushed: abcd
// acts on nothing of this person's while it cannot read their record. An
// origin with no forge holds no withdrawal to read, and a branch on which no
// commit records one has nothing to read either: both are judged by the tip,
// as before. A tip that does not read is judged as the holder's own newest
// push records the release, when both are found; otherwise it is judged by
// the tip, as before.
func (p *prepared) ownRelease(tree string, j *judged) (Settle, error) {
	s := Settle{Subject: p.id, Settled: []Withdrawal{}, Person: p.holder}
	if !isRelease(p.id) || !j.present {
		return s, nil
	}
	unread := j.Unreadable != ""
	if _, _, err := releaseForge.repo(p.root); err != nil {
		return s, nil
	}
	links, complete, err := releaseChain(p.root, p.id, j.Commit, tree)
	if err != nil {
		return s, refuse("could not read the history of %s's branch to tell which withdrawals are yours (%s); %s", p.id, redacted(err), nothingPushed)
	}
	if complete && !slices.ContainsFunc(links, func(l chainLink) bool { return len(l.m.Withdrawn) > 0 }) {
		return s, nil
	}
	k := &asker{c: pushCheck{p.root, p.id, []string{j.Commit}}, links: links, complete: complete, self: p.self}
	defer k.save()
	_, holder, held, why := k.ended()
	own := k.newest(p.self.Login)
	switch {
	case unread && (why != "" || own.why != "" || holder == "" || !held.found):
		return s, nil
	case why != "":
		return s, refuse("%s; %s", unplacedLine(p.id, p.holder, why), nothingPushed)
	case own.why != "":
		return s, refuse("%s; %s", unplacedLine(p.id, p.holder, own.why), nothingPushed)
	}
	// What the commit the act's push is written on records: an unvouched
	// entry under the reader's kept id is carried only over a commit that
	// holds it (merged).
	onto := j.Withdrawn
	if holder != "" && held.found && (held.commit != j.Commit || unread) {
		// The release as its holder's own newest push records it, over the
		// tip the act's push is written on.
		j.Unreadable = ""
		j.message = held.m
		j.verdict = lease.Judge(lease.Stamp{ClaimedAt: held.m.ClaimedAt, ExpiresAt: held.m.ExpiresAt}, lease.Default, p.now)
		j.mine, j.thisCopy = p.self.Owns(held.m.Holder), held.m.Copy == p.copy
	}
	j.Withdrawn = merged(k, j.Withdrawn, onto, own, p.self)
	p.vouched = k.vouched(j.Withdrawn, own)
	s.NotRead = own.above
	return s, nil
}

// pushable is ws as a new commit on a release may carry them: no push
// carries a login, so an entry naming its withdrawer by a login (one recorded
// before the holder id by another holder; the reader's own are named by their
// kept id by then, Self.asKept) is written under preIDMarker while it is
// still to settle (withdrawn, admitted or kept), kept in the record and no
// one's, and one already settled needs nothing more and is left out.
func pushable(ws []Withdrawal) []Withdrawal {
	var out []Withdrawal
	for _, w := range ws {
		switch {
		case validHolderID(w.By) || w.preID():
			out = append(out, w)
		case w.unsettled() || w.State == WithdrawnKept:
			w.By = preIDMarker
			out = append(out, w)
		}
	}
	return out
}

// withdrawalOf is ws's entry for pull request n.
func withdrawalOf(ws []Withdrawal, n int) (Withdrawal, bool) {
	i := slices.IndexFunc(ws, func(w Withdrawal) bool { return w.Number == n })
	if i < 0 {
		return Withdrawal{}, false
	}
	return ws[i], true
}

// withoutWithdrawal is ws less the entry for pull request n.
func withoutWithdrawal(ws []Withdrawal, n int) []Withdrawal {
	return slices.DeleteFunc(slices.Clone(ws), func(w Withdrawal) bool { return w.Number == n })
}

// rearmMine settles each withdrawal in ws still to settle (withdrawn, or
// admitted) that a may act on (actsOn: one naming a's holder themselves, or
// one vouched names), and returns ws
// with their new states, the ones it settled, whether any state changed, and
// the failures. A withdrawn one is re-armed in its own form, pinned to its
// head; an admitted one is checked on the forge, which counts it done when
// it is armed or queued, and arms it at its admitted head when it is not
// (an --admit cut off between its record and its arming). An admitted one
// that merged in the cut changes state without being named as settled. A
// re-arm the forge refuses is counted in the local tier; at the
// maxRearmRefusals-th it is settled as refused, for its owner, and no copy
// tries again. why is set, and nothing is touched, when the forge cannot be
// named or gh acts as another account than a's holder.
func rearmMine(a actor, ws []Withdrawal, vouched map[Withdrawal]bool) (out, settled []Withdrawal, changed bool, fails []ForgeFailure, why string) {
	out = slices.Clone(ws)
	acts := actsOn(a.self, vouched)
	mine := func(w Withdrawal) bool { return w.unsettled() && acts(w) }
	if !slices.ContainsFunc(ws, mine) {
		return out, nil, false, nil, ""
	}
	owner, name, err := releaseForge.repo(a.root)
	if err != nil {
		return out, nil, false, nil, "could not re-arm the withdrawn merges: " + err.Error()
	}
	if why, _ := loginMismatch(a.root, a.self); why != "" {
		return out, nil, false, nil, why
	}
	refusals := readRefusals(a.root, a.subject)
	for i, w := range out {
		if !mine(w) {
			continue
		}
		state, err := releaseForge.inspect(a.root, owner, name, w)
		if err != nil {
			fails = append(fails, ForgeFailure{Number: w.Number, Act: ForgeRearm, Reason: err.Error()})
			continue
		}
		if state == "" {
			if state, err = releaseForge.arm(a.root, w); err != nil {
				// Only the forge refusing the arming counts toward the bound:
				// a failure on the way to it (the network, a 5xx, gh out of
				// time) says nothing of whether it would be taken.
				var ae *armError
				if !errors.As(err, &ae) || !ae.refused {
					fails = append(fails, ForgeFailure{Number: w.Number, Act: ForgeRearm,
						Reason: err.Error() + " (the forge did not answer the arming, so it is not counted as a refusal)"})
					continue
				}
				k := refusalKey(w)
				refusals[k]++
				if refusals[k] < maxRearmRefusals {
					fails = append(fails, ForgeFailure{Number: w.Number, Act: ForgeRearm,
						Reason: fmt.Sprintf("%s (refusal %d of %d before it is left to its owner)", err.Error(), refusals[k], maxRearmRefusals)})
					continue
				}
				state = WithdrawnRefused
			}
		}
		admitted := w.State == WithdrawnAdmitted
		out[i].State = state
		changed = true
		if admitted && (state == WithdrawnMerged || state == WithdrawnFoundArmed) {
			continue // admitted into the cut, and done
		}
		settled = append(settled, out[i])
	}
	keepRefusals(a.root, a.subject, refusals)
	return out, settled, changed, fails, ""
}

// maxRearmRefusals is how many times the forge may refuse one withdrawal's
// re-arm, for a reason other than clean status, before it is settled as
// refused and left to its owner: a refusal that lasts would otherwise be
// asked again on every act of every copy of its holder.
const maxRearmRefusals = 5

// refusalsDir is where each release's re-arm refusals are counted, in the
// local tier: a count is this copy's own, and costs only a few more tries
// when it is lost.
const refusalsDir = stateRelDir + "/rearm-refusals"

// maxRefusalsBytes bounds what readRefusals reads.
const maxRefusalsBytes = 16 << 10

// refusalKey names one withdrawal in a release's refusal count: its number
// and head, so a re-withdrawal at a new head counts afresh.
func refusalKey(w Withdrawal) string { return strconv.Itoa(w.Number) + " " + w.Head }

// readRefusals is the refusal counts kept for release id; empty when none is
// kept or it does not read. A line that does not read is passed over.
func readRefusals(root, id string) map[string]int {
	out := map[string]int{}
	r, err := os.OpenRoot(root)
	if err != nil || !isRelease(id) {
		return out
	}
	defer r.Close()
	data, err := fsutil.ReadGuardedInRoot(r, refusalsDir+"/"+id, maxRefusalsBytes)
	if err != nil {
		return out
	}
	for _, l := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		f := strings.Fields(l)
		if len(f) != 3 || !withdrawnNumberRe.MatchString(f[0]) || !validBlob(f[1]) {
			continue
		}
		if n, err := strconv.Atoi(f[2]); err == nil && n > 0 {
			out[f[0]+" "+f[1]] = n
		}
	}
	return out
}

// keepRefusals keeps counts for release id, best effort: a count lost costs
// only a few more tries.
func keepRefusals(root, id string, counts map[string]int) {
	if len(counts) == 0 || !isRelease(id) {
		return
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k + " " + strconv.Itoa(counts[k]) + "\n")
	}
	_ = writeUnflushedIn(root, refusalsDir, id, []byte(b.String()))
}

// shownDirRel is where the tip at which each release's commits not read, and
// a person's newest push not found, were last named is kept, in the local
// tier, so each is named once for each tip rather than on every act.
const shownDirRel = stateRelDir + "/withdrawals-shown"

// shownOnce clears s's NotRead, Unplaced and NotEnded when they were already
// named at commit, id's tip. Otherwise it returns what keeps commit as the tip
// they were named at, for the caller to run once the settle naming them is
// the one the act returns: a settle a lost push discards names nothing, so it
// marks nothing shown.
func (s *Settle) shownOnce(root, id, commit string) (shown func()) {
	shown = func() {}
	if len(s.NotRead) == 0 && s.Unplaced == "" && s.NotEnded == "" || !isRelease(id) {
		return shown
	}
	if r, err := os.OpenRoot(root); err == nil {
		data, rerr := fsutil.ReadGuardedInRoot(r, shownDirRel+"/"+id, 128)
		r.Close()
		if rerr == nil && strings.TrimSpace(string(data)) == commit {
			s.NotRead, s.Unplaced, s.NotEnded, s.NotEndedClaimed = nil, "", "", false
			return shown
		}
	}
	return func() { _ = writeUnflushedIn(root, shownDirRel, id, []byte(commit+"\n")) }
}

// stranded is the withdrawals in ws still withdrawn that acts does not act
// on: another person's, and those of a declared id this copy does not act on
// (one its paired login's own push does not record, or, with no login
// beside it, one no reservation of that id pushed by the reader's own login
// first recorded).
func stranded(ws []Withdrawal, acts func(Withdrawal) bool) []Withdrawal {
	var out []Withdrawal
	for _, w := range ws {
		if w.Pending() && !acts(w) {
			out = append(out, w)
		}
	}
	return out
}

// unpairedOf is each id self declares with no login beside it that names a
// withdrawal of ws, the stranded ones, still withdrawn, in the order ws
// names them: an id whose withdrawals the reader's own login vouched for is
// acted on, so it is never stranded and never named here.
func unpairedOf(ws []Withdrawal, self Self) []string {
	var out []string
	for _, w := range ws {
		if w.Pending() && self.unpaired(w.By) && !slices.Contains(out, w.By) {
			out = append(out, w.By)
		}
	}
	return out
}

// UnpairedLine is the sentence naming the ids the person declares with no
// login beside them whose withdrawals here this copy acts on none of, since
// no reservation of theirs that the reader's own login pushed recorded them,
// and how to let it: pair each with its account's login in
// DeclaredHolderIDsFile. It names the ids alone, never a login.
func UnpairedLine(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	if len(ids) == 1 {
		return fmt.Sprintf("%s is an id %s declares as yours with no login beside it, and your own login did not push the reservation that recorded these withdrawals, "+
			"so this copy acts on none of them: write the GitHub login of the account it is yours on after it, on its line, and this copy settles them", ids[0], DeclaredHolderIDsFile)
	}
	return fmt.Sprintf("%s are ids %s declares as yours with no login beside them, and your own login did not push the reservations that recorded these withdrawals, "+
		"so this copy acts on none of them: write the GitHub login of the account each is yours on after it, on its line, and this copy settles them", strings.Join(ids, ", "), DeclaredHolderIDsFile)
}

// settleEnded settles id's withdrawals of a's holder when its release has
// ended (settleTip); own is true for the holder's own --release over a tip
// that is already a tombstone. ok is false when there was nothing to say.
func settleEnded(a actor, tree, id string, own bool) (s Settle, ok bool) {
	t, present, err := readSeen(a.root, id, trackingName(id), tree, a.now, true)
	if err != nil || !present {
		return Settle{}, false
	}
	return settleTip(a, tree, id, t, own)
}

// settleTip settles id's withdrawals of a's holder over its tip t, already
// read, once the release has ended (dcn-2610230328220298): its tag on origin,
// or its holder's own newest push a tombstone of theirs (asker.ended), or the
// holder's own --release (own), or, for a's holder alone when the tip is their
// own reservation and their own newest push with nothing above it, its lease
// lapsed past TakeoverMargin with no tag on origin (cutLapsed,
// dcn-2610271926369425). A cut still going on renews its lease, so
// a lapse past the margin is a cut its holder left; another person's lapsed
// cut ends nothing here. Any other tip that is neither tagged nor a tombstone
// is a release still held, and is not read further. Otherwise a's holder's withdrawals are read from their own newest
// push to the branch, as the forge's record of pushes shows it, asked afresh
// (dcn-2610231033516293): the commits above it other logins pushed are
// counted and not read, and when that push cannot be found or placed nothing
// is acted on and the settle says why. A tip this copy already found holding
// nothing more to do for the person is not read again, nor one whose branch
// names no withdrawal of theirs; one whose reading failed is read again on
// every act until lostWindow after this copy's first failed reading of it,
// a time no writer sets. When the person's newest push is not yet
// a tombstone (the tag ended the release, or their --release finds another
// login's tombstone over it), a tombstone over t, carrying what their record
// holds, is pushed first, the compare-and-swap that claims the end: a copy
// whose push loses says nothing, and the next act reads the tip the other
// pushed. Over their own tombstone it is not claimed again: every re-arm is
// pinned to its head and reads the forge first, so a second copy of the same
// person that settles it at the same moment finds each one armed, which is
// harmless. Then each is settled, and a tombstone recording the new states is
// pushed only when a state changed, so a re-arm the forge keeps refusing
// writes nothing to origin until it is settled. A push of that tombstone that
// fails leaves what was re-armed armed, and the next act finds it armed and
// records it.
func settleTip(a actor, tree, id string, t tip, own bool) (s Settle, ok bool) {
	// A tip that does not read ends nothing by itself, but its tag still
	// ends the release, and the holder's record is read beneath it.
	tagged, err := taggedOnOrigin(a.root, id)
	if err != nil {
		return Settle{}, false
	}
	lapsedAt, lapsed := time.Time{}, false
	if !tagged {
		lapsedAt, lapsed = cutLapsed(t, a.self, a.now)
	}
	if !tagged && !lapsed && (t.Unreadable != "" || !t.message.released()) {
		return Settle{}, false
	}
	if c := readTipCache(a.root, id, t.Commit); !own && (c.isDone(a.holder, doneScope(a.self)) || c.gaveUp(a.holder, a.now)) {
		return Settle{}, false
	}
	a.subject = id
	s = Settle{Subject: id, Settled: []Withdrawal{}, Person: a.holder}
	links, complete, err := releaseChain(a.root, id, t.Commit, tree)
	if err != nil {
		s.Failed = "could not read the history of " + id + "'s branch to tell which withdrawals are yours (" + redacted(err) + "), so none was re-armed"
		return s, true
	}
	// An act that ends the release reads it whatever it holds: the holder's
	// next act over their own lapsed cut ends it, and their own --release
	// settles what is under the marker.
	ends := func(w Withdrawal) bool { return w.unsettled() && (lapsed || own && w.preID()) }
	if complete && !slices.ContainsFunc(links, func(l chainLink) bool {
		return slices.ContainsFunc(l.m.Withdrawn, func(w Withdrawal) bool { return a.self.Owns(w.By) || ends(w) })
	}) {
		// No commit on the branch names a withdrawal of theirs, so no push
		// of theirs holds one, and the act ends nothing.
		return Settle{}, false
	}
	k := &asker{c: pushCheck{a.root, id, []string{t.Commit}}, links: links, complete: complete, self: a.self}
	defer k.save()
	rec := k.newest(a.self.Login)
	if rec.why != "" {
		k.cache().markFailed(a.holder, a.now)
		s.Unplaced = rec.why
		s.shownOnce(a.root, id, t.Commit)()
		return s, s.said()
	}
	// A tip's holder, times and lease are whatever its pusher wrote, so a
	// lapse ends the cut only when the tip is the holder's own newest push, as
	// the forge's record of pushes places it, with no commit above it: a
	// commit another login pushed naming them, dated past, ends nothing, and
	// neither does a lapse beneath another login's commit.
	if lapsed && !(rec.found && rec.at == 0 && rec.commit == t.Commit) {
		return Settle{}, false
	}
	acted := pushable(merged(k, t.Withdrawn, t.Withdrawn, rec, a.self))
	vouched := k.vouched(acted, rec)
	// An entry under the marker is settled by the act that ends the release
	// (settledByHand): any settle of the reader's own, and, with none due,
	// the holder's own --release or their next act over their own lapsed
	// cut, which ends it.
	byHandDue := (own || lapsed) && slices.ContainsFunc(acted, func(w Withdrawal) bool { return w.preID() && w.unsettled() })
	// The holder's next act over their own lapsed cut ends it whatever else
	// it holds: its tombstone carries each entry this copy does not act on
	// unchanged, another holder's among them, for that holder's own copy to
	// settle as a release that has ended.
	lapseDue := lapsed && slices.ContainsFunc(acted, func(w Withdrawal) bool { return w.unsettled() && !w.preID() })
	due := slices.ContainsFunc(rec.mine(a.self), Withdrawal.unsettled) || len(vouched) > 0 || byHandDue || lapseDue
	s.NotRead = rec.above
	if !due {
		k.cache().markDone(a.holder, doneScope(a.self))
		s.shownOnce(a.root, id, t.Commit)()
		return s, s.said()
	}
	if !tagged && !own && !lapsed {
		ended, holder, _, why := k.ended()
		switch {
		case why != "":
			k.cache().markFailed(a.holder, a.now)
			s.Unplaced = why
		case !ended && holder != "":
			s.NotEnded, s.NotEndedClaimed = holder, holderClaimed(holder, a.self)
		}
		if !ended {
			s.shownOnce(a.root, id, t.Commit)()
			return s, s.said()
		}
	}
	shown := s.shownOnce(a.root, id, t.Commit)
	if why, _ := loginMismatch(a.root, a.self); why != "" {
		s.Failed = why
		shown()
		return s, true
	}
	base := t.Commit
	if !rec.m.released() {
		claimed, err := a.pushTombstone(tree, id, t.Commit, acted)
		if err != nil {
			return Settle{}, false
		}
		base, s.Commit = claimed, claimed
	}
	// The lapse is reported only once its tombstone ends the hold, so no
	// earlier return says the hold ended while it still stands.
	if lapsed {
		s.LapsedAt = &lapsedAt
	}
	shown()
	ws, settled, changed, fails, why := rearmMine(a, acted, vouched)
	ws, byHand := settledByHand(ws)
	s.Settled, s.Failures, s.Failed = append(s.Settled, settled...), fails, why
	if !changed && len(byHand) == 0 {
		return s, true
	}
	final, err := a.pushTombstone(tree, id, base, ws)
	if err != nil {
		s.Failed = fmt.Sprintf("settled %s, but the tombstone recording it was not pushed (%s); the next act of a copy of yours finds them armed and records them",
			strings.Join(numbers(append(slices.Clone(settled), byHand...)), ", "), redacted(err))
		return s, true
	}
	s.Commit, s.ByHand = final, byHand
	s.Failures = append(s.Failures, commentSettled(a, settled, lapsed)...)
	return s, true
}

// cutLapsed reports whether t is self's own readable release reservation,
// not a tombstone, whose lease lapsed at least TakeoverMargin before now, and
// when it lapsed: a cut its holder left, which ends its hold
// (dcn-2610271926369425). The margin is the one a take-over waits out, so a
// clock running behind the cutter's never ends a cut still renewing.
func cutLapsed(t tip, self Self, now time.Time) (time.Time, bool) {
	if t.Unreadable != "" || t.message.released() || !self.Owns(t.Holder) {
		return time.Time{}, false
	}
	v := lease.Judge(lease.Stamp{ClaimedAt: t.ClaimedAt, ExpiresAt: t.ExpiresAt, FirstSeen: t.FirstSeen}, lease.Default, now)
	if v.Live || now.Before(v.EffectiveExpiry.Add(TakeoverMargin)) {
		return time.Time{}, false
	}
	return v.EffectiveExpiry, true
}

// commentSettled posts the comment for each of settled, once the tombstone
// recording their new states has landed (mergeForge.comment), and returns the
// comments that failed; lapsed says the hold ended because the cut lapsed.
func commentSettled(a actor, settled []Withdrawal, lapsed bool) []ForgeFailure {
	var fails []ForgeFailure
	for _, w := range settled {
		if f := releaseForge.comment(a.root, a.subject, w, "", lapsed); f != nil {
			fails = append(fails, *f)
		}
	}
	return fails
}

// pushTombstone writes, as a, a tombstone over parent on id's branch
// recording ws, and pushes it: a plain fast-forward, so it lands only over
// the tip it was written on.
func (a actor) pushTombstone(tree, id, parent string, ws []Withdrawal) (string, error) {
	m := message{Record: id, Holder: a.holder, Copy: a.copy, ClaimedAt: a.now, ExpiresAt: a.now, ReleasedAt: a.now,
		Withdrawn: ws, AssistedBy: a.label}
	sha, err := writeEmptyCommit(a.root, a.who, a.now, m, tree, parent)
	if err == nil {
		err = push(a.root, pushArgs(sha, id))
	}
	if err != nil {
		return "", err
	}
	return sha, nil
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
	var listed []string
	for _, ref := range strings.Fields(refs) {
		if id := strings.TrimPrefix(ref, TrackingPrefix); isRelease(id) && id != skip {
			listed = append(listed, ref)
		}
	}
	// The release tips in one batch, as the view reads them: every act runs
	// this sweep, and a repository keeps a release branch for every cut.
	reads := readSeenTips(a.root, listed, tree, a.now, true)
	for i, ref := range listed {
		if reads[i].err != nil || !reads[i].ok {
			continue
		}
		if s, ok := settleTip(a, tree, strings.TrimPrefix(ref, TrackingPrefix), reads[i].t, false); ok {
			out = append(out, s)
			pushed = pushed || s.Commit != ""
		}
	}
	return out, pushed
}

// SettleEndedReleases is the fallback re-arm a listing makes (`abcd peers`),
// after its own fetch: as this checkout's person, every ended release's
// withdrawals of theirs not yet re-armed are re-armed and recorded on a
// tombstone (dcn-2610222219297651, dcn-2610230328220298). It asks the forge
// nothing and pushes nothing when no such withdrawal is there, and acts only
// when this copy's person, holder id and key are known without asking the
// network: it never mints a holder id, so a home that keeps none settles
// nothing here, and the next reserve act settles it instead.
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
	self, err := ownSelf(o.home(), h.Login, false)
	if err != nil || self.ID == "" {
		return nil, nil
	}
	key := ownCopyKey(root)
	if key == "" {
		return nil, nil
	}
	tree, err := emptyTree(root)
	if err != nil {
		return nil, err
	}
	settles, _ := sweepEnded(actor{root: root, who: who, holder: self.ID, self: self, copy: key, now: o.now(), label: o.assistedBy()}, tree, "")
	return settles, nil
}

// admitCheck reads pull request n's withdrawal w on the forge before an
// --admit records anything: the state it settles in without arming (already
// armed again counts), "" when it is to be armed, or the refusal saying why
// it is not admitted, nothing pushed.
func (p *prepared) admitCheck(w Withdrawal) (state string, err error) {
	n := w.Number
	owner, name, err := releaseForge.repo(p.root)
	if err != nil {
		return "", refuse("#%d was not admitted: %s; %s", n, err.Error(), nothingPushed)
	}
	if why, _ := loginMismatch(p.root, p.self); why != "" {
		return "", refuse("#%d was not admitted: %s; %s", n, why, nothingPushed)
	}
	state, err = releaseForge.inspect(p.root, owner, name, w)
	switch {
	case err != nil:
		return "", refuse("#%d was not admitted: the forge could not be read for it (%s), so it stays withdrawn; %s", n, err.Error(), nothingPushed)
	case state == WithdrawnMoved:
		return "", refuse("#%d was not admitted: its head moved since it was withdrawn at %s, and an admission re-arms only the commits withdrawn; "+
			"it stays withdrawn, for its owner to re-arm after the release; %s", n, w.Head[:12], nothingPushed)
	case state == WithdrawnClosed || state == WithdrawnMerged:
		return "", refuse("#%d was not admitted: it was %s since it was withdrawn; %s", n, state, nothingPushed)
	case state == WithdrawnMismatch:
		return "", refuse("#%d was not admitted: the node id the reservation records for it is not the forge's, so the record is not trusted to name it; %s", n, nothingPushed)
	}
	return state, nil
}

// settleOwnRelease settles, after --release pushed the tombstone sha
// carrying ws, read from this person's own newest push (dcn-2610231033516293),
// every one of theirs still to settle, and pushes a second tombstone when any
// state changed. It returns the settle, the commit last pushed and the
// withdrawals it records.
func (p *prepared) settleOwnRelease(tree, sha string, ws []Withdrawal) (Settle, string, []Withdrawal) {
	a := p.actor()
	s := Settle{Subject: p.id, Settled: []Withdrawal{}, Commit: sha}
	out, settled, changed, fails, why := rearmMine(a, ws, p.vouched)
	s.Settled, s.Failures, s.Failed = append(s.Settled, settled...), fails, why
	if !changed {
		return s, sha, ws
	}
	final, err := a.pushTombstone(tree, p.id, sha, out)
	if err != nil {
		s.Failed = fmt.Sprintf("settled %s, but the tombstone recording it was not pushed (%s); the next act of a copy of yours finds them armed and records them",
			strings.Join(numbers(settled), ", "), redacted(err))
		return s, sha, ws
	}
	s.Commit = final
	s.Failures = append(s.Failures, commentSettled(a, settled, false)...)
	return s, final, out
}

// stillBeingCut reports whether r, a reservation as the view lists it, is a
// release still being cut: not ended (no tombstone, no tag), and either live
// or, lapsed, still holding a withdrawal it has not settled (withdrawn or
// admitted). A lapse ends the cut only through its own person's next act once
// the take-over margin has passed (dcn-2610271926369425), so until then a
// lapsed release with held merges blocks a second release as a live one does,
// and is what the launch check names as lapsed; one that lapsed holding
// nothing is abandoned, and neither blocks nor is named.
func stillBeingCut(r Reservation) bool {
	if !isRelease(r.Record) {
		return false
	}
	switch r.State {
	case StateLive:
		return true
	case StateExpired:
		return slices.ContainsFunc(r.Withdrawn, Withdrawal.unsettled)
	}
	return false
}

// asCut is r, a reservation as the view lists it, as the release hold reads
// whether it has ended (dcn-2610230328220298, dcn-2610231033516293): a
// release tombstone ends it only when its holder's own newest push, as the
// forge's record of pushes shows it, is a tombstone of theirs (asker.ended),
// or when its tag is on origin. Over one that ends nothing, cut is the release
// as that newest push records it, judged at now, and notEnded is true. why,
// set, is what kept the record (or the branch, or the tags) from settling it,
// which is read as not ended too, with cut r itself; once this copy has failed
// to settle it for longer than lostWindow, a time no writer sets, it is read
// as ended, so a release abandoned while its record could still be read does
// not hold every later one back for good. A branch on which no commit records
// a withdrawal still to settle ends with its tombstone: a second release can
// take nothing from a cut that holds nothing withdrawn or admitted, and the
// record is not asked. Anything else is r as listed. What the record placed
// on each commit is read from the cache kept for the tip, when it holds it.
// reader is whose holds the reading is made for (ReaderSelf): one of their
// own holder ids names a push of theirs only when their login pushed it
// (asker.namesPusher), as it does for their acts, so another account's
// reservation and tombstone naming the reader's id end nothing for them.
func asCut(root string, r Reservation, now time.Time, reader Self) (cut Reservation, notEnded bool, why string) {
	if !isRelease(r.Record) || r.State != StateReleased || r.ReleasedBy != ReleasedByTombstone {
		return r, false, ""
	}
	tagged, err := taggedOnOrigin(root, r.Record)
	if err != nil {
		return r, true, "whether " + releaseTag(r.Record) + " is on " + Remote + " could not be read (" + redacted(err) + ")"
	}
	if tagged {
		return r, false, ""
	}
	tree, err := emptyTree(root)
	if err != nil {
		return r, true, "the history of its branch could not be read (" + redacted(err) + ")"
	}
	links, complete, err := releaseChain(root, r.Record, r.Commit, tree)
	if err != nil {
		return r, true, "the history of its branch could not be read (" + redacted(err) + ")"
	}
	if complete && !slices.ContainsFunc(links, func(l chainLink) bool { return slices.ContainsFunc(l.m.Withdrawn, Withdrawal.unsettled) }) {
		return r, false, ""
	}
	k := &asker{c: pushCheck{root, r.Record, []string{r.Commit}}, links: links, complete: complete, reading: true, self: reader}
	defer k.save()
	ended, holder, h, why := k.ended()
	switch {
	case why != "":
		if k.cache().endGaveUp(now) {
			return r, false, ""
		}
		k.cache().markEndFailed(now)
		return r, true, why
	case ended || holder == "":
		return r, false, ""
	}
	b := h.m
	v := lease.Judge(lease.Stamp{ClaimedAt: b.ClaimedAt, ExpiresAt: b.ExpiresAt}, lease.Default, now)
	cut = r
	cut.Holder, cut.Footprint, cut.ClaimedAt, cut.EffectiveExpiry = b.Holder, b.Footprint, b.ClaimedAt, v.EffectiveExpiry
	cut.Withdrawn, cut.Commit, cut.Stage = b.Withdrawn, h.commit, b.Stage
	cut.ReleasedBy, cut.ReleasedAt, cut.HandedBack, cut.Copy = "", time.Time{}, "", ""
	if own := ownCopyKey(root); own != "" && b.Copy == own {
		cut.Copy = CopyThis
	}
	cut.State = StateExpired
	if v.Live {
		cut.State = StateLive
	}
	return cut, true, ""
}

// otherLiveRelease refuses, with a back-off, a release reserved anew while
// another release is still being cut (stillBeingCut), as last fetched and
// read as the hold reads an end (asCut): one release is cut at a time, and a
// second would withdraw the merges the first admitted. A tombstone whose end
// the forge's record of pushes does not settle is read as ending nothing, and
// the back-off says why. A view of the reservations that does not read
// refuses: without it no other release can be ruled out. self is the person
// reserving, whose own holder ids the reading holds to their login (asCut).
func otherLiveRelease(root, id string, now time.Time, self Self) error {
	snap, err := viewOf(root, now, true, isRelease)
	if err != nil {
		return refuse("could not tell whether another release is being cut: %s; %s", err.Error(), nothingPushed)
	}
	for _, r := range snap.Reservations {
		if r.Record == id {
			continue
		}
		r, notEnded, why := asCut(root, r, now, self)
		if why != "" {
			return backOff("%s's tip is a tombstone, and whether it ends the release is not settled: %s; one release is cut at a time, so it is read as still being cut: "+
				"wait for it to end (its tag on %s, or its holder's --release); %s", r.Record, why, Remote, nothingPushed)
		}
		if !stillBeingCut(r) {
			continue
		}
		holder := claimedName(r.Holder, holderClaimed(r.Holder, self))
		over := ""
		if notEnded {
			over = fmt.Sprintf(" (its tip is a tombstone %s did not push, as the forge's record of pushes says, which ends nothing)", holder)
		}
		if r.State == StateLive {
			return backOff("%s is being cut by %s until %s%s, and one release is cut at a time: a second reservation would withdraw the merges the first admitted; "+
				"wait for it to end (its tag on %s, or its holder's --release); %s", r.Record, holder, stamp(r.EffectiveExpiry), over, Remote, nothingPushed)
		}
		lapseEnds := ""
		if !notEnded {
			lapseEnds = fmt.Sprintf(", or, from %s, at the next act of a copy of %s while the cut's tip is their own newest push", stamp(r.EffectiveExpiry.Add(TakeoverMargin)), holder)
		}
		return backOff("%s is still being cut by %s: its lease lapsed at %s%s, but it still holds merges withdrawn or admitted, and one release is cut at a time; "+
			"it ends when %s renews it (abcd reserve %s) and finishes the cut, or ends it (abcd reserve %s --release)%s; %s",
			r.Record, holder, stamp(r.EffectiveExpiry), over, holder, r.Record, r.Record, lapseEnds, nothingPushed)
	}
	return nil
}

// admitAfter re-arms the admitted withdrawal w, its record already pushed as
// sha, unless the forge already had it armed (state). When the arming fails,
// a renewal puts it back to withdrawn, and the refusal says so.
func (p *prepared) admitAfter(tree, sha string, m message, w Withdrawal, state string) (string, error) {
	if state == WithdrawnFoundArmed {
		return sha, nil
	}
	got, err := releaseForge.arm(p.root, w)
	if err == nil && got == WithdrawnRearmed {
		return sha, nil
	}
	reason := "the forge needs it merged by hand (it can merge now, and there is no merge queue to put it in)"
	if err != nil {
		reason = "the forge refused its re-arm (" + err.Error() + ")"
	}
	back := m
	back.Withdrawn = withWithdrawal(m.Withdrawn, Withdrawal{Number: w.Number, ID: w.ID, Form: w.Form, Head: w.Head, By: w.By, State: WithdrawnPending})
	if _, perr := p.pushRecord(tree, sha, back); perr != nil {
		return sha, refuse("#%d was recorded admitted, but %s, and the renewal putting it back to withdrawn was not pushed (%s): "+
			"it is not armed; run --admit %d again, which reads the forge and arms it, or release the reservation, whose settle arms what it admitted",
			w.Number, reason, redacted(perr), w.Number)
	}
	return sha, refuse("#%d was not admitted: %s, so it stays withdrawn, and a renewal recording that was pushed", w.Number, reason)
}

// validPRNumber reports whether n is a pull request number a trailer may
// carry.
func validPRNumber(n int) bool { return n > 0 && n <= 999999999 }

// ReleaseWithdrawals is one release's withdrawals as a listing names them to
// a reader: on a release still held, those it holds withdrawn and those it
// admitted, with Lapsed set when it is still being cut (stillBeingCut) and
// its lease lapsed before it ended, which ends it for its holder once the
// take-over margin has passed (dcn-2610271926369425); on one ended (its tag on origin, or its holder's
// own tombstone), the reader's own still to re-arm (Due, which the reader's
// next act re-arms) and another person's (Stranded, which only that person
// re-arms); and those its last tombstone skipped for their owners (moved,
// closed, mismatched, needing a manual merge, or refused). The reader's own
// are read from their own newest push to the branch, as the forge's record of
// pushes shows it (dcn-2610231033516293): NotRead counts the commits above it
// other logins pushed, which are not read, and Unplaced, when set, is why
// that push (or whether the release has ended) could not be read, and none of
// the reader's is listed; a reader whose login is not known has no newest
// push to read, so a withdrawal naming their id is listed as stranded, never
// due. A merged one is not listed: it needs nothing more.
type ReleaseWithdrawals struct {
	Subject  string    `json:"subject"`
	Ended    bool      `json:"ended"`
	Lapsed   bool      `json:"lapsed,omitempty"`
	LapsedAt time.Time `json:"lapsed_at,omitempty"`
	Holder   string    `json:"holder,omitempty"`
	// HolderClaimed is true when Holder is a holder id that is not the
	// reader's, which the listing only has the trailer's word for
	// (holderClaimed).
	HolderClaimed bool         `json:"holder_claimed,omitempty"`
	Held          []Withdrawal `json:"held,omitempty"`
	Admitted      []Withdrawal `json:"admitted,omitempty"`
	Due           []Withdrawal `json:"due,omitempty"`
	Stranded      []Withdrawal `json:"stranded,omitempty"`
	Skipped       []Withdrawal `json:"skipped,omitempty"`
	// Reader is whose record NotRead and Unplaced speak of.
	Reader   string      `json:"reader,omitempty"`
	NotRead  []PushCount `json:"not_read,omitempty"`
	Unplaced string      `json:"unplaced,omitempty"`
	// NotEnded, when set, is the holder of a release whose tip is a tombstone
	// that holder's own newest push is not: it ends nothing, so nothing is
	// re-armed yet.
	NotEnded string `json:"not_ended,omitempty"`
	// Unpaired is each id the reader declares with no login beside it that
	// names one of Stranded (UnpairedLine).
	Unpaired []string `json:"unpaired,omitempty"`
	// DueAwaitsHolderID is true when the reader keeps no holder id yet and
	// HolderIDEnv is not set, so no listing settles Due: their next
	// reservation keeps one and settles them, and after an announcement
	// keeps one the next listing does.
	DueAwaitsHolderID bool `json:"due_awaits_holder_id,omitempty"`
	// ownID and loginKnown are the reader's kept holder id and whether
	// their login is known, which the stranded line reads (StrandedNote).
	ownID      string
	loginKnown bool
	// HolderIDRefused is the class of the refusal of the reader's holder id
	// (Self.HolderIDRefused) when Due is not empty: every act refuses until
	// it is fixed, so nothing settles Due (HolderIDRefusedLine).
	HolderIDRefused string `json:"holder_id_refused,omitempty"`
	// ByHand is each entry under preIDMarker on a release that has ended,
	// still withdrawn or settled as left by hand: no copy re-arms one, so
	// its author re-arms it by hand (ByHandLine).
	ByHand []Withdrawal `json:"by_hand,omitempty"`
}

// Lines is each sentence the listing shows for the release's withdrawals.
func (r ReleaseWithdrawals) Lines() []string {
	var out []string
	switch {
	case r.Lapsed && r.NotEnded != "":
		out = append(out, fmt.Sprintf("%s's lease lapsed at %s before the release ended, beneath a tombstone %s did not push: it holds no merge now, and nothing re-arms "+
			"what it withdrew until %s ends it (abcd reserve %s --release, after --take-over from another copy of theirs) or its tag is on %s",
			r.Subject, stamp(r.LapsedAt), claimedName(r.NotEnded, r.HolderClaimed), claimedName(r.NotEnded, r.HolderClaimed), r.Subject, Remote))
	case r.Lapsed:
		out = append(out, fmt.Sprintf("%s's lease lapsed at %s before the release ended: it holds no merge now, and from %s the next act of a copy of %s "+
			"ends it and re-arms what it withdrew, while the cut's tip is their own newest push (else their --release ends it); %s renews it from the cutting copy first (abcd reserve %s) if the cut goes on",
			r.Subject, stamp(r.LapsedAt), stamp(r.LapsedAt.Add(TakeoverMargin)), claimedName(r.Holder, r.HolderClaimed), claimedName(r.Holder, r.HolderClaimed), r.Subject))
	}
	if len(r.Held) > 0 {
		out = append(out, fmt.Sprintf("%s holds %s withdrawn until it ends: the cutting copy admits one with --admit <pr>",
			r.Subject, strings.Join(numbers(r.Held), ", ")))
	}
	if len(r.Admitted) > 0 {
		out = append(out, fmt.Sprintf("%s admitted %s", r.Subject, strings.Join(numbers(r.Admitted), ", ")))
	}
	if r.NotEnded != "" {
		out = append(out, notEndedLine(r.Subject, r.NotEnded, r.HolderClaimed))
	}
	for _, l := range []string{unplacedLine(r.Subject, r.Reader, r.Unplaced), notReadLine(r.Subject, r.Reader, r.NotRead)} {
		if l != "" {
			out = append(out, l)
		}
	}
	switch {
	case len(r.Due) > 0 && r.HolderIDRefused != "":
		out = append(out, fmt.Sprintf("%s has ended, and %s you withdrew for it are not re-armed: %s",
			r.Subject, strings.Join(numbers(r.Due), ", "), HolderIDRefusedLine(r.HolderIDRefused)))
	case len(r.Due) > 0 && r.DueAwaitsHolderID:
		out = append(out, fmt.Sprintf("%s has ended, and %s you withdrew for it are not re-armed yet: held merges from before the holder id stay held "+
			"until your next reservation or announcement; a reservation re-arms them as it keeps your holder id, and after an announcement your next listing does",
			r.Subject, strings.Join(numbers(r.Due), ", ")))
	case len(r.Due) > 0:
		out = append(out, fmt.Sprintf("%s has ended, and %s you withdrew for it are not re-armed yet: your next abcd act that fetches the reservations re-arms them",
			r.Subject, strings.Join(numbers(r.Due), ", ")))
	}
	if len(r.Stranded) > 0 {
		out = append(out, StrandedNote(r.Subject, r.Stranded, r.ownID, r.loginKnown))
	}
	if l := UnpairedLine(r.Unpaired); l != "" {
		out = append(out, l)
	}
	if l := ByHandLine(r.Subject, r.ByHand); l != "" {
		out = append(out, l)
	}
	for _, w := range r.Skipped {
		out = append(out, SettledLine(r.Subject, w))
	}
	return out
}

// ReleaseWithdrawals is every release in the view at now whose tip records a
// withdrawal there is something to say about, for reader (whose holds are the
// listing's own, ReaderSelf; the zero Self owns none). A release that
// holds a withdrawal still to settle is read as the hold reads it (asCut), and
// the reader's own withdrawals on it from their own newest push (asker.newest),
// what the record placed on each commit taken from the cache kept for the tip
// when it holds it, so the record is asked at most once for each tip that
// reads.
func (s Snapshot) ReleaseWithdrawals(root string, reader Self, now time.Time) []ReleaseWithdrawals {
	var out []ReleaseWithdrawals
	for _, r := range s.Reservations {
		if !isRelease(r.Record) {
			continue
		}
		// A tip that does not read, or records no withdrawal, says nothing of
		// the reader's own record beneath it: the branch is read for it only
		// when a commit there records one of theirs still to settle.
		if (len(r.Withdrawn) == 0 || r.State == StateUnreadable) && !holdsUnsettledOf(root, r.Record, r.Commit, reader) {
			continue
		}
		tip := r.Commit
		cut, notEnded, why := asCut(root, r, now, reader)
		r = cut
		rw := ReleaseWithdrawals{Subject: r.Record, Holder: r.Holder, HolderClaimed: holderClaimed(r.Holder, reader), Ended: r.State == StateReleased && !notEnded,
			ownID: reader.ID, loginKnown: reader.Login != ""}
		if notEnded {
			if why == "" {
				rw.NotEnded = r.Holder
			} else {
				rw.Reader, rw.Unplaced = reader.Name(), why
			}
		}
		ws := r.Withdrawn
		// due is what the reader's next act would in fact act on: their own,
		// read from their own newest push, and a declared id's that its
		// paired login's own push records, or, unpaired, that a reservation
		// of that id the reader's own login pushed first recorded (actsOn). Nothing is due with the
		// reader's login unknown: no act of theirs re-arms it from here.
		due := func(Withdrawal) bool { return false }
		if reader.Login != "" && why == "" && (len(ws) == 0 || r.State == StateUnreadable || slices.ContainsFunc(ws, Withdrawal.unsettled)) {
			own, vouched := readerRecord(root, r.Record, tip, reader, &ws)
			rw.Reader, rw.NotRead, rw.Unplaced = reader.Name(), own.above, own.why
			if own.why == "" {
				due = actsOn(reader, vouched)
			}
		}
		for _, w := range ws {
			switch {
			case w.State == WithdrawnMerged:
			case w.preID() && rw.Ended && (w.unsettled() || w.State == WithdrawnManual):
				rw.ByHand = append(rw.ByHand, w)
			case skippedForOwner(w.State):
				rw.Skipped = append(rw.Skipped, w)
			case w.State == WithdrawnAdmitted && !rw.Ended:
				rw.Admitted = append(rw.Admitted, w)
			case !w.Pending():
			case !rw.Ended:
				rw.Held = append(rw.Held, w)
			case due(w):
				rw.Due = append(rw.Due, w)
			default:
				rw.Stranded = append(rw.Stranded, w)
			}
		}
		rw.Unpaired = unpairedOf(rw.Stranded, reader)
		if len(rw.Due) > 0 && reader.ID == "" {
			// Only a home that keeps no id and a variable not set await
			// the id: one refused refuses every act until it is fixed.
			rw.HolderIDRefused = reader.HolderIDRefused
			rw.DueAwaitsHolderID = reader.HolderIDRefused == ""
		}
		if r.State == StateExpired && stillBeingCut(r) {
			rw.Lapsed, rw.LapsedAt = true, r.EffectiveExpiry
		}
		if len(rw.Held)+len(rw.Admitted)+len(rw.Due)+len(rw.Stranded)+len(rw.Skipped)+len(rw.NotRead)+len(rw.ByHand) > 0 || rw.Lapsed || rw.NotEnded != "" || rw.Unplaced != "" {
			out = append(out, rw)
		}
	}
	return out
}

// holdsUnsettledOf reports whether a commit of id's branch at tip, as the walk
// reads it, records a withdrawal of reader's still to settle. It reads the
// branch only, never the forge.
func holdsUnsettledOf(root, id, tip string, reader Self) bool {
	if !reader.Known() || tip == "" {
		return false
	}
	tree, err := emptyTree(root)
	if err != nil {
		return false
	}
	links, _, err := releaseChain(root, id, tip, tree)
	if err != nil {
		return false
	}
	return slices.ContainsFunc(links, func(l chainLink) bool {
		return l.readable && slices.ContainsFunc(l.m.Withdrawn, func(w Withdrawal) bool { return w.unsettled() && reader.Owns(w.By) })
	})
}

// readerRecord is the record on id's release branch at tip of the person the
// forge's record of pushes names by login (asker.newest), read as a listing
// reads it: what the record placed on each commit taken from the cache kept
// for the tip when it holds it, for reader, by their login. With ws set, the
// listing's withdrawals are read for reader as an act's settle reads them:
// when that record is found or placed, *ws becomes what the act would carry
// (merged) and vouched the declared entries it would act on; when it is not,
// the reader's own are dropped from *ws, since none of them can be read.
func readerRecord(root, id, tip string, reader Self, ws *[]Withdrawal) (ownRecord, map[Withdrawal]bool) {
	unread := func(own ownRecord) (ownRecord, map[Withdrawal]bool) {
		if ws != nil {
			*ws = slices.DeleteFunc(slices.Clone(*ws), func(w Withdrawal) bool { return reader.isOwn(w.By) })
		}
		return own, nil
	}
	tree, err := emptyTree(root)
	if err != nil {
		return unread(ownRecord{why: "the history of " + id + "'s branch could not be read (" + redacted(err) + ")"})
	}
	links, complete, err := releaseChain(root, id, tip, tree)
	if err != nil {
		return unread(ownRecord{why: "the history of " + id + "'s branch could not be read (" + redacted(err) + ")"})
	}
	k := &asker{c: pushCheck{root, id, []string{tip}}, links: links, complete: complete, reading: true, self: reader}
	defer k.save()
	own := k.newest(reader.Login)
	switch {
	case ws == nil:
		return own, nil
	case own.why != "":
		return unread(own)
	}
	var onto []Withdrawal
	if len(links) > 0 && links[0].readable {
		onto = links[0].m.Withdrawn
	}
	*ws = merged(k, *ws, onto, own, reader)
	return own, k.vouched(*ws, own)
}
