package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
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
