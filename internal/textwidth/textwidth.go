// Package textwidth is the one display-width measure and word wrap: how many
// terminal columns plain text occupies, and how words are laid into lines of at
// most a given width. The banner wraps its tagline with it, and the question
// limits (internal/core/question) estimate a question's rows with it, so the
// two measure text the same way.
//
// It is a pure leaf: no terminal I/O, no environment, no state. That is what
// lets the transport-agnostic core import it without depending on a package
// that holds raw-mode code (spc-2610030944505997, open question 2, decided (a)).
package textwidth

import (
	"strings"

	"golang.org/x/text/width"
)

// Columns counts the terminal columns plain text occupies: East Asian wide and
// fullwidth runes take two, every other rune one. It measures plain text only;
// an escape sequence is counted rune by rune, so callers measure text before
// any styling is applied.
func Columns(s string) int {
	n := 0
	for _, r := range s {
		switch width.LookupRune(r).Kind() {
		case width.EastAsianWide, width.EastAsianFullwidth:
			n += 2
		default:
			n++
		}
	}
	return n
}

// Wrap word-wraps s into lines of at most limit display columns and never
// breaks a word; a word wider than limit stands alone on its line. The wrap is
// balanced: it keeps the fewest lines a greedy wrap at limit needs, then
// narrows the measure while that count holds, so the last line is never a
// stranded word.
func Wrap(s string, limit int) []string {
	words := strings.Fields(s)
	longest := 0
	for _, w := range words {
		longest = max(longest, Columns(w))
	}
	lines := Greedy(words, limit)
	for measure := limit - 1; measure >= longest; measure-- {
		narrower := Greedy(words, measure)
		if len(narrower) > len(lines) {
			break
		}
		lines = narrower
	}
	return lines
}

// Greedy fills each line with as many words as fit within limit display
// columns, one space between words; a word wider than limit stands alone.
func Greedy(words []string, limit int) []string {
	return fill(words, limit, limit)
}

// Hang lays one line of text out for a hanging indent: s comes back unchanged
// when it fits within first columns, its runs of spaces and its leading indent
// included, and otherwise breaks only at a space, the first line at most first
// columns and each later line at most rest, the caller indenting the later
// lines. No blank is carried to either side of a break, and a word wider than
// its line stands alone. The status board lays every row with it
// (iss-2610031207397996).
func Hang(s string, first, rest int) []string {
	return fill(strings.Split(s, " "), first, rest)
}

// fill is the one greedy fill Greedy and Hang share: words joined by one space,
// the first line at most first columns and every later line at most rest. An
// empty word is a second space in a run (Hang splits on single spaces); it is
// kept inside a line and dropped where the line breaks.
func fill(words []string, first, rest int) []string {
	var lines []string
	limit := first
	line, cols, started := "", 0, false
	for _, w := range words {
		wc := Columns(w)
		switch {
		case !started:
			if w == "" && len(lines) > 0 {
				continue
			}
			line, cols, started = w, wc, true
		case cols+1+wc <= limit:
			line, cols = line+" "+w, cols+1+wc
		default:
			lines = append(lines, strings.TrimRight(line, " "))
			limit = rest
			line, cols, started = w, wc, w != ""
		}
	}
	if started {
		lines = append(lines, line)
	}
	return lines
}
