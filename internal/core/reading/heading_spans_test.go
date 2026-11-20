package reading

import (
	"strings"
	"testing"
)

// A raw heading opener written as code is text to a renderer, so it opens no
// heading and its title is never read (iss-2611040555367558). The scan read an
// opening `<h1>` inside a code span as a raw heading, took everything from the
// tag to the end of the file as its title, and refused every assembly while
// quoting most of the file in the refusal.

// codeSpanTail is a document's remainder below the code: a paragraph naming the
// excluded heading beside a stray bracket, which the backstop refuses when it
// is read as part of a title, then enough prose to make an unbounded title long.
var codeSpanTail = "\n\n## Fixed\n\n- The Audit Notes section [of a spec.\n\n" +
	strings.Repeat("A line of changelog prose that goes on.\n", 2000)

// TestARawHeadingOpenerInACodeSpanIsNoHeading: an opener inside an inline code
// span, of any backtick run length, opens nothing, and neither the title scan
// nor the unbounded-title scan reads one there.
func TestARawHeadingOpenerInACodeSpanIsNoHeading(t *testing.T) {
	rows := map[string]string{
		"a one-backtick span":        "# Changelog\n\n- The page renders `<h1>` as text." + codeSpanTail,
		"a two-backtick span":        "# Changelog\n\n- The page renders ``<h2 id=\"x\">`` as text." + codeSpanTail,
		"a span holding a backtick":  "# Changelog\n\n- The page renders `` `<h3>` `` as text." + codeSpanTail,
		"a span with no bound after": "# A page\n\nThe `<h2>` tag renders as text",
		"a span opening a role":      "# A page\n\nThe `<div role=\"heading\">` element." + codeSpanTail,
	}
	for what, doc := range rows {
		if err := refuses(t, "CHANGELOG.md", doc, nil, refusalHeadings); err != nil {
			t.Errorf("%s was read as a raw heading: %.300v", what, err)
		}
	}
}

// TestARawHeadingOpenerInAFencedBlockIsNoHeading: an opener on a fenced line is
// an example, under either fence character.
func TestARawHeadingOpenerInAFencedBlockIsNoHeading(t *testing.T) {
	rows := map[string]string{
		"a backtick fence": "# Changelog\n\n```html\n<h1>\n```" + codeSpanTail,
		"a tilde fence":    "# Changelog\n\n~~~\n<h2 class=\"x\">\n~~~" + codeSpanTail,
	}
	for what, doc := range rows {
		if err := refuses(t, "CHANGELOG.md", doc, nil, refusalHeadings); err != nil {
			t.Errorf("an opener in %s was read as a raw heading: %.300v", what, err)
		}
	}
}

// TestARawHeadingBesideCodeIsStillFound: skipping code skips nothing else. A
// raw heading outside a span is found on a line that holds one, after a stray
// backtick, and where a renderer reads the backticks as raw HTML's text rather
// than as a span: inside an HTML block, including one opened after a list
// marker. A span the page may pair over a line break is not trusted either.
func TestARawHeadingBesideCodeIsStillFound(t *testing.T) {
	rows := map[string]string{
		"a heading after a span":         "# A page\n\nSee `code` then <h2>Audit Notes</h2>\n\n" + sentinelAuditNotes + "\n",
		"a heading after a stray tick":   "# A page\n\nA stray ` then <h2>Audit Notes</h2>\n\n" + sentinelAuditNotes + "\n",
		"a span inside an HTML block":    "# A page\n\n<div>\n`<h2>Audit Notes</h2>`\n</div>\n\n" + sentinelAuditNotes + "\n",
		"a span on an HTML block's line": "# A page\n\n<div>`<h2>Audit Notes</h2>`</div>\n\n" + sentinelAuditNotes + "\n",
		"a span in a list item's block":  "# A page\n\n- <div>\n  `<h2>Audit Notes</h2>`\n  </div>\n\n" + sentinelAuditNotes + "\n",
		"a span on a list item's block":  "# A page\n\n- <div>`<h2>Audit Notes</h2>`</div>\n\n" + sentinelAuditNotes + "\n",
		"a run left open above":          "# A page\n\nA run ` opens here\n<h2>Audit Notes</h2> and ` closes\n\n" + sentinelAuditNotes + "\n",
	}
	for what, doc := range rows {
		err := refuses(t, "spc-x.md", doc, nil, refusalHeadings)
		if err == nil {
			t.Errorf("%s was admitted; the raw heading under it travels", what)
			continue
		}
		if !strings.Contains(err.Error(), "as raw HTML") {
			t.Errorf("%s was refused for another reason: %v", what, err)
		}
	}
	// And a real opener with no bound is still refused as one.
	if err := refuses(t, "spc-x.md", "# A page\n\n<h2>An ordinary heading", nil, refusalHeadings); err == nil ||
		!strings.Contains(err.Error(), "never closed") {
		t.Errorf("an unbounded real opener was not refused as one: %v", err)
	}
}

// TestARawHeadingRefusalBoundsTheQuotedTitle: a title read over most of the
// document is named in the refusal by its opening words, never quoted whole.
func TestARawHeadingRefusalBoundsTheQuotedTitle(t *testing.T) {
	doc := "# A page\n\n<h2>Audit Notes [x\n" + strings.Repeat("A line of prose that goes on.\n", 2000) + "\n\nend\n"
	err := refuses(t, "spc-x.md", doc, nil, refusalHeadings)
	if err == nil {
		t.Fatal("a raw heading naming the excluded one was admitted")
	}
	msg := err.Error()
	if len(msg) > 400 {
		t.Errorf("the refusal quotes %d bytes; the title it names is not bounded: %.300s", len(msg), msg)
	}
	for _, want := range []string{"spc-x.md", "Audit Notes", "line 3", "as raw HTML"} {
		if !strings.Contains(msg, want) {
			t.Errorf("the refusal does not name %q: %.400s", want, msg)
		}
	}
}
