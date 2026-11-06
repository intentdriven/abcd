package runner

// wall_init_test.go holds the proof's reading of the harness's start-up report to a
// real one: testdata/claude-init-2.1.319.jsonl is the init line claude 2.1.319 wrote
// for the proof's own flags, its session id and paths reduced.

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
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/shellquote"
)

const wallInitHome = "/proof/stage/home"

// wallInitLine is the captured init event, decoded for a row to change.
func wallInitLine(t *testing.T) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "claude-init-2.1.319.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var init map[string]any
	if err := json.Unmarshal(raw, &init); err != nil {
		t.Fatal(err)
	}
	return init
}

// wallApprovedFlags is what the proof's env step reads for $- under the approver's
// wrapper, as a run of the real harness and approver against a stub API showed it.
const wallApprovedFlags = "hpBc"

// wallEnvStep is the tool result of the proof's env step, as the harness writes it.
func wallEnvStep(flags string, names ...string) string {
	var content []string
	if flags != "" {
		content = append(content, wallShellMarker+flags)
	}
	for _, n := range names {
		content = append(content, wallEnvMarker+n)
	}
	raw, _ := json.Marshal(map[string]any{"type": "user", "message": map[string]any{"role": "user", "content": []map[string]any{
		{"tool_use_id": "toolu_02", "type": "tool_result", "content": strings.Join(content, "\n"), "is_error": false}}}})
	return string(raw) + "\n"
}

// wallProofStream is a finished proof after init, in the shapes the same harness
// writes: the shell's home, its variables' names, the guard's refusal and the end.
func wallProofStream(init map[string]any) []byte {
	line, _ := json.Marshal(init)
	return []byte(string(line) + "\n" +
		`{"type":"system","subtype":"hook_response","hook_name":"PreToolUse:Bash","hook_event":"PreToolUse","stdout":"{\"hookSpecificOutput\":{\"permissionDecision\":\"deny\",\"permissionDecisionReason\":\"` + guardRefusalText + ` (git-push-force)\"}}","exit_code":0}` + "\n" +
		`{"type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_01","type":"tool_result","content":"` + wallHomeMarker + wallInitHome + `","is_error":false}]}}` + "\n" +
		wallEnvStep(wallApprovedFlags, "PATH", "HOME") +
		`{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"s"}` + "\n")
}

