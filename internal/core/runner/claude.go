package runner

// claude.go is the claude CLI runner (spc-2609221533057881 scope 2, criterion
// 6): print mode with the bare flag, so the target repository's hooks, plugins,
// CLAUDE.md auto-discovery and configured servers do not run untrusted
// (the intent's Decision 4); the stream-json event stream as the transcript;
// no session persisted by the harness, since abcd's store keeps the record;
// and the role's contract's tools granted without a prompt, everything else
// denied rather than asked (dontAsk), because no one is there to answer.
//
// A role whose contract grants the whole shell is the exception
// (iss-2610161612203555, dcn-2610161624025493): nobody is there to answer the
// host's prompts, so abcd's guard answers them, for that launch only and on
// the command line. The launch names the harness's permission prompt tool,
// served by `abcd guard approve` as the one MCP server it starts
// (--mcp-config inline, --strict-mcp-config), under the default permission
// mode, where a prompt reaches that tool rather than being refused. The shell
// is then no longer granted without a prompt, so every shell command the
// harness would ask about is put to the guard, which approves a command it
// reads whole and clears and refuses everything else with its reason. Bare
// mode skips every hook, those a --settings file declares included, so a
// hook could not carry this; an MCP server --mcp-config names is the one
// thing bare mode starts. Nothing is written to the person's host settings,
// and this adds no sandbox.
//
// Under --bare the harness reads its Anthropic credential from its own
// environment or its own settings only; abcd passes the environment through
// and never names a key.

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// ClaudeCLI is the claude CLI runner.
type ClaudeCLI struct {
	// model is the model route, <provider>/<model>, or "" for the harness's
	// own default.
	model  string
	launch launcher
	// approver is the absolute path of the abcd binary a shell launch starts
	// as its permission prompt answerer.
	approver string
}

// ApproverTool is the permission prompt tool a shell launch names.
const ApproverTool = "mcp__abcd__approve"

// newClaude returns the claude CLI runner asking for model, a
// <provider>/<model> route the configuration admitted, or "" for the harness's
// default.
func newClaude(model string) *ClaudeCLI {
	return &ClaudeCLI{model: model, launch: defaultLauncher(), approver: selfBinary()}
}

// selfBinary is the absolute path of the running abcd, "" when the system
// cannot name it. The approver is the binary already running, so starting it
// trusts nothing new.
func selfBinary() string {
	p, err := os.Executable()
	if err != nil || !filepath.IsAbs(p) {
		return ""
	}
	return filepath.Clean(p)
}

// grantsShell reports whether a contract grants the whole shell, the one
// grant the guard answers prompts for. A scoped grant (Bash(git log:*)) is
// the harness's own rule and stays one.
func grantsShell(tools []string) bool { return slices.Contains(tools, "Bash") }

// approverConfig is the --mcp-config value that starts the approver: one
// stdio server, abcd's own binary, judging commands against the registry of
// the working tree the role runs in, as it stood at base, the commit the run
// started on, with runDirs, the run's own directories, named so a removal
// may reach them.
func (c *ClaudeCLI) approverConfig(dir, base string, runDirs []string) string {
	type server struct {
		Type    string   `json:"type"`
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	args := []string{"guard", "approve", "--dir=" + dir, "--base=" + base}
	for _, d := range runDirs {
		args = append(args, "--run-dir="+d)
	}
	raw, _ := json.Marshal(map[string]map[string]server{"mcpServers": {"abcd": {
		Type: "stdio", Command: c.approver, Args: args,
	}}})
	return string(raw)
}

// Name is the route's name.
func (*ClaudeCLI) Name() string { return Claude }

// args is the launch's argv after the binary.
func (c *ClaudeCLI) args(req Request) []string {
	shell := grantsShell(req.Tools)
	mode := "dontAsk"
	if shell {
		mode = "default"
	}
	a := []string{
		"--print", "--bare",
		"--output-format", "stream-json", "--verbose",
		"--no-session-persistence",
		"--permission-mode", mode,
	}
	// The brief and the receipt sit in the run's own directory: a lane's,
	// outside the working tree the role runs in, or an interview's turns,
	// inside the repository's local tier. Either is added explicitly.
	dirs := []string{filepath.Dir(req.Brief)}
	if d := filepath.Dir(req.Receipt); d != dirs[0] {
		dirs = append(dirs, d)
	}
	tools := req.Tools
	if shell {
		// The guard answers for the shell, so the harness asks it rather
		// than running every command unasked.
		tools = slices.DeleteFunc(slices.Clone(tools), func(t string) bool { return t == "Bash" })
		a = append(a, "--mcp-config="+c.approverConfig(req.Dir, req.Base, dirs), "--strict-mcp-config",
			"--permission-prompt-tool="+ApproverTool)
	}
	if len(tools) > 0 {
		a = append(a, "--allowedTools="+strings.Join(tools, ","))
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
	if grantsShell(req.Tools) && c.approver == "" {
		// Started without the approver, the shell would either run unchecked
		// or have every command refused; neither is the launch asked for.
		return Answer{}, nil, fail(Claude, ReasonFailed,
			"abcd cannot name its own binary to answer the run's permission prompts, so the shell launch was not started")
	}
	if grantsShell(req.Tools) && req.Base == "" {
		// The approver judges against the registry at the run's base; with
		// none named it would have nothing to judge against.
		return Answer{}, nil, fail(Claude, ReasonFailed,
			"the run names no base commit for its permission prompts to be judged against, so the shell launch was not started")
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
	return ans, transcript, err
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
