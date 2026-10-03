package rules

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// The GRILL domain is the asking rules every abcd interview follows
// (itd-201; spc-2610030944505997 step 3). Its rules and recall terms are
// GENERATED from internal/core/question, the one source that also holds the
// question's field limits, never written in defaults/rules.json or in a
// repository override, so the limits the question check enforces and the
// limits the rule text states cannot part (itd-2610030810350727 criterion A7).

func grillRules(t *testing.T) []string {
	t.Helper()
	d, ok := Defaults().Domains[GrillDomain]
	if !ok {
		t.Fatalf("the %s domain is not bundled", GrillDomain)
	}
	return d.Rules
}

// ruleWith returns the first rule carrying every phrase, or "" when none does.
func ruleWith(rules []string, phrases ...string) string {
	for _, r := range rules {
		all := true
		for _, p := range phrases {
			if !strings.Contains(r, p) {
				all = false
				break
			}
		}
		if all {
			return r
		}
	}
	return ""
}

// TestGrillDomainIsGeneratedFromTheAskingRules is the drift test: the bundled
// GRILL domain carries exactly AskingRules(question.Default) and recalls on
// exactly AskingRecall, and it ships active.
func TestGrillDomainIsGeneratedFromTheAskingRules(t *testing.T) {
	d, ok := Defaults().Domains[GrillDomain]
	if !ok {
		t.Fatalf("the %s domain is not bundled", GrillDomain)
	}
	if d.State == StateDormant {
		t.Fatalf("the %s domain ships dormant: it would teach only on *%s", GrillDomain, GrillDomain)
	}
	if want := question.AskingRules(question.Default); !reflect.DeepEqual(d.Rules, want) {
		t.Errorf("%s rules drifted from the asking rules:\n got %q\nwant %q", GrillDomain, d.Rules, want)
	}
	if want := question.AskingRecall(); !reflect.DeepEqual(d.Recall, want) {
		t.Errorf("%s recall drifted from the asking recall:\n got %q\nwant %q", GrillDomain, d.Recall, want)
	}
}

// TestGrillDomainIsNotHandWritten (R2): the embedded rules.json never declares
// GRILL, so there is no second copy of the asking rules to fall out of step.
func TestGrillDomainIsNotHandWritten(t *testing.T) {
	var raw struct {
		Domains map[string]json.RawMessage `json:"domains"`
	}
	if err := json.Unmarshal(defaultsJSON, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw.Domains[GrillDomain]; ok {
		t.Fatalf("defaults/rules.json declares %s by hand; it is generated from internal/core/question", GrillDomain)
	}
}

// TestHandWrittenGrillPanicsAtLoad (R2): bundled defaults that declare GRILL
// by hand are refused at load, as a hand-written SHELL is, so the build cannot
// ship two statements of the rules.
func TestHandWrittenGrillPanicsAtLoad(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("bundled defaults declaring GRILL by hand loaded; want a panic")
		}
		if msg, _ := r.(string); !strings.Contains(msg, GrillDomain) || !strings.Contains(msg, "by hand") {
			t.Fatalf("the panic does not name the hand-written domain: %v", r)
		}
	}()
	withGrillDomain(RuleSet{SchemaVersion: 1, Domains: map[string]Domain{
		GrillDomain: {Recall: []string{"grill"}, Rules: []string{"A hand-written copy."}},
	}})
}

// TestGrillTextAsksOneThingAtATime (R3): one thing at a time is rule text,
// since a numbered list in a message is prose no hook sees. The rule asks the
// parts of one thing as tabs, up to the per-screen limit, a dependent question
// alone after its answer, and never a numbered list in a message; a host with
// no question tool asks one question per message (scope condition 1).
func TestGrillTextAsksOneThingAtATime(t *testing.T) {
	r := ruleWith(grillRules(t), "one thing at a time", "as tabs", "up to four on a screen",
		"asked alone, after that answer", "never as a numbered list in a message", "one question per message")
	if r == "" {
		t.Fatalf("GRILL has no one-thing-at-a-time rule with tabs:\n%s", strings.Join(grillRules(t), "\n"))
	}
}

// TestGrillTextRecordsDeferral (R5): deferral is recorded as an answer and
// silence is never consent, kept from the repository override with its ids
// stripped.
func TestGrillTextRecordsDeferral(t *testing.T) {
	if ruleWith(grillRules(t), "Deferral is a real answer and is recorded as one", "silence is never consent", "asked again rather than assumed") == "" {
		t.Fatalf("GRILL has no deferral rule:\n%s", strings.Join(grillRules(t), "\n"))
	}
}

