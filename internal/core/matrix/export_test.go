package matrix

// The upstream shapes these tests hold the mapping to were read from source,
// at pinned commits, on 2026-10-23:
//
//   - Inspect AI, UKGovernmentBEIS/inspect_ai at
//     157980923b2993d2496a8d150c81f32465f15991 (EvalLog version 2):
//     src/inspect_ai/log/_log.py (EvalSample, EvalSpec, EvalRevision),
//     src/inspect_ai/model/_model_output.py (ModelOutput, ModelUsage,
//     ModelFallback), src/inspect_ai/scorer/_metric.py (Score),
//     src/inspect_ai/_util/error.py (EvalError).
//   - Harbor, harbor-framework/harbor at
//     d0fe0b24703f4e53806be5cb3b5e07d622aebbfa (package 0.24.0):
//     src/harbor/models/trial/result.py (TrialResult, AgentInfo,
//     ExceptionInfo), models/trial/config.py (TrialConfig),
//     models/agent/context.py (AgentContext), models/verifier/result.py
//     (VerifierResult), models/task/id.py (LocalTaskId).
//
// The fixtures were not run through the upstream validators
// (EvalLog.model_validate, TrialResult.model_validate); that stays open.

import (
	"bytes"
	"encoding/json"
	"flag"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
)

var update = flag.Bool("update", false, "rewrite the export fixtures from fullTrial")

// shape is what one object of an upstream document may carry: the names the
// source declares, the ones it requires, and the members that are maps (their
// values held to the shape at "<path>.*") or free-form (not walked).
type shape struct {
	keys     map[string][]string // path -> declared names
	required map[string][]string // path -> required names
	maps     []string            // paths whose values are keyed by data
	free     []string            // paths that are free-form
}

// harborShape is TrialResult at d0fe0b24703f. agent_result.metadata is
// AgentContext's metadata dict (models/agent/context.py), where abcd's own
// record goes.
var harborShape = shape{
	keys: map[string][]string{
		"": {"id", "task_name", "trial_name", "trial_uri", "task_id", "task_checksum", "source", "config", "agent_info",
			"agent_result", "verifier_result", "exception_info", "started_at", "finished_at",
			"environment_setup", "agent_setup", "agent_execution", "verifier", "step_results"},
		"task_id":               {"path", "git_url", "git_commit_id", "org", "name", "ref"},
		"config":                {"task", "agent", "job_id"},
		"config.task":           {"path"},
		"config.agent":          {"model_name"},
		"agent_info":            {"name", "version", "model_info"},
		"agent_info.model_info": {"name", "provider"},
		"agent_result":          {"n_input_tokens", "n_cache_tokens", "n_output_tokens", "cost_usd", "model_usage", "metadata"},
		"verifier_result":       {"rewards"},
		"exception_info":        {"exception_type", "exception_message", "exception_traceback", "occurred_at"},
	},
	required: map[string][]string{
		"":               {"task_name", "trial_name", "trial_uri", "task_id", "task_checksum", "config", "agent_info"},
		"config":         {"task"},
		"agent_info":     {"name", "version"},
		"exception_info": {"exception_type", "exception_message", "exception_traceback", "occurred_at"},
	},
	free: []string{"verifier_result.rewards", "agent_result.metadata", "agent_result.model_usage"},
}

// inspectShape is EvalSample at 157980923b29.
var inspectShape = shape{
	keys: map[string][]string{
		"": {"id", "epoch", "input", "target", "messages", "output", "scores", "metadata", "model_usage", "role_usage",
			"model_fallbacks", "started_at", "completed_at", "total_time", "working_time", "uuid", "error", "error_retries", "limit"},
		"output":            {"model", "completion", "fallback"},
		"scores.*":          {"value"},
		"model_usage.*":     {"input_tokens", "output_tokens", "total_tokens", "input_tokens_cache_write", "input_tokens_cache_read", "reasoning_tokens", "total_cost"},
		"model_fallbacks[]": {"model", "fallback_model", "count", "metadata"},
		"error":             {"message", "traceback", "traceback_ansi"},
	},
	required: map[string][]string{
		"":         {"id", "epoch", "input", "target"},
		"scores.*": {"value"},
		"error":    {"message", "traceback", "traceback_ansi"},
	},
	maps: []string{"scores", "model_usage"},
	free: []string{"metadata"},
}

