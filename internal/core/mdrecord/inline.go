package mdrecord

import (
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// Code spans as the page pairs them.
//
// CommonMark reads a document's blocks first and only then the inline content
// of each paragraph, so a code span pairs over the paragraph's lines: a run of
// backticks closes at the next run of exactly its length, over a line break,
// and a run nothing closes before the paragraph ends is literal backticks.
// Read pairs them that way, one paragraph at a time, as part of its one pass,
// so the HTML comment walk and the code spans it reports agree: an opener
// inside a span opens no comment, and backticks inside a comment pair with
// nothing (the construct that begins first wins, CommonMark's precedence).
//
// Which lines one paragraph holds is read from each line's form (kindOf):
//
//   - a blank line ends it, and so do a line that starts a block that may
//     interrupt a paragraph (an ATX heading, a fence delimiter, a bullet or a
//     numbered item numbered 1 with text after it, a thematic break, a setext
//     underline, all at up to three columns) and a line opening an HTML
//     comment block (`<!--` first on the line, after any list or blockquote
//     markers), which holds no code span after the comment either: the rest
//     of that line is raw HTML;
//   - an ATX heading, a thematic break, a setext underline, a comment block's
//     line and a table row each hold their own spans, so the next line starts
//     a paragraph;
//   - any other line continues the paragraph above it, lazily or not.
//
// Some forms start a block or continue the paragraph according to the
// container around them, which this package does not parse: a blockquote
// line, a line opening an HTML block, a list marker numbered other than 1 or
// with no text after it, a table row, and any block start indented four
// columns or more. Read pairs either side of such a line apart. That pairing is
// the page's as long as the stretch above the line leaves no run unpaired and
// no comment open (a stretch that pairs everything it opens reads the same
// joined to the next one or not); where it leaves one, the pairing from that
// stretch to the next sure paragraph end is open, and those lines report no
// span. A line also reports no span where its pairing turns on a construct
// read before code spans that this package does not read: inside an HTML
// block (to the next blank line, or for pre, script, style and textarea, a
// processing instruction or a CDATA section, to its closer), in a paragraph whose first line is indented four columns or
// more (indented code, or a paragraph of a list item), in a paragraph where a
// `<` that may open a tag or an autolink holds a backtick before its `>`, and
// in a table row where a span covers an unescaped `|` (a cell boundary). A
// line with no span is read whole, so each of those readings reads more of the
// page than it shows as prose, never less. A comment opener on a line whose
// pairing is open, with a backtick before it in that stretch, makes the page
// Unsure from that line (ReadPage).
//
// What this does not follow of CommonMark: entity and numeric character
// references are not decoded, so an encoded backtick is no backtick here
// either; link and image syntax is not read, which code spans outrank anyway;
// the lines of an HTML block are read for comments as markdown is, so an
// opener a span would cover there is not opened (more is read); and an inline
// raw HTML tag's own extent is not read, past the rule above.

// lineKind is how a line bears on the paragraph a code span may pair over.
type lineKind uint8

const (
	// kText continues the paragraph above it, or starts one.
	kText lineKind = iota
	// kBlank ends a paragraph.
	kBlank
	// kStart starts a block that interrupts a paragraph; its text, if any,
	// starts the next one.
	kStart
	// kSingle is a one-line block: an ATX heading, a thematic break or a
	// setext underline.
	kSingle
	// kComment opens an HTML comment block: the line ends the paragraph above
	// it, holds no code span, and the paragraph below it starts fresh.
	kComment
	// kRow is a table row: a one-line block, its cells read apart.
	kRow
	// kMaybe may start a block or continue the paragraph above.
	kMaybe
	// kHTML opens an HTML block, or may: like kMaybe, and its lines hold no
	// code span.
	kHTML
)

var (
	atxRe       = regexp.MustCompile(`^#{1,6}(?:[ \t]|$)`)
	setextRe    = regexp.MustCompile(`^(?:=+|-+)[ \t]*$`)
	thematicRe  = regexp.MustCompile(`^(?:(?:\*[ \t]*){3,}|(?:-[ \t]*){3,}|(?:_[ \t]*){3,})$`)
	bulletRe1   = regexp.MustCompile(`^[-*+][ \t]+\S`)
	orderedOne  = regexp.MustCompile(`^0*1[.)][ \t]+\S`)
	orderedAny  = regexp.MustCompile(`^[0-9]{1,9}[.)](?:[ \t]|$)`)
	emptyItemRe = regexp.MustCompile(`^[*+][ \t]*$`)
	// blockStartRe is any form that starts a block, read at any indent.
	blockStartRe = regexp.MustCompile("^(?:#{1,6}(?:[ \\t]|$)|>|`{3,}|~{3,}|[-*+](?:[ \\t]|$)|[0-9]{1,9}[.)](?:[ \\t]|$)|<[A-Za-z/!?]|(?:[-*_=][ \\t]*)+$)")
	// delimiterRowRe is a table's delimiter row; it holds a `|` as well.
	delimiterRowRe = regexp.MustCompile(`^[ \t]*\|?[ \t]*:?-+:?[ \t]*(?:\|[ \t]*:?-+:?[ \t]*)*\|?[ \t]*$`)
	// htmlBlockRe is a line that opens an HTML block other than a comment
	// (CommonMark 4.6), at up to three columns or after list markers: a raw
	// text element's tag (group 1, kind 1), a processing instruction (group 2,
	// kind 3), a declaration (kind 4), a CDATA section (group 3, kind 5), a
	// block element's tag (kind 6), or any complete open or closing tag alone
	// on the line (kind 7).
	htmlBlockRe = regexp.MustCompile(`(?i)^(?: {0,3}|[ \t]*(?:(?:[-*+]|[0-9]{1,9}[.)])[ \t]+)+)(?:<(pre|script|style|textarea)(?:[ \t>]|$)|(<\?)|<![A-Za-z]|(<!\[CDATA\[)|</?` + htmlBlockTags + `(?:[ \t]|/?>|$)|</?[A-Za-z][A-Za-z0-9-]*(?:[ \t]+[A-Za-z_:][A-Za-z0-9_.:-]*(?:[ \t]*=[ \t]*(?:[^ \t"'=<>` + "`" + `]+|'[^']*'|"[^"]*"))?)*[ \t]*/?>[ \t]*$)`)
)

// htmlBlockTags are the element names that open an HTML block of kind 6.
const htmlBlockTags = `(?:address|article|aside|base|basefont|blockquote|body|caption|center|col|colgroup|dd|details|dialog|dir|div|dl|dt|fieldset|figcaption|figure|footer|form|frame|frameset|h[1-6]|head|header|hr|html|iframe|legend|li|link|main|menu|menuitem|nav|noframes|ol|optgroup|option|p|param|search|section|summary|table|tbody|td|tfoot|th|thead|title|tr|track|ul)`

// kindOf is a line's kind read from its own form.
func kindOf(ln string) lineKind {
	if strings.TrimSpace(ln) == "" {
		return kBlank
	}
	w := indentWidth(ln)
	t := strings.TrimLeft(ln, " \t")
	if w >= 4 {
		if blockStartRe.MatchString(t) {
			return kMaybe
		}
		return kText
	}
	if k := strings.Index(ln, "<!--"); k >= 0 && markersOnlyRe.MatchString(ln[:k]) {
		return kComment
	}
	switch {
	case atxRe.MatchString(t), setextRe.MatchString(t), thematicRe.MatchString(t):
		return kSingle
	case fenceRe.MatchString(ln):
		if _, _, ok := opensFence(ln, TopLevel); ok {
			return kStart
		}
		return kText
	case htmlBlockRe.MatchString(ln):
		return kHTML
	case strings.HasPrefix(t, ">"):
		return kMaybe
	case bulletRe1.MatchString(t), orderedOne.MatchString(t):
		return kStart
	case orderedAny.MatchString(t), emptyItemRe.MatchString(t):
		return kMaybe
	}
	return kText
}

// classify reads every line's kind, a table's rows among them (a line holding
// a `|` with a delimiter row below it, the delimiter row, and each line after
// to a blank line or another block), and which lines an HTML block holds.
func classify(lines []string) (kinds []lineKind, html []bool) {
	kinds = make([]lineKind, len(lines))
	html = make([]bool, len(lines))
	table := false
	for i, raw := range lines {
		ln := strings.TrimRight(raw, "\r")
		k := kindOf(ln)
		switch {
		case table && k == kText:
			k = kRow
		case (k == kText || k == kMaybe) && strings.Contains(ln, "|") && i+1 < len(lines):
			if next := strings.TrimRight(lines[i+1], "\r"); strings.Contains(next, "|") && delimiterRowRe.MatchString(next) {
				k, table = kRow, true
			}
		}
		if k != kRow {
			table = false
		}
		kinds[i] = k
	}
	for i := 0; i < len(lines); i++ {
		if kinds[i] != kHTML {
			continue
		}
		ln := strings.TrimRight(lines[i], "\r")
		closer := ""
		if m := htmlBlockRe.FindStringSubmatch(ln); m != nil {
			switch {
			case m[1] != "":
				closer = "</" + strings.ToLower(m[1]) + ">"
			case m[2] != "":
				closer = "?>"
			case m[3] != "":
				closer = "]]>"
			}
		}
		j := i
		for ; j < len(lines); j++ {
			step()
			l := strings.TrimRight(lines[j], "\r")
			if closer == "" && strings.TrimSpace(l) == "" {
				break
			}
			html[j] = true
			if closer != "" && j > i && strings.Contains(strings.ToLower(l), closer) {
				break
			}
		}
		i = j
	}
	return kinds, html
}

// sure reports whether a kind ends every reading of the paragraph above it.
func (k lineKind) sure() bool {
	return k == kBlank || k == kStart || k == kSingle || k == kComment
}

// single reports whether a kind holds its own spans, so the next line starts
// a paragraph.
func (k lineKind) single() bool { return k == kSingle || k == kComment || k == kRow }

// maybe reports whether a kind may continue the paragraph above it or start a
// block.
func (k lineKind) maybe() bool { return k == kMaybe || k == kRow || k == kHTML }

// lineInline is what Read reads inline on one line: the byte ranges inside a
// code span and inside an HTML comment (the last of which may run to the
// line's end), and the offset of each comment opener on it.
type lineInline struct {
	spans    [][2]int
	comments [][2]int
	openers  []int
}

// atom is a stretch of one paragraph Read pairs as a unit: lines [start,
// end), the first read from byte col, and per line what was read there.
type atom struct {
	start, end int
	lines      []lineInline
	// open is, per line, whether the line ends inside a comment.
	open []bool
	// rawHTML: the stretch is the rest of an HTML block's line, so it pairs no
	// span.
	rawHTML bool
	// paired: every run opened was closed and no comment is left open, so the
	// stretch pairs alike joined to the next one or not.
	paired bool
	// tagged: a `<` that may open a tag or an autolink holds a backtick
	// before its `>`.
	tagged bool
	// pipeSpan: a span on a table row covers an unescaped `|`.
	pipeSpan bool
}

// walkAtom pairs the stretch of the paragraph that starts on line i at byte
// col: that line alone when it is a single-line kind or rawHTML, else it and
// every kText line below it. One pass over the stretch: each byte is visited
// once and each run is paired by one CodeSpanPairer, so the cost is in line
// with the stretch.
func walkAtom(lines []string, kinds []lineKind, i, col int, rawHTML bool) *atom {
	end := i + 1
	if !rawHTML && !kinds[i].single() {
		for end < len(lines) && kinds[end] == kText {
			end++
		}
	}
	a := &atom{start: i, end: end, lines: make([]lineInline, end-i), open: make([]bool, end-i), rawHTML: rawHTML, paired: true}
	offs := make([]int, end-i)
	var s string
	if end == i+1 {
		s = strings.TrimRight(lines[i], "\r")[col:]
	} else {
		var b strings.Builder
		for j := i; j < end; j++ {
			ln := strings.TrimRight(lines[j], "\r")
			if j == i {
				ln = ln[col:]
			} else {
				b.WriteByte('\n')
			}
			offs[j-i] = b.Len()
			b.WriteString(ln)
		}
		s = b.String()
	}
	stepN(end - i)
	if !strings.ContainsAny(s, "`<") {
		return a
	}
	// lineOf is the stretch line holding byte p, found by a cursor that only
	// moves forward, since the walk hands it rising offsets.
	li := 0
	lineOf := func(p int) int {
		for li+1 < len(offs) && offs[li+1] <= p {
			li++
		}
		return li
	}
	colOf := func(k, p int) int {
		if k == 0 {
			return p + col
		}
		return p - offs[k]
	}
	lineEnd := func(k int) int {
		if k+1 < len(offs) {
			return offs[k+1] - 1
		}
		return len(s)
	}
	// split hands each line's part of [from, to) to add, and, for a
	// comment, marks open every line the range runs past the end of.
	split := func(from, to int, add func(k, x, y int), comment bool) {
		for k := lineOf(from); k < len(offs); k++ {
			stepN(1)
			lo, hi := max(from, offs[k]), min(to, lineEnd(k))
			if hi > lo {
				add(k, colOf(k, lo), colOf(k, hi))
			}
			if to <= lineEnd(k) {
				return
			}
			if comment {
				a.open[k] = true
			}
		}
	}
	addSpan := func(k, x, y int) { a.lines[k].spans = append(a.lines[k].spans, [2]int{x, y}) }
	addComment := func(k, x, y int) { a.lines[k].comments = append(a.lines[k].comments, [2]int{x, y}) }

	pr := termsafe.NewCodeSpanPairer(s)
	defer func() { stepN(pr.Steps()) }()
	row := kinds[i] == kRow
	gt, bt := 0, 0 // cursors: the next '>' and the next backtick at or after a '<'
	for p := 0; p < len(s); {
		switch c := s[p]; {
		case c == '`':
			stepN(1)
			q := runEnd(s, p)
			if rawHTML {
				p = q
				continue
			}
			// Outside a span a backtick behind an odd run of backslashes is
			// escaped, a literal character; the rest of its run may open.
			open, n := p, 0
			for k := p; k > 0 && s[k-1] == '\\'; k-- {
				stepN(1)
				n++
			}
			if n%2 == 1 {
				open++
			}
			if open < q {
				if sp, ok := pr.Pair(open, q); ok {
					if row && strings.Contains(strings.ReplaceAll(s[sp.Start:sp.End], `\|`, ""), "|") {
						a.pipeSpan = true
					}
					split(open, sp.End, addSpan, false)
					p = sp.End
					continue
				}
				a.paired = false
			}
			p = q
		case c == '<' && strings.HasPrefix(s[p:], "<!--"):
			stepN(1)
			k := lineOf(p)
			a.lines[k].openers = append(a.lines[k].openers, colOf(k, p))
			e := strings.Index(s[p+len("<!"):], "-->")
			if e < 0 {
				split(p, len(s), addComment, true)
				a.open[len(offs)-1] = true
				if !rawHTML {
					a.paired = false
				}
				p = len(s)
				continue
			}
			to := p + len("<!") + e + len("-->")
			split(p, to, addComment, true)
			p = to
		case c == '<' && !rawHTML && p+1 < len(s) && (isASCIILetter(s[p+1]) || s[p+1] == '/'):
			stepN(1)
			gt, bt = max(gt, p), max(bt, p)
			for gt < len(s) && s[gt] != '>' {
				gt++
			}
			for bt < len(s) && s[bt] != '`' {
				bt++
			}
			if gt < len(s) && bt < gt {
				a.tagged = true
			}
			p++
		default:
			p++
		}
	}
	return a
}

func isASCIILetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

// stepN reports n steps to stepHook.
func stepN(n int) {
	if stepHook != nil {
		for range n {
			stepHook()
		}
	}
}
