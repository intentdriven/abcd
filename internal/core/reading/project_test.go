package reading

import (
	"strings"
	"testing"
	"time"
)

// TestRenderEquivalenceIsDeterministic: a determinism instrument cannot have a
// coin-flip refusal. Decoding entities by ranging a Go map made the verdict for
// `Audit&amp;nbsp;Notes` depend on whether `&amp;` happened to be applied before
// `&nbsp;` — the same input, the same repository state, two different answers.
func TestRenderEquivalenceIsDeterministic(t *testing.T) {
	const probe = "Audit&amp;nbsp;Notes"
	first := sameRendering(probe, "Audit Notes")
	for i := range 50 {
		if got := sameRendering(probe, "Audit Notes"); got != first {
			t.Fatalf("call %d gave %v where the first gave %v; the verdict is not a function of the input",
				i, got, first)
		}
	}
}

// TestNumericCharacterReferenceIsRecognised: a numeric or hex reference renders
// as the letter it names, so a title carrying one is the excluded title.
func TestNumericCharacterReferenceIsRecognised(t *testing.T) {
	for _, probe := range []string{"Audit N&#111;tes", "Audit N&#x6f;tes", "&#65;udit Notes"} {
		if !sameRendering(probe, "Audit Notes") {
			t.Errorf("%q does not read as the excluded heading", probe)
		}
	}
}

// TestRenderedTextLeavesAnAutolinkAlone: stripping tags must not eat an autolink,
// which is a URL a heading may legitimately carry.
func TestRenderedTextLeavesAnAutolinkAlone(t *testing.T) {
	got := strings.Join(renderedTexts("See <https://example.invalid/x>"), " ")
	if !strings.Contains(got, "https://example.invalid/x") {
		t.Errorf("the autolink was stripped as a tag: %q", got)
	}
}

// TestAttributeMaskStaysOnItsOwnLine: an attribute value ends on the line it
// opens on. Ending it at the next matching quote found anywhere in the document
// let one unbalanced quote blank every angle bracket up to some unrelated quote
// thousands of bytes later, erasing a raw HTML heading from the masked reading.
//
// The refusal itself no longer turns on this: the heading scan reads the
// unmasked document too and refuses on either reading, so a runaway mask can no
// longer hide a heading end to end. What it can still do is destroy the masked
// reading, which is the reading that sees a `>` written inside an attribute
// value — so the bound is asserted here, on the mask's own contract.
func TestAttributeMaskStaysOnItsOwnLine(t *testing.T) {
	const runaway = "<div id=\"\n\n<h2>Audit Notes</h2>\n\nprose\n\n\">\n"
	got, _ := maskMarkupData(runaway, true)
	if len(got) != len(runaway) {
		t.Fatalf("the mask changed the document length from %d to %d", len(runaway), len(got))
	}
	if !strings.Contains(got, "<h2>Audit Notes</h2>") {
		t.Errorf("the mask crossed the tag its value opened in and erased a heading: %q", got)
	}

	// And it still masks the value it was built for, which closes on its own line.
	const inline = "<h2 title=\"a>b\">Audit Notes</h2>"
	if m, _ := maskMarkupData(inline, true); !strings.Contains(m, "a b") {
		t.Errorf("a greater-than inside a same-line attribute value was left as structure: %q", m)
	}
}

// TestRawHeadingScanStaysLinearInTheOpenerCount: the bound scan materialised
// every candidate bound in the whole remainder of the document for every heading
// opener, though it breaks at the first hard one. That is quadratic with a large
// constant, and a committed markdown file up to the size cap the assembler sets
// did not finish — a silent hang, which is the one staging a fail-closed floor
// cannot afford. The bound is generous: the walk is milliseconds and the
// materialising scan was minutes.
func TestRawHeadingScanStaysLinearInTheOpenerCount(t *testing.T) {
	var b strings.Builder
	b.WriteString("# A spec\n\n")
	for range 8000 {
		b.WriteString("<h2>Ordinary heading</h2>\n")
	}
	doc := b.String()
	headings := map[string]bool{"Audit Notes": true}
	start := time.Now()
	if err := verifyRedaction("spc-x.md", doc, doc, nil, headings); err != nil {
		t.Fatalf("the scan refused an ordinary document: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("the raw heading scan took %s over %d openers; it materialises the whole "+
			"bound list per opener", elapsed, 8000)
	}
}