// check walks doc and reports every name the upstream shape does not declare
// and every required name missing.
func (s shape) check(t *testing.T, doc []byte) {
	t.Helper()
	var v any
	if err := json.Unmarshal(doc, &v); err != nil {
		t.Fatal(err)
	}
	s.walk(t, "", v)
}

func (s shape) walk(t *testing.T, p string, v any) {
	t.Helper()
	if slices.Contains(s.free, p) {
		return
	}
	switch x := v.(type) {
	case []any:
		for _, e := range x {
			s.walk(t, p+"[]", e)
		}
	case map[string]any:
		if slices.Contains(s.maps, p) {
			for _, e := range x {
				s.walk(t, p+".*", e)
			}
			return
		}
		declared, ok := s.keys[p]
		if !ok {
			t.Errorf("%q: an object where the upstream shape has none", p)
			return
		}
		for k, e := range x {
			if !slices.Contains(declared, k) {
				t.Errorf("%q carries %q, a name the upstream source does not declare", p, k)
				continue
			}
			s.walk(t, join(p, k), e)
		}
		for _, k := range s.required[p] {
			if _, ok := x[k]; !ok {
				t.Errorf("%q lacks %q, which the upstream source requires", p, k)
			}
		}
	}
}

func join(p, k string) string {
	if p == "" {
		return k
	}
	return p + "." + k
}

// The upstream minimal documents the field note built from required fields
// alone: the local target types read every name they carry.
const (
	upstreamInspectSample = `{"id": 1, "epoch": 1, "input": "What is 2+2?", "target": "4",
 "output": {"model": "openai/gpt-x", "completion": "4"},
 "scores": {"match": {"value": "C"}}}`
	upstreamHarborResult = `{"task_name": "t", "trial_name": "t__abc1234", "trial_uri": "file:///jobs/j/t__abc1234",
 "task_id": {"path": "tasks/t"}, "task_checksum": "sha256:0",
 "config": {"task": {"path": "tasks/t"}},
 "agent_info": {"name": "a", "version": "1"},
 "verifier_result": {"rewards": {"reward": 1}}}`
)

func TestTheLocalTargetsReadTheUpstreamMinimalDocuments(t *testing.T) {
	inspectShape.check(t, []byte(upstreamInspectSample))
	harborShape.check(t, []byte(upstreamHarborResult))
	var s InspectSample
	if err := jsonstrict.Decode([]byte(upstreamInspectSample), &s); err != nil {
		t.Fatalf("InspectSample does not read the upstream minimal sample: %v", err)
	}
	var h HarborTrialResult
	if err := jsonstrict.Decode([]byte(upstreamHarborResult), &h); err != nil {
		t.Fatalf("HarborTrialResult does not read the upstream minimal result: %v", err)
	}
	// Neither came from abcd, so neither is a trial record.
	if _, err := s.Trial(); err == nil || !strings.Contains(err.Error(), "abcd") {
		t.Errorf("an Inspect sample abcd did not write became a trial: %v", err)
	}
	if _, err := h.Trial(); err == nil || !strings.Contains(err.Error(), "abcd") {
		t.Errorf("a Harbor result abcd did not write became a trial: %v", err)
	}
}

