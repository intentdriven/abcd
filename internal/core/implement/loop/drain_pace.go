package loop

// drain_pace.go is a running drain's new pace (spc-2610161044461448 step 16;
// iss-2610161015417715, iss-2610161015457468; dcn-2610161040399117). The pace
// a drain runs on is the person's setting for the drain's whole life: every
// lane the drain opens starts on it (Options.pace), rather than resolving its
// own from the configuration, and a running drain given --pace, --sub-agents
// or --fix-rounds takes them instead of refusing them. The new numbers are
// written to the drain's state, the next lane opens on them, and each lane in
// progress takes them into its run's pace, which its next stage reads, once
// the move's state is written: a move refused takes no new pace, in the
// drain's state or in any lane.

import (
	"os"
	"time"
)

// drainRepace is the new pace a move of a running drain was given: the pace
// the flags resolved, and which of its numbers they name.
type drainRepace struct {
	given Pace
	f     paceFlags
}

// over is p with each number the flags name replaced, the rest kept.
func (r drainRepace) over(p *Pace) Pace {
	if p == nil {
		return r.given
	}
	out := *p
	if r.f.work != nil {
		out.WorkMinutes, out.PauseMinutes = r.given.WorkMinutes, r.given.PauseMinutes
	}
	if r.f.subs != nil {
		out.SubAgents = r.given.SubAgents
	}
	if r.f.fix != nil {
		out.FixRounds = r.given.FixRounds
	}
	return out
}

// repaceDrain takes the flags a move of a running drain was given into st's
// pace, each number the flags name replacing the one stored and the rest kept
// as they stand. It returns the new pace, for the lanes in progress to take
// once the move's state is written (lanes), and the move's clause saying
// what it took, "" when the flags name the pace the drain already runs on. A
// move refused after this takes nothing: it is the caller's to put st's pace
// back, and the lanes are never touched.
func repaceDrain(repoRoot string, st *DrainState, o Options, f paceFlags) (*drainRepace, string, error) {
	given, err := resolvePace(o.roots(repoRoot), f)
	if err != nil {
		return nil, "", err
	}
	r := &drainRepace{given: given, f: f}
	next := r.over(st.Pace)
	if st.Pace != nil && next.same(*st.Pace) {
		return r, "", nil
	}
	st.Pace = &next
	return r, "the drain takes the pace " + next.String() + " from this move: the next lane it opens runs on it, and the lane in progress from its next stage", nil
}

// lanes takes the new pace into the run pace of each lane of st in progress,
// which that lane's next stage reads, saying so in its run record. A lane
// already on it is left as it stands, so taking it again changes nothing.
func (r *drainRepace) lanes(repoRoot string, st DrainState, now time.Time) error {
	for _, l := range st.Lanes {
		if l.Outcome != DrainLaneInProgress {
			continue
		}
		if err := mutate(repoRoot, l.RunID, func(_ *os.Root, run *State) (bool, error) {
			if run.Complete() {
				return false, nil
			}
			p := r.over(run.Pace)
			if run.Pace != nil && p.same(*run.Pace) {
				return false, nil
			}
			run.Pace = &p
			run.Record = append(run.Record, Entry{At: now, Stage: StagePace,
				Note: "the drain's pace changed to " + p.String() + "; this lane runs on it from its next stage"})
			run.UpdatedAt = now
			return true, nil
		}); err != nil {
			return err
		}
	}
	return nil
}
