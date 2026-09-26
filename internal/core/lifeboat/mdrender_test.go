package lifeboat

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/site"
)

// The lifeboat half of iss-2609020539188868 (iss-2609251355497247): every
// untrusted field on a line of a markdown file the lifeboat writes goes through
// the file-write cleaner (termsafe.CleanProse), never Sanitize alone; no renderer
// wraps a cleaned value in a delimiter of its own (a code span through
// termsafe.CodeSpan where one is wanted); and a value that opens a block is
// escaped so its leading marker cannot turn it into a heading, list, quote or
// fence. These tests drive each renderer with the hostile value directly, so
// the render's discipline is proved on its own and not only through whatever an
// ingest happened to clean first.

// rawMarkdownHazards are the constructs a cleaned value must never carry live.
var rawMarkdownHazards = []string{"<!--", "<script", "](http"}

func assertNoLiveHazard(t *testing.T, what, md string) {
	t.Helper()
	for _, h := range rawMarkdownHazards {
		if strings.Contains(md, h) {
			t.Errorf("%s carries a live %q from an untrusted field:\n%s", what, h, md)
		}
	}
}

// blockMarkerLeads are line openers that would restructure the document if an
// untrusted value began a block with one.
var blockMarkerLeads = []string{"# ", "- ", "* ", "+ ", "> ", "1. ", "1) ", "```", "~~~", "| ", "---", "==="}

// assertNoLineOpensWith fails when any line of md begins with value — the value
// must reach the line escaped, never as its raw leading marker.
func assertNoLineOpensWith(t *testing.T, what, md, value string) {
	t.Helper()
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(line, value) {
			t.Errorf("%s writes a line opening with the raw untrusted marker %q:\n%s", what, value, md)
		}
	}
}

func TestReviewMDCleansEveryFindingField(t *testing.T) {
	a := ReviewArtefact{
		Verdict: VerdictShip, Mode: ModeDelegated,
		SourceName:     "src<!-- hide",
		ManifestSHA256: "<script>abc",
		Findings: []ReviewFinding{{
			ID:       "fnd-<!--x",
			Severity: "blocker] [forged",
			Finding:  "see [here](http://example.com)",
			Evidence: []string{"brief/<script>.md"},
		}},
	}
	md := renderReviewMD(a)
	assertNoLiveHazard(t, "review .md", md)
	if strings.Contains(md, "[blocker") {
		t.Errorf("review .md wraps a cleaned severity in its own brackets:\n%s", md)
	}
}

func TestPrinciplesMarkdownEscapesALeadingMarker(t *testing.T) {
	for _, lead := range blockMarkerLeads {
		f := PrinciplesFile{Mode: ModeDelegated, Principles: []Principle{{
			ID: "prn-a", Principle: lead + "a principle <!-- hidden", Confidence: ConfidenceHigh,
			Evidence: []string{"brief/[x](http://example.com).md"},
		}}}
		md := renderPrinciplesMarkdown(f)
		assertNoLiveHazard(t, "principles.md", md)
		assertNoLineOpensWith(t, "principles.md", md, lead+"a principle")
	}
}

func TestPressReleaseMarkdownNeverWrapsOrLeadsWithACleanedValue(t *testing.T) {
	for _, lead := range blockMarkerLeads {
		f := PressReleaseFile{
			Mode: ModeDelegated, Headline: "Headline <!-- x",
			Subhead:  lead + "subhead_with_underscores_",
			Body:     lead + "body [a](http://example.com)",
			Quotes:   []PressReleaseQuote{{Text: lead + "quoted <script>", Attribution: "someone <!-- x"}},
			Evidence: []string{"brief/<script>.md"},
		}
		md := renderPressReleaseMarkdown(f)
		assertNoLiveHazard(t, "press-release.md", md)
		if strings.Contains(md, "_"+lead+"subhead") {
			t.Errorf("press-release.md wraps the cleaned subhead in its own emphasis:\n%s", md)
		}
		for _, v := range []string{lead + "subhead", lead + "body", "> " + lead + "quoted"} {
			assertNoLineOpensWith(t, "press-release.md", md, v)
		}
	}
}

