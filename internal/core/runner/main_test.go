package runner

// main_test.go is the fake command-line harness every test drives. No test
// reaches a real harness or a paid model: the test binary re-executes itself
// under the name of the harness a test puts on PATH (claude, opencode), and
// TestMain, seeing ABCD_RUNNER_FAKE, plays that harness instead of running the
// tests. What it does is chosen by the mode in ABCD_RUNNER_FAKE; what it saw
// (its argv and its environment) it writes under ABCD_RUNNER_FAKE_LOG, so a
// test asserts on the launch the runner really made.

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	fakeModeEnv = "ABCD_RUNNER_FAKE"
	fakeLogEnv  = "ABCD_RUNNER_FAKE_LOG"
	// fakeCredEnv is the credential variable a harness reads its key from;
	// the test sets it to a value built at run time and asserts it never
	// reaches an argument, an error or a receipt.
	fakeCredEnv = "ANTHROPIC_API_KEY"
	// fakeStreamEnv names the fixture the stream mode answers with.
	fakeStreamEnv = "ABCD_RUNNER_FAKE_STREAM"
)

func TestMain(m *testing.M) {
	if mode := os.Getenv(fakeModeEnv); mode != "" {
		os.Exit(fakeHarness(mode))
	}
	os.Exit(m.Run())
}

