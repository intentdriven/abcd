package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
)

// The summary's counts at the front door (itd-82 scope 8,
// spc-2609212015054359 step 2): the text and --json both carry every count.

// TestTheDrainTextSummaryCarriesEveryCount: the text summary names the lanes
// by outcome, the pull requests, the hand-backs by route, the rule's flags,
// the judgements, the passed issues and the spend, and a spend whose run
// could not be read says it is a floor.
func TestTheDrainTextSummaryCarriesEveryCount(t *testing.T) {
	res := loop.DrainResult{State: loop.DrainStateRel, Rule: "adr-1", Order: "by category", Next: "drain again",
		Counts: loop.DrainCounts{LanesOpened: 3, InProgress: 1, PullRequests: 1, LanesHandedBack: 1, Promotions: 2,
			DecisionRecords: 1, Homes: 1, RuleFlags: 4, Judgements: 5, Passed: 1,
			Spend: loop.DrainSpend{AgentRuns: 7, Implementers: 3, Validators: 4, Unread: []string{"run-0000000000000009"}}}}
	var b bytes.Buffer
	renderDrainRun(&b, res)
	for _, want := range []string{
		"counts: 3 lane(s) opened (1 in progress, 1 pull request(s) opened, 1 handed back)",
		"hand-backs routed: 2 promoted, 1 flagged for a decision record, 1 flagged with a home",
		"4 flagged by the rule", "5 host judgement(s)", "1 passed",
		"spend:  at least 7 agent run(s) (3 implementer(s), 4 validator(s))",
		"run-0000000000000009",
	} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("the text summary does not carry %q:\n%s", want, b.String())
		}
	}
}

// TestTheDrainJSONSummaryCarriesEveryCount: a bare drain's --json carries the
// counts object with every count and the spend, from the first move on.
func TestTheDrainJSONSummaryCarriesEveryCount(t *testing.T) {
	repo := drainRuleRepo(t, drainrule.ProposalFrontmatter())
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	captureWithRemedy(t, "a nil map is written before it is made", "--category", "bug", "--remedy", "make the map first")
	captureWithRemedy(t, "the parser drops a record", "--category", "bug", "--severity", "major", "--remedy", "rewrite it")

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"drain", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("drain --json exited %d:\n%s%s", code, stdout.String(), stderr.String())
	}
	var res struct {
		Counts map[string]json.RawMessage `json:"counts"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &res); err != nil {
		t.Fatalf("--json is not JSON: %v\n%s", err, stdout.String())
	}
	for _, k := range []string{"lanes_opened", "in_progress", "pull_requests", "lanes_handed_back", "promotions",
		"decision_records", "homes", "rule_flags", "judgements", "passed", "spend", "hand_backs_by_pr", "hand_backs_without_pr"} {
		if _, ok := res.Counts[k]; !ok {
			t.Errorf("--json counts carry no %q: %s", k, stdout.String())
		}
	}
	var spend struct {
		AgentRuns    *int     `json:"agent_runs"`
		Implementers *int     `json:"implementers"`
		Validators   *int     `json:"validators"`
		Unread       []string `json:"unread"`
	}
	if err := json.Unmarshal(res.Counts["spend"], &spend); err != nil || spend.AgentRuns == nil || spend.Implementers == nil ||
		spend.Validators == nil || spend.Unread == nil {
		t.Errorf("the spend carries its agent runs, implementers, validators and unread runs: %v %s", err, res.Counts["spend"])
	}
	if string(res.Counts["rule_flags"]) != "1" {
		t.Errorf("the major issue is the one rule flag: %s", res.Counts["rule_flags"])
	}
}

// TestDrainSummaryHandBacksPerPullRequest (itd-2610301750162657): the
// summary names each pull request the drain opened with its hand-back count
// and kinds, one with none included, then the hand-backs no pull request
// carries, in the text and in --json; a drain with neither prints no line.
func TestDrainSummaryHandBacksPerPullRequest(t *testing.T) {
	res := loop.DrainResult{State: loop.DrainStateRel, Rule: "adr-1", Order: "by category", Next: "drain again",
		Counts: loop.DrainCounts{LanesOpened: 2, PullRequests: 2, Spend: loop.DrainSpend{Unread: []string{}},
			HandBacksByPR: []loop.DrainPRHandBacks{
				{PR: 12, Issue: "iss-1", HandBacks: 1, Kinds: []string{"closed"}},
				{PR: 13, Issue: "iss-2", HandBacks: 0, Kinds: []string{}},
			}, HandBacksNoPR: 3}}
	var b bytes.Buffer
	renderDrainRun(&b, res)
	for _, want := range []string{
		"  pull request #12, iss-1: 1 hand-back(s) (closed)\n",
		"  pull request #13, iss-2: 0 hand-back(s)\n",
		"  no pull request: 3 hand-back(s)\n",
	} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("the text summary does not carry %q:\n%s", want, b.String())
		}
	}
	if i, j := strings.Index(b.String(), "#13"), strings.Index(b.String(), "no pull request"); i < 0 || j < i {
		t.Errorf("the hand-backs with no pull request do not follow the pull requests:\n%s", b.String())
	}
	data, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	var back struct {
		Counts struct {
			ByPR []struct {
				PR        int      `json:"pr"`
				Issue     string   `json:"issue"`
				HandBacks int      `json:"hand_backs"`
				Kinds     []string `json:"kinds"`
			} `json:"hand_backs_by_pr"`
			NoPR *int `json:"hand_backs_without_pr"`
		} `json:"counts"`
	}
	if err := json.Unmarshal(data, &back); err != nil || len(back.Counts.ByPR) != 2 || back.Counts.ByPR[0].PR != 12 ||
		back.Counts.ByPR[0].HandBacks != 1 || back.Counts.ByPR[1].HandBacks != 0 || back.Counts.NoPR == nil || *back.Counts.NoPR != 3 {
		t.Fatalf("--json carries %s (%v)", data, err)
	}
	var none bytes.Buffer
	renderDrainRun(&none, loop.DrainResult{State: loop.DrainStateRel, Rule: "adr-1", Counts: loop.DrainCounts{Spend: loop.DrainSpend{Unread: []string{}}}})
	if strings.Contains(none.String(), "pull request #") || strings.Contains(none.String(), "no pull request") {
		t.Errorf("a drain with no pull request and no hand-back prints a line for them:\n%s", none.String())
	}
}
