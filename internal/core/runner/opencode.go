package runner

// opencode.go is the opencode runner (spc-2609221533057881 scope 2): run mode
// with raw JSON events, the repository as its directory and the brief
// attached, sealed against the repository it runs in (ruling OC1 of
// 2026-10-02), the nearest the harness offers to the claude CLI's bare mode:
//
//   - --pure: no external plugins, so a plugin the repository configures does
//     not run;
//   - OPENCODE_DISABLE_PROJECT_CONFIG=1: no project configuration, so the
//     repository's opencode.json and .opencode/ (settings, agents, commands,
//     skills) and its instruction files (AGENTS.md) are not read;
//   - OPENCODE_DISABLE_CLAUDE_CODE=1: no CLAUDE.md and no .claude/ skills;
//   - OPENCODE_DISABLE_EXTERNAL_SKILLS=1: no skills from the other agents'
//     directories (.agents/).
//
// OPENCODE_DISABLE_PROJECT_CONFIG is read from opencode's source, not its
// documentation, which does not list it (github.com/anomalyco/opencode,
// flag.ts, config.ts and instruction.ts, read 2026-10-02); the other two are
// in both. The owner's global configuration still applies: it names the model
// and the server, and is the person's own. Permissions are left at that
// configuration too: abcd does not pass the flag that approves every request
// unasked.
//
// The runner reaches opencode through run mode only; its server's session
// endpoints, where a server is already up, are a later adapter.
//
// Run mode's JSON events name no model (opencode 1.18.31: tool_use,
// step_start, step_finish, text, reasoning and error), so a run's served model
// is not reported and is recorded so, never as the model the runner asked for
// (itd-166 decision 6, iss-2610161709506207). Its step-finish events carry
// each step's tokens and the cost opencode computed, which a run sums. Run
// mode emits the run's own session's events only, so a sub-agent's steps are
// not among them and the sums are a floor.

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"os"

	"github.com/intentdriven/abcd/internal/core/oracle"
)

// openCodeSeal is the environment every opencode launch is sealed with, set
// over whatever the parent's environment holds for the same names.
var openCodeSeal = []string{
	"OPENCODE_DISABLE_PROJECT_CONFIG=1",
	"OPENCODE_DISABLE_CLAUDE_CODE=1",
	"OPENCODE_DISABLE_EXTERNAL_SKILLS=1",
}

// OpenCodeCLI is the opencode runner.
type OpenCodeCLI struct {
	// model is the model route, <provider>/<model> as opencode spells it too,
	// or "" for the harness's own default.
	model string
	// pin is the machine's pin for the model, nil when it sets none: a
	// pinned run reads it from a sealed configuration of its own (pin.go).
	pin    *oracle.Pin
	launch launcher
}

// newOpenCode returns the opencode runner asking for model, a
// <provider>/<model> route the configuration admitted, or "".
func newOpenCode(model string) *OpenCodeCLI {
	return &OpenCodeCLI{model: model, launch: defaultLauncher()}
}

// Name is the route's name.
func (*OpenCodeCLI) Name() string { return OpenCode }

func (o *OpenCodeCLI) args(req Request) []string {
	a := []string{"run", "--format=json", "--pure", "--dir=" + req.Dir, "--file=" + req.Brief}
	if o.model != "" {
		a = append(a, "--model="+o.model)
	}
	return append(a, "--", prompt(req))
}

// Run runs the role through opencode.
func (o *OpenCodeCLI) Run(ctx context.Context, req Request) (Answer, []byte, error) {
	if err := req.check(); err != nil {
		return Answer{}, nil, err
	}
	bin, err := o.launch.admit(OpenCode, "opencode", req.Dir, req.Checkout)
	if err != nil {
		return Answer{}, nil, err
	}
	seal, version := openCodeSeal, ""
	var sealed sealedPin
	if o.pin != nil {
		// A pinned run reads the pin from a configuration made for it alone
		// (pin.go), and its opencode's version is recorded beside it; one
		// whose parent names a configuration of the person's own is refused
		// first, since the sealed file would set it aside.
		if err := personsOpenCodeConfig(); err != nil {
			return Answer{}, nil, err
		}
		if sealed, err = o.sealPin(req); err != nil {
			return Answer{}, nil, err
		}
		defer os.RemoveAll(sealed.dir)
		seal = pinnedSeal(sealed)
		version = o.version(ctx, bin, seal, req.Dir)
	}
	res, err := o.launch.run(ctx, OpenCode, bin, o.args(req), seal, req.Dir, req.timeout())
	transcript := res.transcript()
	if err != nil {
		return Answer{}, transcript, err
	}
	ans, err := parseOpenCode(res.stdout)
	if err != nil {
		return Answer{}, transcript, err
	}
	ans.ModelAsked, ans.PromptSHA256 = o.model, promptSHA256(req)
	if o.pin != nil {
		pin := *o.pin
		ans.ProviderPin, ans.PinConfigSHA256, ans.HarnessVersion = &pin, sealed.sha256, version
	}
	var un oracle.Unpin
	askedUnpinned(&un, ans.Sides)
	servedUnpinned(&un, ans.Model, ans.Sides, ans.counted)
	ans.Unpinned = un.List()
	return ans, transcript, nil
}

