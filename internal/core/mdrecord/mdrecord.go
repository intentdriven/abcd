// Package mdrecord is the Markdown machinery a RECORD BODY is read and written
// through: which lines are live markdown and which are not, where a section
// starts and stops, what a top-level bullet is, and where a trailing run of
// link-reference definitions ends.
//
// It is a leaf for the reason core/grounds and core/issueschema are. Two record
// families carry the same constructs — the intent record's scope conditions and
// audit notes, the issue ledger's grounds — and a body reader spelled twice is a
// reader the two can disagree about, which is how a bullet one writer appends is
// a bullet the other cannot find. It imports the standard library and
// termsafe, itself a standard-library leaf, for the tree's one code-span pairer:
// no filesystem, no transport, no record store, and no notion of which family a
// body belongs to.
//
// What it does NOT own is any heading's meaning. A caller supplies the heading
// pattern it is looking for; this package only says whether a line matching it
// is live, and which lines lie under it.
package mdrecord

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/termsafe"
)

var (
	// fenceRe matches a fenced-code-block delimiter: a run of three or more
	// backticks or tildes at up to three spaces of indent (CommonMark).
	fenceRe = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})(.*)$")
	// headingRe matches any markdown ATX heading line.
	headingRe = regexp.MustCompile(`^#{1,6}\s`)
	// bulletRe matches a TOP-LEVEL markdown list item. A record's bullets are
	// counted positionally, so only column-0 bullets count — an indented
	// sub-bullet is detail of its parent, not a separate item.
	bulletRe = regexp.MustCompile(`^[-*]\s+\S`)
	// anyBulletRe matches a list item at ANY indent. A line that is a bullet ends
	// the preceding bullet, so an indented sub-bullet is never folded into its
	// parent's text.
	anyBulletRe = regexp.MustCompile(`^\s*[-*]\s+\S`)
	// bulletPrefixRe strips a top-level list marker to leave the bullet's prose.
	bulletPrefixRe = regexp.MustCompile(`^[-*]\s+`)
	// linkRefDefRe matches a markdown link-reference definition (`[label]: dest`),
	// up to three leading spaces per CommonMark. A record can park such a
	// definition at the tail of a section (itd-114's `[iss-80]:` ref); a new block
	// must be inserted ABOVE the trailing run of them rather than appended below
	// it (iss-2608210737265820).
	linkRefDefRe = regexp.MustCompile(`^ {0,3}\[[^\]]+\]:\s+\S`)
)

// Mask flags: why a line is not live markdown. Both are read the same way — the
// line is neither matched nor written — but they are reported separately, so a
// refusal names the thing the reader has to go and look at.
const (
	MaskFence uint8 = 1 << iota
	MaskComment
)

// Mask reports, per line, whether that line lies inside a fenced code block or
// an HTML comment span — the delimiter lines included, since nothing on them is
// live markdown either. A parser that cannot see either reads an EXAMPLE, or a
// deliberately PARKED bullet, as an instruction: a fenced or commented
// `## Scope Conditions` shadows the record's real one, and a bullet inside
// either is counted as a condition and written into by the stamp. Writing a
// marker inside a comment is the worst of the three — the marker's own `-->`
// closes the comment early, so the rest of the parked block starts rendering
// (iss-2608300235388164, iss-2608300259316871).
//
// Neither construct nests in the other: inside a fence `<!--` is literal text,
// and inside a comment a fence delimiter is. An unclosed opener of either kind
// runs to end of file — CommonMark's rule for a fence, and the only safe reading
// of a comment nobody closed.
//
// Mask is Read under the TopLevel rule.
func Mask(lines []string) []uint8 { return Read(lines, TopLevel).Mask }

// Unclosed reports the line that opened a span nothing closes before the end of
// the input, and which construct it is (MaskFence or MaskComment).
//
// An unclosed opener is not itself a fault — running it to end of file is
// CommonMark's rule for a fence and the only safe reading of a comment nobody
// closed, and Mask does exactly that. What it IS, is the reason every line below
// it stops being live markdown, so a writer whose appended line lands there can
// never read it back. That writer has to be able to NAME the line, or its
// refusal describes the symptom and sends the operator to rewrite whatever it
// was appending — which is the one part of the record that is innocent
// (iss-2608301803423101).
//
// Unclosed is Read under the TopLevel rule.
func Unclosed(lines []string) (line int, flag uint8, ok bool) {
	return Read(lines, TopLevel).Unclosed()
}

// Rule is which fences a reading recognises. This package is the tree's one
// notion of a fence and an HTML comment; the rule is the one place a reader may
// ask for more than CommonMark's top level, and it may ask for exactly one thing
// more.
type Rule uint8

