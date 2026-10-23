package matrix

// export.go maps a trial record field by field onto a Harbor trial result and
// an Inspect eval sample, and back (the SOTA note's path 2: a later move to
// either harness replaces only the scheduler). A field the target names lands
// in that field; every other field lands in the target's free-form metadata
// member under the one key "abcd", so every field lands, and the record reads
// back from either document unchanged.
//
// The target shapes are the upstream sources' at pinned commits, read on
// 2026-10-23:
//
//   - Inspect AI (EvalLog version 2), UKGovernmentBEIS/inspect_ai at
//     157980923b2993d2496a8d150c81f32465f15991:
//     https://raw.githubusercontent.com/UKGovernmentBEIS/inspect_ai/157980923b2993d2496a8d150c81f32465f15991/src/inspect_ai/log/_log.py
//     (EvalSample), src/inspect_ai/model/_model_output.py (ModelOutput,
//     ModelUsage, ModelFallback), src/inspect_ai/scorer/_metric.py (Score)
//     and src/inspect_ai/_util/error.py (EvalError) at the same commit. A
//     sample is one member of an .eval log (samples/<id>_epoch_<epoch>.json);
//     the run's own identity belongs in the log's EvalSpec, which this step
//     does not map.
//   - Harbor 0.24.0, harbor-framework/harbor at
//     d0fe0b24703f4e53806be5cb3b5e07d622aebbfa:
//     https://raw.githubusercontent.com/harbor-framework/harbor/d0fe0b24703f4e53806be5cb3b5e07d622aebbfa/src/harbor/models/trial/result.py
//     (TrialResult, written as <trial_dir>/result.json), models/trial/config.py,
//     models/agent/context.py (AgentContext), models/verifier/result.py and
//     models/task/id.py at the same commit.
//
// Open at this step: neither fixture has been run through the upstream
// validators (EvalLog.model_validate, TrialResult.model_validate).
//
// Four differences are handled here rather than papered over:
//
//   - Cached tokens. Harbor's n_input_tokens includes the cached prompt
//     tokens, with the cache reads counted again in n_cache_tokens; Inspect's
//     input_tokens, like abcd's prompt count, excludes them and counts cache
//     reads and writes apart. So Harbor's input is abcd's prompt plus both
//     cache counts, and the cache writes, which Harbor has no field for, ride
//     in the metadata so the prompt count can be recovered exactly. (That
//     n_input_tokens includes cache writes as well as reads is inferred.)
//   - The repeat. Harbor's TrialResult has no attempt index, so the repeat and
//     the cell ride in the trial name, <task>__<cell>__<repeat>, which abcd's
//     ids (no underscore) keep unambiguous. Inspect's epoch is the repeat.
//   - Asked and served. Harbor keeps the asked model in the trial's agent
//     config (config.agent.model_name) and the served one in agent_info, whose
//     provider is the model's vendor, the prefix of its id before the first
//     '/', not the host that served it; the serving provider rides in the
//     metadata. Inspect names a different model serving in place of the one
//     asked for as a ModelFallback. A served id that is the asked id, or the
//     asked id with a date stamp after '-', '@' or ':' (vendor/model served as
//     vendor/model-2026-10 or vendor/model@20261001), is a dated snapshot of
//     the same model and no fallback; any other served id is a different
//     model (sameModel).
//   - Errors. Inspect's error is the error that halted a sample, and Harbor
//     retries an exception type it does not list and drops its reward, so a
//     scored failure travels in its score alone: only a failed or excluded
//     copy with no score (its runner failed, refused or answered
//     unparsably) is exported as an error, its reason the message.

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// metadataKey is the one member abcd writes into a target's metadata.
const metadataKey = "abcd"

// hiddenTestsScore is the score's name in either target.
const hiddenTestsScore = "hidden_tests"

