package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/adapter/repoprompt"
	"github.com/intentdriven/abcd/internal/adapter/repoprompt/mcptest"
)

// writeMachineRouting writes the machine's model-tier routing table.
func writeMachineRouting(t *testing.T, body string) {
	t.Helper()
	dir := abcdhome.Path(os.Getenv("HOME"))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "oracle-routing.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestAStepNamesTheTierItsRoundAsksFor is itd-2609170822093401's criterion 10
// at the front door: `implement step` reads the machine's routing table, so
// the review it hands out is asked for its row's tier, which the result names
// as `tier` and the state records under the round's routes.
func TestAStepNamesTheTierItsRoundAsksFor(t *testing.T) {
	pluginRoot := repoRootForTest(t)
	repo := buildRepo(t)
	repo.Write("AGENTS.md", "# AGENTS.md\n\n- Run make check.\n")
	repo.Commit("conventions")
	t.Setenv("ABCD_PLUGIN_ROOT", pluginRoot)
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	laneAtValidate(t, repo)
	writeMachineRouting(t, `{"schema_version":1,"agents":{"ruthless-reviewer":{"tier":"economy"}}}`)
	var res struct {
		Tier     string `json:"tier"`
		Next     string `json:"next"`
		Awaiting *struct {
			Role string `json:"role"`
		} `json:"awaiting"`
	}
	if err := json.Unmarshal([]byte(mustImplement(t, "implement", "step", "--json")), &res); err != nil {
		t.Fatal(err)
	}
	if res.Awaiting == nil || res.Awaiting.Role != "ruthless-reviewer" || res.Tier != "economy" || !strings.Contains(res.Next, "at model tier economy") {
		t.Fatalf("the step names the tier its round asks the reviewer for: %+v", res)
	}
}

// TestAStepRefusesARoutingTableItCannotRead: the table decides which tier a
// round asks for, so one `implement step` cannot read is refused at the
// routing stage, exit 2, before anything moves.
func TestAStepRefusesARoutingTableItCannotRead(t *testing.T) {
	pluginRoot := repoRootForTest(t)
	repo := buildRepo(t)
	repo.Write("AGENTS.md", "# AGENTS.md\n\n- Run make check.\n")
	repo.Commit("conventions")
	t.Setenv("ABCD_PLUGIN_ROOT", pluginRoot)
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	laneAtValidate(t, repo)
	writeMachineRouting(t, `{"schema_version":1,"agents":{"ruthless-reviewer":{"tier":"cheap"}}}`)
	code, out, errOut := implementCLI(t, "implement", "step")
	if code != 2 || !strings.Contains(out+errOut, "routing") || !strings.Contains(out+errOut, "cheap") {
		t.Fatalf("an unreadable routing table is refused at the routing stage (exit %d):\n%s\n%s", code, out, errOut)
	}
	if !strings.Contains(mustImplement(t, "implement", "status"), "validate") {
		t.Fatal("the refused step moved nothing")
	}
}

// TestAStepPrintsTheProviderDiagnostics: the provider configuration's
// non-fatal reports are said on stderr, as a delegating verb says them.
func TestAStepPrintsTheProviderDiagnostics(t *testing.T) {
	pluginRoot := repoRootForTest(t)
	repo := buildRepo(t)
	repo.Write("AGENTS.md", "# AGENTS.md\n\n- Run make check.\n")
	repo.Commit("conventions")
	t.Setenv("ABCD_PLUGIN_ROOT", pluginRoot)
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	laneAtValidate(t, repo)
	writeMachineConfig(t, `{"oracle":{"roles":{"nobody-agent":"x/y"}}}`)
	code, out, errOut := implementCLI(t, "implement", "step", "--json")
	if code != 0 {
		t.Fatalf("the step exited %d:\n%s\n%s", code, out, errOut)
	}
	if !strings.Contains(errOut, "oracle adapter") || !strings.Contains(errOut, "nobody-agent") {
		t.Fatalf("the provider configuration's diagnostic is said on stderr: %q", errOut)
	}
}

// TestAReviewTheReviewerRouteTakesRecordsNoTierAtTheFrontDoor holds the
// hand-off `implement step` makes from the reviewer route to the routing load
// (iss-2610230015129841): with oracle.review set to rp, a reachable server
// and a routing row that would ask each reviewer for a tier, the ruthless and
// the security review the route takes are each asked for no tier, and the
// run's state records the reviewer route as why neither moves. A step that
// read its routing without the reviewer route would ask both for economy.
func TestAReviewTheReviewerRouteTakesRecordsNoTierAtTheFrontDoor(t *testing.T) {
	pluginRoot := repoRootForTest(t)
	repo := buildRepo(t)
	repo.Write("AGENTS.md", "# AGENTS.md\n\n- Run make check.\n")
	repo.Commit("conventions")
	t.Setenv("ABCD_PLUGIN_ROOT", pluginRoot)
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	laneAtValidate(t, repo)

	srv := &mcptest.Server{Tools: map[string][]string{repoprompt.DefaultTool: {"message", "new_chat"}},
		Answer: func(args map[string]any) any {
			verdict := "SHIP"
			if msg, _ := args["message"].(string); strings.Contains(msg, "name: security-reviewer") {
				verdict = "APPROVE"
			}
			return mcptest.TextResult("### Findings\n\n- none\n\n### Verdict\n\n- **" + verdict + "** — nothing survived.\n")
		}}
	url := mcptest.Start(t, srv)
	repo.Write(".abcd/config.json", `{"oracle":{"review":"rp"}}`)
	writeMachineConfig(t, `{"oracle":{"rp":{"url":"`+url+`"}}}`)
	writeMachineRouting(t, `{"schema_version":1,"agents":{"ruthless-reviewer":{"tier":"economy"},"security-reviewer":{"tier":"economy"}}}`)

	for _, role := range []string{"ruthless-reviewer", "security-reviewer"} {
		var res struct {
			Tier  string `json:"tier"`
			Route *struct {
				Role  string `json:"role"`
				Asked string `json:"asked"`
				Ran   string `json:"ran"`
			} `json:"route"`
		}
		if err := json.Unmarshal([]byte(mustImplement(t, "implement", "step", "--json")), &res); err != nil {
			t.Fatal(err)
		}
		if res.Route == nil || res.Route.Ran != "rp" || res.Tier != "" {
			t.Fatalf("the %s goes over the rp route and is asked for no tier: %+v", role, res)
		}
	}
	if calls := srv.Calls(); len(calls) != 2 {
		t.Fatalf("the ruthless and the security review each go to the route once: %d calls", len(calls))
	}

	states, err := filepath.Glob(filepath.Join(repo.Root(), ".abcd", ".work.local", "run", "run-*", "state.json"))
	if err != nil || len(states) != 1 {
		t.Fatalf("one run's state: %v %v", states, err)
	}
	raw, err := os.ReadFile(states[0])
	if err != nil {
		t.Fatal(err)
	}
	var state struct {
		Lanes []struct {
			Validation []struct {
				Routes []struct {
					Role string `json:"role"`
					Tier string `json:"tier"`
					Note string `json:"note"`
				} `json:"routes"`
			} `json:"validation"`
		} `json:"lanes"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatal(err)
	}
	if len(state.Lanes) != 1 || len(state.Lanes[0].Validation) == 0 {
		t.Fatalf("the lane records its first round: %s", raw)
	}
	recorded := map[string]bool{}
	for _, r := range state.Lanes[0].Validation[0].Routes {
		if r.Role != "ruthless-reviewer" && r.Role != "security-reviewer" {
			continue
		}
		recorded[r.Role] = true
		if r.Tier != "" || !strings.Contains(r.Note, "rp reviewer route") {
			t.Fatalf("the %s the reviewer route takes records that route and no tier: %+v", r.Role, r)
		}
	}
	if !recorded["ruthless-reviewer"] || !recorded["security-reviewer"] {
		t.Fatalf("round 1 records a route for both reviewers: %s", raw)
	}
}