const (
	// TopLevel is CommonMark's fence rule at the top level of a document: a run
	// of three or more backticks or tildes at up to three spaces of indent,
	// closed by a run of the same character at least as long, at up to three
	// spaces of indent, with nothing after it. A backtick opener's info string
	// may not itself hold a backtick.
	TopLevel Rule = iota
	// ListNested is TopLevel plus a fence indented under a list item. The site
	// renders fences written inside list items, where the fence sits at the
	// item's content indent and that may be four columns or more; a reader that
	// saw only the left margin would read the fence's own `#` lines as headings.
	//
	// A run at four or more columns cannot open a fence at the top level (it is
	// indented code there), so under ListNested it opens a fence in the list
	// item holding it. That fence closes on a run of the opener's character at
	// least as long, with nothing after it, indented no more than three columns
	// past the opener (further in is content: CommonMark's closer allowance,
	// measured from the item). And because fenced code never continues lazily,
	// a non-blank line at column 0 ends the item and the fence with it
	// (CommonMark 5.2). A run at up to three columns is read exactly as TopLevel
	// reads it, so the two rules differ only where TopLevel sees no fence at all.
	ListNested

	// listItem is the reading neither exported rule takes: a run at one to
	// three columns opened inside a list item, which ends with the item at a
	// column-0 line. Whether a run at that indent is nested or top-level turns on
	// the list around it, which this package does not parse, so the exported
	// rules take the top-level reading and FencedUnderEveryRule consults this
	// one as well.
	listItem
)

// Span is one fenced block as the half-open line range [Start, End), its
// delimiter lines included. Closed is false when the fence runs to the end of
// the input.
type Span struct {
	Start, End int
	Closed     bool
}

// Reading is one pass over a body under one Rule: the per-line mask, every
// fenced block as its own span (a per-line mask cannot say where one fence ends
// and the next, written directly below it, begins), and the span still open at
// the end, if any.
type Reading struct {
	Mask   []uint8
	Fences []Span

	// comments is every HTML comment left open at the end of the line it opens
	// on, with the byte offset of its opener there.
	comments []commentSpan

	openLine int
	openFlag uint8
}

// commentSpan is one HTML comment over more than one line: its lines as a
// Span, and the byte offset its `<!--` sits at on line Start.
type commentSpan struct {
	Span
	col int
}

// Unclosed reports the line that opened the span still open at the end of the
// input, and which construct it is — see the package-level Unclosed.
func (r Reading) Unclosed() (line int, flag uint8, ok bool) {
	return r.openLine, r.openFlag, r.openLine >= 0
}

// FencedUnderEveryRule reports, per line, whether EVERY reading reads that line
// as inside a fence (its delimiters included): TopLevel, ListNested, and the
// list-item reading of a run indented one to three columns. It is the reading for a reader that
// must never hide live text behind a disagreement between the rules — a gate
// whose missed finding is silent, a floor whose missed heading travels — so a
// line either rule reads as live is live to it. HTML comments are not consulted:
// what a comment hides is the caller's question, and the readers that ask this
// one read commented text.
func FencedUnderEveryRule(lines []string) []bool {
	out := make([]bool, len(lines))
	for i := range out {
		out[i] = true
	}
	for _, rule := range []Rule{TopLevel, ListNested, listItem} {
		for i, m := range Read(lines, rule).Mask {
			out[i] = out[i] && m&MaskFence != 0
		}
	}
	return out
}

// nestedFenceRe matches a fence delimiter at any indent, for the nested readings.
var nestedFenceRe = regexp.MustCompile("^([ \t]*)(`{3,}|~{3,})(.*)$")

// indentWidth is the column a line's first non-blank character sits at, a tab
// advancing to the next multiple of four (CommonMark's tab stop).
func indentWidth(ln string) int {
	w := 0
	for _, c := range ln {
		switch c {
		case ' ':
			w++
		case '\t':
			w += 4 - w%4
		default:
			return w
		}
	}
	return w
}

// fenceOpener reports the run and the indent of a fence delimiter under a rule,
// and whether the line is one at all.
func fenceOpener(ln string, rule Rule) (run, rest string, indent int, ok bool) {
	if rule != TopLevel {
		m := nestedFenceRe.FindStringSubmatch(ln)
		if m == nil {
			return "", "", 0, false
		}
		return m[2], m[3], indentWidth(m[1]), true
	}
	m := fenceRe.FindStringSubmatch(ln)
	if m == nil {
		return "", "", 0, false
	}
	return m[1], m[2], indentWidth(ln), true
}

// opensFence is the tree's one fence-opener predicate: a delimiter under the
// rule, except a backtick run whose info string holds a backtick, which is a
// code span and opens nothing (CommonMark). Read opens every fence through it.
func opensFence(ln string, rule Rule) (run string, indent int, ok bool) {
	run, rest, indent, ok := fenceOpener(ln, rule)
	if !ok || (run[0] == '`' && strings.Contains(rest, "`")) {
		return "", 0, false
	}
	return run, indent, true
}

// OpensFence reports whether a line, read outside any open fence or comment,
// opens a fenced code block under a rule. It is the opener Read applies,
// exported for a reader that judges one line rather than walking a body — the
// site renderer asking whether the block the walk handed it opens with a fence
// — so that reader takes the walk's rule instead of a prefix test of its own,
// which read a balanced span such as three backticks, x, three backticks as a
// fence (iss-2609262309556167).
func OpensFence(ln string, rule Rule) bool {
	_, _, ok := opensFence(strings.TrimRight(ln, "\r"), rule)
	return ok
}

// inItem reports whether a rule reads a fence opened at this indent as held by
// a list item.
func inItem(rule Rule, indent int) bool {
	switch rule {
	case ListNested:
		return indent >= 4
	case listItem:
		return indent > 0
	}
	return false
}

