package loop

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The drain run (itd-82 scope 4, 5 and 7): one issue-keyed lane at a time in
// the drain order, every hand-back routed by kind and said in the summary, and
// the pace rule's window and --max bounding the run.

// drainRepo is issueRepo with a second eligible issue, a documentation one,
// which the drain order takes before the bug.
func drainRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	repo := issueRepo(t)
	fileIssue(t, repo, secondIssue, capture.SeverityMinor, capture.Category("documentation"), "Name the flag in the page.")
	repo.Commit("a second eligible issue")
	return repo
}

// clock is a settable Options.Now.
type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

// handBackLaneOf drives a drain's run to its implement await and hands the
// lane back with hb through the lane's own receipt.
func handBackLaneOf(t *testing.T, repo *gittest.Repo, runID string, o Options, hb LaneHandBack) {
	t.Helper()
	for range len(Sequence) {
		st, err := ReadState(repo.Root(), runID)
		if err != nil {
			t.Fatal(err)
		}
		if st.Lanes[0].Awaiting != nil {
			break
		}
		if _, err := advance(repo.Root(), runID, DefaultStages(), o); err != nil {
			t.Fatalf("advancing the drain's lane: %v", err)
		}
	}
	st, _ := ReadState(repo.Root(), runID)
	l := st.Lanes[0]
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), runID, "lane-1")
	rc := goodReceipt(t, runID, l, dir)
	rc.HandBack = &hb
	if _, err := Receipt(repo.Root(), runID, writeReceipt(t, dir, rc), DefaultStages(), o); err != nil {
		t.Fatalf("the lane's hand-back receipt: %v", err)
	}
}

func routeOf(t *testing.T, rs []DrainRoute, issue string) DrainRoute {
	t.Helper()
	for _, r := range rs {
		if r.Issue == issue {
			return r
		}
	}
	t.Fatalf("no route for %s in %+v", issue, rs)
	return DrainRoute{}
}

// TestADrainOpensOneIssueLaneAtATimeInTheDrainOrder is scope 4 as the drain
// runs it: the first eligible issue in the drain order gets a lane through
// Start, a second invocation while it is in progress opens nothing and names
// the run to drive, and every field hand-back is flagged naming its rule.
func TestADrainOpensOneIssueLaneAtATimeInTheDrainOrder(t *testing.T) {
	repo := drainRepo(t)
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Started || res.Start == nil || res.Lane == nil || res.Lane.Issue != secondIssue {
		t.Fatalf("a new drain opens a lane for the first issue in the drain order (documentation before bug): %+v", res)
	}
	st, err := ReadState(repo.Root(), res.Start.RunID)
	if err != nil || st.Key != secondIssue {
		t.Fatalf("the lane is the loop's issue-keyed run: %+v %v", st, err)
	}
	flag := routeOf(t, res.Flags, majorIssue)
	if flag.Route != RouteRule || flag.Rule != string(capture.RuleSeverity) || !strings.Contains(flag.Reason, "major") {
		t.Fatalf("a major issue is flagged naming the severity rule: %+v", flag)
	}

	again, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if again.Started || again.Start != nil || again.Lane == nil || again.Lane.RunID != res.Start.RunID || !strings.Contains(again.Next, "abcd implement step") {
		t.Fatalf("while a lane is in progress the drain opens nothing and names the run to drive: %+v", again)
	}
	if runs, _ := Runs(repo.Root()); len(runs) != 1 {
		t.Fatalf("one lane at a time: %d runs", len(runs))
	}
}

// TestALaneHandedBackIsRoutedByKindAndTheDrainMovesOn is scope 5: a lane that
// hands its issue back as a user-visible change is promoted to an intent draft
// with the issue gaining the intent in related_intents; one that turns on a
// trust rule is flagged with its question and nothing is minted; each is said
// in the summary, and the drain opens the next issue's lane.
func TestALaneHandedBackIsRoutedByKindAndTheDrainMovesOn(t *testing.T) {
	repo := drainRepo(t)
	first, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	handBackLaneOf(t, repo, first.Start.RunID, Options{}, LaneHandBack{Kind: HandBackUserVisible, Reason: "the page gains a flag users see"})
	recordPath, _ := filepath.Glob(filepath.Join(repo.Root(), ".abcd", "work", "issues", "open", secondIssue+"-*.md"))
	if len(recordPath) != 1 {
		t.Fatalf("one open record for %s: %v", secondIssue, recordPath)
	}
	recBefore, err := os.ReadFile(recordPath[0])
	if err != nil {
		t.Fatal(err)
	}
	second, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	r := routeOf(t, second.Routed, secondIssue)
	if r.Route != RoutePromoted || !strings.HasPrefix(r.Draft, "itd-") {
		t.Fatalf("a user moment is promoted to an intent draft: %+v", r)
	}
	lr, err := capture.List(capture.ListRequest{RepoRoot: repo.Root(), State: capture.StateOpen})
	if err != nil {
		t.Fatal(err)
	}
	for _, iss := range lr.Issues {
		if iss.ID == secondIssue && (len(iss.RelatedIntents) != 1 || iss.RelatedIntents[0] != r.Draft) {
			t.Fatalf("the issue gains the draft in related_intents: %+v", iss.RelatedIntents)
		}
	}
	recAfter, err := os.ReadFile(recordPath[0])
	if err != nil {
		t.Fatal(err)
	}
	var added, removed []string
	bl, al := strings.Split(string(recBefore), "\n"), strings.Split(string(recAfter), "\n")
	for _, ln := range al {
		if !slices.Contains(bl, ln) {
			added = append(added, ln)
		}
	}
	for _, ln := range bl {
		if !slices.Contains(al, ln) {
			removed = append(removed, ln)
		}
	}
	if len(removed) != 0 || len(added) != 1 || !strings.HasPrefix(added[0], "related_intents:") {
		t.Fatalf("the issue gains the intent in related_intents and nothing else: added %q, removed %q", added, removed)
	}
	if second.Start == nil || second.Lane.Issue != eligibleIssue {
		t.Fatalf("the drain moves on to the next eligible issue: %+v", second)
	}

	drafts := func() int {
		n, _ := os.ReadDir(filepath.Join(repo.Root(), ".abcd", "development", "intents", "drafts"))
		return len(n)
	}
	before := drafts()
	handBackLaneOf(t, repo, second.Start.RunID, Options{}, LaneHandBack{Kind: HandBackTrustRule, Reason: "may a guard ever skip the owner check?"})
	third, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	tr := routeOf(t, third.Routed, eligibleIssue)
	if tr.Route != RouteDecisionRecord || !strings.Contains(tr.Question, "owner check") || drafts() != before {
		t.Fatalf("a trust rule is flagged with its question and nothing is minted: %+v (drafts %d -> %d)", tr, before, drafts())
	}
	if third.Stopped != DrainStoppedEmpty || !third.Complete {
		t.Fatalf("with nothing eligible left the drain ends and says so: %+v", third)
	}
	if len(third.HandBacks) != 2 {
		t.Fatalf("the summary carries every hand-back the drain routed: %+v", third.HandBacks)
	}
}

