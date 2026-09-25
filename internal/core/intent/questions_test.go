package intent

import (
	"reflect"
	"testing"
)

// TestOpenQuestionsCountsListItemsAndNothingElse pins what the build's
// open-question check reads (itd-2609201916151817, criterion 1): a top-level
// list item under `## Open Questions` is a question still asked; the italic
// "none open" line every settled record carries, prose, a blockquote, an
// indented continuation and a list under another heading are not.
func TestOpenQuestionsCountsListItemsAndNothingElse(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		content string
		want    []string
	}{
		{"no section", "# t\n\n## Decisions\n\n- a ruling\n", nil},
		{"settled", "# t\n\n## Open Questions\n\n_None open; decisions 1 to 3 settle them._\n\n## Audit Notes\n\n- note\n", nil},
		{"prose only", "## Open Questions\n\nNone beyond the flagged decision above.\n", nil},
		{"blockquote", "## Open Questions\n\n> - asked at the interview and answered there\n", nil},
		{"bullets", "## Open Questions\n\n- **Where does it live?** Either here\n  or there.\n* Who reads it?\n\n## Acceptance Criteria\n\n- Given x\n",
			[]string{"**Where does it live?** Either here", "Who reads it?"}},
		{"numbered", "## Open Questions\n\n1. Which runner?\n2) Which model?\n", []string{"Which runner?", "Which model?"}},
		{"crlf", "## Open Questions\r\n\r\n- Which one?\r\n", []string{"Which one?"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := OpenQuestions(tc.content); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("OpenQuestions = %q, want %q", got, tc.want)
			}
		})
	}
}