// Read is the single pass every reading of fences and comments goes through.
// Spelled twice, two readers could disagree about which lines a span covers and
// which line opened it.
func Read(lines []string, rule Rule) Reading {
	r := Reading{Mask: make([]uint8, len(lines)), openLine: -1}
	fenceOpen, fenceIndent, fenceStart := "", 0, 0
	inComment := false
	closeFence := func(end int) {
		r.Fences = append(r.Fences, Span{Start: fenceStart, End: end, Closed: true})
		fenceOpen = ""
		r.openLine, r.openFlag = -1, 0
	}
	openComment := func(i int, ln string, from int) {
		r.comments = append(r.comments, commentSpan{Span: Span{Start: i, End: len(lines)}, col: openCommentCol(ln, from)})
	}
	closeComment := func(i int) {
		c := &r.comments[len(r.comments)-1]
		c.End, c.Closed = i+1, true
	}
	for i, raw := range lines {
		ln := strings.TrimRight(raw, "\r")
		// A list item ends at a non-blank line at column 0, and a fence it held
		// ends with it. The line itself is then read as live.
		if fenceOpen != "" && inItem(rule, fenceIndent) && strings.TrimSpace(ln) != "" && indentWidth(ln) == 0 {
			closeFence(i)
		}
		switch {
		case inComment:
			r.Mask[i] |= MaskComment
			// Inside a comment every byte is literal, so the raw first `-->` is the
			// closer; the remainder of the line is live markdown again and may open
			// a fresh span.
			if k := strings.Index(ln, "-->"); k >= 0 {
				closeComment(i)
				if inComment = opensCommentFrom(ln, k+len("-->")); inComment {
					r.openLine, r.openFlag = i, MaskComment
					openComment(i, ln, k+len("-->"))
				} else {
					r.openLine, r.openFlag = -1, 0
				}
			}
		case fenceOpen != "":
			r.Mask[i] |= MaskFence
			run, rest, indent, ok := fenceOpener(ln, rule)
			reach := 3
			if inItem(rule, fenceIndent) {
				reach = fenceIndent + 3
			}
			if ok && run[0] == fenceOpen[0] && len(run) >= len(fenceOpen) && strings.TrimSpace(rest) == "" && indent <= reach {
				closeFence(i + 1)
			}
		default:
			if run, indent, ok := opensFence(ln, rule); ok {
				fenceOpen, fenceIndent, fenceStart = run, indent, i
				r.Mask[i] |= MaskFence
				r.openLine, r.openFlag = i, MaskFence
				continue
			}
			if OpensComment(ln) {
				inComment = true
				r.Mask[i] |= MaskComment
				r.openLine, r.openFlag = i, MaskComment
				openComment(i, ln, 0)
			}
		}
	}
	if fenceOpen != "" {
		r.Fences = append(r.Fences, Span{Start: fenceStart, End: len(lines)})
	}
	return r
}

// OpensComment reports whether a line leaves an HTML comment open. It walks the
// line left to right and lets the FIRST construct win, which is CommonMark's own
// precedence: at a backtick run it skips a matched code span whole (or treats an
// unmatched run as literal backticks), and at `<!--` it skips to the `-->` that
// closes it — consuming any backticks in between, because inside a comment they
// are literal text.
//
// Resolving code spans first, as this did, diverged in both directions: a
// backtick inside a LIVE comment re-paired the rest of the line, and a comment
// opener quoted in backticks read as live. One cursor closes both
// (iss-2608300320418618, iss-2608300335369473). A well-formed identity marker
// closes on its own line, so it never opens a span; a genuinely unclosed opener
// masks to end of file.
func OpensComment(ln string) bool { return opensCommentFrom(ln, 0) }

// opensCommentFrom is OpensComment beginning at an offset — the form Mask needs
// when a line closes the span it was already inside and the remainder is live
// markdown again.
func opensCommentFrom(ln string, start int) bool { return commentWalk(ln, start, nil) }

// openCommentCol is the byte offset of the `<!--` a line leaves open, the walk
// starting at start, or -1 when it leaves none open.
func openCommentCol(ln string, start int) int {
	col := -1
	if !commentWalk(ln, start, func(from, _ int) { col = from }) {
		return -1
	}
	return col
}

// commentWalk is the one left-to-right walk OpensComment and ReadPage share:
// from start, a matched code span is skipped whole and an unmatched backtick
// run is literal, and at `<!--` the walk skips to the `-->` closing it. It
// reports whether a comment is still open at the end of the line, and hands
// each comment span it skips, the open one included, to blank when blank is
// not nil.
func commentWalk(ln string, start int, blank func(from, to int)) bool {
	for i := start; i < len(ln); {
		switch {
		case ln[i] == '`':
			if sp, ok := termsafe.PairCodeSpan(ln, i); ok {
				i = sp.End
			} else {
				i = runEnd(ln, i)
			}
		case strings.HasPrefix(ln[i:], "<!--"):
			// The closer may share the opener's dashes: `<!-->` and `<!--->`
			// are whole, empty comments (CommonMark 0.31, and HTML's own
			// reading), so the search starts two bytes in.
			rest := ln[i+len("<!"):]
			k := strings.Index(rest, "-->")
			if k < 0 {
				if blank != nil {
					blank(i, len(ln))
				}
				return true
			}
			end := i + len("<!") + k + len("-->")
			if blank != nil {
				blank(i, end)
			}
			i = end
		default:
			i++
		}
	}
	return false
}