// TestADrainStopsAtItsMaxAndNamesTheCap is scope 7's cap: --max <n> caps the
// lanes a drain opens, and at the cap the run reports and exits.
func TestADrainStopsAtItsMaxAndNamesTheCap(t *testing.T) {
	repo := drainRepo(t)
	first, err := Drain(repo.Root(), Options{}, DrainOptions{Max: 1})
	if err != nil {
		t.Fatal(err)
	}
	handBackLaneOf(t, repo, first.Start.RunID, Options{}, LaneHandBack{Kind: HandBackDesignFinding, Reason: "the flag's name is a design choice", Home: "an intent for the flags page"})
	capped, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if capped.Stopped != DrainStoppedCap || !capped.Complete || capped.Start != nil || !strings.Contains(capped.Next, "--max 1") {
		t.Fatalf("at the cap the drain stops and names it: %+v", capped)
	}
	if h := routeOf(t, capped.Routed, secondIssue); h.Route != RouteHome || !strings.Contains(h.Home, "flags page") {
		t.Fatalf("a design finding is flagged with its home: %+v", h)
	}
	if runs, _ := Runs(repo.Root()); len(runs) != 1 {
		t.Fatalf("no lane past the cap: %d runs", len(runs))
	}
	if _, err := Drain(repo.Root(), Options{}, DrainOptions{Max: -1}); err == nil {
		t.Fatal("a negative --max is refused")
	}
}

// TestADrainPausesAtItsWindowsEnd is scope 7's clock: at the window's end the
// drain writes next_eligible_at and exits, opening nothing; before that time a
// drain opens nothing; after it the next invocation continues.
func TestADrainPausesAtItsWindowsEnd(t *testing.T) {
	repo := drainRepo(t)
	c := &clock{t: time.Date(2026, 9, 30, 9, 0, 0, 0, time.UTC)}
	pace := "1/5"
	o := Options{Now: c.now, Pace: &pace}
	first, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	handBackLaneOf(t, repo, first.Start.RunID, Options{Now: c.now}, LaneHandBack{Kind: HandBackSecondPackage, Reason: "it reaches the site", Home: "the brief"})

	c.t = c.t.Add(2 * time.Minute)
	paused, err := Drain(repo.Root(), Options{Now: c.now}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	want := c.t.Add(5 * time.Minute)
	if paused.NextEligibleAt == nil || !paused.NextEligibleAt.Equal(want) || paused.Start != nil {
		t.Fatalf("at the window's end the drain writes next_eligible_at and opens nothing: %+v", paused)
	}
	raw, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(DrainStateRel)))
	if err != nil || !strings.Contains(string(raw), "\"next_eligible_at\"") {
		t.Fatalf("the drain's state file takes next_eligible_at: %v %s", err, raw)
	}

	c.t = c.t.Add(time.Minute)
	if still, err := Drain(repo.Root(), Options{Now: c.now}, DrainOptions{}); err != nil || still.Start != nil || still.NextEligibleAt == nil {
		t.Fatalf("before next_eligible_at a drain opens nothing: %+v %v", still, err)
	}

	c.t = want.Add(time.Second)
	resumed, err := Drain(repo.Root(), Options{Now: c.now}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Start == nil || resumed.Lane.Issue != eligibleIssue {
		t.Fatalf("after next_eligible_at the next invocation continues: %+v", resumed)
	}
}

// TestADrainRefusesWithoutTheRepositorysRule: the drain reads the rule as the
// dry run does, and refuses without it, writing nothing.
func TestADrainRefusesWithoutTheRepositorysRule(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	if _, err := Drain(repo.Root(), Options{}, DrainOptions{}); err == nil || !strings.Contains(err.Error(), "drain eligibility record") {
		t.Fatalf("a drain without the rule is refused naming it: %v", err)
	}
	runTierAbsent(t, repo.Root())
}
