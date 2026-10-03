package question

import (
	"slices"
	"strconv"
	"strings"
	"testing"
)

// verbs stands in for the binary's verb list, which the surface reads from its
// command tree and passes in (rule 9); the core holds no copy of it.
var verbs = []string{"capture", "intent", "mode", "rules", "spec", "status"}

func productThinker() Addressee { return Addressee{Person: ProductThinker, Verbs: verbs} }
func facilitator() Addressee    { return Addressee{Person: Facilitator, Verbs: verbs} }
func unnamed() Addressee        { return Addressee{Person: Unnamed, Verbs: verbs} }

// wellBuiltTab is A2's question: a "Product Q2" chip, the thing being decided
// quoted first in its own paragraphs, the Now: and Change later: lines, the
// question last, three options with short labels and two-sentence meanings,
// and "Decide later" last. No bold, no "(Recommended)", no record handle and no
// command, so the product thinker may be asked it.
func wellBuiltTab() Tab {
	return Tab{
		Header: "Product Q2",
		Text: strings.Join([]string{
			"Every question names who it is for in a short label above it: for example, the label reads Product Q2 on the second question put to you.",
			"",
			"Now: not applicable",
			"Change later: not applicable",
			"",
			"Should the label name the role as well as the number?",
		}, "\n"),
		Options: []Choice{
			{Label: "Role and number", Description: "It says whom it is for and where you are. It costs a few characters."},
			{Label: "Number only", Description: "The label is shorter. You read the question to learn whom it is for."},
			{Label: "Decide later", Description: "Nothing changes now. The question comes back at the next interview."},
		},
	}
}

func one(t Tab) Fields { return Fields{Tabs: []Tab{t}} }

// rulesOf lists the rules a set of findings names, in order.
func rulesOf(fs []Finding) []Rule {
	out := make([]Rule, len(fs))
	for i, f := range fs {
		out[i] = f.Rule
	}
	return out
}

// onlyRule asserts every finding is under rule r, and that there is at least
// one: the case breaks exactly the rule it is about.
func onlyRule(t *testing.T, fs []Finding, r Rule) {
	t.Helper()
	if len(fs) == 0 {
		t.Fatalf("admitted; want a %s finding", r)
	}
	for _, f := range fs {
		if f.Rule != r {
			t.Fatalf("findings %v; want only %s:\n%s", rulesOf(fs), r, render(fs))
		}
	}
}

func render(fs []Finding) string {
	var b strings.Builder
	for _, f := range fs {
		b.WriteString(f.String())
		b.WriteByte('\n')
	}
	return b.String()
}

// TestWellBuiltQuestionIsAdmitted is A2 at the core: the well-built question
// draws no finding for either addressee, or with no mode naming anyone.
func TestWellBuiltQuestionIsAdmitted(t *testing.T) {
	for name, who := range map[string]Addressee{"product thinker": productThinker(), "facilitator": facilitator(), "unnamed": unnamed()} {
		if fs := CheckLimits(one(wellBuiltTab()), Default, who); len(fs) != 0 {
			t.Errorf("%s: well-built question refused:\n%s", name, render(fs))
		}
	}
	// A chip with a total, admitted though never required (review finding 5).
	tab := wellBuiltTab()
	tab.Header = "Setup Q1/4"
	if fs := CheckLimits(one(tab), Default, facilitator()); len(fs) != 0 {
		t.Errorf("chip with a total refused:\n%s", render(fs))
	}
}

