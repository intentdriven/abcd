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
		"decision_records", "homes", "rule_flags", "judgements", "passed", "spend"} {
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