// checksumPrefix marks Harbor's task_checksum as a git object name: the frozen
// commit is the task's identity at the moment every copy started from.
const checksumPrefix = "git:"

// unknownModel keys Inspect's model usage when no served model was reported.
const unknownModel = "unknown"

// HarborTrialResult is Harbor's TrialResult, the members abcd writes.
type HarborTrialResult struct {
	TaskName       string                `json:"task_name"`
	TrialName      string                `json:"trial_name"`
	TrialURI       string                `json:"trial_uri"`
	TaskID         HarborTaskID          `json:"task_id"`
	TaskChecksum   string                `json:"task_checksum"`
	Config         HarborTrialConfig     `json:"config"`
	AgentInfo      HarborAgentInfo       `json:"agent_info"`
	AgentResult    *HarborAgentContext   `json:"agent_result,omitempty"`
	VerifierResult *HarborVerifierResult `json:"verifier_result,omitempty"`
	ExceptionInfo  *HarborExceptionInfo  `json:"exception_info,omitempty"`
	StartedAt      *time.Time            `json:"started_at,omitempty"`
	FinishedAt     *time.Time            `json:"finished_at,omitempty"`
}

// HarborTaskID is Harbor's LocalTaskId.
type HarborTaskID struct {
	Path string `json:"path"`
}

// HarborTrialConfig is Harbor's TrialConfig, the members abcd writes.
type HarborTrialConfig struct {
	Task  HarborTaskConfig   `json:"task"`
	Agent *HarborAgentConfig `json:"agent,omitempty"`
}

// HarborTaskConfig is a trial config's task.
type HarborTaskConfig struct {
	Path string `json:"path"`
}

// HarborAgentConfig is a trial config's agent: the model asked for.
type HarborAgentConfig struct {
	ModelName string `json:"model_name,omitempty"`
}

// HarborAgentInfo is Harbor's AgentInfo.
type HarborAgentInfo struct {
	Name      string           `json:"name"`
	Version   string           `json:"version"`
	ModelInfo *HarborModelInfo `json:"model_info,omitempty"`
}

// HarborModelInfo is an agent's model.
type HarborModelInfo struct {
	Name     string `json:"name"`
	Provider string `json:"provider,omitempty"`
}

// HarborAgentContext is Harbor's AgentContext: the token counts, the cost and
// the free-form metadata abcd's own fields ride in.
type HarborAgentContext struct {
	NInputTokens  int                        `json:"n_input_tokens"`
	NCacheTokens  int                        `json:"n_cache_tokens"`
	NOutputTokens int                        `json:"n_output_tokens"`
	CostUSD       *float64                   `json:"cost_usd,omitempty"`
	Metadata      map[string]json.RawMessage `json:"metadata,omitempty"`
}

// HarborVerifierResult is Harbor's VerifierResult.
type HarborVerifierResult struct {
	Rewards map[string]float64 `json:"rewards"`
}

// HarborExceptionInfo is Harbor's ExceptionInfo.
type HarborExceptionInfo struct {
	ExceptionType      string    `json:"exception_type"`
	ExceptionMessage   string    `json:"exception_message"`
	ExceptionTraceback string    `json:"exception_traceback"`
	OccurredAt         time.Time `json:"occurred_at"`
}

// harborAbcd is what Harbor's TrialResult has no member for.
type harborAbcd struct {
	SchemaVersion    int                      `json:"schema_version"`
	Run              string                   `json:"run"`
	Kind             TaskKind                 `json:"kind"`
	Intent           string                   `json:"intent,omitempty"`
	Source           Source                   `json:"source"`
	ServedProvider   string                   `json:"served_provider"`
	Settings         oracle.Settings          `json:"settings"`
	Status           TrialStatus              `json:"status"`
	Reason           string                   `json:"reason"`
	Score            *Score                   `json:"score"`
	Roles            []runner.RoleReceipt     `json:"roles"`
	Calls            []oracle.ReceiptRoute    `json:"calls"`
	Fallbacks        []runner.FallbackReceipt `json:"fallbacks"`
	ReasoningTokens  int                      `json:"reasoning_tokens"`
	CacheWriteTokens int                      `json:"cache_write_tokens"`
	// Cost is a cost in a unit other than usd, which cost_usd cannot hold.
	Cost        *oracle.Cost      `json:"cost,omitempty"`
	ModelTimeMS int64             `json:"model_time_ms"`
	GateWaitMS  int64             `json:"gate_wait_ms"`
	Unpinned    []oracle.Unpinned `json:"unpinned"`
}

