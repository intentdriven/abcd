package reading

import (
	"strings"
	"testing"
)

// The exclusion floor and fences (iss-2609250955207041). The redactor found
// headings through the site's section walk and the verifier re-checked with a
// second backtick-only toggle, and the two read one shape the same wrong way: a
// four-backtick fence quoting a bare three-backtick line, then the excluded
// heading, then a tilde block holding a three-backtick line. Both toggles saw
// the heading as fenced, the redactor dropped nothing, the verifier agreed, and
// the section travelled under a manifest asserting its refusal.
//
// The redactor now reads by mdrecord through the site walk. The verifier does
// NOT share that reading — two readers that agree because they are one reader
// verify nothing — and is the stricter one: a line is code to it only where
// every mdrecord reading agrees, a comment hides nothing (its text travels),
// and an unclosed fence refuses.

const fenceProbe = "# A spec\n\n" +
	"````md\n```\n````\n\n" +
	"## Private Notes\n\nthe private body line\n\n" +
	"~~~\n```\n~~~\n"

var privateNotes = []Exclusion{{Rule: "probe", Signal: "heading", Detail: "Private Notes"}}

func TestRedactExcludedDropsTheSectionTheFenceProbeHides(t *testing.T) {
	out, err := redactExcluded("spc-x.md", fenceProbe, privateNotes)
	if err == nil && strings.Contains(out, "the private body line") {
		t.Fatalf("the excluded section travelled with a nil error:\n%s", out)
	}
	if err != nil {
		t.Fatalf("the redactor refused a shape it can resolve: %v", err)
	}
}

// TestVerifyRedactionRefusesWhatAnyReadingShowsLive runs the verifier alone, on
// documents that still carry the excluded heading, so each case is the
// verifier's own verdict rather than the redactor's.
func TestVerifyRedactionRefusesWhatAnyReadingShowsLive(t *testing.T) {
	headings := map[string]bool{"Private Notes": true}
	for name, doc := range map[string]string{
		"the fence probe": fenceProbe,
		"a heading after a list item that ended its fence":  "# A spec\n\n- an item\n  ```\n  code\n## Private Notes\n\nsecret\n  ```\n",
		"a heading parked in a comment, whose text travels": "# A spec\n\n<!--\n## Private Notes\n\nsecret\n-->\n",
		"a heading below a tilde fence nothing closes":      "# A spec\n\n~~~\ncode\n\n## Private Notes\n\nsecret\n",
		"a heading below a backtick fence nothing closes":   "# A spec\n\n```\ncode\n\n## Private Notes\n\nsecret\n",
	} {
		if err := verifyRedaction("spc-x.md", doc, doc, nil, headings); err == nil {
			t.Errorf("%s: the verifier admitted a document still carrying the excluded heading:\n%s", name, doc)
		}
	}
}

// TestVerifyRedactionStillAdmitsAFencedExample is the anti-vacuity half: a
// heading inside a fence every reading agrees on is an example, not a field,
// and a floor refusing it refuses the corpus it exists to pass.
func TestVerifyRedactionStillAdmitsAFencedExample(t *testing.T) {
	headings := map[string]bool{"Private Notes": true}
	for name, doc := range map[string]string{
		"backtick": "# A spec\n\n```md\n## Private Notes\n```\n\nprose\n",
		"tilde":    "# A spec\n\n~~~md\n## Private Notes\n~~~\n\nprose\n",
		"quoted":   "# A spec\n\n````md\n```\n## Private Notes\n```\n````\n\nprose\n",
	} {
		if err := verifyRedaction("spc-x.md", doc, doc, nil, headings); err != nil {
			t.Errorf("%s: a fenced example was refused: %v", name, err)
		}
	}
}
