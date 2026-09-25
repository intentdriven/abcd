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

// An HTML block swallows a fence opener written directly under it: to a
// CommonMark renderer the block runs to the first blank line, so the delimiter
// is raw HTML text and the heading after that blank line is LIVE — while every
// mdrecord reading sees a fence and masks the heading as an example. Both halves
// of the floor agreed on the wrong answer and the section travelled. Which reading
// a reader of the bundle takes is exactly the ambiguity the floor may not guess
// at, so it refuses and names the opener.
func TestVerifyRedactionRefusesAFenceAnHTMLBlockSwallows(t *testing.T) {
	for name, tc := range map[string]struct {
		doc  string
		line string
	}{
		"a div block over a backtick fence": {"# A spec\n\n<div>\n```\n\n## Private Notes\n\nsecret\n```\n", "line 4"},
		"a span block over a tilde fence":   {"# A spec\n\n<span>\n~~~\n\n## Private Notes\n\nsecret\n~~~\n", "line 4"},
		"a block continued above the fence": {"# A spec\n\n<div class=\"x\">\ntext\n```\n\n## Private Notes\n\nsecret\n```\n", "line 5"},
		"a declaration left open":           {"# A spec\n\n<!DOCTYPE html\n```\n\n## Private Notes\n\nsecret\n```\n", "line 4"},
		"a processing instruction":          {"# A spec\n\n<?php\n```\n\n## Private Notes\n\nsecret\n```\n", "line 4"},
		"a closing tag over the fence":      {"# A spec\n\n</div>\n```\n\n## Private Notes\n\nsecret\n```\n", "line 4"},
	} {
		out, err := redactExcluded("spc-x.md", tc.doc, privateNotes)
		if err == nil {
			t.Errorf("%s: admitted (secret travelled: %v):\n%s", name, strings.Contains(out, "secret"), out)
			continue
		}
		if !strings.Contains(err.Error(), tc.line) {
			t.Errorf("%s: the refusal does not name the opener (%s): %v", name, tc.line, err)
		}
	}
}

// The anti-vacuity half: a blank line ends the HTML block, so a fence below it
// is a fence to every reader, and a fenced example there is still admitted.
func TestVerifyRedactionAdmitsAFenceAfterAClosedHTMLBlock(t *testing.T) {
	for name, doc := range map[string]string{
		"a blank line after the block":   "# A spec\n\n<div>\n\n```md\n## Private Notes\n```\n\nprose\n",
		"prose that only mentions a tag": "# A spec\n\nsee the <div> element\n\n```md\n## Private Notes\n```\n",
		// These blocks end on their own line (CommonMark types 2 and 4), so the
		// fence directly below is a fence to every reader.
		"a one-line comment above":   "# A spec\n\n<!-- note -->\n```md\n## Private Notes\n```\n",
		"a one-line declaration":     "# A spec\n\n<!DOCTYPE html>\n```md\n## Private Notes\n```\n",
		"a multi-line comment above": "# A spec\n\n<!--\nnote\n-->\n```md\n## Private Notes\n```\n",
	} {
		if _, err := redactExcluded("spc-x.md", doc, privateNotes); err != nil {
			t.Errorf("%s: a fenced example below a closed HTML block was refused: %v", name, err)
		}
	}
}
