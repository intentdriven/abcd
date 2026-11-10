package runner

// claude.go is the claude CLI runner (spc-2609221533057881 scope 2, criterion
// 6): print mode with the bare flag, so the target repository's hooks, plugins,
// CLAUDE.md auto-discovery and configured servers do not run untrusted
// (the intent's Decision 4); the stream-json event stream as the transcript;
// no session persisted by the harness, since abcd's store keeps the record;
// and the role's contract's tools granted without a prompt, the tools it is
// refused outright in the deny list, and everything else denied rather than
// asked (dontAsk), because no one is there to answer. A grant or a refusal may
// name a path in the harness's permission-rule form (Read(//abs/dir/**)).
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
// reads whole and clears, handing it back to run from the working tree under
// /bin/bash with its profile, rc and environment start-up files ignored (-p),
// and refuses everything else with its reason. Bare
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
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
)

// ClaudeCLI is the claude CLI runner.
type ClaudeCLI struct {
	// model is the model route, <provider>/<model>, or "" for the harness's
	// own default.
	model string
	// local says the route's provider is a block on this machine, whose
	// local fields the record lists (sides.go); claude is handed no local
	// pin, so they stay not configured.
	local bool
	// pinned says the machine pins the route's model (a provider pin or a
	// local pin). claude can hold neither, so a pinned route is refused
	// before launch rather than run unheld.
	pinned bool
	launch launcher
	// approver is the absolute path of the abcd binary a shell launch starts
	// as its permission prompt answerer.
	approver string
	// profile is the walled login a drive's launch runs on (profile.go), nil
	// for the bare launch.
	profile *Profile
}

// claudeVendor is the claude harness's own vendor, as a <provider>/<model>
// route names it (the bundled class map's provider): on a drive's walled
// launch the harness takes this vendor's models by name with no provider block
// (Config.admitModelFor), and no other vendor's.
const claudeVendor = "anthropic"

// approverServer is the name of the one server a shell launch starts, and
// ApproverTool the permission prompt tool it names on that server.
const (
	approverServer = "abcd"
	ApproverTool   = "mcp__" + approverServer + "__approve"
)

// newClaude returns the claude CLI runner asking for model, a
// <provider>/<model> route the configuration admitted, or "" for the harness's
// default.
func newClaude(model string) *ClaudeCLI {
	return &ClaudeCLI{model: model, launch: defaultLauncher(), approver: selfBinary()}
}

// selfBinary is the absolute path of the running abcd, "" when the system
// cannot name it. The approver is the binary already running, so starting it
// trusts nothing new.
var selfBinary = selfNamer(os.Executable)

// grantsShell reports whether a contract grants the whole shell, the one
// grant the guard answers prompts for. A scoped grant (Bash(git log:*)) is
// the harness's own rule and stays one.
func grantsShell(tools []string) bool { return slices.Contains(tools, "Bash") }

