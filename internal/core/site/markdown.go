package site

// The Markdown subset renderer.
//
// The site renders repository prose, and the repository writes a small, stable
// dialect: ATX headings, paragraphs, bold/italic/code spans, links, images,
// fenced code, pipe tables, blockquotes and lists. That dialect is what this
// renders, and everything else is a BUILD ERROR naming file and line.
//
// Refusing is the point. A renderer that passes an unknown construct through
// unrendered publishes raw markdown to readers; a renderer that silently drops
// it publishes a hole. Both are worse than a build that stops and says
// "docs/explanation/roles.md:31 — reference link", because the fix for that is
// one edit and the fix for a silent hole is noticing it exists.

import (
	"fmt"
	"regexp"
	"strings"
)

// UnsupportedError is a markdown construct outside the rendered subset.
type UnsupportedError struct {
	Path      string
	Line      int
	Construct string
	Detail    string
}

func (e *UnsupportedError) Error() string {
	d := ""
	if e.Detail != "" {
		d = ": " + e.Detail
	}
	return fmt.Sprintf("%s:%d: unsupported markdown construct: %s%s — the site renders a fixed subset and never passes text through unrendered",
		e.Path, e.Line, e.Construct, d)
}

var (
	tableDelimRe   = regexp.MustCompile(`^\s*\|?\s*:?-{1,}:?\s*(\|\s*:?-{1,}:?\s*)*\|?\s*$`)
	orderedItemRe  = regexp.MustCompile(`^(\d+)[.)]\s+(.*)$`)
	setextRe       = regexp.MustCompile(`^\s{0,3}(=+|-{2,})\s*$`)
	thematicRe     = regexp.MustCompile(`^\s{0,3}((\*\s*){3,}|(-\s*){3,}|(_\s*){3,})$`)
	rawHTMLStartRe = regexp.MustCompile(`^\s*<[A-Za-z!/]`)
	linkDefRe      = regexp.MustCompile(`^\s{0,3}\[([^\]]+)\]:\s*(\S+)\s*(".*"|'.*'|\(.*\))?\s*$`)
	autolinkRe     = regexp.MustCompile(`^<([A-Za-z][A-Za-z0-9+.\-]{1,31}:[^<>\x00-\x20]*)>`)
	emailAutoRe    = regexp.MustCompile(`^<([A-Za-z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)*)>`)
	htmlTagRe      = regexp.MustCompile(`^</?([A-Za-z][A-Za-z0-9-]*)`)
)

// htmlElements is the set of names a `<…>` may carry and still be HTML.
//
// The distinction matters because the record writes angle brackets for two
// different things. `<div>` is markup, and rendering it would let a record file
// inject elements into a published page — that stays a refusal. `<ts>`,
// `<file>` and `<number>` are PLACEHOLDERS in prose, and every generic HTML
// sanitiser drops them silently, which is the one outcome worse than either
// rendering or refusing: the sentence loses a word and nobody can tell. Those
// are escaped and shown, so what the record wrote is what a reader sees.
var htmlElements = map[string]bool{}

func init() {
	for _, n := range strings.Fields(`a abbr address area article aside audio b base bdi bdo blockquote body br
button canvas caption cite code col colgroup data datalist dd del details dfn dialog div dl dt em embed
fieldset figcaption figure footer form h1 h2 h3 h4 h5 h6 head header hgroup hr html i iframe img input ins
kbd label legend li link main map mark menu meta meter nav noscript object ol optgroup option output p
picture pre progress q rp rt ruby s samp script search section select slot small source span strong style
sub summary sup table tbody td template textarea tfoot th thead time title tr track u ul var video wbr`) {
		htmlElements[n] = true
	}
}

// LinkDefinitions collects a document's link reference definitions — the
// `[label]: destination "title"` lines the record's older entries keep at the
// foot of a file and name from the prose above.
//
// They are document-level, and the renderer works a block at a time, so the
// caller reads them once and hands them to the Renderer. A reference whose
// definition is missing stays a refusal: the destination is lost either way,
// and losing it silently is how a dead cross-reference survives review.
func LinkDefinitions(md string) map[string]string {
	defs := map[string]string{}
	for _, line := range strings.Split(md, "\n") {
		m := linkDefRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		key := normaliseLabel(m[1])
		if _, seen := defs[key]; !seen {
			defs[key] = strings.Trim(m[2], "<>")
		}
	}
	return defs
}