// Harbor maps the trial onto a Harbor trial result. It refuses a trial that
// does not validate or a value that does not encode.
func (t TrialRecord) Harbor() (HarborTrialResult, error) {
	if err := t.validate(); err != nil {
		return HarborTrialResult{}, err
	}
	u := t.Usage
	ctx := &HarborAgentContext{
		NInputTokens:  u.PromptTokens + u.CacheReadTokens + u.CacheWriteTokens,
		NCacheTokens:  u.CacheReadTokens,
		NOutputTokens: u.CompletionTokens,
	}
	meta := harborAbcd{
		SchemaVersion: t.SchemaVersion, Run: t.Run, Kind: t.Kind, Intent: t.Intent, Source: t.Source,
		ServedProvider: t.ServedProvider, Settings: t.Settings, Status: t.Status, Reason: t.Reason, Score: t.Score,
		Roles: t.Roles, Calls: t.Calls, Fallbacks: t.Fallbacks,
		ReasoningTokens: u.ReasoningTokens, CacheWriteTokens: u.CacheWriteTokens,
		ModelTimeMS: t.ModelTimeMS, GateWaitMS: t.GateWaitMS, Unpinned: t.Unpinned,
	}
	if usd, ok := usdOf(t.Cost); ok {
		ctx.CostUSD = &usd
	} else {
		meta.Cost = t.Cost
	}
	raw, err := json.Marshal(meta)
	if err != nil {
		return HarborTrialResult{}, fmt.Errorf("cannot encode the trial %s: %w", t.Key, err)
	}
	ctx.Metadata = map[string]json.RawMessage{metadataKey: raw}
	h := HarborTrialResult{
		TaskName:     t.Key.Task,
		TrialName:    t.Key.Task + "__" + t.Key.Cell + "__" + strconv.Itoa(t.Key.Repeat),
		TrialURI:     trialURI(t.Run, t.Key),
		TaskID:       HarborTaskID{Path: t.Key.Task},
		TaskChecksum: checksumPrefix + t.FrozenCommit,
		Config:       HarborTrialConfig{Task: HarborTaskConfig{Path: t.Key.Task}},
		AgentInfo:    HarborAgentInfo{Name: t.Runner, Version: t.HarnessVersion},
		AgentResult:  ctx,
		StartedAt:    timePtr(t.StartedAt),
		FinishedAt:   timePtr(t.FinishedAt),
	}
	if t.ModelAsked != "" {
		h.Config.Agent = &HarborAgentConfig{ModelName: t.ModelAsked}
	}
	if t.ModelServed != "" {
		h.AgentInfo.ModelInfo = &HarborModelInfo{Name: t.ModelServed, Provider: vendor(t.ModelServed)}
	}
	if s := t.Score; s != nil && s.Total > 0 {
		h.VerifierResult = &HarborVerifierResult{Rewards: map[string]float64{hiddenTestsScore: float64(s.Passed) / float64(s.Total)}}
	}
	if kind := exception(t); kind != "" {
		h.ExceptionInfo = &HarborExceptionInfo{ExceptionType: kind, ExceptionMessage: t.Reason, OccurredAt: t.FinishedAt}
	}
	return h, nil
}

