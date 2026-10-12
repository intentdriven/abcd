package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// recallEnvelope is the prompt router's --json envelope with the memory
// recall's list (spc-2610111627213138 step 2).
type recallEnvelope struct {
	Text     string    `json:"text"`
	Injected []string  `json:"injected"`
	Recalled *[]string `json:"recalled"`
	Active   *[]string `json:"active"`
	Error    string    `json:"error"`
}

func recallRouterJSON(t *testing.T, session, dir, prompt string) (recallEnvelope, string) {
	t.Helper()
	out, errlog := runHook(t, hookInputJSON(t, session, dir, prompt), "hook", "prompt-router", "--json")
	var env recallEnvelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("prompt-router --json is not one JSON document: %v\n%s", err, out)
	}
	return env, errlog
}

// addRaceNote writes the race-flag note through `memory add` in session, and
// returns its id.
func addRaceNote(t *testing.T, session string, extra ...string) string {
	t.Helper()
	out, err := memoryAdd(t, raceBody, append([]string{"--session", session, "--json"}, extra...)...)
	if err != nil {
		t.Fatalf("memory add: %v\n%s", err, out)
	}
	var env struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("memory add envelope: %v\n%s", err, out)
	}
	return env.ID
}

// TestANoteComesBackOnAPromptAboutTests (N1): a note one session writes,
// unasked, comes back in the next session on a prompt about tests — through
// the prompt router the host already runs — once, until a reset.
func TestANoteComesBackOnAPromptAboutTests(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	_, repo := memoryFixture(t)

	// Session one writes the note.
	id := addRaceNote(t, "session-one")

	// Session two asks about tests.
	env, _ := recallRouterJSON(t, "session-two", repo, "why are the tests failing?")
	if env.Recalled == nil || !slices.Equal(*env.Recalled, []string{id}) {
		t.Fatalf("the note was not recalled: recalled = %v\n%s", env.Recalled, env.Text)
	}
	for _, want := range []string{"## MEMORY", id, "The tests need the race flag", strings.TrimSpace(raceBody)} {
		if !strings.Contains(env.Text, want) {
			t.Fatalf("the injected text does not carry %q:\n%s", want, env.Text)
		}
	}

	// The same session asking again is not sent the note twice.
	again, _ := recallRouterJSON(t, "session-two", repo, "run the tests again")
	if strings.Contains(again.Text, "## MEMORY") || again.Recalled == nil || len(*again.Recalled) != 0 {
		t.Fatalf("a recalled note was sent twice in one session: %+v", again)
	}

	// After the session's reset (a compaction or a new start) it comes back.
	runHook(t, hookInputJSON(t, "session-two", repo, ""), "hook", "prompt-router-reset")
	plain, _ := runHook(t, hookInputJSON(t, "session-two", repo, "the tests once more"), "hook", "prompt-router")
	if !strings.Contains(plain, "## MEMORY") || !strings.Contains(plain, "The tests need the race flag") {
		t.Fatalf("the note did not come back after the reset:\n%s", plain)
	}

	// A prompt about something else brings nothing back.
	other, _ := recallRouterJSON(t, "session-three", repo, "paint a landscape in oils")
	if other.Text != "" {
		t.Fatalf("an unrelated prompt injected text:\n%s", other.Text)
	}
}

// TestKillSwitchSilencesRecall: the loader's kill switch silences note recall
// as well as rules, in the hook and in the diagnostic.
func TestKillSwitchSilencesRecall(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	_, repo := memoryFixture(t)
	addRaceNote(t, "session-one")
	writeRepoRules(t, repo, `{"schema_version": 1, "disabled": true, "domains": {}}`)

	env, _ := recallRouterJSON(t, "session-two", repo, "why are the tests failing?")
	if env.Text != "" || env.Recalled == nil || len(*env.Recalled) != 0 {
		t.Fatalf("the kill switch let a note through: %+v", env)
	}
	out, _, err := runCLISplit(t, "rules", "--prompt", "why are the tests failing?")
	if err != nil {
		t.Fatalf("rules --prompt: %v", err)
	}
	if strings.Contains(out, "The tests need the race flag") || !strings.Contains(out, "disabled") {
		t.Fatalf("rules --prompt under the kill switch:\n%s", out)
	}
}

