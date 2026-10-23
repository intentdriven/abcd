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
//     A lapsed lease is not an end: it is reported, and nothing acts on it
//     (dcn-2610230328220298);
//   - every re-arm is in the form the pull request was taken from, pinned to
//     the head it had, and one whose head moved, or that was closed or
//     merged, is skipped and named (dcn-2610222219297487).
//
// A copy of another person never re-arms: it carries the withdrawals forward
// untouched and names them, with whose they are.

import (
	"crypto/sha256"
	"encoding/hex"
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
// pushed (Failed). None of the rest is acted on: Unrecorded is each one
// named as the act's person's that no reservation of theirs on the branch
// recorded first, or whose first record another account pushed; AppPushed
// each whose first record an app pushed; Unverified each the forge's record
// of pushes does not settle yet (UnverifiedWhy), which a later act asks
// about again; Restated each the tip records moved out of withdrawn or
// admitted by a commit its withdrawer did not push, which is acted on as it
// stood before (vouching); NotEnded, when set, is the holder of the
// reservation a tombstone they did not push was written over, which ends
// nothing. Unrecorded, AppPushed, Restated and NotEnded are named once for
// each tip.
type Settle struct {
	Subject       string         `json:"subject"`
	Commit        string         `json:"commit,omitempty"`
	Settled       []Withdrawal   `json:"settled"`
	Failures      []ForgeFailure `json:"failures,omitempty"`
	Failed        string         `json:"failed,omitempty"`
	Unrecorded    []Withdrawal   `json:"unrecorded,omitempty"`
	AppPushed     []Withdrawal   `json:"app_pushed,omitempty"`
	Unverified    []Withdrawal   `json:"unverified,omitempty"`
	UnverifiedWhy string         `json:"unverified_why,omitempty"`
	Restated      []Withdrawal   `json:"restated,omitempty"`
	NotEnded      string         `json:"not_ended,omitempty"`
}

// Notes is each sentence naming the withdrawals the settle did not act on.
func (s Settle) Notes() []string {
	var out []string
	for _, l := range []string{notEndedLine(s.Subject, s.NotEnded), restatedLine(s.Subject, s.Restated), UnrecordedLine(s.Subject, s.Unrecorded),
		appPushedLine(s.Subject, s.AppPushed), unverifiedLine(s.Subject, s.Unverified, s.UnverifiedWhy)} {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

// noted fills s's withdrawals not acted on from v.
func (s *Settle) noted(v vouching) {
	s.Unrecorded, s.AppPushed, s.Unverified, s.UnverifiedWhy, s.Restated = v.Unrecorded, v.AppPushed, v.Unverified, v.Why, v.Restated
}

// said reports whether s says anything.
func (s Settle) said() bool {
	return s.Commit != "" || len(s.Settled)+len(s.Failures)+len(s.Unrecorded)+len(s.AppPushed)+len(s.Unverified)+len(s.Restated) > 0 ||
		s.Failed != "" || s.NotEnded != ""
}

// restatedLine is the sentence naming the withdrawals whose recorded state a
// commit their withdrawer did not push changed.
func restatedLine(subject string, ws []Withdrawal) string {
	if len(ws) == 0 {
		return ""
	}
	return fmt.Sprintf("%s on %s, withdrawn by %s, recorded settled by a commit %s did not push, as the forge's record of pushes says; "+
		"a change of state counts only when its withdrawer pushed it, so abcd reads each as it stood before that commit",
		strings.Join(numbers(ws), ", "), subject, ws[0].By, ws[0].By)
}

// notEndedLine is the sentence naming a release whose tombstone its holder
// did not push.
func notEndedLine(subject, holder string) string {
	if holder == "" {
		return ""
	}
	return fmt.Sprintf("%s's tip is a tombstone %s did not push, as the forge's record of pushes says, over %s's reservation, so it ends nothing "+
		"and no withdrawal is re-armed: the release ends with %s's --release, or when its tag is on %s", subject, holder, holder, holder, Remote)
}

// UnrecordedLine is the sentence naming the withdrawals recorded as holder's
// that no reservation of theirs recorded first: "#11 on rel-v0.16.0 is
// recorded as withdrawn by octo, but no reservation of octo's recorded it:
// it is not re-armed".
func UnrecordedLine(subject string, ws []Withdrawal) string {
	if len(ws) == 0 {
		return ""
	}
	return fmt.Sprintf("%s on %s recorded as withdrawn by %s, but no reservation of %s's on its branch, pushed by %s, recorded them first, "+
		"and abcd re-arms a withdrawal only from the reservation that recorded it, so none of them is re-armed: check each pull request by hand",
		strings.Join(numbers(ws), ", "), subject, ws[0].By, ws[0].By, ws[0].By)
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

// actor is who an act's commits are written as: the person, their holder,
// this copy's key, the clock and the Assisted-by label; subject is the
// release whose withdrawals it settles, which a comment names.
type actor struct {
	root    string
	who     identity
	holder  string
	copy    string
	now     time.Time
	label   string
	subject string
}

func (p *prepared) actor() actor {
	return actor{root: p.root, who: p.who, holder: p.holder, copy: p.copy, now: p.now, label: p.label, subject: p.id}
}

// longestLogin and longestLabel are the room a tombstone leaves for whoever
// writes it: the longest GitHub username, and an Assisted-by label longer
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
// every withdrawal and re-arm is made under, with holder, the reservation's
// holder as the act resolved it. It returns "" when they are the same
// person; otherwise why nothing may be touched, and definite when gh named
// another account (a lookup that failed is not definite).
func loginMismatch(root, holder string) (why string, definite bool) {
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
	case !sameHolder(login, holder):
		return fmt.Sprintf("gh acts as %s, but the reservation's holder is %s, and every withdrawal and re-arm is made under gh's account: "+
			"none was made; switch gh to %s (gh auth switch) and run it again", login, holder, holder), true
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
// again, when not nil, is a live renewal's: it withdraws again only a merge
// m already records as this person's, withdrawn or kept, that the forge
// lists armed now (a process cut off between its record and the forge, an
// arming made since by hand, or a withdrawal the forge refused before), and
// only those again returns, which the branch vouches for (vouchAgain); it
// lists nothing when m records none such, asks again only once the listing
// names one armed, and touches nothing else.
func (p *prepared) withdrawArmed(tree, sha string, m message, again func([]Withdrawal) []Withdrawal) (commit string, recorded message, made []Withdrawal, fails []ForgeFailure, unchecked string) {
	commit, recorded = sha, m
	retryable := func(w Withdrawal) bool {
		return (w.Pending() || w.State == WithdrawnKept) && sameHolder(w.By, p.holder)
	}
	if again != nil && !slices.ContainsFunc(m.Withdrawn, retryable) {
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
	var vouched []Withdrawal
	if again != nil {
		var armedAgain []Withdrawal
		for _, pr := range armed {
			if w, ok := withdrawalOf(m.Withdrawn, pr.Number); ok && !ours(pr) && retryable(w) {
				armedAgain = append(armedAgain, w)
			}
		}
		if len(armedAgain) == 0 {
			return commit, recorded, nil, nil, ""
		}
		vouched = again(armedAgain)
	}
	all := slices.Clone(m.Withdrawn)
	var due []ArmedPR
	for _, pr := range armed {
		if ours(pr) {
			continue // the release pull request itself
		}
		if have, ok := withdrawalOf(all, pr.Number); ok && have.State == WithdrawnAdmitted {
			continue // ruled into the cut already
		}
		if again != nil {
			if _, ok := withdrawalOf(vouched, pr.Number); !ok {
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
	if why, _ := loginMismatch(p.root, p.holder); why != "" {
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
			if f := releaseForge.comment(p.root, p.id, got, p.holder); f != nil {
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

// vouchAgain is what a live renewal, just pushed as sha over the tip it read
// (parent), may withdraw again: the withdrawals of ws, as sha records them,
// that are this person's and still withdrawn or kept, and that the branch's
// history and the forge's record of pushes vouch for (vouch), as a re-arm is
// vouched for. The settle names each of the rest, not acted on: one no
// reservation of theirs recorded first, or whose first record another
// account or an app pushed, or that the record does not settle yet; those
// it named before for the same withdrawals are not named again. ws is only
// those the forge lists armed now (withdrawArmed), so it reads the branch
// and asks the forge's record once, and only when one is.
func (p *prepared) vouchAgain(tree, sha, parent string, ws []Withdrawal) ([]Withdrawal, Settle) {
	retry := func(w Withdrawal) bool { return w.Pending() || w.State == WithdrawnKept }
	s := Settle{Subject: p.id, Settled: []Withdrawal{}}
	if !slices.ContainsFunc(ws, func(w Withdrawal) bool { return retry(w) && sameHolder(w.By, p.holder) }) {
		return nil, s
	}
	links, complete, err := releaseChain(p.root, p.id, sha, tree)
	if err != nil {
		s.Failed = "could not read the history of " + p.id + "'s branch to tell whose its withdrawals are (" + redacted(err) +
			"), so none was withdrawn again; the next renewal tries again"
		return nil, s
	}
	v := vouch(links, complete, ws, p.holder, pushCheck{p.root, p.id, []string{parent, sha}}, retry)
	s.noted(v)
	s.shownOnce(p.root, p.id, againKey(s))()
	return v.Due, s
}

// againKey names the withdrawals a live renewal's settle names as not acted
// on, so shownOnce names the same ones once, not on every renewal: each
// renewal moves the tip, so the tip cannot say they were named.
func againKey(s Settle) string {
	h := sha256.New()
	for _, ws := range [][]Withdrawal{s.Unrecorded, s.AppPushed, s.Restated} {
		for _, w := range ws {
			h.Write([]byte(w.trailer() + "\n"))
		}
		h.Write([]byte("-\n"))
	}
	return "again " + hex.EncodeToString(h.Sum(nil))
}

// restated is ws, the withdrawals on the tombstone tip at commit, with each
// of this person's that a commit they did not push moved out of withdrawn or
// admitted put back as it stood before (vouching.actedOn), so a reservation
// made over that tombstone carries it to the release that re-arms it. A
// branch that does not read leaves ws as it is.
func (p *prepared) restated(tree, commit string, ws []Withdrawal) []Withdrawal {
	if !slices.ContainsFunc(ws, func(w Withdrawal) bool { return !w.unsettled() }) {
		return ws
	}
	links, complete, err := releaseChain(p.root, p.id, commit, tree)
	if err != nil {
		return ws
	}
	return vouch(links, complete, ws, p.holder, pushCheck{p.root, p.id, []string{commit}}, Withdrawal.unsettled).actedOn(ws)
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
// admitted) whose holder is a's and that trust vouches for, and returns ws
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
func rearmMine(a actor, ws []Withdrawal, trust func(Withdrawal) bool) (out, settled []Withdrawal, changed bool, fails []ForgeFailure, why string) {
	out = slices.Clone(ws)
	mine := func(w Withdrawal) bool { return w.unsettled() && sameHolder(w.By, a.holder) && trust(w) }
	if !slices.ContainsFunc(ws, mine) {
		return out, nil, false, nil, ""
	}
	owner, name, err := releaseForge.repo(a.root)
	if err != nil {
		return out, nil, false, nil, "could not re-arm the withdrawn merges: " + err.Error()
	}
	if why, _ := loginMismatch(a.root, a.holder); why != "" {
		return out, nil, false, nil, why
	}
	refusals := readRefusals(a.root, a.subject)
	for i, w := range out {
		if !w.unsettled() || !sameHolder(w.By, a.holder) || !trust(w) {
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

// shownDirRel is where the tip at which each release's unrecorded and
// app-pushed withdrawals were last named is kept, in the local tier, so each
// is named once for each tip rather than on every act.
const shownDirRel = stateRelDir + "/withdrawals-shown"

// shownOnce clears s's Unrecorded, AppPushed, Restated and NotEnded when
// they were already named at commit, id's tip. Otherwise it returns what
// keeps commit as the tip they were named at, for the caller to run once the
// settle naming them is the one the act returns: a settle a lost push
// discards names nothing, so it marks nothing shown.
func (s *Settle) shownOnce(root, id, commit string) (shown func()) {
	shown = func() {}
	if len(s.Unrecorded)+len(s.AppPushed)+len(s.Restated) == 0 && s.NotEnded == "" || !isRelease(id) {
		return shown
	}
	if r, err := os.OpenRoot(root); err == nil {
		data, rerr := fsutil.ReadGuardedInRoot(r, shownDirRel+"/"+id, 128)
		r.Close()
		if rerr == nil && strings.TrimSpace(string(data)) == commit {
			s.Unrecorded, s.AppPushed, s.Restated, s.NotEnded = nil, nil, nil, ""
			return shown
		}
	}
	return func() { _ = writeUnflushedIn(root, shownDirRel, id, []byte(commit+"\n")) }
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

// releaseEnded reports whether a release tip may have ended: its tag on
// origin, or a tombstone, which ends it only when its holder pushed it
// (settleTip). A lease that lapsed is not an end.
func releaseEnded(t tip, tagged bool) bool {
	return tagged || t.message.released()
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
// read, once the release has ended (dcn-2610230328220298): its tag on
// origin, or a tombstone its holder pushed (endedBy), or one the holder's
// own --release finds there (own). A lease that lapsed is not an end: nothing
// renews a release lease on its own, so a slow cut's lease lapses while it is
// still being cut, and a lapse is reported, never acted on. A tombstone the
// forge's record of pushes does not show the holder pushed ends nothing: it
// is named, and nothing is acted on. Only a withdrawal whose first record
// the branch's history vouches for, and the forge's record of pushes says its
// withdrawer pushed, is acted on (vouch), in the state its withdrawer's own
// pushes left it in; any other is named and left as it is. A tip holding
// none of a's withdrawals still to settle is read for a change of state its
// withdrawer did not push only while it is recent (lostWindow), as the walk
// for lost withdrawals is. When t is not yet a tombstone (the
// tag ended it), a tombstone over t, every withdrawal as it stands, is pushed
// first, the compare-and-swap that claims the end: a copy whose push loses
// says nothing, and the next act reads the tip the other pushed. A tip that is
// already a tombstone is not claimed again: every re-arm is pinned to its
// head and reads the forge first, so a second copy of the same person that
// settles it at the same moment finds each one armed, which is harmless. Then
// each is settled, and a tombstone recording the new states is pushed only
// when a state changed, so a re-arm the forge keeps refusing writes nothing
// to origin until it is settled. A push of that tombstone that fails leaves
// what was re-armed armed, and the next act finds it armed and records it.
func settleTip(a actor, tree, id string, t tip, own bool) (s Settle, ok bool) {
	if t.Unreadable != "" || len(t.Withdrawn) == 0 {
		return Settle{}, false
	}
	pending := slices.ContainsFunc(t.Withdrawn, func(w Withdrawal) bool { return w.unsettled() && sameHolder(w.By, a.holder) })
	if !pending && a.now.Sub(t.ClaimedAt) > lostWindow {
		return Settle{}, false
	}
	tagged, err := taggedOnOrigin(a.root, id)
	if err != nil || !releaseEnded(t, tagged) {
		return Settle{}, false
	}
	a.subject = id
	s = Settle{Subject: id, Settled: []Withdrawal{}}
	links, complete, err := releaseChain(a.root, id, t.Commit, tree)
	if err != nil {
		if !pending {
			return Settle{}, false
		}
		s.Failed = "could not read the history of " + id + "'s branch to tell whose its withdrawals are (" + redacted(err) + "), so none was re-armed"
		return s, true
	}
	k := &asker{c: pushCheck{a.root, id, []string{t.Commit}}, links: links}
	v := k.vouch(complete, t.Withdrawn, a.holder, Withdrawal.unsettled)
	if len(v.Due) > 0 && !tagged {
		owner := ""
		if own {
			owner = a.holder
		}
		switch verdict, holder, why := k.endedBy(owner); verdict {
		case pushedByThem:
		case pushNotVerified:
			v.Unverified, v.Why, v.Due = append(v.Unverified, v.Due...), why, nil
		default:
			s.NotEnded, v.Due = holder, nil
		}
	}
	k.save()
	s.noted(v)
	shown := s.shownOnce(a.root, id, t.Commit)
	if len(v.Due) == 0 {
		shown()
		return s, s.said()
	}
	if why, _ := loginMismatch(a.root, a.holder); why != "" {
		s.Failed = why
		shown()
		return s, true
	}
	base, acted := t.Commit, v.actedOn(t.Withdrawn)
	if !t.message.released() {
		claimed, err := a.pushTombstone(tree, id, t.Commit, acted)
		if err != nil {
			return Settle{}, false
		}
		base, s.Commit = claimed, claimed
	}
	shown()
	ws, settled, changed, fails, why := rearmMine(a, acted, func(w Withdrawal) bool { _, ok := entryIn(v.Due, w); return ok })
	s.Settled, s.Failures, s.Failed = append(s.Settled, settled...), fails, why
	if !changed {
		return s, true
	}
	final, err := a.pushTombstone(tree, id, base, ws)
	if err != nil {
		s.Failed = fmt.Sprintf("settled %s, but the tombstone recording it was not pushed (%s); the next act of a copy of yours finds them armed and records them",
			strings.Join(numbers(settled), ", "), redacted(err))
		return s, true
	}
	s.Commit = final
	s.Failures = append(s.Failures, commentSettled(a, settled)...)
	return s, true
}

// commentSettled posts the comment for each of settled, once the tombstone
// recording their new states has landed (mergeForge.comment), and returns the
// comments that failed.
func commentSettled(a actor, settled []Withdrawal) []ForgeFailure {
	var fails []ForgeFailure
	for _, w := range settled {
		if f := releaseForge.comment(a.root, a.subject, w, ""); f != nil {
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
// when this copy's person and key are known without asking the network.
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

// admitVouched refuses an --admit of w, read on the tip at commit, unless the
// commit that first recorded it was pushed by its withdrawer, as the forge's
// record of pushes says (vouch): an admission arms the pull request under
// the cutter's account, as a re-arm does. Nothing is pushed on a refusal.
func (p *prepared) admitVouched(tree, commit string, w Withdrawal) error {
	links, complete, err := releaseChain(p.root, p.id, commit, tree)
	if err != nil {
		return refuse("#%d was not admitted: the history of %s's branch could not be read (%s); %s", w.Number, p.id, redacted(err), nothingPushed)
	}
	v := vouch(links, complete, []Withdrawal{w}, w.By, pushCheck{p.root, p.id, []string{commit}}, Withdrawal.unsettled)
	switch {
	case len(v.Due) == 1:
		return nil
	case len(v.Unverified) == 1:
		return refuse("#%d was not admitted: %s, and an admission arms only a withdrawal the forge's record shows its withdrawer recorded; "+
			"run --admit %d again in a minute, once the forge's record has caught up; %s", w.Number, v.Why, w.Number, nothingPushed)
	case len(v.AppPushed) == 1:
		return refuse("#%d was not admitted: an app pushed the commit that recorded its withdrawal, which says nothing of which person sent it; "+
			"arm it by hand if it belongs in the cut; %s", w.Number, nothingPushed)
	}
	return refuse("#%d was not admitted: no reservation of %s's on %s's branch, pushed by %s, recorded its withdrawal first, so it is not acted on; "+
		"check the pull request by hand; %s", w.Number, w.By, p.id, w.By, nothingPushed)
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
	if why, _ := loginMismatch(p.root, p.holder); why != "" {
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

// settleOwnRelease settles, after --release pushed the tombstone sha over
// the cutting copy's reservation (parent) carrying ws as they stood, every
// one of this person's still to settle that the branch's history vouches for
// and the forge's record says they pushed (vouch), and pushes a second
// tombstone when any state changed. It returns the settle, the commit last
// pushed and the withdrawals it records.
func (p *prepared) settleOwnRelease(tree, sha, parent string, ws []Withdrawal) (Settle, string, []Withdrawal) {
	a := p.actor()
	s := Settle{Subject: p.id, Settled: []Withdrawal{}, Commit: sha}
	links, complete, err := releaseChain(p.root, p.id, sha, tree)
	if err != nil {
		s.Failed = "could not read the history of " + p.id + "'s branch to tell whose its withdrawals are (" + redacted(err) +
			"), so none was re-armed; the next act of a copy of yours tries again"
		return s, sha, ws
	}
	v := vouch(links, complete, ws, p.holder, pushCheck{p.root, p.id, []string{parent, sha}}, Withdrawal.unsettled)
	s.noted(v)
	s.shownOnce(p.root, p.id, sha)()
	out, settled, changed, fails, why := rearmMine(a, v.actedOn(ws), func(w Withdrawal) bool { _, ok := entryIn(v.Due, w); return ok })
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
	s.Failures = append(s.Failures, commentSettled(a, settled)...)
	return s, final, out
}

// stillBeingCut reports whether r, a reservation as the view lists it, is a
// release still being cut: not ended (no tombstone, no tag), and either live
// or, lapsed, still holding a withdrawal it has not settled (withdrawn or
// admitted). A lapse ends nothing (dcn-2610230328220298), so a lapsed release
// with held merges blocks a second release as a live one does, and is what
// the launch check names as lapsed; one that lapsed holding nothing is
// abandoned, and neither blocks nor is named.
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
// whether it has ended (dcn-2610230328220298), by the rule settleTip holds a
// tombstone to: a release tombstone ends it only when the forge's record of
// pushes says the holder of the reservation it was written over pushed it
// (endedBy), or when its tag is on origin. Over one that ends nothing, cut is
// that reservation beneath, judged at now: still being cut (stillBeingCut)
// while it is live or holds a withdrawal it has not settled, and notEnded is
// true. why, set, is what kept the record (or the branch, or the tags) from
// settling it, which is read as not ended too; cut is then r itself when the
// reservation beneath does not read. A tombstone with no reservation beneath
// it, or over one holding no withdrawal it has not settled, ends it: a second
// release can take nothing from a cut that holds nothing withdrawn or
// admitted, and the record is not asked. Anything else is r as listed.
// reading is true for a caller that only reads (peers, the launch reading),
// which may take the end from a verdict kept for the tip; the second-release
// refusal asks the record every time.
func asCut(root string, r Reservation, now time.Time, reading bool) (cut Reservation, notEnded bool, why string) {
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
	i := 0
	for i < len(links) && links[i].readable && links[i].m.released() {
		i++
	}
	switch {
	case i == len(links) && complete:
		return r, false, ""
	case i == len(links) || !links[i].readable:
		return r, true, "the reservation its tombstone ended is not within the commits read, or does not read"
	}
	b := links[i].m
	v := lease.Judge(lease.Stamp{ClaimedAt: b.ClaimedAt, ExpiresAt: b.ExpiresAt}, lease.Default, now)
	cut = r
	cut.Holder, cut.Footprint, cut.ClaimedAt, cut.EffectiveExpiry = b.Holder, b.Footprint, b.ClaimedAt, v.EffectiveExpiry
	cut.Withdrawn, cut.Commit, cut.Stage = b.Withdrawn, links[i].commit, b.Stage
	cut.ReleasedBy, cut.ReleasedAt, cut.HandedBack, cut.Copy = "", time.Time{}, "", ""
	if own := ownCopyKey(root); own != "" && b.Copy == own {
		cut.Copy = CopyThis
	}
	cut.State = StateExpired
	if v.Live {
		cut.State = StateLive
	}
	if !stillBeingCut(cut) || !slices.ContainsFunc(cut.Withdrawn, Withdrawal.unsettled) {
		return r, false, ""
	}
	k := &asker{c: pushCheck{root, r.Record, []string{r.Commit}}, links: links, reading: reading}
	defer k.save()
	switch verdict, _, why := k.endedBy(""); verdict {
	case pushedByThem:
		return r, false, ""
	case pushNotVerified:
		return cut, true, why
	}
	return cut, true, ""
}

// otherLiveRelease refuses, with a back-off, a release reserved anew while
// another release is still being cut (stillBeingCut), as last fetched and
// read as the hold reads an end (asCut): one release is cut at a time, and a
// second would withdraw the merges the first admitted. A tombstone whose end
// the forge's record of pushes does not settle is read as ending nothing, and
// the back-off says why. A view of the reservations that does not read
// refuses: without it no other release can be ruled out.
func otherLiveRelease(root, id string, now time.Time) error {
	snap, err := viewOf(root, now, true, isRelease)
	if err != nil {
		return refuse("could not tell whether another release is being cut: %s; %s", err.Error(), nothingPushed)
	}
	for _, r := range snap.Reservations {
		if r.Record == id {
			continue
		}
		r, notEnded, why := asCut(root, r, now, false)
		if why != "" {
			return backOff("%s's tip is a tombstone, and whether it ends the release is not settled: %s; one release is cut at a time, so it is read as still being cut: "+
				"wait for it to end (its tag on %s, or its holder's --release); %s", r.Record, why, Remote, nothingPushed)
		}
		if !stillBeingCut(r) {
			continue
		}
		over := ""
		if notEnded {
			over = fmt.Sprintf(" (its tip is a tombstone %s did not push, as the forge's record of pushes says, which ends nothing)", r.Holder)
		}
		if r.State == StateLive {
			return backOff("%s is being cut by %s until %s%s, and one release is cut at a time: a second reservation would withdraw the merges the first admitted; "+
				"wait for it to end (its tag on %s, or its holder's --release); %s", r.Record, r.Holder, stamp(r.EffectiveExpiry), over, Remote, nothingPushed)
		}
		return backOff("%s is still being cut by %s: its lease lapsed at %s%s, but it still holds merges withdrawn or admitted, and one release is cut at a time; "+
			"it ends when %s renews it (abcd reserve %s) and finishes the cut, or ends it (abcd reserve %s --release); %s",
			r.Record, r.Holder, stamp(r.EffectiveExpiry), over, r.Holder, r.Record, r.Record, nothingPushed)
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
// its lease lapsed before it ended, which ends nothing
// (dcn-2610230328220298); on one ended (its tag on origin or any
// tombstone), those still to re-arm, split as an act's settle splits them
// (vouch): the reader's own that the branch's history vouches for and the
// forge's record of pushes says they pushed (Due, which the reader's next act
// re-arms); the reader's own that no reservation of theirs recorded first, or
// whose first record another account pushed (Unrecorded), or an app pushed
// (AppPushed), none of which anything re-arms; the reader's own the record
// does not settle yet (Unverified, with UnverifiedWhy, which a later act
// asks about again); and another person's (Stranded, which only that person
// re-arms); and those its last tombstone skipped for their owners (moved,
// closed, mismatched, needing a manual merge, or refused). A merged one is
// not listed: it needs nothing more.
type ReleaseWithdrawals struct {
	Subject    string       `json:"subject"`
	Ended      bool         `json:"ended"`
	Lapsed     bool         `json:"lapsed,omitempty"`
	LapsedAt   time.Time    `json:"lapsed_at,omitempty"`
	Holder     string       `json:"holder,omitempty"`
	Held       []Withdrawal `json:"held,omitempty"`
	Admitted   []Withdrawal `json:"admitted,omitempty"`
	Due        []Withdrawal `json:"due,omitempty"`
	Unrecorded []Withdrawal `json:"unrecorded,omitempty"`
	AppPushed  []Withdrawal `json:"app_pushed,omitempty"`
	Unverified []Withdrawal `json:"unverified,omitempty"`
	// UnverifiedWhy is why the record does not settle Unverified yet.
	UnverifiedWhy string       `json:"unverified_why,omitempty"`
	Stranded      []Withdrawal `json:"stranded,omitempty"`
	Skipped       []Withdrawal `json:"skipped,omitempty"`
	// NotEnded, when set, is the holder of the reservation the release's
	// tombstone was written over, which the forge's record of pushes does not
	// show they pushed: it ends nothing, so nothing of Due is re-armed yet.
	NotEnded string `json:"not_ended,omitempty"`
}

// Lines is each sentence the listing shows for the release's withdrawals.
func (r ReleaseWithdrawals) Lines() []string {
	var out []string
	if r.Lapsed {
		out = append(out, fmt.Sprintf("%s's lease lapsed at %s before the release ended: it holds no merge now, and nothing re-arms what it withdrew; "+
			"%s renews it from the cutting copy (abcd reserve %s), or ends it with --release, after --take-over from another copy of theirs",
			r.Subject, stamp(r.LapsedAt), r.Holder, r.Subject))
	}
	if len(r.Held) > 0 {
		out = append(out, fmt.Sprintf("%s holds %s withdrawn until it ends: the cutting copy admits one with --admit <pr>",
			r.Subject, strings.Join(numbers(r.Held), ", ")))
	}
	if len(r.Admitted) > 0 {
		out = append(out, fmt.Sprintf("%s admitted %s", r.Subject, strings.Join(numbers(r.Admitted), ", ")))
	}
	if r.NotEnded != "" {
		out = append(out, notEndedLine(r.Subject, r.NotEnded))
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
// GitHub username, "" when it is not known, which owns none). The reader's
// own on an ended release are read against the branch's history at root and
// the forge's record of pushes, as an act's settle reads them (vouch), so one
// nothing re-arms never reads as due. The record is asked at most once for
// each such release, and not at all for a withdrawal whose verdict an act
// already kept for the tip (an app's push, or another account's).
func (s Snapshot) ReleaseWithdrawals(root, reader string, now time.Time) []ReleaseWithdrawals {
	var out []ReleaseWithdrawals
	for _, r := range s.Reservations {
		if !isRelease(r.Record) || len(r.Withdrawn) == 0 || r.State == StateUnreadable {
			continue
		}
		cut, notEnded, why := asCut(root, r, now, true)
		r = cut
		rw := ReleaseWithdrawals{Subject: r.Record, Holder: r.Holder, Ended: r.State == StateReleased && !notEnded}
		if notEnded {
			if why == "" {
				rw.NotEnded = r.Holder
			} else {
				rw.UnverifiedWhy = why
			}
		}
		var mine []Withdrawal
		for _, w := range r.Withdrawn {
			switch {
			case w.State == WithdrawnMerged:
			case skippedForOwner(w.State):
				rw.Skipped = append(rw.Skipped, w)
			case w.State == WithdrawnAdmitted && !rw.Ended:
				rw.Admitted = append(rw.Admitted, w)
			case !w.Pending():
			case !rw.Ended:
				rw.Held = append(rw.Held, w)
			case reader != "" && sameHolder(w.By, reader):
				mine = append(mine, w)
			default:
				rw.Stranded = append(rw.Stranded, w)
			}
		}
		if len(mine) > 0 {
			v, ended := vouchedForReader(root, r, reader, mine)
			rw.Due, rw.Unrecorded, rw.AppPushed, rw.Unverified, rw.UnverifiedWhy, rw.NotEnded = v.Due, v.Unrecorded, v.AppPushed, v.Unverified, v.Why, ended
		}
		if r.State == StateExpired && stillBeingCut(r) {
			rw.Lapsed, rw.LapsedAt = true, r.EffectiveExpiry
		}
		if len(rw.Held)+len(rw.Admitted)+len(rw.Due)+len(rw.Unrecorded)+len(rw.AppPushed)+len(rw.Unverified)+len(rw.Stranded)+len(rw.Skipped) > 0 || rw.Lapsed || rw.NotEnded != "" {
			out = append(out, rw)
		}
	}
	return out
}

// vouchedForReader is vouching for ws, the reader's withdrawals on r's
// ended tip (vouch); all of them are taken as not recorded when the branch's
// history cannot be read. When the release is not tagged, the tombstone is
// held to its holder's push as an act holds it (endedBy): one they did not
// push ends nothing, and notEnded names that holder, with none due.
func vouchedForReader(root string, r Reservation, reader string, ws []Withdrawal) (v vouching, notEnded string) {
	tree, err := emptyTree(root)
	if err != nil {
		return vouching{Unrecorded: ws}, ""
	}
	links, complete, err := releaseChain(root, r.Record, r.Commit, tree)
	if err != nil {
		return vouching{Unrecorded: ws}, ""
	}
	k := &asker{c: pushCheck{root, r.Record, []string{r.Commit}}, links: links, reading: true}
	defer k.save()
	v = k.vouch(complete, ws, reader, Withdrawal.Pending)
	if len(v.Due) == 0 {
		return v, ""
	}
	if tagged, err := taggedOnOrigin(root, r.Record); err == nil && tagged {
		return v, ""
	}
	switch verdict, holder, why := k.endedBy(""); verdict {
	case pushedByThem:
	case pushNotVerified:
		v.Unverified, v.Why, v.Due = append(v.Unverified, v.Due...), why, nil
	default:
		v.Due, notEnded = nil, holder
	}
	return v, notEnded
}
