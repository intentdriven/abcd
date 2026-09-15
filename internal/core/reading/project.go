package reading

import (
	"fmt"
	"html"
	"regexp"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/site"
)

// Field projection: the heading-scoped extractor this package owns.
//
// Enumeration of the records comes from the record graph, which reports a
// record's id, store, bucket and path and no body at all. Projection is
// therefore a read of the file the graph named, not a second reading of the
// record's shape — which is the distinction that keeps one parser of the
// record in this binary.
//
// A field resolves as a heading section where the file carries a heading of
// that name, and otherwise as a frontmatter key. Nothing else resolves: a field
// the file does not carry contributes no item, which is what lets one
// projection describe an intent whose sections the record is still growing.

// trimBlankEdges joins a section body, dropping the blank lines at either end so
// a projected field is the text and not the whitespace around it.
func trimBlankEdges(lines []string) string {
	start, end := 0, len(lines)
	for start < end && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return strings.Join(lines[start:end], "\n")
}

// redactExcluded removes the exclusion floor's key-signalled and
// heading-signalled material from a document before anything is taken out of
// it.
//
// It runs on every admitted file, projected or whole. Positive inclusion at
// file granularity is not enough on its own: a brief chapter and a spec travel
// whole, and a frontmatter key stamped on every record by the command that mints
// it would ride along inside them. The floor names those keys and headings
// precisely so the signal is mechanical, and this is where the signal is read.
func redactExcluded(rel, doc string, exclusions []Exclusion) (string, error) {
	// There is no file-extension test here any more, and its absence is the
	// point. Both signals are RECORD shapes only a markdown file carries, and
	// that fact used to be asserted twice: the include table admitted by
	// container shape and this function decided, separately, what to examine.
	// The two described different sets, which is the space every leak in
	// itd-194's evidence set lived in. The DECLARATION now lives on the row —
	// Row.Scan — and collect calls this function for a ScanParsed row and for
	// no other, so admission and examination cannot part company
	// (adr-56 rule 3; iss-2608301450065320).
	keys := map[string]bool{}
	headings := map[string]bool{}
	for _, e := range exclusions {
		switch e.Signal {
		case "frontmatter key":
			keys[e.Detail] = true
		case "heading":
			headings[e.Detail] = true
		}
	}
	lines := strings.Split(doc, "\n")
	drop := make([]bool, len(lines))

	// The key signal. A key's continuation lines are indented, so dropping the
	// indented run below it removes a block value whole rather than leaving its
	// body behind as orphaned prose.
	for key, field := range frontmatter.Fields(lines) {
		if !keys[key] {
			continue
		}
		i := field.Line - 1
		if i < 0 || i >= len(lines) {
			continue
		}
		drop[i] = true
		// A block scalar's continuation lines are indented, and a blank line
		// INSIDE one is still part of it — stopping at the first blank leaves the
		// rest of the value sitting in the frontmatter. The run ends at the first
		// non-blank line that is not indented.
		for j := i + 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == "" {
				drop[j] = true
				continue
			}
			if lines[j][0] != ' ' && lines[j][0] != '\t' {
				break
			}
			drop[j] = true
		}
	}

	// The heading signal, over the fence-aware section scan so a `#` inside a
	// code block is not mistaken for one.
	body, offset := site.StripFrontmatter(doc)
	sections, err := site.Sections(rel, body, offset)
	if err != nil {
		return "", fmt.Errorf("reading: reading the sections of %s: %w", rel, err)
	}
	for i, sec := range sections {
		if sec.Level == 0 || !headings[normaliseHeadingTitle(sec.Title)] {
			continue
		}
		start, end := sectionSpan(sections, i, len(lines))
		for j := start; j < end; j++ {
			if j >= 0 {
				drop[j] = true
			}
		}
	}

	kept := make([]string, 0, len(lines))
	for i, line := range lines {
		if !drop[i] {
			kept = append(kept, line)
		}
	}
	out := strings.Join(kept, "\n")
	if err := verifyRedaction(rel, doc, out, keys, headings); err != nil {
		return "", err
	}
	return out, nil
}