// TestARefusedMachineLayerStillInjectsRules: a machine memory.json the
// guarded read refuses — one that does not parse, one others can write —
// recalls no notes from it, says why on stderr, and the rules still inject.
func TestARefusedMachineLayerStillInjectsRules(t *testing.T) {
	for name, tc := range map[string]struct {
		body string
		mode os.FileMode
	}{
		"unparseable":        {"{not json", 0o600},
		"writable-by-others": {`{"schema_version": 1, "notes": []}`, 0o666},
		"another-schema":     {`{"schema_version": 9, "notes": []}`, 0o600},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
			home, repo := memoryFixture(t)
			path := abcdhome.Path(home, "memory.json")
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(path, tc.mode); err != nil {
				t.Fatal(err)
			}
			env, errlog := recallRouterJSON(t, "refused-"+name, repo, "commit and push the tests")
			if env.Error != "" || !strings.Contains(env.Text, "## COMMITTING") || !slices.Contains(env.Injected, "COMMITTING") {
				t.Fatalf("a refused memory.json stopped the rules: %+v\nstderr:\n%s", env, errlog)
			}
			if strings.Contains(env.Text, "## MEMORY") {
				t.Fatalf("a refused layer recalled a note:\n%s", env.Text)
			}
			if !strings.Contains(errlog, abcdhome.Display("memory.json")) || !strings.Contains(errlog, "recalling no notes from it") {
				t.Fatalf("stderr does not name the refused layer and what it means:\n%s", errlog)
			}
			if strings.Contains(errlog, home) {
				t.Fatalf("stderr carries the expanded home:\n%s", errlog)
			}
		})
	}
}