// TestMaskStaysLinearInTheAssignmentCount: the line bound searched the whole
// remainder of the document for a newline once per attribute assignment, which
// is quadratic — a 4 MiB file of assignments on one line took 21 seconds where
// the walk takes milliseconds, and the size cap bounds one file rather than how
// many of them a repository holds. The cursor onto the next newline only ever
// advances. Measured: 7 ms with the cursor, 12.3 s without it.
func TestMaskStaysLinearInTheAssignmentCount(t *testing.T) {
	var b strings.Builder
	b.WriteString("# S\n\n<a ")
	for range 800000 {
		b.WriteString("=\"x\"")
	}
	doc := b.String()
	start := time.Now()
	if got, _ := maskMarkupData(doc, true); len(got) != len(doc) {
		t.Fatalf("the mask changed the document length from %d to %d", len(doc), len(got))
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("the mask took %s over 800000 assignments on one line; it searches the whole "+
			"remainder for a newline per assignment", elapsed)
	}
}

// The six shapes itd-194 refuses. Each is a markdown document the include table
// admits and the exclusion floor cannot resolve, and each is answered the way
// unresolvableFrontmatterShape already answers a YAML tag or an anchor: a
// refusal naming the document, the line and the shape, never a redaction by
// guess and never a silent admission (adr-56 rule 1; brief invariant 16;
// spc-2609021003136831, "The six refusals").

// excludedKeys and excludedHeadings are the floor's two signal sets as
// verifyRedaction takes them, so a shape test states which half it exercises.
var (
	refusalKeys     = map[string]bool{"origin": true, "production_mode": true}
	refusalHeadings = map[string]bool{"Audit Notes": true}
)

// refuses runs the floor's verifier over one document and returns the refusal.
func refuses(t *testing.T, rel, doc string, keys, headings map[string]bool) error {
	t.Helper()
	return verifyRedaction(rel, doc, doc, keys, headings)
}

// TestAFenceInsideTheFrontmatterRefuses is shape 1 (iss-2608301350533102), the
// set's only critical: the fence mask spanned the frontmatter, so a delimiter
// inside the block toggled the mask and switched off the very key refusal that
// exists to catch a key the field reader cannot see. The block is now located
// before any mask is computed and the mask starts after the block closes, so
// nothing inside the block can toggle it — and the delimiter itself is the
// signal.
func TestAFenceInsideTheFrontmatterRefuses(t *testing.T) {
	const doc = "---\n```\norigin: ABCD-WARM-ORIGIN\n---\n\n# A record\n\nBody.\n"
	err := refuses(t, "spc-1-a-record.md", doc, refusalKeys, refusalHeadings)
	if err == nil {
		t.Fatal("a fence delimiter inside the frontmatter was admitted; it toggles the mask " +
			"that the excluded-key scan reads, so the key travels under a manifest asserting refusal")
	}
	for _, want := range []string{"spc-1-a-record.md", "a fence delimiter inside the frontmatter block"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}

	// A fenced example in the BODY is untouched: a record template showing its
	// own shape is an example, not a field, and refusing it would stop every
	// assembly this repository can run.
	body := "---\nid: spc-1\n---\n\n# A record\n\n```\n---\norigin: an example\n---\n```\n"
	if err := refuses(t, "spc-1-a-record.md", body, refusalKeys, refusalHeadings); err != nil {
		t.Errorf("a fenced example in the body was refused: %v", err)
	}
}

// TestADisplacedFrontmatterBlockRefuses is shape 2 (iss-2608301237456350). The
// block is recognised at line 0 only, so a delimited block preceded by blank
// lines, whitespace or an HTML comment is prose to this binary and frontmatter
// to every reader of the bundle — and what sits inside it travelled.
//
// A delimiter after real prose is a thematic break to every reader and opens
// nothing, so the false-refusal class the line-0 rule closed stays closed.
func TestADisplacedFrontmatterBlockRefuses(t *testing.T) {
	for name, doc := range map[string]string{
		"a blank line first":  "\n---\norigin: ABCD-WARM-ORIGIN\n---\n\n# A record\n",
		"whitespace first":    "   \n---\norigin: ABCD-WARM-ORIGIN\n---\n\n# A record\n",
		"an HTML comment":     "<!-- generated -->\n---\norigin: ABCD-WARM-ORIGIN\n---\n\n# A record\n",
		"a comment and blank": "<!-- generated -->\n\n---\norigin: ABCD-WARM-ORIGIN\n---\n\n# A record\n",
	} {
		err := refuses(t, "docs/reference/a-page.md", doc, refusalKeys, refusalHeadings)
		if err == nil {
			t.Errorf("%s: a displaced frontmatter block was admitted; a reader of the bundle "+
				"reads it as frontmatter and this binary reads it as prose", name)
			continue
		}
		for _, want := range []string{"docs/reference/a-page.md", "displaced from line 0"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("%s: the refusal does not name %q: %v", name, want, err)
			}
		}
	}

	for name, doc := range map[string]string{
		"a thematic break after prose":  "# A page\n\nSome prose.\n\n---\n\nMore prose.\n",
		"an ordinary frontmatter block": "---\nid: spc-1\n---\n\n# A record\n",
		"a thematic break and no block": "# A page\n\n---\n\nProse under a rule.\n",
	} {
		if err := refuses(t, "docs/reference/a-page.md", doc, refusalKeys, refusalHeadings); err != nil {
			t.Errorf("%s was refused: %v", name, err)
		}
	}
}

