package runner

// claude.go is the claude CLI runner (spc-2609221533057881 scope 2, criterion
// 6): print mode with the bare flag, so the target repository's hooks, plugins,
// CLAUDE.md auto-discovery and configured servers do not run untrusted
// (the intent's Decision 4); the stream-json event stream as the transcript;
// no session persisted by the harness, since abcd's store keeps the record;
// and the role's contract's tools granted without a prompt, everything else
// denied rather than asked (dontAsk), because no one is there to answer.
//
// Under --bare the harness reads its Anthropic credential from its own
// environment or its own settings only; abcd passes the environment through
// and never names a key.

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/oracle"
)

// ClaudeCLI is the claude CLI runner.
type ClaudeCLI struct {
	// model is the model route, <provider>/<model>, or "" for the harness's
	// own default.
	model  string
	launch launcher
}

// newClaude returns the claude CLI runner asking for model, a
// <provider>/<model> route the configuration admitted, or "" for the harness's
// default.
func newClaude(model string) *ClaudeCLI { return &ClaudeCLI{model: model, launch: defaultLauncher()} }

// Name is the route's name.
func (*ClaudeCLI) Name() string { return Claude }

// askedModel is the model the runner passes the harness: the route's model
// after its provider, "" for the harness's own default.
func (c *ClaudeCLI) askedModel() string {
	_, m, _ := strings.Cut(c.model, "/")
	return m
}

// args is the launch's argv after the binary.
func (c *ClaudeCLI) args(req Request) []string {
	a := []string{
		"--print", "--bare",
		"--output-format", "stream-json", "--verbose",
		"--no-session-persistence",
		"--permission-mode", "dontAsk",
	}
	if len(req.Tools) > 0 {
		a = append(a, "--allowedTools="+strings.Join(req.Tools, ","))
	}
	// The brief and the receipt sit in the run's own directory: a lane's,
	// outside the working tree the role runs in, or an interview's turns,
	// inside the repository's local tier. Either is added explicitly.
	dirs := []string{filepath.Dir(req.Brief)}
	if d := filepath.Dir(req.Receipt); d != dirs[0] {
		dirs = append(dirs, d)
	}
	for _, d := range dirs {
		a = append(a, "--add-dir="+d)
	}
	if m := c.askedModel(); m != "" {
		a = append(a, "--model="+m)
	}
	return append(a, "--", prompt(req))
}

// Run runs the role through the claude CLI.
func (c *ClaudeCLI) Run(ctx context.Context, req Request) (Answer, []byte, error) {
	if err := req.check(); err != nil {
		return Answer{}, nil, err
	}
	bin, err := c.launch.admit(Claude, "claude", req.Dir, req.Checkout)
	if err != nil {
		return Answer{}, nil, err
	}
	res, err := c.launch.run(ctx, Claude, bin, c.args(req), nil, req.Dir, req.timeout())
	transcript := res.transcript()
	if err != nil {
		// A harness that stopped on a rate limit exits non-zero: what its
		// events say is the reason, not the exit.
		if what := claudeRateLimit(res.stdout); what != "" {
			return Answer{}, transcript, rateLimited(what)
		}
		return Answer{}, transcript, err
	}
	ans, err := parseClaude(res.stdout)
	if err != nil {
		return ans, transcript, err
	}
	ans.ModelAsked, ans.PromptSHA256 = c.askedModel(), promptSHA256(req)
	ans.Unpinned = claudeUnpinned(ans)
	return ans, transcript, nil
}

// rateLimited is the claude runner's rate-limit failure, what naming the
// response in abcd's own words.
func rateLimited(what string) *Failure {
	return fail(Claude, ReasonRateLimited, "its event stream reports %s and the run did not finish", what)
}

// claudeRateLimit names the rate-limit response the event stream carries, ""
// when it carries none. The claude CLI (2.1) reports one in two shapes: a
// rate_limit_event whose rate_limit_info.status is rejected (allowed and
// allowed_warning are a run going on), and an assistant message whose error is
// rate_limit. Each line is read leniently and on its own, so a field this
// reader does not know never turns a limit into an unparsable run.
func claudeRateLimit(out []byte) string {
	for _, line := range bytes.Split(out, []byte("\n")) {
		var ev struct {
			Type  string          `json:"type"`
			Error json.RawMessage `json:"error"`
			Info  json.RawMessage `json:"rate_limit_info"`
		}
		if json.Unmarshal(bytes.TrimSpace(line), &ev) != nil {
			continue
		}
		switch ev.Type {
		case "rate_limit_event":
			var info struct {
				Status string `json:"status"`
			}
			if json.Unmarshal(ev.Info, &info) == nil && info.Status == "rejected" {
				return "a rejected rate_limit_event"
			}
		case "assistant":
			var e string
			if json.Unmarshal(ev.Error, &e) == nil && e == "rate_limit" {
				return "an assistant message whose error is rate_limit"
			}
		}
	}
	return ""
}

// claudeEvent is the part of a stream-json event the runner reads.
type claudeEvent struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype"`
	SessionID string `json:"session_id"`
	Model     string `json:"model"`
	IsError   bool   `json:"is_error"`
	Result    string `json:"result"`
	// Version is the init event's claude_code_version.
	Version string `json:"claude_code_version"`
}

