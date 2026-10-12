package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/repoprompt"
	"github.com/intentdriven/abcd/internal/adapter/repoprompt/mcptest"
	"github.com/intentdriven/abcd/internal/gittest"
)

// laneAtValidate drives a run started on buildRepo's intent through the CLI,
// as the host does, until its lane awaits its validators.
func laneAtValidate(t *testing.T, repo *gittest.Repo) {
	t.Helper()
	mustImplement(t, "build", "itd-10", "--json")
	for _, want := range []string{"worktree", "brief"} {
		if res := mustStep(t, "implement", "step", "--json"); res.PerformedStage != want {
			t.Fatalf("step = %+v, want %s", res, want)
		}
	}
	await := mustStep(t, "implement", "step", "--json")
	if await.Awaiting == nil || await.Awaiting.Role != "implementer" {
		t.Fatalf("the implement stage awaits an implementer: %+v", await)
	}
	var state struct {
		Lanes []struct{ Worktree, Branch string } `json:"lanes"`
		RunID string                             `json:"run_id"`
	}
	laneDir := filepath.Dir(filepath.Join(repo.Root(), filepath.FromSlash(await.Awaiting.Receipt)))
	st, err := os.ReadFile(filepath.Join(filepath.Dir(laneDir), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(st, &state); err != nil {
		t.Fatal(err)
	}
	wt := state.Lanes[0].Worktree
	if err := os.WriteFile(filepath.Join(wt, "built.txt"), []byte("built\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	repo.Git("-C", wt, "add", "built.txt")
	repo.Git("-C", wt, "commit", "-q", "-m", "build it")
	sha := strings.TrimSpace(repo.Git("-C", wt, "rev-parse", "HEAD"))
	receipt := `{"schema_version":1,"run_id":"` + state.RunID + `","lane":"lane-1","branch":"` + state.Lanes[0].Branch +
		`","commits":["` + sha + `"],"definition_of_done":{"command":"make check","exit_code":0,"output":"dod.log"},"report":"report.md"}`
	for name, body := range map[string]string{"receipt.json": receipt, "dod.log": "ok\n", "report.md": "built it\n"} {
		if err := os.WriteFile(filepath.Join(laneDir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if res := mustStep(t, "implement", "receipt", await.Awaiting.Receipt, "--json"); res.Stage != "validate" {
		t.Fatalf("a verified receipt advances the lane to its validators: %+v", res)
	}
}

// TestAReviewTheStepHandsOutGoesToTheMCPOracleWhenConfigured is criterion 1
// at the front door: with oracle.review set to rp and the server named on the
// machine, `implement step` sends the ruthless review to the MCP oracle
// itself, names the route in its payload, and the verdict is the one the loop
// records, as `implement status` renders it.
func TestAReviewTheStepHandsOutGoesToTheMCPOracleWhenConfigured(t *testing.T) {
	pluginRoot := repoRootForTest(t)
	repo := buildRepo(t)
	repo.Write("AGENTS.md", "# AGENTS.md\n\n- Run make check.\n")
	repo.Commit("conventions")
	t.Setenv("ABCD_PLUGIN_ROOT", pluginRoot)
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	laneAtValidate(t, repo)

	srv := &mcptest.Server{Tools: map[string][]string{repoprompt.DefaultTool: {"message", "new_chat"}},
		Answer: func(map[string]any) any {
			return mcptest.TextResult("### Findings\n\n- none\n\n### Verdict\n\n- **SHIP** — nothing survived.\n")
		}}
	url := mcptest.Start(t, srv)
	repo.Write(".abcd/config.json", `{"oracle":{"review":"rp"}}`)
	writeMachineConfig(t, `{"oracle":{"rp":{"url":"`+url+`"}}}`)

	var res struct {
		Stage    string          `json:"stage"`
		Awaiting json.RawMessage `json:"awaiting"`
		Route    *struct {
			Asked string `json:"asked"`
			Ran   string `json:"ran"`
		} `json:"route"`
	}
	if err := json.Unmarshal([]byte(mustImplement(t, "implement", "step", "--json")), &res); err != nil {
		t.Fatal(err)
	}
	if res.Route == nil || res.Route.Asked != "rp" || res.Route.Ran != "rp" || res.Stage != "validate" || len(res.Awaiting) != 0 {
		t.Fatalf("the step ran the review over the route and names it: %+v", res)
	}
	calls := srv.Calls()
	if len(calls) != 1 {
		t.Fatalf("one review request: %v", calls)
	}
	msg, _ := calls[0].Arguments["message"].(string)
	if !strings.Contains(msg, "name: ruthless-reviewer") || !strings.Contains(msg, "+built") {
		t.Fatalf("the request carries the reviewer's agent definition and the lane's diff:\n%s", msg)
	}
	if status := mustImplement(t, "implement", "status"); !strings.Contains(status, "ruthless-reviewer SHIP") {
		t.Fatalf("the status names the verdict the loop recorded:\n%s", status)
	}
}