// fakeHarness plays the harness its own name says it is.
func fakeHarness(mode string) int {
	// The runner executes the resolved binary, so argv[0] names the test
	// binary; which harness this is shows in the launch itself.
	name := "claude"
	if len(os.Args) > 1 && os.Args[1] == "run" {
		name = "opencode"
	}
	logDir := os.Getenv(fakeLogEnv)
	// opencode's version, which a pinned run asks for before it runs; the
	// badversion mode answers with something that is not one.
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		if logDir != "" {
			_ = os.WriteFile(filepath.Join(logDir, "version.env"), []byte(strings.Join(os.Environ(), "\n")), 0o600)
		}
		if mode == "badversion" {
			fmt.Println("\x1b[2Jnot a version")
			return 0
		}
		fmt.Println("1.18.31")
		return 0
	}
	if mode == "badversion" {
		mode = "ok"
	}
	// The configuration a pinned run points opencode at, copied as the run
	// saw it, with its own mode and its directory's: the run deletes it.
	if cfg := os.Getenv("OPENCODE_CONFIG"); cfg != "" && name == "opencode" && logDir != "" {
		if raw, err := os.ReadFile(cfg); err == nil {
			_ = os.WriteFile(filepath.Join(logDir, "opencode.config.json"), raw, 0o600)
		}
		fi, ferr := os.Stat(cfg)
		di, derr := os.Stat(filepath.Dir(cfg))
		if ferr == nil && derr == nil {
			_ = os.WriteFile(filepath.Join(logDir, "opencode.config.mode"),
				[]byte(fmt.Sprintf("%o %o", fi.Mode().Perm(), di.Mode().Perm())), 0o600)
		}
	}
	if logDir != "" && mode != "sleeper" {
		argv, _ := json.Marshal(os.Args[1:])
		_ = os.WriteFile(filepath.Join(logDir, name+".argv.json"), argv, 0o600)
		_ = os.WriteFile(filepath.Join(logDir, name+".env"), []byte(strings.Join(os.Environ(), "\n")), 0o600)
		wd, _ := os.Getwd()
		_ = os.WriteFile(filepath.Join(logDir, name+".cwd"), []byte(wd), 0o600)
	}
	prompt := ""
	if len(os.Args) > 1 {
		prompt = os.Args[len(os.Args)-1]
	}
	switch mode {
	case "sleeper":
		time.Sleep(60 * time.Second)
		return 0
	case "hang":
		child := exec.Command(os.Args[0])
		child.Env = append(os.Environ(), fakeModeEnv+"=sleeper")
		if err := child.Start(); err != nil {
			return 3
		}
		_ = os.WriteFile(filepath.Join(logDir, "child.pid"), []byte(strconv.Itoa(child.Process.Pid)), 0o600)
		time.Sleep(60 * time.Second)
		return 0
	case "flood":
		chunk := strings.Repeat("x", 4096)
		for i := 0; i < 1024; i++ {
			fmt.Fprint(os.Stdout, chunk)
		}
		return 0
	case "exit1":
		cred := os.Getenv(fakeCredEnv)
		fmt.Fprintf(os.Stdout, "auth failed for key %s\n", cred)
		fmt.Fprintf(os.Stderr, "error: invalid key %s\n", cred)
		return 1
	case "garbage":
		fmt.Fprintln(os.Stdout, "this is not a structured event stream")
		return 0
	case "refuse":
		if name == "opencode" {
			fmt.Fprintln(os.Stdout, `{"type":"error","sessionID":"ses_fake1","error":{"name":"APIError","data":{"message":"model refused"}}}`)
		} else {
			fmt.Fprintln(os.Stdout, `{"type":"system","subtype":"init","session_id":"fake-session-1","model":"fake-model"}`)
			fmt.Fprintln(os.Stdout, `{"type":"result","subtype":"error_during_execution","is_error":true,"result":"refused","session_id":"fake-session-1"}`)
		}
		return 0
	case "ratelimit-exit", "ratelimit-result", "ratelimit-assistant", "ratelimit-warning":
		// A claude run that meets a rate limit, in the shapes its event
		// stream carries one (the claude CLI 2.1's rate_limit_event, whose
		// rate_limit_info.status is allowed, allowed_warning or rejected, and
		// an assistant message whose error is rate_limit): rejected and the
		// process exits non-zero; rejected and the result reports the error
		// with a zero exit; the assistant's error alone; and a warning the
		// run finishes past, which is no limit at all.
		fmt.Fprintln(os.Stdout, `{"type":"system","subtype":"init","session_id":"fake-session-1","model":"fake-model"}`)
		switch mode {
		case "ratelimit-warning":
			fmt.Fprintln(os.Stdout, `{"type":"rate_limit_event","rate_limit_info":{"status":"allowed_warning","resetsAt":1791000000,"rateLimitType":"five_hour","utilization":0.91},"session_id":"fake-session-1"}`)
			fmt.Fprintln(os.Stdout, `{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"fake-session-1"}`)
			return 0
		case "ratelimit-assistant":
			fmt.Fprintln(os.Stdout, `{"type":"assistant","message":{"content":[{"type":"text","text":"limited"}]},"error":"rate_limit","session_id":"fake-session-1"}`)
		default:
			fmt.Fprintln(os.Stdout, `{"type":"rate_limit_event","rate_limit_info":{"status":"rejected","resetsAt":1791000000,"rateLimitType":"five_hour"},"session_id":"fake-session-1"}`)
		}
		fmt.Fprintln(os.Stdout, `{"type":"result","subtype":"success","is_error":true,"result":"limit reached","session_id":"fake-session-1"}`)
		if mode == "ratelimit-result" {
			return 0
		}
		return 1
	case "hugemodel", "ctrlmodel", "suffixmodel":
		// A harness whose init event reports a model: one past any bound,
		// one carrying control bytes, and a real id's bracketed suffix.
		model := map[string]string{
			"hugemodel":   strings.Repeat("m", 5<<20),
			"ctrlmodel":   "fake\x1b[2Jmodel\r\n",
			"suffixmodel": "claude-opus-4-6[1m]",
		}[mode]
		init, _ := json.Marshal(map[string]string{"type": "system", "subtype": "init", "session_id": "fake-session-1", "model": model})
		fmt.Fprintln(os.Stdout, string(init))
		fmt.Fprintln(os.Stdout, `{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"fake-session-1"}`)
		return 0
	case "stream":
		// A harness that answers with a recorded event stream: the file
		// fakeStreamEnv names, a fixture under testdata/.
		if rec := promptField(prompt, "Receipt: "); rec != "" {
			_ = os.WriteFile(rec, []byte(`{"ok":true}`), 0o600)
		}
		raw, err := os.ReadFile(os.Getenv(fakeStreamEnv))
		if err != nil {
			return 3
		}
		_, _ = os.Stdout.Write(raw)
		return 0
	case "approve":
		// A harness that puts two permission prompts to the prompt tool its
		// launch names, through the MCP server its --mcp-config starts, and
		// logs the answers it got.
		if err := fakePrompts(os.Args[1:], logDir); err != nil {
			fmt.Fprintf(os.Stderr, "fake harness: %v\n", err)
			return 3
		}
		fmt.Fprintln(os.Stdout, `{"type":"system","subtype":"init","session_id":"fake-session-1","model":"fake-model"}`)
		fmt.Fprintln(os.Stdout, `{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"fake-session-1"}`)
		return 0
	case "ok", "noreceipt":
		if mode == "ok" {
			if rec := promptField(prompt, "Receipt: "); rec != "" {
				_ = os.WriteFile(rec, []byte(`{"ok":true}`), 0o600)
			}
		}
		if name == "opencode" {
			fmt.Fprintln(os.Stdout, `{"type":"step_start","sessionID":"ses_fake1","part":{"type":"step-start"}}`)
			fmt.Fprintln(os.Stdout, `{"type":"text","sessionID":"ses_fake1","part":{"type":"text","text":"done"}}`)
			fmt.Fprintln(os.Stdout, `{"type":"step_finish","sessionID":"ses_fake1","part":{"type":"step-finish"}}`)
		} else {
			fmt.Fprintln(os.Stdout, `{"type":"system","subtype":"init","session_id":"fake-session-1","model":"fake-model"}`)
			fmt.Fprintln(os.Stdout, `{"type":"assistant","message":{"content":[{"type":"text","text":"done"}]},"session_id":"fake-session-1"}`)
			fmt.Fprintln(os.Stdout, `{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"fake-session-1"}`)
		}
		return 0
	}
	fmt.Fprintf(os.Stderr, "fake harness: unknown mode %q\n", mode)
	return 2
}