// Trial reads a trial record back from a Harbor trial result abcd wrote.
func (h HarborTrialResult) Trial() (TrialRecord, error) {
	if h.AgentResult == nil || h.AgentResult.Metadata[metadataKey] == nil {
		return TrialRecord{}, errors.New("the Harbor trial result carries no abcd record in agent_result.metadata; abcd did not write it")
	}
	var meta harborAbcd
	if err := jsonstrict.Decode(h.AgentResult.Metadata[metadataKey], &meta); err != nil {
		return TrialRecord{}, fmt.Errorf("the abcd record in the Harbor trial result does not parse: %w", err)
	}
	parts := strings.Split(h.TrialName, "__")
	if len(parts) != 3 || parts[0] != h.TaskName || h.TaskID.Path != h.TaskName || h.Config.Task.Path != h.TaskName {
		return TrialRecord{}, fmt.Errorf("the trial name %q is not <task>__<cell>__<repeat> of the task %q", clip(h.TrialName), clip(h.TaskName))
	}
	repeat, err := strconv.Atoi(parts[2])
	if err != nil || strconv.Itoa(repeat) != parts[2] {
		return TrialRecord{}, fmt.Errorf("the trial name %q carries no repeat", clip(h.TrialName))
	}
	key := TrialKey{Task: parts[0], Cell: parts[1], Repeat: repeat}
	commit, ok := strings.CutPrefix(h.TaskChecksum, checksumPrefix)
	if !ok || !gitutil.IsFullSHA(commit) {
		return TrialRecord{}, fmt.Errorf("the task checksum %q is not the frozen commit abcd writes there", clip(h.TaskChecksum))
	}
	if h.TrialURI != trialURI(meta.Run, key) {
		return TrialRecord{}, fmt.Errorf("the trial uri %q is not the trial's", clip(h.TrialURI))
	}
	a := h.AgentResult
	// Checked arithmetic: each count is bounded before any is subtracted, so
	// the prompt count cannot wrap into a plausible figure.
	in, reads, writes := int64(a.NInputTokens), int64(a.NCacheTokens), int64(meta.CacheWriteTokens)
	if in < 0 || in > 3*maxTokens || !countOK(reads) || !countOK(writes) || !countOK(int64(a.NOutputTokens)) {
		return TrialRecord{}, fmt.Errorf("the token counts are not counts: input %d, cache reads %d and writes %d, output %d", in, reads, writes, a.NOutputTokens)
	}
	prompt := in - reads - writes
	if !countOK(prompt) {
		return TrialRecord{}, fmt.Errorf("the token counts do not add up: input %d holds cache reads %d and writes %d", in, reads, writes)
	}
	t := TrialRecord{
		SchemaVersion: meta.SchemaVersion, Run: meta.Run, Key: key, Kind: meta.Kind, Intent: meta.Intent,
		FrozenCommit: commit, Source: meta.Source, Runner: h.AgentInfo.Name, HarnessVersion: h.AgentInfo.Version,
		ServedProvider: meta.ServedProvider, Settings: meta.Settings, Status: meta.Status, Reason: meta.Reason, Score: meta.Score,
		Roles: meta.Roles, Calls: meta.Calls, Fallbacks: meta.Fallbacks,
		Usage: oracle.Usage{PromptTokens: int(prompt), CompletionTokens: a.NOutputTokens, ReasoningTokens: meta.ReasoningTokens,
			CacheReadTokens: a.NCacheTokens, CacheWriteTokens: meta.CacheWriteTokens},
		Cost:        meta.Cost,
		StartedAt:   timeOf(h.StartedAt),
		FinishedAt:  timeOf(h.FinishedAt),
		ModelTimeMS: meta.ModelTimeMS, GateWaitMS: meta.GateWaitMS, Unpinned: meta.Unpinned,
	}
	if h.Config.Agent != nil {
		t.ModelAsked = h.Config.Agent.ModelName
	}
	if mi := h.AgentInfo.ModelInfo; mi != nil {
		if mi.Name == "" || mi.Provider != vendor(mi.Name) {
			return TrialRecord{}, fmt.Errorf("the model info %q of %q is not a served model and its vendor", clip(mi.Provider), clip(mi.Name))
		}
		t.ModelServed = mi.Name
	}
	if a.CostUSD != nil {
		if meta.Cost != nil {
			return TrialRecord{}, errors.New("the Harbor trial result carries its cost twice")
		}
		t.Cost = &oracle.Cost{Amount: *a.CostUSD, Unit: oracle.CostUSD}
	}
	if err := t.validate(); err != nil {
		return TrialRecord{}, fmt.Errorf("the Harbor trial result is not a trial record: %w", err)
	}
	kind, ei := exception(t), h.ExceptionInfo
	if (ei != nil) != (kind != "") || ei != nil && (ei.ExceptionType != kind || ei.ExceptionMessage != t.Reason) {
		return TrialRecord{}, errors.New("the Harbor trial result's exception_info is not the one abcd writes for its status and score")
	}
	return t, nil
}