// normaliseLabel folds a reference label the way every reader folds it: case
// insensitively, with internal whitespace collapsed.
func normaliseLabel(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

// isLinkDefLine reports whether a line is nothing but a link reference
// definition. A block made only of them renders as nothing, which is what it
// means everywhere else.
func isLinkDefLine(s string) bool { return linkDefRe.MatchString(s) }

// Renderer turns the markdown subset into the site's HTML. Its two hooks are
// the places the site differs from a generic renderer: an image becomes a
// committed asset (inlined SVG or copied raster) rather than a bare <img>, and
// a repo-relative link becomes a site route.
type Renderer struct {
	// UI is the closed allowlist of interface strings; the renderer adds the
	// copy-button label and nothing else.
	UI UI
	// Image renders one image reference. src is as written in the markdown,
	// relative to the page; alt is its alt text.
	Image func(src, alt string, at Source) (string, error)
	// Link rewrites one href. It never fails: an href it does not recognise is
	// left exactly as the record wrote it.
	Link func(href string, at Source) string
	// Refs are the document's link reference definitions, from LinkDefinitions.
	// A nil map means the document defines none, and every reference link in it
	// is a refusal.
	Refs map[string]string
}

// Source is a position in a repository file.
type Source struct {
	Path string
	Line int
}

// RenderBlocks renders a block sequence in order.
func (r *Renderer) RenderBlocks(path string, blocks []Block) (string, error) {
	var b strings.Builder
	for _, blk := range blocks {
		h, err := r.RenderBlock(path, blk)
		if err != nil {
			return "", err
		}
		b.WriteString(h)
	}
	return b.String(), nil
}

// unrenderedFenceRe matches a fence opener the renderer does not render: a
// tilde run, or a backtick run longer than three, at any indent.
var unrenderedFenceRe = regexp.MustCompile("^[ \t]*(~{3,}|`{4,})")

// RenderBlock renders one top-level block.
func (r *Renderer) RenderBlock(path string, blk Block) (string, error) {
	at := Source{Path: path, Line: blk.Line}
	lines := strings.Split(blk.Text, "\n")
	first := lines[0]

	// The walk that cut this block reads fences by mdrecord's rule, tildes and
	// longer backtick runs included, and this renderer renders one form: a
	// three-backtick run at column 0. Any other form arriving here would render
	// as a paragraph, delimiters and code inlined into prose, with no error, so
	// it is refused. A three-backtick fence's own body is code and is not read
	// (iss-2609251514129841).
	if !strings.HasPrefix(first, "```") || strings.HasPrefix(first, "````") {
		for i, ln := range lines {
			if unrenderedFenceRe.MatchString(ln) {
				return "", &UnsupportedError{at.Path, at.Line + i, "fenced code block opened by a tilde or a run of four or more backticks",
					"only a three-backtick fence renders; any other opener renders as a paragraph"}
			}
		}
	}

	// A fence must open its own block. Without a blank line before it the block
	// walk never sees it start, so the whole run — prose, backticks and code —
	// arrives here as one paragraph, and every backtick would be escaped into the
	// page as visible punctuation with the code inlined into the sentence.
	if !strings.HasPrefix(first, "```") {
		for i, ln := range lines {
			if strings.HasPrefix(ln, "```") {
				return "", &UnsupportedError{at.Path, at.Line + i, "fenced code block without a blank line before it",
					"a fence opens its own block; without the blank line the code renders as part of the paragraph above"}
			}
		}
	}

	switch {
	case strings.HasPrefix(first, "```"):
		return r.fence(at, lines)
	case strings.HasPrefix(strings.TrimSpace(first), "<!--"):
		return r.commentBlock(at, lines)
	case allLinkDefs(lines):
		// A block of nothing but link reference definitions is invisible in
		// every reader; its content is already carried by r.Refs.
		return "", nil
	case headingRe.MatchString(first):
		return r.heading(at, first)
	case thematicRe.MatchString(first):
		// A rule opening a block is a rule. The same characters AFTER a line of
		// text are a setext heading instead, which the paragraph renderer reads.
		if len(lines) == 1 {
			return "<hr>", nil
		}
		rest, err := r.RenderBlock(path, Block{Text: strings.Join(lines[1:], "\n"), Line: blk.Line + 1})
		if err != nil {
			return "", err
		}
		return "<hr>" + rest, nil
	case strings.HasPrefix(strings.TrimSpace(first), ">"):
		return r.blockquote(at, lines)
	case strings.HasPrefix(strings.TrimSpace(first), "|"):
		return r.table(at, lines)
	case isUnorderedItem(first) || orderedItemRe.MatchString(first):
		return r.list(at, lines)
	}
	return r.paragraph(at, lines)
}

// allLinkDefs reports whether every non-blank line of a block is a link
// reference definition.
func allLinkDefs(lines []string) bool {
	any := false
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		if !isLinkDefLine(ln) {
			return false
		}
		any = true
	}
	return any
}

// commentBlock drops an HTML comment and renders whatever follows it. The
// record uses them for machine annotations — review receipts, scope notes — and
// no reader has ever shown one, so showing one here would publish a note
// written for a tool.
func (r *Renderer) commentBlock(at Source, lines []string) (string, error) {
	text := strings.Join(lines, "\n")
	i := strings.Index(text, "-->")
	if i < 0 {
		return "", &UnsupportedError{at.Path, at.Line, "unterminated HTML comment",
			"it swallows the rest of the block, so the text after it silently stops existing"}
	}
	rest := strings.TrimLeft(text[i+3:], "\n")
	if strings.TrimSpace(rest) == "" {
		return "", nil
	}
	skipped := strings.Count(text[:i+3], "\n")
	var out strings.Builder
	for _, b := range Blocks(rest, at.Line+skipped) {
		h, err := r.RenderBlock(at.Path, b)
		if err != nil {
			return "", err
		}
		out.WriteString(h)
	}
	return out.String(), nil
}

