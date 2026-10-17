package cli

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/guard"
	"github.com/intentdriven/abcd/internal/gittest"
)

// rpcReply is one JSON-RPC response the approver wrote.
type rpcReply struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// approverRepo is guardRepo as a repository with one commit, the run's base,
// which it returns with the working tree.
func approverRepo(t *testing.T) (dir, base string) {
	t.Helper()
	repo := gittest.NewRepo(t)
	repo.Write("README", "x\n")
	repo.Commit("base")
	if err := os.MkdirAll(filepath.Join(repo.Root(), ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo.Root())
	return repo.Root(), strings.TrimSpace(repo.Git("rev-parse", "HEAD"))
}

// approverSession runs `guard approve` for the run whose base is base over
// the given request lines and returns its replies by id, failing on any
// stdout line that is not one.
func approverSession(t *testing.T, dir, base string, lines ...string) (map[string]rpcReply, int) {
	t.Helper()
	stdout, stderr, code := runGuard(strings.Join(lines, "\n")+"\n", "guard", "approve", "--dir="+dir, "--base="+base)
	out := map[string]rpcReply{}
	sc := bufio.NewScanner(strings.NewReader(stdout))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var r rpcReply
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil || r.JSONRPC != "2.0" {
			t.Fatalf("stdout line %q is not a JSON-RPC reply (stderr %q)", sc.Text(), stderr)
		}
		out[string(r.ID)] = r
	}
	return out, code
}

// callLine is a tools/call request for the approver's tool.
func callLine(t *testing.T, id int, tool string, input any) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": "tools/call",
		"params": map[string]any{"name": "approve", "arguments": map[string]any{
			"tool_name": tool, "input": input, "tool_use_id": "toolu_1"}}})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// answer decodes the permission answer a tools/call result carries as its one
// text block.
func answer(t *testing.T, r rpcReply) (behavior, message string, updated map[string]any) {
	t.Helper()
	if r.Error != nil {
		t.Fatalf("the call failed: %+v", r.Error)
	}
	var res struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(r.Result, &res); err != nil || len(res.Content) != 1 || res.Content[0].Type != "text" {
		t.Fatalf("result %s is not one text block", r.Result)
	}
	var a struct {
		Behavior     string         `json:"behavior"`
		Message      string         `json:"message"`
		UpdatedInput map[string]any `json:"updatedInput"`
	}
	if err := json.Unmarshal([]byte(res.Content[0].Text), &a); err != nil {
		t.Fatalf("the answer %q is not JSON", res.Content[0].Text)
	}
	return a.Behavior, a.Message, a.UpdatedInput
}

// TestGuardApproveAnswersTheHarnessOverMCP: the approver speaks MCP on stdio
// as the harness's permission prompt tool expects it: it answers the
// handshake and the tool listing, approves a shell command the guard clears
// with the input it judged, refuses a hazard with the guard's reason and any
// other tool's prompt, refuses a method it does not serve, and writes nothing
// for a notification.
func TestGuardApproveAnswersTheHarnessOverMCP(t *testing.T) {
	dir, base := approverRepo(t)
	replies, code := approverSession(t, dir, base,
		`{"jsonrpc":"2.0","id":"server-discover-probe-1","method":"server/discover","params":{}}`,
		`{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"claude-code","version":"2.1.301"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
		callLine(t, 2, "Bash", map[string]any{"command": "go test ./...", "description": "tests"}),
		callLine(t, 3, "Bash", map[string]any{"command": "git push --force origin main"}),
		callLine(t, 4, "Write", map[string]any{"file_path": "/etc/hosts", "content": "x"}),
		`{"jsonrpc":"2.0","id":5,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"other","arguments":{}}}`,
	)
	if code != 0 {
		t.Fatalf("the approver exited %d at the end of its input", code)
	}
	if len(replies) != 8 {
		t.Fatalf("replies = %d, want one per request and none for the notification: %v", len(replies), replies)
	}
	if r := replies[`"server-discover-probe-1"`]; r.Error == nil || r.Error.Code != -32601 {
		t.Errorf("an unserved method: %+v", r)
	}
	var init struct {
		ProtocolVersion string         `json:"protocolVersion"`
		Capabilities    map[string]any `json:"capabilities"`
		ServerInfo      struct {
			Name string `json:"name"`
		} `json:"serverInfo"`
	}
	if err := json.Unmarshal(replies["0"].Result, &init); err != nil || init.ProtocolVersion != "2025-11-25" ||
		init.ServerInfo.Name != "abcd" || init.Capabilities["tools"] == nil {
		t.Errorf("initialize = %s", replies["0"].Result)
	}
	var list struct {
		Tools []struct {
			Name        string         `json:"name"`
			InputSchema map[string]any `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(replies["1"].Result, &list); err != nil || len(list.Tools) != 1 ||
		list.Tools[0].Name != "approve" || list.Tools[0].InputSchema["type"] != "object" {
		t.Errorf("tools/list = %s", replies["1"].Result)
	}
	if b, _, in := answer(t, replies["2"]); b != "allow" || in["command"] != "cd -- '"+dir+"' && /bin/bash --noprofile --norc -p -c 'go test ./...'" || in["description"] != "tests" {
		t.Errorf("a clean command: %s %v", b, in)
	}
	e := guard.Defaults().Entries["git-push-force"]
	if b, msg, _ := answer(t, replies["3"]); b != "deny" || !strings.Contains(msg, e.Why) || !strings.Contains(msg, e.Successor) {
		t.Errorf("a hazard: %s %q", b, msg)
	}
	if b, msg, _ := answer(t, replies["4"]); b != "deny" || !strings.Contains(msg, "Write") {
		t.Errorf("another tool's prompt: %s %q", b, msg)
	}
	if r := replies["5"]; r.Error != nil || string(r.Result) != "{}" {
		t.Errorf("ping = %+v", r)
	}
	if r := replies["6"]; r.Error == nil {
		t.Errorf("a call to a tool it does not serve = %s", r.Result)
	}
}