func fixture(t *testing.T, name string, got []byte) []byte {
	t.Helper()
	p := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(p, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return want
}

func indent(t *testing.T, v any) []byte {
	t.Helper()
	js, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(js, '\n')
}

func TestTheTrialRecordMapsOntoAnInspectSampleFixture(t *testing.T) {
	got := indent(t, inspectOf(t, fullTrial()))
	want := fixture(t, "inspect-eval-sample.json", got)
	if !bytes.Equal(got, want) {
		t.Fatalf("the Inspect export drifted from its fixture (go test -update rewrites it):\n%s", got)
	}
	inspectShape.check(t, want)
	var s map[string]json.RawMessage
	_ = json.Unmarshal(want, &s)
	if string(s["id"]) != `"feature-a"` || string(s["epoch"]) != "2" {
		t.Errorf("id %s epoch %s; want the task and the repeat", s["id"], s["epoch"])
	}
}

func TestTheTrialRecordMapsOntoAHarborTrialResultFixture(t *testing.T) {
	got := indent(t, harborOf(t, fullTrial()))
	want := fixture(t, "harbor-trial-result.json", got)
	if !bytes.Equal(got, want) {
		t.Fatalf("the Harbor export drifted from its fixture (go test -update rewrites it):\n%s", got)
	}
	harborShape.check(t, want)
}

// Both fixtures read back into exactly the record they were written from.
func TestATrialRoundTripsThroughBothFixtures(t *testing.T) {
	want := fullTrial()
	for name, read := range map[string]func([]byte) (TrialRecord, error){
		"inspect-eval-sample.json": func(b []byte) (TrialRecord, error) {
			var s InspectSample
			if err := jsonstrict.Decode(b, &s); err != nil {
				return TrialRecord{}, err
			}
			return s.Trial()
		},
		"harbor-trial-result.json": func(b []byte) (TrialRecord, error) {
			var h HarborTrialResult
			if err := jsonstrict.Decode(b, &h); err != nil {
				return TrialRecord{}, err
			}
			return h.Trial()
		},
	} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		got, err := read(data)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s does not read back into the record it was written from:\ngot  %+v\nwant %+v", name, got, want)
		}
	}
}

// Every field of the trial record lands in both mappings: the full trial sets
// every field (so a field added to the record and not to the test fails here),
// and it, a trial that never started and one costed in credits each come back
// unchanged through both.
func TestEveryTrialFieldHasADestinationInBothMappings(t *testing.T) {
	full := fullTrial()
	v := reflect.ValueOf(full)
	for i := range v.NumField() {
		if v.Field(i).IsZero() {
			t.Errorf("fullTrial leaves %s unset; set it so the mappings are held to it", v.Type().Field(i).Name)
		}
	}
	credits := fullTrial()
	credits.Cost = &oracle.Cost{Amount: 3.5, Unit: oracle.CostCredits}
	credits.ModelServed, credits.ServedProvider = "", "provider-b"
	credits.Kind, credits.Intent = KindCheck, ""
	credits.Status = StatusExcluded
	credits.Fallbacks = []runner.FallbackReceipt{{At: time.Date(2026, 10, 23, 1, 3, 0, 0, time.UTC), Role: "builder",
		Asked: "runner:opencode", Reason: runner.ReasonRefused, Detail: "refused", Ran: "none"}}
	sparse := TrialRecord{SchemaVersion: TrialSchema, Run: "mx-261023012949-ab12", Key: TrialKey{"doc-page", "host-big", 1},
		Kind: KindWriting, FrozenCommit: strings.Repeat("c", 40), Source: SourceHost, Runner: "host", Status: StatusNotStarted}
	for name, tr := range map[string]TrialRecord{"full": full, "credits": credits, "not started": sparse} {
		if err := tr.validate(); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		h, err := roundTripHarbor(tr)
		if err != nil || !reflect.DeepEqual(h, tr) {
			t.Errorf("%s through Harbor: %v\ngot  %+v\nwant %+v", name, err, h, tr)
		}
		s, err := roundTripInspect(tr)
		if err != nil || !reflect.DeepEqual(s, tr) {
			t.Errorf("%s through Inspect: %v\ngot  %+v\nwant %+v", name, err, s, tr)
		}
		harborShape.check(t, indent(t, harborOf(t, tr)))
		inspectShape.check(t, indent(t, inspectOf(t, tr)))
	}
}

func roundTripHarbor(tr TrialRecord) (TrialRecord, error) {
	v, err := tr.Harbor()
	if err != nil {
		return TrialRecord{}, err
	}
	js, err := json.Marshal(v)
	if err != nil {
		return TrialRecord{}, err
	}
	var h HarborTrialResult
	if err := jsonstrict.Decode(js, &h); err != nil {
		return TrialRecord{}, err
	}
	return h.Trial()
}

