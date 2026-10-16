package runner

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/guard"
)

// shellRequest is a request whose contract grants the whole shell, as a build
// lane's implementer's and reviewers' do.
func (f fakeEnv) shellRequest(role string) Request {
	r := f.request(role)
	r.Tools = []string{"Read", "Edit", "Bash", "Grep"}
	r.Base = fakeBase
	return r
}

// fakeBase is the commit a shell request says its run started on.
const fakeBase = "0123456789abcdef0123456789abcdef01234567"

// mcpLaunch is the approver the launch's --mcp-config names.
type mcpLaunch struct {
	MCPServers map[string]struct {
		Type    string   `json:"type"`
		Command string   `json:"command"`
		Args    []string `json:"args"`
	} `json:"mcpServers"`
}

// flagValue returns the value of the --name=value argument in argv.
func flagValue(argv []string, name string) (string, bool) {
	for _, a := range argv {
		if v, ok := strings.CutPrefix(a, name+"="); ok {
			return v, true
		}
	}
	return "", false
}

// TestAShellLaunchCarriesTheGuardApprover: a headless launch whose contract
// grants the shell hands the harness abcd's guard as the answerer of its
// permission prompts, for that launch only, on the command line: the prompt
// tool, the one server it names, no other server, and the default mode, under
// which a prompt reaches the tool. The shell is no longer granted without a
// prompt, so every shell command the harness would ask about reaches the
// guard; the other tools stay granted.
func TestAShellLaunchCarriesTheGuardApprover(t *testing.T) {
	f := newFake(t, "ok", Claude)
	c := newClaude("")
	c.approver = "/opt/abcd/bin/abcd"
	if _, _, err := c.Run(context.Background(), f.shellRequest("implementer")); err != nil {
		t.Fatalf("run: %v", err)
	}
	argv := f.argv(t, Claude)
	for _, want := range []string{"--print", "--bare", "--permission-prompt-tool=" + ApproverTool, "--strict-mcp-config",
		"--allowedTools=Read,Edit,Grep"} {
		if !slices.Contains(argv, want) {
			t.Errorf("claude argv %q lacks %q", argv, want)
		}
	}
	if i := slices.Index(argv, "--permission-mode"); i < 0 || argv[i+1] != "default" {
		t.Errorf("claude argv %q does not ask for the default permission mode", argv)
	}
	if slices.Contains(argv, "dontAsk") {
		t.Errorf("claude argv %q still refuses every prompt unasked", argv)
	}
	raw, ok := flagValue(argv, "--mcp-config")
	if !ok {
		t.Fatalf("claude argv %q names no MCP server", argv)
	}
	var cfg mcpLaunch
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatalf("--mcp-config is not JSON: %v (%s)", err, raw)
	}
	srv, ok := cfg.MCPServers["abcd"]
	if len(cfg.MCPServers) != 1 || !ok {
		t.Fatalf("--mcp-config servers = %+v; want abcd alone", cfg.MCPServers)
	}
	if srv.Type != "stdio" || srv.Command != c.approver ||
		!slices.Equal(srv.Args, []string{"guard", "approve", "--dir=" + f.repo, "--base=" + fakeBase}) {
		t.Errorf("the approver server = %+v", srv)
	}
	if argv[len(argv)-2] != "--" {
		t.Errorf("the prompt is not behind the end-of-options marker: %q", argv)
	}
}

// TestALaunchWithoutTheShellIsUnchanged: a contract that grants no shell has
// no shell prompt for the guard to answer, so its launch is the one it was:
// every prompt refused unasked, and no server started.
func TestALaunchWithoutTheShellIsUnchanged(t *testing.T) {
	f := newFake(t, "ok", Claude)
	c := newClaude("")
	c.approver = "/opt/abcd/bin/abcd"
	if _, _, err := c.Run(context.Background(), f.request("ruthless-reviewer")); err != nil {
		t.Fatalf("run: %v", err)
	}
	argv := f.argv(t, Claude)
	if i := slices.Index(argv, "--permission-mode"); i < 0 || argv[i+1] != "dontAsk" {
		t.Errorf("claude argv %q does not refuse prompts unasked", argv)
	}
	for _, a := range argv {
		if strings.HasPrefix(a, "--mcp-config") || strings.HasPrefix(a, "--permission-prompt-tool") || a == "--strict-mcp-config" {
			t.Errorf("a launch with no shell carries %q", a)
		}
	}
}

// TestAShellLaunchWithNoApproverIsNotStarted: when abcd cannot name its own
// binary for the approver, the shell launch is refused before it starts,
// rather than started with the shell granted unchecked or every prompt
// refused.
func TestAShellLaunchWithNoApproverIsNotStarted(t *testing.T) {
	f := newFake(t, "ok", Claude)
	c := newClaude("")
	c.approver = ""
	_, _, err := c.Run(context.Background(), f.shellRequest("implementer"))
	var fl *Failure
	if !errors.As(err, &fl) || fl.Reason != ReasonFailed {
		t.Fatalf("err = %v; want a failed launch", err)
	}
	if f.launched(Claude) {
		t.Fatal("the harness was started with no approver")
	}
}

