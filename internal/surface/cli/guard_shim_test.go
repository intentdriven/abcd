package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// preToolUseGuardCommand reads the committed hook manifest and returns the shell
// command the host runs for the guard's PreToolUse entry, plus the matcher it is
// scoped to. The test drives the REAL installed string: a shim that fails open in
// theory and not in the manifest would be worth nothing.
func preToolUseGuardCommand(t *testing.T) (matcher, command string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "hooks", "hooks.json"))
	if err != nil {
		t.Fatalf("cannot read hooks/hooks.json: %v", err)
	}
	var doc struct {
		Hooks map[string][]struct {
			Matcher string `json:"matcher"`
			Hooks   []struct {
				Type    string `json:"type"`
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("hooks/hooks.json is not valid JSON: %v", err)
	}
	for _, group := range doc.Hooks["PreToolUse"] {
		for _, h := range group.Hooks {
			if body := resolveHookCommand(t, h.Command); strings.Contains(body, "guard hook") {
				return group.Matcher, body
			}
		}
	}
	t.Fatal("hooks/hooks.json declares no PreToolUse entry running `guard hook`; the guard plane is not installed")
	return "", ""
}

// fakePluginRoot builds a plugin root holding an `abcd` executable that exits
// with the given status, so the shim can be driven through every outcome the real
// binary can produce — including not existing at all (script == "").
func fakePluginRoot(t *testing.T, script string) string {
	t.Helper()
	root := t.TempDir()
	if script == "" {
		return root
	}
	bin := filepath.Join(root, "abcd")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin[:len(bin)-len("/abcd")]
}

// runShim executes the manifest's command string the way the host does: a POSIX
// shell, with CLAUDE_PLUGIN_ROOT pointing at the plugin root. PATH is
// controlled — system utilities plus one caller-supplied directory — because
// the shim's last resolution rung is a PATH lookup and the developer machine
// running these tests may itself carry a real `abcd` there.
func runShim(t *testing.T, command, pluginRoot, pathDir string) (stderr string, code int) {
	t.Helper()
	return runShimHome(t, command, pluginRoot, pathDir, t.TempDir())
}

// runShimHome is runShim with an explicit HOME. The PATH rung is owned-only —
// it reads `$HOME/.abcd.noindex/path-entry` and runs a PATH binary only when that record
// names it — so a test that means to exercise the rung's ACCEPT branch has to
// own the home the shim reads.
func runShimHome(t *testing.T, command, pluginRoot, pathDir, home string) (stderr string, code int) {
	t.Helper()
	return runShimPayload(t, command, pluginRoot, pathDir, home, shellPayload)
}

// shellPayload and questionPayload are the two tool calls the PreToolUse entry
// is matched for: a shell command and a question to the human.
const (
	shellPayload    = `{"tool_name":"Bash","tool_input":{"command":"ls"}}`
	questionPayload = `{"tool_name":"AskUserQuestion","tool_input":{"questions":[{"question":"Ship it?"}]}}`
)

// runShimPayload is runShimHome with the hook payload the shim reads on stdin.
func runShimPayload(t *testing.T, command, pluginRoot, pathDir, home, payload string) (stderr string, code int) {
	t.Helper()
	_, stderr, code = runShimOutput(t, command, pluginRoot, pathDir, home, payload)
	return stderr, code
}

// runShimOutput is runShimPayload with the shim's stdout kept too: the host
// reads a deny there, so a test of the pass-through needs it.
func runShimOutput(t *testing.T, command, pluginRoot, pathDir, home, payload string) (stdout, stderr string, code int) {
	t.Helper()
	pathEnv := "/usr/bin:/bin"
	if pathDir != "" {
		pathEnv = pathDir + ":" + pathEnv
	}
	cmd := exec.Command("/bin/sh", "-c", command)
	// Hermetic env: the shim's last resort is `command -v abcd`, so a
	// developer machine with abcd installed on PATH (this one ships
	// ~/.local/bin/abcd) would answer the "binary absent" case with the REAL
	// binary — a silent allow, no UNGUARDED notice, and a failure that
	// reproduces only on machines that use the tool. PATH keeps the system
	// directories the shim's POSIX utilities live in (/bin/sh, find, printf);
	// everything user-local is out of reach, and HOME moves off-machine for
	// the same reason. Same lesson as iss-219's setupHermetic.
	for _, e := range os.Environ() {
		switch {
		case strings.HasPrefix(e, "PATH="), strings.HasPrefix(e, "HOME="),
			strings.HasPrefix(e, "CLAUDE_PLUGIN_ROOT="):
		default:
			cmd.Env = append(cmd.Env, e)
		}
	}
	cmd.Env = append(cmd.Env,
		// pathEnv, not a literal: the caller-supplied stub directory is what the
		// PATH-fallback case needs to find, and hardcoding the system pair here
		// would strip it — hermetic against the developer's PATH, not against the
		// test's own fixture.
		"PATH="+pathEnv,
		"HOME="+home,
		"CLAUDE_PLUGIN_ROOT="+pluginRoot)
	cmd.Stdin = strings.NewReader(payload)
	var so, se strings.Builder
	cmd.Stderr = &se
	cmd.Stdout = &so
	err := cmd.Run()
	if err == nil {
		return so.String(), se.String(), 0
	}
	var ee *exec.ExitError
	if ok := asExitError(err, &ee); ok {
		return so.String(), se.String(), ee.ExitCode()
	}
	t.Fatalf("running the shim failed structurally: %v", err)
	return "", "", 0
}

func asExitError(err error, out **exec.ExitError) bool {
	e, ok := err.(*exec.ExitError)
	if ok {
		*out = e
	}
	return ok
}

// TestGuardHookIsInstalledForBashCalls holds the wiring itself: the guard is
// reachable from a live session or it does not exist. The entry must be scoped to
// the shell tool and the host's question tools (itd-2609212130146198), so the
// guard is not asked about every unrelated tool call, and the question tools it
// names are exactly the ones the adapter gates, so the two cannot drift.
func TestGuardHookIsInstalledForBashCalls(t *testing.T) {
	matcher, command := preToolUseGuardCommand(t)
	want := strings.Join(append([]string{"Bash"}, questionTools...), "|")
	if matcher != want {
		t.Errorf("the guard entry must be scoped to the shell tool and the question tools; matcher = %q, want %q", matcher, want)
	}
	if !strings.Contains(command, "CLAUDE_PLUGIN_ROOT") {
		t.Errorf("the entry must invoke the plugin-root binary; command = %q", command)
	}
}

// TestGuardShimFailsOpenLoud is AC 1's other half — the one that cannot be
// satisfied in Go, because the failure being guarded against is the Go binary not
// running at all. A missing or broken binary must never produce the host's
// blocking status, and must never be silent about it.
func TestGuardShimFailsOpenLoud(t *testing.T) {
	_, command := preToolUseGuardCommand(t)
	cases := []struct {
		name   string
		script string
	}{
		{"binary absent", ""},
		{"binary not executable as a program", "exit 127"},
		{"binary crashes", "exit 3"},
		{"binary killed", "kill -TERM $$"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stderr, code := runShim(t, command, fakePluginRoot(t, tc.script), "")
			if code == 2 {
				t.Errorf("a broken guard must never block the session (exit 2); stderr = %q", stderr)
			}
			if !strings.Contains(stderr, "UNGUARDED") {
				t.Errorf("a broken guard must be unmissable; stderr = %q", stderr)
			}
		})
	}
}

