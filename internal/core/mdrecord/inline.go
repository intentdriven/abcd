package mdrecord

import (
	"strings"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// Code spans, blanked only where the readings of the paragraph agree.
//
// The page pairs a code span over the lines of a paragraph: a run of
// backticks closes at the next run of exactly its length, over a line break,
// and a run nothing closes before the paragraph ends is literal backticks.
// Which lines one paragraph holds turns on the blocks around it (lists,
// blockquotes, tables, HTML, link reference definitions, lazy continuation
// lines), and this package does not parse those, so it does not pair a span
// over a line break. It blanks a span only where the pairing does not turn on
// the paragraph's extent, and reads the text everywhere else:
//
//   - A stretch is a run of lines with no blank line between them, a blank
//     line being one that is empty or holds nothing but spaces and tabs. No
//     heading, fence, list item, blockquote, table or HTML line ends a
//     stretch, so a paragraph the page forms lies inside one stretch.
//   - Spans are paired on one line (termsafe.CodeSpanPairer, PairCodeSpan's
//     rule), and a backtick behind an odd run of backslashes opens none, as
//     on the page outside a span.
//   - Walking a stretch top to bottom, a line that leaves a backtick run with
//     no closer of its length later on that line leaves the run open. From
//     that run to the stretch's end no comment opener opens a comment, and
//     the lines below it report no span: they are read whole. The line that
//     holds the run keeps its one-line spans, before the run and after it:
//     the page pairs those the same whether the run is literal or pairs with
//     a backtick on a later line, since a span the run opens covers them.
//   - A span over a line break is therefore not reported, and its text is
//     read as prose.
//   - A comment opener inside a one-line span opens nothing: the walk steps
//     over the span whole. Backticks inside a comment are literal.
//
// Older limits this reading keeps: a backtick inside an HTML tag or an
// autolink is paired as if it stood in text; a comment opener inside a tag's
// attribute opens a comment; a one-line span or comment inside an indented
// code block or an HTML block is blanked, and so is a span over a table
// cell's `|`; and a carriage return alone is not a line end.

// lineRead is what one line's walk reads: the byte ranges inside a one-line
// code span and inside an HTML comment (the last of which may run to the
// line's end), the offset of each comment opener that opened a comment, and
// whether a comment is open at the line's end.
type lineRead struct {
	spans    [][2]int
	comments [][2]int
	openers  []int
	open     bool
	// loose: a backtick run on the line has no closer of its length after
	// it on the line.
	loose bool
}

// walkLine walks ln left to right from byte from, the first construct
// winning: at a backtick run it pairs a span on the line and steps over it
// whole, or steps over the run as literal backticks; with comments, at a
// `<!--` it steps to the `-->` closing it, or to the line's end. With
// pastLoose a comment opener after a run the line leaves open still opens a
// comment; without it, none does. Each byte is visited a bounded number of
// times, and each step the walk takes is reported to stepHook.
func walkLine(ln string, from int, comments, pastLoose bool) lineRead {
	var w lineRead
	if from >= len(ln) {
		return w
	}
	stepN(len(ln) - from)
	var pr *termsafe.CodeSpanPairer
	if strings.IndexByte(ln[from:], '`') >= 0 {
		pr = termsafe.NewCodeSpanPairer(ln)
		defer func() { stepN(pr.Steps()) }()
	}
	for p := from; p < len(ln); {
		stepN(1)
		switch c := ln[p]; {
		case c == '`':
			q := runEnd(ln, p)
			stepN(q - p)
			// Outside a span a backtick behind an odd run of backslashes is
			// escaped, a literal character; the rest of its run may open.
			open, n := p, 0
			for k := p; k > 0 && ln[k-1] == '\\'; k-- {
				stepN(1)
				n++
			}
			if n%2 == 1 {
				open++
			}
			if open < q {
				if sp, ok := pr.Pair(open, q); ok {
					w.spans = append(w.spans, [2]int{open, sp.End})
					p = sp.End
					continue
				}
				w.loose = true
			}
			p = q
		case comments && c == '<' && strings.HasPrefix(ln[p:], "<!--") && (pastLoose || !w.loose):
			// The closer may share the opener's dashes: `<!-->` and `<!--->`
			// are whole, empty comments (CommonMark 0.31, and HTML's own
			// reading), so the search starts two bytes in.
			w.openers = append(w.openers, p)
			rest := ln[p+len("<!"):]
			k := strings.Index(rest, "-->")
			if k < 0 {
				stepN(len(rest))
				w.comments = append(w.comments, [2]int{p, len(ln)})
				w.open = true
				return w
			}
			stepN(k + len("-->"))
			end := p + len("<!") + k + len("-->")
			w.comments = append(w.comments, [2]int{p, end})
			p = end
		default:
			p++
		}
	}
	return w
}

// blankLine reports whether a line ends a stretch: it is empty or holds
// nothing but spaces and tabs. A line of other white space, which the page
// reads as text, does not.
func blankLine(ln string) bool {
	for i := 0; i < len(ln); i++ {
		stepN(1)
		if ln[i] != ' ' && ln[i] != '\t' {
			return false
		}
	}
	return true
}

// textSpans is Page.Spans: the one-line code spans of each line of text, the
// page's text with its comments already blanked, walking each stretch of src
// top to bottom; a line skip names reports none and leaves the stretch as it
// was, and from a run a line leaves open the lines below it in its stretch
// report none.
func textSpans(src, text []string, skip func(int) bool) [][][2]int {
	out := make([][][2]int, len(text))
	loose := false
	for i := range text {
		if blankLine(strings.TrimRight(src[i], "\r")) {
			loose = false
			continue
		}
		if skip(i) || loose {
			continue
		}
		w := walkLine(text[i], 0, false, false)
		out[i], loose = w.spans, w.loose
	}
	return out
}

// stepN reports n steps to stepHook.
func stepN(n int) {
	if stepHook != nil {
		for range n {
			stepHook()
		}
	}
}