// TestEveryRuleRefusesOnItsOwn is one refusal case per rule: each case breaks
// one rule of the well-built question and must draw findings under that rule
// alone, naming the tab, the part, the value and the limit.
func TestEveryRuleRefusesOnItsOwn(t *testing.T) {
	cases := []struct {
		name string
		rule Rule
		who  Addressee
		edit func(*Fields)
		part string
	}{
		{"header missing", RuleHeader, facilitator(), func(f *Fields) { f.Tabs[0].Header = "" }, "header"},
		{"header not a chip", RuleHeader, facilitator(), func(f *Fields) { f.Tabs[0].Header = "Pick a theme" }, "header"},
		{"no questions", RuleQuestionsPerCall, facilitator(), func(f *Fields) { f.Tabs = nil }, "questions"},
		{"five questions", RuleQuestionsPerCall, facilitator(), func(f *Fields) {
			for i := 0; i < 4; i++ {
				f.Tabs = append(f.Tabs, wellBuiltTab())
			}
		}, "questions"},
		{"one option", RuleOptions, facilitator(), func(f *Fields) { f.Tabs[0].Options = f.Tabs[0].Options[2:] }, "options"},
		{"five options", RuleOptions, facilitator(), func(f *Fields) {
			o := f.Tabs[0].Options
			f.Tabs[0].Options = []Choice{o[0], o[1], {Label: "Third way", Description: "A third answer."}, {Label: "Fourth way", Description: "A fourth answer."}, o[2]}
		}, "options"},
		{"label of six words", RuleLabelWords, facilitator(), func(f *Fields) { f.Tabs[0].Options[0].Label = "Keep the role and the number" }, "option 1 label"},
		{"three sentences", RuleMeaningSentences, facilitator(), func(f *Fields) {
			f.Tabs[0].Options[1].Description = "The label is shorter. You read the question to learn whom it is for. Nothing else moves."
		}, "option 2 description"},
		{"meaning only in the preview", RuleMeaningSentences, facilitator(), func(f *Fields) {
			f.Tabs[0].Options[1].Preview = "The label is shorter."
			f.Tabs[0].Options[1].Description = ""
		}, "option 2 description"},
		{"no decide-later option", RuleDecideLater, facilitator(), func(f *Fields) {
			f.Tabs[0].Options[2] = Choice{Label: "Neither", Description: "Leave the label as it is."}
		}, "options"},
		{"decide later not last", RuleDecideLater, facilitator(), func(f *Fields) {
			o := f.Tabs[0].Options
			o[1], o[2] = o[2], o[1]
		}, "option 2 label"},
		{"two decide-later options", RuleDecideLater, facilitator(), func(f *Fields) {
			f.Tabs[0].Options[1] = Choice{Label: "None of these", Description: "Neither answer fits."}
		}, "option 2 label"},
		{"bold in the text", RuleNoBold, facilitator(), func(f *Fields) {
			f.Tabs[0].Text = strings.Replace(f.Tabs[0].Text, "short label", "**short** label", 1)
		}, "question text"},
		{"bold in a description", RuleNoBold, facilitator(), func(f *Fields) {
			f.Tabs[0].Options[0].Description = "The label says __whom__ it is for."
		}, "option 1 description"},
		{"recommended label", RuleNeverRecommended, facilitator(), func(f *Fields) { f.Tabs[0].Options[0].Label = "Role and number (Recommended)" }, "option 1 label"},
		{"record handle for the product thinker", RuleRegister, productThinker(), func(f *Fields) {
			f.Tabs[0].Options[0].Description = "As iss-2609202058058301 asked. It costs a few characters."
		}, "option 1 description"},
		{"no Now: line", RuleNowAndChangeLater, facilitator(), func(f *Fields) {
			f.Tabs[0].Text = strings.Replace(f.Tabs[0].Text, "Now: not applicable\n", "", 1)
		}, "question text"},
		{"Change later: with no value", RuleNowAndChangeLater, facilitator(), func(f *Fields) {
			f.Tabs[0].Text = strings.Replace(f.Tabs[0].Text, "Change later: not applicable", "Change later:", 1)
		}, "question text"},
		{"question not last", RuleThingFirst, facilitator(), func(f *Fields) { f.Tabs[0].Text += "\n\nThank you." }, "question text"},
		{"em dash in a list item", RuleEmDashListItem, facilitator(), func(f *Fields) {
			f.Tabs[0].Text = "- the label — who it is for\n\n" + f.Tabs[0].Text
		}, "question text"},
		{"too tall", RuleRows, facilitator(), func(f *Fields) {
			f.Tabs[0].Text = strings.Repeat("A paragraph of material that runs on and on.\n\n", 9) + f.Tabs[0].Text
		}, "question text"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := one(wellBuiltTab())
			c.edit(&f)
			fs := CheckLimits(f, Default, c.who)
			onlyRule(t, fs, c.rule)
			got := fs[0]
			if got.Part != c.part {
				t.Errorf("part = %q, want %q:\n%s", got.Part, c.part, render(fs))
			}
			if got.Limit == "" || got.Remedy == "" {
				t.Errorf("finding names no limit or no remedy: %+v", got)
			}
			wantTab := 1
			if c.rule == RuleQuestionsPerCall {
				wantTab = 0
			}
			if got.Tab != wantTab {
				t.Errorf("tab = %d, want %d", got.Tab, wantTab)
			}
		})
	}
}

