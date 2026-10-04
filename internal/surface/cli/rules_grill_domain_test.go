package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/question"
)

// The GRILL domain is generated into the binary's bundled defaults from
// internal/core/question (spc-2610030944505997 step 3), so every repository abcd
// manages receives it through the binary, and abcd's own repository runs the
// same text: its .abcd/rules.json no longer declares the domain.

// grillJSON is one domain as `abcd rules GRILL --json` prints it.
type grillJSON struct {
	Name   string   `json:"name"`
	Source string   `json:"source"`
	Rules  []string `json:"rules"`
}

func rulesGrillJSON(t *testing.T) grillJSON {
	t.Helper()
	var got grillJSON
	out := rulesJSON(t, "rules", "GRILL", "--json")
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("rules GRILL --json is not JSON: %v\n%s", err, out)
	}
	return got
}

// TestManagedRepositoryGetsGrillFromTheBinary (itd-201 R1): a managed
// repository whose .abcd/rules.json is the empty skeleton ahoy writes
// receives GRILL from the binary. An asking prompt injects it through the
// prompt hook, and `abcd rules GRILL --json` reports it bundled, its rules the
// asking rules generated from the field limits.
func TestManagedRepositoryGetsGrillFromTheBinary(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The skeleton ahoy's stepRules writes: no domain declared, every bundled
	// default inherited.
	skeleton := `{"schema_version":1,"disabled":false,"domains":{}}`
	if err := os.WriteFile(filepath.Join(dir, ".abcd", "rules.json"), []byte(skeleton), 0o644); err != nil {
		t.Fatal(err)
	}

	out, errlog := runHook(t, hookInputJSON(t, "grill-managed", dir, "which option should we choose"), "hook", "prompt-router")
	if !strings.Contains(out, "## GRILL\n") || !strings.Contains(out, "one thing at a time") {
		t.Fatalf("an asking prompt did not inject the bundled GRILL domain:\n%s\nstderr:\n%s", out, errlog)
	}

	t.Chdir(dir)
	got := rulesGrillJSON(t)
	if got.Name != "GRILL" || got.Source != "bundled" {
		t.Fatalf("rules GRILL --json = %+v; want the bundled GRILL domain", got)
	}
	if want := question.AskingRules(question.Default); !reflect.DeepEqual(got.Rules, want) {
		t.Fatalf("the managed repository's GRILL rules are not the generated asking rules:\n got %q\nwant %q", got.Rules, want)
	}
}

// TestAbcdsOwnRulesReportGrillBundled: abcd's own repository declares no GRILL
// override (itd-201 decision 8), so `abcd rules GRILL --json` run in it
// reports the bundled domain, the text every managed repository runs.
func TestAbcdsOwnRulesReportGrillBundled(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(testRepoRoot(), ".abcd", "rules.json"))
	if err != nil {
		t.Fatal(err)
	}
	var own struct {
		Domains map[string]json.RawMessage `json:"domains"`
	}
	if err := json.Unmarshal(data, &own); err != nil {
		t.Fatalf("parse .abcd/rules.json: %v", err)
	}
	if _, ok := own.Domains["GRILL"]; ok {
		t.Fatal(".abcd/rules.json declares GRILL; the domain is generated into the binary and the override is deleted")
	}
	root, err := filepath.Abs(testRepoRoot())
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	if got := rulesGrillJSON(t); got.Source != "bundled" {
		t.Fatalf("abcd rules GRILL --json in abcd's own repository reports source %q, want bundled", got.Source)
	}
}

// TestGrillQuotingRuleSaysProseIsInvisibleWhileTheQuestionShows holds the
// quoting rule (the thing being decided is quoted in the question) to the
// claim the example rule rests on: prose written between tool calls is
// invisible while the question shows. A softer "does not reliably reach"
// beside the firmer claim reads as two rules disagreeing about one fact
// (review of iss-2609291925134691, MINOR 4).
func TestGrillQuotingRuleSaysProseIsInvisibleWhileTheQuestionShows(t *testing.T) {
	t.Chdir(t.TempDir())
	found := false
	for _, r := range rulesGrillJSON(t).Rules {
		if strings.Contains(r, "does not reliably reach") {
			t.Errorf("a GRILL rule still says prose %q; the domain states it as invisible while the question shows: %s", "does not reliably reach", r)
		}
		if strings.Contains(r, "quoted in full in the question itself") {
			found = true
			if !strings.Contains(r, "invisible while the question shows") {
				t.Errorf("the GRILL quoting rule does not say prose is %q: %s", "invisible while the question shows", r)
			}
		}
	}
	if !found {
		t.Error("GRILL carries no rule that quotes the thing being decided in the question itself")
	}
}

// TestIntentInterviewPageCarriesGrillVisibilityAndAddresseeLines holds the
// planning interview's generated asking-rules block in commands/intent.md
// (written by cmd/asking-sync from the rules GRILL carries,
// spc-2610030944505997) to the two GRILL rules the product thinker's
// 2026-09-29 captures asked for, since that page ships to every repository
// abcd is installed in and is where an adopter's agent reads how to ask: the
// example sits in the question text and each option's meaning in its
// description, and no question carries a side preview, which hides every
// description (iss-2609291925134691; the layout intent's decision 20), and a mixed
// interview re-sets the addressee per question (iss-2609291925149138).
func TestIntentInterviewPageCarriesGrillVisibilityAndAddresseeLines(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(testRepoRoot(), "commands", "intent.md"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(data)
	const begin, end = "<!-- generated: asking-rules -->", "<!-- /generated -->"
	start := strings.Index(page, begin)
	if start < 0 {
		t.Fatal("commands/intent.md has no generated asking-rules block")
	}
	stop := strings.Index(page[start:], end)
	if stop < 0 {
		t.Fatal("commands/intent.md opens the asking-rules block and never closes it")
	}
	block := strings.Join(strings.Fields(page[start:start+stop]), " ")
	if !strings.Contains(block, "**How every question is asked") {
		t.Error("the generated block does not open with the \"How every question is asked\" lead")
	}
	for _, want := range []struct {
		issue, phrase string
	}{
		{"iss-2609291925134691", "one example of the thing being decided, in the question text"},
		{"iss-2609291925134691", "each option's description says what choosing that option means"},
		{"iss-2609291925134691", "carries no side preview"},
		{"iss-2609291925134691", "invisible while the question shows"},
		{"iss-2609291925149138", "classify each question's addressee first"},
		{"iss-2609291925149138", "a mixed interview re-sets the mode per question"},
	} {
		if !strings.Contains(block, want.phrase) {
			t.Errorf("the generated asking-rules block does not say %q (%s)", want.phrase, want.issue)
		}
	}
	if strings.Contains(block, "does not reliably reach") {
		t.Error("the generated block still says prose \"does not reliably reach\" the human; GRILL states it as invisible while a question shows")
	}
}