// FirstContent is the tree's one leading-comment locator: the first line
// holding anything but blanks and HTML comments, and the byte column in that
// line where the content starts. A byte-order mark on line 0 is not content.
// Comments are read as Read reads them: the first `-->` closes one, a line may
// hold several, and content after a closer is content. It returns
// (len(lines), 0) when there is nothing else — a document of blanks and
// comments, or one whose comment never closes.
//
// Four readers once skipped a leading attribution comment each by its own
// walk on HasPrefix and Contains, and each read a closer followed by text, or
// two comments on one line, its own way (iss-2609251517210637).
func FirstContent(lines []string) (line, col int) {
	inComment := false
	for i, raw := range lines {
		ln := strings.TrimRight(raw, "\r")
		pos := 0
		if i == 0 && strings.HasPrefix(ln, "\ufeff") {
			pos = len("\ufeff")
		}
		if inComment {
			k := strings.Index(ln[pos:], "-->")
			if k < 0 {
				continue
			}
			pos += k + len("-->")
			inComment = false
		}
		for {
			for pos < len(ln) && (ln[pos] == ' ' || ln[pos] == '\t') {
				pos++
			}
			if pos == len(ln) || !strings.HasPrefix(ln[pos:], "<!--") {
				break
			}
			k := strings.Index(ln[pos+len("<!--"):], "-->")
			if k < 0 {
				inComment = true
				pos = len(ln)
				break
			}
			pos += len("<!--") + k + len("-->")
		}
		if pos < len(ln) {
			return i, pos
		}
	}
	return len(lines), 0
}

// masked reports whether a line is not live markdown, for any reason. It is the
// package's own predicate: every consumer asks a question about a SECTION, a
// BULLET or a RANGE (AnyMasked), and none has ever needed to ask about one line.
func masked(mask []uint8, i int) bool { return i < len(mask) && mask[i] != 0 }

// AnyMasked reports whether any line of [start, end) carries the given flag.
func AnyMasked(mask []uint8, start, end int, flag uint8) bool {
	for i := start; i < end && i < len(mask); i++ {
		if mask[i]&flag != 0 {
			return true
		}
	}
	return false
}

// IsHeading reports whether a line is a markdown ATX heading. It judges the line
// alone: whether that heading is LIVE is Mask's answer, not this one's.
func IsHeading(ln string) bool { return headingRe.MatchString(ln) }

// IsTopLevelBullet reports whether a line opens a column-0 list item.
func IsTopLevelBullet(ln string) bool { return bulletRe.MatchString(ln) }

// TrimBulletPrefix strips a top-level list marker, leaving the bullet's prose.
func TrimBulletPrefix(ln string) string { return bulletPrefixRe.ReplaceAllString(ln, "") }

// CountHeadings counts the live headings matching headRe.
func CountHeadings(lines []string, mask []uint8, headRe *regexp.Regexp) int {
	n := 0
	for i, ln := range lines {
		if !masked(mask, i) && headRe.MatchString(strings.TrimRight(ln, "\r")) {
			n++
		}
	}
	return n
}

// SectionLineRange returns the [start, end) line bounds of the body of the
// first live section whose heading matches headRe, and whether such a heading
// exists, which is what separates an absent section from an empty one.
//
// This is the single notion of where a section starts and stops, so every reader
// and every writer of a record body agrees about which lines belong to which
// section.
func SectionLineRange(lines []string, headRe *regexp.Regexp) (start, end int, ok bool) {
	return SectionLineRangeIn(lines, Mask(lines), headRe)
}

// SectionLineRangeIn is SectionLineRange over a mask the caller already
// computed. A masked line — fenced or commented — is neither the heading that
// opens a section nor the heading that closes one.
func SectionLineRangeIn(lines []string, mask []uint8, headRe *regexp.Regexp) (start, end int, ok bool) {
	for i, ln := range lines {
		if masked(mask, i) || !headRe.MatchString(strings.TrimRight(ln, "\r")) {
			continue
		}
		end = len(lines)
		for j := i + 1; j < len(lines); j++ {
			if !masked(mask, j) && headingRe.MatchString(strings.TrimRight(lines[j], "\r")) {
				end = j
				break
			}
		}
		return i + 1, end, true
	}
	return 0, 0, false
}

// SectionLineRangeNestedIn is SectionLineRangeIn for a section that holds
// sub-sections of its own: its body runs to the next live heading of the same
// depth or shallower, so a `### Relay` under `## Open Questions` is inside it
// and a `## Decisions` after it is not.
func SectionLineRangeNestedIn(lines []string, mask []uint8, headRe *regexp.Regexp) (start, end int, ok bool) {
	for i, ln := range lines {
		ln = strings.TrimRight(ln, "\r")
		if masked(mask, i) || !headRe.MatchString(ln) {
			continue
		}
		depth := headingDepth(ln)
		end = len(lines)
		for j := i + 1; j < len(lines); j++ {
			next := strings.TrimRight(lines[j], "\r")
			if !masked(mask, j) && headingRe.MatchString(next) && headingDepth(next) <= depth {
				end = j
				break
			}
		}
		return i + 1, end, true
	}
	return 0, 0, false
}