// fence renders a fenced code block as a copyable command block: the button's
// label is a ui.json string, and its payload is the block's own raw text.
func (r *Renderer) fence(at Source, lines []string) (string, error) {
	info := strings.TrimSpace(strings.TrimPrefix(lines[0], "```"))
	if strings.ContainsAny(info, " \t") {
		return "", &UnsupportedError{at.Path, at.Line, "fenced-code info string", "only a bare language word is rendered, got " + quote(info)}
	}
	body := lines[1:]
	if n := len(body); n > 0 && strings.HasPrefix(strings.TrimSpace(body[n-1]), "```") {
		body = body[:n-1]
	}
	raw := strings.Join(body, "\n")
	cls := ""
	if info != "" {
		cls = ` class="language-` + escapeAttr(info) + `"`
	}
	// The button's two labels both travel in the markup: the page's script adds
	// no words of its own, so every string a reader sees comes from ui.json.
	return `<div class="cmd"><pre><code` + cls + `>` + escapeText(raw) + "\n" +
		`</code></pre><button class="copy" data-copy="` + escapeAttr(raw) +
		`" data-copied="` + escapeAttr(r.UI.Copied) + `">` + escapeText(r.UI.Copy) + `</button></div>`, nil
}

// heading renders an ATX heading.
func (r *Renderer) heading(at Source, line string) (string, error) {
	m := headingRe.FindStringSubmatch(line)
	level := len(m[1])
	inner, err := r.inline(at, strings.TrimSpace(m[2]))
	if err != nil {
		return "", err
	}
	tag := fmt.Sprintf("h%d", level)
	return "<" + tag + ">" + inner + "</" + tag + ">", nil
}

// blockquote renders a `>` block by stripping the marker and rendering what is
// left as blocks in their own right.
//
// That recursion is what carries the structures a quote legitimately holds —
// paragraphs, lists, a quote inside a quote, a table. Flattening any of them
// would publish a document the source does not have, with nothing on the page to
// tell a reader that it happened.
//
// A LAZY continuation — a line inside the block that drops the marker — is still
// refused. It is genuinely ambiguous: the same bytes are a quote in one reader
// and a quote followed by a paragraph in another, and guessing publishes one of
// them as if it were certain.
func (r *Renderer) blockquote(at Source, lines []string) (string, error) {
	var stripped []string
	for i, ln := range lines {
		t := strings.TrimSpace(ln)
		if !strings.HasPrefix(t, ">") {
			return "", &UnsupportedError{at.Path, at.Line + i, "lazy blockquote continuation", "every line of a quoted block must start with '>'"}
		}
		stripped = append(stripped, strings.TrimPrefix(strings.TrimPrefix(t, ">"), " "))
	}
	var b strings.Builder
	b.WriteString("<blockquote>\n")
	for _, blk := range Blocks(strings.Join(stripped, "\n"), at.Line) {
		h, err := r.RenderBlock(at.Path, blk)
		if err != nil {
			return "", err
		}
		if h == "" {
			continue
		}
		b.WriteString(h + "\n")
	}
	b.WriteString("</blockquote>")
	return b.String(), nil
}

