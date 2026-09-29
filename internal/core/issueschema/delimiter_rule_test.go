package issueschema

import "testing"

// TestParseDispositionOpensOnTheOneDelimiterRule: a disposition's block is read
// on frontmatter.Close's terms, the strict ledger parser's, so an indented
// opening rule is no block at all — the record is not well-formed, and its
// fields are not trusted (iss-2608270908348042).
func TestParseDispositionOpensOnTheOneDelimiterRule(t *testing.T) {
	t.Parallel()
	rec := ParseDisposition("dsp-1", "  ---\nstate: held\n---\n")
	if rec.WellFormed || rec.State != "" {
		t.Fatalf("ParseDisposition read an indented opener as a block: %+v", rec)
	}
	rec = ParseDisposition("dsp-1", "---\nstate: held\n---\n")
	if rec.State != "held" {
		t.Fatalf("ParseDisposition = %+v, want state held", rec)
	}
}
