package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
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

// writeGuardFile lays a repository's own .abcd/guard.json into dir.
func writeGuardFile(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".abcd", "guard.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestRulesTeachesTheRepositorysOwnGuardEntries (ruling CK1): an entry a
// repository adds in its .abcd/guard.json is taught through both front doors,
// marked as the repository's; a guard.json the guard refuses is named on
// stderr from both, and SHELL still teaches the bundled lessons.
func TestRulesTeachesTheRepositorysOwnGuardEntries(t *testing.T) {
	const entry = `{"schema_version":1,"entries":{"deploy-prod":{
		"tier":"blocker",
		"pattern":{"command":"make","subcommand":"deploy"},
		"why":"It deploys to production from a laptop.",
		"successor":"Open a release pull request; CI deploys it."}}}`
	const lesson = "- Refused by the guard (deploy-prod) (repo): `make deploy`. It deploys to production from a laptop. Instead: Open a release pull request; CI deploys it.\n"

	t.Run("taught", func(t *testing.T) {
		t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
		cwd := t.TempDir()
		writeGuardFile(t, cwd, entry)
		out, errlog := runHook(t, hookInputJSON(t, "teach-repo", cwd, "make deploy the docs site"), "hook", "prompt-router")
		if !strings.Contains(out, "## SHELL\n") || !strings.Contains(out, lesson) {
			t.Fatalf("the repository's own hazard was not taught by the hook:\n%s\nstderr:\n%s", out, errlog)
		}
		t.Chdir(cwd)
		so, se, err := runCLISplit(t, "rules", "shell")
		if err != nil || !strings.Contains(so, lesson) || se != "" {
			t.Fatalf("abcd rules shell: err=%v\nstdout:\n%s\nstderr:\n%s", err, so, se)
		}
	})

	t.Run("refused loudly", func(t *testing.T) {
		t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
		cwd := t.TempDir()
		writeGuardFile(t, cwd, strings.Replace(entry, `"why":"It deploys to production from a laptop.",`, "", 1))
		out, errlog := runHook(t, hookInputJSON(t, "teach-repo-bad", cwd, "rm -rf the old build output"), "hook", "prompt-router")
		if !strings.Contains(out, "(rm-rf-root-or-home)") || strings.Contains(out, "deploy-prod") {
			t.Fatalf("a refused guard.json changed the bundled lessons:\n%s", out)
		}
		for _, want := range []string{"SHELL", ".abcd/guard.json", "deploy-prod has no why", "refused and not taught"} {
			if !strings.Contains(errlog, want) {
				t.Fatalf("the hook's stderr does not name the refusal (%q):\n%s", want, errlog)
			}
		}
		t.Chdir(cwd)
		_, se, err := runCLISplit(t, "rules", "shell")
		if err != nil || !strings.Contains(se, "refused and not taught") {
			t.Fatalf("abcd rules shell did not name the refused guard.json: err=%v\nstderr:\n%s", err, se)
		}
	})
}