// TestTheProofReadsARealStartUpReport: the walls hold on what the harness really says,
// each row below changes one thing and is refused by name, and the refusal says the
// harness's version.
func TestTheProofReadsARealStartUpReport(t *testing.T) {
	named := []string{"Read", "Bash"}
	servers := func(s ...map[string]any) []any {
		out := []any{}
		for _, one := range s {
			out = append(out, one)
		}
		return out
	}
	plugin := func(name, path, source string) map[string]any {
		return map[string]any{"name": name, "path": path, "source": source}
	}
	for _, c := range []struct {
		name   string
		change func(init map[string]any)
		wall   string
		says   string
	}{
		{name: "as captured: connected, four built-ins, no server tool listed", change: func(map[string]any) {}},
		{name: "the approver listed among the tools", change: func(init map[string]any) {
			init["tools"] = []any{"Bash", "Read", ApproverTool}
		}},
		{name: "a built-in no release named before", change: func(init map[string]any) {
			init["plugins"] = append(init["plugins"].([]any), plugin("cc-plugin-new-in-a-later-release", "builtin", "cc-plugin-new-in-a-later-release@builtin"))
		}},
		{name: "no plugin at all", change: func(init map[string]any) { init["plugins"] = []any{} }},
		{name: "the server failed", change: func(init map[string]any) {
			init["mcp_servers"] = servers(map[string]any{"name": "abcd", "status": "failed", "source": "dynamic"})
		}, wall: WallServers, says: "failed"},
		{name: "the server pending", change: func(init map[string]any) {
			init["mcp_servers"] = servers(map[string]any{"name": "abcd", "status": "pending", "source": "dynamic"})
		}, wall: WallServers, says: "pending"},
		{name: "the server reports no status", change: func(init map[string]any) {
			init["mcp_servers"] = servers(map[string]any{"name": "abcd", "source": "dynamic"})
		}, wall: WallServers, says: "no status"},
		{name: "no server", change: func(init map[string]any) { init["mcp_servers"] = servers() }, wall: WallServers, says: "no server"},
		{name: "the list of servers absent", change: func(init map[string]any) { delete(init, "mcp_servers") }, wall: WallServers, says: "no server"},
		{name: "a second server", change: func(init map[string]any) {
			init["mcp_servers"] = servers(map[string]any{"name": "abcd", "status": "connected"}, map[string]any{"name": "persons-own", "status": "connected"})
		}, wall: WallServers, says: "persons-own"},
		{name: "the approver's name twice", change: func(init map[string]any) {
			init["mcp_servers"] = servers(map[string]any{"name": "abcd", "status": "connected"}, map[string]any{"name": "abcd", "status": "connected"})
		}, wall: WallServers, says: "more than one"},
		{name: "a plugin from anywhere else", change: func(init map[string]any) {
			init["plugins"] = append(init["plugins"].([]any), plugin("persons-plugin", "/plugins/persons-plugin", "persons-plugin@a-marketplace"))
		}, wall: WallPlugins, says: "persons-plugin"},
		{name: "a plugin from a marketplace named builtin", change: func(init map[string]any) {
			init["plugins"] = append(init["plugins"].([]any), plugin("lookalike", "/plugins/lookalike", "lookalike@builtin"))
		}, wall: WallPlugins, says: "lookalike"},
		{name: "a plugin at a path named builtin", change: func(init map[string]any) {
			init["plugins"] = append(init["plugins"].([]any), plugin("lookalike", "builtin", "lookalike@a-marketplace"))
		}, wall: WallPlugins, says: "lookalike"},
		{name: "a plugin that says nothing of where it came from", change: func(init map[string]any) {
			init["plugins"] = append(init["plugins"].([]any), map[string]any{"name": "bare-plugin"})
		}, wall: WallPlugins, says: "bare-plugin"},
		{name: "another server tool", change: func(init map[string]any) {
			init["tools"] = []any{"Bash", "Read", "mcp__abcd__other"}
		}, wall: WallTools, says: "mcp__abcd__other"},
		{name: "another server's tool", change: func(init map[string]any) {
			init["tools"] = []any{"Bash", "Read", ApproverTool, "mcp__persons-own__read"}
		}, wall: WallTools, says: "mcp__persons-own__read"},
		{name: "a tool it did not name", change: func(init map[string]any) {
			init["tools"] = []any{"Bash", "Read", "WebFetch"}
		}, wall: WallTools, says: "WebFetch"},
		{name: "a named tool missing", change: func(init map[string]any) {
			init["tools"] = []any{"Read"}
		}, wall: WallTools, says: "Bash"},
		{name: "the list of tools absent", change: func(init map[string]any) { delete(init, "tools") }, wall: WallTools},
		{name: "a key in use", change: func(init map[string]any) { init["apiKeySource"] = "ANTHROPIC_API_KEY" }, wall: WallKey},
	} {
		t.Run(c.name, func(t *testing.T) {
			init := wallInitLine(t)
			c.change(init)
			err := readWallProof(wallProofStream(init), named, wallInitHome)
			if c.wall == "" {
				if err != nil {
					t.Fatalf("every wall holds: %v", err)
				}
				return
			}
			var we *WallError
			if !errors.As(err, &we) || we.Wall != c.wall {
				t.Fatalf("the refusal names the wall %q: %v", c.wall, err)
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Errorf("the refusal says %q: %v", c.says, err)
			}
			if !strings.Contains(err.Error(), "2.1.319") {
				t.Errorf("the refusal says the harness's version: %v", err)
			}
		})
	}
}

// TestARefusalReadsTheHarnessVersionOnlyWhenItIsOne: a report with no version says none,
// and a value that is no version never reaches the refusal.
func TestARefusalReadsTheHarnessVersionOnlyWhenItIsOne(t *testing.T) {
	for name, version := range map[string]any{"absent": nil, "not a version": "2.1 \x1b[2Jrun this", "a number": 2.1} {
		t.Run(name, func(t *testing.T) {
			init := wallInitLine(t)
			delete(init, "claude_code_version")
			if version != nil {
				init["claude_code_version"] = version
			}
			init["tools"] = []any{"Bash", "Read", "WebFetch"}
			err := readWallProof(wallProofStream(init), []string{"Read", "Bash"}, wallInitHome)
			if err == nil {
				t.Fatal("the extra tool is refused")
			}
			if strings.Contains(err.Error(), "harness version") || strings.Contains(err.Error(), "run this") {
				t.Fatalf("no version to say: %v", err)
			}
		})
	}
}

