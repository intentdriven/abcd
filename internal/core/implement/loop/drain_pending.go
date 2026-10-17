package loop

// drain_pending.go is the hand-back a yes routed and could not publish on
// origin (the review of spc-2610161044461448 steps 1 to 18): the hold the yes
// would have released with the hand-back on its tombstone had moved on by the
// move that took the answer (this person renewed or reserved the issue again
// by hand, another took it over, or a person released it), or the drain had
// taken none because this person held the issue live from this copy, or the
// release did not reach origin. A hand-back is a person's to settle by
// changing the record, so none of those may erase it: the drain keeps it
// pending in its state, carries it into the next drain, and publishes it over
// the issue's next release on origin, a tombstone that marks no hand-back, as
// that tombstone's child, fenced on that commit and on the blob the yes was
// judged over (claim.Request.HandBackBlob). A tip already marking a hand-back,
// a record finished or changed on origin's default branch, a record origin
// holds in no one file, and a tombstone the hand-back would take past the cap
// each end it with nothing to publish, and the move says it was dropped and
// why; a default branch of origin that could not be read keeps it pending
// (dcn-2610161826443524, refined by dcn-2610161931130911).

import (
	"slices"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/peers"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// DrainPendingHandBack is a hand-back a yes routed that the move could not
// publish on origin: its issue, its kind, the blob of the issue's record on
// origin's default branch the yes was judged over ("" when it could not be
// read), why it is pending, and when it was routed.
type DrainPendingHandBack struct {
	Issue string    `json:"issue"`
	Kind  string    `json:"kind"`
	Blob  string    `json:"blob,omitempty"`
	Why   string    `json:"why"`
	At    time.Time `json:"at"`
}

// judgedBlob is the blob of issue's record at the commit of origin's default
// branch the move planned from, "" with no origin or when that commit holds
// the record in no one file, or cannot be read.
func judgedBlob(repoRoot, issue string, origin *DrainOrigin) string {
	if origin == nil {
		return ""
	}
	recs, err := peers.RecordsAt(repoRoot, origin.Commit)
	if err != nil || len(recs[issue]) != 1 {
		return ""
	}
	return recs[issue][0].Blob
}

// pend keeps issue's hand-back of kind pending in st, in place of any the
// drain already keeps for it, and returns the route's clause saying so.
func pend(st *DrainState, issue, kind, blob, why string, now time.Time) string {
	st.PendingHandBacks = slices.DeleteFunc(st.PendingHandBacks, func(p DrainPendingHandBack) bool { return p.Issue == issue })
	st.PendingHandBacks = append(st.PendingHandBacks, DrainPendingHandBack{Issue: issue, Kind: kind, Blob: blob, Why: why, At: now})
	return "the hand-back is not yet published on " + claim.Remote + ": " + why +
		"; the drain keeps it pending and publishes it over the issue's next release there"
}

// publishPending publishes, at now, each pending hand-back whose issue's tip
// in view is a release marking no hand-back, and ends each that is no longer
// due; it returns the move's clause for each it ended, saying whether it was
// published or dropped, and why. A pending hand-back whose tip still holds
// (live, lapsed, unreadable, another's) waits on, and so does one whose
// release did not reach origin, or met a default branch of origin that could
// not be read. Without a view it waits too.
func publishPending(repoRoot string, st *DrainState, view *claim.Snapshot, now time.Time) []string {
	if view == nil || len(st.PendingHandBacks) == 0 {
		return nil
	}
	var out []string
	kept := st.PendingHandBacks[:0:0]
	for _, p := range st.PendingHandBacks {
		r, ok := view.Lookup(p.Issue)
		switch {
		case !ok || r.State != claim.StateReleased:
			kept = append(kept, p)
			continue
		case r.ReleasedBy == claim.ReleasedByDefaultBranch:
			out = append(out, p.Issue+" is finished on "+claim.Remote+"'s default branch, so its pending "+p.Kind+" hand-back is no longer due")
		case r.HandedBack != "":
			out = append(out, p.Issue+"'s tip on "+claim.Remote+" already marks a "+r.HandedBack+" hand-back, so its pending "+p.Kind+" hand-back is not published again")
		default:
			rel := releaseAt(repoRoot, p.Issue, r.Commit, p.Kind, p.Blob, now)
			if rel.pending != "" {
				kept = append(kept, p)
				continue
			}
			// A release that pushed no hand-back ends it only when it is no
			// longer due (the record changed, origin holds no one record of
			// it, the cap, or a tip that already marks one), and says so.
			c := "published the hand-back of " + p.Issue + " pending since " + stamp(p.At) + " over its release " + shortSHA(r.Commit) + ": " + rel.clause
			if !rel.published {
				c = "dropped the hand-back of " + p.Issue + " pending since " + stamp(p.At) + " unpublished, as it is no longer due at its release " +
					shortSHA(r.Commit) + ": " + rel.clause
			}
			out = append(out, c)
			noteRoute(st, p.Issue, c)
		}
	}
	if len(kept) == 0 {
		kept = nil
	}
	st.PendingHandBacks = kept
	return out
}

// noteRoute adds clause to what the last route of issue from the judgement
// wrote, so the route says what came of its pending hand-back.
func noteRoute(st *DrainState, issue, clause string) {
	for i := len(st.HandBacks) - 1; i >= 0; i-- {
		if r := &st.HandBacks[i]; r.Issue == issue && r.From == DrainFromJudgement {
			r.Wrote += "; " + clause
			return
		}
	}
}

// pendingWritten reports whether every pending hand-back st keeps is one the
// drain writes: an issue, a kind from the closed vocabulary, and a blob that
// is an object name or none.
func (st DrainState) pendingWritten() bool {
	return !slices.ContainsFunc(st.PendingHandBacks, func(p DrainPendingHandBack) bool {
		return !validIssueKey(p.Issue) || !slices.Contains(claim.HandBackKinds, p.Kind) || p.Blob != "" && !gitutil.IsFullSHA(p.Blob)
	})
}
