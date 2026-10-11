package loop

import (
	"os"
	"strings"
	"testing"
)

// The summary's counts (itd-82 scope 8, spc-2609212015054359 step 2): the
// pull requests opened, the promotions made and the spend are counted in the
// summary, every hand-back by its route, and a drain whose lanes all ended
// without a pull request says so as a normal ending.

// landLaneOf writes a drain's run as its landing would leave it: two
// implementers' receipts (the implement stage's and one fix round's), two
// rounds of the two reviewers, and the lane done with its pull request.
func landLaneOf(t *testing.T, repo interface{ Root() string }, runID string, pr int) {
	t.Helper()
	err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		l := &st.Lanes[0]
		l.Receipts = []ReceiptRecord{{Role: RoleImplementer, Receipt: "r1.json"}, {Role: RoleImplementer, Receipt: "r2.json"}}
		l.Validation = []ValidationRound{
			{Round: 1, HeadSHA: "a", Validators: []ValidatorRun{{Role: RoleRuthless, Verdict: "NEEDS_WORK"}, {Role: RoleSecurity, Verdict: "SHIP", Pass: true}}, Fix: "r2.json"},
			{Round: 2, HeadSHA: "b", Validators: []ValidatorRun{{Role: RoleRuthless, Verdict: "SHIP", Pass: true}, {Role: RoleSecurity, Verdict: "SHIP", Pass: true}}},
		}
		l.Landing = &Landing{RecordsDone: true, Merge: "queue", Armed: true}
		l.PR, l.Stage, l.Awaits = pr, StageDone, nil
		st.Pending = nil
		return true, nil
	})
	if err != nil {
		t.Fatalf("landing the drain's lane: %v", err)
	}
}

// TestTheDrainSummaryCountsPullRequestsPromotionsAndSpend: a lane that landed
// its pull request and a remedy the host judgement promoted are each counted,
// the spend is the agent runs the lanes' runs record (each implementer's
// verified receipt and each validator's return), and a lane done before the
// drain read it keeps its pull request number.
func TestTheDrainSummaryCountsPullRequestsPromotionsAndSpend(t *testing.T) {
	repo := drainRepo(t)
	first := drainToLane(t, repo, Options{}, DrainOptions{})
	if c := first.Counts; c.LanesOpened != 1 || c.InProgress != 1 || c.Judgements != 1 || c.Spend.AgentRuns != 0 {
		t.Fatalf("the move that opens a lane counts it, in progress, and the judgement it took: %+v", c)
	}
	landLaneOf(t, repo, first.Start.RunID, 41)

	next, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if next.Lanes[0].Outcome != DrainLaneDone || next.Lanes[0].PR != 41 {
		t.Fatalf("a lane done before the drain read it keeps its pull request: %+v", next.Lanes[0])
	}
	end := judge(t, repo, Options{}, next, JudgementYes, HandBackUserVisible, "the page gains a flag users see")
	if end.Stopped != DrainStoppedEmpty {
		t.Fatalf("with nothing eligible left the drain ends: %+v", end)
	}
	c := end.Counts
	if c.LanesOpened != 1 || c.InProgress != 0 || c.PullRequests != 1 || c.LanesHandedBack != 0 {
		t.Errorf("one lane opened, which reached its pull request: %+v", c)
	}
	if c.Promotions != 1 || c.DecisionRecords != 0 || c.Homes != 0 {
		t.Errorf("the judgement's user-visible hand-back is one promotion: %+v", c)
	}
	if c.Judgements != 2 || c.RuleFlags != len(end.Flags) || c.RuleFlags == 0 || c.Passed != len(end.Passed) {
		t.Errorf("the judgements, the rule's flags and the passed issues are counted: %+v (flags %d)", c, len(end.Flags))
	}
	if s := c.Spend; s.Implementers != 2 || s.Validators != 4 || s.AgentRuns != 6 || len(s.Unread) != 0 {
		t.Errorf("the spend is every implementer's receipt and every validator's return: %+v", s)
	}
	if strings.Contains(end.Next, "no lane ended in a pull request") {
		t.Errorf("a drain that opened a pull request does not say none did: %s", end.Next)
	}
}

// TestADrainWhoseLanesAllEndedWithoutAPullRequestSaysSo: a capped drain whose
// one lane was handed back counts no pull request, the hand-back by its
// route and the lane's implementer in the spend, and its end says that no lane
// ended in a pull request, a normal ending.
func TestADrainWhoseLanesAllEndedWithoutAPullRequestSaysSo(t *testing.T) {
	repo := drainRepo(t)
	first := drainToLane(t, repo, Options{}, DrainOptions{Max: 1})
	handBackLaneOf(t, repo, first.Start.RunID, Options{}, LaneHandBack{Kind: HandBackTrustRule, Reason: "may a guard skip the owner check?"})
	capped, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	c := capped.Counts
	if capped.Stopped != DrainStoppedCap || c.PullRequests != 0 || c.LanesHandedBack != 1 || c.DecisionRecords != 1 || c.Promotions != 0 {
		t.Fatalf("the capped drain counts its handed-back lane and the decision-record flag: %+v", c)
	}
	if s := c.Spend; s.Implementers != 1 || s.Validators != 0 || s.AgentRuns != 1 {
		t.Errorf("the hand-back's implementer is the lane's spend: %+v", s)
	}
	if !strings.Contains(capped.Next, "no lane ended in a pull request") || !strings.Contains(capped.Next, "normal ending") {
		t.Errorf("a drain ending without a pull request says so as a normal ending: %s", capped.Next)
	}
}

// TestASpendWhoseRunCannotBeReadIsNamed: a lane whose run state is gone is
// named in the spend, which is then a floor, rather than refusing the drain.
func TestASpendWhoseRunCannotBeReadIsNamed(t *testing.T) {
	repo := drainRepo(t)
	first := drainToLane(t, repo, Options{}, DrainOptions{Max: 1})
	st := DrainState{Lanes: []DrainLane{{Issue: first.Lane.Issue, RunID: "run-0000000000000001", Outcome: DrainLanePullRequest, PR: 3}}}
	c := drainCounts(repo.Root(), st, nil, nil)
	if len(c.Spend.Unread) != 1 || c.Spend.Unread[0] != "run-0000000000000001" || c.PullRequests != 1 {
		t.Fatalf("an unreadable run is named in the spend: %+v", c)
	}
}