func roundTripInspect(tr TrialRecord) (TrialRecord, error) {
	v, err := tr.Inspect()
	if err != nil {
		return TrialRecord{}, err
	}
	js, err := json.Marshal(v)
	if err != nil {
		return TrialRecord{}, err
	}
	var s InspectSample
	if err := jsonstrict.Decode(js, &s); err != nil {
		return TrialRecord{}, err
	}
	return s.Trial()
}

// Harbor counts cached prompt tokens inside n_input_tokens; Inspect's
// input_tokens, like abcd's prompt count, leaves them out.
func TestTheCachedTokensAreCountedTheWayEachTargetCountsThem(t *testing.T) {
	tr := fullTrial()
	u := tr.Usage
	a := harborOf(t, tr).AgentResult
	if a == nil || a.NInputTokens != u.PromptTokens+u.CacheReadTokens+u.CacheWriteTokens || a.NCacheTokens != u.CacheReadTokens || a.NOutputTokens != u.CompletionTokens {
		t.Fatalf("Harbor counts = %+v; want input %d including cache, cache %d", a, u.PromptTokens+u.CacheReadTokens+u.CacheWriteTokens, u.CacheReadTokens)
	}
	if a.CostUSD == nil || *a.CostUSD != tr.Cost.Amount {
		t.Fatalf("Harbor cost_usd = %v, want %v", a.CostUSD, tr.Cost.Amount)
	}
	mu := inspectOf(t, tr).ModelUsage
	got, ok := mu[tr.ModelServed]
	if len(mu) != 1 || !ok || got.InputTokens != u.PromptTokens || got.InputTokensCacheRead != u.CacheReadTokens ||
		got.InputTokensCacheWrite != u.CacheWriteTokens || got.ReasoningTokens != u.ReasoningTokens || got.OutputTokens != u.CompletionTokens ||
		got.TotalCost == nil || *got.TotalCost != tr.Cost.Amount {
		t.Fatalf("Inspect model_usage = %+v; want the served model's counts with cache apart", mu)
	}
	// A count that does not add up is refused, never read as a negative.
	h := harborOf(t, tr)
	h.AgentResult.NInputTokens = 5
	if _, err := h.Trial(); err == nil {
		t.Fatal("a Harbor result whose input count is below its cache count was read")
	}
}

// Harbor has no attempt index, so the repeat rides in the trial name (with the
// cell); Inspect's epoch is the repeat.
func TestTheRepeatLandsInTheTrialNameAndTheEpoch(t *testing.T) {
	tr := fullTrial()
	h := harborOf(t, tr)
	if h.TrialName != "feature-a__or-cheap__2" || h.TaskName != "feature-a" {
		t.Fatalf("trial_name %q task_name %q", h.TrialName, h.TaskName)
	}
	if s := inspectOf(t, tr); s.Epoch != 2 {
		t.Fatalf("epoch = %d, want the repeat", s.Epoch)
	}
	for _, bad := range []string{"feature-a__or-cheap", "feature-a__or-cheap__0", "other__or-cheap__2", "feature-a__or-cheap__2__x", "feature-a__Or__2"} {
		h := harborOf(t, tr)
		h.TrialName = bad
		if _, err := h.Trial(); err == nil {
			t.Errorf("trial_name %q was read", bad)
		}
	}
	// Harbor keeps the asked model in the trial's agent config and the served
	// one in agent_info.
	if h.Config.Agent == nil || h.Config.Agent.ModelName != tr.ModelAsked || h.AgentInfo.ModelInfo == nil || h.AgentInfo.ModelInfo.Name != tr.ModelServed {
		t.Fatalf("Harbor config.agent %+v, agent_info %+v", h.Config.Agent, h.AgentInfo)
	}
}