func TestBriefSectionDocCleansItsEvidence(t *testing.T) {
	md := string(briefSectionDoc(SectionCoverage{
		Name: "product/press-release", Status: StatusGrounded,
		Evidence: []string{"docs/<script>x.md", "docs/[a](http://example.com).md", "docs/<!--c.md"},
	}))
	assertNoLiveHazard(t, "brief section doc", md)
}

// siteRender renders md through the site renderer, the strictest reader the
// record has: it refuses inline HTML and an unclosed code span outright, so
// it is the oracle for "did this value land as live markup".
func siteRender(t *testing.T, md string) (string, error) {
	t.Helper()
	r := &site.Renderer{
		UI:    site.UI{Copy: "copy", Copied: "copied"},
		Image: func(src, alt string, _ site.Source) (string, error) { return "", nil },
		Link:  func(href string, _ site.Source) string { return href },
	}
	return r.RenderBlocks("lifeboat.md", site.Blocks(md, 1))
}

// TestBlockValueNeverDefinesALinkReference is the two-field attack
// (iss-2609262237352137): one field shaped like a link reference definition,
// another carrying the matching shortcut reference. Left unescaped, the first
// renders as nothing and arms the second as a live link to its destination.
// Every block field of both renderers is driven with it.
func TestBlockValueNeverDefinesALinkReference(t *testing.T) {
	const def, use = "[label]: http://example.com/trap", "see [label] for details"
	docs := map[string]string{
		"principles.md": renderPrinciplesMarkdown(PrinciplesFile{Mode: ModeDelegated, Principles: []Principle{
			{ID: "prn-a", Principle: def, Confidence: ConfidenceHigh},
			{ID: "prn-b", Principle: use, Confidence: ConfidenceHigh},
		}}),
		"press-release.md (subhead)": renderPressReleaseMarkdown(PressReleaseFile{
			Mode: ModeDelegated, Headline: "Headline", Subhead: def, Body: use,
		}),
		"press-release.md (quote)": renderPressReleaseMarkdown(PressReleaseFile{
			Mode: ModeDelegated, Headline: "Headline", Body: use,
			Quotes: []PressReleaseQuote{{Text: def, Attribution: "someone"}},
		}),
	}
	for what, md := range docs {
		assertNoLineOpensWith(t, what, md, "[label]:")
		assertNoLineOpensWith(t, what, md, "> [label]:")
		html, err := siteRender(t, md)
		if err != nil {
			t.Errorf("%s does not render: %v\n%s", what, err, md)
			continue
		}
		if !strings.Contains(html, "label]: http://example.com/trap") {
			t.Errorf("%s: the definition-shaped value was consumed rather than shown:\n%s", what, html)
		}
	}
}

// TestBlockValueKeepsABalancedLeadingCodeSpan (iss-2609262237415400): the
// cleaner shelters a tag inside a code span, which holds only while the value
// is parsed as the string it was cleaned as. Escaping a balanced leading span's
// opening backtick kills the span and republishes the tag as live HTML; only
// an unbalanced leading run opens a fence, and that one is still escaped.
func TestBlockValueKeepsABalancedLeadingCodeSpan(t *testing.T) {
	got := mdBlock("`<details>` everything after this is concealed")
	html, err := siteRender(t, got)
	if err != nil {
		t.Fatalf("mdBlock = %q renders as live markup: %v", got, err)
	}
	if !strings.Contains(html, "<code>&lt;details&gt;") {
		t.Errorf("mdBlock = %q rendered as %q; want the tag sheltered inside a code span", got, html)
	}
	for _, s := range []string{"```go unclosed fence", "`stray opener"} {
		if got := mdBlock(s); !strings.HasPrefix(got, `\`) {
			t.Errorf("mdBlock(%q) = %q, want the unbalanced leading run escaped", s, got)
		}
	}
}
