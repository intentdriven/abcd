package loop

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/github"
)

// TestADrainLaneStartedWhileTheForgeFailsSaysSoAndALaterActAsksAgain
// (spc-2610161044461448, step 10): a drain lane whose reservation could not
// ask the forge whether origin is a fork says so in its run record and in the
// drain's result, as reserve does, and the failure is not kept for the
// process: a later act, past the minute's backoff, asks again.
func TestADrainLaneStartedWhileTheForgeFailsSaysSoAndALaterActAsksAgain(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	var asked []string
	forgeErr := errors.New("unreachable")
	restore := claim.SwapForkCheckForTest(
		func(string) ([]string, error) { return []string{"https://github.com/octo/abcd.git"}, nil },
		func(_, owner, name string) (github.Repo, error) {
			asked = append(asked, owner+"/"+name)
			return github.Repo{}, forgeErr
		})
	t.Cleanup(restore)
	c := &clock{t: time.Now().UTC().Truncate(time.Second)}
	o := Options{Now: c.now}

	first := drainToLane(t, repo, o, DrainOptions{})
	if first.Start == nil || first.Lane == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	const warning = "not checked: whether origin is a fork (octo/abcd: "
	if !strings.HasPrefix(first.Start.ForkUnchecked, "octo/abcd: ") {
		t.Fatalf("the drain's result names the skipped check: %q", first.Start.ForkUnchecked)
	}
	if note := recordNote(t, repo, first.Start.RunID, StageReservation); !strings.Contains(note, warning) {
		t.Fatalf("the run record names the skipped check: %q", note)
	}
	if len(asked) != 1 {
		t.Fatalf("the lane's reservation asked the forge %d times, want once", len(asked))
	}

	handBackLaneOf(t, repo, first.Start.RunID, o, LaneHandBack{Kind: HandBackDesignFinding, Reason: "the page needs a decision", Home: "the brief"})
	forgeErr = nil
	c.t = c.t.Add(2 * time.Minute)
	second := drainToLane(t, repo, o, DrainOptions{})
	if second.Start == nil || second.Lane == nil || second.Lane.Issue != eligibleIssue {
		t.Fatalf("the drain moves to the bug's lane: %+v", second)
	}
	if len(asked) != 2 {
		t.Fatalf("a later act asked the forge %d times in all, want twice: the failure was kept", len(asked))
	}
	if second.Start.ForkUnchecked != "" {
		t.Fatalf("an answered check warns: %q", second.Start.ForkUnchecked)
	}
}

// TestADrainLaneOnTheAllowedForkSaysSo (dcn-2610161223212218): a drain lane
// reserving on the fork the repository's reservation.fork_origin names goes
// through, and its run record and the drain's result say the check was
// waived for that fork, in reserve's words, so the waiver is never silent.
func TestADrainLaneOnTheAllowedForkSaysSo(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	repoConfig(t, repo, `{"reservation":{"fork_origin":"octo/abcd"}}`)
	restore := claim.SwapForkCheckForTest(
		func(string) ([]string, error) { return []string{"https://github.com/octo/abcd.git"}, nil },
		func(_, _, _ string) (github.Repo, error) {
			return github.Repo{Fork: true, Parent: "intentdriven/abcd"}, nil
		})
	t.Cleanup(restore)

	got := drainToLane(t, repo, Options{}, DrainOptions{})
	if got.Start == nil || got.Lane == nil {
		t.Fatalf("the drain opens a lane on the allowed fork: %+v", got)
	}
	if got.Start.ForkAllowed != "octo/abcd" || got.Start.ForkUnchecked != "" {
		t.Fatalf("the drain's result names the allowed fork: allowed %q, unchecked %q", got.Start.ForkAllowed, got.Start.ForkUnchecked)
	}
	if note := recordNote(t, repo, got.Start.RunID, StageReservation); !strings.Contains(note, claim.ForkAllowedLine("octo/abcd")) {
		t.Fatalf("the run record names the allowed fork: %q", note)
	}
}