// tableCells splits one table row into its cells, honouring `\|` as content.
// An escaped pipe is a pipe a cell contains, not a column boundary, and
// splitting on it shifts every cell after it one column to the left — a table
// that is quietly wrong, which is worse than one that fails to build. The
// escape itself is left in place for the inline pass, which already knows `\|`
// means `|`.
func tableCells(ln string) []string {
	t := strings.TrimSpace(ln)
	t = strings.TrimPrefix(t, "|")
	t = strings.TrimSuffix(t, "|")
	var out []string
	var cur strings.Builder
	for i := 0; i < len(t); i++ {
		switch {
		case t[i] == '\\' && i+1 < len(t):
			cur.WriteByte(t[i])
			cur.WriteByte(t[i+1])
			i++
		case t[i] == '|':
			out = append(out, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteByte(t[i])
		}
	}
	return append(out, strings.TrimSpace(cur.String()))
}

// table renders a pipe table, inside its own overflow container.
//
// The container is emitted HERE rather than by each layout, because a table is
// the one block whose width its author does not choose: the widest cell decides
// it, and without a box to scroll inside, that cell sets the width of the whole
// page on a phone. Every consumer — a composed chapter, a quoted span, a record
// body rendered verbatim — gets the same treatment, so no new caller has to
// remember.
//
// Column alignment is not rendered — the site's stylesheet aligns every column
// the same way — so an alignment row is accepted and its colons ignored.
func (r *Renderer) table(at Source, lines []string) (string, error) {
	if len(lines) < 2 || !tableDelimRe.MatchString(lines[1]) {
		return "", &UnsupportedError{at.Path, at.Line, "pipe table", "a table needs a header row and a |---|---| delimiter row"}
	}
	var b strings.Builder
	b.WriteString(`<div class="tablewrap">` + "<table>\n<thead>\n<tr>\n")
	for _, c := range tableCells(lines[0]) {
		inner, err := r.inline(Source{at.Path, at.Line}, c)
		if err != nil {
			return "", err
		}
		b.WriteString("<th>" + inner + "</th>\n")
	}
	b.WriteString("</tr>\n</thead>\n<tbody>\n")
	for i, ln := range lines[2:] {
		b.WriteString("<tr>\n")
		for _, c := range tableCells(ln) {
			inner, err := r.inline(Source{at.Path, at.Line + 2 + i}, c)
			if err != nil {
				return "", err
			}
			b.WriteString("<td>" + inner + "</td>\n")
		}
		b.WriteString("</tr>\n")
	}
	b.WriteString("</tbody>\n</table></div>")
	return b.String(), nil
}

// list renders an unordered or ordered list, nesting and all.
//
// Nesting is carried by INDENTATION, so each item's continuation lines are
// dedented and handed back to the block walk: a list inside an item is then a
// list block like any other, and an item that carries a table or a quoted line
// gets it rendered rather than flattened into the sentence.
func (r *Renderer) list(at Source, lines []string) (string, error) {
	ordered := orderedItemRe.MatchString(strings.TrimLeft(lines[0], " \t"))
	base := indentOf(lines[0])

	type item struct {
		body []string
		line int
	}
	var items []item
	for i, ln := range lines {
		trimmed := strings.TrimLeft(ln, " \t")
		isItem := isUnorderedItem(trimmed) || orderedItemRe.MatchString(trimmed)
		if isItem && indentOf(ln) <= base {
			if orderedItemRe.MatchString(trimmed) != ordered {
				return "", &UnsupportedError{at.Path, at.Line + i, "mixed list", "a list is ordered or unordered, not both"}
			}
			rest := trimmed[2:]
			if ordered {
				rest = orderedItemRe.FindStringSubmatch(trimmed)[2]
			}
			items = append(items, item{body: []string{strings.TrimSpace(rest)}, line: at.Line + i})
			continue
		}
		if len(items) == 0 {
			return "", &UnsupportedError{at.Path, at.Line + i, "list", "a list item starts with '- ' or '1. '"}
		}
		items[len(items)-1].body = append(items[len(items)-1].body, ln)
	}

	tag := "ul"
	if ordered {
		tag = "ol"
	}
	var b strings.Builder
	b.WriteString("<" + tag + ">\n")
	for _, it := range items {
		inner, err := r.listItem(Source{at.Path, it.line}, it.body)
		if err != nil {
			return "", err
		}
		b.WriteString("<li>" + inner + "</li>\n")
	}
	b.WriteString("</" + tag + ">")
	return b.String(), nil
}

// listItem renders one item: its own lead text as a sentence, and anything with
// structure of its own — a nested list above all — as blocks under it.
func (r *Renderer) listItem(at Source, body []string) (string, error) {
	body = dedentContinuations(body)
	// The lead runs until a line that opens a structure of its own. A nested
	// list may follow the sentence with no blank line between them, which is
	// exactly how the record writes one.
	split := len(body)
	for i := 1; i < len(body); i++ {
		t := strings.TrimLeft(body[i], " \t")
		if isUnorderedItem(t) || orderedItemRe.MatchString(t) || strings.HasPrefix(t, "```") ||
			strings.HasPrefix(t, ">") || strings.HasPrefix(t, "|") || headingRe.MatchString(t) {
			split = i
			break
		}
	}
	lead, err := r.inline(at, strings.Join(body[:split], "\n"))
	if err != nil {
		return "", err
	}
	if split == len(body) {
		return lead, nil
	}
	var b strings.Builder
	b.WriteString(lead)
	for _, blk := range Blocks(strings.Join(body[split:], "\n"), at.Line+split) {
		h, err := r.RenderBlock(at.Path, blk)
		if err != nil {
			return "", err
		}
		b.WriteString(h)
	}
	return b.String(), nil
}

// dedentContinuations strips the shared indentation from an item's continuation
// lines, so a nested list reads as a list rather than as indented prose. The
// first line has already had its marker removed and is left alone.
func dedentContinuations(body []string) []string {
	least := -1
	for _, ln := range body[1:] {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		if n := indentOf(ln); least < 0 || n < least {
			least = n
		}
	}
	if least <= 0 {
		return body
	}
	out := append([]string{}, body[:1]...)
	for _, ln := range body[1:] {
		cut := 0
		for cut < least && cut < len(ln) && (ln[cut] == ' ' || ln[cut] == '\t') {
			cut++
		}
		out = append(out, ln[cut:])
	}
	return out
}

// indentOf counts a line's leading spaces, a tab counting as one.
func indentOf(s string) int {
	n := 0
	for n < len(s) && (s[n] == ' ' || s[n] == '\t') {
		n++
	}
	return n
}

// paragraph renders a run of prose.
//
// Two block constructs can only be recognised HERE, because both are a line of
// text plus the line under it: a setext heading (`Title` over `===` or `---`)
// and, in every reader the record is read in, the same underline turning the
// text above it into a heading rather than into a rule. The paragraph is cut at
// the underline and the head becomes the heading.
//
// A RAW HTML block is refused, and only when it opens the block: the same
// pattern in the middle of a paragraph is a code span wrapped across a line, and
// refusing that would fail the build on correct prose.
func (r *Renderer) paragraph(at Source, lines []string) (string, error) {
	for i, ln := range lines {
		// A LIST may interrupt a paragraph, and the record writes one that way
		// constantly — a bold lead-in and its bullets under it with no blank
		// line between. Rendering the whole run as one paragraph flattens the
		// bullets into stray hyphens in a run-on sentence, which is a different
		// document from the one on the forge.
		if i > 0 && listInterrupts(ln) {
			head, err := r.paragraph(at, lines[:i])
			if err != nil {
				return "", err
			}
			rest, err := r.RenderBlock(at.Path, Block{Text: strings.Join(lines[i:], "\n"), Line: at.Line + i})
			if err != nil {
				return "", err
			}
			return head + rest, nil
		}
		if i > 0 && setextRe.MatchString(ln) {
			level := 2
			if strings.Contains(ln, "=") {
				level = 1
			}
			head, err := r.inline(at, strings.Join(lines[:i], "\n"))
			if err != nil {
				return "", err
			}
			tag := fmt.Sprintf("h%d", level)
			out := "<" + tag + ">" + head + "</" + tag + ">"
			if i+1 >= len(lines) {
				return out, nil
			}
			rest, err := r.RenderBlock(at.Path, Block{Text: strings.Join(lines[i+1:], "\n"), Line: at.Line + i + 1})
			if err != nil {
				return "", err
			}
			return out + rest, nil
		}
		if i == 0 && rawHTMLStartRe.MatchString(ln) && isHTMLElementStart(strings.TrimLeft(ln, " \t")) {
			return "", &UnsupportedError{at.Path, at.Line + i, "raw HTML block", "every picture is a committed asset and every element is generated"}
		}
		if i == 0 && (strings.HasPrefix(ln, "    ") || strings.HasPrefix(ln, "\t")) {
			return "", &UnsupportedError{at.Path, at.Line, "indented code block", "fence code with ``` so it renders as a copyable command"}
		}
	}
	inner, err := r.inline(at, strings.Join(lines, "\n"))
	if err != nil {
		return "", err
	}
	// An image on its own line is a block-level figure, exactly as the source
	// markdown reads it; the layouts detect one by this shape.
	return "<p>" + inner + "</p>", nil
}

// isHTMLElementStart reports whether a `<…` opens a real HTML element rather
// than a prose placeholder.
//
// The element name alone does not settle it: the record writes
// `collaborators/<u>/permission` in a sentence, and `u` happens to be an HTML
// element. Real markup shows itself — it carries attributes, it closes itself,
// or a matching closing tag follows it. A bare known name with none of those is
// a placeholder, and is shown as the record wrote it.
func isHTMLElementStart(s string) bool {
	if strings.HasPrefix(s, "<!") {
		return !strings.HasPrefix(s, "<!--")
	}
	m := htmlTagRe.FindStringSubmatch(s)
	if m == nil {
		return false
	}
	name := strings.ToLower(m[1])
	if !htmlElements[name] {
		return false
	}
	if strings.HasPrefix(s, "</") || voidElements[name] {
		return true
	}
	end := strings.IndexByte(s, '>')
	if end < 0 {
		return false
	}
	if strings.TrimSpace(s[len(m[0]):end]) != "" {
		return true
	}
	return strings.Contains(strings.ToLower(s[end:]), "</"+name+">")
}

// voidElements are the HTML elements that never carry a closing tag, so their
// bare form is still markup.
var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true, "hr": true,
	"img": true, "input": true, "link": true, "meta": true, "source": true,
	"track": true, "wbr": true,
}

