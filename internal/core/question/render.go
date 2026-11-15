package question

import (
	"fmt"
	"io"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// The grouped refusal and its one renderer (spc-2610301917245442, step 2;
// itd-2610230756259215). The guard hook's deny and the text render of
// `abcd guard question` name findings through GroupFindings and RenderGroups,
// so a line the check shows is the line the hook refuses with, byte for byte.
//
// A rule can break many times on one tab: rule 12 adds one finding per line of
// question text, so a thousand dashed list lines are a thousand findings. A
// group is one rule on one tab, so the lines a refusal takes follow the rules
// and the tabs the check reads, not the payload, and no refusal needs a cut.

// Group is one rule's findings on one tab: the first of them, whose part,
// echoed value, limit and remedy its line names, and how many there are.
type Group struct {
	Tab   int
	Rule  Rule
	First Finding
	Count int
}

// GroupFindings folds findings into one group per rule and tab, in the order
// the first finding of each appeared, keeping that finding and a count.
func GroupFindings(findings []Finding) []Group {
	type key struct {
		tab  int
		rule Rule
	}
	var out []Group
	at := map[key]int{}
	for _, f := range findings {
		k := key{f.Tab, f.Rule}
		if i, ok := at[k]; ok {
			out[i].Count++
			continue
		}
		at[k] = len(out)
		out = append(out, Group{Tab: f.Tab, Rule: f.Rule, First: f, Count: 1})
	}
	return out
}

// String is the group's one line: the first finding's line, closed by how
// many more such lines the rule drew on the tab when it broke more than once,
// through the terminal mask.
func (g Group) String() string {
	line := g.First.String()
	switch more := g.Count - 1; {
	case more == 1:
		line += " (and 1 more such line)"
	case more > 1:
		line += fmt.Sprintf(" (and %d more such lines)", more)
	}
	return termsafe.Sanitize(line)
}

// RenderGroups writes one line per group, in order. It is the one writer of a
// finding line: the hook's deny and the verb's text render both call it.
func RenderGroups(w io.Writer, groups []Group) {
	for _, g := range groups {
		fmt.Fprintln(w, g.String())
	}
}
