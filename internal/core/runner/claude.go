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
	"strings"
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
	if _, m, ok := strings.Cut(c.model, "/"); ok {
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
		// A harness refused at a usage limit exits non-zero: the response it
		// printed first is the reason, not the exit.
		if claudeRateLimited(res.stdout) {
			return Answer{}, transcript, claudeLimit()
		}
		return Answer{}, transcript, err
	}
	ans, err := parseClaude(res.stdout)
	return ans, transcript, err
}

// claudeRateLimit is the event a rate limit is reported in, and the status
// that says the provider refused the run. The shape is the Agent SDK's
// rate-limit event as this build assumes it (a rate_limit_event whose
// rate_limit_info.status is "rejected"; "allowed" and "allowed_warning" let
// the run go on); the live shape is owed to a person's check, as the opencode
// events' are.
const (
	claudeRateLimitType     = "rate_limit_event"
	claudeRateLimitRejected = "rejected"
)

// claudeLimit is the failure a rate-limit response is. Its detail is abcd's
// own: nothing of the harness's text reaches it.
func claudeLimit() *Failure {
	return fail(Claude, ReasonRateLimited, "its provider refused the run at a usage limit (a rejected %s)", claudeRateLimitType)
}

// claudeRateLimited reports whether out carries a rejected rate-limit event
// and no successful result after it. Lines that are not events are passed
// over: a stream cut short by the exit is read for what it holds.
func claudeRateLimited(out []byte) bool {
	limited := false
	for _, line := range bytes.Split(out, []byte("\n")) {
		var ev claudeEvent
		if json.Unmarshal(bytes.TrimSpace(line), &ev) != nil {
			continue
		}
		switch {
		case ev.Type == claudeRateLimitType && ev.RateLimit != nil && ev.RateLimit.Status == claudeRateLimitRejected:
			limited = true
		case ev.Type == "result" && !ev.IsError && ev.Subtype == "success":
			limited = false
		}
	}
	return limited
}

// claudeEvent is the part of a stream-json event the runner reads.
type claudeEvent struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype"`
	SessionID string `json:"session_id"`
	Model     string `json:"model"`
	IsError   bool   `json:"is_error"`
	Result    string `json:"result"`
	// RateLimit is a rate_limit_event's report; nil on every other event.
	RateLimit *struct {
		Status string `json:"status"`
	} `json:"rate_limit_info"`
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
		}
		if ans.SessionID == "" && ev.SessionID != "" {
			ans.SessionID = ev.SessionID
		}
		if ev.Type == "result" {
			e := ev
			result = &e
		}
	}
	if (result == nil || result.IsError || result.Subtype != "success") && claudeRateLimited(out) {
		return Answer{}, claudeLimit()
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