// Inspect's ModelFallback is a different model serving in place of the one
// asked for. A dated snapshot of the asked model (vendor/model served as
// vendor/model-2026-10) is the same model, so it is no fallback; a different
// model is.
func TestOnlyADifferentServedModelIsAnInspectFallback(t *testing.T) {
	tr := fullTrial()
	if s := inspectOf(t, tr); len(s.ModelFallbacks) != 0 {
		t.Fatalf("a dated snapshot of the asked model was exported as a fallback: %+v", s.ModelFallbacks)
	}
	for _, served := range []string{"vendor/model@20261001", "vendor/model:2026-10-01", "vendor/model-20261001", "vendor/model"} {
		tr.ModelServed = served
		if s := inspectOf(t, tr); len(s.ModelFallbacks) != 0 {
			t.Errorf("served %q: exported as a fallback: %+v", served, s.ModelFallbacks)
		}
	}
	for _, served := range []string{"other/model", "vendor/model-mini", "vendor/modelx"} {
		tr.ModelServed = served
		s := inspectOf(t, tr)
		if len(s.ModelFallbacks) != 1 || s.ModelFallbacks[0].Model != tr.ModelAsked || s.ModelFallbacks[0].FallbackModel != served {
			t.Errorf("served %q: model_fallbacks = %+v; want one fallback from the asked model", served, s.ModelFallbacks)
		}
		if back, err := roundTripInspect(tr); err != nil || !reflect.DeepEqual(back, tr) {
			t.Errorf("served %q: round trip %v", served, err)
		}
		inspectShape.check(t, indent(t, s))
	}
	// A document whose fallback list is not the one abcd writes is not read.
	tr = fullTrial()
	s := inspectOf(t, tr)
	s.ModelFallbacks = []InspectModelFallback{{Model: "x", FallbackModel: "y", Count: 1}}
	if _, err := s.Trial(); err == nil {
		t.Fatal("an Inspect sample whose model_fallbacks disagree with its models was read")
	}
}

// Harbor's ModelInfo.provider is the model's vendor (the prefix of its id),
// not the host that served it; the serving provider rides in abcd's metadata.
func TestHarborsModelProviderIsTheVendorAndTheServingProviderRidesInTheMetadata(t *testing.T) {
	tr := fullTrial()
	h := harborOf(t, tr)
	if mi := h.AgentInfo.ModelInfo; mi == nil || mi.Name != tr.ModelServed || mi.Provider != "vendor" {
		t.Fatalf("model_info = %+v; want the served model and its vendor", h.AgentInfo.ModelInfo)
	}
	if !strings.Contains(string(h.AgentResult.Metadata[metadataKey]), `"served_provider":"provider-a"`) {
		t.Fatalf("the serving provider is not in the metadata: %s", h.AgentResult.Metadata[metadataKey])
	}
	h.AgentInfo.ModelInfo.Provider = "provider-a"
	if _, err := h.Trial(); err == nil {
		t.Fatal("a Harbor result whose model provider is not the served model's vendor was read")
	}
}