// TestRulesPromptShowsMatchedDomainsAndNotes: `rules --prompt "<text>"`
// shows what that one prompt would bring back to a fresh session: each
// matched domain with the recall words that hit, and each matched note.
func TestRulesPromptShowsMatchedDomainsAndNotes(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	_, _ = memoryFixture(t)
	id := addRaceNote(t, "session-one")

	out, stderr, err := runCLISplit(t, "rules", "--prompt", "commit once the tests pass")
	if err != nil {
		t.Fatalf("rules --prompt: %v\n%s", err, stderr)
	}
	for _, want := range []string{"COMMITTING", "commit", id, "The tests need the race flag", "test"} {
		if !strings.Contains(out, want) {
			t.Fatalf("rules --prompt does not show %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, strings.TrimSpace(raceBody)) {
		t.Fatalf("the diagnostic prints a note's body; it names the note:\n%s", out)
	}

	jout, stderr, err := runCLISplit(t, "rules", "--prompt", "commit once the tests pass", "--json")
	if err != nil {
		t.Fatalf("rules --prompt --json: %v\n%s", err, stderr)
	}
	var env struct {
		Disabled bool `json:"disabled"`
		Domains  []struct {
			Name string   `json:"name"`
			Hits []string `json:"hits"`
			Form string   `json:"form"`
		} `json:"domains"`
		Notes []struct {
			ID    string   `json:"id"`
			Title string   `json:"title"`
			Layer string   `json:"layer"`
			Hits  []string `json:"hits"`
			Form  string   `json:"form"`
		} `json:"notes"`
		Unreadable []struct{} `json:"unreadable"`
	}
	if err := json.Unmarshal([]byte(jout), &env); err != nil {
		t.Fatalf("rules --prompt --json is not one document: %v\n%s", err, jout)
	}
	var committing bool
	for _, d := range env.Domains {
		if d.Name == "COMMITTING" && slices.Contains(d.Hits, "commit") && d.Form == "full" {
			committing = true
		}
	}
	if !committing {
		t.Fatalf("COMMITTING with its hit word is missing: %+v", env.Domains)
	}
	if len(env.Notes) != 1 || env.Notes[0].ID != id || env.Notes[0].Layer != "machine" ||
		!slices.Equal(env.Notes[0].Hits, []string{"test"}) || env.Notes[0].Form != "full" {
		t.Fatalf("notes = %+v", env.Notes)
	}
	if env.Unreadable == nil {
		t.Fatalf("unreadable is absent, not an empty list:\n%s", jout)
	}
}

// treeSnapshot is every path under root with its size and mode, so a test
// can tell that nothing was created, removed or rewritten.
func treeSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		out[p] = fmt.Sprintf("%v %v %d", info.Mode(), info.ModTime(), info.Size())
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestRulesPromptWritesNothing: the diagnostic reads against an empty ledger
// and writes nothing — no session state, nothing in the home, nothing in the
// repository.
func TestRulesPromptWritesNothing(t *testing.T) {
	state := t.TempDir()
	t.Setenv("ABCD_RULES_STATE_DIR", state)
	home, repo := memoryFixture(t)
	addRaceNote(t, "session-one")
	before := map[string]map[string]string{"state": treeSnapshot(t, state), "home": treeSnapshot(t, home), "repo": treeSnapshot(t, repo)}

	out, stderr, err := runCLISplit(t, "rules", "--prompt", "commit once the tests pass")
	if err != nil {
		t.Fatalf("rules --prompt: %v\n%s", err, stderr)
	}
	if !strings.Contains(out, "The tests need the race flag") {
		t.Fatalf("rules --prompt did not show the note it should:\n%s", out)
	}
	after := map[string]map[string]string{"state": treeSnapshot(t, state), "home": treeSnapshot(t, home), "repo": treeSnapshot(t, repo)}
	for where := range before {
		for p, v := range after[where] {
			if before[where][p] != v {
				t.Errorf("%s: rules --prompt wrote %s", where, p)
			}
		}
		for p := range before[where] {
			if _, ok := after[where][p]; !ok {
				t.Errorf("%s: rules --prompt removed %s", where, p)
			}
		}
	}
	// The diagnostic shows a fresh session: a session that already had the
	// note does not change what it shows.
	runHook(t, hookInputJSON(t, "seen", repo, "the tests"), "hook", "prompt-router")
	if again, _, _ := runCLISplit(t, "rules", "--prompt", "commit once the tests pass"); again != out {
		t.Fatalf("rules --prompt read a session's ledger:\n%s\n---\n%s", out, again)
	}
}

// TestRulesPromptRefusesAPositionalDomain: --prompt and a DOMAIN are two
// forms; together they are refused, exit 2, naming both.
func TestRulesPromptRefusesAPositionalDomain(t *testing.T) {
	_, _ = memoryFixture(t)
	out, _, err := runCLISplit(t, "rules", "COMMITTING", "--prompt", "commit it")
	var ee *exitError
	if !errors.As(err, &ee) || ee.Code != 2 {
		t.Fatalf("rules DOMAIN --prompt = %v, want exit 2\n%s", err, out)
	}
	for _, want := range []string{"abcd rules <DOMAIN>", `abcd rules --prompt "<text>"`} {
		if !strings.Contains(ee.Msg, want) {
			t.Fatalf("the refusal does not name %q: %s", want, ee.Msg)
		}
	}
	if out != "" {
		t.Fatalf("a refused form rendered:\n%s", out)
	}
}

// writeMemoryPage writes one memory page into dir, with a recall: list when
// recall is given.
func writeRecallPage(t *testing.T, dir, name, body string, mode os.FileMode, recall ...string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	fm := "---\n"
	if len(recall) > 0 {
		fm += "recall: [" + strings.Join(recall, ", ") + "]\n"
	}
	fm += "source:\n  class: session_memory\n---\n"
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(fm+body), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, mode); err != nil {
		t.Fatal(err)
	}
}

// TestANoteAndAPageComeBackTogether (N2): a note and a page that both match
// a prompt about tests come back together, in one block, through the one
// recall the prompt router runs.
func TestANoteAndAPageComeBackTogether(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	_, repo := memoryFixture(t)
	id := addRaceNote(t, "session-one")
	writeRecallPage(t, filepath.Join(repo, ".abcd", "memory"), "fact_eng_test-layout.md",
		"# The tests live beside their package\n\nTable tests, one file per verb.\n", 0o644, "test")

	env, _ := recallRouterJSON(t, "session-two", repo, "why are the tests failing?")
	if env.Recalled == nil || !slices.Equal(slices.Sorted(slices.Values(*env.Recalled)), []string{id, "page:fact_eng_test-layout.md"}) {
		t.Fatalf("recalled = %v, want the note and the page\n%s", env.Recalled, env.Text)
	}
	if strings.Count(env.Text, "## MEMORY") != 1 {
		t.Fatalf("the note and the page did not come back in one block:\n%s", env.Text)
	}
	for _, want := range []string{"The tests need the race flag", "The tests live beside their package", "Table tests, one file per verb."} {
		if !strings.Contains(env.Text, want) {
			t.Fatalf("the block does not carry %q:\n%s", want, env.Text)
		}
	}
}

// TestRulesPromptShowsMatchedPages: `rules --prompt` lists each page the
// prompt recalls, from either scope, with its key, layer and hitting words,
// and names no page the prompt does not match.
func TestRulesPromptShowsMatchedPages(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	home, repo := memoryFixture(t)
	workspace := filepath.Join(repo, ".abcd", "memory")
	writeRecallPage(t, workspace, "fact_eng_auth-flow.md", "# Auth uses short-lived tokens\n\nThe page's own words.\n", 0o644, "auth", "login")
	writeRecallPage(t, workspace, "fact_eng_billing.md", "# Billing runs nightly\n", 0o644, "billing")
	writeRecallPage(t, abcdhome.Path(home, "memory"), "fact_eng_auth-keys.md", "# Auth keys live in the keychain\n", 0o600, "auth")

	out, stderr, err := runCLISplit(t, "rules", "--prompt", "the auth login is broken")
	if err != nil {
		t.Fatalf("rules --prompt: %v\n%s", err, stderr)
	}
	for _, want := range []string{"Pages:", "page:fact_eng_auth-flow.md  Auth uses short-lived tokens (project layer) — hit by auth, login",
		"page:fact_eng_auth-keys.md  Auth keys live in the keychain (machine layer) — hit by auth"} {
		if !strings.Contains(out, want) {
			t.Fatalf("rules --prompt does not show %q:\n%s", want, out)
		}
	}
	for _, not := range []string{"Billing", "The page's own words."} {
		if strings.Contains(out, not) {
			t.Fatalf("rules --prompt shows %q:\n%s", not, out)
		}
	}

	jout, stderr, err := runCLISplit(t, "rules", "--prompt", "the auth login is broken", "--json")
	if err != nil {
		t.Fatalf("rules --prompt --json: %v\n%s", err, stderr)
	}
	var env struct {
		Pages []struct {
			Key      string   `json:"key"`
			Filename string   `json:"filename"`
			Title    string   `json:"title"`
			Layer    string   `json:"layer"`
			Hits     []string `json:"hits"`
			Form     string   `json:"form"`
		} `json:"pages"`
	}
	if err := json.Unmarshal([]byte(jout), &env); err != nil {
		t.Fatalf("rules --prompt --json is not one document: %v\n%s", err, jout)
	}
	if len(env.Pages) != 2 {
		t.Fatalf("pages = %+v", env.Pages)
	}
	p := env.Pages[0]
	if p.Key != "page:fact_eng_auth-flow.md" || p.Filename != "fact_eng_auth-flow.md" || p.Title != "Auth uses short-lived tokens" ||
		p.Layer != "project" || !slices.Equal(p.Hits, []string{"auth", "login"}) || p.Form != "full" {
		t.Fatalf("pages[0] = %+v", p)
	}
	if env.Pages[1].Key != "page:fact_eng_auth-keys.md" || env.Pages[1].Layer != "machine" {
		t.Fatalf("pages[1] = %+v", env.Pages[1])
	}
}

// TestARefusedPageIsNamedAndTheRestComeBack: a machine page others can write
// is refused and named on stderr, in tilde form; the scope's other pages and
// the rules still come back.
func TestARefusedPageIsNamedAndTheRestComeBack(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	home, repo := memoryFixture(t)
	machine := abcdhome.Path(home, "memory")
	writeRecallPage(t, machine, "fact_eng_auth-flow.md", "# Kept to this account\n", 0o600, "commit")
	writeRecallPage(t, machine, "fact_eng_auth-keys.md", "# Anyone can rewrite this\n", 0o666, "commit")

	env, errlog := recallRouterJSON(t, "refused-page", repo, "commit the fix")
	if !slices.Contains(env.Injected, "COMMITTING") || env.Recalled == nil || !slices.Equal(*env.Recalled, []string{"page:fact_eng_auth-flow.md"}) {
		t.Fatalf("injected %v recalled %v:\n%s\nstderr:\n%s", env.Injected, env.Recalled, env.Text, errlog)
	}
	if strings.Contains(env.Text, "Anyone can rewrite this") {
		t.Fatalf("a page others can write came back:\n%s", env.Text)
	}
	if !strings.Contains(errlog, abcdhome.Display("memory", "fact_eng_auth-keys.md")) || !strings.Contains(errlog, "recalling nothing from it") {
		t.Fatalf("stderr does not name the refused page:\n%s", errlog)
	}
	if strings.Contains(errlog, home) {
		t.Fatalf("stderr carries the expanded home:\n%s", errlog)
	}
}