// headingDepth is the number of `#` opening an ATX heading line, indented up
// to three columns, or 7 for a line that is not one, which no heading's depth
// reaches.
func headingDepth(ln string) int {
	t := strings.TrimLeft(ln, " ")
	if len(ln)-len(t) > 3 || !headingRe.MatchString(t) {
		return 7
	}
	return len(t) - len(strings.TrimLeft(t, "#"))
}

// Page is a body as a reader of what the page SAYS reads it (ReadPage).
type Page struct {
	// Text is each line as a reader of the rendered page sees it: a line
	// inside a fenced block, its delimiters included, is "", and every HTML
	// comment span on a line (one that opens and closes there, the tail of one
	// opened above, the head of one that closes below) is replaced by spaces of
	// the same length, so the text before a comment's opener and after its
	// closer is read where it sits, at its own offsets. From the line Unsure
	// names on, every line is its own text, nothing blanked.
	Text []string
	// Mask is the per-line mask of the same reading, for a section's bounds:
	// a line is masked only where Text does not read it.
	Mask []uint8

	unsure   int
	openLine int
	openFlag uint8
}

// Unsure reports the first line from which the page's fences and comments
// can be read more than one way, and whether there is one: see ReadPage.
func (p Page) Unsure() (line int, ok bool) { return p.unsure, p.unsure >= 0 }

// Unclosed reports the line that opened a span nothing closes, and which
// construct it is, under the page's reading. An opener on or below the line
// Unsure names is not reported, since every line from there is read.
func (p Page) Unclosed() (line int, flag uint8, ok bool) {
	return p.openLine, p.openFlag, p.openLine >= 0
}

// ReadPage is the one reading for a reader of what a page says that must not
// read less than the page shows: a gate that refuses on what the text states.
// Its fences are ListNested's and its comments are Read's: the first raw `-->`
// closes a comment, a code span outside one is skipped whole (a `<!--` quoted
// in backticks opens nothing), and inside a fence a comment opener is literal.
// Mask leaves a comment that opens and closes on one line live, because a
// writer may touch that line; Text blanks it.
//
// Every line is either returned as text or lies in a fenced block or an HTML
// comment that every reading here agrees on. This package does not parse
// lists, blockquotes or HTML blocks, so where the extent of a fence or a
// comment can turn on one of them the page is Unsure from the first such line,
// and that line and every line below it are returned as their own text. That
// first line is the first where:
//
//   - TopLevel, ListNested and the list-item reading mask the line
//     differently, as each reads a fence indented four columns or more;
//   - a non-blank line inside a fence or a comment whose opener is indented
//     and sits in a list item is indented less than the opener, or less than
//     the item's content column;
//   - an HTML comment that runs past its line opens at four columns or more,
//     or after other text on that line and is not closed inside the same
//     paragraph (inParagraph; the opener's line is named);
//   - a list marker and a fence delimiter share a line;
//   - a fence delimiter sits in an HTML block (a line opening with a tag, to
//     the next blank line, or for pre, script, style and textarea to the
//     element's closing tag), or a comment opener sits in one of those four.
func ReadPage(lines []string) Page {
	rs := [...]Reading{Read(lines, TopLevel), Read(lines, ListNested), Read(lines, listItem)}
	nl := rs[1]
	u := len(lines)
	at := func(i int) {
		if i < u {
			u = i
		}
	}
	for i := range lines {
		if rs[0].Mask[i] != nl.Mask[i] || rs[2].Mask[i] != nl.Mask[i] {
			at(i)
			break
		}
	}
	for _, r := range rs {
		for _, s := range r.Fences {
			leftOfSpan(lines, r.Mask, s, indentWidth(strings.TrimRight(lines[s.Start], "\r")), at)
		}
		for _, c := range r.comments {
			lead := strings.TrimRight(lines[c.Start], "\r")[:c.col]
			switch {
			case strings.TrimSpace(lead) == "" && indentWidth(lead) < 4:
				leftOfSpan(lines, r.Mask, c.Span, indentWidth(lead), at)
			case !inParagraph(lines, c):
				at(c.Start)
			}
		}
	}
	for i, raw := range lines {
		if nl.Mask[i] == 0 && markerFence(strings.TrimRight(raw, "\r")) {
			at(i)
			break
		}
	}
	htmlBlocks(lines, nl.Mask, at)

	p := Page{Text: liveText(lines, nl.Mask), Mask: nl.Mask, unsure: -1, openLine: nl.openLine, openFlag: nl.openFlag}
	if u < len(lines) {
		p.unsure = u
		p.Mask = append([]uint8(nil), nl.Mask...)
		for j := u; j < len(lines); j++ {
			p.Text[j], p.Mask[j] = strings.TrimRight(lines[j], "\r"), 0
		}
		if p.openLine >= u {
			p.openLine, p.openFlag = -1, 0
		}
	}
	return p
}

