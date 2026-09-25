package surface

// sentence.go — itd-2609212113220149: every visible verb and sub-verb carries one
// sentence an agent can act on, naming what the verb does, what it writes (or
// that it writes nothing) and when it refuses, in that order (the product
// thinker's decision 1, 2026-09-21). The sentence is declared once, in the
// manifest in sentences.go, and rendered from there onto the command list, the
// verb's own --help, the agents block and the verb's plugin page, so the places
// a reader meets it cannot disagree.
//
// The form is checked mechanically. The three clauses are found by their
// declared separators, a colon after the doing clause and a semicolon before the
// refusing clause (spc-2609212139583822 scope 2). The writing clause opens with
// "Writes", capitalised because the writing-style guide puts a capital after a
// colon, and the refusing clause opens with "refuses" or "never refuses", lower
// case because the guide puts lower case after a semicolon. Keying the clauses
// on those words is what lets a test tell a missing clause from a clause that
// is merely short.

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// SentenceCap is the most characters a sentence may run to (spc-2609212139583822
// scope 2). It counts characters, not bytes, because a reader counts characters
// and a `×` or a `—` is one of them.
const SentenceCap = 160

// Sentence is one verb's sentence split into its three clauses, each without its
// separator.
type Sentence struct {
	Does    string
	Writes  string
	Refuses string
}

// The words the second and third clauses open with.
const (
	writesOpening       = "Writes"
	refusesOpening      = "refuses"
	neverRefusesOpening = "never refuses"
)

// ParseSentence splits s into its three clauses, or refuses it with an error
// naming the defect: empty, more than one line, over the cap, not one sentence,
// no closing period, a missing, doubled or out-of-order separator, an empty
// doing clause or one opening in lower case, a writing clause that does not open
// with "Writes", or a refusing clause that opens with neither "refuses" nor
// "never refuses".
func ParseSentence(s string) (Sentence, error) {
	if strings.TrimSpace(s) == "" {
		return Sentence{}, errors.New("the sentence is empty")
	}
	if strings.ContainsAny(s, "\r\n") {
		return Sentence{}, errors.New("the sentence runs over more than one line")
	}
	if n := utf8.RuneCountInString(s); n > SentenceCap {
		return Sentence{}, fmt.Errorf("the sentence is %d characters, over the cap of %d", n, SentenceCap)
	}
	if !strings.HasSuffix(s, ".") {
		return Sentence{}, errors.New("the sentence does not close with a period")
	}
	body := strings.TrimSuffix(s, ".")
	if strings.Contains(body, ". ") {
		return Sentence{}, errors.New("the text is more than one sentence")
	}

	colons, semicolons := strings.Count(body, ":"), strings.Count(body, ";")
	switch {
	case colons == 0:
		return Sentence{}, errors.New("no colon closes the doing clause")
	case colons > 1:
		return Sentence{}, fmt.Errorf("the sentence has %d colons; it takes exactly one colon, after the doing clause", colons)
	case semicolons == 0:
		return Sentence{}, errors.New("no semicolon opens the refusing clause")
	case semicolons > 1:
		return Sentence{}, fmt.Errorf("the sentence has %d semicolons; it takes exactly one semicolon, before the refusing clause", semicolons)
	}
	does, rest, _ := strings.Cut(body, ":")
	writes, refuses, ok := strings.Cut(rest, ";")
	if !ok {
		// The one semicolon sits before the colon.
		return Sentence{}, errors.New("the semicolon comes before the colon; the doing clause ends at the colon and the refusing clause follows the semicolon")
	}

	if strings.TrimSpace(does) == "" {
		return Sentence{}, errors.New("the doing clause before the colon is empty")
	}
	if r, _ := utf8.DecodeRuneInString(does); !unicode.IsUpper(r) {
		return Sentence{}, errors.New("the doing clause does not open with a capital letter")
	}
	if !strings.HasPrefix(writes, " ") || !opensWith(writes[1:], writesOpening) {
		return Sentence{}, fmt.Errorf("the writing clause after the colon does not open with %q", writesOpening)
	}
	if !strings.HasPrefix(refuses, " ") ||
		!(opensWith(refuses[1:], refusesOpening) || opensWith(refuses[1:], neverRefusesOpening)) {
		return Sentence{}, fmt.Errorf("the refusing clause after the semicolon does not open with %q or %q",
			refusesOpening, neverRefusesOpening)
	}
	return Sentence{Does: does, Writes: writes[1:], Refuses: refuses[1:]}, nil
}

// opensWith reports whether clause opens with the word w, as a whole word: w is
// the whole clause or is followed by a space or a comma.
func opensWith(clause, w string) bool {
	return clause == w || strings.HasPrefix(clause, w+" ") || strings.HasPrefix(clause, w+",")
}

// SentenceChanges names every command whose sentence differs between committed
// and current, one line each in canonical path order: "abcd capture: sentence
// reworded". It is PlacementChanges' twin for the sentence field: Diff never
// reads a sentence, because a rewording changes no invocation, so the byte
// comparison behind the drift test and the release gate's stale-surface refusal
// is the only thing that sees one, and naming the command is what makes either
// actionable. A command present on one side only is an addition or a removal,
// which Diff and the drift test already report, and is not listed.
func SentenceChanges(committed, current Snapshot) []string {
	committed, current = canonical(committed), canonical(current)
	was := make(map[string]string, len(committed.Commands))
	for _, c := range committed.Commands {
		was[c.Path] = c.Sentence
	}
	var out []string
	for _, now := range current.Commands {
		if before, ok := was[now.Path]; ok && before != now.Sentence {
			out = append(out, now.Path+": sentence reworded")
		}
	}
	return out
}