var (
	// excludedKeyLineRe matches an excluded key inside a frontmatter block.
	//
	// Two spellings the field reader does not report, and so does not redact,
	// are matched here instead. A QUOTED key is the same key. And a block whose
	// keys all carry leading indent is valid YAML that the reader — which wants
	// a key at column 0 — looks straight past, so `origin` would travel intact.
	// Matching the indent means a nested mapping's key is matched too; that is
	// the fail-closed direction, and an excluded name nested under another key
	// is not a shape any record in this corpus has.
	excludedKeyLineRe = regexp.MustCompile(`^\s*(?:"([A-Za-z_][A-Za-z0-9_-]*)"|'([A-Za-z_][A-Za-z0-9_-]*)'|([A-Za-z_][A-Za-z0-9_-]*))\s*:`)
	// atxCloseRe matches an ATX heading's optional closing sequence. `## X ##`
	// and `## X` are one heading, and the section scan reports the closing hashes
	// as part of the title, so they are normalised away before any comparison.
	atxCloseRe = regexp.MustCompile(`\s+#+\s*$`)
	// setextRuleRe matches the underline that turns the line above it into a
	// heading. The section scan does not model setext at all.
	setextRuleRe = regexp.MustCompile(`^\s{0,3}(=+|-+)\s*$`)
	// indentedATXRe matches an ATX heading carrying the one-to-three-space indent
	// CommonMark allows. The section scan anchors its pattern at column 0, so it
	// reads such a line as prose — while every renderer, and every human, reads
	// it as the heading it is. Four spaces would make it an indented code block,
	// which is why the bound is three.
	// The indent is SPACES, one to three. A tab makes an indented code block,
	// not a heading, so `\s` would refuse a line no renderer treats as one.
	indentedATXRe = regexp.MustCompile(`^[ ]{1,3}#{1,6}\s+(.*)$`)
	// rawHeadingOpenRe matches an element that OPENS a heading: an h1-h6 tag, or
	// any element carrying a heading role, which renders and is announced as a
	// heading while no h-tag appears. Matching the opening tag alone, rather
	// than a closed pair, is what covers the unclosed, self-closing and
	// multi-line forms — a document does not have to be well-formed for a reader
	// to see a heading in it.
	rawHeadingOpenRe = regexp.MustCompile(
		`(?is)<(h[1-6])(?:\s[^>]*)?/?>|<([a-z][a-z0-9-]*)\s[^>]*role\s*=\s*["']?heading\b[^>]*>`)
	// questionLineRe matches any explicit-key line. Whether it is READABLE is a
	// second question, asked against explicitYAMLKeyRe: a `?` line that pattern
	// cannot fully read is a key this package cannot resolve.
	questionLineRe = regexp.MustCompile(`^\s*\?(\s|$)`)
	// htmlCommentRe and htmlTagRe strip the markup a title can carry without
	// changing how it reads on the page.
	htmlCommentRe = regexp.MustCompile(`(?s)<!--.*?-->`)
	// The tag name is bounded so an AUTOLINK is left alone: `<https://x>` looks
	// like a tag until the colon, and stripping it turns a heading carrying a URL
	// into a different heading.
	htmlTagRe = regexp.MustCompile(`</?[A-Za-z][A-Za-z0-9-]*(?:\s[^>]*)?/?>`)
	// mdLinkRe unwraps `[text](target)` to the text a reader sees.
	mdLinkRe = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	// explicitYAMLKeyRe matches YAML's explicit-key form, `? origin`.
	explicitYAMLKeyRe = regexp.MustCompile(`^\s*\?\s+["']?([A-Za-z_][A-Za-z0-9_-]*)["']?\s*$`)
	// flowKeyRe matches a key inside a flow mapping, at top level or nested.
	flowKeyRe = regexp.MustCompile(`[{,]\s*["']?([A-Za-z_][A-Za-z0-9_-]*)["']?\s*:`)
	// doubleQuotedKeyRe captures a double-quoted key's raw spelling, escapes and
	// all, so escapedQuotedKey can judge it. The whitespace is `\s`, YAML's own
	// class, because a carriage return between the key and its colon is a break
	// to a YAML reader and nothing at all to a scan over spaces and tabs.
	doubleQuotedKeyRe = regexp.MustCompile(`^\s*"([^"]*)"\s*:`)
	// fenceOpenRe matches a fenced code block's delimiter, on the section scan's
	// own rule so the two agree about what is inside a fence.
	fenceOpenRe = regexp.MustCompile("^[ \t]*```")
	// rawHeadingBoundRe matches every candidate end of a raw heading's text: a
	// closing tag of ANY element, the next heading open, or a blank line. The
	// element name is CAPTURED so one static pattern serves every element — the
	// alternative was a pattern compiled per element name and cached forever
	// under a key the document chooses, which is a store an input can grow.
	// The blank-line alternative admits a carriage return: a CRLF document's
	// blank line is "\n\r\n", and `[ \t]*` did not match the `\r`, so the sole
	// bound an unclosed element has was missing there and the title was read
	// past the blank line into whatever followed (iss-2608301421380392).
	rawHeadingBoundRe = regexp.MustCompile(`(?is)</([a-z][a-z0-9-]*)\s*>|<h[1-6](?:\s[^>]*)?/?>|\n[ \t\r]*\n`)
	// nestedMappingRe matches a block-sequence entry whose item opens a mapping:
	// `- key:` at any indent, bare or quoted. A key nested that way is invisible
	// to a reader anchored to the line, and the fix the records ask for is to
	// refuse the NESTING rather than to learn one more spelling of the key
	// (iss-2608301237450573, iss-2608301251398360).
	nestedMappingRe = regexp.MustCompile(`^\s*-\s+(?:"[^"]*"|'[^']*'|[A-Za-z_][A-Za-z0-9_-]*)\s*:(\s|$)`)
	// flowExplicitKeyRe matches YAML's explicit-key indicator inside a flow
	// mapping: a `?` following `{` or `,`. Same class, same answer.
	flowExplicitKeyRe = regexp.MustCompile(`[{,]\s*\?`)
)

// Two shapes this floor does NOT see, disclosed rather than claimed. A heading
// nested inside a blockquote or a list item is indented and prefixed, so neither
// the section scan nor the raw-line patterns read it as a heading. And a title
// reaching the excluded one through a homoglyph or an invisible format character
// slugs differently by construction, because the slug compares code points. Both
// are residue; neither is caught.
//
// namesExcludedHeading reports whether a heading title is one of the excluded
// ones, under the ONE equality this floor uses: a case fold, or the same
// rendering. It exists so the three refusal paths — the section scan, the
// indented ATX line, the setext underline — cannot drift apart on what "the same
// heading" means. They did: the render comparison was added on the first path
// only, which closed the class on one of three.
func namesExcludedHeading(title string, headings map[string]bool) (string, bool) {
	for want := range headings {
		if strings.EqualFold(title, want) || sameRendering(title, want) {
			return want, true
		}
	}
	return "", false
}

// sameRendering reports whether two heading titles come out as the same heading
// on the page. `## **Audit Notes**`, "## `Audit Notes`" and a title carrying a
// non-breaking space differ in bytes and are the same heading to every reader,
// so a byte comparison is the wrong test for what the floor is trying to name.
//
// The site's own anchor slug is the comparison: it drops emphasis and code
// marks, lower-cases, and collapses every other run of non-alphanumerics to a
// hyphen — which is exactly the equivalence "renders as the same heading" needs,
// and it is one function rather than a table of markup shapes to keep current.
func sameRendering(a, b string) bool {
	for _, x := range renderedTexts(a) {
		slug := site.Slug(x)
		if slug == "" {
			continue
		}
		for _, y := range renderedTexts(b) {
			if slug == site.Slug(y) {
				return true
			}
		}
	}
	return false
}

// renderedTexts reduces a heading title to the text a reader sees — HTML
// comments and tags removed, link wrappers unwrapped to their label, character
// references decoded — and returns EVERY reading of it rather than one. The
// slug then compares what the page shows rather than what the source spells.
//
// A removed tag has two readings and neither is the title on its own. `<br>` is
// a line break and `</em>` closes a word, so dropping either without the
// boundary it stands for spells `Audit<br>Notes` as one word; and a tag written
// INSIDE a word draws no boundary at all, so standing a space in for it splits
// `Audi<i>t</i> Notes` into three. Replacing every tag with a space closed the
// first shape and opened the second; replacing every tag with nothing did the
// reverse. Both readings are returned and the caller refuses on either, which is
// the doctrine the heading bound already uses: a title read two ways is excluded
// if EITHER way names an excluded heading. A comment is dropped outright under
// both, because a comment draws no boundary either way.
//
// Decoding is html.UnescapeString, one pass over the whole string. A hand list
// of entities applied by ranging a map was not merely incomplete — it was
// NONDETERMINISTIC: `Audit&amp;nbsp;Notes` decoded to the excluded title or not
// depending on whether `&amp;` was applied before `&nbsp;` that time round, so a
// determinism instrument had a coin-flip refusal. One pass also covers the
// numeric and hex character references a short list could never enumerate.
func renderedTexts(title string) []string {
	out := htmlCommentRe.ReplaceAllString(title, "")
	out = mdLinkRe.ReplaceAllString(out, "$1")
	spaced := strings.TrimSpace(html.UnescapeString(htmlTagRe.ReplaceAllString(out, " ")))
	joined := strings.TrimSpace(html.UnescapeString(htmlTagRe.ReplaceAllString(out, "")))
	if joined == spaced {
		return []string{spaced}
	}
	return []string{spaced, joined}
}

// normaliseHeadingTitle reduces a heading to the text it names: surrounding
// whitespace and the optional ATX closing sequence removed.
func normaliseHeadingTitle(title string) string {
	return strings.TrimSpace(atxCloseRe.ReplaceAllString(strings.TrimSpace(title), ""))
}

