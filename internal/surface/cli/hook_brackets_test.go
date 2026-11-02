package cli

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/rules"
)

// hook_brackets_test.go — the context brackets at the front doors
// (spc-2610111627213138 step 6): the prompt router reads the fill before it
// injects and acts on it, and `rules --prompt` names the bracket and takes
// --fill.

// threeRules is a domain of three rules, recalled by "widget".
const threeRules = `{"schema_version": 1, "domains": {"WIDGETS": {"recall": ["widget"], "rules": ["Count widgets twice.", "Never paint a widget blue.", "Ship widgets in pairs."]}}}`

// setFill records a used share of the window for session, as the status line
// does, making the fill directory first as the session-start reset does.
func setFill(t *testing.T, home, session string, used float64) {
	t.Helper()
	if err := rules.ResetFill(home, ""); err != nil {
		t.Fatal(err)
	}
	if err := rules.WriteFill(home, session, used, time.Now()); err != nil {
		t.Fatal(err)
	}
}

// bracketEnvelope is the router's --json envelope with the bracket.
type bracketEnvelope struct {
	Text     string      `json:"text"`
	Injected []string    `json:"injected"`
	Recalled []string    `json:"recalled"`
	Fill     *rules.Fill `json:"fill"`
	Bracket  string      `json:"bracket"`
}

func bracketRouter(t *testing.T, session, dir, prompt string) (bracketEnvelope, string) {
	t.Helper()
	out, errlog := runHook(t, hookInputJSON(t, session, dir, prompt), "hook", "prompt-router", "--json")
	var env bracketEnvelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("prompt-router --json is not one JSON document: %v\n%s", err, out)
	}
	return env, errlog
}

// TestHookRulesFollowTheFillsBracket: the router reads the fill before it
// injects: with 9% of the window free a domain comes back as its heading,
// first rule and the pointer to `abcd rules WIDGETS`; with no fill it comes
// back in full, as before the brackets; the plain form and the envelope
// carry the same text.
func TestHookRulesFollowTheFillsBracket(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := t.TempDir()
	writeRepoRules(t, dir, threeRules)
	setFill(t, home, "full-up", 91)

	env, errlog := bracketRouter(t, "full-up", dir, "count the widget")
	if env.Bracket != "depleted" {
		t.Fatalf("bracket = %q, want depleted for a 91%% fill", env.Bracket)
	}
	if !strings.Contains(env.Text, "Count widgets twice.") || strings.Contains(env.Text, "Never paint a widget blue.") ||
		!strings.Contains(env.Text, "`abcd rules WIDGETS`") {
		t.Fatalf("WIDGETS is not shrunk to its first rule and the pointer:\n%s", env.Text)
	}
	if !strings.Contains(errlog, "shrunk") {
		t.Errorf("stderr does not say the domain was shrunk:\n%s", errlog)
	}
	plain, _ := runHook(t, hookInputJSON(t, "full-up-plain", dir, "count the widget"), "hook", "prompt-router")
	if !strings.Contains(plain, "Never paint a widget blue.") {
		t.Fatalf("with no fill the domain did not come back in full:\n%s", plain)
	}
}

// TestHookUnknownFillKeepsRulesFullAndShortensNotes: with no source for the
// fill the rules come back in full, as before the brackets, and a matched
// note as one line: its id and title, never its body.
func TestHookUnknownFillKeepsRulesFullAndShortensNotes(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	_, repo := memoryFixture(t)
	writeRepoRules(t, repo, threeRules)
	id := addRaceNote(t, "session-one")

	env, _ := bracketRouter(t, "session-two", repo, "count the widget, then run the tests")
	if env.Bracket != "unknown" || env.Fill == nil || env.Fill.Source != rules.FillSourceUnknown {
		t.Fatalf("bracket %q fill %+v, want unknown", env.Bracket, env.Fill)
	}
	if !strings.Contains(env.Text, "Ship widgets in pairs.") {
		t.Fatalf("an unknown fill shrank the rules:\n%s", env.Text)
	}
	if !slices.Equal(env.Recalled, []string{id}) || !strings.Contains(env.Text, id) || !strings.Contains(env.Text, "The tests need the race flag") {
		t.Fatalf("the note's line did not come back: %v\n%s", env.Recalled, env.Text)
	}
	if strings.Contains(env.Text, strings.TrimSpace(raceBody)) {
		t.Fatalf("an unknown fill sent the note's body:\n%s", env.Text)
	}
}

