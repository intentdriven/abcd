// Package mdrecord is the Markdown machinery a RECORD BODY is read and written
// through: which lines are live markdown and which are not, where a section
// starts and stops, what a top-level bullet is, and where a trailing run of
// link-reference definitions ends.
//
// It is a leaf for the reason core/grounds and core/issueschema are. Two record
// families carry the same constructs — the intent record's scope conditions and
// audit notes, the issue ledger's grounds — and a body reader spelled twice is a
// reader the two can disagree about, which is how a bullet one writer appends is
// a bullet the other cannot find. It imports only the standard library: no
// filesystem, no transport, no record store, and no notion of which family a
// body belongs to.
//
// What it does NOT own is any heading's meaning. A caller supplies the heading
// pattern it is looking for; this package only says whether a line matching it
// is live, and which lines lie under it.
package mdrecord

import (
	"regexp"
	"strings"
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

	openLine int
	openFlag uint8
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
				if inComment = opensCommentFrom(ln, k+len("-->")); inComment {
					r.openLine, r.openFlag = i, MaskComment
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
			// A backtick opener's info string may not itself contain a backtick.
			if run, rest, indent, ok := fenceOpener(ln, rule); ok && !(run[0] == '`' && strings.Contains(rest, "`")) {
				fenceOpen, fenceIndent, fenceStart = run, indent, i
				r.Mask[i] |= MaskFence
				r.openLine, r.openFlag = i, MaskFence
				continue
			}
			if OpensComment(ln) {
				inComment = true
				r.Mask[i] |= MaskComment
				r.openLine, r.openFlag = i, MaskComment
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
func opensCommentFrom(ln string, start int) bool {
	for i := start; i < len(ln); {
		switch {
		case ln[i] == '`':
			j := backtickRunEnd(ln, i)
			if _, end, ok := findBacktickRun(ln, j, j-i); ok {
				i = end
			} else {
				i = j
			}
		case strings.HasPrefix(ln[i:], "<!--"):
			rest := ln[i+len("<!--"):]
			k := strings.Index(rest, "-->")
			if k < 0 {
				return true
			}
			i += len("<!--") + k + len("-->")
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

// CodeSpanRanges returns the byte ranges of the inline code spans on one line:
// a run of backticks closed by a run of the same length (CommonMark).
func CodeSpanRanges(ln string) [][2]int {
	var out [][2]int
	for i := 0; i < len(ln); {
		if ln[i] != '`' {
			i++
			continue
		}
		j := backtickRunEnd(ln, i)
		closeStart, closeEnd, ok := findBacktickRun(ln, j, j-i)
		if !ok {
			i = j
			continue
		}
		_ = closeStart
		out = append(out, [2]int{i, closeEnd})
		i = closeEnd
	}
	return out
}

// backtickRunEnd returns the index just past the run of backticks at i.
func backtickRunEnd(ln string, i int) int {
	for i < len(ln) && ln[i] == '`' {
		i++
	}
	return i
}

// findBacktickRun finds the next run of exactly n backticks at or after from.
//
// The search is per-run and walks the bytes between runs, so a line whose runs
// all have DISTINCT lengths asks once per length and the cost is superlinear in
// the line's length (iss-2608301803425790). That shape is left in place on a
// MEASUREMENT rather than a shrug, and this package's own benchmarks are the
// measurement: a line of 120 distinct-length runs costs ~200us, an ordinary
// record line ~76ns, and both candidate fixes cost more than they save.
// Precomputing the runs into a slice takes the bad line to ~7us and the ordinary
// line to ~115ns with one allocation — every line paying for a shape no record
// body has. Stepping between runs with strings.IndexByte leaves the ordinary
// line alone and takes the bad line to ~243us, the gaps being too short to repay
// the call. Re-run BenchmarkOpensCommentDistinctRuns and
// BenchmarkOpensCommentTypicalLine before revisiting this.
func findBacktickRun(ln string, from, n int) (start, end int, ok bool) {
	for k := from; k < len(ln); {
		if ln[k] != '`' {
			k++
			continue
		}
		e := backtickRunEnd(ln, k)
		if e-k == n {
			return k, e, true
		}
		k = e
	}
	return 0, 0, false
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
