package claim

// withdraw.go is the release hold's withdrawal record (iss-2610222014550692;
// dcn-2610222219289525, dcn-2610222219297651, dcn-2610222219297487). A release
// reservation holds only the merges armed after it, so when one is taken anew
// it also withdraws every merge already armed or queued on origin's default
// branch, and records each pull request it withdrew on the reservation, one
// Abcd-Withdrawn trailer a pull request: its number, its node id, the form it
// was armed in, its head when withdrawn, whose withdrawal it was, and where
// it stands. Every field is fixed-form (a number, an object name, a word from
// a closed vocabulary, a holder id), so no free text reaches the
// remote (dcn-2610150000000001), and the reader refuses any other shape: a
// release tip carrying one reads as unreadable, as a bad hand-back does.

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// keyWithdrawn is the trailer that records one withdrawn pull request on a
// release reservation, its renewals and its tombstone.
const keyWithdrawn = "Abcd-Withdrawn"

// The forms a withdrawn merge was armed in, and is re-armed in
// (dcn-2610222219297487): a merge-queue entry, or auto-merge with its merge
// method.
const (
	FormQueue      = "queue"
	FormAutoMerge  = "auto:merge"
	FormAutoSquash = "auto:squash"
	FormAutoRebase = "auto:rebase"
)

// preIDMarker is the one holder name a withdrawal carries in place of the
// login of another holder who recorded it before the holder id
// (iss-2610290809526033): nothing abcd pushes carries a login, so an act that
// writes such an entry, still to settle, into a new commit writes it under
// this marker, meaning "a holder from before the holder id". It is neither a
// holder id nor a GitHub username (the underscore occurs in no username, and
// it is no h_ id), so it is no reader's own: no first record, own record,
// vouching or lost-withdrawal walk counts it as anyone's, and no copy re-arms,
// withdraws again or admits it. The act that ends the release settles it as
// left to be re-armed by hand (WithdrawnManual, settledByHand). A release
// written by abcd v0.15.x ignores the withdrawal trailer, and one before the
// holder id reads every commit naming a holder id, which every commit
// carrying the marker is, as unreadable, so no older reader reads it as
// anyone's either.
const preIDMarker = "pre_id"

// WithdrawalForms is the closed vocabulary of forms.
var WithdrawalForms = []string{FormQueue, FormAutoMerge, FormAutoSquash, FormAutoRebase}

// The states a withdrawn pull request is recorded in. Withdrawn is the one
// still to settle; every other is settled, and no copy acts on it again.
const (
	// WithdrawnPending: taken off the forge's merge path and not yet put back.
	WithdrawnPending = "withdrawn"
	// WithdrawnAdmitted: the cutting copy ruled it in with --admit, and
	// re-armed it then.
	WithdrawnAdmitted = "admitted"
	// WithdrawnRearmed: re-armed, in its own form and pinned to its head, once
	// the release ended.
	WithdrawnRearmed = "rearmed"
	// WithdrawnFoundArmed: already armed or queued again when the re-arm came
	// to it, which counts as done.
	WithdrawnFoundArmed = "armed"
	// WithdrawnMoved, WithdrawnClosed and WithdrawnMerged: skipped, for its
	// owner to re-arm: its head moved since it was withdrawn, or it was closed
	// or merged.
	WithdrawnMoved  = "moved"
	WithdrawnClosed = "closed"
	WithdrawnMerged = "merged"
	// WithdrawnMismatch: skipped, never acted on: the node id recorded for it
	// is not the one the forge answers for its number, so the record is not
	// trusted to name it.
	WithdrawnMismatch = "mismatch"
	// WithdrawnManual: the forge refused its re-arm because the pull request
	// is already mergeable and the repository has no merge queue to put it
	// in ("clean status"); abcd never merges someone else's pull request
	// outright (dcn-2610230328230096), so it needs a manual merge by its
	// owner, and no copy tries again. An entry under preIDMarker is settled
	// in it too, without the forge being asked, when the release ends: no
	// copy re-arms it, so its author re-arms it by hand.
	WithdrawnManual = "manual"
	// WithdrawnRefused: the forge refused its re-arm maxRearmRefusals times
	// for a reason other than clean status, so no copy tries again and its
	// owner re-arms it.
	WithdrawnRefused = "refused"
	// WithdrawnKept: recorded as about to be withdrawn, but the forge refused
	// the withdrawal, so it was left armed as it was. The correction that
	// says so keeps the entry in this state rather than dropping it, so the
	// walk for lost withdrawals never reads it as a record a later commit
	// lost. This person's own renewals carry it, and a live one withdraws it
	// again when the forge still has it armed; a tombstone ends it.
	WithdrawnKept = "kept"
)