// TestRecommendedStarredOrLongHeaderIsRefused is A3 at the core: a label
// "Keep it (Recommended)", a label "★ Keep it", and a header "Product thinker
// Q2" are each refused naming the rule, the value, and the limit.
func TestRecommendedStarredOrLongHeaderIsRefused(t *testing.T) {
	cases := []struct {
		name  string
		rule  Rule
		edit  func(*Tab)
		value string
		limit string
	}{
		{"recommended", RuleNeverRecommended, func(tb *Tab) { tb.Options[0].Label = "Keep it (Recommended)" }, "Keep it (Recommended)", "(Recommended)"},
		{"starred", RuleNeverRecommended, func(tb *Tab) { tb.Options[0].Label = "★ Keep it" }, "★ Keep it", "star"},
		{"long header", RuleHeader, func(tb *Tab) { tb.Header = "Product thinker Q2" }, "Product thinker Q2", "12 columns"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tab := wellBuiltTab()
			c.edit(&tab)
			fs := CheckLimits(one(tab), Default, facilitator())
			onlyRule(t, fs, c.rule)
			var hit bool
			for _, f := range fs {
				if f.Value == c.value && strings.Contains(f.Limit, c.limit) {
					hit = true
				}
			}
			if !hit {
				t.Errorf("no finding names value %q and a limit naming %q:\n%s", c.value, c.limit, render(fs))
			}
		})
	}
}

// TestRecommendedOrStarredOptionIsRefused is R6 at the core: the refusal names
// the label, and its remedy names the host's own instruction and abcd's rule
// reversing it, verbatim from the spec, so the agent does not loop.
func TestRecommendedOrStarredOptionIsRefused(t *testing.T) {
	const remedy = "The host's own instruction for this tool asks for a recommended first option; abcd's asking rule reverses it: no option is marked, styled, or ordered as recommended. Remove the mark and ask again; if the person asks for a recommendation, give it in prose beside the question."
	for _, label := range []string{"Role and number (recommended)", "Role and number (RECOMMENDED)", "★ Role and number", "Role and number ☆", "⭐️ Role and number", "* Role and number", "Role and number *"} {
		tab := wellBuiltTab()
		tab.Options[0].Label = label
		fs := CheckLimits(one(tab), Default, facilitator())
		onlyRule(t, fs, RuleNeverRecommended)
		if fs[0].Value != label || fs[0].Part != "option 1 label" {
			t.Errorf("%q: finding does not name the label: %+v", label, fs[0])
		}
		if fs[0].Remedy != remedy {
			t.Errorf("%q: remedy = %q, want the spec's text verbatim", label, fs[0].Remedy)
		}
	}
	// A star inside a label is not a mark.
	tab := wellBuiltTab()
	tab.Options[0].Label = "Rate it 5★ now"
	if fs := CheckLimits(one(tab), Default, facilitator()); len(fs) != 0 {
		t.Errorf("an interior star refused:\n%s", render(fs))
	}
}

