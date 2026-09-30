package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestClaudeLaunchPassesTheBareFlag is criterion 6: the claude CLI runner
// launches in print mode with --bare, so the target repository's hooks and
// configured servers do not run, and grants the role's contract's tools
// without a prompt.
func TestClaudeLaunchPassesTheBareFlag(t *testing.T) {
	f := newFake(t, "ok", Claude)
	ans, transcript, err := newClaude("").Run(context.Background(), f.request("ruthless-reviewer"))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	argv := f.argv(t, Claude)
	for _, want := range []string{"--print", "--bare", "--output-format", "stream-json", "--no-session-persistence",
		"--permission-mode", "dontAsk", "--allowedTools=Read,Grep"} {
		if !slices.Contains(argv, want) {
			t.Errorf("claude argv %q lacks %q", argv, want)
		}
	}
	if argv[len(argv)-2] != "--" {
		t.Errorf("the prompt is not behind the end-of-options marker: %q", argv)
	}
	if ans.SessionID != "fake-session-1" || ans.Model != "fake-model" || ans.Text != "done" {
		t.Errorf("answer = %+v", ans)
	}
	if !strings.Contains(string(transcript), `"type":"result"`) {
		t.Errorf("transcript does not carry the event stream: %q", transcript)
	}
}

// TestClaudeModelRouteReachesTheLaunch: a configured model route is passed as
// the model the harness asks for, as one argument.
func TestClaudeModelRouteReachesTheLaunch(t *testing.T) {
	f := newFake(t, "ok", Claude)
	if _, _, err := newClaude("local/qwen3-coder").Run(context.Background(), f.request("scribe")); err != nil {
		t.Fatalf("run: %v", err)
	}
	if argv := f.argv(t, Claude); !slices.Contains(argv, "--model=qwen3-coder") {
		t.Errorf("claude argv %q lacks the model", argv)
	}
}

// TestOpenCodeLaunch: the opencode runner uses run mode with raw JSON events,
// no external plugins, the repository as its directory, and the brief attached.
func TestOpenCodeLaunch(t *testing.T) {
	f := newFake(t, "ok", OpenCode)
	req := f.request("ruthless-reviewer")
	ans, _, err := newOpenCode("openrouter/qwen/qwen3-coder").Run(context.Background(), req)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	argv := f.argv(t, OpenCode)
	if argv[0] != "run" {
		t.Errorf("opencode argv %q does not start with run", argv)
	}
	for _, want := range []string{"--format=json", "--pure", "--dir=" + req.Dir, "--file=" + req.Brief,
		"--model=openrouter/qwen/qwen3-coder"} {
		if !slices.Contains(argv, want) {
			t.Errorf("opencode argv %q lacks %q", argv, want)
		}
	}
	if argv[len(argv)-2] != "--" {
		t.Errorf("the prompt is not behind the end-of-options marker: %q", argv)
	}
	if ans.SessionID != "ses_fake1" || ans.Text != "done" || ans.Model != "openrouter/qwen/qwen3-coder" {
		t.Errorf("answer = %+v", ans)
	}
}

