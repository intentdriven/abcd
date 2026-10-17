package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/intentdriven/abcd/internal/core/oracle"
)

// The tests in this file are itd-166's runner half: a run records the model
// the runner passed beside what the harness says answered, with the token
// counts and cost its events carry, and lists every field of the fixed lists
// it could not pin with its reason.
//
// testdata/claude-stream-2.1.302.jsonl is a real stream: one call of the
// claude CLI 2.1.302 in print mode with --output-format stream-json
// --verbose, asking haiku to reply "ok". It was trimmed of what identifies the
// machine or the account (the hook events, the init event's paths, tools and
// plugins, the rate-limit details and the thinking signature) and its session
// id replaced; the usage, cost, model and version are as the harness sent
// them.
//
// testdata/opencode-run-1.18.31.jsonl is not a real stream: no opencode call
// was made for it. It is built from the shape opencode 1.18.31's run mode
// documents for `run --format json` (step_start, tool_use, reasoning, text and
// step_finish events, the step-finish part carrying reason, snapshot, cost and
// tokens{total,input,output,reasoning,cache{read,write}}), and like the real
// one it names no model.

// reasons indexes a record's list by field.
func reasons(us []oracle.Unpinned) map[oracle.Field]oracle.UnpinnedReason {
	m := map[oracle.Field]oracle.UnpinnedReason{}
	for _, u := range us {
		m[u.Field] = u.Reason
	}
	return m
}

// assertFixedLists holds every listed field to the fixed lists, once each.
func assertFixedLists(t *testing.T, us []oracle.Unpinned) {
	t.Helper()
	known := map[oracle.Field]bool{}
	for _, f := range append(append([]oracle.Field{}, oracle.AskedFields...), oracle.ServedFields...) {
		known[f] = true
	}
	seen := map[oracle.Field]bool{}
	for _, u := range us {
		if !known[u.Field] || seen[u.Field] {
			t.Errorf("listed field %q is off the fixed lists or listed twice: %v", u.Field, us)
		}
		seen[u.Field] = true
	}
}

func streamFixture(t *testing.T, name string) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(fakeStreamEnv, p)
	return p
}

func promptHash(req Request) string {
	sum := sha256.Sum256([]byte(prompt(req)))
	return hex.EncodeToString(sum[:])
}

// TestClaudeRecordsBothSidesFromARealStream is criterion 2: the run's answer
// carries the asked model the runner passed, the served models the result
// event's per-model usage names, its token counts and its cost (computed by
// the harness, and listed so), and the harness version from the init event.
func TestClaudeRecordsBothSidesFromARealStream(t *testing.T) {
	f := newFake(t, "stream", Claude)
	streamFixture(t, "claude-stream-2.1.302.jsonl")
	req := f.request("ruthless-reviewer")
	ans, _, err := newClaude("anthropic/haiku").Run(context.Background(), req)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if ans.Model != "claude-haiku-5-5" || ans.ModelAsked != "haiku" ||
		!reflect.DeepEqual(ans.ServedModels, []string{"claude-haiku-5-5"}) {
		t.Fatalf("models: served %q (%v), asked %q", ans.Model, ans.ServedModels, ans.ModelAsked)
	}
	if ans.HarnessVersion != "2.1.302" || ans.FinishReason != "end_turn" || ans.ServedProvider != "firstParty" {
		t.Fatalf("version %q, finish %q, provider %q", ans.HarnessVersion, ans.FinishReason, ans.ServedProvider)
	}
	wantUsage := oracle.Usage{PromptTokens: 2, CompletionTokens: 76, ReasoningTokens: 72, CacheReadTokens: 11228, CacheWriteTokens: 18650}
	if ans.Usage == nil || *ans.Usage != wantUsage {
		t.Fatalf("usage = %+v, want %+v", ans.Usage, wantUsage)
	}
	if ans.Cost == nil || ans.Cost.Unit != oracle.CostUSD || ans.Cost.Amount < 0.00388 || ans.Cost.Amount > 0.00389 {
		t.Fatalf("cost = %+v", ans.Cost)
	}
	if ans.PromptSHA256 != promptHash(req) {
		t.Fatalf("prompt hash = %q, want the prompt's", ans.PromptSHA256)
	}
	got := reasons(ans.Unpinned)
	want := map[oracle.Field]oracle.UnpinnedReason{
		oracle.AskedProviderPin: oracle.NotConfigured, oracle.AskedTemperature: oracle.NotConfigured,
		oracle.AskedTopP: oracle.NotConfigured, oracle.AskedSeed: oracle.NotConfigured,
		oracle.AskedEffort: oracle.NotConfigured, oracle.AskedContext: oracle.NotConfigured,
		oracle.ServedGenerationID: oracle.NotReported, oracle.ServedCost: oracle.HarnessComputed,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unpinned = %v, want %v", got, want)
	}
	assertFixedLists(t, ans.Unpinned)
}

