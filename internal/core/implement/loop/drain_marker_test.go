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

// The decision-record marker on the issue (itd-82 decision 10;
// spc-2609212015054359 step 6): a trust-rule hand-back, a lane's or the host
// judgement's, writes `decision_needed:` holding its question onto the open
// issue and changes nothing else in the record; the dry run and the next
// drain pass the marked issue over with the marker as its reason.

// openRecordOf is the path and bytes of issue's open record.
func openRecordOf(t *testing.T, repo *gittest.Repo, issue string) (string, string) {
	t.Helper()
	m, _ := filepath.Glob(filepath.Join(repo.Root(), ".abcd", "work", "issues", "open", issue+"-*.md"))
	if len(m) != 1 {
		t.Fatalf("one open record for %s: %v", issue, m)
	}
	raw, err := os.ReadFile(m[0])
	if err != nil {
		t.Fatal(err)
	}
	return m[0], string(raw)
}

// lineDiff is the lines after holds that before does not, and those before
// holds that after does not.
func lineDiff(before, after string) (added, removed []string) {
	bl, al := strings.Split(before, "\n"), strings.Split(after, "\n")
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
	return added, removed
}

func dispositionOf(t *testing.T, vs []capture.DrainVerdict, issue string) capture.DrainVerdict {
	t.Helper()
	for _, v := range vs {
		if v.ID == issue {
			return v
		}
	}
	t.Fatalf("no disposition for %s in %+v", issue, vs)
	return capture.DrainVerdict{}
}

// TestALanesTrustRuleHandBackMarksTheOpenIssueAndNothingElse: the lane's
// trust-rule hand-back is flagged with its question in the summary, and the
// open issue gains `decision_needed:` holding that question, the one line the
// record changes; no decision record or draft is minted, and the drain counts
// the hand-back once, not again as the rule's flag.
func TestALanesTrustRuleHandBackMarksTheOpenIssueAndNothingElse(t *testing.T) {
	repo := drainRepo(t)
	first := drainToLane(t, repo, Options{}, DrainOptions{})
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	_, before := openRecordOf(t, repo, secondIssue)
	drafts := func() int {
		n, _ := os.ReadDir(filepath.Join(repo.Root(), ".abcd", "development", "intents", "drafts"))
		return len(n)
	}
	draftsBefore := drafts()
	const question = "may a guard ever skip the owner check?"
	handBackLaneOf(t, repo, first.Start.RunID, Options{}, LaneHandBack{Kind: HandBackTrustRule, Reason: question})
	next, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	r := routeOf(t, next.Routed, secondIssue)
	if r.Route != RouteDecisionRecord || r.Question != question || !strings.Contains(r.Wrote, "decision_needed: on "+secondIssue) {
		t.Fatalf("the trust rule is flagged with its question and says it marked the issue: %+v", r)
	}
	_, after := openRecordOf(t, repo, secondIssue)
	added, removed := lineDiff(before, after)
	if len(removed) != 0 || len(added) != 1 || added[0] != "decision_needed: \""+question+"\"" {
		t.Fatalf("the issue gains decision_needed: holding the question and nothing else: added %q, removed %q", added, removed)
	}
	if drafts() != draftsBefore {
		t.Fatalf("nothing is minted: drafts %d -> %d", draftsBefore, drafts())
	}
	// The move after reads the ledger the marker is on.
	after2, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if v := dispositionOf(t, after2.Dispositions, secondIssue); v.Rule != capture.RuleDecisionNeeded || !strings.Contains(v.Reason, question) {
		t.Fatalf("the marked issue reads as decision-needed on the move after: %+v", v)
	}
	if slices.ContainsFunc(after2.Flags, func(f DrainRoute) bool { return f.Issue == secondIssue }) {
		t.Fatalf("the drain's own hand-back is not flagged again as the rule's: %+v", after2.Flags)
	}
	if c := after2.Counts; c.DecisionRecords != 1 || c.RuleFlags != len(after2.Flags) {
		t.Fatalf("the hand-back is counted once, as a decision record: %+v", c)
	}
}

// TestAMarkedIssueIsPassedOverByTheDryRunAndTheNextDrain: the host
// judgement's trust-rule yes marks the issue as a lane's would; the dry run
// hands it back by the rule decision-needed with its question, and a later
// drain, whose state holds none of this one's hand-backs, passes it over by
// the marker alone, flagging it with that reason and asking no judgement of
// it.
func TestAMarkedIssueIsPassedOverByTheDryRunAndTheNextDrain(t *testing.T) {
	repo := drainRepo(t)
	first, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if first.Judging == nil || first.Judging.Issue != secondIssue {
		t.Fatalf("the drain asks the judgement over the documentation issue first: %+v", first)
	}
	const question = "may the page name a token?"
	yes := judge(t, repo, Options{}, first, JudgementYes, HandBackTrustRule, question)
	if r := routeOf(t, yes.Routed, secondIssue); r.From != DrainFromJudgement || r.Route != RouteDecisionRecord || !strings.Contains(r.Wrote, "decision_needed:") {
		t.Fatalf("the judgement's trust-rule yes is flagged and marked: %+v", r)
	}
	if _, rec := openRecordOf(t, repo, secondIssue); !strings.Contains(rec, "\ndecision_needed: \""+question+"\"\n") {
		t.Fatalf("the open issue carries the marker:\n%s", rec)
	}

	pv, err := DrainDryRun(repo.Root(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	v := dispositionOf(t, pv.Dispositions, secondIssue)
	if v.Outcome != capture.DrainHandBack || v.Rule != capture.RuleDecisionNeeded || !strings.Contains(v.Reason, "the question: "+question) {
		t.Fatalf("the dry run hands the marked issue back with the marker as its reason: %+v", v)
	}

	// A later drain: this one's state, and with it every hand-back it
	// routed, is gone, so only the marker on the record holds the issue.
	if err := os.Remove(filepath.Join(repo.Root(), filepath.FromSlash(DrainStateRel))); err != nil {
		t.Fatal(err)
	}
	later, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !later.Started || later.Judging == nil || later.Judging.Issue != eligibleIssue {
		t.Fatalf("the later drain asks no judgement of the marked issue and goes on to the next: %+v", later)
	}
	f := routeOf(t, later.Flags, secondIssue)
	if f.From != "field" || f.Route != RouteRule || f.Rule != string(capture.RuleDecisionNeeded) || !strings.Contains(f.Reason, "the question: "+question) {
		t.Fatalf("the later drain flags the marked issue with the marker as its reason: %+v", f)
	}
}

// TestAMarkerAKilledCallWroteIsNotWrittenTwice: a trust-rule hand-back routed
// again over a record already carrying its question leaves the record as it
// is and says the marker is there.
func TestAMarkerAKilledCallWroteIsNotWrittenTwice(t *testing.T) {
	repo := drainRepo(t)
	const question = "may a guard ever skip the owner check?"
	if _, err := capture.MarkDecisionNeeded(capture.DecisionNeededRequest{RepoRoot: repo.Root(), ID: secondIssue, Question: question}); err != nil {
		t.Fatal(err)
	}
	_, before := openRecordOf(t, repo, secondIssue)
	r, err := routeHandBack(repo.Root(), secondIssue, "lane", HandBack{Kind: HandBackTrustRule, Reason: question}, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, after := openRecordOf(t, repo, secondIssue); after != before || !strings.Contains(r.Wrote, "decision_needed: on "+secondIssue) {
		t.Fatalf("the marker is found, not written twice: %+v\n%s", r, after)
	}
}