// TestSameBriefAndContractEveryRoute is criterion 1's input half: every
// runner is handed the one prompt prompt renders, naming the role, the brief
// and the receipt path the host sub-agent is handed.
func TestSameBriefAndContractEveryRoute(t *testing.T) {
	f := newFake(t, "ok", Claude, OpenCode)
	req := f.request("security-reviewer")
	if _, _, err := newClaude("").Run(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if _, _, err := newOpenCode("").Run(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	want := prompt(req)
	for _, h := range []string{Claude, OpenCode} {
		argv := f.argv(t, h)
		if got := argv[len(argv)-1]; got != want {
			t.Errorf("%s prompt = %q, want %q", h, got, want)
		}
	}
	for _, part := range []string{"security-reviewer", "Brief: " + req.Brief, "Receipt: " + req.Receipt} {
		if !strings.Contains(want, part) {
			t.Errorf("prompt %q lacks %q", want, part)
		}
	}
}

// TestFailureKinds: each way a harness can fail is a Failure naming its kind.
func TestFailureKinds(t *testing.T) {
	for _, tc := range []struct {
		mode    string
		harness string
		want    Reason
	}{
		{"exit1", Claude, ReasonFailed},
		{"garbage", Claude, ReasonUnparsable},
		{"refuse", Claude, ReasonRefused},
		{"exit1", OpenCode, ReasonFailed},
		{"garbage", OpenCode, ReasonUnparsable},
		{"refuse", OpenCode, ReasonRefused},
	} {
		t.Run(tc.harness+"-"+tc.mode, func(t *testing.T) {
			f := newFake(t, tc.mode, tc.harness)
			var r Runner = newClaude("")
			if tc.harness == OpenCode {
				r = newOpenCode("")
			}
			_, _, err := r.Run(context.Background(), f.request("scribe"))
			var fl *Failure
			if !errors.As(err, &fl) || fl.Reason != tc.want {
				t.Fatalf("err = %v, want a %s failure", err, tc.want)
			}
		})
	}
}

// TestAbsentBinaryIsAbsent: a harness that is not on PATH is absent, and
// nothing runs.
func TestAbsentBinaryIsAbsent(t *testing.T) {
	f := newFake(t, "ok")
	_, _, err := newClaude("").Run(context.Background(), f.request("scribe"))
	var fl *Failure
	if !errors.As(err, &fl) || fl.Reason != ReasonAbsent {
		t.Fatalf("err = %v, want absent", err)
	}
}

// TestBinaryInsideTheRepositoryIsRefused: a PATH entry that resolves inside
// the repository the role runs in is repository content, never run.
func TestBinaryInsideTheRepositoryIsRefused(t *testing.T) {
	f := newFake(t, "ok")
	self, _ := os.Executable()
	planted := filepath.Join(f.repo, "tools")
	if err := os.Mkdir(planted, 0o700); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(planted, Claude), raw, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", planted)
	_, _, err = newClaude("").Run(context.Background(), f.request("scribe"))
	var fl *Failure
	if !errors.As(err, &fl) || fl.Reason != ReasonAbsent {
		t.Fatalf("err = %v, want the planted binary refused as absent", err)
	}
	if f.launched(Claude) {
		t.Fatal("the binary inside the repository was launched")
	}
}

// TestLaunchEnvironmentIsScrubbed: an inherited repository-selection or config
// injection variable never reaches the harness, and the launch runs in the
// repository directory with no stdin.
func TestLaunchEnvironmentIsScrubbed(t *testing.T) {
	f := newFake(t, "ok", Claude)
	t.Setenv("GIT_DIR", "/elsewhere/.git")
	t.Setenv("GIT_CONFIG_PARAMETERS", "'core.hooksPath'='/elsewhere'")
	if _, _, err := newClaude("").Run(context.Background(), f.request("scribe")); err != nil {
		t.Fatal(err)
	}
	env, _ := os.ReadFile(filepath.Join(f.log, Claude+".env"))
	for _, bad := range []string{"GIT_DIR=", "GIT_CONFIG_PARAMETERS="} {
		if strings.Contains(string(env), "\n"+bad) || strings.HasPrefix(string(env), bad) {
			t.Errorf("the harness inherited %s", bad)
		}
	}
	cwd, _ := os.ReadFile(filepath.Join(f.log, Claude+".cwd"))
	want, _ := filepath.EvalSymlinks(f.repo)
	if got, _ := filepath.EvalSymlinks(string(cwd)); got != want {
		t.Errorf("harness ran in %q, want %q", got, want)
	}
}

// TestNoCredentialInArgumentOrError: the harness's own credential stays in its
// environment; abcd never puts it on the command line, and a harness that
// echoes it while failing does not carry it into the error abcd returns.
func TestNoCredentialInArgumentOrError(t *testing.T) {
	f := newFake(t, "exit1", Claude)
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	cred := "abcd-fake-credential-" + hex.EncodeToString(b)
	t.Setenv(fakeCredEnv, cred)
	_, _, err := newClaude("").Run(context.Background(), f.request("scribe"))
	if err == nil {
		t.Fatal("a failing harness returned no error")
	}
	if strings.Contains(err.Error(), cred) {
		t.Errorf("the error carries the credential: %v", err)
	}
	for _, a := range f.argv(t, Claude) {
		if strings.Contains(a, cred) {
			t.Errorf("an argument carries the credential: %q", a)
		}
	}
	env, _ := os.ReadFile(filepath.Join(f.log, Claude+".env"))
	if !strings.Contains(string(env), fakeCredEnv+"="+cred) {
		t.Error("the harness did not receive its own credential from the environment")
	}
}

// TestTimeoutKillsTheProcessGroup: a harness past its time is killed with
// every process in the group the runner started, and reported as failed.
func TestTimeoutKillsTheProcessGroup(t *testing.T) {
	f := newFake(t, "hang", Claude)
	req := f.request("scribe")
	req.Timeout = 2 * time.Second
	_, _, err := newClaude("").Run(context.Background(), req)
	var fl *Failure
	if !errors.As(err, &fl) || fl.Reason != ReasonFailed || !strings.Contains(fl.Detail, "time") {
		t.Fatalf("err = %v, want a failure for running out of time", err)
	}
	raw, rerr := os.ReadFile(filepath.Join(f.log, "child.pid"))
	if rerr != nil {
		t.Fatalf("the fake never started its child: %v", rerr)
	}
	pid, _ := strconv.Atoi(string(raw))
	deadline := time.Now().Add(10 * time.Second)
	for syscall.Kill(pid, 0) == nil {
		if time.Now().After(deadline) {
			_ = syscall.Kill(pid, syscall.SIGKILL) // our own fake's child, by its pid
			t.Fatalf("the harness's child %d outlived the kill", pid)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// TestOutputIsBounded: a harness writing past the bound is cut off and
// reported, never read into memory whole.
func TestOutputIsBounded(t *testing.T) {
	f := newFake(t, "flood", Claude)
	c := newClaude("")
	c.launch.maxStdout = 64 << 10
	_, transcript, err := c.Run(context.Background(), f.request("scribe"))
	var fl *Failure
	if !errors.As(err, &fl) || !strings.Contains(fl.Detail, "bound") {
		t.Fatalf("err = %v, want a failure naming the output bound", err)
	}
	if len(transcript) > 64<<10+4096 {
		t.Errorf("transcript is %d bytes, past the bound", len(transcript))
	}
}

// TestRequestIsCheckedBeforeLaunch: a relative path, a tool list that would
// split the flag, or a role that is not a plain name refuses before anything
// runs.
func TestRequestIsCheckedBeforeLaunch(t *testing.T) {
	f := newFake(t, "ok", Claude)
	for name, mut := range map[string]func(*Request){
		"relative brief": func(r *Request) { r.Brief = "brief.md" },
		"relative dir":   func(r *Request) { r.Dir = "repo" },
		"comma tool":     func(r *Request) { r.Tools = []string{"Read,Bash"} },
		"role":           func(r *Request) { r.Role = "../x" },
		"no session":     func(r *Request) { r.SessionID = "" },
	} {
		req := f.request("scribe")
		mut(&req)
		if _, _, err := newClaude("").Run(context.Background(), req); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if f.launched(Claude) {
		t.Error("a refused request launched the harness")
	}
}

// TestBinaryInsideTheCheckoutIsRefused: a lane's worktree lives outside the
// checkout the run belongs to, so a program planted in that checkout and put
// on PATH is refused too, though it is not inside the directory the role
// runs in.
func TestBinaryInsideTheCheckoutIsRefused(t *testing.T) {
	f := newFake(t, "ok")
	self, _ := os.Executable()
	checkout := t.TempDir()
	planted := filepath.Join(checkout, "tools")
	if err := os.Mkdir(planted, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(self, filepath.Join(planted, OpenCode)); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", planted)
	req := f.request("scribe")
	req.Checkout = checkout
	_, _, err := newOpenCode("").Run(context.Background(), req)
	var fl *Failure
	if !errors.As(err, &fl) || fl.Reason != ReasonAbsent {
		t.Fatalf("err = %v, want the binary planted in the checkout refused as absent", err)
	}
	if f.launched(OpenCode) {
		t.Fatal("the binary inside the checkout was launched")
	}
}
