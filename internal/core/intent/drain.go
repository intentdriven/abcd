package intent

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// drain.go — the queue behind `abcd intent audit --owed` (itd-53,
// spc-2609211930059886).
//
// The owed set is the one reader's (Reviews, owed.go): nothing here re-reads a
// marker or re-decides what is owed. This file adds what the drain needs on
// top of the listing — an order and a cap (OwedQueue, which never writes) —
// and one step that re-emits the head's request through the single audit's own
// emit (NextOwedAudit). Running the audits is the host's: the plugin page takes
// the queue one entry at a time through the request/ingest pair a single audit
// uses, and a host without the page drives the same pair by hand.
//
// Oldest means the day the intent entered shipped/, because that is the day
// its review fell owed. The day is a fact about git history, which this
// package does not read: the caller supplies it (the front door passes the
// site package's one history walk, whose EnteredBucket is exactly this). An
// intent with no day — shipped in the working tree and not yet committed, or a
// tree with no history at all — is the newest there is, so it sorts last.
// Ties, including a whole undated tree, fall to the order the ids were minted
// in: numerically, so itd-9 precedes itd-10 and every ordinal id precedes
// every timestamp id (adr-45).

// ShippedOn reports the day (YYYY-MM-DD) the intent file at a repo-relative
// path entered the bucket it sits in, or "" when it is not known.
type ShippedOn func(relPath string) string

// QueuedReview is one owed review in the drain queue: the reader's entry plus
// the day its intent shipped ("" when there is none to give) and which of the
// three facts about that day holds (ShippedState).
type QueuedReview struct {
	ReviewEntry
	Shipped      string `json:"shipped,omitempty"`
	ShippedState string `json:"shipped_state"`
	// EmitError is why the drain step could not emit this entry's request (a
	// malformed spec_id, an unreadable file); the step moves on to the next
	// entry, so one bad record never blocks the drain. "" when it was not
	// tried or was emitted.
	EmitError string `json:"emit_error,omitempty"`
}

// The states of a queued review's shipped day. An undated entry is either
// uncommitted (the history was read and does not hold it yet) or unknown (no
// history was read at all); the two sort alike and mean different things.
const (
	ShippedDated       = "dated"
	ShippedUncommitted = "uncommitted"
	ShippedUnknown     = "unknown"
)

// ReviewQueue is the owed reviews, oldest shipped first, capped at Max.
// Owed is the whole owed total before the cap; Remaining is how many the cap
// left out. Max 0 is no cap.
type ReviewQueue struct {
	Queue     []QueuedReview `json:"queue"`
	Owed      int            `json:"owed"`
	Max       int            `json:"max"`
	Remaining int            `json:"remaining"`
}

// OwedQueue orders the owed fidelity reviews oldest shipped first and caps the
// list at max (0: no cap; negative: refused). shippedOn may be nil, which
// leaves every day unknown and the queue in mint order. It never writes.
func OwedQueue(repoRoot string, max int, shippedOn ShippedOn) (ReviewQueue, error) {
	if err := CheckOwedCap(max); err != nil {
		return ReviewQueue{}, err
	}
	corpus, err := Load(repoRoot)
	if err != nil {
		return ReviewQueue{}, err
	}
	listing, err := reviewsOf(repoRoot, corpus)
	if err != nil {
		return ReviewQueue{}, err
	}
	paths := make(map[string]string, len(corpus.Intents))
	for _, it := range corpus.Intents {
		if it.Bucket == BucketShipped {
			paths[it.ID] = filepath.ToSlash(it.Path)
		}
	}

	q := ReviewQueue{Queue: []QueuedReview{}, Owed: listing.Owed, Max: max}
	for _, e := range listing.Entries {
		if !e.IsOwed() {
			continue
		}
		day, state := "", ShippedUnknown
		if shippedOn != nil {
			day, state = shippedOn(paths[e.IntentID]), ShippedDated
			if day == "" {
				state = ShippedUncommitted
			}
		}
		q.Queue = append(q.Queue, QueuedReview{ReviewEntry: e, Shipped: day, ShippedState: state})
	}
	sort.SliceStable(q.Queue, func(i, j int) bool {
		a, b := q.Queue[i], q.Queue[j]
		if a.Shipped != b.Shipped {
			switch {
			case a.Shipped == "":
				return false
			case b.Shipped == "":
				return true
			}
			return a.Shipped < b.Shipped
		}
		return mintedBefore(a.IntentID, b.IntentID)
	})
	if max > 0 && len(q.Queue) > max {
		q.Queue = q.Queue[:max]
	}
	q.Remaining = q.Owed - len(q.Queue)
	return q, nil
}

// CheckOwedCap refuses a negative cap (0 is no cap). A front door calls it
// before any costlier work, such as the history walk that supplies ShippedOn.
func CheckOwedCap(max int) error {
	if max < 0 {
		return fmt.Errorf("intent: --max must be zero (no cap) or a positive count, got %d", max)
	}
	return nil
}

// DrainStep is one step of the drain: the queue, and the request for the
// first entry that could be emitted, re-emitted so a host can hand it to the
// auditor. Next is nil when nothing is owed, or when no listed entry could be
// emitted (each then carries its EmitError).
type DrainStep struct {
	ReviewQueue
	Next *AuditEmitResult `json:"next,omitempty"`
}

// NextOwedAudit is OwedQueue plus the single audit's own emit on the head of
// the queue: the head's request is (re-)written and named, and a markerless
// head has its receipt minted, which the queue then reports. Only one entry is
// emitted; nothing runs a reviewer. The next step, after the host ingests that
// entry's verdict, finds the queue one shorter. A head whose emit fails keeps
// its error on its own entry and the next entry is tried, so a record that
// needs a hand fix stays listed without blocking the ones behind it. opts is
// what the front door adds to the request, as it adds it to a single audit's
// (the routing section).
func NextOwedAudit(repoRoot string, max int, shippedOn ShippedOn, opts AuditEmitOptions) (DrainStep, error) {
	q, err := OwedQueue(repoRoot, max, shippedOn)
	if err != nil {
		return DrainStep{}, err
	}
	step := DrainStep{ReviewQueue: q}
	for i := range step.Queue {
		e := &step.Queue[i]
		res, err := ReEmitAuditWith(repoRoot, e.IntentID, opts)
		if err != nil {
			// A failed emit parks no stub, so the row keeps the receipt state
			// the reader gave it; an already-parked receipt the emit named is
			// reported as the OWED receipt it is.
			e.EmitError = err.Error()
			if res.Status == "already_owed" {
				e.State, e.ReceiptID = ReviewOwed, res.ReceiptID
			}
			continue
		}
		step.Next = &res
		e.State, e.ReceiptID = ReviewOwed, res.ReceiptID
		break
	}
	return step, nil
}

// mintedBefore orders two intent ids by the number they carry, compared as a
// decimal string so a timestamp id never overflows: fewer digits is smaller,
// then the digits themselves. Leading zeros are trimmed first, so itd-007 and
// itd-7 compare as one number (recordid.SameID's canonical form).
func mintedBefore(a, b string) bool {
	na, nb := idDigits(a), idDigits(b)
	if len(na) != len(nb) {
		return len(na) < len(nb)
	}
	if na != nb {
		return na < nb
	}
	return a < b
}

func idDigits(id string) string {
	n := strings.TrimLeft(strings.TrimPrefix(id, "itd-"), "0")
	if n == "" {
		return "0"
	}
	return n
}