// fakePrompts plays the harness's side of the permission prompt tool: it
// starts the one server --mcp-config names, as the harness does, shakes
// hands, and calls the tool --permission-prompt-tool names once per command
// in fakePromptCommands, writing each answer's text to approver.answers.json.
func fakePrompts(argv []string, logDir string) error {
	var cfg struct {
		MCPServers map[string]struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"mcpServers"`
	}
	var tool string
	for _, a := range argv {
		if v, ok := strings.CutPrefix(a, "--mcp-config="); ok {
			if err := json.Unmarshal([]byte(v), &cfg); err != nil {
				return err
			}
		}
		if v, ok := strings.CutPrefix(a, "--permission-prompt-tool="); ok {
			tool = v
		}
	}
	server, name, ok := strings.Cut(strings.TrimPrefix(tool, "mcp__"), "__")
	srv, found := cfg.MCPServers[server]
	if !ok || !found {
		return fmt.Errorf("the prompt tool %q names no server the launch starts", tool)
	}
	cmd := exec.Command(srv.Command, srv.Args...)
	cmd.Env = os.Environ()
	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	out := bufio.NewScanner(outPipe)
	out.Buffer(make([]byte, 1<<20), 1<<20)
	call := func(id int, method string, params any) (json.RawMessage, error) {
		b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
		if _, err := fmt.Fprintf(in, "%s\n", b); err != nil {
			return nil, err
		}
		if !out.Scan() {
			return nil, fmt.Errorf("the server closed before answering %s", method)
		}
		var r struct {
			Result json.RawMessage `json:"result"`
			Error  json.RawMessage `json:"error"`
		}
		if err := json.Unmarshal(out.Bytes(), &r); err != nil || r.Error != nil {
			return nil, fmt.Errorf("%s: %s", method, out.Bytes())
		}
		return r.Result, nil
	}
	if _, err := call(0, "initialize", map[string]any{"protocolVersion": "2025-11-25", "capabilities": map[string]any{},
		"clientInfo": map[string]any{"name": "fake", "version": "0"}}); err != nil {
		return err
	}
	fmt.Fprintln(in, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	answers := map[string]string{}
	for i, c := range fakePromptCommands {
		res, err := call(i+1, "tools/call", map[string]any{"name": name, "arguments": map[string]any{
			"tool_name": "Bash", "input": map[string]any{"command": c}, "tool_use_id": fmt.Sprintf("toolu_%d", i)}})
		if err != nil {
			return err
		}
		var content struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := json.Unmarshal(res, &content); err != nil || len(content.Content) != 1 {
			return fmt.Errorf("the answer %s is not one text block", res)
		}
		answers[c] = content.Content[0].Text
	}
	_ = in.Close()
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("the server did not exit cleanly: %w", err)
	}
	raw, _ := json.Marshal(answers)
	return os.WriteFile(filepath.Join(logDir, "approver.answers.json"), raw, 0o600)
}

// fakePromptCommands are the shell commands the approve mode puts to the
// prompt tool: a hazard and a command the guard clears.
var fakePromptCommands = []string{fakeHazard, "go test ./..."}

// fakeHazard is the hazard the approve mode puts to the prompt tool.
const fakeHazard = "git push --force origin main"

// promptField returns the value of the prompt line that starts with prefix.
func promptField(prompt, prefix string) string {
	sc := bufio.NewScanner(strings.NewReader(prompt))
	for sc.Scan() {
		if v, ok := strings.CutPrefix(sc.Text(), prefix); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// fakeEnv is one test's fake harness set-up: a PATH directory holding the
// harnesses as links to the test binary, a log directory, and a repository
// directory with a brief in it.
type fakeEnv struct {
	bin, log, repo, lane string
}

// newFake puts the named harnesses on PATH (and nothing else of the test's),
// sets the mode, and returns the directories.
func newFake(t *testing.T, mode string, harnesses ...string) fakeEnv {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	f := fakeEnv{
		bin:  filepath.Join(root, "bin"),
		log:  filepath.Join(root, "log"),
		repo: filepath.Join(root, "repo"),
		lane: filepath.Join(root, "lane"),
	}
	for _, d := range []string{f.bin, f.log, f.repo, f.lane} {
		if err := os.Mkdir(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for _, h := range harnesses {
		if err := os.Symlink(self, filepath.Join(f.bin, h)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(f.lane, "brief.md"), []byte("# brief\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", f.bin)
	// The person's own opencode configuration, if the machine running the
	// tests names one, is not the test's: a test that wants one sets it.
	t.Setenv("OPENCODE_CONFIG", "")
	t.Setenv("OPENCODE_CONFIG_CONTENT", "")
	t.Setenv(fakeModeEnv, mode)
	t.Setenv(fakeLogEnv, f.log)
	return f
}

// request is the request every test hands a runner.
func (f fakeEnv) request(role string) Request {
	return Request{
		Role:      role,
		Brief:     filepath.Join(f.lane, "brief.md"),
		Receipt:   filepath.Join(f.lane, "receipt.json"),
		Dir:       f.repo,
		Tools:     []string{"Read", "Grep"},
		SessionID: "run-2609300400000000-lane-1-" + role,
		Timeout:   20 * time.Second,
	}
}

// argv returns what the named fake harness was launched with.
func (f fakeEnv) argv(t *testing.T, harness string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(f.log, harness+".argv.json"))
	if err != nil {
		t.Fatalf("the %s fake was not launched: %v", harness, err)
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// launched reports whether the named fake harness ran at all.
func (f fakeEnv) launched(harness string) bool {
	_, err := os.Stat(filepath.Join(f.log, harness+".argv.json"))
	return err == nil
}

// validReceipt is the contract's validator the tests hand the dispatcher: the
// receipt exists and is the JSON the fake writes.
func validReceipt(_ string, req Request, _ Answer) error {
	raw, err := os.ReadFile(req.Receipt)
	if err != nil {
		return fmt.Errorf("no receipt at the contract's path")
	}
	var v struct {
		OK bool `json:"ok"`
	}
	if json.Unmarshal(raw, &v) != nil || !v.OK {
		return fmt.Errorf("the receipt is not the contract's shape")
	}
	return nil
}