// TestProductThinkerQuestionNamesNoRecordOrCommand is R4 at the core: for the
// product thinker a record handle, a backtick span, a `go run`, a slash command
// and abcd followed by one of the binary's verbs are each refused, in any field
// previews included; the same question for the facilitator is admitted; with
// no mode naming anyone the "Product" chip stands in, and a "Tech" chip does
// not.
func TestProductThinkerQuestionNamesNoRecordOrCommand(t *testing.T) {
	cases := []struct {
		name  string
		edit  func(*Tab)
		value string
	}{
		{"record handle in a description", func(tb *Tab) { tb.Options[0].Description = "As iss-2609202058058301 asked." }, "iss-2609202058058301"},
		{"command in the text", func(tb *Tab) {
			tb.Text = "Run abcd mode before every stop.\n\n" + tb.Text
		}, "abcd mode"},
		{"label in backticks", func(tb *Tab) { tb.Options[1].Label = "`number` only" }, "`number`"},
		{"go run in a preview", func(tb *Tab) { tb.Options[1].Preview = "It runs go run ./cmd/abcd." }, "go run"},
		{"slash command", func(tb *Tab) { tb.Options[1].Description = "Run /abcd:intent next." }, "/abcd:intent"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tab := wellBuiltTab()
			c.edit(&tab)
			fs := CheckLimits(one(tab), Default, productThinker())
			onlyRule(t, fs, RuleRegister)
			if fs[0].Value != c.value {
				t.Errorf("value = %q, want %q", fs[0].Value, c.value)
			}
			if fs := CheckLimits(one(tab), Default, facilitator()); len(fs) != 0 {
				t.Errorf("facilitator refused:\n%s", render(fs))
			}
			if fs := CheckLimits(one(tab), Default, unnamed()); len(fs) == 0 || fs[0].Rule != RuleRegister {
				t.Errorf("no mode and a Product chip: want a register finding, got:\n%s", render(fs))
			}
			tab.Header = "Tech Q2"
			if fs := CheckLimits(one(tab), Default, unnamed()); len(fs) != 0 {
				t.Errorf("no mode and a Tech chip refused:\n%s", render(fs))
			}
		})
	}
	// "abcd" followed by a word that is not a verb is product prose.
	tab := wellBuiltTab()
	tab.Text = "abcd asks every question the same way.\n\n" + tab.Text
	if fs := CheckLimits(one(tab), Default, productThinker()); len(fs) != 0 {
		t.Errorf("abcd before a non-verb refused:\n%s", render(fs))
	}
}

// TestCriterionQuestionQuotesTheCriterionFirst is S1 at the core: a criterion's
// full text, a blank line, the Now: and Change later: lines and "Does it
// stand?" are admitted; "Does criterion A4 stand?" alone is refused under the
// thing-first rule.
func TestCriterionQuestionQuotesTheCriterionFirst(t *testing.T) {
	tab := wellBuiltTab()
	tab.Header = "Tech Q4"
	tab.Text = "Given a question the check lets through, when it is shown at 80 columns, then it fits within 24 rows.\n\nNow: not applicable\nChange later: not applicable\n\nDoes it stand?"
	if fs := CheckLimits(one(tab), Default, facilitator()); len(fs) != 0 {
		t.Errorf("criterion quoted first refused:\n%s", render(fs))
	}
	tab.Text = "Now: not applicable\nChange later: not applicable\n\nDoes criterion A4 stand?"
	fs := CheckLimits(one(tab), Default, facilitator())
	onlyRule(t, fs, RuleThingFirst)
}

// TestPressReleaseQuestionQuotesTheParagraph is S2 at the core.
func TestPressReleaseQuestionQuotesTheParagraph(t *testing.T) {
	tab := wellBuiltTab()
	tab.Text = "Now: not applicable\nChange later: not applicable\n\nConfirm or change paragraph 2?"
	onlyRule(t, CheckLimits(one(tab), Default, productThinker()), RuleThingFirst)
	tab.Text = "Whenever abcd asks you something, you can tell at a glance who the question is for.\n\n" + tab.Text
	if fs := CheckLimits(one(tab), Default, productThinker()); len(fs) != 0 {
		t.Errorf("paragraph quoted first refused:\n%s", render(fs))
	}
}

