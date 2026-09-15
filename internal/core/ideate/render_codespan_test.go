package ideate

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/site"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// renderMarkdown renders one markdown block through the site renderer, which
// refuses inline HTML outright. It is the strictest reader the record has, so it
// is the right oracle for "did this line land as live markup".
func renderMarkdown(md string) (string, error) {
	r := &site.Renderer{
		UI:    site.UI{Copy: "copy", Copied: "copied"},
		Image: func(src, alt string, _ site.Source) (string, error) { return "", nil },
		Link:  func(href string, _ site.Source) string { return href },
	}
	return r.RenderBlocks("record.md", site.Blocks(md, 1))
}

// TestBlockTextKeepsACleanedCodeSpanIntact pins the contract between the cleaner
// and this document's own escaper.
//
// termsafe's HTML-opener rule exempts code spans because a renderer parses no raw
// HTML inside one — an exemption that is only sound while the field is PARSED as
// the exact string it was CLEANED as. blockText broke that: it escaped a leading
// backtick unconditionally, which kills the span the cleaner had relied on and
// republishes its sheltered content as live markup. `<details>` is the sharp
// case: it renders as a collapsed disclosure widget, so everything after it in the
// record is concealed behind a closed triangle.
//
// The escape exists to stop a leading marker opening a block, and a backtick run
// that opens a BALANCED span opens no block: a backtick fence's info string may
// not contain backticks, so a run with a matching closer on the same line is an
// inline span by construction. Only an UNBALANCED leading run needs the escape —
// and the cleaner no longer emits one.
func TestBlockTextKeepsACleanedCodeSpanIntact(t *testing.T) {
	const raw = "`<details> everything after this is concealed`"
	cleaned := termsafe.CleanProse(raw, 4096)
	got := blockText(cleaned)

	html, err := renderMarkdown(got)
	if err != nil {
		t.Fatalf("blockText(%q) = %q renders as live markup: %v", cleaned, got, err)
	}
	// The refusal above is the gate; this is the positive form of the same claim.
	// The tag must land escaped INSIDE the code element the span produces.
	if !strings.Contains(html, "<code>&lt;details&gt;") {
		t.Errorf("blockText(%q) = %q rendered as %q; want the tag sheltered inside a code span", cleaned, got, html)
	}
	if strings.HasPrefix(got, `\`) {
		t.Errorf("blockText escaped a balanced code span: %q", got)
	}
}

// TestBlockTextStillEscapesAnUnbalancedLeadingRun keeps the escaper armed for the
// shape that does open a block: a leading backtick run with no closer is a code
// fence, and everything after it in the document becomes fenced text.
func TestBlockTextStillEscapesAnUnbalancedLeadingRun(t *testing.T) {
	for _, s := range []string{"```go unclosed fence", "`stray opener"} {
		if got := blockText(s); !strings.HasPrefix(got, `\`) {
			t.Errorf("blockText(%q) = %q, want the leading run escaped", s, got)
		}
	}
}