// unfencedBody joins the document with every fenced line BLANKED rather than
// dropped, for a scan that reads the whole text at once instead of filtering its
// matches by line the way rawHTMLHeading does.
//
// Blanked rather than dropped, because both properties are needed: the fenced
// example must say nothing to the scan, and a match's offset must still count
// the document's own newlines, or the line a refusal names would be the line of
// some shorter text nobody holds.
func unfencedBody(lines []string, fenced []bool) string {
	out := make([]string, len(lines))
	for i, line := range lines {
		if i < len(fenced) && fenced[i] {
			continue
		}
		out[i] = line
	}
	return strings.Join(out, "\n")
}

// verifyRedaction is the key-and-heading half of the exclusion floor made
// fail-closed, the way the path half already is.
//
// Redaction is a positive act over what a parser reported, and three shapes slip
// past a parser that reports one value per key and matches a title exactly. A
// DUPLICATED key keeps its second copy, because Fields keeps the first
// occurrence and drops the rest silently. A frontmatter block closed with four
// dashes is cut from the body by StripFrontmatter, which closes on a `---`
// PREFIX, while Fields wants the delimiter exactly and so reads no fields at all.
// And a heading spelled in another case is not the title the redactor looked for.
//
// In each case the field travels and the manifest still asserts it was refused,
// which is the one thing the manifest exists not to do. A floor a file can
// quietly walk through is a disclosure, not a gate — so a file that still
// carries an excluded shape after redaction refuses the run and names the shape.
func verifyRedaction(rel, original, redacted string, keys, headings map[string]bool) error {
	lines := strings.Split(redacted, "\n")

	// The block is located BEFORE any mask is computed, and the fence mask then
	// starts after the block closes. The previous order let a fence delimiter
	// written inside the frontmatter toggle the mask, which switched off the
	// excluded-key scan over the rest of the block — the set's only critical
	// (iss-2608301350533102). Nothing inside the block can toggle the mask now,
	// and the delimiter itself is refused below as a shape this floor cannot
	// resolve.
	unmasked := make([]bool, len(lines))
	fmOpen, fmClose, hasBlock := firstBlockRange(lines, unmasked)
	fenced := bodyFenceMask(lines, hasBlock, fmClose)

	if len(keys) > 0 {
		for _, dup := range frontmatter.Duplicates(strings.Split(original, "\n")) {
			if keys[dup.Key] {
				return fmt.Errorf("reading: %s declares the excluded key %q more than once (line %d); "+
					"only the first occurrence is redactable, so the rest would travel", rel, dup.Key, dup.Line)
			}
		}
		// A block this binary reads as prose and a reader of the bundle reads as
		// frontmatter. Refused before the block-shape scan, which looks at line 0
		// and would find nothing to say about it (iss-2608301237456350).
		if line, shape, ok := displacedFrontmatter(lines); ok {
			return fmt.Errorf("reading: %s carries %s at line %d; a reader of the bundle reads it "+
				"as frontmatter and this package reads what precedes it as prose, so the block's "+
				"keys are refused rather than read two ways", rel, shape, line)
		}
		if line, shape, ok := unresolvableFrontmatterShape(lines, fenced); ok {
			return fmt.Errorf("reading: %s uses the YAML construction %q at line %d in its frontmatter, "+
				"whose keys this package cannot resolve without becoming a YAML parser; a record has no "+
				"reason to use one, so it is refused rather than guessed at", rel, shape, line)
		}
		if line, key, ok := excludedKeyInFirstBlock(lines, fenced, keys); ok {
			return fmt.Errorf("reading: %s still carries the excluded key %q at line %d after redaction; "+
				"the frontmatter block is not closed the way the field reader expects it", rel, key, line)
		}
	}
	if len(headings) == 0 {
		return nil
	}

	// The heading check runs over the SAME fence-aware scan the redactor spans
	// by. Reading raw lines instead made this floor fire on a fenced example of
	// the record template — a heading inside a code block is an example, not a
	// field, and the redactor rightly left it alone while this refused the run.
	// `offset` is what the stripper reports as the first body line. The raw-HTML
	// scan uses it as a floor; the setext scan does NOT, because that offset can
	// overshoot a block the stripper did not recognise the close of, and a scan
	// that trusts it then skips real body lines. Setext bounds itself by shape
	// instead — see below.
	body, offset := site.StripFrontmatter(redacted)
	sections, err := site.Sections(rel, body, offset)
	if err != nil {
		return fmt.Errorf("reading: re-reading the sections of %s: %w", rel, err)
	}
	for _, sec := range sections {
		if sec.Level == 0 {
			continue
		}
		if want, ok := namesExcludedHeading(normaliseHeadingTitle(sec.Title), headings); ok {
			return fmt.Errorf("reading: %s still carries the excluded heading %q at line %d after "+
				"redaction; the floor names %q, and a heading is excluded however it is spelled",
				rel, sec.Title, sec.Line, want)
		}
	}

	// An indented ATX heading is refused for the same reason, and in the same
	// place. The section scan does not see one, so the redactor has no span to
	// delete and the section travels whole; widening that scan is not this
	// package's call to make, because the site renderer's own output turns on it.
	// A refusal here costs an edit and names the line; the alternative is a leak
	// under a manifest asserting the opposite.
	for i := offset; i < len(lines); i++ {
		line := lines[i]
		if fenced[i] {
			continue
		}
		m := indentedATXRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if want, ok := namesExcludedHeading(normaliseHeadingTitle(m[1]), headings); ok {
			return fmt.Errorf("reading: %s indents the excluded heading %q at line %d; the floor "+
				"names %q, and a heading is excluded however it is spelled", rel, strings.TrimSpace(line), i+1, want)
		}
	}

	// The markup mask declines silently on one shape, and it is refused before
	// the heading scan that depends on the mask runs: a value whose opening
	// quote sits on the line after its equals sign. The skip after `=` is space
	// and tab, so it lands on the newline, the quote is never found and the
	// brackets inside the value are read as structure — at which point a heading
	// every browser renders as the excluded one is judged as something else.
	// Resolving the shape means taking HTML's own whitespace rule, which is
	// comprehension the 2026-08-30 ruling declined (iss-2608301350534164).
	//
	// The scan reads the UNFENCED BODY, the way the raw-HTML scan below reads it
	// and for the same reason: a code block showing the shape is an example, not
	// markup the bundle carries, and reading the joined document made any
	// admitted markdown file holding one refuse every assembly at every position
	// — a floor refusing the corpus it exists to pass.
	if shape := markupShapeOf(unfencedBody(lines, fenced)); shape != "" {
		return fmt.Errorf("reading: %s carries %s; the markup mask cannot bound the value, and "+
			"a mask that declines silently is how a heading written past it travelled", rel, shape)
	}

	// A raw HTML heading is refused on the same ground: the site's page reader
	// REFUSES a raw HTML block rather than admitting one, so a heading spelled
	// that way is a heading to every other reader of the file and nothing at all
	// to the markdown scan — and there is again no span for the redactor.
	//
	// The scan runs over the unfenced body JOINED, not line by line, because
	// `<h2>` and its text and its close need not share a line. The match offset
	// maps back to a line so the refusal still names one.
	if line, title, ok := rawHTMLHeading(lines, fenced, offset, headings); ok {
		if want, hit := namesExcludedHeading(title, headings); hit {
			return fmt.Errorf("reading: %s carries the excluded heading %q as raw HTML at line %d; "+
				"the floor names %q, and a heading is excluded however it is spelled",
				rel, title, line, want)
		}
	}

	// An opener that reaches the end of the document with neither a hard nor a
	// soft bound has its title read over the whole remainder, which is the shape
	// that admitted the heading sitting under it. The refusal removes the
	// shape's admission and claims nothing about the scan's COST, which stays
	// with iss-2608301421382564 (iss-2608301421380392).
	if line, ok := unboundedRawHeading(lines, fenced, offset); ok {
		return fmt.Errorf("reading: %s opens a raw heading element that is never closed at line "+
			"%d, and nothing bounds its title short of the end of the document; the title is "+
			"refused rather than read over the remainder", rel, line)
	}

	// Setext headings are a refusal rather than a redaction. The section scan
	// does not model them, so there is no span to delete, and inventing a second
	// heading scanner here to compute one is the second parser this package
	// exists not to grow. A record underlining its Audit Notes is rare and a
	// refusal names it; a leak would not.
	// The setext scan runs from line 0 and skips only an INDENTED line inside the
	// first block. Confining it to the stripper's offset trusted a closer that
	// can overshoot; skipping by shape instead needs no offset at all, and still
	// keeps a block scalar's last line — always indented — from being read as an
	// underlined heading.
	inFirstBlock := func(i int) bool {
		return hasBlock && i > fmOpen && (fmClose < 0 || i < fmClose)
	}
	for i := 0; i+1 < len(lines); i++ {
		if fenced[i] || fenced[i+1] || !setextRuleRe.MatchString(lines[i+1]) {
			continue
		}
		if inFirstBlock(i) && strings.HasPrefix(lines[i], " ") {
			continue
		}
		// The closing `---` sits directly under the block's last line, so reading
		// it as an underline makes any record whose last key names an excluded
		// heading refuse. It closes a block; it underlines nothing.
		if hasBlock && i+1 == fmClose {
			continue
		}
		title := normaliseHeadingTitle(lines[i])
		if title == "" {
			continue
		}
		if want, ok := namesExcludedHeading(title, headings); ok {
			return fmt.Errorf("reading: %s underlines the excluded heading %q at line %d; the floor "+
				"names %q, and a heading is excluded however it is spelled", rel, lines[i], i+1, want)
		}
	}
	return nil
}

