package spec

import (
	"reflect"
	"testing"
)

// TestOpenQuestionsPinsEachQuestionsLine: a question is reported on the line
// its item opens on, for a bullet, a numbered item and an item under a
// sub-heading, and a settled item is skipped.
func TestOpenQuestionsPinsEachQuestionsLine(t *testing.T) {
	content := "---\nid: spc-1\n---\n# T\n\n## Open Questions\n\n" +
		"- Which cache?\n" + // line 8
		"  It is read on every turn.\n" +
		"2. Which size?\n" + // line 10
		"- resolved: LRU, in dcn-1.\n" + // settled
		"\n### Relay\n\n" +
		"- Which relay?\n" + // line 15
		"\n## Decisions\n\n- d\n"
	want := []Question{{Line: 8, Text: "Which cache?"}, {Line: 10, Text: "Which size?"}, {Line: 15, Text: "Which relay?"}}
	if got := OpenQuestions(content); !reflect.DeepEqual(got, want) {
		t.Fatalf("OpenQuestions = %+v, want %+v", got, want)
	}
}

// TestOpenQuestionsReadsWhatThePageShows: the section's items and their
// settled markers are read as the page shows them.
func TestOpenQuestionsReadsWhatThePageShows(t *testing.T) {
	const head = "# T\n\n"
	cases := []struct {
		name string
		body string
		want []int // the lines questions open on
	}{
		{"a settled marker in a comment settles nothing", "## Open Questions\n\n- Which cache?\n  <!--\n  resolved: placeholder\n  -->\n", []int{5}},
		{"a settled marker in a one-line comment settles nothing", "## Open Questions\n\n- Which cache? <!-- resolved: placeholder -->\n", []int{5}},
		{"a settled marker in a fence settles nothing", "## Open Questions\n\n- Which cache?\n\n  ```\n  resolved: placeholder\n  ```\n", []int{5}},
		{"a bullet in a fence is not a question", "## Open Questions\n\n```\n- an example\n```\n", nil},
		{"a bullet in a comment is not a question", "## Open Questions\n\n<!--\n- parked\n-->\n", nil},
		{"a plus bullet", "## Open Questions\n\n+ Which cache?\n", []int{5}},
		{"a bullet one to three columns in", "## Open Questions\n\n   - Which cache?\n", []int{5}},
		{"a sibling nearer the margin than the content", "## Open Questions\n\n1.  Which cache?\n  - Which size?\n", []int{5, 6}},
		{"an item whose text starts on the next line", "## Open Questions\n\n-\n  Which cache?\n", []int{5}},
		{"a line left of the content after a blank settles nothing", "## Open Questions\n\n- Which cache?\n\n resolved: placeholder\n", []int{5}},
		{"a heading in lower case", "## Open questions\n\n- Which cache?\n", []int{5}},
		{"an indented heading with a closing run", "  ## Open Questions ##\n\n- Which cache?\n", []int{5}},
		{"a comment opened after text runs past a blank line", "Text <!-- note\n\n## Open Questions\n\n- which cache?\n\n-->\n", []int{7}},
		{"a settled section in a comment settles nothing", "## Open Questions\n\n<!-- _All resolved._ -->\n- Which cache?\n", []int{6}},
		{"an empty item", "## Open Questions\n\n-\n\n- Which cache?\n", []int{5, 7}},
		{"a thematic break of stars", "## Open Questions\n\n* * *\n", []int{5}},
		{"a two-column bullet under a numbered item", "## Open Questions\n\n1. Which cache?\n  - Which size?\n", []int{5, 6}},
		{"a numbered item's marker left of its content after a blank", "## Open Questions\n\n1. Which cache?\n\n  resolved: placeholder\n", []int{5}},
		{"an indented heading ends the section", "## Open Questions\n\n- Which cache?\n\n   ## Decisions\n\n- decided\n", []int{5}},
		{"a setext heading opens no section", "Open Questions\n--------------\n\n- Which cache?\n", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got []int
			for _, q := range OpenQuestions(head + c.body) {
				got = append(got, q.Line)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("question lines = %v, want %v", got, c.want)
			}
		})
	}
}