// listInterrupts reports whether a line opening a list may cut the paragraph
// above it.
//
// A bullet always may. An ordered item may only when it is numbered ONE, which
// is CommonMark's rule and exists for exactly the sentence the record also
// writes: a paragraph that wraps onto a line beginning "2. " is prose, not a
// list, and cutting it there would invent a list nobody wrote.
func listInterrupts(ln string) bool {
	if isUnorderedItem(ln) {
		return true
	}
	m := orderedItemRe.FindStringSubmatch(ln)
	return m != nil && m[1] == "1"
}

// isUnorderedItem reports whether a line opens an unordered list item.
func isUnorderedItem(ln string) bool {
	return len(ln) > 2 && (ln[0] == '-' || ln[0] == '*' || ln[0] == '+') && ln[1] == ' '
}

// inlineNode is one piece of a scanned span run: either finished HTML, or a
// run of emphasis delimiters still looking for its partner.
type inlineNode struct {
	// html is the rendered output of a finished piece.
	html string
	// delim is '*' or '_' for a delimiter run, and 0 otherwise.
	delim byte
	// n is how many characters of the run are still unspent; origN the run's
	// original length, which the matching rules are stated in terms of.
	n, origN int
	// canOpen and canClose are the matching walk's state: what this run may
	// still do. origOpen and origClose are the flanking rules' verdict, which is
	// a fact about the text and never changes.
	canOpen, canClose   bool
	origOpen, origClose bool
	// before holds closing tags and after opening tags, so a run that both
	// closes an inner span and opens nothing still emits them in the right
	// order around whatever characters it has left over.
	before, after string
}