// TestOpenQuestionQuotesAndOffersDecideLater is S3 at the core: an open
// question asked without its text first, and without "Decide later", is
// refused naming each part.
func TestOpenQuestionQuotesAndOffersDecideLater(t *testing.T) {
	tab := wellBuiltTab()
	tab.Text = "Now: not applicable\nChange later: not applicable\n\nResolve or defer the height budget?"
	tab.Options = []Choice{
		{Label: "Resolve it now", Description: "The budget is set today."},
		{Label: "Leave it open", Description: "The budget waits."},
	}
	fs := CheckLimits(one(tab), Default, productThinker())
	got := rulesOf(fs)
	if !slices.Contains(got, RuleThingFirst) || !slices.Contains(got, RuleDecideLater) || len(got) != 2 {
		t.Errorf("want one thing-first and one decide-later finding, got:\n%s", render(fs))
	}
}

// TestPreviewQuestionKeepsDecideLater is A5's automatic half at the core: a
// question with previews and no decide-later option is refused (the preview
// removes the host's free-text row, which "Decide later" stands in for), and
// the same question with it is admitted.
func TestPreviewQuestionKeepsDecideLater(t *testing.T) {
	tab := wellBuiltTab()
	for i := range tab.Options {
		tab.Options[i].Preview = "What changes:\n- the label\n- nothing else"
	}
	if fs := CheckLimits(one(tab), Default, productThinker()); len(fs) != 0 {
		t.Errorf("preview question with Decide later refused:\n%s", render(fs))
	}
	tab.Options[2] = Choice{Label: "Neither", Description: "Leave the label as it is.", Preview: "Nothing changes."}
	onlyRule(t, CheckLimits(one(tab), Default, productThinker()), RuleDecideLater)
}

// criterion is a criterion of about three wrapped lines at the host's text
// measure, the material of one criteria-walk question.
func criterion(n int) string {
	return "Criterion " + strconv.Itoa(n) +
		": given a question the check lets through, when it is shown in the terminal view at eighty columns, then the label, the whole question, every option and what it means are visible without scrolling, and it fits."
}

func criteriaTab(header string, ns ...int) Tab {
	var b strings.Builder
	for _, n := range ns {
		b.WriteString(criterion(n))
		b.WriteString("\n\n")
	}
	b.WriteString("Now: not applicable\nChange later: not applicable\n\nDoes it stand?")
	return Tab{
		Header: header,
		Text:   b.String(),
		Options: []Choice{
			{Label: "It stands", Description: "The criterion is kept as written."},
			{Label: "Change it", Description: "You say what to change next."},
			{Label: "Decide later", Description: "It is asked again at the next interview."},
		},
	}
}

// TestQuestionFitsTwentyFourRowsAt80 is A4's automatic half at the core: the
// row estimate admits the well-built question, refuses five criteria in one
// question with the split remedy, and admits the same five as four tabs and
// then one question, each tab on its own.
func TestQuestionFitsTwentyFourRowsAt80(t *testing.T) {
	if rows := estimateRows(wellBuiltTab(), Default); rows > Default.Rows {
		t.Errorf("well-built question estimated at %d rows, over %d", rows, Default.Rows)
	}
	all := criteriaTab("Product Q1", 1, 2, 3, 4, 5)
	fs := CheckLimits(one(all), Default, productThinker())
	onlyRule(t, fs, RuleRows)
	if !strings.Contains(fs[0].Remedy, "tabs") || !strings.Contains(fs[0].Remedy, "one part per question") {
		t.Errorf("remedy does not say to split into tabs or successive questions: %q", fs[0].Remedy)
	}
	four := Fields{Tabs: []Tab{
		criteriaTab("Product Q1", 1), criteriaTab("Product Q2", 2),
		criteriaTab("Product Q3", 3), criteriaTab("Product Q4", 4),
	}}
	if fs := CheckLimits(four, Default, productThinker()); len(fs) != 0 {
		t.Errorf("four tabs refused:\n%s", render(fs))
	}
	if fs := CheckLimits(one(criteriaTab("Product Q5", 5)), Default, productThinker()); len(fs) != 0 {
		t.Errorf("the fifth alone refused:\n%s", render(fs))
	}
}

