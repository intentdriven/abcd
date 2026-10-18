package autonomy

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"time"
)

// OwedKind is one of the four kinds of owed work, the reservation kind split
// in two (a renewal due, and work no pull request carries) and the pull
// request kind split in two (failing, behind).
type OwedKind string

const (
	// OwedRenewal is a live reservation under this copy's key whose renewal
	// is due. It is never capped.
	OwedRenewal OwedKind = "renewal"
	// OwedFailing is an open pull request of the session's own failing its
	// checks.
	OwedFailing OwedKind = "failing"
	// OwedBehind is an open pull request of the session's own behind its base
	// (or conflicting).
	OwedBehind OwedKind = "behind"
	// OwedStacked is an own branch stacked on a change still open, while the
	// clearing rule owes it.
	OwedStacked OwedKind = "stacked"
	// OwedUnpushed is an own branch with commits not pushed.
	OwedUnpushed OwedKind = "unpushed"
	// OwedUncarried is a live reservation under this copy's key whose work no
	// open pull request carries yet.
	OwedUncarried OwedKind = "uncarried"
)

// OwedKinds returns the kinds in the order the next item is chosen.
func OwedKinds() []OwedKind {
	return []OwedKind{OwedRenewal, OwedFailing, OwedBehind, OwedStacked, OwedUnpushed, OwedUncarried}
}

func kindRank(k OwedKind) int {
	for i, o := range OwedKinds() {
		if o == k {
			return i
		}
	}
	return len(OwedKinds())
}

// OwedItem is one owed item.
type OwedItem struct {
	Kind OwedKind `json:"kind"`
	// Capped marks an item the session can act on itself, which a qualifying
	// wake releases at most twice in a row (dcn-2610181035361997).
	Capped bool `json:"capped"`
	// Branch is the branch the item is about, when it is about one.
	Branch string `json:"branch,omitempty"`
	// Record is the reservation's record id, when the item is about one.
	Record string `json:"record,omitempty"`
	// PR is the pull request's number, when the item has one.
	PR int `json:"pr,omitempty"`
	// Head is the branch's head commit, when known.
	Head string `json:"head,omitempty"`
	// Reason says, in plain words, why the item is owed and what to do. It is
	// not part of the item's identity.
	Reason string `json:"reason,omitempty"`
}

func (i OwedItem) key() string {
	return strings.Join([]string{string(i.Kind), i.Branch, i.Record, strconv.Itoa(i.PR), i.Head}, "\x00")
}

// Owed is the owed set a stop check judges.
type Owed struct {
	Items []OwedItem `json:"items"`
	// Notes say what was not read and how the set was judged instead, such
	// as pull requests not read when the forge listing failed.
	Notes []string `json:"notes,omitempty"`
	// NextRenewal is the earliest renewal due among the live reservations
	// under this copy's key, due or not yet; zero when there is none. The
	// stop verdict's deadline is it less lease.Skew, when that is sooner than
	// the ceiling (dcn-2610181003455731).
	NextRenewal time.Time `json:"next_renewal,omitempty"`
	// HandedBack are the items a queued step handed back this check, taken
	// out of Items by the reader (dcn-2610181035355741). The verdict accepts
	// only an unpushed branch's item, with its head, under --hold-pushes,
	// and puts any other back among the owed items.
	HandedBack []OwedItem `json:"handed_back,omitempty"`
}

// Empty reports whether nothing is owed.
func (o Owed) Empty() bool { return len(o.Items) == 0 }

// HasCapped reports whether any owed item is capped.
func (o Owed) HasCapped() bool {
	for _, i := range o.Items {
		if i.Capped {
			return true
		}
	}
	return false
}

// Next is the item the session is handed: the first by OwedKinds' order, the
// items of one kind in the order they were given.
func (o Owed) Next() (OwedItem, bool) {
	if len(o.Items) == 0 {
		return OwedItem{}, false
	}
	best := 0
	for i := 1; i < len(o.Items); i++ {
		if kindRank(o.Items[i].Kind) < kindRank(o.Items[best].Kind) {
			best = i
		}
	}
	return o.Items[best], true
}

// Fingerprint identifies the owed set by its items' kind, branch, record,
// pull request and head, whatever their order and reasons.
func (o Owed) Fingerprint() string { return fingerprint(o.Items, false) }

// CappedFingerprint is Fingerprint over the capped items alone, so a renewal
// falling due or done does not change it.
func (o Owed) CappedFingerprint() string { return fingerprint(o.Items, true) }

func fingerprint(items []OwedItem, cappedOnly bool) string {
	keys := make([]string, 0, len(items))
	for _, i := range items {
		if cappedOnly && !i.Capped {
			continue
		}
		keys = append(keys, i.key())
	}
	sort.Strings(keys)
	sum := sha256.Sum256([]byte(strings.Join(keys, "\n")))
	return hex.EncodeToString(sum[:16])
}
