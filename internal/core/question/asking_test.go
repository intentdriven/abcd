package question

import (
	"slices"
	"strings"
	"testing"
)

// joined is the whole asking-rule text as one string, for phrase checks.
func joined(rules []string) string { return strings.Join(rules, "\n") }

// TestAskingRulesFillEveryLimitFromTheirArgument is the GRILL half of
// criterion A7: every limit the rule text states is read from the Limits it is
// handed, so one edit to a limit moves its statement in the text. Each case
// changes one limit on a copy of Default and asserts the old statement goes and
// the new one appears.
func TestAskingRulesFillEveryLimitFromTheirArgument(t *testing.T) {
	base := joined(AskingRules(Default))
	for _, c := range []struct {
		name     string
		edit     func(*Limits)
		was, now string
	}{
		{"label words", func(l *Limits) { l.LabelWords = 6 }, "at most five words", "at most six words"},
		{"header columns", func(l *Limits) { l.HeaderColumns = 14 }, "at most twelve columns", "at most fourteen columns"},
		{"options", func(l *Limits) { l.OptionsPerQ = [2]int{3, 5} }, "two to four options", "three to five options"},
		{"tabs", func(l *Limits) { l.QuestionsPerCall = [2]int{1, 3} }, "up to four on a screen", "up to three on a screen"},
		{"meaning sentences", func(l *Limits) { l.MeaningSentences = 3 }, "at most two sentences", "at most three sentences"},
		{"rows", func(l *Limits) { l.Rows = 30 }, "twenty-four rows", "thirty rows"},
		{"columns", func(l *Limits) { l.Columns = 90 }, "at eighty columns", "at ninety columns"},
		{"chip roles", func(l *Limits) { l.ChipRoles = []string{"Owner", "Builder"} }, `"Product Q2"`, `"Owner Q2"`},
		{"later labels", func(l *Limits) { l.LaterLabels = []string{"Not yet"} }, `"Decide later"`, `"Not yet"`},
		{"now prefix", func(l *Limits) { l.NowPrefix = "Today:" }, `"Now:"`, `"Today:"`},
		{"change-later prefix", func(l *Limits) { l.ChangeLaterPrefix = "Later:" }, `"Change later:"`, `"Later:"`},
		{"not applicable", func(l *Limits) { l.NotApplicable = "n/a" }, `"not applicable"`, `"n/a"`},
	} {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(base, c.was) {
				t.Fatalf("the text from Default does not state %q:\n%s", c.was, base)
			}
			l := Default
			l.ChipRoles = slices.Clone(Default.ChipRoles)
			l.LaterLabels = slices.Clone(Default.LaterLabels)
			c.edit(&l)
			got := joined(AskingRules(l))
			if strings.Contains(got, c.was) {
				t.Errorf("after the edit the text still states %q", c.was)
			}
			if !strings.Contains(got, c.now) {
				t.Errorf("after the edit the text does not state %q:\n%s", c.now, got)
			}
		})
	}
	if again := joined(AskingRules(Default)); again != base {
		t.Fatal("an edit to a copy of Default changed the text Default renders")
	}
}

// TestAskingRulesWriteNumbersAsWords: the limits are written as words in the
// prose (the spec's "numbers written as words"), so the text reads as rules
// rather than a table of figures.
func TestAskingRulesWriteNumbersAsWords(t *testing.T) {
	text := joined(AskingRules(Default))
	for _, digits := range []string{"at most 5", "at most 12", "2 to 4", "24 rows", "80 columns", "up to 4"} {
		if strings.Contains(text, digits) {
			t.Errorf("the text states a limit in digits, %q", digits)
		}
	}
	for n, want := range map[int]string{0: "zero", 1: "one", 5: "five", 12: "twelve", 19: "nineteen", 20: "twenty", 24: "twenty-four", 80: "eighty", 99: "ninety-nine", 100: "100", -1: "-1"} {
		if got := numberWord(n); got != want {
			t.Errorf("numberWord(%d) = %q, want %q", n, got, want)
		}
	}
}

// TestAskingRulesSayTheRowsIncludeTheOptions: the rows a question fits are
// the whole tab's, its options and their descriptions included, and the rule
// says so, or the asker budgets them for the question text alone
// (iss-2610071538055431).
func TestAskingRulesSayTheRowsIncludeTheOptions(t *testing.T) {
	text := joined(AskingRules(Default))
	const want = "fits twenty-four rows at eighty columns, counting the header, the host's frame, the question text, and every option's label and description"
	if !strings.Contains(text, want) {
		t.Errorf("the rule text does not say %q:\n%s", want, text)
	}
}

// TestAskingRecallIsTheGrillOverridesTerms: the recall terms are the ones the
// repository override carried before the domain was generated (itd-201
// decision 9 accepts their cost), and a caller cannot change them for the next.
func TestAskingRecallIsTheGrillOverridesTerms(t *testing.T) {
	want := []string{"grill", "interview", "options", "decide", "decision", "plan", "choose", "which", "product thinker", "facilitator", "register"}
	got := AskingRecall()
	if !slices.Equal(got, want) {
		t.Fatalf("AskingRecall() = %q, want %q", got, want)
	}
	got[0] = "changed"
	if AskingRecall()[0] != "grill" {
		t.Fatal("a caller's edit to the returned terms reached the next caller")
	}
}

// TestAskingRulesStateTheRowsExceptionAndTheDrafter: the layout rule says
// what the check does with a question over the rows limit — it is shown, and
// the agent is told afterwards — rather than claiming the check refuses it,
// and one rule sends every abcd question through the drafter agent, which
// counts rows as the check does. Neither names a record or a source-tree
// command (itd-201 decision 8).
func TestAskingRulesStateTheRowsExceptionAndTheDrafter(t *testing.T) {
	text := joined(AskingRules(Default))
	for _, want := range []string{
		"The rows limit is the one the check does not refuse on",
		"a question over it is shown, and the agent is told afterwards to keep the next question within it",
		"abcd:question-drafter",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the asking rules do not state %q:\n%s", want, text)
		}
	}
	drafter := ""
	for _, r := range AskingRules(Default) {
		if strings.Contains(r, "abcd:question-drafter") {
			if drafter != "" {
				t.Errorf("more than one rule names the drafter:\n%s\n%s", drafter, r)
			}
			drafter = r
		}
	}
	for _, want := range []string{"the material", "whom the question is for", "answers"} {
		if !strings.Contains(drafter, want) {
			t.Errorf("the drafter rule must say what to hand it (%q): %s", want, drafter)
		}
	}
	for _, banned := range []string{"iss-", "itd-", "spc-", "adr-", "go run", "make "} {
		if strings.Contains(text, banned) {
			t.Errorf("the asking rules name %q; they reach repositories that have no record and no source tree", banned)
		}
	}
}