// TestGuardApproveReadsTheRepositorysRegistry: the approver judges against
// the registry of the working tree it was started for, the repository's own
// entries included.
func TestGuardApproveReadsTheRepositorysRegistry(t *testing.T) {
	dir, base := approverRepo(t)
	if err := os.WriteFile(filepath.Join(dir, ".abcd", "guard.json"), []byte(`{"schema_version":1,"entries":{"make-deploy":{"pattern":{"command":"make","subcommand":"deploy"},"tier":"blocker","why":"deploys from a lane.","successor":"leave deploys to the person."}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	replies, _ := approverSession(t, dir, base, callLine(t, 1, "Bash", map[string]any{"command": "make deploy"}))
	if b, msg, _ := answer(t, replies["1"]); b != "deny" || !strings.Contains(msg, "make-deploy") {
		t.Errorf("a repository hazard: %s %q", b, msg)
	}
}

// TestGuardApproveJudgesAgainstTheRunsBase: the approver judges against the
// registry at the commit the run started on, so a weakening committed during
// the run leaves no registry in force and every command is refused.
func TestGuardApproveJudgesAgainstTheRunsBase(t *testing.T) {
	dir, base := approverRepo(t)
	if err := os.WriteFile(filepath.Join(dir, ".abcd", "guard.json"), []byte(`{"schema_version":1,"disabled":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", ".abcd/guard.json"}, {"-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "-m", "off"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	replies, _ := approverSession(t, dir, base, callLine(t, 1, "Bash", map[string]any{"command": "go test ./..."}))
	if b, msg, _ := answer(t, replies["1"]); b != "deny" || !strings.Contains(msg, "registry-not-in-force") {
		t.Errorf("a command under a registry switched off during the run: %s %q", b, msg)
	}
}

// TestGuardApproveRefusesARelativeDirectory: the directory it judges for is
// named whole, and the run's base as a commit's full object name, or nothing
// is served.
func TestGuardApproveRefusesARelativeDirectory(t *testing.T) {
	dir := guardRepo(t)
	for _, args := range [][]string{{"guard", "approve"}, {"guard", "approve", "--dir=sub"},
		{"guard", "approve", "--dir=" + dir}, {"guard", "approve", "--dir=" + dir, "--base=HEAD"}} {
		stdout, _, code := runGuard("", args...)
		if code != 2 || stdout != "" {
			t.Errorf("%v: exit %d, stdout %q", args, code, stdout)
		}
	}
}

// TestGuardApproveLetsARemovalReachTheRunsOwnDirectories: a removal outside
// the working tree is refused, except in a directory --run-dir names, the
// run's own; a --run-dir that is not a clean absolute path is refused.
func TestGuardApproveLetsARemovalReachTheRunsOwnDirectories(t *testing.T) {
	dir, base := approverRepo(t)
	// Outside the temporary directory, where the test's home may be; the
	// lines are judged, never run.
	run := "/abcd-approve-test-run-dir-never-created"
	other := "/abcd-approve-test-elsewhere-never-created"
	lines := []string{
		callLine(t, 1, "Bash", map[string]any{"command": "rm -f " + filepath.Join(run, "receipt.json")}),
		callLine(t, 2, "Bash", map[string]any{"command": "rm -f " + filepath.Join(other, "receipt.json")}),
	}
	stdout, stderr, code := runGuard(strings.Join(lines, "\n")+"\n", "guard", "approve", "--dir="+dir, "--base="+base, "--run-dir="+run)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	replies := map[string]rpcReply{}
	for _, l := range strings.Split(strings.TrimSpace(stdout), "\n") {
		var r rpcReply
		if err := json.Unmarshal([]byte(l), &r); err != nil {
			t.Fatalf("stdout line %q: %v", l, err)
		}
		replies[string(r.ID)] = r
	}
	if b, msg, _ := answer(t, replies["1"]); b != "allow" {
		t.Errorf("a removal in the run's own directory: %s %q", b, msg)
	}
	if b, msg, _ := answer(t, replies["2"]); b != "deny" || !strings.Contains(msg, "removal-outside-tree") {
		t.Errorf("a removal outside the tree and the run's directories: %s %q", b, msg)
	}
	for _, bad := range []string{"--run-dir=rel", "--run-dir=" + run + "/../x"} {
		if stdout, _, code := runGuard("", "guard", "approve", "--dir="+dir, "--base="+base, bad); code != 2 || stdout != "" {
			t.Errorf("%s: exit %d, stdout %q", bad, code, stdout)
		}
	}
}