// TestGrillTextCarriesTheAmendments: each amendment the spec lists is present
// in the generated text, and the words it replaces are gone.
func TestGrillTextCarriesTheAmendments(t *testing.T) {
	rules := grillRules(t)
	for _, c := range []struct {
		amendment string
		phrases   []string
	}{
		{"quoted in full", []string{"quoted in full in the question itself", "in paragraphs and lists", "never referred to", "one part per question"}},
		{"the layout", []string{"at most twelve columns", `"Product Q2"`, `"Now:"`, `"Change later:"`, "ends with the question", "two to four options", "at most five words", "at most two sentences", `"Decide later"`, "no bold", "twenty-four rows at eighty columns", "question check refuses"}},
		{"examples", []string{"one example of the thing being decided", "in the question text", "description says what choosing", "never holds it alone"}},
		{"gain and cost", []string{"gain and its cost", "never one option's alone", "trade-offs"}},
		{"recommendation only on request", []string{"never marked, styled, or ordered as recommended", "only when the person asks for one", "in prose beside the question"}},
		{"only real choices (kept)", []string{"two or more answers are each defensible", "never alternatives made up to fill a set", "not asked"}},
	} {
		if ruleWith(rules, c.phrases...) == "" {
			t.Errorf("GRILL has no rule carrying the %s amendment %q", c.amendment, c.phrases)
		}
	}

	// The knowledge floor collapses to one line pointing at the glossary page,
	// which holds it in full (itd-201 decision 5).
	var floor []string
	for _, r := range rules {
		if strings.Contains(r, "knowledge floor") {
			floor = append(floor, r)
		}
	}
	if len(floor) != 1 || !strings.Contains(floor[0], ".abcd/development/brief/glossary/interview/knowledge-floor.md") || !strings.Contains(floor[0], "plugin root") {
		t.Errorf("want exactly one knowledge-floor rule pointing at the glossary page under the plugin root, got %q", floor)
	}

	text := strings.Join(rules, "\n")
	for _, gone := range []string{"one sentence of context", "a checkout, a branch", "a symbolic link", "(recommended)", "ONE AT A TIME"} {
		if strings.Contains(text, gone) {
			t.Errorf("the generated text still says %q, which an amendment replaced", gone)
		}
	}
}

// TestGrillTextScopesRegisterToAbcdInterviews (itd-201 decision 10): the
// register rule, the ask-the-role-first rule and the mode rules hold for
// abcd's own interviews only, and the mode rule names the mode verb with no
// checkout-specific form.
func TestGrillTextScopesRegisterToAbcdInterviews(t *testing.T) {
	const scope = "In abcd's own interviews"
	rules := grillRules(t)
	for _, c := range []struct {
		rule    string
		phrases []string
	}{
		{"register", []string{"in their register", "outcomes and choices in product terms", "the mechanism, the ids, and the trade-offs"}},
		{"role first", []string{"which role the person holds", "the first question asks that"}},
		{"mode verb", []string{"`abcd mode facilitator`", "`abcd mode product-thinker`", "`abcd mode managed`", "relay it verbatim"}},
		{"addressee first", []string{"classify each question's addressee first", "a mixed interview re-sets the mode per question"}},
	} {
		r := ruleWith(rules, c.phrases...)
		if r == "" {
			t.Errorf("GRILL has no %s rule carrying %q", c.rule, c.phrases)
			continue
		}
		if !strings.HasPrefix(r, scope) {
			t.Errorf("the %s rule does not open with %q: %s", c.rule, scope, r)
		}
	}
	text := strings.Join(rules, "\n")
	for _, form := range []string{"./cmd/abcd", "this checkout", "plugin-root binary"} {
		if strings.Contains(text, form) {
			t.Errorf("the generated text names a checkout-specific form %q", form)
		}
	}
}

// TestGrillTextCarriesNoRecordHandleOrGoRun (itd-201 decision 8): the text
// reaches every managed repository, which has none of abcd's records and no
// source tree to run, so it names neither.
func TestGrillTextCarriesNoRecordHandleOrGoRun(t *testing.T) {
	for _, r := range grillRules(t) {
		if h, ok := recordid.HandleInText(r); ok {
			t.Errorf("a GRILL rule names the record handle %s: %s", h, r)
		}
		if strings.Contains(r, "go run") {
			t.Errorf("a GRILL rule names `go run`: %s", r)
		}
		if strings.Contains(r, "—") {
			t.Errorf("a GRILL rule carries an em dash: %s", r)
		}
	}
}

// TestGrillDomainRecallsAskingPrompts: a prompt about choosing or an interview
// recalls the domain.
func TestGrillDomainRecallsAskingPrompts(t *testing.T) {
	rs := Defaults()
	for _, prompt := range []string{
		"which option should we choose",
		"let's grill the draft",
		"run the planning interview",
		"ask the facilitator to decide",
	} {
		if !has(rs.Match(prompt), GrillDomain) {
			t.Errorf("asking prompt %q did not recall %s, got %v", prompt, GrillDomain, names(rs.Match(prompt)))
		}
	}
}