// TestAShellLaunchWithNoBaseIsNotStarted: the approver judges against the
// registry at the commit the run started on, so a shell launch that names no
// such commit is refused before it starts, and so is one whose base is not a
// commit's full object name.
func TestAShellLaunchWithNoBaseIsNotStarted(t *testing.T) {
	for _, base := range []string{"", "HEAD", "0123456", "--all"} {
		f := newFake(t, "ok", Claude)
		c := newClaude("")
		c.approver = "/opt/abcd/bin/abcd"
		req := f.shellRequest("implementer")
		req.Base = base
		if _, _, err := c.Run(context.Background(), req); err == nil {
			t.Errorf("base %q: the shell launch was started", base)
		}
		if f.launched(Claude) {
			t.Errorf("base %q: the harness was started", base)
		}
	}
}

// TestAHostRoleCarriesNoPolicy: a role the host session runs is handed back
// to it untouched; abcd launches nothing, so no prompt of the host's is
// answered by abcd.
func TestAHostRoleCarriesNoPolicy(t *testing.T) {
	f := newFake(t, "ok", Claude, OpenCode)
	h := newHarness(t, mustLoad(t, routedMachine, ""), true)
	out, err := h.d.Dispatch(context.Background(), f.shellRequest("implementer"))
	if err != nil {
		t.Fatal(err)
	}
	if !out.Handoff || f.launched(Claude) || f.launched(OpenCode) {
		t.Fatalf("a host role: outcome %+v, launched claude %v, opencode %v", out, f.launched(Claude), f.launched(OpenCode))
	}
}

// TestARoutedShellRoleCarriesTheApproverThroughDispatch: the dispatcher's own
// claude launch, as a build lane makes it, carries the approver.
func TestARoutedShellRoleCarriesTheApproverThroughDispatch(t *testing.T) {
	f := newFake(t, "ok", Claude)
	machine := `{"roles":{"implementer":{"runner":"claude"}},` + strings.TrimPrefix(routedMachine, "{")
	h := newHarness(t, mustLoad(t, machine, ""), true)
	out, err := h.d.Dispatch(context.Background(), f.shellRequest("implementer"))
	if err != nil {
		t.Fatal(err)
	}
	if out.Handoff || out.Receipt.Route.Ran != Claude {
		t.Fatalf("outcome = %+v", out)
	}
	if argv := f.argv(t, Claude); !slices.Contains(argv, "--permission-prompt-tool="+ApproverTool) {
		t.Errorf("the dispatched launch %q carries no approver", argv)
	}
}

// TestARunnerLaunchsPromptsAreAnsweredByTheGuard is the end-to-end proof: a
// shell launch through the runner hands a fake harness on PATH the flags it
// received, the harness starts the approver its --mcp-config names (the real
// abcd binary, built here) and puts two permission prompts to it, and the
// guard answers a hazard with deny and its reason and a clean command with
// allow.
func TestARunnerLaunchsPromptsAreAnsweredByTheGuard(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the abcd binary")
	}
	abcd := filepath.Join(t.TempDir(), "abcd")
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-o", abcd, "./cmd/abcd")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building abcd: %v\n%s", err, out)
	}
	// The fake's PATH is its own bin directory; the build has already run,
	// so nothing after this needs the toolchain.
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("no git on PATH")
	}
	f := newFake(t, "approve", Claude)
	t.Setenv("HOME", t.TempDir())
	// The approver reads the registry at the run's base, so the working tree
	// is a repository with one commit, and git is on the approver's PATH.
	for _, args := range [][]string{{"init", "-q"}, {"-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "base"}} {
		cmd := exec.Command(git, append([]string{"-C", f.repo}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	head, err := exec.Command(git, "-C", f.repo, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", f.bin+string(os.PathListSeparator)+filepath.Dir(git))
	c := newClaude("")
	c.approver = abcd
	req := f.shellRequest("implementer")
	req.Base = strings.TrimSpace(string(head))
	if _, _, err := c.Run(context.Background(), req); err != nil {
		t.Fatalf("run: %v", err)
	}
	if argv := f.argv(t, Claude); !slices.Contains(argv, "--permission-prompt-tool="+ApproverTool) {
		t.Fatalf("the launch %q names no prompt tool", argv)
	}
	raw, err := os.ReadFile(filepath.Join(f.log, "approver.answers.json"))
	if err != nil {
		t.Fatalf("the harness recorded no answers: %v", err)
	}
	var answers map[string]string
	if err := json.Unmarshal(raw, &answers); err != nil {
		t.Fatal(err)
	}
	type answer struct {
		Behavior     string         `json:"behavior"`
		Message      string         `json:"message"`
		UpdatedInput map[string]any `json:"updatedInput"`
	}
	var hazard, clean answer
	if err := json.Unmarshal([]byte(answers[fakeHazard]), &hazard); err != nil {
		t.Fatalf("the hazard's answer %q: %v", answers[fakeHazard], err)
	}
	e := guard.Defaults().Entries["git-push-force"]
	if hazard.Behavior != "deny" || !strings.Contains(hazard.Message, e.Why) || !strings.Contains(hazard.Message, e.Successor) {
		t.Errorf("the hazard was answered %+v; want deny with the guard's why and successor", hazard)
	}
	if err := json.Unmarshal([]byte(answers["go test ./..."]), &clean); err != nil {
		t.Fatal(err)
	}
	if clean.Behavior != "allow" || clean.UpdatedInput["command"] != "go test ./..." {
		t.Errorf("the clean command was answered %+v; want allow with the command judged", clean)
	}
}
