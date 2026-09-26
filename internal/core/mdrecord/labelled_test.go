package mdrecord

import (
	"strings"
	"testing"
)

// TestLabelledParagraphFindsTheFirstLiveStatement pins the one reading of a
// labelled paragraph the principles lint and the reading assembler share
// (spc-2609020626042471): the first live paragraph opening with the label, to
// the next blank line, never one inside a fence or continuing a paragraph.
func TestLabelledParagraphFindsTheFirstLiveStatement(t *testing.T) {
	doc := strings.Join([]string{
		"# A principle", // 0
		"",
		"```",
		"**The rule.** A fenced example, never the statement.", // 3
		"```",
		"",
		"Prose that mentions",
		"**The rule.** mid-paragraph, which is not a paragraph of its own.", // 7
		"",
		"**The rule.** The statement,", // 9
		"over two lines.",
		"",
		"**Why.** The reasons.",
		"",
		"**The rule.** A second statement is never read.",
	}, "\n")
	lines := strings.Split(doc, "\n")
	start, end, ok := LabelledParagraph(lines, "The rule")
	if !ok || start != 9 || end != 11 {
		t.Fatalf("LabelledParagraph = %d, %d, %v; want 9, 11, true", start, end, ok)
	}
	if _, _, ok := LabelledParagraph(lines, "Bounds"); ok {
		t.Error("a label the document does not carry was found")
	}
	if s, e, ok := LabelledParagraph(lines, "Why"); !ok || s != 12 || e != 13 {
		t.Errorf("the Why paragraph = %d, %d, %v; want 12, 13, true", s, e, ok)
	}
}