// A failure that was scored is a result, not an error: Inspect's error is the
// "error that halted sample" and Harbor retries an unclassified exception and
// drops its reward, so a scored failure travels in its score alone and only a
// copy with no score (its runner failed, refused or answered unparsably) is
// exported as an error.
func TestOnlyAnUnscoredFailureIsExportedAsAnError(t *testing.T) {
	scored := fullTrial()
	if h := harborOf(t, scored); h.ExceptionInfo != nil || h.VerifierResult == nil {
		t.Fatalf("a scored failure: exception_info %+v verifier_result %+v; want the reward and no exception", h.ExceptionInfo, h.VerifierResult)
	}
	if s := inspectOf(t, scored); s.Error != nil || s.Scores == nil {
		t.Fatalf("a scored failure: error %+v scores %+v; want the score and no error", s.Error, s.Scores)
	}
	unscored := fullTrial()
	unscored.Score, unscored.Reason = nil, "the runner answered unparsably"
	h := harborOf(t, unscored)
	if h.ExceptionInfo == nil || h.ExceptionInfo.ExceptionMessage != unscored.Reason || h.VerifierResult != nil {
		t.Fatalf("an unscored failure: exception_info %+v verifier_result %+v", h.ExceptionInfo, h.VerifierResult)
	}
	if s := inspectOf(t, unscored); s.Error == nil || s.Error.Message != unscored.Reason {
		t.Fatalf("an unscored failure: error %+v", s.Error)
	}
	// The error members are held to the upstream shapes as the fixtures are.
	harborShape.check(t, indent(t, h))
	inspectShape.check(t, indent(t, inspectOf(t, unscored)))
	for name, rt := range map[string]func(TrialRecord) (TrialRecord, error){"Harbor": roundTripHarbor, "Inspect": roundTripInspect} {
		if back, err := rt(unscored); err != nil || !reflect.DeepEqual(back, unscored) {
			t.Errorf("an unscored failure through %s: %v", name, err)
		}
	}
	// A document that marks a scored copy as an error is not abcd's.
	h = harborOf(t, scored)
	h.ExceptionInfo = &HarborExceptionInfo{ExceptionType: "abcd.failed", ExceptionMessage: scored.Reason, OccurredAt: scored.FinishedAt}
	if _, err := h.Trial(); err == nil {
		t.Fatal("a Harbor result marking a scored failure as an exception was read")
	}
	s := inspectOf(t, scored)
	s.Error = &InspectEvalError{Message: scored.Reason}
	if _, err := s.Trial(); err == nil {
		t.Fatal("an Inspect sample marking a scored failure as an error was read")
	}
}

// Token counts are read with checked arithmetic both ways: a negative count,
// a count past its bound and a sum that does not add up are refused, never
// wrapped into a plausible figure.
func TestTokenCountsAreCheckedBothWays(t *testing.T) {
	tr := fullTrial()
	for name, mutate := range map[string]func(*HarborAgentContext){
		"wrapping input":     func(a *HarborAgentContext) { a.NInputTokens, a.NCacheTokens = math.MinInt64, 1 },
		"input past bound":   func(a *HarborAgentContext) { a.NInputTokens = maxTokens*3 + 1 },
		"negative output":    func(a *HarborAgentContext) { a.NOutputTokens = -1 },
		"cache past bound":   func(a *HarborAgentContext) { a.NInputTokens, a.NCacheTokens = math.MaxInt64, maxTokens+1 },
		"cost not readable":  func(a *HarborAgentContext) { c := math.Inf(1); a.CostUSD = &c },
		"cost below nothing": func(a *HarborAgentContext) { c := -0.5; a.CostUSD = &c },
	} {
		h := harborOf(t, tr)
		mutate(h.AgentResult)
		if _, err := h.Trial(); err == nil {
			t.Errorf("Harbor %s: read", name)
		}
	}
	for name, mutate := range map[string]func(*InspectModelUsage){
		"negative input":   func(u *InspectModelUsage) { u.InputTokens = -1 },
		"input past bound": func(u *InspectModelUsage) { u.InputTokens = maxTokens + 1 },
		"total not a sum":  func(u *InspectModelUsage) { u.TotalTokens = 7 },
		"wrapping total":   func(u *InspectModelUsage) { u.InputTokens, u.TotalTokens = math.MaxInt64, math.MinInt64 },
	} {
		s := inspectOf(t, tr)
		mu := s.ModelUsage[tr.ModelServed]
		mutate(&mu)
		s.ModelUsage[tr.ModelServed] = mu
		if _, err := s.Trial(); err == nil {
			t.Errorf("Inspect %s: read", name)
		}
	}
	big := fullTrial()
	big.Usage.PromptTokens = maxTokens + 1
	if _, err := big.Harbor(); err == nil {
		t.Error("Harbor exported a count past its bound")
	}
	if _, err := big.Inspect(); err == nil {
		t.Error("Inspect exported a count past its bound")
	}
}

