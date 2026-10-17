package loop

// drain_pace.go is a running drain's new pace (spc-2610161044461448 step 16;
// iss-2610161015417715, iss-2610161015457468; dcn-2610161040399117). The pace
// a drain runs on is the person's setting for the drain's whole life: every
// lane the drain opens starts on it (Options.pace), rather than resolving its
// own from the configuration, and a running drain given --pace, --sub-agents
// or --fix-rounds takes them instead of refusing them. The new numbers are
// written to the drain's state, the next lane opens on them, and each lane in
// progress takes them into its run's pace, which its next stage reads.

import (
	"os"
	"time"
)

// repaceDrain takes the flags a move of a running drain was given into st's
// pace, and into the pace of each lane in progress, each number the flags
// name replacing the one stored and the rest kept as they stand. It returns
// the move's clause saying what it took, "" when the flags name the pace the
// drain already runs on.
func repaceDrain(repoRoot string, st *DrainState, o Options, f paceFlags, now time.Time) (string, error) {
	given, err := resolvePace(o.roots(repoRoot), f)
	if err != nil {
		return "", err
	}
	over := func(p *Pace) Pace {
		if p == nil {
			return given
		}
		out := *p
		if f.work != nil {
			out.WorkMinutes, out.PauseMinutes = given.WorkMinutes, given.PauseMinutes
		}
		if f.subs != nil {
			out.SubAgents = given.SubAgents
		}
		if f.fix != nil {
			out.FixRounds = given.FixRounds
		}
		return out
	}
	next := over(st.Pace)
	if st.Pace != nil && next.same(*st.Pace) {
		return "", nil
	}
	st.Pace = &next
	for _, l := range st.Lanes {
		if l.Outcome != DrainLaneInProgress {
			continue
		}
		if err := mutate(repoRoot, l.RunID, func(_ *os.Root, run *State) (bool, error) {
			if run.Complete() {
				return false, nil
			}
			p := over(run.Pace)
			if run.Pace != nil && p.same(*run.Pace) {
				return false, nil
			}
			run.Pace = &p
			run.Record = append(run.Record, Entry{At: now, Stage: StagePace,
				Note: "the drain's pace changed to " + p.String() + "; this lane runs on it from its next stage"})
			run.UpdatedAt = now
			return true, nil
		}); err != nil {
			return "", err
		}
	}
	return "the drain takes the pace " + next.String() + " from this move: the next lane it opens runs on it, and the lane in progress from its next stage", nil
}