// TestANestedMappingInASequenceRefuses is shape 3 (iss-2608301237450573, first
// half): a key nested inside a block-sequence entry is invisible to a reader
// anchored to the line, so the floor stops recognising the nesting by the key's
// spelling and refuses the nesting itself. A sequence of SCALARS, which
// committed records carry, is not that shape.
func TestANestedMappingInASequenceRefuses(t *testing.T) {
	const doc = "---\nid: spc-1\nlinks:\n  - origin: ABCD-WARM-ORIGIN\n---\n\n# A record\n"
	err := refuses(t, "spc-1-a-record.md", doc, refusalKeys, refusalHeadings)
	if err == nil {
		t.Fatal("a mapping nested in a block sequence was admitted")
	}
	if !strings.Contains(err.Error(), "a mapping nested in a block sequence") {
		t.Errorf("the refusal does not name the shape: %v", err)
	}

	const scalars = "---\nid: spc-1\nbuilds_on:\n  - itd-183\n  - itd-199\n---\n\n# A record\n"
	if err := refuses(t, "spc-1-a-record.md", scalars, refusalKeys, refusalHeadings); err != nil {
		t.Errorf("a sequence of scalars was refused: %v", err)
	}
}

// TestAFlowExplicitKeyRefuses is shape 4 (iss-2608301251398360), which states
// outright that it and shape 3 want ONE fix rather than two: both are refused
// whatever the key is named, because the floor is not entitled to assume a name
// it cannot resolve.
func TestAFlowExplicitKeyRefuses(t *testing.T) {
	for name, doc := range map[string]string{
		"after a brace": "---\nid: spc-1\nmeta: {? origin: ABCD-WARM-ORIGIN}\n---\n\n# A record\n",
		"after a comma": "---\nid: spc-1\nmeta: {a: 1, ? origin: ABCD-WARM-ORIGIN}\n---\n\n# A record\n",
	} {
		err := refuses(t, "spc-1-a-record.md", doc, refusalKeys, refusalHeadings)
		if err == nil {
			t.Errorf("%s: an explicit key in a flow mapping was admitted", name)
			continue
		}
		if !strings.Contains(err.Error(), "an explicit key in a flow mapping") {
			t.Errorf("%s: the refusal does not name the shape: %v", name, err)
		}
	}

	const flow = "---\nid: spc-1\nrelated: [itd-183, itd-199]\n---\n\n# A record\n"
	if err := refuses(t, "spc-1-a-record.md", flow, refusalKeys, refusalHeadings); err != nil {
		t.Errorf("an ordinary flow sequence was refused: %v", err)
	}
}

// TestAnAttributeValueOnTheNextLineRefuses is shape 5 (iss-2608301350534164):
// the markup mask's blank skip after `=` is space and tab, so a value whose
// opening quote sits on the next line was never masked and the mask declined
// silently. The HTML-whitespace skip the record proposes is deliberately NOT
// taken: a resolved mask on that shape is comprehension, and comprehension is
// what the 2026-08-30 ruling declined.
func TestAnAttributeValueOnTheNextLineRefuses(t *testing.T) {
	const doc = "# A page\n\n<h2 title=\n\"a>b\">Audit Notes</h2>\n"
	err := refuses(t, "docs/reference/a-page.md", doc, nil, refusalHeadings)
	if err == nil {
		t.Fatal("an attribute value opening on the line after its equals sign was admitted")
	}
	for _, want := range []string{
		"docs/reference/a-page.md",
		"an attribute value that opens on the line after its equals sign",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}

	// A value that opens on its own line is the shape; one that opens beside its
	// equals sign is masked as before and refuses only on its heading.
	const inline = "# A page\n\n<h2 title=\"a>b\">An ordinary heading</h2>\n"
	if err := refuses(t, "docs/reference/a-page.md", inline, nil, refusalHeadings); err != nil {
		t.Errorf("a same-line attribute value was refused: %v", err)
	}
}