// TestTheProofNeedsTheShellStepAsTheApproverHandsItBack: the env step reads $-, and only
// the approver's wrapper runs it with p, whichever shell would run it otherwise; a step
// refused, or showing no flags or others, refuses, and says the harness's version.
func TestTheProofNeedsTheShellStepAsTheApproverHandsItBack(t *testing.T) {
	line, _ := json.Marshal(wallInitLine(t))
	const home = `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"` + wallHomeMarker + wallInitHome + `","is_error":false}]}}` + "\n"
	const guarded = `{"type":"system","subtype":"hook_response","hook_event":"PreToolUse","stdout":"` + guardRefusalText + `"}` + "\n"
	const end = `{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"s"}` + "\n"
	refused := `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"Refused by the abcd guard (registry-not-in-force)","is_error":true}]}}` + "\n"
	closed := `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"<tool_use_error>Error calling tool (Bash): Connection closed</tool_use_error>","is_error":true}]}}` + "\n"
	quoted := `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"` + strings.ReplaceAll(wallEnvCommand, `"`, `\"`) + `","is_error":true}]}}` + "\n"
	for _, c := range []struct {
		name, step string
		wall       string
	}{
		{"the approver's wrapper, as measured", wallEnvStep("hpBc", "PATH", "HOME"), ""},
		{"the approver's wrapper, with other flags beside p", wallEnvStep("himpBHs", "PATH"), ""},
		{"bash without the wrapper", wallEnvStep("hBc", "PATH", "HOME"), WallApprover},
		{"zsh without the wrapper", wallEnvStep("569X", "PATH", "HOME"), WallApprover},
		{"no flags in the marker", wallEnvStep("", "PATH", "HOME"), WallApprover},
		{"the marker with nothing after it", `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"` + wallShellMarker + `\n` + wallEnvMarker + `PATH"}]}}` + "\n", WallApprover},
		{"the step refused", refused, WallApprover},
		{"the approver gone", closed, WallApprover},
		{"the step's own text and no output", quoted, WallApprover},
		{"the flags with no names", wallEnvStep("hpBc"), WallEnv},
		{"a name it may not hold", wallEnvStep("hpBc", "PATH", "CLAUDE_CODE_MESSAGING_TOKEN"), WallEnv},
		{"a socket it may not hold", wallEnvStep("hpBc", "PATH", "CLAUDE_CODE_MESSAGING_SOCKET"), WallEnv},
		{"the walled directory by name", wallEnvStep("hpBc", "PATH", "ABCD_STAGE_PROTECT"), WallEnv},
	} {
		t.Run(c.name, func(t *testing.T) {
			stream := string(line) + "\n" + guarded + home + c.step + end
			err := readWallProof([]byte(stream), []string{"Read", "Bash"}, wallInitHome)
			if c.wall == "" {
				if err != nil {
					t.Fatalf("every wall holds: %v", err)
				}
				return
			}
			var we *WallError
			if !errors.As(err, &we) || we.Wall != c.wall {
				t.Fatalf("the refusal names the wall %q: %v", c.wall, err)
			}
			if !strings.Contains(err.Error(), "2.1.319") {
				t.Errorf("the refusal says the harness's version: %v", err)
			}
		})
	}
}