// WithdrawalStates is the closed vocabulary of states.
var WithdrawalStates = []string{WithdrawnPending, WithdrawnAdmitted, WithdrawnRearmed, WithdrawnFoundArmed,
	WithdrawnMoved, WithdrawnClosed, WithdrawnMerged, WithdrawnMismatch, WithdrawnManual, WithdrawnRefused, WithdrawnKept}

// skippedForOwner reports whether state is one a re-arm settles a withdrawal in
// without re-arming it, for its owner to act on.
func skippedForOwner(state string) bool {
	return slices.Contains([]string{WithdrawnMoved, WithdrawnClosed, WithdrawnMerged, WithdrawnMismatch, WithdrawnManual, WithdrawnRefused}, state)
}

// maxWithdrawals bounds the withdrawals one release tip records: a writer
// can push a tip with more, and a reader refuses it as unreadable, so a
// carried list never grows without bound (the cap and the outbound policy
// bound each message besides).
const maxWithdrawals = 40

// Withdrawal is one pull request a release reservation withdrew, as its
// trailer records it. Every value is validated data, never a writer's text.
type Withdrawal struct {
	// Number is the pull request's number, ID its node id on the forge.
	Number int    `json:"number"`
	ID     string `json:"id"`
	// Form is how it was armed, one of WithdrawalForms.
	Form string `json:"form"`
	// Head is its head commit when it was withdrawn: every re-arm is pinned
	// to it (expectedHeadOid).
	Head string `json:"head"`
	// By is the holder whose reservation withdrew it: their holder id
	// (holderid.go), or, on a withdrawal recorded before the id, their GitHub
	// username, which a push of abcd's writes as preIDMarker for any holder
	// but the pusher. Only a copy of that person re-arms it
	// (dcn-2610222219297651), and none re-arms one under the marker.
	By string `json:"by"`
	// State is where it stands, one of WithdrawalStates.
	State string `json:"state"`
}

// preID reports whether the withdrawal names its holder by preIDMarker.
func (w Withdrawal) preID() bool { return w.By == preIDMarker }

// Pending reports whether the withdrawal is still to settle.
func (w Withdrawal) Pending() bool { return w.State == WithdrawnPending }

// unsettled reports whether the withdrawal is one a release still being cut
// holds: withdrawn, or admitted into the cut. Both are settled when the
// release ends: a withdrawn one re-armed, an admitted one checked on the
// forge and armed when it is not.
func (w Withdrawal) unsettled() bool {
	return w.State == WithdrawnPending || w.State == WithdrawnAdmitted
}

// trailer is the withdrawal's trailer value: "<number> <id> <form> <head>
// <by> <state>".
func (w Withdrawal) trailer() string {
	return strings.Join([]string{strconv.Itoa(w.Number), w.ID, w.Form, w.Head, w.By, w.State}, " ")
}

// reasonWithdrawn is why a tip whose withdrawals do not read is unreadable.
const reasonWithdrawn = "a pull request it records as withdrawn is not one number, node id, form, head, holder and state, " +
	"each pull request once, at most 40, on a release"

// withdrawnNumberRe is a pull request's number as a trailer carries it: no sign, no
// leading zero, at most nine digits.
var withdrawnNumberRe = regexp.MustCompile(`^[1-9][0-9]{0,8}$`)

// nodeIDRe is a forge node id as a trailer carries it: the forge's global
// ids are base64 or base64url text with a type prefix, and nothing else is
// written.
var nodeIDRe = regexp.MustCompile(`^[A-Za-z0-9_=+/-]{1,100}$`)

// validNodeID reports whether s is a node id a trailer may carry.
func validNodeID(s string) bool { return nodeIDRe.MatchString(s) }

