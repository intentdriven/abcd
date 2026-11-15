package question

import (
	"strings"
	"testing"
)

// The grouped refusal (spc-2610301917245442, step 2): findings fold into one
// group per rule and tab, and one renderer writes the line each group takes,
// so the hook's deny and the verb's text render cannot word a finding apart.

// TestGroupFindingsIsOneRuleOneTab: findings of one rule on one tab fold into
// one group, in the order the first of each appeared, keeping the first
// finding's part, value, limit and remedy and counting the rest; the same
// rule on another tab is its own group. Each group renders as one line, and
// only a group of more than one finding says how many more it holds.
func TestGroupFindingsIsOneRuleOneTab(t *testing.T) {
	f := func(tab int, part string, r Rule, value string) Finding {
		return Finding{Tab: tab, Part: part, Rule: r, Value: value, Limit: "the limit", Remedy: "Fix it."}
	}
	findings := []Finding{
		f(0, "questions", RuleQuestionsPerCall, "5 questions"),
		f(1, "option 1 description", RuleMeaningSentences, "first"),
		f(1, "question text", RuleEmDashListItem, "- a \u2014 one"),
		f(1, "option 2 description", RuleMeaningSentences, "second"),
		f(2, "option 1 description", RuleMeaningSentences, "other tab"),
		f(1, "question text", RuleEmDashListItem, "- a \u2014 two"),
		f(1, "question text", RuleEmDashListItem, "- a \u2014 three"),
	}
	groups := GroupFindings(findings)
	want := []struct {
		tab   int
		rule  Rule
		part  string
		value string
		count int
	}{
		{0, RuleQuestionsPerCall, "questions", "5 questions", 1},
		{1, RuleMeaningSentences, "option 1 description", "first", 2},
		{1, RuleEmDashListItem, "question text", "- a \u2014 one", 3},
		{2, RuleMeaningSentences, "option 1 description", "other tab", 1},
	}
	if len(groups) != len(want) {
		t.Fatalf("groups = %+v, want %d groups", groups, len(want))
	}
	total := 0
	for i, w := range want {
		g := groups[i]
		if g.Tab != w.tab || g.Rule != w.rule || g.First.Part != w.part || g.First.Value != w.value || g.Count != w.count {
			t.Errorf("group %d = %+v, want tab %d rule %s part %q value %q count %d", i, g, w.tab, w.rule, w.part, w.value, w.count)
		}
		total += g.Count
	}
	if total != len(findings) {
		t.Errorf("the groups count %d findings, want all %d", total, len(findings))
	}

	var b strings.Builder
	RenderGroups(&b, groups)
	lines := strings.Split(strings.TrimRight(b.String(), "\n"), "\n")
	if len(lines) != len(groups) {
		t.Fatalf("rendered %d lines for %d groups:\n%s", len(lines), len(groups), b.String())
	}
	for i, g := range groups {
		if !strings.HasPrefix(lines[i], g.First.String()) {
			t.Errorf("line %d does not open with the first finding's line %q:\n%s", i, g.First.String(), lines[i])
		}
		more := strings.Contains(lines[i], "more such line")
		if more != (g.Count > 1) {
			t.Errorf("line %d (count %d) says more: %v:\n%s", i, g.Count, more, lines[i])
		}
	}
	if !strings.HasSuffix(lines[1], "(and 1 more such line)") || !strings.HasSuffix(lines[2], "(and 2 more such lines)") {
		t.Errorf("the count of the rest is not stated:\n%s", b.String())
	}
	if GroupFindings(nil) != nil {
		t.Error("no findings make no groups")
	}
}

// TestRenderGroupsSanitisesEachLine (invariant 13): a group's line passes the
// terminal mask, so a value carrying a control or bidirectional rune is never
// written raw, whichever caller renders it.
func TestRenderGroupsSanitisesEachLine(t *testing.T) {
	g := []Group{{Tab: 1, Rule: RuleNoBold, Count: 1, First: Finding{Tab: 1, Part: "header", Rule: RuleNoBold, Value: "a\x1b[2Jb\u202e", Limit: "l", Remedy: "r"}}}
	var b strings.Builder
	RenderGroups(&b, g)
	for _, r := range strings.TrimRight(b.String(), "\n") {
		if r < 0x20 || r == '\u202e' {
			t.Fatalf("the line carries the unsafe rune %U: %q", r, b.String())
		}
	}
}