// InspectSample is Inspect's EvalSample, the members abcd writes.
type InspectSample struct {
	// ID is the sample's id, a string or an integer upstream; abcd writes the
	// task's id.
	ID             json.RawMessage              `json:"id"`
	Epoch          int                          `json:"epoch"`
	Input          string                       `json:"input"`
	Target         string                       `json:"target"`
	Output         *InspectModelOutput          `json:"output,omitempty"`
	Scores         map[string]InspectScore      `json:"scores,omitempty"`
	Metadata       map[string]json.RawMessage   `json:"metadata,omitempty"`
	ModelUsage     map[string]InspectModelUsage `json:"model_usage,omitempty"`
	ModelFallbacks []InspectModelFallback       `json:"model_fallbacks,omitempty"`
	StartedAt      *time.Time                   `json:"started_at,omitempty"`
	CompletedAt    *time.Time                   `json:"completed_at,omitempty"`
	TotalTime      *float64                     `json:"total_time,omitempty"`
	WorkingTime    *float64                     `json:"working_time,omitempty"`
	Error          *InspectEvalError            `json:"error,omitempty"`
}

// InspectModelOutput is Inspect's ModelOutput, the members abcd writes: the
// model that generated, and no completion text (a trial records hashes, never
// a model's text).
type InspectModelOutput struct {
	Model      string `json:"model"`
	Completion string `json:"completion"`
}

// InspectScore is Inspect's Score; its value is any JSON value upstream.
type InspectScore struct {
	Value json.RawMessage `json:"value"`
}

// InspectModelUsage is Inspect's ModelUsage. InputTokens excludes the cached
// prompt tokens, which are counted apart.
type InspectModelUsage struct {
	InputTokens           int      `json:"input_tokens"`
	OutputTokens          int      `json:"output_tokens"`
	TotalTokens           int      `json:"total_tokens"`
	InputTokensCacheWrite int      `json:"input_tokens_cache_write"`
	InputTokensCacheRead  int      `json:"input_tokens_cache_read"`
	ReasoningTokens       int      `json:"reasoning_tokens"`
	TotalCost             *float64 `json:"total_cost,omitempty"`
}

// InspectModelFallback is Inspect's ModelFallback: the model requested and the
// one that served instead.
type InspectModelFallback struct {
	Model         string `json:"model"`
	FallbackModel string `json:"fallback_model"`
	Count         int    `json:"count"`
}

// InspectEvalError is Inspect's EvalError.
type InspectEvalError struct {
	Message       string `json:"message"`
	Traceback     string `json:"traceback"`
	TracebackANSI string `json:"traceback_ansi"`
}

