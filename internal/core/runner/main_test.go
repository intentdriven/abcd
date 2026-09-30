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
func validReceipt(req Request, _ Answer) error {
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
