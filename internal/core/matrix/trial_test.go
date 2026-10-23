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

// The acceptance criteria count a copy carrying a fallback receipt as a
// failure; a copy the compliance check excluded broke a rule of the run, not
// the model's task, so it is named and counted apart, neither a pass nor a
// failure (the SOTA line: reject rule-breaking runs before ranking).
func TestACopyTheComplianceCheckExcludedIsCountedApart(t *testing.T) {
	c := fullTrial()
	c.Status, c.Reason = StatusExcluded, "edited the hidden tests"
	if err := c.validate(); err != nil {
		t.Fatalf("a compliance exclusion is refused: %v", err)
	}
	if c.Failed() || c.Passed() || !c.Attempted() {
		t.Fatalf("a compliance exclusion: failed %v passed %v; want counted apart, neither", c.Failed(), c.Passed())
	}
	f := fullTrial()
	f.Status = StatusExcluded
	f.Fallbacks = []runner.FallbackReceipt{{Role: "builder", Asked: "runner:opencode", Reason: runner.ReasonFailed, Ran: "host"}}
	if !f.Failed() {
		t.Fatal("a copy carrying a fallback receipt is not counted a failure")
	}
}

// A trial record is read at a trust boundary (an imported Harbor or Inspect
// document becomes one), so every count is held to zero or more and to a
// bound, every cost to a readable figure in a known unit, and every string,
// down to the receipts it carries, to bounded plain text: nothing that could
// move a terminal's cursor, reorder a line or hide text reaches a record.
func TestATrialHoldsItsCountsAndTextToBounds(t *testing.T) {
	esc := "\x1b]0;x\a\x1b[2J"
	long := strings.Repeat("r", maxReasonBytes+1)
	cases := map[string]func(*TrialRecord){
		"negative prompt tokens":  func(t *TrialRecord) { t.Usage.PromptTokens = -1 },
		"unbounded cache reads":   func(t *TrialRecord) { t.Usage.CacheReadTokens = maxTokens + 1 },
		"negative cost":           func(t *TrialRecord) { t.Cost = &oracle.Cost{Amount: -1, Unit: oracle.CostUSD} },
		"cost in no known unit":   func(t *TrialRecord) { t.Cost = &oracle.Cost{Amount: 1, Unit: "eur"} },
		"unbounded model time":    func(t *TrialRecord) { t.ModelTimeMS = maxMS + 1 },
		"unbounded score":         func(t *TrialRecord) { t.Score = &Score{Passed: 1, Total: maxTests + 1} },
		"escape in harness":       func(t *TrialRecord) { t.HarnessVersion = esc },
		"escape in served model":  func(t *TrialRecord) { t.ModelServed = "\x1b[31mRED" },
		"bidi in asked model":     func(t *TrialRecord) { t.ModelAsked = "vendor/‮model" },
		"long served provider":    func(t *TrialRecord) { t.ServedProvider = strings.Repeat("p", oracle.MaxModelBytes+1) },
		"escape in reason":        func(t *TrialRecord) { t.Reason = "failed" + esc },
		"newline in reason":       func(t *TrialRecord) { t.Reason = "failed\nforged line" },
		"long reason":             func(t *TrialRecord) { t.Reason = long },
		"escape in score detail":  func(t *TrialRecord) { t.Score.Detail = esc },
		"escape in a setting":     func(t *TrialRecord) { t.Settings = oracle.Settings{"temp\x1b": json.RawMessage(`1`)} },
		"escape in a role":        func(t *TrialRecord) { t.Roles[0].Route.FinishReason = esc },
		"escape in a call":        func(t *TrialRecord) { t.Calls[0].Override = esc },
		"escape in a call record": func(t *TrialRecord) { t.Calls[0].ProviderCall.GenerationID = esc },
		"negative receipt tokens": func(t *TrialRecord) {
			u := *t.Roles[0].Route.Usage
			u.CompletionTokens = -5
			t.Roles[0].Route.Usage = &u
		},
		"escape in unpinned": func(t *TrialRecord) {
			t.Unpinned = []oracle.Unpinned{{Field: oracle.Field(esc), Reason: oracle.NotReported}}
		},
		"too many calls": func(t *TrialRecord) {
			for len(t.Calls) <= maxReceipts {
				t.Calls = append(t.Calls, t.Calls[0])
			}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			tr := fullTrial()
			mutate(&tr)
			if tr.validate() == nil {
				t.Fatal("validate admitted it")
			}
		})
	}
}
