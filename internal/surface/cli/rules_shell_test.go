package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// The SHELL domain is itd-103's teaching plane (iss-151): the guard registry's
// lessons, generated into a bundled rules domain. These tests drive it through
// both front doors the loader has — the `abcd rules` verb and the prompt-router
// hook — so the plane is wired, not only generated.

// TestRulesRendersTheShellDomain: `abcd rules shell` renders the generated
// domain bare (bundled), carrying a registry entry's lesson, and --json says
// it is bundled.
func TestRulesRendersTheShellDomain(t *testing.T) {
	t.Chdir(t.TempDir())
	out := string(runCLI(t, "rules", "shell"))
	for _, want := range []string{"## SHELL\n", "(git-push-force)", "Instead: Push without a force flag"} {
		if !strings.Contains(out, want) {
			t.Fatalf("abcd rules shell does not carry %q:\n%s", want, out)
		}
	}
	var got map[string]any
	if err := json.Unmarshal(rulesJSON(t, "rules", "SHELL", "--json"), &got); err != nil {
		t.Fatal(err)
	}
	if got["name"] != "SHELL" || got["source"] != "bundled" {
		t.Fatalf("rules SHELL --json = %v; want the bundled SHELL domain", got)
	}
}

// TestHookPromptRouterTeachesShellHazards: a shell-heavy prompt injects the
// SHELL domain before the agent acts, and the same prompt again in the same
// session injects nothing (the per-session dedup).
func TestHookPromptRouterTeachesShellHazards(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	cwd := t.TempDir()
	prompt := "cd into scratch and rm -rf the old output"
	out, errlog := runHook(t, hookInputJSON(t, "teach", cwd, prompt), "hook", "prompt-router")
	if !strings.Contains(out, "## SHELL\n") || !strings.Contains(out, "(rm-rf-after-cd-chain)") {
		t.Fatalf("a shell-heavy prompt did not inject the SHELL lessons:\n%s\nstderr:\n%s", out, errlog)
	}
	again, _ := runHook(t, hookInputJSON(t, "teach", cwd, prompt), "hook", "prompt-router")
	if strings.Contains(again, "## SHELL") {
		t.Fatalf("the unchanged SHELL domain re-injected within one session:\n%s", again)
	}
}