// blankQuoted reads one frontmatter line by POSITION rather than by byte. It
// reports the line with every quoted token's interior blanked to spaces — so a
// scan over the rest of the line cannot see inside a string — together with the
// quoted tokens that turned out to be KEYS, each with its quotes still on. The
// length is kept so a caller counting braces or offsets is measuring the same
// line it started with.
//
// Two rules, one class each.
//
// A quote opens a token only in SCALAR POSITION, and a position is decided by a
// YAML indicator carrying the whitespace YAML requires of it. `:` is a mapping
// indicator only before whitespace or the end of the line; `-` and `?` are
// indicators only at the start of a line and before whitespace; `{`, `[` and `,`
// are indicators wherever they stand; and the start of a line is a position of
// its own. Reading a bare `:` or a bare `-` as an opener let a quote sitting
// inside a plain scalar — `a:'b` and `a - 'b` — pair with a later one and blank
// the excluded key between them, so the flow scan never saw it. Reading EVERY
// quote as an opener did the same to an apostrophe in ordinary prose.
//
// A quoted token followed by a colon is a KEY, not a scalar. Blanking it as a
// scalar hid `{"origin": x}` from the flow scan entirely, and the excluded key
// travelled under a manifest asserting its refusal. Such a token is reported by
// name instead, and its interior is still blanked: the name is the token's own
// text, so `{"a, origin: b": 1}` names one key called `a, origin: b` — which is
// what YAML reads there — rather than the key `origin` a scan of the interior
// would have found.
//
// Disclosed, and NARROWER than it looks: the double-quoted form is escape-aware,
// as YAML is, so `\"` continues the token. A line whose double-quoted token is
// never closed is therefore read as unterminated, and NOTHING from the opening
// quote on is blanked. For the flow scan that is the fail-closed direction — it
// then reads the rest of the line in full — but for the QUOTED KEYS this
// function reports it is not: an unterminated token is never reported as a key,
// so `"origin\": x`, whose closing quote is itself escaped, yields no quoted key
// and the escaped-key refusal carried on this path is never reached. The
// line-level escapedQuotedKey is what covers that spelling, over the raw bytes;
// neither rule subsumes the other.
//
// The blanks skipped between a token and its colon are skipBlanks's, which is
// YAML's whitespace rather than space and tab alone.
func blankQuoted(line string) (string, []string) {
	out := []byte(line)
	var quotedKeys []string
	scalar := true    // the start of a line is a scalar position
	lineStart := true // only whitespace and block indicators seen so far
	for i := 0; i < len(line); {
		c := line[i]
		if c == ' ' || c == '\t' {
			i++
			continue
		}
		if scalar && (c == '"' || c == '\'') {
			end, ok := quotedEnd(line, i)
			if !ok {
				break
			}
			if j := skipBlanks(line, end); j < len(line) && line[j] == ':' {
				quotedKeys = append(quotedKeys, line[i:end])
			}
			for k := i + 1; k < end-1; k++ {
				out[k] = ' '
			}
			i, scalar, lineStart = end, false, false
			continue
		}
		switch c {
		case '{', '[', ',':
			scalar = true
		case '}', ']':
			scalar = false
		case ':':
			scalar = i+1 >= len(line) || line[i+1] == ' ' || line[i+1] == '\t'
		case '-', '?':
			// A sequence entry or an explicit key opens a position of its own,
			// and leaves the line still at its start: `- - 'x'` nests.
			if lineStart && (i+1 >= len(line) || line[i+1] == ' ' || line[i+1] == '\t') {
				scalar = true
				i++
				continue
			}
			scalar = false
		default:
			scalar = false
		}
		lineStart = false
		i++
	}
	return string(out), quotedKeys
}

// quotedEnd reports the offset just past the closing quote of the token opening
// at i, and whether the token is closed at all. Both YAML escapes are honoured:
// a backslash escape inside double quotes, a doubled quote inside single ones.
func quotedEnd(s string, i int) (int, bool) {
	if s[i] == '"' {
		for j := i + 1; j < len(s); j++ {
			switch s[j] {
			case '\\':
				j++
			case '"':
				return j + 1, true
			}
		}
		return 0, false
	}
	for j := i + 1; j < len(s); j++ {
		if s[j] != '\'' {
			continue
		}
		if j+1 < len(s) && s[j+1] == '\'' {
			j++
			continue
		}
		return j + 1, true
	}
	return 0, false
}