// inspectAbcd is what Inspect's EvalSample has no member for. The frozen
// commit, the run and the cell are the enclosing log's (EvalSpec's revision,
// run_id and model) and ride here so one sample stands alone.
type inspectAbcd struct {
	SchemaVersion  int                      `json:"schema_version"`
	Run            string                   `json:"run"`
	Cell           string                   `json:"cell"`
	Kind           TaskKind                 `json:"kind"`
	Intent         string                   `json:"intent,omitempty"`
	FrozenCommit   string                   `json:"frozen_commit"`
	Source         Source                   `json:"source"`
	Runner         string                   `json:"runner"`
	HarnessVersion string                   `json:"harness_version"`
	ModelAsked     string                   `json:"model_asked"`
	ServedProvider string                   `json:"served_provider"`
	Settings       oracle.Settings          `json:"settings"`
	Status         TrialStatus              `json:"status"`
	Reason         string                   `json:"reason"`
	Score          *Score                   `json:"score"`
	Roles          []runner.RoleReceipt     `json:"roles"`
	Calls          []oracle.ReceiptRoute    `json:"calls"`
	Fallbacks      []runner.FallbackReceipt `json:"fallbacks"`
	// Cost is a cost in a unit other than usd, which total_cost cannot hold.
	Cost        *oracle.Cost      `json:"cost,omitempty"`
	ModelTimeMS int64             `json:"model_time_ms"`
	GateWaitMS  int64             `json:"gate_wait_ms"`
	Unpinned    []oracle.Unpinned `json:"unpinned"`
}

// Inspect maps the trial onto an Inspect eval sample. It refuses a trial that
// does not validate or a value that does not encode.
func (t TrialRecord) Inspect() (InspectSample, error) {
	if err := t.validate(); err != nil {
		return InspectSample{}, err
	}
	id, _ := json.Marshal(t.Key.Task)
	meta := inspectAbcd{
		SchemaVersion: t.SchemaVersion, Run: t.Run, Cell: t.Key.Cell, Kind: t.Kind, Intent: t.Intent,
		FrozenCommit: t.FrozenCommit, Source: t.Source, Runner: t.Runner, HarnessVersion: t.HarnessVersion,
		ModelAsked: t.ModelAsked, ServedProvider: t.ServedProvider, Settings: t.Settings,
		Status: t.Status, Reason: t.Reason, Score: t.Score, Roles: t.Roles, Calls: t.Calls, Fallbacks: t.Fallbacks,
		ModelTimeMS: t.ModelTimeMS, GateWaitMS: t.GateWaitMS, Unpinned: t.Unpinned,
	}
	s := InspectSample{
		ID: id, Epoch: t.Key.Repeat, Input: t.Key.Task, Target: "",
		StartedAt: timePtr(t.StartedAt), CompletedAt: timePtr(t.FinishedAt),
	}
	usd, isUSD := usdOf(t.Cost)
	if !isUSD {
		meta.Cost = t.Cost
	}
	if u := t.Usage; u != (oracle.Usage{}) || isUSD {
		mu := InspectModelUsage{
			InputTokens: u.PromptTokens, OutputTokens: u.CompletionTokens, ReasoningTokens: u.ReasoningTokens,
			InputTokensCacheRead: u.CacheReadTokens, InputTokensCacheWrite: u.CacheWriteTokens,
			// Inferred: the total counts the input, the output and both
			// cache counts once each; it is derived, and read back only to
			// check it.
			TotalTokens: totalTokens(u),
		}
		if isUSD {
			mu.TotalCost = &usd
		}
		key := t.ModelServed
		if key == "" {
			key = unknownModel
		}
		s.ModelUsage = map[string]InspectModelUsage{key: mu}
	}
	if t.ModelServed != "" {
		s.Output = &InspectModelOutput{Model: t.ModelServed}
	}
	s.ModelFallbacks = inspectFallbacks(t)
	if sc := t.Score; sc != nil && sc.Total > 0 {
		s.Scores = map[string]InspectScore{hiddenTestsScore: {Value: json.RawMessage(strconv.FormatFloat(float64(sc.Passed)/float64(sc.Total), 'g', -1, 64))}}
	}
	if !t.StartedAt.IsZero() && !t.FinishedAt.IsZero() {
		total := t.FinishedAt.Sub(t.StartedAt).Seconds()
		working := total - float64(t.GateWaitMS)/1000
		s.TotalTime, s.WorkingTime = &total, &working
	}
	if exception(t) != "" {
		s.Error = &InspectEvalError{Message: t.Reason}
	}
	raw, err := json.Marshal(meta)
	if err != nil {
		return InspectSample{}, fmt.Errorf("cannot encode the trial %s: %w", t.Key, err)
	}
	s.Metadata = map[string]json.RawMessage{metadataKey: raw}
	return s, nil
}

