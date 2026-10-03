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