// TestPreviewIsHeldToTheRowsLeftIt: a preview taller than the rows the frame
// leaves it is refused, so the host never cuts it.
func TestPreviewIsHeldToTheRowsLeftIt(t *testing.T) {
	tab := wellBuiltTab()
	tab.Options[0].Preview = strings.Repeat("a line\n", 20)
	fs := CheckLimits(one(tab), Default, facilitator())
	onlyRule(t, fs, RuleRows)
	if fs[0].Part != "option 1 preview" {
		t.Errorf("part = %q, want option 1 preview", fs[0].Part)
	}
}

// TestMeaningSentencesMaskAbbreviations: "e.g.", "i.e." and "etc." end no
// sentence, and a closing fragment without a stop counts as one.
func TestMeaningSentencesMaskAbbreviations(t *testing.T) {
	cases := map[string]int{
		"Keeps it, e.g. as now. Costs nothing.":     2,
		"One thing, i.e. the label, etc. and more.": 1,
		"No stop at all":        1,
		"First! Second? Third.": 3,
		"Version 1.5 stays.":    1,
		"":                      0,
	}
	for in, want := range cases {
		if got := sentences(in); got != want {
			t.Errorf("sentences(%q) = %d, want %d", in, got, want)
		}
	}
}

// TestFindingValuesAreSanitisedAndCapped: an echoed value carries no control
// byte and is capped, as the mode store's echo is.
func TestFindingValuesAreSanitisedAndCapped(t *testing.T) {
	tab := wellBuiltTab()
	tab.Header = "\x1b[31mProduct\x1b[0m " + strings.Repeat("Q", 200)
	fs := CheckLimits(one(tab), Default, facilitator())
	onlyRule(t, fs, RuleHeader)
	for _, f := range fs {
		if strings.ContainsRune(f.Value, '\x1b') {
			t.Errorf("value carries an escape: %q", f.Value)
		}
		if len(f.Value) > echoCap {
			t.Errorf("value is %d bytes, over the %d cap", len(f.Value), echoCap)
		}
	}
}

// TestRaisingALimitInOnePlaceMovesTheCheck is the core half of A7: a copy of
// Default with LabelWords raised from five to six admits a six-word label the
// default refuses, with no other edit.
func TestRaisingALimitInOnePlaceMovesTheCheck(t *testing.T) {
	tab := wellBuiltTab()
	tab.Options[0].Label = "Keep the role and number too"
	onlyRule(t, CheckLimits(one(tab), Default, facilitator()), RuleLabelWords)
	l := Default
	l.LabelWords = 6
	if fs := CheckLimits(one(tab), l, facilitator()); len(fs) != 0 {
		t.Errorf("six-word label refused at LabelWords 6:\n%s", render(fs))
	}
}

// TestFindingsComeAllAtOnce: one call returns every finding, across tabs, so
// the agent fixes them in one retry; each names its own tab.
func TestFindingsComeAllAtOnce(t *testing.T) {
	a, b := wellBuiltTab(), wellBuiltTab()
	a.Header = ""
	b.Options[0].Label = "Keep it (Recommended)"
	b.Options[1].Description = ""
	fs := CheckLimits(Fields{Tabs: []Tab{a, b}}, Default, facilitator())
	if len(fs) != 3 || fs[0].Tab != 1 || fs[1].Tab != 2 || fs[2].Tab != 2 {
		t.Errorf("want three findings across tabs 1 and 2, got:\n%s", render(fs))
	}
}

