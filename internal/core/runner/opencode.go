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

import (
	"bytes"
	"context"
	"encoding/json"
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
	model  string
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
	res, err := o.launch.run(ctx, OpenCode, bin, o.args(req), openCodeSeal, req.Dir, req.timeout())
	transcript := res.transcript()
	if err != nil {
		// A harness refused at a usage limit may exit non-zero: the error
		// event it printed first is the reason, not the exit.
		if openCodeRateLimited(res.stdout) {
			return Answer{}, transcript, openCodeLimit()
		}
		return Answer{}, transcript, err
	}
	ans, err := parseOpenCode(res.stdout)
	if err != nil {
		return Answer{}, transcript, err
	}
	ans.Model = o.model
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
	// Error is an error event's error; its status code is the provider's
	// HTTP status when the error is an API error.
	Error *struct {
		Data struct {
			StatusCode int `json:"statusCode"`
		} `json:"data"`
	} `json:"error"`
}

// openCodeRateLimitStatus is the provider's status a rate limit is reported
// with: an error event whose error.data.statusCode is 429 (HTTP's Too Many
// Requests). The field is opencode's API error as this build assumes it; the
// live shape is owed to a person's check, as its other events' are.
const openCodeRateLimitStatus = 429

// openCodeLimit is the failure a rate-limit response is, in abcd's words.
func openCodeLimit() *Failure {
	return fail(OpenCode, ReasonRateLimited, "its provider refused the run at a rate limit (an error event with status %d)", openCodeRateLimitStatus)
}

// limited reports whether the event is a rate-limit response.
func (ev openCodeEvent) limited() bool {
	return ev.Type == "error" && ev.Error != nil && ev.Error.Data.StatusCode == openCodeRateLimitStatus
}

// openCodeRateLimited reports whether out carries a rate-limit response.
// Lines that are not events are passed over.
func openCodeRateLimited(out []byte) bool {
	for _, line := range bytes.Split(out, []byte("\n")) {
		var ev openCodeEvent
		if json.Unmarshal(bytes.TrimSpace(line), &ev) == nil && ev.limited() {
			return true
		}
	}
	return false
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
			if ev.limited() {
				return Answer{}, openCodeLimit()
			}
			return Answer{}, fail(OpenCode, ReasonRefused, "it reported an error event")
		case "text":
			ans.Text = ev.Part.Text
			seen = true
		case "step_finish":
			seen = true
		}
	}
	if !seen {
		return Answer{}, fail(OpenCode, ReasonUnparsable, "its output carries no text and no finished step")
	}
	return ans, nil
}
