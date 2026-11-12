package loop

// drive_profile.go holds a run to the profile it started on: a drive names one
// (Options.Profile), the run keeps the name in its state, and a drive on another
// profile does not take the run up, since that would go on on another subscription
// with no one having said so. The front door asks before its question
// (LanesOnAnotherProfile); the loop asks before it drives a lane, which also covers a
// lane that comes back after the question, as one at its pull request does for its
// sync. A person who means to go on elsewhere has a session of their own take the run
// over (`abcd implement resume --take`), or runs the other profile in another checkout,
// whose runs and drain are its own. A profile is a separate login and settings on one
// account, so this holds the spending to one login and walls nothing.

import (
	"errors"
	"fmt"
	"os"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// StageProfile is the run record's stage for a drive marking a run that named no
// profile with its own.
const StageProfile = "profile"

// LaneOnProfile is a lane a drive would take up and the profile its run started on.
type LaneOnProfile struct {
	RunID, Issue, Profile string
}

// OnAnotherProfile says why a drive on profile does not take l up, and what the
// person may do instead.
func OnAnotherProfile(l LaneOnProfile, profile string) string {
	key, was, now := termsafe.Sanitize(l.Issue), termsafe.Sanitize(l.Profile), termsafe.Sanitize(profile)
	return fmt.Sprintf("%s, %s, started on profile %s, and this drive is on profile %s, another subscription: "+
		"give --profile %s to go on with it, run the other in another checkout, whose runs are its own, "+
		"or take the run over from a session of your own with `abcd implement resume --take %s`",
		termsafe.Sanitize(l.RunID), key, was, now, was, termsafe.Sanitize(l.RunID))
}

// LanesOnAnotherProfile names each lane this checkout's drain would drive next that
// started on another profile than this one: its lanes in progress, while the drain is
// live. Only a drain's own move rewrites the outcome it stored, so the run says too: one
// a session finished, or at its pull request, needs no agent until it comes back, and a
// drain that ended takes nothing up. A run that names no profile holds to none yet.
func LanesOnAnotherProfile(repoRoot, profile string) ([]LaneOnProfile, error) {
	st, live, err := readDrain(repoRoot)
	if err != nil || !live {
		return nil, err
	}
	var out []LaneOnProfile
	for _, l := range st.Lanes {
		if l.Outcome != DrainLaneInProgress {
			continue
		}
		run, err := ReadState(repoRoot, l.RunID)
		if err != nil {
			return nil, err
		}
		if outcome, _, _ := laneOutcome(run); outcome != DrainLaneInProgress {
			continue
		}
		if run.Profile != "" && run.Profile != profile {
			out = append(out, LaneOnProfile{RunID: l.RunID, Issue: l.Issue, Profile: run.Profile})
		}
	}
	return out, nil
}

// onAnotherProfile reports whether this drive is on another profile than run started on.
func (t *terminal) onAnotherProfile(run State) bool {
	return t.o.Options.Profile != "" && run.Profile != "" && run.Profile != t.o.Options.Profile
}

// resumeCommand is what a person types to go on with this drive's work later.
func (t *terminal) resumeCommand() string {
	if p := t.o.Options.Profile; p != "" {
		return TerminalResumeCommand + " --profile " + p
	}
	return TerminalResumeCommand
}

// holdToProfile is the check before a drive drives run runID: another profile ends the
// run, before anything claims or reserves the lane, as a wall that no longer holds does.
// It returns why the run ends, "" to go on.
func (t *terminal) holdToProfile(runID string) (string, error) {
	if t.o.Options.Profile == "" {
		return "", nil
	}
	run, err := ReadState(t.repoRoot, runID)
	if err != nil || !t.onAnotherProfile(run) {
		return "", err
	}
	t.heldToProfile = true
	t.stoppedBy(errors.New(OnAnotherProfile(LaneOnProfile{RunID: runID, Issue: run.Key, Profile: run.Profile}, t.o.Options.Profile)))
	return TerminalWall, nil
}

// markProfile has a run that names no profile take this drive's, said and recorded, so a
// later drive on another is held to it. It waits until the drive has taken the lane up:
// one that waited on another driver and ended ran nothing on this profile.
func (t *terminal) markProfile(runID string) error {
	profile := t.o.Options.Profile
	if profile == "" {
		return nil
	}
	now, marked := t.now(), false
	note := "this run named no profile, and from here a drive runs it on profile " + termsafe.Sanitize(profile)
	err := saveRun(t.repoRoot, runID, t.o.Options, func(_ *os.Root, st *State) (bool, error) {
		if st.Profile != "" {
			return false, nil
		}
		st.Profile, marked = profile, true
		st.Record = append(st.Record, Entry{At: now, Stage: StageProfile, Note: note})
		st.UpdatedAt = now
		return true, nil
	})
	if r, ok := AsRefusal(err); ok && r.Contention {
		return nil
	}
	if err != nil {
		return err
	}
	if marked {
		t.say("%s: %s", runID, note)
	}
	return nil
}