// What a target document names about its harness and models is untrusted
// text: a terminal escape in it refuses the document.
func TestImportedHarnessAndModelTextIsHeldToPlainText(t *testing.T) {
	for name, mutate := range map[string]func(*HarborTrialResult){
		"version": func(h *HarborTrialResult) { h.AgentInfo.Version = "]0;pwn[2J" },
		"name":    func(h *HarborTrialResult) { h.AgentInfo.Name = "open[2Jcode" },
		"served":  func(h *HarborTrialResult) { h.AgentInfo.ModelInfo.Name = "[31mRED" },
		"asked":   func(h *HarborTrialResult) { h.Config.Agent.ModelName = "vendor/​model" },
	} {
		h := harborOf(t, fullTrial())
		mutate(&h)
		if _, err := h.Trial(); err == nil {
			t.Errorf("Harbor %s: read", name)
		}
	}
	s := inspectOf(t, fullTrial())
	s.Output.Model = "[31mRED"
	s.ModelUsage = map[string]InspectModelUsage{s.Output.Model: s.ModelUsage[fullTrial().ModelServed]}
	if _, err := s.Trial(); err == nil {
		t.Error("an Inspect sample whose served model carries an escape was read")
	}
}

// An abcd record smuggled through a target is read as strictly as any other
// trust-boundary document: an unknown key in it is refused.
func TestTheAbcdRecordInsideATargetIsReadStrictly(t *testing.T) {
	s := inspectOf(t, fullTrial())
	s.Metadata[metadataKey] = json.RawMessage(strings.Replace(string(s.Metadata[metadataKey]), `{`, `{"extra":1,`, 1))
	if _, err := s.Trial(); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("Trial = %v; want an unknown-field refusal", err)
	}
	h := harborOf(t, fullTrial())
	h.AgentResult.Metadata[metadataKey] = json.RawMessage(`{"schema_version":1,"schema_version":2}`)
	if _, err := h.Trial(); err == nil {
		t.Fatal("a repeated key in the Harbor metadata was read")
	}
	names := []string{}
	for k := range s.Metadata {
		names = append(names, k)
	}
	sort.Strings(names)
	if strings.Join(names, ",") != metadataKey {
		t.Fatalf("metadata keys = %v; abcd writes one member, %q", names, metadataKey)
	}
}

func harborOf(t *testing.T, tr TrialRecord) HarborTrialResult {
	t.Helper()
	h, err := tr.Harbor()
	if err != nil {
		t.Fatalf("Harbor: %v", err)
	}
	return h
}

func inspectOf(t *testing.T, tr TrialRecord) InspectSample {
	t.Helper()
	s, err := tr.Inspect()
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	return s
}

// A trial that does not validate, or carries a value that does not encode, is
// refused by either mapping rather than exported half-formed.
func TestAnInvalidTrialIsNotExported(t *testing.T) {
	tr := fullTrial()
	tr.Settings = oracle.Settings{"temperature": json.RawMessage(`{not json`)}
	if _, err := tr.Harbor(); err == nil {
		t.Error("Harbor exported a setting that is not JSON")
	}
	if _, err := tr.Inspect(); err == nil {
		t.Error("Inspect exported a setting that is not JSON")
	}
	tr = fullTrial()
	tr.Key.Repeat = 0
	if _, err := tr.Harbor(); err == nil {
		t.Error("Harbor exported a trial with no repeat")
	}
	if _, err := tr.Inspect(); err == nil {
		t.Error("Inspect exported a trial with no repeat")
	}
}

// A sample id that is not the task's string is quoted back in the refusal,
// never printed raw: a carriage return, a right-to-left override and a C1
// control inside it reach the refusal escaped.
func TestASampleIdIsQuotedInItsRefusal(t *testing.T) {
	s := inspectOf(t, fullTrial())
	s.ID = json.RawMessage("{\r\"a\": \"\u202e\u009b2J\"}")
	if !json.Valid(s.ID) {
		t.Fatal("the probe id is not JSON")
	}
	_, err := s.Trial()
	if err == nil {
		t.Fatal("an object sample id was read")
	}
	for _, r := range []string{"\r", "\u202e", "\u009b"} {
		if strings.Contains(err.Error(), r) {
			t.Errorf("the refusal carries %q raw: %q", r, err)
		}
	}
}
