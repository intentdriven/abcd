package question

import (
	"fmt"
	"io"
	"slices"
	"strings"

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
// The register rule's further findings each name another word to replace, so
// its line names their values too (iss-2611252121330106): they are at most one
// per field and one per word below the floor, so the line stays bounded. A
// word below the knowledge floor is grouped apart from a record number or a
// command, and each further word is named with its plain words, so the line
// hands over every replacement the retry needs.

// Group is one rule's findings on one tab (the register rule's words below the
// floor a group of their own): the first of them, whose part, echoed value,
// limit and remedy its line names, how many there are, and for the register
// rule the further findings' values, each once, a word below the floor with
// its plain words.
type Group struct {
	Tab   int
	Rule  Rule
	First Finding
	Count int
	More  []string
}

// GroupFindings folds findings into one group per rule and tab, in the order
// the first finding of each appeared, keeping that finding, a count, and the
// register rule's further values.
func GroupFindings(findings []Finding) []Group {
	type key struct {
		tab   int
		rule  Rule
		floor bool
	}
	var out []Group
	at := map[key]int{}
	for _, f := range findings {
		k := key{f.Tab, f.Rule, f.Plain != ""}
		if i, ok := at[k]; ok {
			out[i].Count++
			v := fmt.Sprintf("%q", f.Value)
			if f.Plain != "" {
				v += fmt.Sprintf(" (say %q)", f.Plain)
			}
			if g := &out[i]; f.Rule == RuleRegister && f.Value != g.First.Value && !slices.Contains(g.More, v) {
				g.More = append(g.More, v)
			}
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
	named := ""
	if len(g.More) > 0 {
		named = ": " + strings.Join(g.More, ", ")
	}
	switch more := g.Count - 1; {
	case more == 1:
		line += " (and 1 more such line" + named + ")"
	case more > 1:
		line += fmt.Sprintf(" (and %d more such lines%s)", more, named)
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