// leftOfSpan names, through at, the first non-blank line of s indented less
// than its lines must be to stay in the block, when its opener is indented
// `indent` columns and sits in a list item: the opener's indent, or the item's
// content column when that is nearer the margin. A list item ends at such a
// line, and the block with it.
func leftOfSpan(lines []string, mask []uint8, s Span, indent int, at func(int)) {
	if indent == 0 {
		return
	}
	keep, ok := itemContentCol(lines, mask, s.Start, indent)
	if !ok {
		return
	}
	keep = min(keep, indent)
	for j := s.Start + 1; j < s.End; j++ {
		if ln := strings.TrimRight(lines[j], "\r"); strings.TrimSpace(ln) != "" && indentWidth(ln) < keep {
			at(j)
			return
		}
	}
}

// paragraphBreakRe is a line that ends a paragraph or starts a block that
// interrupts one, or a table's: a heading, a blockquote, a fence delimiter, a
// list item, an HTML tag, a thematic break or setext underline, or a line
// holding a `|`.
var paragraphBreakRe = regexp.MustCompile("^ {0,3}(?:#{1,6}(?:[ \t]|$)|>|`{3,}|~{3,}|[-*+](?:[ \t]|$)|[0-9]{1,9}[.)](?:[ \t]|$)|<[A-Za-z/!?]|(?:[-*_=][ \t]*)+$)|\\|")

// inParagraph reports whether a comment opened after other text on its line
// closes inside the paragraph that line is in, where the page reads it as a
// comment: the opener's line is indented less than four columns and is not a
// heading, its text before the opener is more than list markers and `>`, and
// every line after it up to the closer's is neither blank nor a
// paragraphBreakRe line. A comment the page does not close there is text on
// the page.
func inParagraph(lines []string, c commentSpan) bool {
	ln := strings.TrimRight(lines[c.Start], "\r")
	lead := ln[:c.col]
	if !c.Closed || indentWidth(ln) >= 4 || headingRe.MatchString(strings.TrimLeft(ln, " ")) ||
		strings.Contains(ln, "|") || markersOnlyRe.MatchString(lead) {
		return false
	}
	for j := c.Start + 1; j < c.End; j++ {
		l := strings.TrimRight(lines[j], "\r")
		if strings.TrimSpace(l) == "" || paragraphBreakRe.MatchString(l) {
			return false
		}
	}
	return true
}

// markersOnlyRe is the text before a comment opener that is nothing but list
// markers, blockquote markers and blanks.
var markersOnlyRe = regexp.MustCompile(`^[ \t>]*(?:(?:[-*+]|[0-9]{1,9}[.)])[ \t]+[ \t>]*)*$`)

// itemContentCol is the content column of the list item the line at `at`,
// indented `indent` columns, sits in, and whether it sits in one: walking up,
// a list marker line indented less than `indent` comes before a blank line
// that has a line at column 0 below it. A line at column 0 with no blank line
// above it may continue an item's paragraph, so the walk goes on past it.
func itemContentCol(lines []string, mask []uint8, at, indent int) (int, bool) {
	col0 := false
	for j := at - 1; j >= 0; j-- {
		ln := strings.TrimRight(lines[j], "\r")
		if strings.TrimSpace(ln) == "" {
			if col0 {
				return 0, false
			}
			continue
		}
		if mask[j] != 0 {
			col0 = false
			continue
		}
		w := indentWidth(ln)
		if w < indent {
			if c, ok := listContentCol(ln); ok {
				return c, true
			}
		}
		col0 = w == 0
	}
	return 0, false
}

// listMarkerRe is a list item's marker line at any indent: a bullet or a
// number, the blanks after it, and the item's text.
var listMarkerRe = regexp.MustCompile(`^([ \t]*)([-*+]|[0-9]{1,9}[.)])([ \t]*)(.*)$`)

// listContentCol is the column a list item's content starts at, for its
// marker line, and whether the line is one (CommonMark 5.2): one past the
// marker for an empty item or one whose text is indented five columns or more
// past it, else where the text starts.
func listContentCol(ln string) (int, bool) {
	m := listMarkerRe.FindStringSubmatch(ln)
	if m == nil || (m[3] == "" && m[4] != "") {
		return 0, false
	}
	end := indentWidth(m[1]) + len(m[2])
	col := end
	for _, c := range m[3] {
		if c == '\t' {
			col += 4 - col%4
		} else {
			col++
		}
	}
	if m[4] == "" || col-end >= 5 {
		return end + 1, true
	}
	return col, true
}

// markerFenceRe is one or more list markers (a blockquote's `>` among them)
// and then a fence delimiter on the same line.
var markerFenceRe = regexp.MustCompile("^[ \t>]*(?:(?:[-*+]|[0-9]{1,9}[.)])[ \t]+[ \t>]*)+(`{3,}|~{3,})(.*)$")

// markerFence reports whether a line opens a list item with a fenced block,
// which neither reading here opens, so every delimiter below it may pair with
// another than the one the page pairs it with.
func markerFence(ln string) bool {
	m := markerFenceRe.FindStringSubmatch(ln)
	return m != nil && !(m[1][0] == '`' && strings.Contains(m[2], "`"))
}

