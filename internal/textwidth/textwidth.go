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
	var lines []string
	line, cols := "", 0
	for _, w := range words {
		wc := Columns(w)
		switch {
		case line == "":
			line, cols = w, wc
		case cols+1+wc <= limit:
			line, cols = line+" "+w, cols+1+wc
		default:
			lines = append(lines, line)
			line, cols = w, wc
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}
