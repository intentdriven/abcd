package loop

import (
	"slices"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// A drain lane's stage rides on its reservation's renewal
// (spc-2610161044461448 step 18, the loop's half; iss-2610161015434746;
// dcn-2610161040394970): every renewal the loop pushes for a lane names the
// lane's stage from the loop's own names, which claim holds as LaneStages.

// everyStage is every Stage the loop gives a lane.
var everyStage = []Stage{StageWorktree, StageBrief, StageImplement, StageValidate, StageLand,
	StageDone, StageHandedBack, StageHeld, StageDiscarded}

// TestEveryStageTheLoopRenewsWithIsAClaimLaneStage keeps the loop's stage
// names and claim's LaneStages from drifting: claim cannot import the loop,
// so the loop holds the two lists together. Each stage a lane performs is
// renewed under its own name, which claim accepts, and a lane with nothing
// left to perform (done, handed back, held, discarded) renews with none.
func TestEveryStageTheLoopRenewsWithIsAClaimLaneStage(t *testing.T) {
	performed := []Stage{StageWorktree, StageBrief, StageImplement, StageValidate, StageLand}
	for _, s := range everyStage {
		got := renewalStage(State{Key: eligibleIssue, Lanes: []Lane{{ID: "lane-1", Stage: s}}})
		if got != "" && !claim.ValidLaneStage(got) {
			t.Errorf("a lane at %s renews with %q, which claim refuses", s, got)
		}
		if want := string(s); slices.Contains(performed, s) && got != want {
			t.Errorf("a lane at %s renews with %q, want %q", s, got, want)
		}
		if !slices.Contains(performed, s) && got != "" {
			t.Errorf("a lane at %s renews with %q, want none", s, got)
		}
	}
	for _, s := range claim.LaneStages {
		if !slices.Contains(performed, Stage(s)) {
			t.Errorf("claim names %q, a stage the loop never performs", s)
		}
	}
	if got := renewalStage(State{Key: eligibleIssue}); got != "" {
		t.Errorf("a run with no lane renews with %q, want none", got)
	}
}

// TestADrainMovesRenewalCarriesTheLanesStage: the renewal a waiting drain
// move pushes for its lane in progress names that lane's stage.
func TestADrainMovesRenewalCarriesTheLanesStage(t *testing.T) {
	t0 := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)
	c := &clock{t: t0}
	o := Options{Now: c.now}
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	first := drainToLane(t, repo, o, DrainOptions{})
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	var renewals []claim.Request
	was := reserveRenewal
	reserveRenewal = func(root string, r claim.Request, co claim.Options) (claim.Result, error) {
		if r.Renew {
			renewals = append(renewals, r)
		}
		return was(root, r, co)
	}
	t.Cleanup(func() { reserveRenewal = was })
	c.t = t0.Add(61 * time.Minute)
	if _, err := Drain(repo.Root(), o, DrainOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(renewals) != 1 || renewals[0].Record != secondIssue || renewals[0].Stage != string(StageWorktree) {
		t.Fatalf("the lane's renewal names its stage, worktree: %+v", renewals)
	}
}