// skipBlanks returns the offset of the first character at or after i that is not
// blank, where blank is wider than space and tab. A line whose bytes are
// `"origin"\r: x` is a top-level `origin` key to a YAML reader — the carriage
// return belongs to the line break, not to the name — and stopping at it read
// the token as a scalar instead of a key, so the key was never reported and
// never refused.
//
// The class is deliberately NOT YAML's `s-white`, which is space and tab alone:
// a carriage return is a line break to YAML and a form feed is not a permitted
// character in a YAML stream at all. Both are admitted here because neither has
// any legitimate place inside a frontmatter line in this corpus, and skipping
// them can only report MORE keys.
func skipBlanks(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\r' || s[i] == '\f') {
		i++
	}
	return i
}

// blockCloser reports whether a line closes a frontmatter block. YAML closes a
// document with `---` or `...`, and both end the block a key scan is walking.
func blockCloser(line string) bool {
	// Column 0, not "after trimming". YAML closes a document at the left margin,
	// and trimming first made an ellipsis or a rule INSIDE a block scalar close
	// the block — so a record was refused for a shape it does not have.
	return strings.HasPrefix(line, "---") || strings.HasPrefix(line, "...")
}

// submatches returns a match's non-empty capture groups, so a pattern spelling
// one name in several alternatives is read the same way as one that does not.
func submatches(m []string) []string {
	if len(m) < 2 {
		return nil
	}
	out := make([]string, 0, len(m)-1)
	for _, g := range m[1:] {
		if g != "" {
			out = append(out, g)
		}
	}
	return out
}