// TestGuardShimNamesTheToolItLetThrough (iss-2609260100391018): the entry
// guards two tools, so when the binary is missing or broken the warning names
// the one whose call is going through unchecked. A question is not a shell
// command, and telling the reader "shell commands run UNGUARDED" on a question
// call names the wrong hole.
func TestGuardShimNamesTheToolItLetThrough(t *testing.T) {
	_, command := preToolUseGuardCommand(t)
	for _, script := range []string{"", "exit 3"} {
		for _, tc := range []struct {
			payload, want, not string
		}{
			{shellPayload, "shell commands run UNGUARDED", "AskUserQuestion"},
			{questionPayload, "AskUserQuestion", "shell commands"},
		} {
			stderr, code := runShimPayload(t, command, fakePluginRoot(t, script), "", t.TempDir(), tc.payload)
			if code == 2 {
				t.Errorf("a broken guard must never block (script %q); stderr = %q", script, stderr)
			}
			if !strings.Contains(stderr, "UNGUARDED") || !strings.Contains(stderr, tc.want) {
				t.Errorf("script %q, payload %s: the warning must name %q and say UNGUARDED; stderr = %q", script, tc.payload, tc.want, stderr)
			}
			if strings.Contains(stderr, tc.not) {
				t.Errorf("script %q, payload %s: the warning names %q, which was not called; stderr = %q", script, tc.payload, tc.not, stderr)
			}
		}
	}
}