// TestGrillDomainObeysTheLoaderContracts: the generated domain is an ordinary
// bundled domain to every loader contract, as SHELL is: bundled provenance,
// per-field user and repository overrides, dormant, the star command, the kill
// switch, and a stable render for dedup.
func TestGrillDomainObeysTheLoaderContracts(t *testing.T) {
	prompt := "which option should we choose"

	t.Run("bundled provenance", func(t *testing.T) {
		d, ok := mustLoad(t, t.TempDir()).Lookup(GrillDomain)
		if !ok || d.Source != SourceBundled {
			t.Fatalf("Lookup(%s) = %+v, %v; want a bundled domain", GrillDomain, d, ok)
		}
		if !strings.HasPrefix(Render([]ResolvedDomain{d}), "# abcd rules — 1 domain(s) active\n## GRILL\n- ") {
			t.Fatalf("the bundled domain does not render bare:\n%s", Render([]ResolvedDomain{d}))
		}
	})

	t.Run("the skeleton ahoy writes inherits it", func(t *testing.T) {
		dir := t.TempDir()
		writeRepoRules(t, dir, `{"schema_version":1,"disabled":false,"domains":{}}`)
		rs := mustLoad(t, dir)
		if !has(rs.Match(prompt), GrillDomain) {
			t.Fatal("a repository with the empty skeleton did not recall GRILL")
		}
		d, _ := rs.Lookup(GrillDomain)
		if d.Source != SourceBundled || !reflect.DeepEqual(d.Rules, question.AskingRules(question.Default)) {
			t.Fatalf("the skeleton's GRILL is not the generated one: %+v", d)
		}
	})

	t.Run("dormant silences, star activates", func(t *testing.T) {
		dir := t.TempDir()
		writeRepoRules(t, dir, `{"schema_version":1,"domains":{"GRILL":{"state":"dormant"}}}`)
		rs := mustLoad(t, dir)
		if has(rs.Match(prompt), GrillDomain) {
			t.Fatal("a dormant GRILL still recalled")
		}
		if !has(rs.Match("*GRILL "+prompt), GrillDomain) {
			t.Fatal("*GRILL did not activate the dormant domain")
		}
		d, _ := rs.Lookup(GrillDomain)
		if d.Source != SourceRepo || !reflect.DeepEqual(d.Rules, question.AskingRules(question.Default)) {
			t.Fatalf("a state-only override should keep the generated rules and read as the repo's: %+v", d)
		}
	})

	t.Run("kill switch", func(t *testing.T) {
		dir := t.TempDir()
		writeRepoRules(t, dir, `{"schema_version":1,"disabled":true,"domains":{}}`)
		if got := mustLoad(t, dir).Match("*GRILL " + prompt); len(got) != 0 {
			t.Fatalf("the kill switch let %v through", names(got))
		}
	})

	t.Run("rules override replaces the list", func(t *testing.T) {
		dir := t.TempDir()
		writeRepoRules(t, dir, `{"schema_version":1,"domains":{"GRILL":{"rules":["Ask in our own words."]}}}`)
		d, _ := mustLoad(t, dir).Lookup(GrillDomain)
		if d.Source != SourceRepo || !reflect.DeepEqual(d.Rules, []string{"Ask in our own words."}) {
			t.Fatalf("the repo's rules did not replace the generated list: %+v", d)
		}
		if !strings.HasPrefix(Render([]ResolvedDomain{d}), "# abcd rules — 1 domain(s) active\n## GRILL (repo override)\n") {
			t.Fatalf("the overridden domain does not name its layer:\n%s", Render([]ResolvedDomain{d}))
		}
	})

	t.Run("user override", func(t *testing.T) {
		home := t.TempDir()
		writeUserRules(t, home, `{"schema_version":1,"domains":{"GRILL":{"state":"dormant"}}}`)
		t.Setenv("HOME", home)
		rs := mustLoad(t, t.TempDir())
		if has(rs.Match(prompt), GrillDomain) {
			t.Fatal("a user-layer dormant GRILL still recalled")
		}
		if d, _ := rs.Lookup(GrillDomain); d.Source != SourceUser {
			t.Fatalf("source %q, want %q", d.Source, SourceUser)
		}
	})

	t.Run("render is stable", func(t *testing.T) {
		a, _ := Defaults().Lookup(GrillDomain)
		b, _ := mustLoad(t, t.TempDir()).Lookup(GrillDomain)
		if Signature(a) != Signature(b) {
			t.Fatal("two loads of the generated domain sign differently; dedup would re-inject it every prompt")
		}
	})
}
