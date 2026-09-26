package guard

import (
	"strconv"
	"strings"
)

// Bounded brace expansion (iss-2608282026038930). bash rewrites an unquoted
// brace group into several words before the command runs — `mkdir -p
// foo/{a,b}` is `mkdir -p foo/a foo/b`, and `git push {--force,} origin main`
// is a force push — so the guard expands the group the same way and checks the
// argv the shell will actually build. The expansion follows bash 5.3's own
// algorithm (braces.c: brace_expand, brace_gobbler, expand_seqterm), read on a
// word whose bytes each carry whether they reached the tokenizer unquoted: only
// an unquoted `{`, `,`, `}` or `..` is structure, exactly as in bash.
//
// It is BOUNDED, and the bound is the only place it refuses. A group can
// multiply a word without limit (`{1..99999999}`, a dozen adjacent pairs), so
// the words and bytes one tokenize call may produce, and the scanning the
// expander may spend, are capped; past any cap the word is left unexpanded and
// its segment carries braceGroup, which Check folds into the fail-closed block.

const (
	// braceMaxWords caps the words brace expansion may produce across one
	// tokenize call. Everyday groups produce a handful (`foo/{a,b}/{c,d}` is
	// four); a range over a directory of numbered files, a few hundred.
	braceMaxWords = 4096
	// braceMaxBytes caps the bytes those words may hold, so a long word
	// multiplied by a modest count cannot allocate without limit either.
	braceMaxBytes = 1 << 20
	// braceMaxWork caps the byte steps the expander's scans may take across one
	// tokenize call. The opener search restarts after every `{` that fails to
	// open a group, so a word of nothing but braces is quadratic without it.
	braceMaxWork = 1 << 18
)

// Per-byte flags the tokenizer records beside a word it is building.
const (
	// wordStruct marks a byte that reached the tokenizer unquoted and
	// unescaped, so bash reads it as structure where structure is possible.
	wordStruct byte = 1 << iota
	// wordNotOpener marks a `{` bash never opens a group at: the first raw
	// byte of its word with a raw `}` or blank directly after it (`{},a}` is a
	// literal). Only the raw line can say so — `{""},a}` does open one — so the
	// tokenizer decides it and the expander obeys.
	wordNotOpener
)

// braceLimits is the shared budget for one tokenize call.
type braceLimits struct {
	words, bytes, work int
}

func newBraceLimits() braceLimits {
	return braceLimits{words: braceMaxWords, bytes: braceMaxBytes, work: braceMaxWork}
}

// bword is a word under expansion: its bytes and, parallel to them, the flags
// above.
type bword struct {
	b []byte
	m []byte
}

func (w bword) slice(lo, hi int) bword { return bword{b: w.b[lo:hi], m: w.m[lo:hi]} }

func (w bword) structAt(i int) bool { return i >= 0 && i < len(w.m) && w.m[i]&wordStruct != 0 }

// concat joins words into a fresh one.
func concat(parts ...bword) bword {
	n := 0
	for _, p := range parts {
		n += len(p.b)
	}
	out := bword{b: make([]byte, 0, n), m: make([]byte, 0, n)}
	for _, p := range parts {
		out.b = append(out.b, p.b...)
		out.m = append(out.m, p.m...)
	}
	return out
}

// globbed reports whether the word holds an unquoted glob metacharacter, the
// per-token record the matcher reads (tokenize's curGlob, per expansion).
func (w bword) globbed() bool {
	for i, c := range w.b {
		if (c == '*' || c == '?' || c == '[') && w.m[i]&wordStruct != 0 {
			return true
		}
	}
	return false
}

// expandBraces expands every brace group in w, returning the words bash would
// produce (an empty result is legal: `{,}` produces none). ok is false when a
// limit was reached, and the caller then keeps the word unexpanded and refuses
// the segment. Every intermediate list is bounded by the words still left in the
// cap; the words returned are charged against it here.
func expandBraces(w bword, lim *braceLimits) (out []bword, ok bool) {
	out, ok = braceExpand(w, lim)
	if !ok {
		return nil, false
	}
	if lim.words -= len(out); lim.words < 0 {
		return nil, false
	}
	// bash drops the words an expansion leaves empty (`{--force,}` is one word).
	kept := out[:0]
	for _, x := range out {
		if len(x.b) > 0 {
			kept = append(kept, x)
		}
	}
	return kept, true
}