// TestChipRole reads the role word from a header in the chip grammar.
func TestChipRole(t *testing.T) {
	for h, want := range map[string]string{"Product Q2": "Product", "Setup Q1/4": "Setup", "Tech Q10": "Tech"} {
		if got, ok := ChipRole(h, Default); !ok || got != want {
			t.Errorf("ChipRole(%q) = %q, %v; want %q", h, got, ok, want)
		}
	}
	for _, h := range []string{"", "Pick a theme", "Product", "Product Q0", "product Q2", "Product Q2/", "Builder Q1"} {
		if _, ok := ChipRole(h, Default); ok {
			t.Errorf("ChipRole(%q) read a role", h)
		}
	}
}

// TestCheckReadsOnlyTheTabsAndOptionsItCounts: a call carrying more tabs than
// QuestionsPerCall allows, or a tab more options than OptionsPerQ, is refused on
// the count, and only the first tabs and options the limits allow are checked
// field by field, so the findings stay bounded however many the payload holds
// (review-askGuard-security finding 1).
func TestCheckReadsOnlyTheTabsAndOptionsItCounts(t *testing.T) {
	tabs := make([]Tab, 40000)
	for i := range tabs {
		tabs[i] = Tab{Header: "Product Q1"}
	}
	fs := CheckLimits(Fields{Tabs: tabs}, Default, unnamed())
	if len(fs) == 0 || fs[0].Rule != RuleQuestionsPerCall {
		t.Fatalf("want the count finding first; got %v", rulesOf(fs[:min(len(fs), 5)]))
	}
	for _, f := range fs {
		if f.Tab > Default.QuestionsPerCall[1] {
			t.Fatalf("tab %d was checked past the %d the limits allow (%d findings)", f.Tab, Default.QuestionsPerCall[1], len(fs))
		}
	}

	tab := wellBuiltTab()
	opts := make([]Choice, 30000)
	for i := range opts {
		opts[i] = Choice{Label: "a (Recommended)", Description: "**b**"}
	}
	tab.Options = opts
	fs = CheckLimits(one(tab), Default, unnamed())
	if !slices.Contains(rulesOf(fs), RuleOptions) {
		t.Fatalf("want the options count finding; got %v", rulesOf(fs))
	}
	for _, f := range fs {
		if !strings.HasPrefix(f.Part, "option ") {
			continue
		}
		n, err := strconv.Atoi(strings.Fields(f.Part)[1])
		if err != nil || n > Default.OptionsPerQ[1] {
			t.Fatalf("%q was checked past the %d options the limits allow (%d findings)", f.Part, Default.OptionsPerQ[1], len(fs))
		}
	}

	// The decide-later option is judged against the payload's last option,
	// not the last one checked: a flood that ends in it is refused on the
	// count alone.
	tab = wellBuiltTab()
	later := tab.Options[2]
	tab.Options = slices.Repeat(tab.Options[:2], 15000)
	tab.Options = append(tab.Options, later)
	onlyRule(t, CheckLimits(one(tab), Default, unnamed()), RuleOptions)
}

// TestAWordWiderThanTheMeasureCountsItsRows: the host hard-wraps a word wider
// than its text measure, so the row estimate counts the rows it fills, and a
// 400-column URL in a description is refused at the limits that refuse the
// same bytes broken by spaces (review-askGuard-security finding 2).
func TestAWordWiderThanTheMeasureCountsItsRows(t *testing.T) {
	if got := blockRows(strings.Repeat("x", 400), 76); got != 6 {
		t.Errorf("a 400-column word at 76 columns = %d rows, want 6", got)
	}
	url := "https://example.com/" + strings.Repeat("a", 380)
	spaced := []byte(url)
	for i := 29; i < len(spaced); i += 10 {
		spaced[i] = ' '
	}
	long, words := wellBuiltTab(), wellBuiltTab()
	long.Options[0].Description = url
	words.Options[0].Description = string(spaced)
	l := Default
	l.Rows = estimateRows(words, l) - 1
	onlyRule(t, CheckLimits(one(words), l, facilitator()), RuleRows)
	onlyRule(t, CheckLimits(one(long), l, facilitator()), RuleRows)
}