// Trial reads a trial record back from an Inspect eval sample abcd wrote.
func (s InspectSample) Trial() (TrialRecord, error) {
	if s.Metadata[metadataKey] == nil {
		return TrialRecord{}, errors.New("the Inspect sample carries no abcd record in its metadata; abcd did not write it")
	}
	var meta inspectAbcd
	if err := jsonstrict.Decode(s.Metadata[metadataKey], &meta); err != nil {
		return TrialRecord{}, fmt.Errorf("the abcd record in the Inspect sample does not parse: %w", err)
	}
	var task string
	if err := json.Unmarshal(s.ID, &task); err != nil {
		return TrialRecord{}, fmt.Errorf("the sample id %q is not the task id abcd writes", clip(string(s.ID)))
	}
	t := TrialRecord{
		SchemaVersion: meta.SchemaVersion, Run: meta.Run, Key: TrialKey{Task: task, Cell: meta.Cell, Repeat: s.Epoch},
		Kind: meta.Kind, Intent: meta.Intent, FrozenCommit: meta.FrozenCommit, Source: meta.Source,
		Runner: meta.Runner, HarnessVersion: meta.HarnessVersion, ModelAsked: meta.ModelAsked,
		ServedProvider: meta.ServedProvider, Settings: meta.Settings, Status: meta.Status, Reason: meta.Reason,
		Score: meta.Score, Roles: meta.Roles, Calls: meta.Calls, Fallbacks: meta.Fallbacks, Cost: meta.Cost,
		StartedAt: timeOf(s.StartedAt), FinishedAt: timeOf(s.CompletedAt),
		ModelTimeMS: meta.ModelTimeMS, GateWaitMS: meta.GateWaitMS, Unpinned: meta.Unpinned,
	}
	if s.Output != nil {
		t.ModelServed = s.Output.Model
	}
	switch len(s.ModelUsage) {
	case 0:
	case 1:
		for model, mu := range s.ModelUsage {
			if model != t.ModelServed && !(t.ModelServed == "" && model == unknownModel) {
				return TrialRecord{}, fmt.Errorf("the sample's usage is keyed by %q, not its served model %q", clip(model), clip(t.ModelServed))
			}
			t.Usage = oracle.Usage{PromptTokens: mu.InputTokens, CompletionTokens: mu.OutputTokens, ReasoningTokens: mu.ReasoningTokens,
				CacheReadTokens: mu.InputTokensCacheRead, CacheWriteTokens: mu.InputTokensCacheWrite}
			// Each count is bounded before the total is checked against
			// their sum, so the sum cannot wrap.
			if err := checkUsage(t.Usage); err != nil {
				return TrialRecord{}, fmt.Errorf("the Inspect sample's usage: %w", err)
			}
			if mu.TotalTokens != totalTokens(t.Usage) {
				return TrialRecord{}, fmt.Errorf("the sample's total_tokens %d is not its counts' sum %d", mu.TotalTokens, totalTokens(t.Usage))
			}
			if mu.TotalCost != nil {
				if meta.Cost != nil {
					return TrialRecord{}, errors.New("the Inspect sample carries its cost twice")
				}
				t.Cost = &oracle.Cost{Amount: *mu.TotalCost, Unit: oracle.CostUSD}
			}
		}
	default:
		return TrialRecord{}, fmt.Errorf("the sample's usage names %d models; one copy's usage is its served model's", len(s.ModelUsage))
	}
	if err := t.validate(); err != nil {
		return TrialRecord{}, fmt.Errorf("the Inspect sample is not a trial record: %w", err)
	}
	if want := inspectFallbacks(t); len(want) != len(s.ModelFallbacks) || len(want) > 0 && want[0] != s.ModelFallbacks[0] {
		return TrialRecord{}, errors.New("the Inspect sample's model_fallbacks are not the ones abcd writes for its asked and served models")
	}
	if (s.Error != nil) != (exception(t) != "") || s.Error != nil && (s.Error.Message != t.Reason || s.Error.Traceback != "" || s.Error.TracebackANSI != "") {
		return TrialRecord{}, errors.New("the Inspect sample's error is not the one abcd writes for its status and score")
	}
	return t, nil
}

