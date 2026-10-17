package reservesweep

import (
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// Why a branch is spent.
const (
	// SpentTerminal: the record is resolved, won't-fixed, shipped or
	// superseded on origin's default branch, so internal/core/claim already
	// counts the reservation as released whatever its tip says.
	SpentTerminal = "terminal"
	// SpentReleased: the tip is a plain tombstone and no hand-back stands
	// anywhere on the branch.
	SpentReleased = "released"
	// SpentLapsed: the tip is a reservation whose lease lapsed, over a plain
	// tombstone, and no hand-back stands anywhere on the branch.
	SpentLapsed = "lapsed"
	// SpentUnreleased: every commit on the branch is a reservation and the
	// tip's lease lapsed, so any session may already take it over and the
	// branch protects nothing.
	SpentUnreleased = "lapsed_unreleased"
)

// Why a branch is kept.
const (
	KeptUnreadable = "unreadable"
	KeptRecent     = "recent"
	KeptUnknown    = "unknown_kind"
	KeptHandBack   = "hand_back"
	KeptHistory    = "history_unread"
)

// facts is what the sweep read about one reservation branch.
type facts struct {
	// Chain is the branch's first-parent line, tip first.
	Chain []commitInfo
	// ChainWhole is true when the walk reached the root commit; false when
	// it stopped at the bound or a commit it could not read.
	ChainWhole bool
	// Terminal is true when the record sits only in terminal folders on
	// origin's default branch as just fetched; always false when that branch
	// could not be read.
	Terminal bool
	// Unreadable is the claim reader's reason the tip cannot be read.
	Unreadable string
	// EffectiveExpiry is the claim reader's lease.Judge verdict on the tip:
	// when it stops holding.
	EffectiveExpiry time.Time
}

// verdict is one branch judged.
type verdict struct {
	Spent  bool
	Reason string
	// SettledAt is when the tip last held or was written: the latest of its
	// commit date, its own times and its effective expiry past the take-over
	// margin. The branch is spent no earlier than SettledAt plus the age.
	SettledAt time.Time
}

// judge decides whether a branch is spent at now, untouched for olderThan.
// It fails closed: anything it cannot classify keeps the branch.
//
// A branch is spent only when its tip has settled, every time it carries,
// its lease included past claim.TakeoverMargin, olderThan before now, and:
//
//   - (a) the record is terminal on origin's default branch and the tip is a
//     commit the sweep can classify; or
//   - (b) every commit on its first-parent line, read to the root, is a
//     reservation or a plain tombstone, and the tip is a plain tombstone or a
//     lapsed reservation. A reservation that lapsed without ever being
//     released counts: any session may already take it over, so the branch
//     protects nothing once the lapse is older than the age.
//
// So a hand-back anywhere on a branch whose record is still open keeps it,
// and so does a commit of any kind the sweep does not know.
func judge(f facts, now time.Time, olderThan time.Duration) verdict {
	if f.Unreadable != "" || len(f.Chain) == 0 {
		return verdict{Reason: KeptUnreadable}
	}
	tip := f.Chain[0]
	v := verdict{SettledAt: latest(tip.CommittedAt, tip.ClaimedAt, tip.ReleasedAt, f.EffectiveExpiry.Add(claim.TakeoverMargin))}
	if tip.Kind == KindUnknown {
		v.Reason = KeptUnknown
		return v
	}
	if now.Before(v.SettledAt.Add(olderThan)) {
		v.Reason = KeptRecent
		return v
	}
	if f.Terminal {
		v.Spent, v.Reason = true, SpentTerminal
		return v
	}
	released := false
	for _, c := range f.Chain {
		switch c.Kind {
		case KindHandBack:
			v.Reason = KeptHandBack
			return v
		case KindUnknown:
			v.Reason = KeptUnknown
			return v
		case KindPlainTombstone:
			released = true
		}
	}
	switch {
	case !f.ChainWhole:
		v.Reason = KeptHistory
	case !released:
		v.Spent, v.Reason = true, SpentUnreleased
	case tip.Kind == KindPlainTombstone:
		v.Spent, v.Reason = true, SpentReleased
	default:
		v.Spent, v.Reason = true, SpentLapsed
	}
	return v
}

// latest is the latest of ts.
func latest(ts ...time.Time) time.Time {
	var out time.Time
	for _, t := range ts {
		if t.After(out) {
			out = t
		}
	}
	return out
}
