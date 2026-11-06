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
// live. A lane at its pull request is no such lane until it needs an agent again, and a
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
		if run.Profile != "" && run.Profile != profile {
			out = append(out, LaneOnProfile{RunID: l.RunID, Issue: l.Issue, Profile: run.Profile})
		}
	}
	return out, nil
}

// holdToProfile is the check before a drive drives run runID: another profile ends the
// run, before anything claims or reserves the lane, as a wall that no longer holds does.
// A run that names no profile goes on and takes this one, said and recorded, so a later
// drive on another is held to it. It returns why the run ends, "" to go on.
func (t *terminal) holdToProfile(runID string) (string, error) {
	profile := t.o.Options.Profile
	if profile == "" {
		return "", nil
	}
	run, err := ReadState(t.repoRoot, runID)
	if err != nil {
		return "", err
	}
	switch run.Profile {
	case profile:
		return "", nil
	case "":
	default:
		t.stoppedBy(errors.New(OnAnotherProfile(LaneOnProfile{RunID: runID, Issue: run.Key, Profile: run.Profile}, profile)))
		return TerminalWall, nil
	}
	now := t.now()
	note := "this run named no profile, and from here a drive runs it on profile " + termsafe.Sanitize(profile)
	err = mutate(t.repoRoot, runID, func(_ *os.Root, st *State) (bool, error) {
		st.Profile = profile
		st.Record = append(st.Record, Entry{At: now, Stage: StageProfile, Note: note})
		st.UpdatedAt = now
		return true, nil
	})
	if r, ok := AsRefusal(err); ok && r.Contention {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	t.say("%s: %s", runID, note)
	return "", nil
}