// approverConfig is the --mcp-config value that starts the approver: one
// stdio server, abcd's own binary, judging commands against the registry of
// the working tree the role runs in, as it stood at base, the commit the run
// started on, with runDirs, the run's own directories, named so a removal
// may reach them.
func (c *ClaudeCLI) approverConfig(dir, base string, runDirs, declared []string) string {
	type server struct {
		Type    string   `json:"type"`
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	args := []string{"guard", "approve", "--dir=" + dir, "--base=" + base}
	for _, d := range runDirs {
		args = append(args, "--run-dir="+d)
	}
	if c.profile != nil {
		args = append(args, "--stage-home="+c.profile.ShellHome, "--protect="+c.profile.Dir)
		if harness := provedHarness(*c.profile); harness != "" {
			args = append(args, "--harness="+harness)
		}
		for _, d := range declared {
			args = append(args, "--declared="+d)
		}
	}
	raw, _ := json.Marshal(map[string]map[string]server{"mcpServers": {approverServer: {
		Type: "stdio", Command: c.approver, Args: args,
	}}})
	return string(raw)
}

// Name is the route's name.
func (*ClaudeCLI) Name() string { return Claude }

// askedModel is the model the runner passes the harness for req: the
// request's model route, else the runner's, after its provider; "" for the
// harness's own default.
func (c *ClaudeCLI) askedModel(req Request) string {
	_, m, _ := strings.Cut(req.modelOr(c.model), "/")
	return m
}

// args is the launch's argv after the binary.
func (c *ClaudeCLI) args(req Request) []string {
	if c.profile != nil {
		return append(c.walledFlags(req), "--", prompt(req))
	}
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
	// A refusal is the harness's deny rule, which it judges before any grant.
	if len(req.Denied) > 0 {
		a = append(a, "--disallowedTools="+strings.Join(req.Denied, ","))
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
		a = append(a, "--mcp-config="+c.approverConfig(req.Dir, req.Base, dirs, req.Declared), "--strict-mcp-config",
			"--permission-prompt-tool="+ApproverTool)
	}
	if len(tools) > 0 {
		a = append(a, "--allowedTools="+strings.Join(tools, ","))
	}
	for _, d := range dirs {
		a = append(a, "--add-dir="+d)
	}
	if m := c.askedModel(req); m != "" {
		a = append(a, "--model="+m)
	}
	return append(effortFlag(a, req), "--", prompt(req))
}

// effortFlag is a with the harness's effort flag for a request at a class
// (itd-2611101419006280); a request with no class carries no effort.
func effortFlag(a []string, req Request) []string {
	if req.Effort == "" {
		return a
	}
	return append(a, "--effort="+req.Effort)
}

// effortNamed adds to a failed launch at a class the effort and the class it
// was launched at, so a harness that refused the effort is named with it.
func effortNamed(fl *Failure, req Request) {
	if req.Effort != "" {
		fl.Detail += fmt.Sprintf("; it was launched at effort %s, the %s class's", req.Effort, req.Class)
	}
}

// walledDirRule is the harness's rule for everything below dir, in the rule's own form for
// an absolute path.
func walledDirRule(dir string) string { return "Read(/" + dir + "/**)" }

// walledFlags is the walled launch's flags (spc-2610230813348813): hooks on,
// so no --bare; the tools the stage holds and no others; the shell answered by
// the approver, the only server the launch may start; and the profile's own
// arguments.
func (c *ClaudeCLI) walledFlags(req Request) []string {
	shell := grantsShell(req.Tools)
	mode := "dontAsk"
	if shell {
		mode = "default"
	}
	a := []string{"--print", "--output-format", "stream-json", "--verbose", "--permission-mode", mode}
	a = append(a, c.profile.Args...)
	// The stage's file tools answer to no approver, so the walled directory is a deny rule.
	a = append(a, "--disallowedTools="+strings.Join(append(slices.Clone(req.Denied), walledDirRule(c.profile.Dir)), ","))
	dirs := []string{filepath.Dir(req.Brief)}
	if d := filepath.Dir(req.Receipt); d != dirs[0] {
		dirs = append(dirs, d)
	}
	a = append(a, "--tools="+strings.Join(heldTools(req.Tools), ","))
	tools := req.Tools
	if shell {
		tools = slices.DeleteFunc(slices.Clone(tools), func(t string) bool { return t == "Bash" })
		a = append(a, "--mcp-config="+c.approverConfig(req.Dir, req.Base, dirs, req.Declared), "--permission-prompt-tool="+ApproverTool)
	} else {
		a = append(a, `--mcp-config={"mcpServers":{}}`)
	}
	if len(tools) > 0 {
		a = append(a, "--allowedTools="+strings.Join(tools, ","))
	}
	for _, d := range dirs {
		a = append(a, "--add-dir="+d)
	}
	if m := c.askedModel(req); m != "" {
		a = append(a, "--model="+m)
	}
	return effortFlag(a, req)
}

// stageWriteDirs are where a walled launch's file tools write: its working tree
// and the directories of its brief and receipt.
func stageWriteDirs(req Request) []string {
	dirs := []string{req.Dir}
	for _, d := range []string{filepath.Dir(req.Brief), filepath.Dir(req.Receipt)} {
		if !slices.Contains(dirs, d) {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

// heldTools are the tools a launch holds for the grants it names: each grant's
// tool, its path left to the allow-list. The harness judges a new file's write
// against the Edit rules, so an edit grant holds Write as well.
func heldTools(grants []string) []string {
	var held []string
	add := func(t string) {
		if !slices.Contains(held, t) {
			held = append(held, t)
		}
	}
	for _, g := range grants {
		name, _, _ := strings.Cut(g, "(")
		add(name)
		if name == "Edit" {
			add("Write")
		}
	}
	return held
}

// Run runs the role through the claude CLI. A launch reaches the ledger through
// the dispatcher that called it (RecordLaunch).
func (c *ClaudeCLI) Run(ctx context.Context, req Request) (Answer, []byte, error) {
	if err := req.check(); err != nil {
		return Answer{}, nil, err
	}
	if c.profile != nil {
		// A stage starts only on the settings abcd wrote.
		if err := CheckWallSettings(*c.profile, c.approver); err != nil {
			return Answer{}, nil, fail(Claude, ReasonRefused, "%s", err.Error())
		}
	}
	if c.pinned {
		return Answer{}, nil, fail(Claude, ReasonFailed,
			"the machine pins %s, and the claude runner can hold neither a provider pin nor a local pin, so the run is "+
				"not started: route the role through opencode (runner.opencode.model), which holds the pin, or remove the pin",
			layered.BoundKey(c.model))
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
	admits := c.launch
	if c.profile != nil {
		admits = admits.pinnedTo(*c.profile)
	}
	bin, err := admits.admit(Claude, "claude", req.Dir, req.Checkout)
	if err != nil {
		return Answer{}, nil, err
	}
	var seal []string
	launch := c.launch
	if c.profile != nil {
		if err := ResetStageHome(*c.profile); err != nil {
			return Answer{}, nil, fail(Claude, ReasonRefused, "%s", err.Error())
		}
		list := string(os.PathListSeparator)
		seal = append(slices.Clone(c.profile.Env), StageWritesEnv+"="+strings.Join(stageWriteDirs(req), list),
			StageDeclaredEnv+"="+strings.Join(req.Declared, list), StageBaseEnv+"="+req.Base, StageHarnessEnv+"="+bin)
		launch.carry = stageEnv
	}
	res, err := launch.run(ctx, Claude, bin, c.args(req), seal, req.Dir, req.timeout())
	transcript := res.transcript()
	if err != nil {
		// A harness that stopped on a rate limit exits non-zero, and one that
		// reported a limit and then waited until it was killed is that limit
		// too: what its events say is the reason, not the exit.
		if what := claudeRateLimit(res.stdout); what != "" {
			fl := limitFailure(what, res.stdout)
			fl.Seen, fl.Model = claudeSeen(res), req.modelOr(c.model)
			return Answer{}, transcript, fl
		}
		var fl *Failure
		if errors.As(err, &fl) {
			keepSeen(fl, res, req.modelOr(c.model))
			effortNamed(fl, req)
			if errors.Is(err, ErrKilled) {
				// Killed, it wrote no result event: what it spent is read
				// from the stream it did write (iss-2611021751406429).
				fl.Spent = spendOf(claudeSpent(res.stdout))
			}
		}
		return Answer{}, transcript, err
	}
	ans, err := parseClaude(res.stdout)
	if err != nil {
		var fl *Failure
		if errors.As(err, &fl) {
			keepSeen(fl, res, req.modelOr(c.model))
			effortNamed(fl, req)
		}
		return ans, transcript, err
	}
	ans.ModelAsked, ans.PromptSHA256 = c.askedModel(req), promptSHA256(req)
	ans.EffortAsked = req.Effort
	ans.Local = c.local
	ans.Unpinned = claudeUnpinned(ans)
	return ans, transcript, nil
}

// rateLimited is the claude runner's rate-limit failure, what naming the
// response in abcd's own words.
func rateLimited(what string) *Failure {
	return fail(Claude, ReasonRateLimited, "its event stream reports %s and the run did not finish", what)
}

// limitFailure is the rate-limit failure for what the event stream out reports,
// with the reset it names.
func limitFailure(what string, out []byte) *Failure {
	fl := rateLimited(what)
	fl.ResetsAt = claudeRateLimitReset(out)
	return fl
}

// claudeRateLimitReset is when the limit the event stream ends on resets, the zero
// time when it names none.
func claudeRateLimitReset(out []byte) time.Time {
	_, at := claudeLimitState(out)
	return at
}

// claudeRateLimit names the rate-limit response the event stream ends on, "" when
// it ends on none.
func claudeRateLimit(out []byte) string {
	what, _ := claudeLimitState(out)
	return what
}

// claudeLimitState reads the stream's latest state of its limits: what it reports,
// "" for none, and the reset it names, the zero time for none. Three shapes are
// read: a rate_limit_event whose rate_limit_info.status is rejected (allowed and
// allowed_warning are a run going on), an assistant message whose error is
// rate_limit, and a failed result that names the API's limit status. Which of them
// a limit produces in a headless run has not been observed here, so each stands on
// its own, and the evidence file a drive leaves says which one came. A later event
// takes back an earlier one: the same window's status, an assistant message with no
// error, or a result that is no error. Windows are told apart, so one window's
// status lifts no other's, and of several still rejected the latest reset stands.
// Each line is read leniently and on its own, so a field this reader does not know
// never turns a limit into an unparsable run.
func claudeLimitState(out []byte) (string, time.Time) {
	rejected := map[string]time.Time{}
	var assistant, result bool
	clearAll := func() {
		clear(rejected)
		assistant, result = false, false
	}
	for _, line := range bytes.Split(out, []byte("\n")) {
		line = bytes.TrimSpace(line)
		var ev struct {
			Type  string          `json:"type"`
			Error json.RawMessage `json:"error"`
			Info  json.RawMessage `json:"rate_limit_info"`
		}
		if json.Unmarshal(line, &ev) != nil {
			continue
		}
		switch ev.Type {
		case "rate_limit_event":
			var info struct {
				Status   string  `json:"status"`
				ResetsAt float64 `json:"resetsAt"`
				Window   string  `json:"rateLimitType"`
			}
			if json.Unmarshal(ev.Info, &info) != nil {
				continue
			}
			switch {
			case info.Status == "rejected":
				var at time.Time
				if info.ResetsAt > 0 {
					at = time.Unix(int64(info.ResetsAt), 0).UTC()
				}
				rejected[info.Window] = at
			case info.Window == "":
				clear(rejected)
			default:
				delete(rejected, info.Window)
			}
		case "assistant":
			var e string
			switch {
			case len(ev.Error) == 0:
				clearAll()
			case json.Unmarshal(ev.Error, &e) == nil && e == "rate_limit":
				assistant = true
			}
		case "result":
			var r struct {
				IsError bool   `json:"is_error"`
				Status  any    `json:"api_error_status"`
				API     string `json:"api_error"`
			}
			if json.Unmarshal(line, &r) != nil {
				continue
			}
			switch {
			case !r.IsError:
				clearAll()
			case fmt.Sprint(r.Status) == "429" || r.API == "usage_limit_reached":
				result = true
			}
		}
	}
	var reset time.Time
	for _, at := range rejected {
		if at.After(reset) {
			reset = at
		}
	}
	switch {
	case len(rejected) > 0:
		return "a rejected rate_limit_event", reset
	case assistant:
		return "an assistant message whose error is rate_limit", reset
	case result:
		return "a failed result naming the API's limit", reset
	}
	return "", reset
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

// claudeFacts is what the ledger reads from a claude launch's stream beyond
// what the run's record keeps.
type claudeFacts struct {
	// window is true when the stream carries a rate-limit event.
	window bool
	// apiMillis is the result event's duration_api_ms, nil when it has none.
	apiMillis *int64
	// model is the init event's model, "" when it names none readably.
	model string
}

// claudeLaunchFacts reads a stream line by line, leniently, and stops at the
// harness's stderr, which is not its event stream.
func claudeLaunchFacts(transcript []byte) claudeFacts {
	var f claudeFacts
	for _, line := range bytes.Split(transcript, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if string(line) == stderrMarker {
			break
		}
		var ev struct {
			Type    string `json:"type"`
			Subtype string `json:"subtype"`
			Model   string `json:"model"`
			APIMs   *int64 `json:"duration_api_ms"`
		}
		if len(line) == 0 || json.Unmarshal(line, &ev) != nil {
			continue
		}
		switch ev.Type {
		case "rate_limit_event":
			f.window = true
		case "result":
			if ev.APIMs != nil && *ev.APIMs >= 0 {
				f.apiMillis = ev.APIMs
			}
		case "system":
			if ev.Subtype == "init" && modelRe.MatchString(ev.Model) {
				f.model = ev.Model
			}
		}
	}
	return f
}

// claudeSpent reads what a killed claude run spent from the stream it wrote
// (iss-2611021751406429): the result event's usage and cost when one came
// before the kill, else the usage its assistant messages carry, summed once
// per message (the stream repeats a message's usage on each of its content
// blocks, so the last seen for an id stands) and with no cost, which only the
// result event reports. Every line is read leniently: a line that is not an
// event is skipped, never a reason to drop what the others show.
func claudeSpent(out []byte) Answer {
	type usage struct {
		InputTokens              *int `json:"input_tokens"`
		OutputTokens             *int `json:"output_tokens"`
		CacheReadInputTokens     *int `json:"cache_read_input_tokens"`
		CacheCreationInputTokens *int `json:"cache_creation_input_tokens"`
	}
	var ans Answer
	byID := map[string]usage{}
	var order []string
	for _, line := range bytes.Split(out, []byte("\n")) {
		line = bytes.TrimSpace(line)
		var ev struct {
			Type    string `json:"type"`
			Message struct {
				ID    string `json:"id"`
				Usage *usage `json:"usage"`
			} `json:"message"`
		}
		if len(line) == 0 || json.Unmarshal(line, &ev) != nil {
			continue
		}
		switch ev.Type {
		case "result":
			var r Answer
			meterClaude(&r, line)
			if r.Usage != nil || r.Cost != nil {
				return r
			}
		case "assistant":
			if ev.Message.Usage == nil {
				continue
			}
			id := ev.Message.ID
			if id == "" {
				id = strconv.Itoa(len(order))
			}
			if _, seen := byID[id]; !seen {
				order = append(order, id)
			}
			byID[id] = *ev.Message.Usage
		}
	}
	if len(order) == 0 {
		return ans
	}
	sum := &oracle.Usage{}
	add := func(to *int, n *int) {
		if n != nil && *n >= 0 && *to <= math.MaxInt-*n {
			*to += *n
		}
	}
	for _, id := range order {
		u := byID[id]
		add(&sum.PromptTokens, u.InputTokens)
		add(&sum.CompletionTokens, u.OutputTokens)
		add(&sum.CacheReadTokens, u.CacheReadInputTokens)
		add(&sum.CacheWriteTokens, u.CacheCreationInputTokens)
	}
	ans.Usage = sum
	return ans
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
			return Answer{}, limitFailure(what, out)
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