// inline renders the span-level subset in two passes.
//
// The first pass reads left to right and finishes everything whose boundaries
// are unambiguous — escapes, code spans, autolinks, images, links — leaving the
// emphasis delimiters as unspent runs. The second pass matches those runs.
//
// The order is what makes `**`+"`"+`.abcd/**`+"`"+` stays**` render: a code span binds
// tighter than emphasis in every reader, so the asterisks inside one are
// content, and a single left-to-right pass that met the `**` first would take
// its closer from inside the code and then refuse an unclosed span.
func (r *Renderer) inline(at Source, s string) (string, error) {
	nodes, err := r.scanInline(at, s)
	if err != nil {
		return "", err
	}
	matchEmphasis(nodes)
	var b strings.Builder
	for _, nd := range nodes {
		if nd.delim == 0 {
			b.WriteString(nd.html)
			continue
		}
		b.WriteString(nd.before)
		if nd.n > 0 {
			b.WriteString(strings.Repeat(string(nd.delim), nd.n))
		}
		b.WriteString(nd.after)
	}
	return b.String(), nil
}

// scanInline is the first pass.
func (r *Renderer) scanInline(at Source, s string) ([]*inlineNode, error) {
	var nodes []*inlineNode
	text := func(h string) {
		if n := len(nodes); n > 0 && nodes[n-1].delim == 0 {
			nodes[n-1].html += h
			return
		}
		nodes = append(nodes, &inlineNode{html: h})
	}
	for i := 0; i < len(s); {
		switch c := s[i]; {
		case c == '\\' && i+1 < len(s) && isEscapable(s[i+1]):
			text(escapeText(string(s[i+1])))
			i += 2
		case c == '`':
			n := runLen(s, i, '`')
			end := strings.Index(s[i+n:], s[i:i+n])
			if end < 0 {
				return nil, &UnsupportedError{at.Path, at.Line, "unclosed code span", quote(clip(s[i:]))}
			}
			code := s[i+n : i+n+end]
			if n > 1 {
				code = strings.Trim(code, " ")
			}
			text("<code>" + escapeText(code) + "</code>")
			i += n + end + n
		case c == '!' && i+1 < len(s) && s[i+1] == '[':
			label, href, next, kind := parseLink(s, i+1)
			switch kind {
			case linkLiteral:
				text("!")
				i++
				continue
			case linkRefused, linkRef:
				return nil, &UnsupportedError{at.Path, at.Line, "image", "only ![alt](src) is rendered, got " + quote(clip(s[i:]))}
			}
			html, err := r.Image(href, label, at)
			if err != nil {
				return nil, err
			}
			text(html)
			i = next
		case c == '[':
			label, href, next, kind := parseLink(s, i)
			switch kind {
			case linkLiteral:
				// A bracket that opens no link is a bracket. Refusing it would
				// blocker-fail the build over prose like "the array [0]", which
				// every markdown reader renders literally.
				text("[")
				i++
				continue
			case linkRef:
				dest, ok := r.Refs[normaliseLabel(href)]
				if !ok {
					// A document that defines no references is not writing one:
					// `[A-Za-z][A-Za-z0-9._-]*` is a regular expression, and
					// every reader shows it as the brackets it is. A document
					// that DOES define references is held to them, so a typo in
					// a label fails the build rather than quietly losing a
					// destination.
					if len(r.Refs) == 0 {
						text("[")
						i++
						continue
					}
					return nil, &UnsupportedError{at.Path, at.Line, "link",
						"the reference " + quote(href) + " has no [" + href + "]: definition in this document"}
				}
				href = dest
			case linkRefused:
				return nil, &UnsupportedError{at.Path, at.Line, "link",
					"only [text](href) and a defined [text][ref] are rendered; footnotes and inline link titles are not in the subset"}
			}
			h, err := r.anchor(at, label, href)
			if err != nil {
				return nil, err
			}
			text(h)
			i = next
		case c == '*' || c == '_':
			n := runLen(s, i, c)
			opens, closes := flanking(s, i, n, c)
			nodes = append(nodes, &inlineNode{delim: c, n: n, origN: n,
				canOpen: opens, canClose: closes, origOpen: opens, origClose: closes})
			i += n
		case c == '<':
			h, next, err := r.angle(at, s, i)
			if err != nil {
				return nil, err
			}
			text(h)
			i = next
		default:
			j := i
			for j < len(s) && !strings.ContainsRune("\\`![*_<", rune(s[j])) {
				j++
			}
			if j == i {
				j = i + 1
			}
			text(escapeText(s[i:j]))
			i = j
		}
	}
	return nodes, nil
}