// escapedQuotedKey refuses a double-quoted frontmatter key containing a
// backslash. YAML decodes escapes inside double quotes, so "origin" IS
// `origin` to the reader and is nothing at all to a pattern over the bytes.
// Rather than grow a YAML decoder here, the escape itself is the signal: a
// record has no reason to spell a key that way, and refusing is the fail-closed
// answer to a name this package cannot resolve.
//
// The positional scanner carries the same refusal on the flow path, and does not
// subsume this one. Its quoted token is escape-aware, as YAML is, so a key whose
// own closing quote is escaped — `"origin\": x` — never closes and is never
// reported as a key at all; the escape refusal it carries is then never reached.
// A line-level pattern over the raw bytes sees that spelling exactly because it
// does not model the escape. Two rules, because neither reaches the other's
// class.
func escapedQuotedKey(line string) (string, bool) {
	m := doubleQuotedKeyRe.FindStringSubmatch(line)
	if m == nil || !strings.Contains(m[1], `\`) {
		return "", false
	}
	return m[1], true
}

// rawHeadingTitleEnds bounds the text one heading element introduces, and
// returns EVERY reading of that bound rather than one.
//
// The hard bound is the element's OWN closing tag or the next heading open,
// whichever comes first. Bounding at any closing tag instead cut the title at
// the first inline element inside the heading — `<h2><a id="x"></a>Audit
// Notes</h2>` ended at `</a>` and yielded nothing at all, and
// `<h2><em>Audit</em> Notes</h2>` yielded "Audit". Both were admitted. The name
// is folded against the opener's, so one static pattern serves every element
// name and nothing is compiled from a name the document chose.
//
// A blank line is a SOFT bound, and only that. It is the sole bound an element
// that is never closed has, so dropping it would admit `<h2>Audit Notes`
// followed by a paragraph; and it is not a bound at all to a renderer, which
// reads a blank line inside a heading element as whitespace — so applying it
// unconditionally emptied the title of `<h2>\n\nAudit Notes</h2>` and admitted
// that. Neither reading is the heading on its own, so both are returned and the
// caller refuses on either: a title read two ways is excluded if EITHER way
// names an excluded heading.
//
// The walk is incremental rather than a materialised match list. Listing every
// candidate bound in the whole remainder for every opener, only to break at the
// first hard one, is quadratic with a large constant: a committed markdown file
// of repeated openers up to the size cap this package sets did not finish, and a
// silent hang is the one staging a fail-closed floor cannot afford.
// The second return says whether ANY bound was found. An opener with neither a
// hard nor a soft bound has its title read over the whole remainder of the
// document, which is the shape that admitted the heading sitting under it; the
// caller refuses it rather than reading the remainder as a title
// (iss-2608301421380392).
func rawHeadingTitleEnds(rest, name string) ([]int, bool) {
	hard, soft := -1, -1
	for off := 0; off < len(rest); {
		m := rawHeadingBoundRe.FindStringSubmatchIndex(rest[off:])
		if m == nil {
			break
		}
		at := off + m[0]
		switch {
		case rest[at] == '\n': // a blank line
			if soft < 0 {
				soft = at
			}
			off += m[1]
			continue
		case m[2] >= 0: // a closing tag, this element's or an inner one's
			if !strings.EqualFold(rest[off+m[2]:off+m[3]], name) {
				off += m[1]
				continue
			}
		}
		hard = at
		break
	}
	bounded := hard >= 0 || soft >= 0
	if hard < 0 {
		hard = len(rest)
	}
	ends := []int{hard}
	if soft >= 0 && soft < hard {
		ends = append(ends, soft)
	}
	return ends, bounded
}

// maskMarkupData blanks, length-preservingly, the angle brackets that stand
// INSIDE an HTML comment or inside a quoted attribute value.
//
// One class: a `<` or a `>` written in either place is data, and the heading
// scan read it as structure. A `</h2>` inside a comment or inside a `title`
// attribute bounded the title short of the heading it belongs to, and a `>`
// inside an attribute value ended the opening tag early — in each case a heading
// every browser renders as the excluded one was judged as something else.
//
// Only the brackets are blanked, never the whole span: an attribute VALUE is
// still read by the opener pattern, which recognises `role="heading"` by its
// value, and a comment's text is still stripped when the title is rendered.
//
// Where an attribute value ENDS is genuinely ambiguous, so lineBound selects one
// of the two honest answers and the caller takes both.
//
// A browser's own tokenizer ends an unterminated value at the end of the input,
// which is lineBound=false: the closing quote is the next matching quote
// anywhere ahead. That reading is right about `<h2 title="a>` continued on the
// next line — a shape every renderer shows as the excluded heading, and one the
// opener pattern cannot parse at all until the `>` inside the value is blanked.
// It is wrong about a stray quote, which then blanks every angle bracket up to
// some unrelated quote thousands of bytes later.
//
// The conservative answer is lineBound=true: a value ends on the line it opens
// on, and a value that does not close there masks nothing. That reading is right
// about the stray quote, and about a legitimate value whose masking the stray
// one would otherwise have consumed — and it is the reading that is wrong about
// the value spanning a line.
//
// Neither is the document on its own, so neither may DECIDE. Both are handed to
// the caller alongside the unmasked text, and a heading is refused if any
// reading names an excluded one: masking can add a reading and can never take
// one away. Substituting one of these maskings for the other, rather than adding
// it, is precisely how the line-bounded reading arrived carrying a leak.
// The second return names the one shape the mask cannot bound: an attribute
// value whose opening quote sits on the line AFTER its equals sign. The skip
// after `=` is space and tab, so it lands on the newline and no quote is found
// — the mask then declines silently, and the brackets inside the value are read
// as structure. Taking HTML's own whitespace class here would resolve the
// shape, and resolving is comprehension the 2026-08-30 ruling declined, so the
// shape is reported and the caller refuses it (iss-2608301350534164).
func maskMarkupData(s string, lineBound bool) (string, string) {
	shape := ""
	out := []byte(s)
	// A MONOTONE cursor onto the next newline, because the walk's own offsets
	// only ever advance. Searching the whole remainder for one per attribute
	// assignment is quadratic: a 4 MiB file of assignments on one line took 21
	// seconds where the walk takes milliseconds, and the size cap bounds one
	// file rather than the number of them.
	nl := -1
	nextNewline := func(from int) int {
		if nl < from {
			if n := strings.IndexByte(s[from:], '\n'); n < 0 {
				nl = len(s)
			} else {
				nl = from + n
			}
		}
		return nl
	}
	for i := 0; i < len(s); {
		if strings.HasPrefix(s[i:], "<!--") {
			end := strings.Index(s[i+4:], "-->")
			if end < 0 {
				i += 4
				continue
			}
			maskAngles(out, i+4, i+4+end)
			i += 4 + end + 3
			continue
		}
		if !opensTag(s, i) {
			i++
			continue
		}
		for i++; i < len(s) && s[i] != '>'; {
			if s[i] != '=' {
				i++
				continue
			}
			q := skipBlanks(s, i+1)
			if q < len(s) && s[q] == '\n' {
				if next := skipSpaceAndNewlines(s, q); next < len(s) &&
					(s[next] == '"' || s[next] == '\'') {
					shape = "an attribute value that opens on the line after its equals sign"
				}
			}
			if q >= len(s) || (s[q] != '"' && s[q] != '\'') {
				i++
				continue
			}
			limit := len(s)
			if lineBound {
				limit = nextNewline(q + 1)
			}
			end := strings.IndexByte(s[q+1:limit], s[q])
			if end < 0 {
				i++
				continue
			}
			maskAngles(out, q+1, q+1+end)
			i = q + end + 2
		}
		i++
	}
	return string(out), shape
}

// skipSpaceAndNewlines advances over whitespace INCLUDING line breaks. It is
// used only to decide whether the byte a declined attribute value would have
// opened at is a quote, so the shape report names a real value rather than any
// equals sign followed by a newline.
func skipSpaceAndNewlines(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\r' || s[i] == '\f' || s[i] == '\n') {
		i++
	}
	return i
}

// opensTag reports whether s[i] begins an HTML tag, on htmlTagRe's own rule: a
// `<` followed by a name, or by a slash and a name. An autolink and a bare `<`
// in prose open nothing, so neither drags the attribute walk over them.
func opensTag(s string, i int) bool {
	if s[i] != '<' {
		return false
	}
	j := i + 1
	if j < len(s) && s[j] == '/' {
		j++
	}
	return j < len(s) && (s[j] >= 'A' && s[j] <= 'Z' || s[j] >= 'a' && s[j] <= 'z')
}

// maskAngles blanks the angle brackets in s[from:to], leaving every other byte —
// newlines included — where it stands.
func maskAngles(out []byte, from, to int) {
	for i := from; i < to && i < len(out); i++ {
		if out[i] == '<' || out[i] == '>' {
			out[i] = ' '
		}
	}
}

// rawHTMLHeading finds the first raw HTML heading in the unfenced body whose
// text names an excluded heading, and reports the line it sits on.
//
// The body is joined before scanning because a raw heading need not fit on one
// line: `<h2>`, its text and its close can sit on three. Joining costs the line
// number, so the match offset is mapped back to one by counting newlines ahead
// of it — the refusal has to name a line a human can go and look at.
//
// An opener sitting on a fenced line is skipped, so an example inside a code
// block still cannot fire.
//
// The document is read TWICE: once as it stands, and once with its markup DATA
// masked — see maskMarkupData — because the opener and the bound are structure
// and a `<` or a `>` written inside a comment or an attribute value is not.
// Masking is length- and newline-preserving, so one set of offsets names the
// same lines in both readings.
//
// Both readings are taken because a mask that DECIDES is a mask that can hide.
// Scanning only the masked copy meant a heading written wholly inside an HTML
// comment was never discovered at all, and the section travelled — while the
// blind reader receives raw markdown, so the file plainly still carries the
// heading. The mask exists to stop a comment's brackets from being read as
// structure; it must never stop them from being read as content. Reading both
// makes the mask purely additive: every refusal the unmasked text supports still
// stands, and the masked text can only add more.
func rawHTMLHeading(lines []string, fenced []bool, offset int, headings map[string]bool) (int, string, bool) {
	raw := strings.Join(lines, "\n")
	lineBounded, _ := maskMarkupData(raw, true)
	unbounded, _ := maskMarkupData(raw, false)
	readings := []string{raw}
	for _, masked := range []string{lineBounded, unbounded} {
		if !slices.Contains(readings, masked) {
			readings = append(readings, masked)
		}
	}

	for _, text := range readings {
		for _, open := range rawHeadingOpenRe.FindAllStringSubmatchIndex(text, -1) {
			line := strings.Count(text[:open[0]], "\n")
			if line < offset || (line < len(fenced) && fenced[line]) {
				continue
			}
			name := ""
			for _, g := range [][2]int{{open[2], open[3]}, {open[4], open[5]}} {
				if g[0] >= 0 {
					name = text[g[0]:g[1]]
				}
			}
			if name == "" {
				continue
			}
			rests := make([]string, 0, len(readings))
			for _, r := range readings {
				rests = append(rests, r[open[1]:])
			}
			if title, ok := excludedRawTitle(rests, name, headings); ok {
				return line + 1, title, true
			}
		}
	}
	return 0, "", false
}

// excludedRawTitle reports the excluded heading the text after one raw opener
// names, under every reading of that text this floor takes: each reading of the
// document bounds the title, and each reading is then read for the title itself.
//
// The two are separate questions. The mask answers where the title ENDS, since a
// `</h2>` written inside an attribute value bounds nothing; the unmasked text
// answers what the title SAYS, since a heading written inside a comment is still
// carried by the file. Taking either alone lost the other.
func excludedRawTitle(rests []string, name string, headings map[string]bool) (string, bool) {
	seen := map[int]bool{}
	for _, bound := range rests {
		ends, _ := rawHeadingTitleEnds(bound, name)
		for _, end := range ends {
			if seen[end] {
				continue
			}
			seen[end] = true
			for _, text := range rests {
				for _, read := range renderedTexts(text[:end]) {
					title := normaliseHeadingTitle(read)
					if title == "" {
						continue
					}
					if _, ok := namesExcludedHeading(title, headings); ok {
						return title, true
					}
				}
			}
		}
	}
	return "", false
}

// unresolvableFrontmatterShape reports a construction in the first block whose
// keys this package cannot resolve, or a block whose bounds it cannot trust.
//
// The reasoning is the escaped quoted key's — a key spelled `"ori\u0067in"` IS
// `origin` to YAML and is nothing at all to a compare over the bytes — applied
// to its whole class: resolving any of these means a YAML parser, a record has
// no reason to use one, so the construction itself is the signal and the answer
// is a refusal rather than a guess. Any line-initial `!` is a tag — the double-bang shorthand, a
// single-bang local tag, a verbatim `!<…>` tag alike. Any `&` is an anchor. And
// any explicit-key line the readable-key pattern cannot fully read is a key
// whose name this package is not entitled to assume.
//
// The block BOUNDS matter for the same reason the keys do. The frontmatter
// stripper closes on `---`, so a block closed by `...`, or opened and never
// closed, makes the offset it reports overshoot into the body — and a scan that
// trusted that offset skipped past real body lines. Refusing both shapes means
// no scan downstream has to reason about an offset that lies.
func unresolvableFrontmatterShape(lines []string, fenced []bool) (int, string, bool) {
	open, close, ok := firstBlockRange(lines, fenced)
	if !ok {
		return 0, "", false
	}
	if close < 0 {
		return open + 1, "a frontmatter block that is never closed", true
	}
	if strings.HasPrefix(strings.TrimSpace(lines[close]), "...") {
		return close + 1, "a frontmatter block closed by `...`", true
	}
	for i := open + 1; i < close; i++ {
		if fenced[i] {
			continue
		}
		trimmed := strings.TrimLeft(lines[i], " \t")
		switch {
		// The fence delimiter is first because it is the shape that used to
		// switch the rest of this scan off. It can no longer do so — the mask
		// starts after the block closes — and it is refused rather than
		// tolerated, because a delimiter inside a frontmatter block is a
		// document this binary and a reader of the bundle read differently
		// (iss-2608301350533102).
		case fenceOpenRe.MatchString(lines[i]):
			return i + 1, "a fence delimiter inside the frontmatter block", true
		case strings.HasPrefix(trimmed, "!"):
			return i + 1, "a YAML tag", true
		case strings.HasPrefix(trimmed, "&"):
			return i + 1, "a YAML anchor", true
		case nestedMappingRe.MatchString(lines[i]):
			return i + 1, "a mapping nested in a block sequence", true
		case flowExplicitKeyRe.MatchString(lines[i]):
			return i + 1, "an explicit key in a flow mapping", true
		case questionLineRe.MatchString(lines[i]) && !explicitYAMLKeyRe.MatchString(lines[i]):
			return i + 1, "an explicit key this package cannot read", true
		}
	}
	return 0, "", false
}

// bodyFenceMask reports, per line, whether that line sits inside a fenced code
// block in the document's BODY — the region after the frontmatter block closes.
//
// It exists because the mask and the block bounds used to be computed in the
// wrong order. The mask ran over the whole document, so a fence delimiter
// written inside the frontmatter toggled the mask and every line after it in
// the block was reported as fenced; excludedKeyInFirstBlock skips fenced lines,
// so the key scan was switched off from inside the very block it exists to read
// (iss-2608301350533102). The block is located first, over an unmasked
// document, and the toggle starts after it — so a document cannot decide
// whether its own frontmatter is scanned.
//
// A document with no frontmatter block is masked from line 0, exactly as before.
// A block that never closes leaves no body to mask, and the never-closed shape
// is itself refused.
func bodyFenceMask(lines []string, hasBlock bool, closeAt int) []bool {
	start := 0
	if hasBlock {
		if closeAt < 0 {
			start = len(lines)
		} else {
			start = closeAt + 1
		}
	}
	mask := make([]bool, len(lines))
	inside := false
	for i := start; i < len(lines); i++ {
		if fenceOpenRe.MatchString(lines[i]) {
			inside = !inside
			mask[i] = true // the delimiter itself belongs to the block
			continue
		}
		mask[i] = inside
	}
	return mask
}

// displacedFrontmatter reports a delimited block that does not open at line 0
// but opens one to a reader of the bundle, because only blank lines,
// whitespace, a byte-order mark or an HTML comment precede it.
//
// firstBlockRange recognises a block at line 0 alone, and rightly: reading the
// first `---` found anywhere made a thematic break in an ordinary documentation
// page open a phantom block. But the rule has a gap on the other side. A
// document whose first content is a `---` after two blank lines is frontmatter
// to every YAML-aware reader and prose to this binary, so its keys were never
// scanned and its excluded fields travelled under a manifest asserting their
// refusal (iss-2608301237456350).
//
// A delimiter after REAL prose is a thematic break to every reader and opens
// nothing, so it is not this shape and the false-refusal class the line-0 rule
// closed stays closed. What is refused is exactly the document the two readers
// disagree about.
func displacedFrontmatter(lines []string) (int, string, bool) {
	inComment := false
	for i, raw := range lines {
		line := raw
		if i == 0 {
			line = frontmatter.TrimBOM(line)
		}
		trimmed := strings.TrimSpace(line)
		if inComment {
			if idx := strings.Index(trimmed, "-->"); idx >= 0 {
				inComment = false
				trimmed = strings.TrimSpace(trimmed[idx+len("-->"):])
			} else {
				continue
			}
		}
		for strings.HasPrefix(trimmed, "<!--") {
			idx := strings.Index(trimmed, "-->")
			if idx < 0 {
				inComment = true
				trimmed = ""
				break
			}
			trimmed = strings.TrimSpace(trimmed[idx+len("-->"):])
		}
		if trimmed == "" {
			continue
		}
		if i == 0 || !strings.HasPrefix(trimmed, "---") {
			// Either the block opens where this package already reads it, or the
			// document's first content is not a delimiter and there is nothing
			// here two readers can disagree about.
			return 0, "", false
		}
		return i + 1, fmt.Sprintf("a frontmatter block displaced from line 0 by %d line(s)", i), true
	}
	return 0, "", false
}

// unboundedRawHeading reports a raw heading opener whose title reaches the end
// of the document with neither a hard nor a soft bound.
//
// It reads the UNMASKED document alone. The mask exists to stop a comment's or
// an attribute value's brackets from being read as structure, and a bound it
// erases was never a bound; a mask that could manufacture this refusal would be
// a mask deciding, which is what every other reading here refuses to let it do.
func unboundedRawHeading(lines []string, fenced []bool, offset int) (int, bool) {
	text := strings.Join(lines, "\n")
	for _, open := range rawHeadingOpenRe.FindAllStringSubmatchIndex(text, -1) {
		line := strings.Count(text[:open[0]], "\n")
		if line < offset || (line < len(fenced) && fenced[line]) {
			continue
		}
		name := ""
		for _, g := range [][2]int{{open[2], open[3]}, {open[4], open[5]}} {
			if g[0] >= 0 {
				name = text[g[0]:g[1]]
			}
		}
		if name == "" {
			continue
		}
		if _, bounded := rawHeadingTitleEnds(text[open[1]:], name); !bounded {
			return line + 1, true
		}
	}
	return 0, false
}

// markupShapeOf reports the one markup shape the mask cannot bound, or "" when
// the document carries none. See the maskMarkupData return it reads.
func markupShapeOf(s string) string {
	if _, shape := maskMarkupData(s, true); shape != "" {
		return shape
	}
	_, shape := maskMarkupData(s, false)
	return shape
}

// firstBlockRange locates the document's first delimiter-fenced region: the line
// that opens it and the line that closes it, or -1 for a block never closed.
func firstBlockRange(lines []string, fenced []bool) (int, int, bool) {
	// Line 0 only, a byte-order mark allowed ahead of it. That is the ONLY place
	// the frontmatter stripper recognises a block, and reading the first `---`
	// found anywhere read a thematic break as a frontmatter opener: an ordinary
	// documentation page with a rule in it was then refused as an unclosed
	// block, or its next line read as a tag or an anchor. Every docs page is
	// admitted, so that was a floor refusing the corpus it exists to pass.
	if len(lines) == 0 || (len(fenced) > 0 && fenced[0]) {
		return 0, 0, false
	}
	if !strings.HasPrefix(frontmatter.TrimBOM(lines[0]), "---") {
		return 0, 0, false
	}
	for i := 1; i < len(lines); i++ {
		if !fenced[i] && blockCloser(lines[i]) {
			return 0, i, true
		}
	}
	return 0, -1, true
}

// excludedKeyInFirstBlock reports an excluded key inside the document's
// frontmatter block.
//
// The block is firstBlockRange's, which is the ONE definition of it this package
// holds. Finding the block wherever three dashes first appeared was a second
// definition, and the two disagreed exactly where the false-refusal class lives:
// a thematic break in an ordinary documentation page opened a phantom block, and
// a line of prose spelled `origin: …` beneath it refused the run. There is no
// key to lose by agreeing — a document whose line 0 is not a delimiter has no
// frontmatter to any reader in this binary, so what sits under its rules is body
// prose, which travels because inclusion admits it and not because redaction
// missed it.
//
// The looseness kept is the block's own bounds: any line OPENING with three
// dashes delimits it, not an exact `---`, because that is the rule the
// frontmatter stripper applies and the gap between the two rules is where a key
// survives.
func excludedKeyInFirstBlock(lines []string, fenced []bool, keys map[string]bool) (int, string, bool) {
	open, closed, ok := firstBlockRange(lines, fenced)
	if !ok {
		return 0, "", false
	}
	end := len(lines)
	if closed >= 0 {
		end = closed
	}
	depth := 0
	for i := open + 1; i < end; i++ {
		if fenced[i] {
			continue
		}
		// Four spellings, because the field reader reports one of them. A plain
		// or quoted key at any indent; YAML's explicit-key form; a key inside a
		// flow mapping at top level or nested; and a double-quoted key whose name
		// is spelled with an escape, which no pattern over the raw bytes can see
		// — so a quoted key carrying a backslash is refused on the backslash
		// rather than on the name it hides.
		for _, m := range [][]string{
			excludedKeyLineRe.FindStringSubmatch(lines[i]),
			explicitYAMLKeyRe.FindStringSubmatch(lines[i]),
		} {
			for _, key := range submatches(m) {
				if keys[key] {
					return i + 1, key, true
				}
			}
		}
		if key, ok := escapedQuotedKey(lines[i]); ok {
			return i + 1, key, true
		}
		// The flow scan runs UNANCHORED over the line with its quoted scalars
		// blanked. Blanking is what closes the false positive — a quoted reason
		// string that merely quotes a flow mapping no longer matches — without
		// giving up the nested shapes an anchored scan could not see: a map in a
		// sequence, a map in a flow sequence, a tagged or anchored map.
		//
		// `depth` carries an open brace across lines, so a key on a continuation
		// line is read as the flow key it is rather than as prose.
		bare, quotedKeys := blankQuoted(lines[i])
		// A quoted key is reported BY the scanner rather than found in the
		// blanked line: its interior is blanked precisely so nothing is read out
		// of it, and its name is the token's own text.
		for _, tok := range quotedKeys {
			name := tok[1 : len(tok)-1]
			// The escaped-key refusal, on the flow path too. YAML decodes escapes
			// inside double quotes, so "ori\u0067in" IS `origin` and is nothing
			// at all to a compare over the bytes — the escape is the signal
			// wherever the key stands, and escapedQuotedKey below reaches only
			// the line-anchored spelling of it.
			if tok[0] == '"' && strings.Contains(name, `\`) {
				return i + 1, name, true
			}
			if keys[name] {
				return i + 1, name, true
			}
		}
		scan := bare
		if depth > 0 {
			scan = "," + bare
		}
		for _, m := range flowKeyRe.FindAllStringSubmatch(scan, -1) {
			for _, key := range submatches(m) {
				if keys[key] {
					return i + 1, key, true
				}
			}
		}
		depth += strings.Count(bare, "{") + strings.Count(bare, "[") -
			strings.Count(bare, "}") - strings.Count(bare, "]")
		if depth < 0 {
			depth = 0
		}
	}
	return 0, "", false
}

// sectionSpan is the half-open line range one heading OWNS: the heading itself
// through everything under it, ending at the next heading of the same level or
// shallower. Indices are 0-based over the whole document, frontmatter included.
//
// It is shared by the redactor and the projector because the two must agree
// about where a section ends. The section scan's own Body ends at the next
// heading of ANY level, which is right for rendering a page and wrong here: a
// `###` under a projected `##` would be dropped from the item while the redactor
// treated it as part of the section it belongs to, so a field would travel short
// and the manifest would hash the short version.
func sectionSpan(sections []site.Section, i, total int) (int, int) {
	start := sections[i].Line - 1
	for _, next := range sections[i+1:] {
		if next.Level > 0 && next.Level <= sections[i].Level {
			return start, next.Line - 1
		}
	}
	return start, total
}

// projectField extracts one named field from a record's text. Only a record is
// ever projected, and a record is markdown, so the same scope holds here.
func projectField(rel, doc, field string) (string, bool, error) {
	body, offset := site.StripFrontmatter(doc)
	sections, err := site.Sections(rel, body, offset)
	if err != nil {
		return "", false, fmt.Errorf("reading: projecting %s from %s: %w", field, rel, err)
	}
	lines := strings.Split(doc, "\n")
	for i, sec := range sections {
		if sec.Level < 2 || sec.Title != field {
			continue
		}
		start, end := sectionSpan(sections, i, len(lines))
		return trimBlankEdges(lines[min(start+1, len(lines)):min(end, len(lines))]), true, nil
	}
	fields := frontmatter.Fields(strings.Split(doc, "\n"))
	if f, ok := fields[field]; ok && !frontmatter.IsNull(f.Value) {
		return f.Value, true, nil
	}
	return "", false, nil
}
