package mdrecord

import (
	"strings"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// Code spans and comments, read line by line, and where the lines above make
// that reading unsure.
//
// The page pairs a code span over the lines of a paragraph: a run of
// backticks closes at the next run of exactly its length, over a line break,
// and a run nothing closes before the paragraph ends is literal backticks.
// Which lines one paragraph holds turns on the blocks around it (lists,
// blockquotes, tables, HTML, link reference definitions, lazy continuation
// lines), and this package does not parse those, so it does not pair a span
// over a line break. It pairs spans on each line alone and says, for each
// line, whether that pairing is sure; each reader takes the side that is safe
// for it where it is not:
//
//   - Spans are paired on one line (termsafe.CodeSpanPairer, PairCodeSpan's
//     rule), and a backtick behind an odd run of backslashes opens none, as
//     on the page outside a span; a closer is found with no escape test, as
//     inside a span a backslash is literal. CodeSpanRanges is that pairing.
//   - A stretch is a run of lines with no blank line between them, a blank
//     line being one that is empty or holds nothing but spaces and tabs,
//     inside a fence or comment as well as outside. No heading, fence, list
//     item, blockquote, table or HTML line ends a stretch, so a paragraph the
//     page forms lies inside one stretch.
//   - Walking a stretch top to bottom, a line that leaves a backtick run with
//     no closer of its length later on that line leaves the run open. The
//     lines below it, to the stretch's end, are unsure, and so is the rest of
//     the run's own line after the run: the page may pair the run with a
//     backtick below. The spans on the run's own line, before the run and
//     after it, are sure: the page pairs those the same whether the run is
//     literal or pairs with a backtick on a later line, since a span the run
//     opens covers them.
//   - A comment opener inside a one-line span on a sure line opens nothing:
//     the walk steps over the span whole. One outside a span opens a comment.
//     Backticks inside a comment are literal.
//   - Where a line is unsure, a comment opener that is the first thing on
//     its line, at up to three columns, opens a comment whatever the lines
//     above leave open, as an HTML comment block starts at such a line. Any
//     other opener is read as a line read alone reads it, and where it has no
//     `-->` after it on the line, met by the walk or inside a span the walk
//     steps over, the reading is Unsure from that line (Reading.Unsure).
//
// ReadPage, the reading the readiness gate takes, blanks no span on a line
// below an open run, reads the text of a comment whose opener is unsure and
// closes on its line, and reads every line whole from the line Unsure names.
// Mask, and the readers that pair spans with CodeSpanRanges (the
// scope-condition and hand-audit readers and the prose-citation rule), read
// every line as a line read alone, sure or not.
//
// Older limits these readings keep: a backtick inside an HTML tag, an
// autolink, an inline link's destination or title, or a processing
// instruction, declaration or CDATA section in the middle of a line, is
// paired as if it stood in text; a comment opener inside a tag's attribute
// opens a comment; a one-line comment whose text holds two hyphens is read as
// a comment; a one-line span or comment inside an indented code block or an
// HTML block is blanked, and so is a span over a table cell's `|`; a backslash
// before a backtick inside an indented code block, an HTML block or a tag,
// where the page reads it as a literal backslash, still makes the backtick
// literal; and a carriage return alone is not a line end.

// lineRead is what one line's walk reads.
type lineRead struct {
	// spans are the byte ranges inside a one-line code span, paired on the
	// line alone, outside the comments the walk stepped over.
	spans [][2]int
	// comments are the byte ranges inside an HTML comment the walk opened
	// where it is sure of the opener (on a sure line, or first on its line at
	// up to three columns), the last of which may run to the line's end, and
	// Read adds the part of the line inside a comment an earlier line opened:
	// Page.Text blanks them. A comment opened where the line is unsure and not
	// first on its line is not among them.
	comments [][2]int
	// openers are the offsets of the comment openers that opened a comment.
	openers []int
	// open is whether a comment is open at the line's end.
	open bool
	// loose is whether a backtick run on the line has no closer of its
	// length after it on the line.
	loose bool
	// under is whether the walk began below a run a line above it, in its
	// stretch, left open: the line's spans are not certain.
	under bool
	// doubt is whether, where the line is unsure, a comment opener that is
	// not the first thing on its line, at up to three columns, has no closer
	// after it on the line: the page may read a comment there that runs past
	// the line, or none.
	doubt bool
}

// walkLine walks ln left to right from byte from, the first construct
// winning, as a line read alone is read: at a backtick run it pairs a span on
// the line and steps over it whole, or steps over the run as literal
// backticks; with comments, at a `<!--` it steps to the `-->` closing it, or
// to the line's end.
//
// unsure is whether the walk begins below a run a line above it left open.
// From there, or from a run this line leaves open, the line is unsure: the
// page may pair that run over the line break and hold any comment opener
// after it inside a span. There a comment opener that is the first thing on
// its line, at up to three columns, opens a comment whatever the lines above
// leave open, as an HTML comment block starts at such a line; any other
// opener with no `-->` after it on the line, met by the walk or inside a span
// it steps over, sets doubt.
// Each byte is visited a bounded number of times, and each step the walk
// takes is reported to walkHook.
func walkLine(ln string, from int, comments, unsure bool) lineRead {
	w := lineRead{under: unsure}
	if from >= len(ln) {
		return w
	}
	stepN(len(ln) - from)
	var pr *termsafe.CodeSpanPairer
	if strings.IndexByte(ln[from:], '`') >= 0 {
		pr = termsafe.NewCodeSpanPairer(ln)
		defer func() { stepN(pr.Steps()) }()
	}
	// closed is the offset past which no `-->` starts on the line: an opener
	// at p has a closer after it when the last one starts at p+2 or later.
	closed := -1
	if comments && unsure || comments && pr != nil {
		stepN(len(ln))
		closed = strings.LastIndex(ln, "-->")
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
					if comments && unsure {
						stepN(sp.End - open)
						if k := strings.LastIndex(ln[open:sp.End], "<!--"); k >= 0 && closed < open+k+2 {
							w.doubt = true
						}
					}
					p = sp.End
					continue
				}
				w.loose = true
			}
			p = q
		case comments && c == '<' && strings.HasPrefix(ln[p:], "<!--"):
			sure := !unsure && !w.loose || from == 0 && p <= 3 && strings.TrimLeft(ln[:p], " ") == ""
			w.openers = append(w.openers, p)
			// The closer may share the opener's dashes: `<!-->` and `<!--->`
			// are whole, empty comments (CommonMark 0.31, and HTML's own
			// reading), so the search starts two bytes in.
			rest := ln[p+len("<!"):]
			k := strings.Index(rest, "-->")
			if k < 0 {
				stepN(len(rest))
				if sure {
					w.comments = append(w.comments, [2]int{p, len(ln)})
				} else {
					w.doubt = true
				}
				w.open = true
				return w
			}
			stepN(k + len("-->"))
			end := p + len("<!") + k + len("-->")
			if sure {
				w.comments = append(w.comments, [2]int{p, end})
			}
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

// walkHook, when set, is called once per step the line walk takes, each
// byte it visits, scans or lists counted (stepN); a test counts the calls.
// stepHook counts the walks ReadPage makes beyond Read's own pass.
var walkHook func()

// stepN reports n steps of the line walk to walkHook.
func stepN(n int) {
	if walkHook != nil {
		for range n {
			walkHook()
		}
	}
}