// anchor renders one link, refusing the two shapes HTML cannot carry.
func (r *Renderer) anchor(at Source, label, href string) (string, error) {
	if scheme, bad := executableScheme(href); bad {
		return "", &UnsupportedError{at.Path, at.Line, "link scheme",
			quote(scheme) + " runs code in the reader's browser; the site links to pages and files"}
	}
	inner, err := r.inline(at, label)
	if err != nil {
		return "", err
	}
	// HTML has no nested anchor: a browser closes the outer one at the inner, so
	// the markup a reader gets is not the markup written here and the outer
	// link's tail stops being clickable. Markdown itself forbids the construct;
	// this says so instead of emitting it.
	if strings.Contains(inner, "<a ") {
		return "", &UnsupportedError{at.Path, at.Line, "nested link",
			"HTML has no anchor inside an anchor; the browser would silently close the outer one"}
	}
	return `<a href="` + escapeAttr(r.Link(href, at)) + `">` + inner + "</a>", nil
}

// angle reads a `<` and decides which of the four things it is: an autolink, an
// HTML comment (dropped, as every reader drops one), an HTML element (refused),
// or a prose placeholder like `<ts>` (shown as the record wrote it).
func (r *Renderer) angle(at Source, s string, i int) (string, int, error) {
	rest := s[i:]
	if strings.HasPrefix(rest, "<!--") {
		end := strings.Index(rest, "-->")
		if end < 0 {
			return "", 0, &UnsupportedError{at.Path, at.Line, "unterminated HTML comment",
				"it swallows the rest of the line, so the text after it silently stops existing"}
		}
		return "", i + end + 3, nil
	}
	if m := autolinkRe.FindStringSubmatch(rest); m != nil {
		h, err := r.anchor(at, m[1], m[1])
		if err != nil {
			return "", 0, err
		}
		return h, i + len(m[0]), nil
	}
	if m := emailAutoRe.FindStringSubmatch(rest); m != nil {
		h, err := r.anchor(at, m[1], "mailto:"+m[1])
		if err != nil {
			return "", 0, err
		}
		return h, i + len(m[0]), nil
	}
	if isHTMLElementStart(rest) {
		return "", 0, &UnsupportedError{at.Path, at.Line, "inline HTML or autolink",
			"wrap it in a code span or write it as [text](href): " + quote(clip(rest))}
	}
	return "&lt;", i + 1, nil
}

// flanking applies CommonMark's left- and right-flanking rules to a delimiter
// run: whether it may open emphasis, close it, or neither. It is what keeps
// `snake_case` a word and `npx-create-*` a filename pattern.
func flanking(s string, i, n int, c byte) (canOpen, canClose bool) {
	before := byte(' ')
	if i > 0 {
		before = s[i-1]
	}
	after := byte(' ')
	if i+n < len(s) {
		after = s[i+n]
	}
	left := !isSpace(after) && (!isPunct(after) || isSpace(before) || isPunct(before))
	right := !isSpace(before) && (!isPunct(before) || isSpace(after) || isPunct(after))
	if c == '*' {
		return left, right
	}
	return left && (!right || isPunct(before)), right && (!left || isPunct(after))
}

// matchEmphasis is the second pass: CommonMark's closer-then-opener walk over
// the unspent delimiter runs. A run that finds no partner keeps its characters
// and renders as the punctuation it is.
func matchEmphasis(nodes []*inlineNode) {
	for ci := 0; ci < len(nodes); ci++ {
		c := nodes[ci]
		if c.delim == 0 || !c.canClose || c.n == 0 {
			continue
		}
		oi := -1
		for j := ci - 1; j >= 0; j-- {
			o := nodes[j]
			if o.delim != c.delim || !o.canOpen || o.n == 0 {
				continue
			}
			// The "rule of three": a run that can both open and close only
			// pairs with one whose combined length is not a multiple of three,
			// unless both lengths are. It is what stops `**a*b**` pairing the
			// wrong two runs.
			//
			// It is asked of the FLANKING verdicts, which never change, and not
			// of canOpen/canClose, which the walk turns off as it consumes runs:
			// reading the mutable pair here made a closer that had already given
			// up look like a run that cannot open, and the wrong two runs paired.
			if (c.origOpen || o.origClose) && (o.origN+c.origN)%3 == 0 &&
				!(o.origN%3 == 0 && c.origN%3 == 0) {
				continue
			}
			oi = j
			break
		}
		if oi < 0 {
			c.canClose = false
			continue
		}
		o := nodes[oi]
		tag := "em"
		use := 1
		if o.n >= 2 && c.n >= 2 {
			tag, use = "strong", 2
		}
		o.n -= use
		c.n -= use
		o.after = "<" + tag + ">" + o.after
		c.before += "</" + tag + ">"
		// A run trapped inside the pair can no longer match anything, but its
		// CHARACTERS are still characters: `**MET_WITH_CONCERNS.**` keeps its
		// underscores, exactly as every reader keeps them. Zeroing `n` here
		// deleted them from the page instead — a silent difference between the
		// record and what a reader was shown, which is the one thing this
		// renderer exists to prevent.
		for j := oi + 1; j < ci; j++ {
			if nodes[j].delim != 0 {
				nodes[j].canOpen, nodes[j].canClose = false, false
			}
		}
		if c.n > 0 {
			// The same closer may still close an outer span.
			ci--
		}
	}
}

