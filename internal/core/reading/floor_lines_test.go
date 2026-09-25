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