// TestHookSaysTheUnknownFillOncePerSession: one line on stderr says the fill
// is unknown on the first prompt of a session it is unknown on, and not on
// the next; a session whose fill is known never hears it; after a reset it
// is said again.
func TestHookSaysTheUnknownFillOncePerSession(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := t.TempDir()
	writeRepoRules(t, dir, threeRules)
	const said = "how full this session is, is unknown"

	_, first := runHook(t, hookInputJSON(t, "quiet", dir, "count the widget"), "hook", "prompt-router")
	if strings.Count(first, said) != 1 {
		t.Fatalf("the first prompt did not say the fill is unknown once:\n%s", first)
	}
	_, second := runHook(t, hookInputJSON(t, "quiet", dir, "paint a landscape"), "hook", "prompt-router")
	if strings.Contains(second, said) {
		t.Fatalf("the second prompt said it again:\n%s", second)
	}
	runHook(t, hookInputJSON(t, "quiet", dir, ""), "hook", "prompt-router-reset")
	if _, again := runHook(t, hookInputJSON(t, "quiet", dir, "count the widget"), "hook", "prompt-router"); !strings.Contains(again, said) {
		t.Fatalf("after a reset the unknown fill was not said:\n%s", again)
	}
	setFill(t, home, "known", 20)
	if _, known := runHook(t, hookInputJSON(t, "known", dir, "count the widget"), "hook", "prompt-router"); strings.Contains(known, said) {
		t.Fatalf("a known fill was said to be unknown:\n%s", known)
	}
}

// TestHookBracketChangeNeverResendsAnItem: within a session a domain and a
// note sent shrunk are not sent again in full when the status line next
// reads the window freer, nor are they sent again at all.
func TestHookBracketChangeNeverResendsAnItem(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	home, repo := memoryFixture(t)
	writeRepoRules(t, repo, threeRules)
	id := addRaceNote(t, "session-one")
	setFill(t, home, "rising", 95)

	first, _ := bracketRouter(t, "rising", repo, "count the widget, then run the tests")
	if first.Bracket != "depleted" || !slices.Equal(first.Recalled, []string{id}) || !slices.Contains(first.Injected, "WIDGETS") {
		t.Fatalf("first prompt: %+v", first)
	}
	if err := rules.WriteFill(home, "rising", 5, time.Now()); err != nil {
		t.Fatal(err)
	}
	again, _ := bracketRouter(t, "rising", repo, "count the widget, then run the tests")
	if again.Bracket != "fresh" || again.Text != "" || len(again.Recalled) != 0 || len(again.Injected) != 0 {
		t.Fatalf("a bracket change sent an item again: %+v", again)
	}
}

// rulesPromptBracket is the bracket part of `rules --prompt --json`.
type rulesPromptBracketEnv struct {
	Bracket struct {
		Name           string   `json:"name"`
		Source         string   `json:"source"`
		UsedPercentage *float64 `json:"used_percentage"`
		Describe       string   `json:"describe"`
	} `json:"bracket"`
	Domains []struct {
		Name string `json:"name"`
		Form string `json:"form"`
	} `json:"domains"`
	Notes []struct {
		ID   string `json:"id"`
		Form string `json:"form"`
	} `json:"notes"`
}

func rulesPromptJSON(t *testing.T, args ...string) rulesPromptBracketEnv {
	t.Helper()
	out, stderr, err := runCLISplit(t, append([]string{"rules", "--json"}, args...)...)
	if err != nil {
		t.Fatalf("rules %v: %v\n%s", args, err, stderr)
	}
	var env rulesPromptBracketEnv
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("rules --prompt --json is not one document: %v\n%s", err, out)
	}
	return env
}

