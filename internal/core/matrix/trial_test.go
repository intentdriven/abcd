package matrix

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// fullTrial is a trial with every field set, so a test over it notices a field
// a mapping drops.
func fullTrial() TrialRecord {
	start := time.Date(2026, 10, 23, 1, 2, 3, 0, time.UTC)
	usage := oracle.Usage{PromptTokens: 1200, CompletionTokens: 340, ReasoningTokens: 56, CacheReadTokens: 7000, CacheWriteTokens: 800}
	cost := oracle.Cost{Amount: 0.0123, Unit: oracle.CostUSD}
	return TrialRecord{
		SchemaVersion:  TrialSchema,
		Run:            "mx-261023012949-ab12",
		Key:            TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 2},
		Kind:           KindCode,
		Intent:         "itd-2610161733316518",
		FrozenCommit:   strings.Repeat("c", 40),
		Source:         SourceOpenRouter,
		Runner:         "opencode",
		HarnessVersion: "1.2.3",
		ModelAsked:     "vendor/model",
		ModelServed:    "vendor/model-2026-10",
		ServedProvider: "provider-a",
		Settings:       oracle.Settings{"temperature": json.RawMessage(`0.6`)},
		Status:         StatusFailed,
		Reason:         "hidden tests failed",
		Score:          &Score{Passed: 7, Total: 9, Detail: "two edge cases"},
		Roles: []runner.RoleReceipt{{
			Role: "builder", Brief: "brief.md", Contract: "receipt.json",
			Route: runner.RouteRecord{Asked: "runner:opencode", Ran: "runner:opencode", Model: "vendor/model-2026-10",
				Sides: runner.Sides{ModelAsked: "vendor/model", ServedModels: []string{"vendor/model-2026-10"}, ServedProvider: "provider-a",
					FinishReason: "stop", HarnessVersion: "1.2.3", PromptSHA256: strings.Repeat("9", 64), Usage: &usage, Cost: &cost,
					Unpinned: []oracle.Unpinned{{Field: oracle.ServedCost, Reason: oracle.HarnessComputed}}}},
		}},
		Calls: []oracle.ReceiptRoute{{
			Agent: "reviewer", TierAsked: "", ConnectionTried: "or", ConnectionUsed: "or", Override: "reviewer=runner:opencode",
			SettingsSent: oracle.Settings{}, ModelReported: "vendor/model",
			ProviderCall: &oracle.CallRecord{Provider: "openrouter", ModelAsked: "vendor/model", ModelReported: "vendor/model-2026-10",
				PromptSHA256: strings.Repeat("8", 64), AdapterAdded: []string{"stream_options"}, ServedProvider: "provider-a",
				GenerationID: "gen-1", FinishReason: "stop", Usage: &usage, Cost: &oracle.Cost{Amount: 0.01, Unit: oracle.CostCredits},
				Unpinned: []oracle.Unpinned{}},
			Unpinned: nil,
		}},
		Fallbacks:   []runner.FallbackReceipt{},
		Usage:       usage,
		Cost:        &cost,
		StartedAt:   start,
		FinishedAt:  start.Add(95 * time.Second),
		ModelTimeMS: 61000,
		GateWaitMS:  12000,
		Unpinned:    []oracle.Unpinned{{Field: oracle.ServedCost, Reason: oracle.HarnessComputed}, {Field: oracle.AskedSeed, Reason: oracle.NotConfigured}},
	}
}

// Decision 4: each copy is keyed by what it is a copy of (the task, which for
// a code task is its intent), the cell and the repeat, and the key is the
// record's path in the run.
func TestATrialIsKeyedByIntentCellAndRepeat(t *testing.T) {
	k := TrialKey{Task: "feature-a", Cell: "or-cheap", Repeat: 2}
	if k.String() != "feature-a/or-cheap/2" || k.rel() != "trials/feature-a/or-cheap/2.json" {
		t.Fatalf("key = %q, rel %q", k.String(), k.rel())
	}
	for _, bad := range []TrialKey{{"", "c", 1}, {"t", "", 1}, {"t", "c", 0}, {"T/../x", "c", 1}, {"t", "c", MaxRepeats + 1}} {
		if bad.validate() == nil {
			t.Errorf("%+v was admitted as a trial key", bad)
		}
	}
	tr := fullTrial()
	if err := tr.validate(); err != nil {
		t.Fatalf("the full trial is refused: %v", err)
	}
	tr.Intent = ""
	if tr.validate() == nil {
		t.Error("a code trial with no intent was admitted")
	}
}