// parseWithdrawal reads one Abcd-Withdrawn value, strictly: six fields
// separated by one space each, every one in its form.
func parseWithdrawal(v string) (Withdrawal, bool) {
	f := strings.Split(v, " ")
	if len(f) != 6 || !withdrawnNumberRe.MatchString(f[0]) || !validNodeID(f[1]) || !slices.Contains(WithdrawalForms, f[2]) ||
		!validBlob(f[3]) || !validHolder(f[4]) && f[4] != preIDMarker || !slices.Contains(WithdrawalStates, f[5]) {
		return Withdrawal{}, false
	}
	n, err := strconv.Atoi(f[0])
	if err != nil {
		return Withdrawal{}, false
	}
	return Withdrawal{Number: n, ID: f[1], Form: f[2], Head: f[3], By: f[4], State: f[5]}, true
}

// validateWithdrawn reads a message's withdrawals into m, which
// validateReservation has filled: none, or each pull request once, on a
// release subject; a reservation that is not released carries only those
// still withdrawn or admitted, and those its own correction kept (a
// withdrawal the forge refused), since only a tombstone settles one. It returns
// reasonWithdrawn for anything else.
func validateWithdrawn(m *message, t trailers) string {
	vals := t.vals[keyWithdrawn]
	if len(vals) == 0 {
		return ""
	}
	if !isRelease(m.Record) || len(vals) > maxWithdrawals {
		return reasonWithdrawn
	}
	seen := map[int]bool{}
	out := make([]Withdrawal, 0, len(vals))
	for _, v := range vals {
		w, ok := parseWithdrawal(v)
		if !ok || seen[w.Number] {
			return reasonWithdrawn
		}
		if !m.released() && !w.unsettled() && w.State != WithdrawnKept {
			return reasonWithdrawn
		}
		seen[w.Number] = true
		out = append(out, w)
	}
	m.Withdrawn = out
	return ""
}

// withdrawalFormOf is the form a pull request armed as the forge reports it
// is withdrawn as: a merge-queue entry when it sits in the queue (an arming
// there is what queued it), else auto-merge with its method, merge when the
// forge names none.
func withdrawalFormOf(queued bool, method string) string {
	if queued {
		return FormQueue
	}
	switch strings.ToUpper(method) {
	case "SQUASH":
		return FormAutoSquash
	case "REBASE":
		return FormAutoRebase
	}
	return FormAutoMerge
}

// mergeMethodOf is the forge's merge method an auto-merge form re-arms with.
func mergeMethodOf(form string) string {
	return strings.ToUpper(strings.TrimPrefix(form, "auto:"))
}

// carried is the withdrawals a new commit on a release subject keeps from the
// tip it is written over: from this person's own reservation (renewed, or
// taken over from their other copy), every one still withdrawn or admitted,
// and every one kept (a withdrawal the forge refused, which a live renewal
// tries again); from any other tip (a tombstone, a lapsed reservation of
// another person's), only those still withdrawn, which nobody has settled
// yet. A settled one stays on the commit that settled it.
func carried(tip []Withdrawal, ownReservation bool) []Withdrawal {
	var out []Withdrawal
	for _, w := range tip {
		if w.Pending() || ownReservation && (w.State == WithdrawnAdmitted || w.State == WithdrawnKept) {
			out = append(out, w)
		}
	}
	return out
}

// settledByHand is ws with each entry under preIDMarker still to settle
// (withdrawn or admitted) settled as WithdrawnManual, for its author to re-arm
// by hand, and those entries as settled: what the act that ends a release
// records, since no copy re-arms one.
func settledByHand(ws []Withdrawal) (out, byHand []Withdrawal) {
	out = slices.Clone(ws)
	for i, w := range out {
		if w.preID() && w.unsettled() {
			out[i].State = WithdrawnManual
			byHand = append(byHand, out[i])
		}
	}
	return out, byHand
}

// withWithdrawal is ws with w in place of the entry for its pull request, or
// appended when ws has none.
func withWithdrawal(ws []Withdrawal, w Withdrawal) []Withdrawal {
	out := slices.Clone(ws)
	for i := range out {
		if out[i].Number == w.Number {
			out[i] = w
			return out
		}
	}
	return append(out, w)
}

// numbers is each withdrawal's pull request number as "#N", in order.
func numbers(ws []Withdrawal) []string {
	out := make([]string, len(ws))
	for i, w := range ws {
		out[i] = "#" + strconv.Itoa(w.Number)
	}
	return out
}