// exception is the error type a target names for a failed or excluded copy
// that has no score (its runner failed, refused or answered unparsably), ""
// for any other copy: a scored failure is a result, not an error. Such a copy
// always names its reason and finish time (validate), so it never exports as
// a copy that produced nothing.
func exception(t TrialRecord) string {
	if (t.Status == StatusFailed || t.Status == StatusExcluded) && t.Score == nil {
		return "abcd." + string(t.Status)
	}
	return ""
}

// inspectFallbacks is the ModelFallback a trial's asked and served models
// make: one when a different model served (sameModel), none otherwise.
func inspectFallbacks(t TrialRecord) []InspectModelFallback {
	if t.ModelAsked == "" || t.ModelServed == "" || sameModel(t.ModelAsked, t.ModelServed) {
		return nil
	}
	return []InspectModelFallback{{Model: t.ModelAsked, FallbackModel: t.ModelServed, Count: 1}}
}

// snapshotRe is a date stamp a served id may add to the asked one: the asked
// model's dated snapshot (-2026-10, -20261001, @20261001, :2026-10-01).
var snapshotRe = regexp.MustCompile(`^[-@:]([0-9]{8}|[0-9]{4}-[0-9]{2}(-[0-9]{2})?)$`)

// sameModel reports whether served is the asked model: the same id, or the
// asked id with a date stamp after it. A four-digit stamp (-0613) is not
// read as one: it cannot be told from a size or a version without a vendor's
// list, and a different model read as the same would hide a substitution, so
// it exports as a fallback, which a reader can see is a snapshot.
func sameModel(asked, served string) bool {
	rest, ok := strings.CutPrefix(served, asked)
	return ok && (rest == "" || snapshotRe.MatchString(rest))
}

// vendor is a model id's vendor, the prefix before its first '/', "" for an id
// with none.
func vendor(model string) string {
	v, _, ok := strings.Cut(model, "/")
	if !ok {
		return ""
	}
	return v
}

// totalTokens is the total Inspect derives from the counts (inferred: input,
// output and both cache counts once each). Every count is bounded by
// maxTokens before it is summed, so the sum cannot overflow.
func totalTokens(u oracle.Usage) int {
	return u.PromptTokens + u.CompletionTokens + u.CacheReadTokens + u.CacheWriteTokens
}

// trialURI names a trial without naming a path on any machine.
func trialURI(run string, k TrialKey) string { return "abcd://matrix/" + run + "/" + k.String() }

// usdOf is a cost's amount when it is in usd.
func usdOf(c *oracle.Cost) (float64, bool) {
	if c == nil || c.Unit != oracle.CostUSD {
		return 0, false
	}
	return c.Amount, true
}

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func timeOf(p *time.Time) time.Time {
	if p == nil {
		return time.Time{}
	}
	return *p
}
