package loop

import (
	"slices"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// handBackRelease is what the release a hand-back's save makes after its write
// came to (releaseAfterWrite).
type handBackRelease struct {
	// ran is set once the save's write returned without error and the step
	// ran; a save refused or failed leaves every field zero.
	ran bool
	// made is set when the step tried the release on the remote; failed when
	// that release did not reach it, so the reservation is still held.
	made, failed bool
	// autoPlan is set when the release was an auto-plan run's (releaseAutoPlan),
	// not an issue run's (releaseOnHandBack).
	autoPlan bool
	// clause is the run record's line for the release (releaseOnHandBack).
	clause string
	// until is when a reservation the release failed to free lapses, as this
	// checkout last fetched it, "" where that view does not show it as this
	// copy's; ended is set when that time is already past.
	until string
	ended bool
	// unwritten is the second write's error: the line is not in the record.
	unwritten error
	// state is the run as the step left it, the release's line included.
	state State
}

// releaseAfterWrite registers, on the state a save's callback has just handed
// lane back in, or finished an auto-plan run's lane in, the step
// (State.afterWritten) that releases the run's reservation once that save's
// write has returned without error, under the lock the save took: the
// hand-back or the finish is written, then the release is made on the remote
// (releaseOnHandBack for an issue run, releaseAutoPlan for an auto-plan run),
// then the release's line is written to the run record by a second write
// under the same hold. No lock is taken again, so nothing can come between
// the move and its release, and a save refused or failed releases nothing and
// moves nothing.
//
// The step reads the lane as it stands in the state just written, never as the
// callback saw it: a lane neither handed back there nor, on an auto-plan run,
// done is left alone, and so is a run that holds no reservation, is an
// intent's other than an auto-plan's, or whose record already holds a
// reservation_released line for the reservation it holds now
// (releasedThisHold), so a reservation is never released twice.
//
// A crash between the write and the release leaves the lane handed back and
// the reservation held, with neither reservation_released nor
// reservation_release_failed in the record: a drain routing the hand-back
// releases it (releaseHandedBack), and otherwise it lapses at its lease's end,
// since nothing renews a handed-back issue run's reservation
// (handedBackIssue).
func releaseAfterWrite(repoRoot string, st *State, lane string, now time.Time) *handBackRelease {
	out := &handBackRelease{}
	st.afterWritten(func(s *State, rewrite func() error) {
		out.ran = true
		defer func() { out.state = *s }()
		at := slices.IndexFunc(s.Lanes, func(l Lane) bool { return l.ID == lane })
		if at < 0 || releasedThisHold(*s) {
			return
		}
		l := s.Lanes[at]
		logged := len(s.Record)
		switch {
		case l.AutoPlan != nil && (l.HandBack != nil || l.Stage == StageDone):
			out.autoPlan = true
			releaseAutoPlan(repoRoot, s, now)
		case l.HandBack != nil:
			releaseOnHandBack(repoRoot, s, *l.HandBack, now)
		default:
			return
		}
		if len(s.Record) == logged {
			// No reservation, or an intent's run: nothing to release.
			return
		}
		e := s.Record[len(s.Record)-1]
		out.made, out.failed, out.clause = true, e.Stage == StageReservationReleaseFailed, e.Note
		if out.failed && !out.autoPlan {
			out.until, out.ended = leaseEnd(repoRoot, *s, now)
		}
		out.unwritten = rewrite()
	})
	return out
}

// releasedThisHold reports whether the run's record holds a
// reservation_released line since the reservation it holds was last made: the
// record read from its end back to the last reservation line.
func releasedThisHold(st State) bool {
	for k := len(st.Record) - 1; k >= 0; k-- {
		switch st.Record[k].Stage {
		case StageReservationReleased:
			return true
		case StageReservation:
			return false
		}
	}
	return false
}

// leaseEnd is when the run's reservation lapses, as this checkout last
// fetched it (claim.View reads no remote), "" where that view does not show
// it as this copy's; ended is set when that time is not after now.
func leaseEnd(repoRoot string, st State, now time.Time) (until string, ended bool) {
	snap, err := claim.View(repoRoot, now)
	if err != nil {
		return "", false
	}
	own, _, lost := renewalDue(st, snap, now)
	if own == nil || lost != "" || own.Copy != claim.CopyThis || (own.State != claim.StateLive && own.State != claim.StateExpired) {
		return "", false
	}
	return stamp(own.EffectiveExpiry), own.State == claim.StateExpired || !own.EffectiveExpiry.After(now)
}

// next is what the release adds to a result's next step: nothing where it was
// made and recorded, or where none was made; "not released" where the remote
// did not take it; "released, not recorded" where it did and the record's
// second write failed. An issue run's lease is told as this checkout last
// fetched it: "ends at" a time to come, "ended at" one already past.
func (r *handBackRelease) next() string {
	if r == nil || !r.made {
		return ""
	}
	if r.autoPlan {
		if !r.failed {
			if r.unwritten == nil {
				return ""
			}
			return "the auto-plan run's reservation was released, not recorded: " + r.clause + r.unwrittenNote("say so")
		}
		return "the auto-plan run's reservation was not released: " + r.clause + r.unwrittenNote("say the release is owed")
	}
	if !r.failed {
		if r.unwritten == nil {
			return ""
		}
		return "the lane was handed back and its reservation released, not recorded: " + r.clause + r.unwrittenNote("say so")
	}
	s := "the lane was handed back and its reservation not released: " + r.clause
	if r.until != "" {
		verb := "ends"
		if r.ended {
			verb = "ended"
		}
		s += "; its lease " + verb + " at " + r.until + " as last fetched, and nothing renews a handed-back lane's reservation (a run `abcd build` started has no drain routing it)"
	} else {
		s += "; nothing renews a handed-back lane's reservation (a run `abcd build` started has no drain routing it)"
	}
	return s + r.unwrittenNote("say the release is owed")
}

// said is note, the run record's last line a caller read in the save's
// callback, as the caller says it once the save has returned: where a release
// was made, the release's own line where it was recorded, and what it came to
// (next) where it was not released or not recorded; note where none was made.
func (r *handBackRelease) said(note string) string {
	if r == nil || !r.made {
		return note
	}
	if n := r.next(); n != "" {
		return n
	}
	return r.clause
}

// unwrittenNote is the clause naming the record's second write that failed,
// "" where it was written.
func (r *handBackRelease) unwrittenNote(what string) string {
	if r == nil || r.unwritten == nil {
		return ""
	}
	return "; the run record could not be written to " + what + ": " + termsafe.Sanitize(fsutil.RedactHome(r.unwritten.Error()))
}