// Decision 14: the runners' receipts, with both sides of each run, are carried
// as they are, never copied into a parallel shape.
func TestATrialCarriesTheRoleReceiptsBothSidesUnchanged(t *testing.T) {
	tr := fullTrial()
	js, err := json.Marshal(tr)
	if err != nil {
		t.Fatal(err)
	}
	var back TrialRecord
	if err := json.Unmarshal(js, &back); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back.Roles, tr.Roles) || !reflect.DeepEqual(back.Calls, tr.Calls) {
		t.Fatalf("the receipts changed in the trial record:\n%+v\n%+v", back.Roles, tr.Roles)
	}
	role, _ := json.Marshal(tr.Roles[0])
	if !strings.Contains(string(js), string(role)) {
		t.Fatalf("the role receipt is not embedded as runner writes it:\n%s\nnot in\n%s", role, js)
	}
	if reflect.TypeOf(TrialRecord{}.Roles).Elem() != reflect.TypeOf(runner.RoleReceipt{}) ||
		reflect.TypeOf(TrialRecord{}.Calls).Elem() != reflect.TypeOf(oracle.ReceiptRoute{}) ||
		reflect.TypeOf(TrialRecord{}.Fallbacks).Elem() != reflect.TypeOf(runner.FallbackReceipt{}) {
		t.Fatal("the trial record holds a copy of a receipt type, not the type")
	}
}

// Decision 6: an outage before a copy starts is retried on the same pin and
// never scored as the model's failure.
func TestAnOutageBeforeStartIsNotStartedNeverFailed(t *testing.T) {
	tr := TrialRecord{SchemaVersion: TrialSchema, Run: "mx-261023012949-ab12", Key: TrialKey{"feature-a", "or-cheap", 1},
		Kind: KindCode, Intent: "itd-1", FrozenCommit: strings.Repeat("c", 40), Source: SourceOpenRouter, Runner: "opencode",
		Status: StatusNotStarted, Reason: "provider down before start"}
	if err := tr.validate(); err != nil {
		t.Fatalf("a not-started trial is refused: %v", err)
	}
	if tr.Attempted() || tr.Failed() {
		t.Fatal("a copy that never started counts as an attempt or a failure")
	}
	tr.Score = &Score{Passed: 0, Total: 9}
	if tr.validate() == nil {
		t.Fatal("a not-started trial carrying a score was admitted")
	}
	tr.Score = nil
	tr.Usage.PromptTokens = 10
	if tr.validate() == nil {
		t.Fatal("a not-started trial carrying spent tokens was admitted")
	}

	// A copy that failed, refused or answered unparsably in its cell is a
	// failed attempt; one carrying a fallback receipt is excluded and still
	// counts as a failure, never as the model's pass.
	f := fullTrial()
	if !f.Attempted() || !f.Failed() {
		t.Fatal("a failed trial is not a failed attempt")
	}
	f.Status = StatusPassed
	f.Fallbacks = []runner.FallbackReceipt{{Role: "builder", Asked: "runner:opencode", Reason: runner.ReasonFailed, Ran: "host"}}
	if f.validate() == nil {
		t.Fatal("a passed trial carrying a fallback receipt was admitted")
	}
	f.Status = StatusExcluded
	if err := f.validate(); err != nil || !f.Failed() {
		t.Fatalf("an excluded trial: %v, failed %v; want admitted and counted a failure", err, f.Failed())
	}
	// An unexamined copy is reported apart: neither a pass nor a failure.
	u := fullTrial()
	u.Status = StatusUnexamined
	if u.validate() != nil || u.Failed() || u.Passed() {
		t.Fatal("an unexamined trial is ranked")
	}
	u.Status = "skipped"
	if u.validate() == nil {
		t.Fatal("an unknown status was admitted")
	}
}
