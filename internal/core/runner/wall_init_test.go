package runner

// wall_init_test.go holds the proof's reading of the harness's start-up report to a
// real one: testdata/claude-init-2.1.319.jsonl is the init line claude 2.1.319 wrote
// for the proof's own flags, its session id and paths reduced.

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

// wallProofStream is a finished proof after init, in the shapes the same harness
// writes: the shell's home, its variables' names, the guard's refusal and the end.
func wallProofStream(init map[string]any) []byte {
	line, _ := json.Marshal(init)
	return []byte(string(line) + "\n" +
		`{"type":"system","subtype":"hook_response","hook_name":"PreToolUse:Bash","hook_event":"PreToolUse","stdout":"{\"hookSpecificOutput\":{\"permissionDecision\":\"deny\",\"permissionDecisionReason\":\"` + guardRefusalText + ` (git-push-force)\"}}","exit_code":0}` + "\n" +
		`{"type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_01","type":"tool_result","content":"` + wallHomeMarker + wallInitHome + `","is_error":false}]}}` + "\n" +
		`{"type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_02","type":"tool_result","content":"` + wallEnvMarker + `PATH\n` + wallEnvMarker + `HOME","is_error":false}]}}` + "\n" +
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

// TestTheShellStepTheApproverRefusesIsNoProof: a finished launch whose approver refused
// the step that shows the names, with the rest of the proof as it was, is refused.
func TestTheShellStepTheApproverRefusesIsNoProof(t *testing.T) {
	line, _ := json.Marshal(wallInitLine(t))
	stream := string(line) + "\n" +
		`{"type":"system","subtype":"hook_response","hook_event":"PreToolUse","stdout":"` + guardRefusalText + `"}` + "\n" +
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"` + wallHomeMarker + wallInitHome + `","is_error":false}]}}` + "\n" +
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"Refused by the abcd guard (registry-not-in-force)","is_error":true}]}}` + "\n" +
		`{"type":"result","subtype":"success","is_error":false,"result":"done","session_id":"s"}` + "\n"
	var we *WallError
	if err := readWallProof([]byte(stream), []string{"Read", "Bash"}, wallInitHome); !errors.As(err, &we) || we.Wall != WallEnv {
		t.Fatalf("no names shown is the environment's wall: %v", err)
	}
}

// TestWhatTheHarnessAddsToAStagesShellIsAllowed: the names claude 2.1.319 and the shell
// add under a walled launch, read from a real proof's output. Allowed, a stage's shell
// passes on them; a credential among them still would not.
func TestWhatTheHarnessAddsToAStagesShellIsAllowed(t *testing.T) {
	for _, name := range []string{"ABCD_PERSON_HOME", "ABCD_STAGE_HOME", "ABCD_STAGE_PROTECT", "AI_AGENT", "CLAUDECODE",
		"CLAUDE_CODE_CHILD_SESSION", "CLAUDE_CODE_DISABLE_AUTO_MEMORY", "CLAUDE_CODE_DISABLE_CLAUDE_MDS", "CLAUDE_CODE_ENTRYPOINT",
		"CLAUDE_CODE_EXECPATH", "CLAUDE_CODE_SESSION_ATTENDED", "CLAUDE_CODE_SESSION_ID", "CLAUDE_EFFORT", "CLAUDE_ENV_FILE", "CLAUDE_PID",
		"COREPACK_ENABLE_AUTO_PIN", "GIT_EDITOR", "HOME", "LOGNAME", "NoDefaultCurrentDirectoryInExePath", "PATH", "PWD", "SHELL", "SHLVL", "TMPDIR", "_"} {
		if !stageShellAllowed(name) {
			t.Errorf("%s is what the harness adds, and a stage's shell may not hold it", name)
		}
	}
	for _, name := range []string{"ANTHROPIC_API_KEY", "ANTHROPIC_BASE_URL", "GH_TOKEN"} {
		if stageShellAllowed(name) {
			t.Errorf("%s stays outside what a stage's shell holds", name)
		}
	}
}
