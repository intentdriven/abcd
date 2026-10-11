package reading

import (
	"slices"
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

// TestTheCaseFoldIsASCIIOnly: Unicode simple case folding takes the long s
// (U+017F) to an s and the Kelvin sign (U+212A) to a k, so a title spelled with
// either was the excluded heading and silently redacted, although it renders
// differently. Only an ASCII case difference is the same heading; these are
// near-matches, which every verifier path refuses.
func TestTheCaseFoldIsASCIIOnly(t *testing.T) {
	cases := []struct{ title, want string }{
		{"Open Queſtions", "Open Questions"},
		{"Kept Notes", "Kept Notes"},
		{"Kept ſCOPE", "kept scope"},
	}
	for _, c := range cases {
		if _, got := namesExcludedHeading(c.title, map[string]bool{c.want: true}); got != nearHeading {
			t.Errorf("namesExcludedHeading(%q, %q) = %v, want nearHeading", c.title, c.want, got)
		}
	}
	if _, got := namesExcludedHeading("OPEN questions", map[string]bool{"Open Questions": true}); got != sameHeading {
		t.Errorf("an ASCII case variant is no longer the same heading: %v", got)
	}
}

// TestLinkSyntaxTheScannerCannotReadRefusesAnExcludedHeading: the floor's second
// layer. A title the link scanner leaves with link or image syntax in it is
// one the floor has not read, so it is refused whenever its letters still
// carry an excluded heading's, whatever the rest of it says. Each title below
// is malformed, unbalanced or nested past the pass cap, and each travelled
// under the link pattern, which unwrapped what it could and compared the rest
// (iss-2610101930329211). The controls show the layer is narrow: a link the
// scanner reads is the same heading and redacted, and a title with brackets
// or another language and no excluded words travels.
func TestLinkSyntaxTheScannerCannotReadRefusesAnExcludedHeading(t *testing.T) {
	headings := map[string]bool{"Audit Notes": true, "Open Questions": true}
	refused := []string{
		"[Audit Notes](a b c",
		"[Audit Notes(x)",
		"Audit Notes]",
		"[[Audit Notes]",
		"[Open Questions](x \"unclosed title)",
		"[![Audit Notes](a.png)](x y z)",
		strings.Repeat("[", 10) + "Audit Notes" + strings.Repeat("]", 10),
		"[A]udit Notes](x)",
		// Many unreadable tails spend the pass's budget, and the valid link
		// after them is left as written rather than read.
		strings.Repeat("[a](x(", 200) + " [Audit Notes](https://x)",
	}
	for _, title := range refused {
		if want, got := namesExcludedHeading(title, headings); got != nearHeading {
			t.Errorf("namesExcludedHeading(%q) = %v (%q), want nearHeading", title, got, want)
		}
	}
	if _, got := namesExcludedHeading("[Audit Notes](https://x)", headings); got != sameHeading {
		t.Errorf("a plain inline link is no longer the same heading: %v", got)
	}
	for _, title := range []string{"Notas de auditor\u00eda", "See [the guide](https://x) for notes", "[Release notes](a b", "Notes [1] and audits]"} {
		if want, got := namesExcludedHeading(title, headings); got != noHeading {
			t.Errorf("namesExcludedHeading(%q) = %v (%q), want noHeading", title, got, want)
		}
	}
}

// TestTheLinkPassIsLinear: a heading title is the document's to choose, so the
// link scanner's pass must cost a bounded multiple of the title's length on
// any input. The shapes below are the ones that make a naive scanner re-read:
// inline tails that open and never close, unbalanced parentheses, an unclosed
// angle destination or title, and brackets nested deep. The pass charges
// every failed tail to a budget of the title's length and copies the rest as
// written once it is spent, so its work stays under three times the length.
func TestTheLinkPassIsLinear(t *testing.T) {
	const reps = 4000
	for _, s := range []string{
		strings.Repeat("[a](x", reps),
		strings.Repeat("[a](x(", reps) + " ",
		strings.Repeat("[a](<x", reps),
		strings.Repeat("[a](x \"", reps),
		strings.Repeat("[a](x (", reps),
		strings.Repeat("[", reps) + strings.Repeat("]", reps),
		strings.Repeat("![[a]", reps),
	} {
		if _, work := unwrapLinkPass(s); work > 3*len(s) {
			t.Errorf("a pass over %q... (%d bytes) did %d bytes of work, over three times its length",
				s[:12], len(s), work)
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

// TestRawHeadingScanStaysLinearInTheOpenerCount: the raw heading scan finishes
// in time linear in the document over every shape of opener run, not only the
// cheap one.
//
// Three shapes, because each was once the expensive one:
//
//   - openers that each close at once. The bound scan materialised every
//     candidate bound in the whole remainder for every opener, and a committed
//     markdown file up to the size cap did not finish. At the cap the line each
//     opener names was also counted from the top of the document, once per
//     opener.
//   - openers that never close, with no h-tag and no blank line anywhere
//     (iss-2608301421382564). Each title ran to the end of the document, so the
//     scan rendered the whole remainder once per opener: 4 000 openers took
//     13.5 s and 8 000 took 57 s, where the closed shape above took 0.1 s. The
//     test covered only the closed shape, so its name over-claimed.
//   - openers that all share one bound far below them. Each title is bounded,
//     so nothing refuses the shape as unbounded, and each is still read over
//     most of the document.
//
// The last two are refusals as well as costs: a title that is never bounded is
// refused rather than read, and titles that overlap past the floor's read budget
// are refused rather than read in quadratic time. The bound is generous: the
// linear scan is milliseconds. It is a bound on the process's CPU time
// (processCPU), not the wall clock, so a loaded machine that keeps the scan off
// a core cannot trip it (iss-2609240046582859). It is not asserted under -race
// (raceEnabled), where the verdicts still are.
func TestRawHeadingScanStaysLinearInTheOpenerCount(t *testing.T) {
	headings := map[string]bool{"Audit Notes": true}
	build := func(opener string, n int, tail string) string {
		var b strings.Builder
		b.WriteString("# A spec\n\n")
		for range n {
			b.WriteString(opener)
		}
		b.WriteString(tail)
		return b.String()
	}
	closedAtCap := (MaxFileBytes - 64) / len("<h2>Ordinary heading</h2>\n")
	cases := []struct {
		name   string
		doc    string
		refuse string
	}{
		{"closed openers up to the size cap", build("<h2>Ordinary heading</h2>\n", closedAtCap, ""), ""},
		{"unclosed openers with no bound", build(`<p role="heading">x`, 8000, ""), "never closed"},
		{"unclosed openers sharing one far bound", build(`<p role="heading">x`, 8000, "</p>\n"), "read budget"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			start := processCPU()
			err := verifyRedaction("spc-x.md", c.doc, c.doc, nil, headings)
			elapsed := processCPU() - start
			switch {
			case c.refuse == "" && err != nil:
				t.Fatalf("the scan refused an ordinary document: %v", err)
			case c.refuse != "" && err == nil:
				t.Fatalf("the scan admitted a run of openers it cannot read in linear time")
			case c.refuse != "" && !strings.Contains(err.Error(), c.refuse):
				t.Errorf("the refusal does not name %q: %v", c.refuse, err)
			}
			if !raceEnabled && elapsed > 10*time.Second {
				t.Errorf("the raw heading scan took %s of CPU over a %d-byte document; it reads each "+
					"title over the remainder, or counts each line from the top", elapsed, len(c.doc))
			}
		})
	}
}

// TestIndexedHeadingBoundsAgreeWithTheWalk: the bound index answers exactly what
// the walk from each opener answered, in every reading the scan takes, including
// the masked reading where an opener's own `>` is blanked and a bound match can
// straddle it — the one case the index hands back to the walk — and including
// element names that fold under Go's case folding but not under lower-casing
// (the long s, the Kelvin sign).
func TestIndexedHeadingBoundsAgreeWithTheWalk(t *testing.T) {
	docs := []string{
		"<h2>Audit Notes</h2>\n\n<h3><em>x</em> y</h3>",
		"<h2>\n\nAudit Notes</h2>\n<h2>never closed\n",
		"<p role=\"heading\">a</P><div role=heading>b\n\nc</div><h4 id=x>d",
		"<h2 title=\"<h3>\"\n\n>Audit Notes</h2>",
		"<!--\n<h2\n>\n--> <h2>x</h2>",
		"<!-- <h2>a <h3>b --> <h2>c\n\n</h2>",
		"<\u017fpan role=\"heading\">Audit Notes</span> <bloc\u212a role=heading>x</block>",
		"<h2 a=\"x\ny\">z</h2>\r\n\r\n<h1>w",
	}
	for _, doc := range docs {
		lineBounded, _ := maskMarkupData(doc, true)
		unbounded, _ := maskMarkupData(doc, false)
		readings := []string{doc, lineBounded, unbounded}
		for _, found := range readings {
			for _, open := range rawHeadingOpenRe.FindAllStringSubmatchIndex(found, -1) {
				name := openerName(found, open)
				for _, r := range readings {
					budget := newTitleReadBudget(len(r))
					got, gotBounded := indexRawHeadingBounds(r).titleEnds(open[1], name, budget)
					want, wantBounded, _ := walkRawHeadingBounds(r[open[1]:], name)
					if !slices.Equal(got, want) || gotBounded != wantBounded {
						t.Errorf("opener %q at %d over %q: the index says %v (bounded %v), the walk %v (bounded %v)",
							found[open[0]:open[1]], open[1], r, got, gotBounded, want, wantBounded)
					}
				}
			}
		}
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
	start := processCPU()
	if got, _ := maskMarkupData(doc, true); len(got) != len(doc) {
		t.Fatalf("the mask changed the document length from %d to %d", len(doc), len(got))
	}
	if elapsed := processCPU() - start; elapsed > 5*time.Second {
		t.Errorf("the mask took %s of CPU over 800000 assignments on one line; it searches the whole "+
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

// TestANestedMappingRefusesBehindEveryBlockIndicator is shape 3's class, not its
// one spelling (iss-2608301237450573). The refusal read one `- ` and then a key,
// so every other way of reaching a compact nested mapping travelled: a second
// sequence indicator, a node property between the indicator and the key, an
// explicit key inside the entry, an explicit key's value on its `:` line, and a
// single-pair mapping inside a flow sequence. Each is an `origin` key to YAML and
// was nothing to the floor, and the manifest asserted its refusal.
func TestANestedMappingRefusesBehindEveryBlockIndicator(t *testing.T) {
	const pre, post = "---\nid: spc-1\n", "---\n\n# A record\n"
	for name, front := range map[string]string{
		"a sequence of sequences":        "links:\n  - - origin: ABCD-WARM-ORIGIN\n",
		"a tab after the indicator":      "links:\n  -\t- origin: ABCD-WARM-ORIGIN\n",
		"an anchored entry":              "links:\n  - &a origin: ABCD-WARM-ORIGIN\n",
		"a tagged entry":                 "links:\n  - !t origin: ABCD-WARM-ORIGIN\n",
		"an explicit key in an entry":    "links:\n  - ? origin\n    : ABCD-WARM-ORIGIN\n",
		"an explicit value's mapping":    "? meta\n: origin: ABCD-WARM-ORIGIN\n",
		"a flow pair in a flow sequence": "links: [origin: ABCD-WARM-ORIGIN]\n",
		// Siblings refused before this change, kept refused.
		"the recorded shape":                "links:\n  - origin: ABCD-WARM-ORIGIN\n",
		"a quoted key in an entry":          "links:\n  - \"origin\": ABCD-WARM-ORIGIN\n",
		"a flow mapping in an entry":        "links:\n  - {origin: ABCD-WARM-ORIGIN}\n",
		"an anchored flow mapping":          "base: &b {origin: ABCD-WARM-ORIGIN}\nuse: *b\n",
		"a key under a bare indicator":      "links:\n  -\n    origin: ABCD-WARM-ORIGIN\n",
		"a multi-line flow mapping":         "meta: {a: 1,\n  origin: ABCD-WARM-ORIGIN}\n",
		"a block scalar holding the key":    "note: |\n  origin: ABCD-WARM-ORIGIN\n",
		"a quoted pair in a flow sequence":  "links: [\"origin\": ABCD-WARM-ORIGIN]\n",
		"a second key in a nested mapping":  "links:\n  - name: a\n    origin: ABCD-WARM-ORIGIN\n",
		"a flow pair after a flow sequence": "links: [a, origin: ABCD-WARM-ORIGIN]\n",
	} {
		err := refuses(t, "spc-1-a-record.md", pre+front+post, refusalKeys, refusalHeadings)
		if err == nil {
			t.Errorf("%s: admitted; the key is an origin key to YAML and travels", name)
			continue
		}
		if !strings.Contains(err.Error(), "spc-1-a-record.md") {
			t.Errorf("%s: the refusal does not name the document: %v", name, err)
		}
	}

	// The anti-vacuity half: what committed records carry is admitted.
	for name, front := range map[string]string{
		"a sequence of scalars":       "builds_on:\n  - itd-183\n  - \"itd-199\"\n",
		"a flow sequence of scalars":  "related: [itd-183, itd-199]\n",
		"a URL in a flow sequence":    "sources: [https://example.com/a]\n",
		"an explicit key and a value": "? meta\n: a plain value\n",
		"an entry under a bare dash":  "builds_on:\n  -\n    itd-183\n",
	} {
		if err := refuses(t, "spc-1-a-record.md", pre+front+post, refusalKeys, refusalHeadings); err != nil {
			t.Errorf("%s was refused: %v", name, err)
		}
	}
}

// TestAnAliasInAKeyPositionRefuses (iss-2609261900095459). An anchor sits
// wherever a node can, a value included, and an alias written where a key
// stands IS that anchored scalar to YAML: `k: &a origin` then `*a : X` reads as
// {origin: X}. The anchor refusal fired only at line start and behind a block
// indicator, and nothing read `*` at all, so the key travelled. The refusal is
// of the alias in every key position — line start, behind a block indicator,
// behind `{`, `[` or `,` — whatever the anchored scalar says.
func TestAnAliasInAKeyPositionRefuses(t *testing.T) {
	const pre, post = "---\nid: spc-1\n", "---\n\n# A record\n"
	for name, front := range map[string]string{
		"an alias key at line start":          "k: &a origin\n*a : ABCD-WARM-ORIGIN\n",
		"an alias key in a flow mapping":      "k: &a origin\nm: {*a : ABCD-WARM-ORIGIN}\n",
		"an alias key after a flow comma":     "k: &a origin\nm: {x: 1, *a : ABCD-WARM-ORIGIN}\n",
		"an alias pair in a flow sequence":    "k: &a origin\nm: [*a : ABCD-WARM-ORIGIN]\n",
		"an alias key on a flow continuation": "k: &a origin\nm: {x: 1,\n  *a : ABCD-WARM-ORIGIN}\n",
		"a comma-first flow continuation":     "k: &a origin\nm: {x: 1\n  , *a : ABCD-WARM-ORIGIN}\n",
		"an alias key in a nested mapping":    "k: &a origin\nm:\n  *a : ABCD-WARM-ORIGIN\n",
		"an anchor behind a tag":              "k: !!str &a origin\n*a : ABCD-WARM-ORIGIN\n",
		"an anchor in a flow mapping's value": "m: {k: &a origin}\n*a : ABCD-WARM-ORIGIN\n",
		"an anchor in a flow sequence":        "l: [&a origin]\n*a : ABCD-WARM-ORIGIN\n",
		"a tag before a flow alias key":       "k: &a origin\nm: {!!str *a : ABCD-WARM-ORIGIN}\n",
		"a CRLF alias key":                    "k: &a origin\r\n*a : ABCD-WARM-ORIGIN\r\n",
		// Siblings refused before this change, kept refused.
		"an alias as an explicit key":           "k: &a origin\n? *a\n: ABCD-WARM-ORIGIN\n",
		"an alias key in a sequence entry":      "k: &a origin\nlinks:\n  - *a : ABCD-WARM-ORIGIN\n",
		"an alias key behind an explicit value": "k: &a origin\n? meta\n: *a : ABCD-WARM-ORIGIN\n",
		"a merge over an anchored flow map":     "base: &m {origin: ABCD-WARM-ORIGIN}\nuse:\n  <<: *m\n",
		"a merge over an anchored block map":    "base: &m\n  origin: ABCD-WARM-ORIGIN\nuse:\n  <<: *m\n",
	} {
		err := refuses(t, "spc-1-a-record.md", pre+front+post, refusalKeys, refusalHeadings)
		if err == nil {
			t.Errorf("%s: admitted; the alias is an origin key to YAML and travels", name)
			continue
		}
		if !strings.Contains(err.Error(), "spc-1-a-record.md") {
			t.Errorf("%s: the refusal does not name the document: %v", name, err)
		}
	}

	// The anti-vacuity half: an alias in a VALUE position copies a node whose
	// own text the floor already read where the anchor sits, and an asterisk
	// that is not an alias is prose.
	for name, front := range map[string]string{
		"an alias as a value":            "k: &a origin\nuse: *a\n",
		"an alias in a sequence entry":   "k: &a origin\nlist:\n  - *a\n",
		"a merge over a harmless map":    "base: &m {name: x}\nuse:\n  <<: *m\n",
		"an asterisk in a quoted value":  "note: \"see [*] and {*a : b}\"\n",
		"an asterisk inside a plain one": "note: a*b, c *d\n",
	} {
		if err := refuses(t, "spc-1-a-record.md", pre+front+post, refusalKeys, refusalHeadings); err != nil {
			t.Errorf("%s was refused: %v", name, err)
		}
	}
}

// TestTheEscapedKeyRefusalStatesOnlyWhatItKnows (iss-2608301421381157). The
// escaped-key refusal shared the excluded-key message, which asserted that the
// document still carried an excluded key and that its block was not closed the
// way the field reader expects. Neither is known of an escape: the package does
// not decode one, so which key it spells is exactly what it cannot say, and the
// block is closed as expected. The refusal stands; its stated reason is the
// escape.
func TestTheEscapedKeyRefusalStatesOnlyWhatItKnows(t *testing.T) {
	for name, doc := range map[string]string{
		"a line-anchored escaped key":  "---\nid: spc-1\n\"C:\\tmp\\x\": v\n---\n\n# A record\n",
		"an escaped key in a flow map": "---\nid: spc-1\nmeta: {a: 1, \"C:\\tmp\\x\": v}\n---\n\n# A record\n",
	} {
		err := refuses(t, "spc-1-a-record.md", doc, map[string]bool{"origin": true}, nil)
		if err == nil {
			t.Errorf("%s: an escaped key was admitted", name)
			continue
		}
		msg := err.Error()
		for _, want := range []string{"spc-1-a-record.md", "line 3", "escape", `C:\\tmp\\x`} {
			if !strings.Contains(msg, want) {
				t.Errorf("%s: the refusal does not state %q: %v", name, want, err)
			}
		}
		for _, claim := range []string{"excluded key", "not closed"} {
			if strings.Contains(msg, claim) {
				t.Errorf("%s: the refusal asserts %q, which is not known of an escape: %v", name, claim, err)
			}
		}
	}

	// The general refusal names the key and the line, and claims no block shape
	// it did not observe: a quoted key survives redaction in a block closed
	// exactly as the field reader expects.
	const quoted = "---\nid: spc-1\n\"origin\": ABCD-WARM-ORIGIN\n---\n\n# A record\n"
	err := refuses(t, "spc-1-a-record.md", quoted, map[string]bool{"origin": true}, nil)
	if err == nil {
		t.Fatal("a quoted excluded key was admitted")
	}
	for _, want := range []string{"spc-1-a-record.md", `"origin"`, "line 3"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not state %q: %v", want, err)
		}
	}
	if strings.Contains(err.Error(), "not closed") {
		t.Errorf("the refusal asserts a block shape the document does not have: %v", err)
	}
}

// TestOpensTagIsHTMLTagResRule is iss-2608301251394412: the attribute walk's
// opensTag and the title stripper's htmlTagRe are one definition of what opens
// a tag, so on any input the pattern can read to its `>`, the walk opens exactly
// where the pattern matches. The hand-written copy took a `<` and a letter for
// a tag, so it opened on an autolink its own comment said opens nothing.
func TestOpensTagIsHTMLTagResRule(t *testing.T) {
	for _, s := range []string{
		"<h2>", "</h2>", "<h2 id=\"a\">", "<br/>", "<br />", "<my-tag>", "<a\nhref=\"x\">",
		"<https://example.com>", "<mailto:someone@example.com>", "<h2:x>",
		"< h2>", "<2>", "<-x>", "<>", "</>", "a < b >",
	} {
		loc := htmlTagRe.FindStringIndex(s)
		want := loc != nil && loc[0] == 0
		if got := opensTag(s, 0); got != want {
			t.Errorf("opensTag(%q) = %v, htmlTagRe matches at 0 = %v", s, got, want)
		}
	}
}

// TestIsASCIIPunctIsCommonMarksClass holds isASCIIPunct to the 32 bytes
// CommonMark 2.1 names as ASCII punctuation, and to nothing else in a byte.
func TestIsASCIIPunctIsCommonMarksClass(t *testing.T) {
	const commonMark = "!\"#$%&'()*+,-./:;<=>?@[\\]^_\x60{|}~"
	for c := 0; c < 256; c++ {
		want := strings.IndexByte(commonMark, byte(c)) >= 0
		if got := isASCIIPunct(byte(c)); got != want {
			t.Errorf("isASCIIPunct(%#x) = %v, want %v", c, got, want)
		}
	}
}
