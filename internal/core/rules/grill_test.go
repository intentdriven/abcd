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
		{"quoted in full", []string{"quoted in full in the question itself", "in paragraphs and lists", "never referred to", "quoted shorter and faithfully", "several decisions or too many answers"}},
		{"the layout", []string{"at most twelve columns", `"Q1"`, `"Q2/5"`, `"Product Q2"`, "is still read", `"Now:"`, `"Change later:"`, "ends with the question", "two to four options", "at most five words", "at most two sentences", `"Decide later"`, "no bold", "thirty-two rows at eighty columns", "question check refuses"}},
		{"examples", []string{"one example of the thing being decided", "in the question text", "description says what choosing", "carries no side preview", "hides every option's description"}},
		{"gain and cost", []string{"description names its gain and its cost", "never one option's alone", "trade-offs"}},
		{"recommendation only on request", []string{"never marked, styled, or ordered as recommended", "only when the person asks for one", "in prose beside the question"}},
		{"verdict confirmation", []string{"naming the person (an embargo triage verdict)", "even where the record settles it", "question text itself quotes each item's evidence", "the verdict the record supports", "why each other verdict does not fit", "only on their yes", "asked neutrally among them"}},
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
	for _, gone := range []string{"one sentence of context", "a checkout, a branch", "a symbolic link", "(recommended)", "ONE AT A TIME",
		"the chip sets the mode", "needs no `abcd mode` before it", "drafts the question itself, to the same rules"} {
		if strings.Contains(text, gone) {
			t.Errorf("the generated text still says %q, which an amendment replaced", gone)
		}
	}
}

// TestGrillTextScopesRegisterToAbcdInterviews (itd-201 decision 10): the
// register rule, the ask-the-role-first rule, the mode rules and the writer
// rule hold for abcd's own interviews only, and the mode rules name the mode
// verb with no checkout-specific form: the agent sets the mode to the
// question's addressee before asking it, and the answer sets it back
// (spc-2610301917245442, step 4).
func TestGrillTextScopesRegisterToAbcdInterviews(t *testing.T) {
	const scope = "In abcd's own interviews"
	rules := grillRules(t)
	for _, c := range []struct {
		rule    string
		phrases []string
	}{
		{"register", []string{"in their register", "outcomes and choices in product terms", "the mechanism, the ids, and the trade-offs"}},
		{"role first", []string{"which role the person holds", "the first question asks that"}},
		{"mode verb", []string{"`abcd mode facilitator`", "`abcd mode product-thinker`", "`abcd mode managed`", "relay it verbatim", "a stop that is not a question"}},
		{"mode set before the question", []string{"`abcd mode product-thinker` or `abcd mode facilitator` before the question is written", "the answer sets the mode back to managed"}},
		{"addressee first", []string{"classify each question's addressee first", "a mixed interview re-sets the mode per question"}},
		{"the writer", []string{"abcd:question-drafter", "`abcd guard question`", "only the question the writer returns"}},
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

// TestLoadWithARefusalWritesItIntoGrillAndTheNotes (spc-2610301950037469,
// approach 8): under a refused question limit the GRILL domain states abcd's
// defaults, names the refused key and why, and says the check refuses every
// abcd question; the notes carry the refusal as one line; the domain stays
// bundled; and every other bundled domain loads as before.
func TestLoadWithARefusalWritesItIntoGrillAndTheNotes(t *testing.T) {
	const reason = ".abcd/config.json (repo layer): question.rows is 100: outside 24 to 60"
	note := question.LimitsNote{Refused: &question.RefusedLimit{Key: question.RowsKey, File: ".abcd/config.json", Reason: reason}}
	rs, err := LoadWith(t.TempDir(), question.Default, note)
	if err != nil {
		t.Fatal(err)
	}
	d, ok := rs.Lookup(GrillDomain)
	if !ok || d.Source != SourceBundled {
		t.Fatalf("Lookup(%s) = %+v, %v; want the bundled domain", GrillDomain, d, ok)
	}
	if want := question.AskingRulesFrom(question.Default, note); !reflect.DeepEqual(d.Rules, want) {
		t.Fatalf("GRILL under a refusal is not the asking rules with the refusal written in:\n got %q\nwant %q", d.Rules, want)
	}
	layout := ruleWith(d.Rules, "Every question has one layout")
	for _, want := range []string{"thirty-two rows", "abcd's defaults", "`question.rows` in `.abcd/config.json` is outside", "refuses every abcd question"} {
		if !strings.Contains(layout, want) {
			t.Errorf("the layout rule under a refusal does not say %q:\n%s", want, layout)
		}
	}
	var lines []string
	for _, n := range rs.Notes() {
		if strings.Contains(n, reason) {
			lines = append(lines, n)
		}
	}
	if len(lines) != 1 || strings.Contains(lines[0], "\n") || !strings.Contains(lines[0], GrillDomain) {
		t.Errorf("the notes must carry the refusal as one line naming %s; got %q of %q", GrillDomain, lines, rs.Notes())
	}
	// The note says nothing of what GRILL states: a rules.json layer, a
	// dormant state or the kill switch decide that, not the refusal.
	for _, n := range lines {
		if strings.Contains(n, "states abcd's defaults") {
			t.Errorf("the note claims what GRILL states: %q", n)
		}
	}
	for name := range Defaults().Domains {
		if _, ok := rs.Domains[name]; !ok {
			t.Errorf("the bundled domain %s did not load beside the refusal", name)
		}
	}
	plain, err := LoadWith(t.TempDir(), question.Default, question.LimitsNote{})
	if err != nil {
		t.Fatal(err)
	}
	if len(plain.Notes()) != 0 {
		t.Errorf("a load with no refusal carries notes: %q", plain.Notes())
	}
	if p, _ := plain.Lookup(GrillDomain); !reflect.DeepEqual(p.Rules, question.AskingRules(question.Default)) {
		t.Errorf("LoadWith at the defaults with no note is not the bundled GRILL: %q", p.Rules)
	}
}