// braceExpand is bash's brace_expand over one word.
func braceExpand(w bword, lim *braceLimits) ([]bword, bool) {
	// Find the first opening brace that has a matching, separated close.
	open, close := -1, -1
	for i := 0; ; i++ {
		o, ok := braceGobble(w, i, '{', lim)
		if !ok {
			return nil, false
		}
		if o < 0 {
			return []bword{w}, true // no group: the word is literal
		}
		c, ok := braceGobble(w, o+1, '}', lim)
		if !ok {
			return nil, false
		}
		if c >= 0 {
			open, close = o, c
			break
		}
		i = o // the next search starts past this brace
	}

	pre, amble, post := w.slice(0, open), w.slice(open+1, close), w.slice(close+1, len(w.b))

	var tack []bword
	if sep, ok := braceSplit(amble, lim); !ok {
		return nil, false
	} else if len(sep) == 1 {
		// No separator: the amble is a sequence expression or nothing.
		seq, ok, valid := braceSequence(amble, lim)
		if !ok {
			return nil, false
		}
		switch {
		case valid:
			tack = seq
		case len(post.b) > 0:
			// bash keeps the braces as literal text and still expands what
			// follows them.
			tack = []bword{literal(w.slice(open, close+1))}
		default:
			return []bword{w}, true
		}
	} else {
		for _, part := range sep {
			sub, ok := braceExpand(part, lim)
			if !ok {
				return nil, false
			}
			if tack = append(tack, sub...); len(tack) > lim.words {
				return nil, false
			}
		}
	}

	posts := []bword{{}}
	if len(post.b) > 0 {
		var ok bool
		if posts, ok = braceExpand(post, lim); !ok {
			return nil, false
		}
	}
	n := len(tack) * len(posts)
	if n > lim.words {
		return nil, false
	}
	out := make([]bword, 0, n)
	for _, t := range tack {
		for _, p := range posts {
			x := concat(pre, t, p)
			if lim.bytes -= len(x.b); lim.bytes < 0 {
				return nil, false
			}
			out = append(out, x)
		}
	}
	return out, true
}

// literal returns w with every structural flag cleared, so no later pass reads
// its braces as structure.
func literal(w bword) bword {
	return bword{b: append([]byte(nil), w.b...), m: make([]byte, len(w.b))}
}

// braceGobble is bash's brace_gobbler: from index i, find the byte satisfy
// (`{` or `}`) at nesting level zero, returning its index or -1. Looking for a
// close, it answers only once a separator (a `,`, or a `..` not directly
// before the close) has been seen at level zero, which is why `{msg},x}` closes
// at the second brace. `${` opens a level without being an opener itself, and
// a brace marked wordNotOpener is passed over.
func braceGobble(w bword, i int, satisfy byte, lim *braceLimits) (int, bool) {
	level, seps := 0, 0
	if satisfy == '{' {
		seps = 1
	}
	for ; i < len(w.b); i++ {
		if lim.work--; lim.work < 0 {
			return -1, false
		}
		if !w.structAt(i) {
			continue
		}
		c := w.b[i]
		if c == '$' && w.structAt(i+1) && w.b[i+1] == '{' {
			i++
			level++
			continue
		}
		if c == satisfy && level == 0 && seps > 0 {
			if c == '{' && w.m[i]&wordNotOpener != 0 {
				continue
			}
			return i, true
		}
		switch {
		case c == '{':
			level++
		case c == '}' && level > 0:
			level--
		case satisfy == '}' && c == ',' && level == 0:
			seps++
		case satisfy == '}' && c == '.' && level == 0 && w.structAt(i+1) && w.b[i+1] == '.' &&
			!(w.structAt(i+2) && w.b[i+2] == '}'):
			seps++
		}
	}
	return -1, true
}