// TestClaudeWithoutUsageListsEachMissing is criterion 4 on the claude
// runner: a stream that carries no usage, cost, version or stop reason lists
// each of them as not reported, and a run passed no model lists the asked
// model as not configured.
func TestClaudeWithoutUsageListsEachMissing(t *testing.T) {
	f := newFake(t, "ok", Claude)
	ans, _, err := newClaude("").Run(context.Background(), f.request("ruthless-reviewer"))
	if err != nil {
		t.Fatal(err)
	}
	if ans.Usage != nil || ans.Cost != nil || ans.ModelAsked != "" {
		t.Fatalf("answer = %+v", ans)
	}
	got := reasons(ans.Unpinned)
	for f, r := range map[oracle.Field]oracle.UnpinnedReason{
		oracle.AskedModel: oracle.NotConfigured, oracle.AskedHarnessVersion: oracle.NotReported,
		oracle.ServedPromptTokens: oracle.NotReported, oracle.ServedCompletionTokens: oracle.NotReported,
		oracle.ServedReasoningTokens: oracle.NotReported, oracle.ServedCacheReadTokens: oracle.NotReported,
		oracle.ServedCacheWriteTokens: oracle.NotReported, oracle.ServedCost: oracle.NotReported,
		oracle.ServedFinishReason: oracle.NotReported, oracle.ServedProvider: oracle.NotReported,
	} {
		if got[f] != r {
			t.Errorf("%s = %q, want %q", f, got[f], r)
		}
	}
	assertFixedLists(t, ans.Unpinned)
}

// TestOpenCodeNeverStampsTheAskedModelAsServed is criterion 3 and
// iss-2610161709506207: opencode's run mode names no model, so the served
// model is empty and listed as not reported, never a copy of the asked one,
// and the token counts and cost are summed across its step-finish events.
func TestOpenCodeNeverStampsTheAskedModelAsServed(t *testing.T) {
	f := newFake(t, "stream", OpenCode)
	streamFixture(t, "opencode-run-1.18.31.jsonl")
	req := f.request("ruthless-reviewer")
	ans, _, err := newOpenCode("local/qwen3-coder").Run(context.Background(), req)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if ans.Model != "" || ans.ModelAsked != "local/qwen3-coder" || len(ans.ServedModels) != 0 {
		t.Fatalf("served %q (%v), asked %q: the asked model must never stand in for the served one", ans.Model, ans.ServedModels, ans.ModelAsked)
	}
	wantUsage := oracle.Usage{PromptTokens: 1600, CompletionTokens: 300, ReasoningTokens: 50, CacheReadTokens: 350, CacheWriteTokens: 100}
	if ans.Usage == nil || *ans.Usage != wantUsage {
		t.Fatalf("usage = %+v, want %+v", ans.Usage, wantUsage)
	}
	if ans.Cost == nil || ans.Cost.Unit != oracle.CostUSD || ans.Cost.Amount < 0.001999 || ans.Cost.Amount > 0.002001 {
		t.Fatalf("cost = %+v, want 0.002 usd", ans.Cost)
	}
	if ans.FinishReason != "stop" || ans.Text != "done" || ans.PromptSHA256 != promptHash(req) {
		t.Fatalf("finish %q, text %q, hash %q", ans.FinishReason, ans.Text, ans.PromptSHA256)
	}
	got := reasons(ans.Unpinned)
	for f, r := range map[oracle.Field]oracle.UnpinnedReason{
		oracle.ServedModel: oracle.NotReported, oracle.ServedProvider: oracle.NotReported,
		oracle.ServedGenerationID: oracle.NotReported, oracle.ServedCost: oracle.HarnessComputed,
		oracle.AskedHarnessVersion: oracle.NotReported,
	} {
		if got[f] != r {
			t.Errorf("%s = %q, want %q", f, got[f], r)
		}
	}
	if _, listed := got[oracle.AskedModel]; listed {
		t.Errorf("the asked model the runner passed is listed: %v", ans.Unpinned)
	}
	assertFixedLists(t, ans.Unpinned)
}

