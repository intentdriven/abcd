package termsafe

import (
	"slices"
	"strings"
)

// PairedSpan is where one CommonMark code span sits in the string it was paired
// in: s[Start:End] is the whole span, delimiters included, and
// s[ContentStart:ContentEnd] is the raw text between the two backtick runs.
type PairedSpan struct {
	Start, ContentStart, ContentEnd, End int
}

// Raw is the span's text as written between its delimiters, before CodeSpanText
// normalises it.
func (c PairedSpan) Raw(s string) string { return s[c.ContentStart:c.ContentEnd] }

// PairCodeSpan is the tree's one code-span pairer. It reads the backtick run
// that begins at s[i] and reports the span that run opens, or ok=false when it
// opens none and is literal backticks.
//
// The rule is CommonMark's: a backtick string is a maximal run, so the closer is
// the first later run of EXACTLY the opening length, and a run of any other
// length is span content skipped whole — a longer run never closes by its
// prefix and a shorter one never closes at all. Line endings are content, so a
// span paired over a paragraph may cross lines.
//
// i is the first backtick of the run: callers walk runs whole and never hand it
// the middle of one. Whether the run may open a span at all — a backtick behind
// a backslash is escaped, outside a span only — is the caller's walk to decide,
// because only the caller knows whether it reads escapes. Asking this function
// is how every reader and every escaper draws the SAME boundaries: the site
// renderer, the prose cleaner, the record's comment and span readers and the
// block escapers once each paired runs by their own walk, and the renderer's
// walk closed a span where the opening run's TEXT first recurred, so it refused
// a span a two-backtick run opened around a three-backtick run while the
// escapers had judged it balanced and left it unescaped (iss-2609262322244502).
//
// The walk visits each byte once per call, so a line whose runs all have
// DISTINCT lengths, asked once per run, costs time superlinear in its length
// (iss-2608301803425790). That shape is left in place on a measurement rather
// than a shrug: a line of 120 distinct-length runs costs ~200us, an ordinary
// record line ~76ns, and both candidate fixes cost more than they save.
// Precomputing the runs into a slice takes the bad line to ~7us and the
// ordinary line to ~115ns with one allocation — every line paying for a shape
// no record body has; stepping between runs with strings.IndexByte leaves the
// ordinary line alone and takes the bad line to ~243us, the gaps being too
// short to repay the call. Re-run mdrecord's BenchmarkOpensCommentDistinctRuns
// and BenchmarkOpensCommentTypicalLine before revisiting this.
func PairCodeSpan(s string, i int) (span PairedSpan, ok bool) {
	if i < 0 || i >= len(s) || s[i] != '`' {
		return PairedSpan{}, false
	}
	open := backtickRunEnd(s, i)
	n := open - i
	for j := open; j < len(s); {
		if s[j] != '`' {
			j++
			continue
		}
		k := backtickRunEnd(s, j)
		if k-j == n {
			return PairedSpan{Start: i, ContentStart: open, ContentEnd: j, End: k}, true
		}
		j = k
	}
	return PairedSpan{}, false
}

// CodeSpanPairer pairs every backtick run of one string by PairCodeSpan's
// rule, for a walk that asks about its runs left to right: the page readers'
// walk of one line (mdrecord). Asking PairCodeSpan at each run costs
// a scan to the closer, and to the end of the string for a run nothing closes,
// so a string of runs that never close costs the square of its length; the
// pairer lists the string's runs once, by length, and answers each question by
// moving that length's cursor forward, so a whole walk costs time in line with
// the string.
type CodeSpanPairer struct {
	runs  map[int][]int // each maximal run's start, by its length, in order
	next  map[int]int   // per length, the first listed run not yet passed
	steps int
}

// Steps is the pairer's whole cost so far: one step per byte it listed and
// per run a cursor passed, for a caller that holds its own cost to a bound.
func (p *CodeSpanPairer) Steps() int { return p.steps }

// NewCodeSpanPairer lists the runs of s.
func NewCodeSpanPairer(s string) *CodeSpanPairer {
	p := &CodeSpanPairer{runs: map[int][]int{}, next: map[int]int{}}
	for i := 0; i < len(s); {
		p.steps++
		if s[i] != '`' {
			i++
			continue
		}
		j := backtickRunEnd(s, i)
		p.runs[j-i] = append(p.runs[j-i], i)
		i = j
	}
	return p
}

// Pair reports the span opened by the backticks s[i:end], where end is the end
// of the maximal run they sit in and i may sit inside it (a run whose first
// backtick is escaped opens past it): the first maximal run starting at or
// after end of exactly end-i backticks closes it, as PairCodeSpan pairs it.
// Calls must come in rising order of end.
func (p *CodeSpanPairer) Pair(i, end int) (PairedSpan, bool) {
	n := end - i
	starts := p.runs[n]
	k := p.next[n]
	for k < len(starts) && starts[k] < end {
		p.steps++
		k++
	}
	p.next[n] = k
	if k == len(starts) {
		return PairedSpan{}, false
	}
	j := starts[k]
	return PairedSpan{Start: i, ContentStart: end, ContentEnd: j, End: j + n}, true
}

// The fence characters LongestFenceRun counts: CommonMark builds code spans
// and fenced blocks from backticks, and fenced blocks from tildes too. They
// are named here so a writer outside this package asks for a backtick run
// without spelling a backtick, which TestNoSecondCodeSpanPairer reads as a
// hand-written scan.
const (
	FenceBacktick byte = '`'
	FenceTilde    byte = '~'
)

// LongestFenceRun returns the length of the longest unbroken run of one fence
// character in s, counting only the characters listed in fences, or 0 when s
// holds none of them. It is the tree's one counter for a writer that sizes a
// code span or a fence one longer than anything in the text it wraps, so the
// text cannot close it.
//
// A run is one character repeated: a backtick followed by a tilde starts a new
// run rather than lengthening the old one. That is CommonMark's own reading,
// since a closing fence must repeat its opener's character, so a mixed stretch
// can close nothing longer than its longest single-character part. Every
// writer today wants exactly that, so there is no mixed-run mode.
//
// The scan is bytewise and fences must be ASCII. Every byte of a multi-byte
// UTF-8 sequence is at or above 0x80, so text around a run never matches a
// fence character, and the result equals a rune-by-rune count.
func LongestFenceRun(s string, fences ...byte) int {
	best, run := 0, 0
	var prev byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case run > 0 && c == prev:
			run++
		case slices.Contains(fences, c):
			run = 1
		default:
			run = 0
		}
		prev = c
		best = max(best, run)
	}
	return best
}

// backtickRunEnd returns the index just past the run of backticks at i.
func backtickRunEnd(s string, i int) int {
	for i < len(s) && s[i] == '`' {
		i++
	}
	return i
}

// codeSpanLineEndings is CommonMark's first content rule: a line ending inside
// a span is a space.
var codeSpanLineEndings = strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ")

// CodeSpanText is the text a span's raw content renders as, by CommonMark's two
// content rules: every line ending becomes a space, and then ONE leading and
// ONE trailing space are stripped, only when both are present and the content
// is not all spaces. So a span of a lone space keeps it, a span padded on one
// side keeps its padding, and a span padded to hold a backtick at its edge
// gives up exactly the padding the writer added.
func CodeSpanText(raw string) string {
	s := codeSpanLineEndings.Replace(raw)
	if len(s) >= 2 && s[0] == ' ' && s[len(s)-1] == ' ' && strings.Trim(s, " ") != "" {
		return s[1 : len(s)-1]
	}
	return s
}
