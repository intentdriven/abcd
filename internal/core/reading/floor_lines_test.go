package reading

import (
	"strings"
	"testing"
)

// Two line shapes every renderer reads and the floor's line split did not. A
// byte-order mark before an ATX heading on line 0 is not part of the line to a
// renderer, so the heading is live; and a lone carriage return is a line ending
// (CommonMark 2.1), so a CR-only document is many lines to a renderer and one
// line to a floor that splits on "\n". Each travelled with a nil error.
func TestVerifyRedactionRefusesABOMHeadingAndALoneCarriageReturn(t *testing.T) {
	for name, doc := range map[string]string{
		"a BOM before the excluded heading":  "\uFEFF## Private Notes\n\nsecret\n",
		"a BOM before an underlined heading": "\uFEFFPrivate Notes\n===\n\nsecret\n",
		"a CR-only document":                 "# A spec\r\r## Private Notes\r\rsecret\r",
		"a lone CR inside a CRLF document":   "# A spec\r\n\r\nintro\r## Private Notes\r\n\r\nsecret\r\n",
	} {
		if out, err := redactExcluded("spc-x.md", doc, privateNotes); err == nil {
			t.Errorf("%s: admitted (secret travelled: %v): %q", name, strings.Contains(out, "secret"), out)
		}
	}
	// A CRLF document is not the shape: its CR always stands before a newline.
	if _, err := redactExcluded("spc-x.md", "# A spec\r\n\r\nprose\r\n", privateNotes); err != nil {
		t.Errorf("a CRLF document was refused: %v", err)
	}
}

// A CRLF document is redacted to the bytes its LF twin is redacted to, line
// endings aside, wherever the excluded section sits (iss-2609251600019863).
//
// The redactor splits on "\n", so each line's carriage return is the first half
// of the CRLF pair that ends it. Dropping a section at the END of the document
// drops the newline after the last kept line, and the join left that line's
// carriage return behind with no newline: a lone CR the source does not carry,
// which the verifier then refused as the source's. The pair is dropped whole,
// which is what the LF document already loses there, so the refusal of a lone
// carriage return stays a statement about the source.
func TestACRLFDocumentRedactsLikeItsLFTwinWhereverTheSectionSits(t *testing.T) {
	for name, doc := range map[string]string{
		"the section first": "## Private Notes\r\n\r\nsecret\r\n\r\n## Public\r\n\r\nkept\r\n",
		"the section in the middle": "# A spec\r\n\r\nintro\r\n\r\n## Private Notes\r\n\r\nsecret\r\n\r\n" +
			"## Public\r\n\r\nkept\r\n",
		"the section last":                    "# A spec\r\n\r\nintro\r\n\r\n## Private Notes\r\n\r\nsecret\r\n",
		"the section last, no blank above it": "# A spec\r\nintro\r\n## Private Notes\r\nsecret\r\n",
		"the section last, no final ending":   "# A spec\r\n\r\nintro\r\n\r\n## Private Notes\r\n\r\nsecret",
	} {
		out, err := redactExcluded("spc-x.md", doc, privateNotes)
		if err != nil {
			t.Errorf("%s: refused: %v", name, err)
			continue
		}
		if strings.Contains(out, "secret") {
			t.Errorf("%s: the excluded section travelled: %q", name, out)
		}
		if strings.Contains(strings.ReplaceAll(out, "\r\n", ""), "\r") {
			t.Errorf("%s: the redaction carries a lone carriage return: %q", name, out)
		}
		lf, err := redactExcluded("spc-x.md", strings.ReplaceAll(doc, "\r\n", "\n"), privateNotes)
		if err != nil {
			t.Fatalf("%s: the LF twin was refused: %v", name, err)
		}
		if got := strings.ReplaceAll(out, "\r\n", "\n"); got != lf {
			t.Errorf("%s: the CRLF redaction is %q, and the LF twin's is %q", name, got, lf)
		}
	}

	// The refusal stays true of a source that does carry one: a lone carriage
	// return ending the last KEPT line is the document's own, not the drop's.
	const own = "# A spec\r\n\r\n## Private Notes\r\n\r\nsecret\r\n\r\n# Next\r\n\r\nend\r"
	if _, err := redactExcluded("spc-x.md", own, privateNotes); err == nil ||
		!strings.Contains(err.Error(), "lone carriage return") {
		t.Errorf("a source ending in a lone carriage return was not refused for it: %v", err)
	}
}