// openCodeEvent is the part of a run-mode JSON event the runner reads.
type openCodeEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionID"`
	Part      struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"part"`
}

// openCodeStep is the metering of a step_finish event's part: why the step
// stopped, the cost opencode computed and the step's tokens. It is read
// leniently and apart from openCodeEvent, so a field of an unexpected shape
// is a detail not reported, never an unparsable run.
type openCodeStep struct {
	Part struct {
		Reason string   `json:"reason"`
		Cost   *float64 `json:"cost"`
		Tokens *struct {
			Input     *int `json:"input"`
			Output    *int `json:"output"`
			Reasoning *int `json:"reasoning"`
			Cache     struct {
				Read  *int `json:"read"`
				Write *int `json:"write"`
			} `json:"cache"`
		} `json:"tokens"`
	} `json:"part"`
}

// meterOpenCode adds one step_finish event's metering to ans: its tokens and
// cost summed onto the run's, its reason the run's finish reason until a
// later step's replaces it.
func meterOpenCode(ans *Answer, line []byte) {
	var st openCodeStep
	if json.Unmarshal(line, &st) != nil {
		return
	}
	p := st.Part
	if p.Reason != "" {
		ans.FinishReason = cleanHarness(p.Reason)
	}
	// A step's cost is summed when it is readable (oracle.ReadableCost); a
	// sum that stops being finite is dropped, not reported for the rest of
	// the run, since +Inf is no cost and no record can write it.
	if p.Cost != nil && !ans.costLost && oracle.ReadableCost(*p.Cost) {
		sum := *p.Cost
		if ans.Cost != nil {
			sum += ans.Cost.Amount
		}
		if oracle.ReadableCost(sum) {
			ans.Cost = &oracle.Cost{Amount: sum, Unit: oracle.CostUSD}
		} else {
			ans.Cost, ans.costLost = nil, true
		}
	}
	t := p.Tokens
	if t == nil {
		return
	}
	if ans.Usage == nil {
		ans.Usage = &oracle.Usage{}
	}
	// A count is summed when present and not negative; a sum that would
	// overflow is unreadable, zeroed and not reported for the rest of the run.
	add := func(sum *int, n *int, seen, lost *bool) {
		switch {
		case *lost || n == nil || *n < 0:
		case *sum > math.MaxInt-*n:
			*sum, *seen, *lost = 0, false, true
		default:
			*sum += *n
			*seen = true
		}
	}
	c, l := &ans.counted, &ans.lost
	add(&ans.Usage.PromptTokens, t.Input, &c.prompt, &l.prompt)
	add(&ans.Usage.CompletionTokens, t.Output, &c.completion, &l.completion)
	add(&ans.Usage.ReasoningTokens, t.Reasoning, &c.reasoning, &l.reasoning)
	add(&ans.Usage.CacheReadTokens, t.Cache.Read, &c.cacheRead, &l.cacheRead)
	add(&ans.Usage.CacheWriteTokens, t.Cache.Write, &c.cacheWrite, &l.cacheWrite)
}

// parseOpenCode reads the event stream: one JSON object per line. An error
// event is a refusal; a line that is not an event, or a stream with no text
// and no finished step, is unparsable. The last text part is the final
// message.
func parseOpenCode(out []byte) (Answer, error) {
	var ans Answer
	seen := false
	for _, line := range bytes.Split(out, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var ev openCodeEvent
		if json.Unmarshal(line, &ev) != nil || ev.Type == "" {
			return Answer{}, fail(OpenCode, ReasonUnparsable, "a line of its output is not a JSON event")
		}
		if ans.SessionID == "" && ev.SessionID != "" {
			ans.SessionID = ev.SessionID
		}
		switch ev.Type {
		case "error":
			return Answer{}, fail(OpenCode, ReasonRefused, "it reported an error event")
		case "text":
			ans.Text = ev.Part.Text
			seen = true
		case "step_finish":
			seen = true
			meterOpenCode(&ans, line)
		}
	}
	if !seen {
		return Answer{}, fail(OpenCode, ReasonUnparsable, "its output carries no text and no finished step")
	}
	return ans, nil
}