// htmlStartRe is a line that opens with an HTML tag, at up to three columns.
var htmlStartRe = regexp.MustCompile(`^ {0,3}</?([A-Za-z][A-Za-z0-9-]*)(?:[ \t>/]|$)`)

// rawTextTags are the elements whose block runs to their closing tag and
// holds no markdown, a comment opener included.
var rawTextTags = map[string]bool{"pre": true, "script": true, "style": true, "textarea": true}

// htmlBlocks names, through at, the first fence delimiter in an HTML block,
// where the page reads it as HTML, and the first comment opener in a block of
// one of rawTextTags, where the page reads it as text. A block opens at a
// line htmlStartRe matches, outside a fence or comment, and runs to the line
// before the next blank line, or for rawTextTags to the line holding the
// element's closing tag.
func htmlBlocks(lines []string, mask []uint8, at func(int)) {
	for i := 0; i < len(lines); i++ {
		ln := strings.TrimRight(lines[i], "\r")
		m := htmlStartRe.FindStringSubmatch(ln)
		if mask[i] != 0 || m == nil {
			continue
		}
		tag := strings.ToLower(m[1])
		rawText := rawTextTags[tag] && !strings.HasPrefix(strings.TrimLeft(ln, " "), "</")
		j := i
		for ; j < len(lines); j++ {
			l := strings.TrimRight(lines[j], "\r")
			if !rawText && strings.TrimSpace(l) == "" {
				break
			}
			if j > i && mask[j]&MaskComment == 0 && nestedFenceRe.MatchString(l) {
				at(j)
				return
			}
			if rawText && strings.Contains(l, "<!--") {
				at(j)
				return
			}
			if rawText && strings.Contains(strings.ToLower(l), "</"+tag+">") {
				break
			}
		}
		i = j
	}
}

// liveText is Page.Text over a fence mask Read computed.
func liveText(lines []string, mask []uint8) []string {
	out := make([]string, len(lines))
	in := false
	for i, raw := range lines {
		if mask[i]&MaskFence != 0 {
			continue
		}
		out[i], in = blankComments(strings.TrimRight(raw, "\r"), in)
	}
	return out
}

// blankComments is one line of Page.Text: in is whether a comment is open as
// the line begins, and the second result whether one is open as it ends.
// Inside a comment the first raw `-->` closes it, as Read reads it; the rest
// of the line is commentWalk's.
func blankComments(ln string, in bool) (string, bool) {
	b := []byte(ln)
	blank := func(from, to int) {
		for j := from; j < to; j++ {
			b[j] = ' '
		}
	}
	start := 0
	if in {
		k := strings.Index(ln, "-->")
		if k < 0 {
			blank(0, len(b))
			return string(b), true
		}
		start = k + len("-->")
		blank(0, start)
	}
	open := commentWalk(ln, start, blank)
	return string(b), open
}

// BulletBlock is one top-level bullet and the continuation lines folded into
// it, as the half-open line range [Start, End).
type BulletBlock struct{ Start, End int }

// BulletBlocks splits a section body into its top-level bullets. It is the
// single notion of where an item starts and stops, so a reader and a writer can
// never disagree about which lines belong to which item. A masked line — fenced
// or commented — neither opens a bullet nor continues one.
func BulletBlocks(lines []string, mask []uint8, start, end int) []BulletBlock {
	var blocks []BulletBlock
	for i := start; i < end; i++ {
		if masked(mask, i) || !bulletRe.MatchString(strings.TrimRight(lines[i], "\r")) {
			continue
		}
		j := i + 1
		for ; j < end; j++ {
			cont := strings.TrimRight(lines[j], "\r")
			if masked(mask, j) || strings.TrimSpace(cont) == "" || anyBulletRe.MatchString(cont) {
				break
			}
		}
		blocks = append(blocks, BulletBlock{Start: i, End: j})
	}
	return blocks
}

// PeelTrailingLinkRefs removes a trailing run of markdown link-reference
// definitions (and any blank lines interspersed with them) from *section and
// returns that run, so an appending writer can re-emit it BELOW the new block.
// The run must contain at least one real definition — a tail of pure blank lines
// is not refs and is left to the caller's blank-trimming. Both the newly exposed
// end of *section and the leading blanks of the returned run are trimmed, so the
// caller reinstates exactly one separator on each side.
func PeelTrailingLinkRefs(section *[]string) []string {
	s := *section
	start := len(s)
	for i := len(s) - 1; i >= 0; i-- {
		t := strings.TrimRight(s[i], "\r")
		if strings.TrimSpace(t) == "" || linkRefDefRe.MatchString(t) {
			start = i
			continue
		}
		break
	}
	hasRef := false
	for _, ln := range s[start:] {
		if linkRefDefRe.MatchString(strings.TrimRight(ln, "\r")) {
			hasRef = true
			break
		}
	}
	if !hasRef {
		return nil
	}
	refs := s[start:]
	s = s[:start]
	for len(s) > 0 && strings.TrimSpace(s[len(s)-1]) == "" {
		s = s[:len(s)-1]
	}
	for len(refs) > 0 && strings.TrimSpace(refs[0]) == "" {
		refs = refs[1:]
	}
	*section = s
	return refs
}