// TestAHarnessNameThatReadsAsASecretIsToldApart: a name stopped for its words says so,
// with the name and the harness's version; one merely off the list says that.
func TestAHarnessNameThatReadsAsASecretIsToldApart(t *testing.T) {
	line, _ := json.Marshal(wallInitLine(t))
	const home = `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"` + wallHomeMarker + wallInitHome + `","is_error":false}]}}` + "\n"
	const end = `{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"s"}` + "\n"
	for name, want := range map[string]string{
		"CLAUDE_BG_RENDEZVOUS_SOCK": "reads as a credential or a socket",
		"CLAUDE_CODE_OAUTH_TOKEN":   "reads as a credential or a socket",
		"ANTHROPIC_API_KEY":         "reads as a credential or a socket",
		"AN_ORDINARY_NAME":          "not on the list it carries",
	} {
		err := readWallProof([]byte(string(line)+"\n"+home+wallEnvStep("hpBc", "PATH", name)+end), []string{"Read", "Bash"}, wallInitHome)
		if err == nil || !strings.Contains(err.Error(), name) || !strings.Contains(err.Error(), want) || !strings.Contains(err.Error(), "2.1.319") {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestTheEnvStepsFlagsShowTheApproversWrapper runs the proof's real env command: as the
// guard hands it back, it shows p; as written, in a shell of its own, it does not.
func TestTheEnvStepsFlagsShowTheApproversWrapper(t *testing.T) {
	gittest.Env(t)
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	base, err := wallProofRepository(dir)
	if err != nil {
		t.Fatal(err)
	}
	input, _ := json.Marshal(map[string]string{"command": wallEnvCommand, "description": "the proof's env step"})
	approval, updated := guard.ApprovePrompt(guard.LoadRepoAt(dir, base), guard.NewFiles(dir), "Bash", input)
	if !approval.Allow {
		t.Fatalf("the guard clears the proof's env step: %s", approval.Reason)
	}
	var handed struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(updated, &handed); err != nil {
		t.Fatal(err)
	}
	if handed.Command == wallEnvCommand {
		t.Fatalf("the guard hands the command back as written: %q", handed.Command)
	}
	run := func(shell, command string) string {
		out, err := exec.Command(shell, "-c", command).Output()
		if err != nil {
			t.Fatalf("%s: %v", shell, err)
		}
		return string(out)
	}
	// dash's $- is empty under -c, so the step as written may show no flags at all;
	// what it may never show is the flag the approver's shell carries.
	for _, shell := range []string{"/bin/sh", "/bin/dash", "/bin/bash", "/bin/zsh"} {
		if _, err := os.Stat(shell); err != nil {
			continue
		}
		if flags, shown := wallShellFlags([]byte(run(shell, handed.Command))); !shown || !strings.Contains(flags, "p") {
			t.Errorf("as handed back, under %s, the step shows %q", shell, flags)
		}
		out := run(shell, wallEnvCommand)
		if !strings.Contains(out, wallShellMarker) {
			t.Errorf("as written, under %s, the step did not run: %q", shell, out)
		}
		if flags, _ := wallShellFlags([]byte(out)); strings.Contains(flags, "p") {
			t.Errorf("as written, under %s, the step shows %q", shell, flags)
		}
	}
}

// TestWhatTheHarnessAddsToAStagesShellIsAllowed: the names claude 2.1.319 and the shell
// add under a walled launch, read from a run of the real harness and approver against
// a stub API. Allowed, a stage's shell
// passes on them; a credential, a key or a socket among the harness's names does not,
// whichever release names it.
func TestWhatTheHarnessAddsToAStagesShellIsAllowed(t *testing.T) {
	for _, name := range []string{"ABCD_PERSON_HOME", "ABCD_STAGE_HOME", "AI_AGENT", "CLAUDECODE",
		"CLAUDE_CODE_CHILD_SESSION", "CLAUDE_CODE_DISABLE_AUTO_MEMORY", "CLAUDE_CODE_DISABLE_CLAUDE_MDS", "CLAUDE_CODE_ENTRYPOINT",
		"CLAUDE_CODE_EXECPATH", "CLAUDE_CODE_SESSION_ATTENDED", "CLAUDE_CODE_SESSION_ID", "CLAUDE_EFFORT", "CLAUDE_ENV_FILE", "CLAUDE_PID",
		"COREPACK_ENABLE_AUTO_PIN", "DISABLE_UPDATES", "GIT_EDITOR", "HOME", "LOGNAME", "NoDefaultCurrentDirectoryInExePath", "PATH", "PWD", "SHELL", "SHLVL", "TMPDIR", "_",
		"CLAUDE_CODE_A_NAME_NO_RELEASE_HAS_YET", "CLAUDE_CODE_MAX_OUTPUT_TOKENS"} {
		if !stageShellAllowed(name) {
			t.Errorf("%s is what the harness adds, and a stage's shell may not hold it", name)
		}
	}
	for _, name := range []string{"ANTHROPIC_API_KEY", "ANTHROPIC_BASE_URL", "GH_TOKEN",
		"CLAUDE_CODE_MESSAGING_TOKEN", "CLAUDE_CODE_MESSAGING_SOCKET", "CLAUDE_CODE_OAUTH_TOKEN", "CLAUDE_CODE_HOST_CREDS_FILE",
		"CLAUDE_BG_SOCKET_TOKENS_PATH", "CLAUDE_CODE_CCR_WORKER_SOCKET", "CLAUDE_CODE_HOST_AUTH_ENV_VAR", "CLAUDE_A_SECRET", "CLAUDE_AN_API_KEY",
		"CLAUDE_BG_RENDEZVOUS_SOCK", "CLAUDE_A_JWT", "CLAUDE_A_BEARER", "CLAUDE_CODE_PROXY_AUTHENTICATE", "CLAUDE_CODE_APITOKEN", "CLAUDE_CODE_WEBSOCKET",
		"CLAUDE_CODE_CLIENT_KEY_PASSPHRASE", "CLAUDE_AN_ACCESSTOKEN", "CLAUDE_ARTIFACT_HOST_GRANT",
		"ABCD_STAGE_PROTECT", StageHarnessEnv} {
		if stageShellAllowed(name) {
			t.Errorf("%s stays outside what a stage's shell holds", name)
		}
	}
}

// TestAStageRunsTheHarnessTheStartResolved: a link moved after the start changes nothing
// a walled launch runs, the login check, the proof and each stage alike, and with the
// file gone the check stops the stage before anything starts.
func TestAStageRunsTheHarnessTheStartResolved(t *testing.T) {
	f := newFake(t, "wall-ok", Claude)
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	versions := t.TempDir()
	which := filepath.Join(f.log, "which")
	version := func(name string) string {
		path := filepath.Join(versions, name)
		script := "#!/bin/sh\nprintf '%s\\n' " + name + " >> \"$" + fakeLogEnv + "/which\"\nexec " + shellquote.Single(self) + " \"$@\"\n"
		if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
		return path
	}
	first, second := version("first"), version("second")
	link := filepath.Join(f.bin, Claude)
	repoint := func(to string) {
		t.Helper()
		if err := os.Remove(link); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(to, link); err != nil {
			t.Fatal(err)
		}
	}
	repoint(first)
	p := wallProfile(t)
	c := newClaude("")
	c.profile = &p
	c.approver = wallBin
	if _, err := WriteWallSettings(p, c.approver); err != nil {
		t.Fatal(err)
	}
	ran := func() []string {
		t.Helper()
		raw, err := os.ReadFile(which)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			t.Fatal(err)
		}
		return strings.Fields(string(raw))
	}
	repoint(second)
	if err := CheckWallSettings(p, c.approver); err != nil {
		t.Fatalf("the file the start resolved is as it was: %v", err)
	}
	if err := CheckLogin(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if err := ProveWall(context.Background(), p, c.approver); err != nil {
		t.Fatal(err)
	}
	t.Setenv(fakeModeEnv, "ok")
	if _, _, err := c.Run(context.Background(), shellRequest(f)); err != nil {
		t.Fatal(err)
	}
	proved, err := filepath.EvalSymlinks(first)
	if err != nil {
		t.Fatal(err)
	}
	if got := argWith(f.argv(t, Claude), "--mcp-config="); !strings.Contains(got, "--harness="+proved) {
		t.Errorf("the approver is told the harness the run proved: %s", got)
	}
	raw, err := os.ReadFile(filepath.Join(f.log, "claude.env"))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(strings.Split(string(raw), "\n"), StageHarnessEnv+"="+proved) {
		t.Errorf("the hooks are told the harness the run proved")
	}
	// Which harness answers is what this holds, not what it answers.
	_, _ = c.probeLimit(context.Background(), p.Ledger, "probe-model")
	got := ran()
	if len(got) != 4 || slices.Contains(got, "second") {
		t.Fatalf("each launch runs the file the start resolved: %q", got)
	}
	if err := os.Remove(first); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(which); err != nil {
		t.Fatal(err)
	}
	var we *WallError
	if err := CheckWallSettings(p, c.approver); !errors.As(err, &we) || we.Wall != WallSettingsFile || !strings.Contains(err.Error(), "changed while the drive ran") {
		t.Fatalf("with the file gone the check stops the drive: %v", err)
	}
	if _, _, err := c.Run(context.Background(), shellRequest(f)); err == nil || !strings.Contains(err.Error(), "changed while the drive ran") {
		t.Fatalf("with the file gone no stage starts: %v", err)
	}
	if got := ran(); len(got) != 0 {
		t.Fatalf("nothing started: %q", got)
	}
}