// TestRulesPromptFillSetsTheBracket: --fill <percent-used> sets the bracket
// the diagnostic renders: a domain shrunk and a note as one line when the
// window is nearly full, both in full when it is fresh.
func TestRulesPromptFillSetsTheBracket(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	_, repo := memoryFixture(t)
	writeRepoRules(t, repo, threeRules)
	id := addRaceNote(t, "session-one")
	const prompt = "count the widget, then run the tests"

	for _, tc := range []struct {
		fill, bracket, domainForm, noteForm string
	}{
		{"95", "depleted", "shrunk", "one-line"},
		{"50", "moderate", "shrunk", "full"},
		{"10", "fresh", "full", "full"},
	} {
		env := rulesPromptJSON(t, "--prompt", prompt, "--fill", tc.fill)
		if env.Bracket.Name != tc.bracket || env.Bracket.Source != "flag" || env.Bracket.UsedPercentage == nil || env.Bracket.Describe == "" {
			t.Errorf("--fill %s: bracket %+v, want %s from the flag", tc.fill, env.Bracket, tc.bracket)
		}
		if len(env.Domains) != 1 || env.Domains[0].Form != tc.domainForm {
			t.Errorf("--fill %s: domains %+v, want WIDGETS %s", tc.fill, env.Domains, tc.domainForm)
		}
		if len(env.Notes) != 1 || env.Notes[0].ID != id || env.Notes[0].Form != tc.noteForm {
			t.Errorf("--fill %s: notes %+v, want the note %s", tc.fill, env.Notes, tc.noteForm)
		}
	}
	out, _, err := runCLISplit(t, "rules", "--prompt", prompt, "--fill", "95")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Bracket: depleted", "--fill 95", "as one line", "heading and first rule"} {
		if !strings.Contains(out, want) {
			t.Errorf("the plain diagnostic does not say %q:\n%s", want, out)
		}
	}
}

// TestRulesPromptWithoutFillRendersTheUnknownForm: without --fill the
// diagnostic has no session, so the fill is unknown, and it says it renders
// the unknown-fill form: rules in full, notes and pages as one line each.
func TestRulesPromptWithoutFillRendersTheUnknownForm(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	_, repo := memoryFixture(t)
	writeRepoRules(t, repo, threeRules)
	addRaceNote(t, "session-one")
	const prompt = "count the widget, then run the tests"

	env := rulesPromptJSON(t, "--prompt", prompt)
	if env.Bracket.Name != "unknown" || env.Bracket.Source != "unknown" || env.Bracket.UsedPercentage != nil {
		t.Fatalf("bracket %+v, want unknown", env.Bracket)
	}
	if len(env.Domains) != 1 || env.Domains[0].Form != "full" || len(env.Notes) != 1 || env.Notes[0].Form != "one-line" {
		t.Fatalf("domains %+v notes %+v, want the domain in full and the note as one line", env.Domains, env.Notes)
	}
	out, _, err := runCLISplit(t, "rules", "--prompt", prompt)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Bracket: unknown", "unknown-fill form", "--fill"} {
		if !strings.Contains(out, want) {
			t.Errorf("the plain diagnostic does not say %q:\n%s", want, out)
		}
	}
}

// TestRulesPromptRefusesAStrayOrBadFill: --fill goes with --prompt only, and
// takes a share of the window used, 0 to 100; anything else is refused,
// exit 2, and nothing renders.
func TestRulesPromptRefusesAStrayOrBadFill(t *testing.T) {
	_, _ = memoryFixture(t)
	for _, args := range [][]string{
		{"rules", "--fill", "50"},
		{"rules", "COMMITTING", "--fill", "50"},
		{"rules", "--prompt", "commit it", "--fill", "101"},
		{"rules", "--prompt", "commit it", "--fill", "-1"},
		{"rules", "--prompt", "commit it", "--fill", "NaN"},
	} {
		out, _, err := runCLISplit(t, args...)
		var ee *exitError
		if !errors.As(err, &ee) || ee.Code != 2 || !strings.Contains(ee.Msg, "--fill") {
			t.Errorf("%v = %v, want exit 2 naming --fill", args, err)
		}
		if out != "" {
			t.Errorf("%v rendered:\n%s", args, out)
		}
	}
}