// isPunct reports whether a byte is ASCII punctuation, the class the flanking
// rules are stated over.
func isPunct(c byte) bool {
	return (c >= '!' && c <= '/') || (c >= ':' && c <= '@') ||
		(c >= '[' && c <= '`') || (c >= '{' && c <= '~')
}

// The three things a '[' can turn out to be.
const (
	// linkInline is `[text](href)`: rendered.
	linkInline = iota
	// linkRefused is a construct that MEANS a link but is not in the subset — a
	// reference link, a footnote, a link title. Rendering it literally would
	// publish the markup; dropping it would lose the destination.
	linkRefused
	// linkLiteral is a bracket that opens no link at all. It is text.
	linkLiteral
	// linkRef is `[text][label]` or `[text][]`: the destination is defined
	// elsewhere in the document, and the caller resolves it.
	linkRef
)

// parseLink classifies a '[' at i and, for an inline link, returns its text,
// its href, and the index just past the closing paren.
func parseLink(s string, i int) (text, href string, next, kind int) {
	depth := 0
	j := i
	for ; j < len(s); j++ {
		switch s[j] {
		case '[':
			depth++
		case ']':
			depth--
		}
		if depth == 0 {
			break
		}
	}
	if j >= len(s) || j+1 >= len(s) {
		return "", "", 0, linkLiteral
	}
	switch s[j+1] {
	case '(':
		// An inline link, or an attempt at one.
	case '[':
		// `[text][label]` — a reference link. The label is what sits in the
		// second pair of brackets, or the text itself when they are empty.
		end := strings.IndexByte(s[j+2:], ']')
		if end < 0 {
			return "", "", 0, linkLiteral
		}
		text = s[i+1 : j]
		label := s[j+2 : j+2+end]
		if strings.TrimSpace(label) == "" {
			label = text
		}
		return text, label, j + 2 + end + 1, linkRef
	default:
		return "", "", 0, linkLiteral
	}
	text = s[i+1 : j]
	k := j + 2
	pd := 1
	for ; k < len(s); k++ {
		if s[k] == '(' {
			pd++
		} else if s[k] == ')' {
			pd--
			if pd == 0 {
				break
			}
		}
	}
	if k >= len(s) {
		return "", "", 0, linkRefused
	}
	href = strings.TrimSpace(s[j+2 : k])
	if href == "" || strings.ContainsAny(href, " \t") {
		// A link title (`[t](u "title")`) is not in the subset: it renders as an
		// attribute nothing on the site reads, and hiding prose in one would
		// smuggle unsourced text onto the page.
		return "", "", 0, linkRefused
	}
	return text, href, k + 1, linkInline
}

// executableScheme reports whether an href names a scheme that executes rather
// than navigates. Escaping an attribute is not enough on its own: a perfectly
// well-formed `javascript:` href needs no quote to break out of, and the site
// renders text from a repository whose files an outside contributor can edit.
// The comparison folds case and strips the whitespace and control characters a
// browser ignores inside a scheme, because those are exactly what a bypass is
// written with.
func executableScheme(href string) (string, bool) {
	var b strings.Builder
	for i := 0; i < len(href); i++ {
		c := href[i]
		if c == ':' {
			scheme := strings.ToLower(b.String())
			switch scheme {
			case "javascript", "vbscript", "data":
				return scheme + ":", true
			}
			return "", false
		}
		if c == '/' || c == '?' || c == '#' {
			return "", false
		}
		if c <= ' ' || c == 0x7f {
			continue
		}
		b.WriteByte(c)
	}
	return "", false
}

func isEscapable(c byte) bool {
	return strings.IndexByte("\\`*_{}[]()#+-.!|<>&\"'~", c) >= 0
}

// runLen counts the run of c starting at i.
func runLen(s string, i int, c byte) int {
	n := 0
	for i+n < len(s) && s[i+n] == c {
		n++
	}
	return n
}

// escapeText escapes a text node.
func escapeText(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

// escapeAttr escapes an attribute value.
func escapeAttr(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;").Replace(s)
}

func quote(s string) string { return `"` + s + `"` }

// clip shortens a fragment for an error message.
//
// It counts RUNES, because the record is written in prose that carries
// em-dashes and accents: slicing by bytes lands mid-character and puts a
// replacement glyph in the middle of the one message somebody reads to find
// the line the build refused.
func clip(s string) string {
	const max = 40
	s = strings.ReplaceAll(s, "\n", " ")
	r := []rune(s)
	if len(r) > max {
		return string(r[:max]) + "…"
	}
	return s
}
