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
	// until is the end of the lease of a reservation the release failed to
	// free, as this checkout last fetched it, "" where that view does not show
	// it as this copy's; ended is set when that time is already past: the
	// reservation has lapsed, and nothing revives it.
	until string
	ended bool
	// arming is the record's line on the withdrawal of the arming the
	// hand-back owed (armingOwed), "" where it owed none; armed is set when
	// that withdrawal did not take, so the pull request may still merge and
	// nothing was released.
	arming string
	armed  bool
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
// releases it (releaseHandedBack), and otherwise it lapses once its lease runs
// out unrenewed. No step and no run renewal renews a handed-back issue run's
// reservation (handedBackIssue), but a load-gate waiter in this copy
// (claim.RenewDue, which reads no run) may still renew it, as gateWaiterRenews
// says, so the lease can run past the time the answer gives.
//
// A lane handed back with its pull request still armed (armingOwed) has the
// arming withdrawn first, here, after the write, so a refused save leaves the
// forge as it was (withdrawLateArming), and the record says what came of it.
// Until the forge reports no arming nothing is released: the lane stays where
// each later step tries the withdrawal again (laneWant), and that step's own
// release is made once it has.
func releaseAfterWrite(repoRoot string, st *State, lane string, now time.Time) *handBackRelease {
	out := &handBackRelease{}
	st.afterWritten(func(s *State, rewrite func() error) {
		out.ran = true
		defer func() { out.state = *s }()
		at := slices.IndexFunc(s.Lanes, func(l Lane) bool { return l.ID == lane })
		if at < 0 {
			return
		}
		if armingOwed(s.Lanes[at]) {
			l := s.Lanes[at]
			c := Context{RepoRoot: repoRoot, RunDir: runRel(s.RunID), State: *s, Now: now}
			note, err := withdrawLateArming(c, &l, StageHandedBack, lateHandBack(l), "it merges only if a person arms or merges it")
			if r, ok := AsRefusal(err); ok {
				note = r.Reason + "; " + r.Remedy
			} else if err != nil {
				note = termsafe.Sanitize(fsutil.RedactHome(err.Error()))
			}
			s.Lanes[at] = l
			s.Record = append(s.Record, Entry{At: now, Lane: l.ID, Stage: string(StageHandedBack), Note: note})
			s.UpdatedAt = now
			out.arming, out.armed = note, armingOwed(l)
		}
		if out.armed || releasedThisHold(*s) {
			if out.arming != "" {
				out.unwritten = rewrite()
			}
			return
		}
		l := s.Lanes[at]
		logged := len(s.Record)
		switch {
		case l.AutoPlan != nil && (l.HandBack != nil || l.Stage == StageDone):
			out.autoPlan = true
			releaseAutoPlan(repoRoot, s, now)
		case l.HandBack != nil:
			out.until, out.ended = releaseOnHandBack(repoRoot, s, *l.HandBack, now)
		}
		if len(s.Record) == logged {
			// No reservation, or an intent's run: nothing to release.
			if out.arming != "" {
				out.unwritten = rewrite()
			}
			return
		}
		e := s.Record[len(s.Record)-1]
		out.made, out.failed, out.clause = true, e.Stage == StageReservationReleaseFailed, e.Note
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

// leaseEnd is the end of the run's reservation's lease as this checkout last
// fetched it (claim.View reads no remote), which a load-gate waiter in this
// copy may put off while the reservation is live (gateWaiterRenews), "" where
// that view does not show it as this copy's; ended is set when that time is
// not after now: the reservation has lapsed, and nothing revives it.
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
// fetched it: "runs to" a time to come, which a load-gate waiter in this copy
// may put off (waiterRenews), "lapsed at" one already past, which nothing
// revives, so it gets no waiter's clause; the record's line already says when
// it lapsed (releaseOnHandBack), and the answer adds only that nothing revives
// it. An arming the hand-back owed is said first: what its withdrawal came to,
// and, where it did not take, that nothing was released.
func (r *handBackRelease) next() string {
	if r == nil {
		return ""
	}
	if r.arming != "" {
		s := r.arming
		if r.armed {
			return s + "; the run's reservation is not released while it is armed" + r.unwrittenNote("say so")
		}
		if !r.made {
			return s + r.unwrittenNote("say so")
		}
		return joinNext(s, r.releaseNext())
	}
	return r.releaseNext()
}

// releaseNext is next's word on the release alone.
func (r *handBackRelease) releaseNext() string {
	if !r.made {
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
	switch {
	case r.until != "" && r.ended:
		s += ", and nothing revives it"
	case r.until != "":
		s += "; its lease runs to " + r.until + " as last fetched, longer if a load-gate waiter in this copy renews it: " + waiterRenews
	default:
		s += "; " + waiterRenews
	}
	return s + " (a run `abcd build` started has no drain routing it)" + r.unwrittenNote("say the release is owed")
}

// waiterRenews is what renews a handed-back lane's reservation, and what does
// not: no step and no run renewal (handedBackIssue), but the load gate's
// waiter (gateWaiterRenews).
const waiterRenews = "no step and no run renewal renews a handed-back lane's reservation, but " + gateWaiterRenews

// gateWaiterRenews says the renewal no run governs, as claim.RenewDue makes
// it (renewdue.go: live, not a release, half through, the tip this copy's last
// push; renew-only and live-only) and the waiter runs it (startHoldRenewal in
// implement_load.go, at once and every round until the command exits), a
// reservation whose release failed included. "Never revives" is that
// renewal's alone: the same waiter's run renewal (renewruns.go) revives a
// lapsed reservation still this copy's of a run it governs. Every text that says what the
// waiter renews uses it, and commands/implement.md says the same words
// (TestThePageSaysWhatTheGateWaiterRenews).
const gateWaiterRenews = "a load-gate waiter in this copy (`abcd implement load --wait`, and `--race-gate`, which preflight's race stage uses) " +
	"renews, while it waits and while its command runs, each live reservation that is not a release, is half through its lease, " +
	"and whose tip is the commit this copy last pushed; that renewal, which reads no run, never revives one that has lapsed or been released"

// unlessWaiterRenews is the clause a lapse said of a reservation that may still
// be live and this copy's last push takes: a load-gate waiter in this copy may
// renew it (gateWaiterRenews), so the lapse is not promised. A reservation that
// is certainly not live (lapsed, or released) takes no such clause.
const unlessWaiterRenews = "unless a load-gate waiter in this copy renews it"

// said is note, the run record's last line a caller read in the save's
// callback, as the caller says it once the save has returned: where a release
// was made, the release's own line where it was recorded, and what it came to
// (next) where it was not released or not recorded; note where none was made,
// and what the withdrawal of an arming the hand-back owed came to beside it.
func (r *handBackRelease) said(note string) string {
	if r == nil {
		return note
	}
	if !r.made {
		if r.arming != "" {
			return joinNext(note, r.next())
		}
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
