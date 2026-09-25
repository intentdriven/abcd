package reading

import (
	"strings"
	"testing"
)

// A heading nested in a list item or a blockquote renders as the heading it
// is, and the floor read it as prose: its ATX pattern wants the line's first
// non-blank character to be `#`, and the section walk reads column 0 alone. The
// residue was disclosed in a code comment, which is not a recorded deferral;
// the floor now refuses it, however deep the nesting.
func TestVerifyRedactionRefusesAHeadingNestedInAListOrBlockquote(t *testing.T) {
	for name, doc := range map[string]string{
		"a list-prefixed heading":                    "# A spec\n\n- ## Private Notes\n\n  secret\n",
		"an ordered-list-prefixed heading":           "# A spec\n\n1. ## Private Notes\n\n   secret\n",
		"a heading at a list content indent of four": "# A spec\n\n1.  item\n    ## Private Notes\n    secret\n",
		"a blockquoted heading":                      "# A spec\n\n> ## Private Notes\n> secret\n",
		"a heading in a list in a blockquote":        "# A spec\n\n> - ## Private Notes\n>   secret\n",
		"a tab-indented heading under a list item":   "# A spec\n\n- item\n\n\t## Private Notes\n\n\tsecret\n",
	} {
		if out, err := redactExcluded("spc-x.md", doc, privateNotes); err == nil {
			t.Errorf("%s: admitted (secret travelled: %v):\n%s", name, strings.Contains(out, "secret"), out)
		}
	}
	// The anti-vacuity half: a nested heading naming anything else, and a
	// bullet that only mentions the excluded title in prose, are admitted.
	for name, doc := range map[string]string{
		"another nested heading": "# A spec\n\n- ## Public Notes\n\n> ## Also Public\n",
		"a bullet naming it":     "# A spec\n\n- see Private Notes below\n",
		"a fenced nested quote":  "# A spec\n\n```md\n> ## Private Notes\n```\n",
	} {
		if _, err := redactExcluded("spc-x.md", doc, privateNotes); err != nil {
			t.Errorf("%s: refused: %v", name, err)
		}
	}
}
