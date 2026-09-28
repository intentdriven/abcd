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

// TestOpenQuestionsReadsTheSettledConvention pins the two markers the record
// uses for a settled section, taken from the planned intents that carry them
// (the 2026-09-25 DECISIONS entry): a section that opens with an italic
// "_All resolved …_" line (itd-111, itd-93), and an item explicitly marked
// resolved or deferred (itd-76, itd-60, itd-117, itd-111). It stays
// fail-closed: an item led "Open", an item that only points elsewhere, a
// question that merely mentions deferral, and an opener that does not open the
// section each still count.
func TestOpenQuestionsReadsTheSettledConvention(t *testing.T) {
	t.Parallel()
	const head = "## Open Questions\n\n"
	cases := []struct {
		name    string
		content string
		want    []string
	}{
		{"all resolved or explicitly deferred (itd-111)", head +
			"_All resolved or explicitly deferred at planning (2026-08-15):_\n\n" +
			"- **Sampled re-surfacing** — graduated to its own\n  capture, iss-230.\n" +
			"- **Refusal breadth** — resolved: deliberately narrow.\n", nil},
		{"all four resolved (itd-93)", head +
			"_All four resolved in the 2026-07-24 grill (see DECISIONS.md,\n2026-07-24 entries)._\n\n" +
			"- **Which surface scaffolds it?** RESOLVED: a `launch` sub-verb.\n", nil},
		{"explicitly deferred, bold after the lead (itd-76)", head +
			"- Ledger ownership once work spans machines: **explicitly deferred** (ruling, 2026-08-16).\n", nil},
		{"an explicit deferral split over two lines (itd-111)", head +
			"- **Harness portability of the channel** — **explicit\n  deferral** to the itd-22 lineage.\n", nil},
		{"resolved with a colon (itd-111, itd-93)", head +
			"- **Explicit check naming** — resolved: `abcd version --check`.\n" +
			"- **How much is templated?** RESOLVED: self-scaffold parity.\n", nil},
		{"a bold resolved or deferred lead (itd-60, itd-117)", head +
			"- **Resolved — where the pass hooks.** Two points.\n" +
			"- **Deferred to a follow-up intent**: detecting duplication.\n" +
			"- **Deferred**: whether conventions migrate.\n", nil},
		{"an Open lead is a question whatever follows (itd-60)", head +
			"- **Resolved — what built reality is.** Two layers.\n" +
			"- **Open, and not gating scope** — whether this pass becomes a\n  discipline. Deferred: the answer changes where.\n",
			[]string{"**Open, and not gating scope** — whether this pass becomes a"}},
		{"a pointer elsewhere is not a marker (itd-76)", head +
			"- The share/ingest questions travel with [itd-126](../drafts/itd-126.md).\n",
			[]string{"The share/ingest questions travel with [itd-126](../drafts/itd-126.md)."}},
		{"a question that mentions deferral", head +
			"- Should the check be deferred until the runner ships?\n- **Out of scope, recorded for clarity**: a workspace layer.\n",
			[]string{"Should the check be deferred until the runner ships?", "**Out of scope, recorded for clarity**: a workspace layer."}},
		{"a label mid-sentence is not a marker (iss-2609260932374727)", head +
			"- Which id wins once the split is resolved: the old or the new?\n" +
			"- Once the flag is deferred: who picks it up?\n" +
			"- Which runner?\n  It stays a question until it is resolved: see below.\n",
			[]string{"Which id wins once the split is resolved: the old or the new?",
				"Once the flag is deferred: who picks it up?", "Which runner?"}},
		{"a label opening the item or a continuation line (iss-2609260932374727)", head +
			"- Resolved: the local runner.\n" +
			"- Which runner?\n  Deferred: to the runner intent.\n", nil},
		{"a label after a closing bold and a parenthetical dash (itd-93)", head +
			"- **Relationship to itd-73** (derived versioning) — RESOLVED: the CHANGELOG.\n", nil},
		{"an opener below the first item does not open the section", head +
			"- Which runner?\n\n_All resolved at planning._\n", []string{"Which runner?"}},
		{"an opener that settles only some", head +
			"_All but one resolved at planning:_\n\n- Which runner?\n", []string{"Which runner?"}},
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
