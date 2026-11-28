package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAhoyTextPrintsTheStepOfAGapAbcdCannotResolve (iss-2611252138558553): a
// gap that is not required and that no `ahoy install` closes — a hooks manifest
// from before the sub-agent start was wired — is named with its step in the
// text renders too, not only under --json. Bare ahoy prints the step on the
// line under the gap's title, and doctor names the gap and its step.
func TestAhoyTextPrintsTheStepOfAGapAbcdCannotResolve(t *testing.T) {
	hermeticRepo(t)
	runCLI(t, "ahoy", "install", "--yes", "--adopt",
		"--visibility", "private", "--docs-target", "agents_md",
		"--oracle-backend", "host-delegated", "--scan-deep", "false", "--json")
	manifest := filepath.Join(os.Getenv("ABCD_PLUGIN_ROOT"), "hooks", "hooks.json")
	if err := os.WriteFile(manifest, []byte(strings.ReplaceAll(validHooksJSON, "SubagentStart", "SubagentStop")), 0o644); err != nil {
		t.Fatal(err)
	}
	const title, step = "hooks/hooks.json wires no sub-agent start", "Update the plugin."

	// stepUnder reports whether the line after the one whose text is title
	// carries step and nothing else.
	stepUnder := func(out string) bool {
		lines := strings.Split(out, "\n")
		for i, l := range lines {
			if strings.HasSuffix(strings.TrimSpace(l), title) && i+1 < len(lines) {
				return strings.TrimSpace(lines[i+1]) == step
			}
		}
		return false
	}

	bare := string(runCLI(t, "ahoy"))
	if !strings.Contains(bare, title) {
		t.Fatalf("bare ahoy does not name the gap:\n%s", bare)
	}
	if !stepUnder(bare) {
		t.Errorf("bare ahoy prints no step under the gap's title:\n%s", bare)
	}

	doctor := string(runCLI(t, "ahoy", "doctor"))
	if !strings.Contains(doctor, title) {
		t.Fatalf("ahoy doctor does not name the gap:\n%s", doctor)
	}
	if !stepUnder(doctor) {
		t.Errorf("ahoy doctor prints no step under the gap's title:\n%s", doctor)
	}
}
