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
	// clause is the run record's line for the release (releaseOnHandBack).
	clause string
	// until is when a reservation the release failed to free lapses, "" where
	// this checkout's last fetch does not show it as this copy's.
	until string
	// unwritten is the second write's error: the line is not in the record.
	unwritten error
	// state is the run as the step left it, the release's line included.
	state State
}

// releaseAfterWrite registers, on the state a save's callback has just handed
// lane back in, the step (State.afterWritten) that releases an issue run's
// reservation once that save's write has returned without error, under the
// lock the save took: the hand-back is written, then the release is made on
// the remote (releaseOnHandBack), then the release's line is written to the
// run record by a second write under the same hold. No lock is taken again, so
// nothing can come between the hand-back and its release, and a save refused
// or failed releases nothing and hands nothing back.
//
// The step reads the lane as it stands in the state just written, never as the
// callback saw it: a lane not handed back there is left alone, and so is a run
// that holds no reservation, is not an issue's, or whose record already holds
// a reservation_released line for the reservation it holds now
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
		if at < 0 || s.Lanes[at].HandBack == nil || releasedThisHold(*s) {
			return
		}
		logged := len(s.Record)
		releaseOnHandBack(repoRoot, s, *s.Lanes[at].HandBack, now)
		if len(s.Record) == logged {
			// No reservation, or an intent's run: nothing to release.
			return
		}
		e := s.Record[len(s.Record)-1]
		out.made, out.failed, out.clause = true, e.Stage == StageReservationReleaseFailed, e.Note
		if out.failed {
			out.until = leaseEnd(repoRoot, *s, now)
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
// fetched it, "" where that view does not show it as this copy's.
func leaseEnd(repoRoot string, st State, now time.Time) string {
	snap, err := claim.View(repoRoot, now)
	if err != nil {
		return ""
	}
	own, _, lost := renewalDue(st, snap, now)
	if own == nil || lost != "" || own.Copy != claim.CopyThis || (own.State != claim.StateLive && own.State != claim.StateExpired) {
		return ""
	}
	return stamp(own.EffectiveExpiry)
}

// next is what the release adds to a result's next step: nothing where it was
// made and recorded, or where none was made; "not released" where the remote
// did not take it; "released, not recorded" where it did and the record's
// second write failed.
func (r *handBackRelease) next() string {
	if r == nil || !r.made {
		return ""
	}
	if !r.failed {
		if r.unwritten == nil {
			return ""
		}
		return "the lane was handed back and its reservation released, not recorded: " + r.clause + r.unwrittenNote("say so")
	}
	s := "the lane was handed back and its reservation not released: " + r.clause
	if r.until != "" {
		s += "; its lease ends at " + r.until + ", and nothing renews a handed-back lane's reservation (a run `abcd build` started has no drain routing it)"
	} else {
		s += "; nothing renews a handed-back lane's reservation (a run `abcd build` started has no drain routing it)"
	}
	return s + r.unwrittenNote("say the release is owed")
}

// unwrittenNote is the clause naming the record's second write that failed,
// "" where it was written.
func (r *handBackRelease) unwrittenNote(what string) string {
	if r == nil || r.unwritten == nil {
		return ""
	}
	return "; the run record could not be written to " + what + ": " + termsafe.Sanitize(fsutil.RedactHome(r.unwritten.Error()))
}