// claudeMeter is the part of the result event that meters the run (the
// claude CLI 2.1): its usage, its cost, why it stopped and the usage per
// model. It is read leniently and apart from claudeEvent, so a metering field
// of an unexpected shape is a detail not reported, never an unparsable run.
type claudeMeter struct {
	StopReason   string   `json:"stop_reason"`
	TotalCostUSD *float64 `json:"total_cost_usd"`
	Usage        struct {
		InputTokens              *int `json:"input_tokens"`
		OutputTokens             *int `json:"output_tokens"`
		CacheReadInputTokens     *int `json:"cache_read_input_tokens"`
		CacheCreationInputTokens *int `json:"cache_creation_input_tokens"`
		OutputTokensDetails      struct {
			ThinkingTokens *int `json:"thinking_tokens"`
		} `json:"output_tokens_details"`
	} `json:"usage"`
	ModelUsage map[string]struct {
		Provider string `json:"provider"`
	} `json:"modelUsage"`
}

// claudeCounted reports which of the counts a result event carries readably:
// present and not negative, as the provider adapter reads a count. A count
// below zero is unreadable, listed as not reported.
func claudeCounted(m claudeMeter) counted {
	u := m.Usage
	ok := func(p *int) bool { return p != nil && *p >= 0 }
	return counted{prompt: ok(u.InputTokens), completion: ok(u.OutputTokens),
		reasoning: ok(u.OutputTokensDetails.ThinkingTokens),
		cacheRead: ok(u.CacheReadInputTokens), cacheWrite: ok(u.CacheCreationInputTokens)}
}

// meterClaude reads the result event's metering onto ans: the token counts
// and the cost the harness computed, the stop reason, the models its
// per-model usage names (the init event's model stays the canonical one, and
// stands in only when the init event named none and one model served), and
// the provider it names for the canonical model.
func meterClaude(ans *Answer, line []byte) {
	var m claudeMeter
	if json.Unmarshal(line, &m) != nil {
		return
	}
	ans.FinishReason = cleanHarness(m.StopReason)
	u := m.Usage
	c := claudeCounted(m)
	ans.counted = c
	if c.prompt || c.completion || c.reasoning || c.cacheRead || c.cacheWrite {
		n := func(p *int) int {
			if p == nil || *p < 0 {
				return 0
			}
			return *p
		}
		ans.Usage = &oracle.Usage{PromptTokens: n(u.InputTokens), CompletionTokens: n(u.OutputTokens),
			ReasoningTokens: n(u.OutputTokensDetails.ThinkingTokens),
			CacheReadTokens: n(u.CacheReadInputTokens), CacheWriteTokens: n(u.CacheCreationInputTokens)}
	}
	if m.TotalCostUSD != nil && oracle.ReadableCost(*m.TotalCostUSD) {
		ans.Cost = &oracle.Cost{Amount: *m.TotalCostUSD, Unit: oracle.CostUSD}
	}
	keys := make([]string, 0, len(m.ModelUsage))
	for k := range m.ModelUsage {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if ans.Model == "" && len(keys) == 1 && modelRe.MatchString(keys[0]) {
		ans.Model = keys[0]
	}
	provider := ""
	if mu, ok := m.ModelUsage[ans.Model]; ok {
		provider = mu.Provider
	} else if len(keys) == 1 {
		provider = m.ModelUsage[keys[0]].Provider
	}
	ans.ServedProvider = cleanHarness(provider)
	ans.ServedModels = nil
	for _, k := range keys {
		if len(ans.ServedModels) == maxServedModels {
			break
		}
		ans.ServedModels = append(ans.ServedModels, cleanHarness(k))
	}
}

// claudeUnpinned lists the fields a claude run could not pin. The cost is
// the harness's own computation, from its price table.
func claudeUnpinned(ans Answer) []oracle.Unpinned {
	var un oracle.Unpin
	askedUnpinned(&un, ans.Sides)
	servedUnpinned(&un, ans.Model, ans.Sides, ans.counted)
	return un.List()
}

// parseClaude reads the event stream: one JSON object per line, the init
// event's model and session, and the closing result event. A line that is not
// an event, or a stream with no result, is unparsable; a result that reports
// an error is a refusal.
func parseClaude(out []byte) (Answer, error) {
	var ans Answer
	var result *claudeEvent
	for _, line := range bytes.Split(out, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var ev claudeEvent
		if json.Unmarshal(line, &ev) != nil || ev.Type == "" {
			return Answer{}, fail(Claude, ReasonUnparsable, "a line of its output is not a stream-json event")
		}
		if ev.Type == "system" && ev.Subtype == "init" {
			if ev.Model != "" && !modelRe.MatchString(ev.Model) {
				// Refused rather than trimmed: an answer is never recorded
				// under a model the harness did not report.
				return Answer{}, fail(Claude, ReasonUnparsable, "its init event reports a model that is not a plain model id of at most 128 characters")
			}
			ans.Model = ev.Model
			ans.HarnessVersion = cleanHarness(ev.Version)
		}
		if ans.SessionID == "" && ev.SessionID != "" {
			ans.SessionID = ev.SessionID
		}
		if ev.Type == "result" {
			e := ev
			result = &e
			meterClaude(&ans, line)
		}
	}
	if result == nil || result.IsError || result.Subtype != "success" {
		// A run that did not finish on a rate limit is that limit, whatever
		// its result says.
		if what := claudeRateLimit(out); what != "" {
			return Answer{}, rateLimited(what)
		}
	}
	if result == nil {
		return Answer{}, fail(Claude, ReasonUnparsable, "its output carries no result event")
	}
	if result.IsError || result.Subtype != "success" {
		return Answer{}, fail(Claude, ReasonRefused, "its result reports %s", safeWord(result.Subtype))
	}
	ans.Text = result.Result
	return ans, nil
}

// safeWord renders a harness-supplied status word for a detail: a short token
// of plain characters, else "an error". Nothing longer from the harness
// reaches a detail.
func safeWord(s string) string {
	if len(s) == 0 || len(s) > 40 {
		return "an error"
	}
	for _, r := range s {
		if !(r == '_' || r == '-' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return "an error"
		}
	}
	return s
}
