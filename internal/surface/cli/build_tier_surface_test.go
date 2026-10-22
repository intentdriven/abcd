package cli

import (
	"strings"
	"testing"
)

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