// braceSplit splits an amble at its level-zero unquoted commas (bash's
// expand_amble). A single part means the amble held no separator.
func braceSplit(amble bword, lim *braceLimits) ([]bword, bool) {
	var parts []bword
	level, start := 0, 0
	for i := 0; i < len(amble.b); i++ {
		if lim.work--; lim.work < 0 {
			return nil, false
		}
		if !amble.structAt(i) {
			continue
		}
		switch c := amble.b[i]; {
		case c == '$' && amble.structAt(i+1) && amble.b[i+1] == '{':
			i++
			level++
		case c == '{':
			level++
		case c == '}' && level > 0:
			level--
		case c == ',' && level == 0:
			parts = append(parts, amble.slice(start, i))
			start = i + 1
		}
	}
	return append(parts, amble.slice(start, len(amble.b))), true
}

// braceSequence is bash's expand_seqterm: `{x..y}` or `{x..y..incr}` over
// integers (zero-padded when an end is written with a leading zero) or single
// letters. valid is false when the amble is not a sequence expression, which
// bash leaves as literal text; ok is false when the sequence is past the cap.
// Every byte must be unquoted — `{'1'..3}` is literal in bash.
func braceSequence(amble bword, lim *braceLimits) (out []bword, ok, valid bool) {
	for i := range amble.b {
		if !amble.structAt(i) {
			return nil, true, false
		}
	}
	s := string(amble.b)
	lhs, rest, found := strings.Cut(s, "..")
	if !found {
		return nil, true, false
	}
	rhs, incrText, hasIncr := strings.Cut(rest, "..")
	incr := int64(1)
	if hasIncr {
		n, err := strconv.ParseInt(incrText, 10, 64)
		if err != nil {
			return nil, true, false
		}
		incr = n
	}
	if incr < 0 {
		incr = -incr
	}
	if incr == 0 {
		incr = 1
	}

	lo, lerr := strconv.ParseInt(lhs, 10, 64)
	hi, rerr := strconv.ParseInt(rhs, 10, 64)
	var start, end int64
	isChar := false
	switch {
	case lerr == nil && rerr == nil:
		start, end = lo, hi
	case len(lhs) == 1 && len(rhs) == 1 && isLetter(lhs[0]) && isLetter(rhs[0]):
		start, end, isChar = int64(lhs[0]), int64(rhs[0]), true
	default:
		return nil, true, false
	}

	span := end - start
	if span < 0 {
		span = -span
	}
	count := span/incr + 1
	if count > int64(lim.words) {
		return nil, false, true
	}
	width := 0
	if !isChar {
		width = seqWidth(lhs, rhs)
	}
	step := incr
	if start > end {
		step = -incr
	}
	out = make([]bword, 0, count)
	for v, n := start, int64(0); n < count; v, n = v+step, n+1 {
		var text string
		if isChar {
			text = string(rune(v))
		} else {
			text = padInt(v, width)
		}
		if lim.bytes -= len(text); lim.bytes < 0 {
			return nil, false, true
		}
		out = append(out, bword{b: []byte(text), m: make([]byte, len(text))})
	}
	return out, true, true
}

func isLetter(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }

// seqWidth is the zero-padding width bash gives an integer sequence: the
// length of an end written with a leading zero (`01`, `-02`), else none.
func seqWidth(lhs, rhs string) int {
	width := 0
	for _, e := range []string{lhs, rhs} {
		if (len(e) > 1 && e[0] == '0') || (len(e) > 2 && e[0] == '-' && e[1] == '0') {
			if len(e) > width {
				width = len(e)
			}
		}
	}
	return width
}

// padInt renders v zero-padded to width, the sign counted in the width as bash
// counts it (`{-02..2}` is -02 -01 000 001 002).
func padInt(v int64, width int) string {
	if v < 0 {
		digits := strconv.FormatInt(-v, 10)
		for len(digits)+1 < width {
			digits = "0" + digits
		}
		return "-" + digits
	}
	digits := strconv.FormatInt(v, 10)
	for len(digits) < width {
		digits = "0" + digits
	}
	return digits
}
