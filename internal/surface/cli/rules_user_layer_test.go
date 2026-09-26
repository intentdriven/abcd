package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// userScope points HOME at a fresh directory holding body as ~/.abcd/rules.json
// and chdirs into a bare repo directory with no rules file of its own, so every
// verb reads the fixture's user layer — never the developer's (spc-23).
func userScope(t *testing.T, body string) (home, repo string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".abcd", "rules.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	repo = t.TempDir()
	t.Chdir(repo)
	return home, repo
}

const userLayerFixture = `{"schema_version":1,"domains":{
	"PII":{"rules":["house pii rule"]},
	"HOUSE":{"recall":["widget"],"rules":["widgets are named in the singular"]}}}`

// AC6 on the CLI: `abcd rules` and `abcd rules --json` say a domain came from
// the user scope, beside the bundled and repo labels they already carry.
func TestRulesVerbLabelsTheUserLayer(t *testing.T) {
	_, repo := userScope(t, userLayerFixture)
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "rules.json"),
		[]byte(`{"schema_version":1,"domains":{"ROADMAP":{"rules":["repo roadmap"]}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"PII": "user", "HOUSE": "user", "ROADMAP": "repo", "COMMITTING": "bundled"} {
		var got map[string]any
		out := runCLI(t, "rules", name, "--json")
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("rules %s --json not JSON: %v\n%s", name, err, out)
		}
		if got["source"] != want {
			t.Errorf("rules %s --json source = %v, want %q", name, got["source"], want)
		}
	}
	out := string(runCLI(t, "rules"))
	for _, want := range []string{"## PII (user override)\n- house pii rule\n", "## HOUSE (user override)\n", "## ROADMAP (repo override)\n", "## COMMITTING\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("bare rules render lacks %q:\n%s", want, out)
		}
	}
}

// A user-scope kill switch is reported against the file that set it.
func TestRulesVerbNamesTheUserKillSwitch(t *testing.T) {
	userScope(t, `{"schema_version":1,"disabled":true}`)
	out := string(runCLI(t, "rules"))
	if !strings.Contains(out, "disabled") || !strings.Contains(out, "~/.abcd/rules.json") {
		t.Fatalf("a user-scope kill switch must be reported against ~/.abcd/rules.json:\n%s", out)
	}
}

// A malformed user layer fails the verb loudly, naming the file.
func TestRulesVerbRefusesABrokenUserLayer(t *testing.T) {
	userScope(t, `{ broken`)
	out, err := runCLIStdinErr(t, "", "rules")
	if err == nil {
		t.Fatalf("a malformed ~/.abcd/rules.json must fail the verb, got:\n%s", out)
	}
	if !strings.Contains(err.Error(), "~/.abcd/rules.json") {
		t.Fatalf("the refusal must name the user file: %v", err)
	}
}

// AC4 through the hook: a user custom domain injects in a repo that never
// declares it, and both channels name its layer.
func TestHookPromptRouterInjectsTheUserLayer(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	_, repo := userScope(t, userLayerFixture)
	out, errlog := runHook(t, hookInputJSON(t, "user-layer", repo, "rename the widget"), "hook", "prompt-router")
	if !strings.Contains(out, "## HOUSE (user override)\n- widgets are named in the singular\n") {
		t.Fatalf("the injected block lacks the user domain:\n%s", out)
	}
	if !strings.Contains(errlog, "HOUSE (user override)") {
		t.Fatalf("the diagnostic does not name the user layer:\n%s", errlog)
	}
}

// AC5 through the hook: a broken user layer injects NOTHING and says why on
// stderr — never a partial set built from the layers that loaded.
func TestHookPromptRouterRefusesABrokenUserLayer(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	_, repo := userScope(t, `{ broken`)
	out, errlog := runHook(t, hookInputJSON(t, "user-broken", repo, "commit and push"), "hook", "prompt-router")
	if out != "" {
		t.Fatalf("a broken user layer must inject nothing, got:\n%s", out)
	}
	if !strings.Contains(errlog, "~/.abcd/rules.json") || !strings.Contains(errlog, "injecting nothing") {
		t.Fatalf("the refusal must be loud and name the user file:\n%s", errlog)
	}
}