// TestOpenCodeWithoutStepUsageListsItNotReported: a stream whose steps carry
// no tokens and no cost lists every count and the cost as not reported.
func TestOpenCodeWithoutStepUsageListsItNotReported(t *testing.T) {
	f := newFake(t, "ok", OpenCode)
	ans, _, err := newOpenCode("").Run(context.Background(), f.request("ruthless-reviewer"))
	if err != nil {
		t.Fatal(err)
	}
	if ans.Usage != nil || ans.Cost != nil || ans.Model != "" {
		t.Fatalf("answer = %+v", ans)
	}
	got := reasons(ans.Unpinned)
	for _, f := range []oracle.Field{oracle.ServedPromptTokens, oracle.ServedCost, oracle.ServedModel, oracle.ServedFinishReason} {
		if got[f] != oracle.NotReported {
			t.Errorf("%s = %q, want not_reported", f, got[f])
		}
	}
	if got[oracle.AskedModel] != oracle.NotConfigured {
		t.Errorf("asked model with none passed = %q", got[oracle.AskedModel])
	}
}

// TestTheRouteRecordKeepsTheAskedModelBesideTheServedOne: a dispatch through
// a runner puts both sides of its answer on the role's route record.
func TestTheRouteRecordKeepsTheAskedModelBesideTheServedOne(t *testing.T) {
	f := newFake(t, "stream", Claude, OpenCode)
	streamFixture(t, "opencode-run-1.18.31.jsonl")
	h := newHarness(t, mustLoad(t, routed(routedMachine), ""), true)
	out, err := h.d.Dispatch(context.Background(), f.request("ruthless-reviewer"))
	if err != nil {
		t.Fatal(err)
	}
	r := out.Receipt.Route
	if r.Ran != OpenCode || r.Model != "" || r.ModelAsked != "local/qwen3-coder" || r.Usage == nil || r.Cost == nil || len(r.Unpinned) == 0 {
		t.Fatalf("route = %+v", r)
	}
	if !reflect.DeepEqual(RecordOf(OpenCode, OpenCode, *out.Answer), r) {
		t.Fatalf("RecordOf = %+v, want the route the dispatch recorded %+v", RecordOf(OpenCode, OpenCode, *out.Answer), r)
	}
}

// TestHarnessStringsAreCleanedAndBounded: a served model, provider, finish
// reason or version the harness supplied is percent-encoded of hidden and
// control runes and bounded before it reaches a record.
func TestHarnessStringsAreCleanedAndBounded(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "stream.jsonl")
	long := make([]byte, 400)
	for i := range long {
		long[i] = 'v'
	}
	stream := `{"type":"system","subtype":"init","session_id":"s1","model":"claude-x","claude_code_version":"2.1\u001b[2J` + string(long) + `"}` + "\n" +
		`{"type":"result","subtype":"success","is_error":false,"result":"ok","session_id":"s1","stop_reason":"end‮turn",` +
		`"modelUsage":{"claude-x\u0007":{"provider":"first​Party"}}}` + "\n"
	if err := os.WriteFile(p, []byte(stream), 0o600); err != nil {
		t.Fatal(err)
	}
	ans, err := parseClaude([]byte(stream))
	if err != nil {
		t.Fatal(err)
	}
	vals := append([]string{ans.HarnessVersion, ans.FinishReason, ans.ServedProvider}, ans.ServedModels...)
	for _, v := range vals {
		for _, r := range v {
			if r < 0x20 || r == 0x7f || r == '‮' || r == '​' {
				t.Errorf("%q carries rune %U", v, r)
			}
		}
		if len(v) > maxHarnessText+len("...") {
			t.Errorf("%q is %d bytes, over the bound", v, len(v))
		}
	}
}