// TestGuardShimHandsTheBinaryThePayload: the shim reads the payload to name
// the tool in its warnings, and the binary still receives it byte for byte.
func TestGuardShimHandsTheBinaryThePayload(t *testing.T) {
	_, command := preToolUseGuardCommand(t)
	root := fakePluginRoot(t, `cat > "${0%/abcd}/stdin"; exit 0`)
	for _, payload := range []string{shellPayload, questionPayload} {
		if stderr, code := runShimPayload(t, command, root, "", t.TempDir(), payload); code != 0 {
			t.Fatalf("exit %d; stderr = %q", code, stderr)
		}
		got, err := os.ReadFile(filepath.Join(root, "stdin"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != payload {
			t.Errorf("the binary read %q, want the payload %q", got, payload)
		}
	}
}

// TestGuardShimPropagatesRealDecisions is the other side of the same contract: the
// shim must not swallow the decisions the binary DID make. A block stays a block
// and an allow stays an allow.
func TestGuardShimPropagatesRealDecisions(t *testing.T) {
	_, command := preToolUseGuardCommand(t)

	if stderr, code := runShim(t, command, fakePluginRoot(t, `echo "blocked" >&2; exit 2`), ""); code != 2 {
		t.Errorf("a real block must reach the host as exit 2; got %d (stderr %q)", code, stderr)
	}
	if stderr, code := runShim(t, command, fakePluginRoot(t, "exit 0"), ""); code != 0 {
		t.Errorf("a real allow must reach the host as exit 0; got %d (stderr %q)", code, stderr)
	}
	if stderr, _ := runShim(t, command, fakePluginRoot(t, "exit 0"), ""); strings.Contains(stderr, "UNGUARDED") {
		t.Errorf("a working guard must not cry wolf; stderr = %q", stderr)
	}
	// Exit 1 is the adapter's own fail-open-loud status: it RAN, it could not
	// decide, and it allowed. The shim must pass that through untouched — adding
	// its own "FAILED TO RUN" on top would tell the reader a lie about which part
	// broke.
	stderr, code := runShim(t, command, fakePluginRoot(t, `echo "abcd guard: NOT CHECKED" >&2; exit 1`), "")
	if code == 2 {
		t.Errorf("the adapter's fail-open status must never become a block; got %d", code)
	}
	if strings.Contains(stderr, "FAILED TO RUN") {
		t.Errorf("a binary that ran and reported must not be described as failing to run; stderr = %q", stderr)
	}
}

// TestGuardShimPassesADenyThrough: the binary blocks by printing the host's
// deny on stdout and exiting 0, so the shim must hand that stdout to the host
// byte for byte, add nothing to stderr, and pass the exit 0 through. The deny
// the stub prints is the one writeHookDeny writes, so the test reads the real
// encoding and not a copy of it.
func TestGuardShimPassesADenyThrough(t *testing.T) {
	_, command := preToolUseGuardCommand(t)
	var deny strings.Builder
	if err := writeHookDeny(&deny, "Blocked by the abcd guard (test): it's a block."); err != nil {
		t.Fatal(err)
	}
	root := fakePluginRoot(t, "cat >/dev/null; cat \"${0%/abcd}/deny.json\"; exit 0")
	if err := os.WriteFile(filepath.Join(root, "deny.json"), []byte(deny.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, payload := range []string{shellPayload, questionPayload} {
		stdout, stderr, code := runShimOutput(t, command, root, "", t.TempDir(), payload)
		if reason := mustDeny(t, stdout, stderr, code); reason != "Blocked by the abcd guard (test): it's a block." {
			t.Errorf("the deny reached the host changed: reason %q", reason)
		}
		if stdout != deny.String() {
			t.Errorf("the shim must pass the binary's stdout through byte for byte:\n got %q\nwant %q", stdout, deny.String())
		}
	}
}

// TestGuardShimFallsBackToAnOwnedPathBinary pins the resolution ladder's second
// rung: an empty plugin root with THIS MACHINE'S abcd on PATH — the one
// `~/.abcd.noindex/path-entry` records — still guards the session, so a block stays a
// block and no UNGUARDED warning prints. (iss-275: without a controlled PATH the
// binary-absent case above exercised this rung by accident on any machine that
// dogfoods the install, instead of proving the shim fails open.)
func TestGuardShimFallsBackToAnOwnedPathBinary(t *testing.T) {
	_, command := preToolUseGuardCommand(t)
	pathDir := t.TempDir()
	home := t.TempDir()
	stub := filepath.Join(pathDir, "abcd")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\necho \"blocked\" >&2\nexit 2\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeHookPathEntry(t, home, stub)
	stderr, code := runShimHome(t, command, t.TempDir(), pathDir, home)
	if code != 2 {
		t.Errorf("the PATH rung must guard the session; exit = %d (stderr %q)", code, stderr)
	}
	if strings.Contains(stderr, "UNGUARDED") {
		t.Errorf("a session guarded via PATH must not warn UNGUARDED; stderr = %q", stderr)
	}
}

// TestGuardShimRefusesAnUnrecordedPathBinary is GHSA-gx3m-3224-qqcv at the
// guard's own surface: an `abcd` nothing recorded, planted first on PATH and
// exiting 0, must not become the session's guard. The exit 0 it offers is the
// harness's word for "approved", so the only safe reading of an unvouched
// binary is not to run it — UNGUARDED and exit 1, the same degraded path a
// missing binary takes.
func TestGuardShimRefusesAnUnrecordedPathBinary(t *testing.T) {
	_, command := preToolUseGuardCommand(t)
	pathDir := t.TempDir()
	stub := filepath.Join(pathDir, "abcd")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\ncat >/dev/null\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	stderr, code := runShim(t, command, t.TempDir(), pathDir)
	if code != 1 {
		t.Errorf("an unrecorded PATH binary must not decide the command; exit = %d (stderr %q)", code, stderr)
	}
	if !strings.Contains(stderr, "UNGUARDED") {
		t.Errorf("a refused PATH binary leaves the session unguarded and must say so; stderr = %q", stderr)
	}
	if !strings.Contains(stderr, pathRefusalUnowned) {
		t.Errorf("the refusal must name the ownership reason; stderr = %q", stderr)
	}
}