// TestAnUnboundedRawHeadingRefusesAndACRLFBlankLineBounds is shape 6
// (iss-2608301421380392). Two halves of one element bound.
//
// A raw heading opener with no hard and no soft bound had its title read over
// the whole remainder of the document, which is how the heading under it was
// admitted; the shape is refused instead. And a blank line is the sole bound an
// unclosed element has, so a CRLF blank line has to bound one as an LF blank
// line does — it did not, and a CRLF document's heading travelled.
func TestAnUnboundedRawHeadingRefusesAndACRLFBlankLineBounds(t *testing.T) {
	const unbounded = "# A page\n\n<h2>An ordinary heading and then the rest of the document"
	err := refuses(t, "docs/reference/a-page.md", unbounded, nil, refusalHeadings)
	if err == nil {
		t.Fatal("a raw heading element that is never closed was admitted; its title is read " +
			"over the remainder, which is what admitted the heading under it")
	}
	for _, want := range []string{"docs/reference/a-page.md", "a raw heading element that is never closed"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}

	// The CRLF half: the same document in both line endings must reach the same
	// verdict, because the soft bound is what makes the title readable at all.
	const lf = "# A page\n\n<h2>Audit Notes\n\nprose</h2>\n"
	crlf := strings.ReplaceAll(lf, "\n", "\r\n")
	lfErr := refuses(t, "docs/reference/a-page.md", lf, nil, refusalHeadings)
	crlfErr := refuses(t, "docs/reference/a-page.md", crlf, nil, refusalHeadings)
	if lfErr == nil {
		t.Fatal("the LF document was admitted; the blank line bounds the title at the excluded heading")
	}
	if crlfErr == nil {
		t.Error("the CRLF document was admitted where the LF one was refused; a blank line is " +
			"a blank line in either line ending")
	}
}

// TestAnUnresolvableDocumentIsRefusedByName is ac-1 end to end: a markdown
// document the include table admits whose frontmatter the floor cannot resolve
// stops the whole assembly, the refusal names the document and the shape, and
// no part of the document reaches a bundle.
func TestAnUnresolvableDocumentIsRefusedByName(t *testing.T) {
	const warm = "SENTINEL-DISPLACED-BLOCK"
	root := fixtureRepo(t)
	writeFile(t, root, ".abcd/development/brief/01-product/08-displaced.md",
		"\n---\norigin: "+warm+"\n---\n\n# A displaced record\n\nBody prose.\n")
	gitCommitAll(t, root)

	res, err := Assemble(AssembleRequest{
		RepoRoot: root, Position: PositionWidening, Target: "HEAD", DryRun: true,
	})
	if err == nil {
		t.Fatal("a markdown document the floor cannot resolve was assembled")
	}
	for _, want := range []string{
		".abcd/development/brief/01-product/08-displaced.md",
		"displaced from line 0",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
	if len(res.Bundle.Items) != 0 || len(res.Manifest.Items) != 0 {
		t.Error("the refusal returned a result carrying items; a refused assembly produces no bundle")
	}
}

// TestAFencedMarkupExampleIsNotTheShape: shape 5's scan reads the UNFENCED
// BODY, the way every other scan in this verifier reads it.
//
// A code block showing an HTML attribute whose value opens on the line after
// its equals sign is an EXAMPLE, not markup the bundle carries — the same
// argument the fenced record template rests on in shape 1. Reading the joined
// document instead made any admitted markdown file holding such an example
// refuse every assembly at every position, which is a floor refusing the corpus
// it exists to pass (spc-2609021003136831, "The six refusals"; adr-56 rule 1).
func TestAFencedMarkupExampleIsNotTheShape(t *testing.T) {
	const fenced = "---\nid: spc-1\n---\n\n# A record\n\nHow the shape looks:\n\n" +
		"```html\n<img alt=\n\"a>b\">\n```\n\nProse after the example.\n"
	if err := refuses(t, "spc-1-a-record.md", fenced, refusalKeys, refusalHeadings); err != nil {
		t.Errorf("a fenced markup example was refused: %v", err)
	}

	// The negative control: the same shape OUTSIDE a fence is markup the file
	// carries, the mask still cannot bound it, and it still refuses.
	const bare = "---\nid: spc-1\n---\n\n# A record\n\n<img alt=\n\"a>b\">\n"
	err := refuses(t, "spc-1-a-record.md", bare, refusalKeys, refusalHeadings)
	if err == nil {
		t.Fatal("an unfenced attribute value opening on the line after its equals sign was " +
			"admitted; scoping the scan to the body must not switch the refusal off")
	}
	if !strings.Contains(err.Error(), "an attribute value that opens on the line after its equals sign") {
		t.Errorf("the refusal does not name the shape: %v", err)
	}
}
