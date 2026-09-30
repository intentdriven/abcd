package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// routerEnvelope is what a snapshotting client reads from
// `abcd hook prompt-router --json` (ruling J15, iss-2608261550580260).
type routerEnvelope struct {
	Text     string    `json:"text"`
	Injected []string  `json:"injected"`
	Active   *[]string `json:"active"`
	Error    string    `json:"error"`
}

func writeRepoRules(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".abcd", "rules.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

const widgetsRules = `{"schema_version": 1, "domains": {"WIDGETS": {"recall": ["widget"], "rules": ["Widgets are counted twice."]}}}`

func routerJSON(t *testing.T, session, dir, prompt string) routerEnvelope {
	t.Helper()
	out, _ := runHook(t, hookInputJSON(t, session, dir, prompt), "hook", "prompt-router", "--json")
	var env routerEnvelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("prompt-router --json is not one JSON document: %v\n%s", err, out)
	}
	return env
}

// TestHookPromptRouterJSONNamesStoppedDomain is the client's view of ruling
// J15: it saw WIDGETS on turn 1, and turn 2's output alone tells it WIDGETS
// stopped — deleted, renamed or made dormant — because the set it is handed
// every turn no longer names it.
func TestHookPromptRouterJSONNamesStoppedDomain(t *testing.T) {
	for how, next := range map[string]string{
		"deleted": `{"schema_version": 1, "domains": {}}`,
		"renamed": `{"schema_version": 1, "domains": {"GADGETS": {"recall": ["widget"], "rules": ["Widgets are counted twice."]}}}`,
		"dormant": `{"schema_version": 1, "domains": {"WIDGETS": {"state": "dormant", "recall": ["widget"], "rules": ["Widgets are counted twice."]}}}`,
	} {
		t.Run(how, func(t *testing.T) {
			t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
			dir := t.TempDir()
			writeRepoRules(t, dir, widgetsRules)
			first := routerJSON(t, "j15-"+how, dir, "count the widget")
			if !strings.Contains(first.Text, "## WIDGETS") || !slices.Equal(first.Injected, []string{"WIDGETS"}) {
				t.Fatalf("turn 1 did not inject WIDGETS: %+v", first)
			}
			if first.Active == nil || !slices.Contains(*first.Active, "WIDGETS") {
				t.Fatalf("turn 1 active set does not name WIDGETS: %+v", first)
			}

			writeRepoRules(t, dir, next)
			second := routerJSON(t, "j15-"+how, dir, "paint a landscape")
			if second.Active == nil {
				t.Fatalf("turn 2 carries no active set: %+v", second)
			}
			if slices.Contains(*second.Active, "WIDGETS") {
				t.Fatalf("turn 2 active set %v still names WIDGETS after it was %s", *second.Active, how)
			}
			if !slices.Contains(*second.Active, "COMMITTING") {
				t.Fatalf("turn 2 active set %v is not the full set (a bundled domain is missing)", *second.Active)
			}
		})
	}
}

// TestHookPromptRouterJSONKillSwitchIsAnEmptySet: nothing in force is an
// empty list, present and parseable, never an absent field.
func TestHookPromptRouterJSONKillSwitchIsAnEmptySet(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	dir := t.TempDir()
	writeRepoRules(t, dir, `{"schema_version": 1, "disabled": true, "domains": {}}`)
	out, _ := runHook(t, hookInputJSON(t, "j15-kill", dir, "commit and push"), "hook", "prompt-router", "--json")
	if !strings.Contains(out, `"active": []`) {
		t.Fatalf("a disabled set must carry an explicit empty active list:\n%s", out)
	}
}

// TestHookPromptRouterJSONLoadErrorCarriesNoSet: a rules.json that will not
// load says so, and hands the client no set at all — "unknown, change
// nothing" — rather than an empty one that would read as every domain
// stopped over a typo.
func TestHookPromptRouterJSONLoadErrorCarriesNoSet(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	dir := t.TempDir()
	writeRepoRules(t, dir, "{not json")
	env := routerJSON(t, "j15-err", dir, "commit and push")
	if env.Error == "" || env.Active != nil || env.Text != "" {
		t.Fatalf("a load failure must carry an error, no text and no active set: %+v", env)
	}
}

// TestHookPromptRouterPlainOutputUnchanged: the set rides the JSON envelope
// alone. The host's own invocation (no --json) still writes nothing at all to
// the model-facing stream on a turn with nothing new.
func TestHookPromptRouterPlainOutputUnchanged(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	dir := t.TempDir()
	writeRepoRules(t, dir, widgetsRules)
	in := hookInputJSON(t, "j15-plain", dir, "count the widget")
	if out, _ := runHook(t, in, "hook", "prompt-router"); strings.Contains(out, `"active"`) || !strings.HasPrefix(out, "# abcd rules") {
		t.Fatalf("plain stdout must be the rendered block alone:\n%s", out)
	}
	if out, _ := runHook(t, in, "hook", "prompt-router"); out != "" {
		t.Fatalf("a deduped turn must add zero model-facing bytes, got %q", out)
	}
}

// brokenStdout is a stdout whose every write fails, the shape a host that
// closed its end of the pipe hands the hook.
type brokenStdout struct{}

func (brokenStdout) Write([]byte) (int, error) { return 0, os.ErrClosed }

// TestHookPromptRouterJSONEncodeFailureExitsZero: the --json envelope fails
// open like every other hook verb. An envelope the hook cannot write — on an
// evaluated prompt or on a refused one — is named on stderr and the process
// still exits 0, so a broken stdout can never wedge a session.
func TestHookPromptRouterJSONEncodeFailureExitsZero(t *testing.T) {
	for name, rulesBody := range map[string]string{
		"evaluated": widgetsRules,
		"refused":   "{not json",
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
			dir := t.TempDir()
			writeRepoRules(t, dir, rulesBody)
			cmd := NewRootCommand()
			var se strings.Builder
			cmd.SetOut(brokenStdout{})
			cmd.SetErr(&se)
			cmd.SetIn(strings.NewReader(hookInputJSON(t, "j15-broken-"+name, dir, "count the widget")))
			cmd.SetArgs([]string{"hook", "prompt-router", "--json"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("an unwritable envelope must still exit 0, got %v\n%s", err, se.String())
			}
			if !strings.Contains(se.String(), "envelope") {
				t.Fatalf("the unwritable envelope is not named on stderr:\n%s", se.String())
			}
		})
	}
}

// TestHookPromptRouterHelpNamesItsEnvelope: the generic --json text promises
// an {"abcd":"error"} document and a non-zero exit, which this verb never
// writes, so its own help states the envelope it does write and that an error
// still exits 0.
func TestHookPromptRouterHelpNamesItsEnvelope(t *testing.T) {
	out, _ := runHook(t, "", "hook", "prompt-router", "--help")
	for _, want := range []string{"text", "injected", "active", "error", "exits 0"} {
		if !strings.Contains(out, want) {
			t.Fatalf("prompt-router --help does not name %q:\n%s", want, out)
		}
	}
}