// CodeSpanRanges returns the byte ranges of the inline code spans on one line,
// each paired by termsafe.PairCodeSpan (CommonMark: a run of backticks closed
// by the next run of exactly the same length).
func CodeSpanRanges(ln string) [][2]int {
	var out [][2]int
	for i := 0; i < len(ln); {
		if ln[i] != '`' {
			i++
			continue
		}
		sp, ok := termsafe.PairCodeSpan(ln, i)
		if !ok {
			i = runEnd(ln, i)
			continue
		}
		out = append(out, [2]int{sp.Start, sp.End})
		i = sp.End
	}
	return out
}

// MaskCodeSpans hides each inline code span in s behind an opaque token, so a
// rewrite of the text around the spans (a link read as its text, an HTML tag
// dropped, a character reference decoded) cannot reach into one, where
// CommonMark reads every character literally. Spans pair by
// termsafe.PairCodeSpan, so an unpaired run is literal backticks, and a
// backtick behind a backslash opens nothing (a backslash behind one does). A token is a NUL, the span's
// ordinal and a NUL, which no markdown construct matches; a string already
// holding a NUL is left unmasked.
//
// unmask puts each span back into a rewritten string, as written or, with
// rendered, as the text it renders (termsafe.CodeSpanText); a token the
// rewrite dropped stays dropped.
func MaskCodeSpans(s string) (masked string, unmask func(masked string, rendered bool) string) {
	if strings.IndexByte(s, 0) >= 0 {
		return s, func(m string, _ bool) string { return m }
	}
	var spans []termsafe.PairedSpan
	var b strings.Builder
	last := 0
	for i := 0; i < len(s); {
		switch {
		case s[i] == '\\' && i+1 < len(s) && isASCIIPunct(s[i+1]):
			i += 2
		case s[i] == '`':
			sp, ok := termsafe.PairCodeSpan(s, i)
			if !ok {
				i = runEnd(s, i)
				continue
			}
			b.WriteString(s[last:sp.Start])
			fmt.Fprintf(&b, "\x00%d\x00", len(spans))
			spans = append(spans, sp)
			last, i = sp.End, sp.End
		default:
			i++
		}
	}
	if spans == nil {
		return s, func(m string, _ bool) string { return m }
	}
	b.WriteString(s[last:])
	return b.String(), func(m string, rendered bool) string {
		return codeSpanTokenRe.ReplaceAllStringFunc(m, func(tok string) string {
			k, err := strconv.Atoi(tok[1 : len(tok)-1])
			if err != nil || k >= len(spans) {
				return tok
			}
			sp := spans[k]
			if rendered {
				return termsafe.CodeSpanText(s[sp.ContentStart:sp.ContentEnd])
			}
			return s[sp.Start:sp.End]
		})
	}
}

// codeSpanTokenRe is the token MaskCodeSpans puts in a span's place.
var codeSpanTokenRe = regexp.MustCompile("\x00[0-9]+\x00")

// runEnd returns the index just past the run of backticks at i: an unpaired run
// is literal backticks, stepped over whole so none of them opens a span.
func runEnd(ln string, i int) int {
	for i < len(ln) && ln[i] == '`' {
		i++
	}
	return i
}

// InAnyRange reports whether pos falls inside one of the ranges.
func InAnyRange(ranges [][2]int, pos int) bool {
	for _, r := range ranges {
		if pos >= r[0] && pos < r[1] {
			return true
		}
	}
	return false
}

// LabelledParagraph returns the [start, end) line bounds of the first live
// paragraph whose first line opens with the bold label `**<label>.**`, and
// whether one exists. The paragraph runs to the next blank line, the next live
// heading, the next masked line, or the end of the file, whichever is first.
//
// It is the one notion of a labelled paragraph, shared by the record lint that
// judges a principle's statement and the reading assembler that projects it
// (spc-2609020626042471): a principle carries one heading, its H1, and a body
// of labelled paragraphs (`**The rule.**`, `**Why.**`, `**Bounds.**`), so its
// statement is found by label rather than by heading, and two readers of that
// label that disagree are a gate refusing a paragraph the projection never
// sends, or the reverse.
//
// A label inside a fence or an HTML comment is an example, not a statement, and
// is never matched. The label must open the line: a paragraph that mentions
// `**The rule.**` part-way through is not the rule.
func LabelledParagraph(lines []string, label string) (start, end int, ok bool) {
	mask := Mask(lines)
	opener := "**" + label + ".**"
	for i, ln := range lines {
		if masked(mask, i) || !strings.HasPrefix(strings.TrimRight(ln, "\r"), opener) {
			continue
		}
		// The paragraph must BEGIN here: a line directly continuing a previous
		// paragraph is inside that paragraph, whatever it opens with.
		if i > 0 && !masked(mask, i-1) && strings.TrimSpace(lines[i-1]) != "" && !IsHeading(lines[i-1]) {
			continue
		}
		end = len(lines)
		for j := i + 1; j < len(lines); j++ {
			if masked(mask, j) || strings.TrimSpace(lines[j]) == "" || IsHeading(strings.TrimRight(lines[j], "\r")) {
				end = j
				break
			}
		}
		return i, end, true
	}
	return 0, 0, false
}
