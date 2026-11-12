package spec

// readiness.go reads what a spec's own text says against its being built
// (iss-2611101318042398): the readiness gate and record-lint's
// spec_unsettled rule both ask Unsettled, so the two never disagree about
// what a spec states. A spec says it is not ready in one place, its
// `## Open Questions` section, which the readiness gate reads as it reads an
// intent's; Unsettled names the other ways a spec has said it, which no reader
// saw: a sentence opening "Not READY", a bold "Open:" label on any line but
// the one an open question's item opens on, a fence or comment left open,
// which hides everything below it from both readers, and fences or comments
// whose extent turns on the layout around them.
//
// The spec is text written by agents and people, so it is read as untrusted
// input: the caller's read is bounded, every pattern is RE2 (linear in the
// line, no backtracking), and the text a finding quotes is sanitised and
// clipped before any surface renders it.

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/mdrecord"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// OpenQuestionsHeadingRe matches the `## Open Questions` heading at any depth:
// the one section a record asks its open questions in, for an intent and a
// spec alike. It matches the heading as a page shows it: indented up to three
// columns, in any letter case, with or without a closing run of `#`.
var OpenQuestionsHeadingRe = regexp.MustCompile(`(?i)^ {0,3}#{1,6}[ \t]+Open[ \t]+Questions(?:[ \t]+#+)?[ \t]*$`)

// The kinds of statement Unsettled reports.
const (
	// UnsettledNotReady is a sentence that opens "Not READY" (or "NOT READY").
	UnsettledNotReady = "not_ready"
	// UnsettledOpenLabel is a bold "Open:" label (`**Open:**`, `**Open**:`,
	// `**Open question:**`, `__Open:__`, …) on any line but the one an open
	// question's item opens on.
	UnsettledOpenLabel = "open_label"
	// UnsettledUnclosed is a fenced block or an HTML comment that nothing
	// closes: every line below its opener is hidden from the readers, the
	// `## Open Questions` heading included, so the spec is read as saying
	// nothing it does not say.
	UnsettledUnclosed = "unclosed"
	// UnsettledLayout is the first line from which the spec's fences and
	// comments can be read more than one way (mdrecord.ReadPage's Unsure):
	// that line and every line below it are read as their own text.
	UnsettledLayout = "layout"
)

// Statement is one place a spec's text says it is not ready to build.
type Statement struct {
	// Line is the 1-based line it is on.
	Line int
	// Kind is UnsettledNotReady, UnsettledOpenLabel, UnsettledUnclosed or
	// UnsettledLayout.
	Kind string
	// Text is the line, sanitised for a terminal, its whitespace collapsed and
	// clipped to maxStatementRunes.
	Text string
}

// maxStatementRunes caps the text a statement quotes.
const maxStatementRunes = 160

var (
	// lineLeadRe is what may open a line before its first word: a blockquote,
	// a heading's marker, or a list marker (bulleted or numbered) with an
	// optional task box, then emphasis and opening brackets.
	lineLeadRe = regexp.MustCompile(`^[ \t>]*(?:#{1,6}[ \t]+|(?:[-*+]|[0-9]{1,4}[.)])[ \t]+(?:\[[ xX]\][ \t]+)?)?[*_(\[]*`)
	// notReadyLeadRe is "Not READY" opening the line's text. READY is matched
	// in capitals alone, "Not" capitalised or in capitals.
	notReadyLeadRe = regexp.MustCompile(`^(?:Not|NOT)[ \t]+READY(?:[^A-Za-z0-9]|$)`)
	// notReadyAfterRe is "Not READY" opening a sentence, a clause or a table
	// cell inside the line: after a sentence's closing ".", "!" or "?", after a
	// colon ("Status: Not READY") or a semicolon, after a dash with a space on
	// each side ("Draft — Not READY"), after an opening bracket ("(Not READY:
	// …)"), or at a table cell's start ("| Not READY |").
	notReadyAfterRe = regexp.MustCompile(`(?:[.!?:;][*_)\]]*[ \t]+|[ \t][-–—][ \t]+|\|[ \t]*|[(\[])[*_(\[]*(?:Not|NOT)[ \t]+READY(?:[^A-Za-z0-9]|$)`)
	// readyHeadRe is a line whose text opens with READY, the second half of a
	// "Not READY" a hard wrap split.
	readyHeadRe = regexp.MustCompile(`^[ \t>]*[*_]*READY(?:[^A-Za-z0-9]|$)`)
	// openLabelRe is a bold "Open" label, in `**` or `__`: the word, optionally
	// "question(s)", "point(s)" or "design question(s)", then a colon inside or
	// straight after the bold. A bold span that merely opens with the word
	// ("**Open pull requests**", "**Open, and not gating**") is not one.
	openLabelRe = regexp.MustCompile(`(?i)(?:\*\*|__)open(?:[ \t]+(?:questions?|points?|design[ \t]+questions?))?(?::|(?:\*\*|__)[ \t]*:)`)
)

// Unsettled returns every statement in content, a spec's whole text, that
// says the spec is not ready to build, in document order:
//
//   - a sentence that opens "Not READY" or "NOT READY" anywhere in the body:
//     at the start of a line's text (after a blockquote, a heading's `#`
//     marker, a list marker and its task box `[ ]`, emphasis or an opening
//     bracket), after a sentence's closing ".", "!" or "?", after a colon or a
//     semicolon, after a dash with a space on each side, after an opening
//     bracket, at a table cell's start, and also when a hard wrap puts "Not"
//     at the end of one line and "READY" at the start of the next (reported on
//     the first). "Not" is matched capitalised or in capitals and READY in
//     capitals alone, so "not READY" and "Not ready" are not reported;
//   - a bold "Open:" label, in any letter case, on any line but the one an
//     open question's item opens on, the other lines of `## Open Questions`
//     included: `**Open:**`, `**Open**:`, `__Open:__`, and the same with
//     "question(s)", "point(s)" or "design question(s)" after the word;
//   - a fenced block or an HTML comment nothing closes, on its opener's line,
//     because everything below it is hidden from this reader and from the
//     one that reads the section's questions;
//   - the first line from which the fences and comments can be read more than
//     one way, from where every line is read as its own text
//     (LayoutStatements).
//
// It reads what a person reading the page reads, through mdrecord.ReadPage,
// the reading OpenQuestions takes. The frontmatter, inline code spans that open and close on one line, fenced
// blocks and HTML comments are not read, a comment that opens and closes on
// one line included; the text before a comment's opener and after its closer,
// on the same line, is, and so is an indented code block. A line carrying a
// sentence and a label is reported once, as not ready.
func Unsettled(content string) []Statement {
	lines := strings.Split(content, "\n")
	body := 0
	if c := frontmatter.Close(lines); c >= 0 {
		body = c + 1
	}
	var out []Statement
	page := mdrecord.ReadPage(lines)
	// A label is exempt only on the line a counted question opens on.
	asked := map[int]bool{}
	for _, q := range openQuestionsIn(page) {
		asked[q.Line] = true
	}
	live := page.Text
	for i := body; i < len(lines); i++ {
		ln := blankCodeSpans(live[i])
		kind := ""
		switch {
		case notReadyAt(ln):
			kind = UnsettledNotReady
		case notReadyAt(ln+" READY") && readyHeadRe.MatchString(nextLine(live, i)):
			kind = UnsettledNotReady
		case openLabelRe.MatchString(ln) && !asked[i+1]:
			kind = UnsettledOpenLabel
		}
		if kind != "" {
			out = append(out, Statement{Line: i + 1, Kind: kind, Text: statementText(lines[i])})
		}
	}
	for _, st := range layoutStatements(page) {
		out = insertStatement(out, st)
	}
	return out
}

// LayoutStatements returns what a record's layout alone says against reading
// it whole, in line order: an opener nothing closes, on its own line, and the
// first line from which its fences and comments can be read more than one way
// (mdrecord.ReadPage).
// Unsettled reports both for a spec, and an intent's open-questions row
// withholds readiness on both, since each reads the record through the same
// reading.
func LayoutStatements(content string) []Statement {
	return layoutStatements(mdrecord.ReadPage(strings.Split(content, "\n")))
}

func layoutStatements(p mdrecord.Page) []Statement {
	var out []Statement
	line, flag, open := p.Unclosed()
	if open {
		out = append(out, Statement{Line: line + 1, Kind: UnsettledUnclosed, Text: unclosedText(flag)})
	}
	if l, ok := p.Unsure(); ok {
		out = insertStatement(out, Statement{Line: l + 1, Kind: UnsettledLayout, Text: "where a fence or a comment ends from here turns on the layout around it"})
	}
	return out
}

// notReadyAt reports whether a line's live text holds a sentence opening
// "Not READY".
func notReadyAt(ln string) bool {
	return notReadyLeadRe.MatchString(ln[len(lineLeadRe.FindString(ln)):]) || notReadyAfterRe.MatchString(ln)
}

// nextLine is the live text of the line after i, blanked of code spans, or
// "" past the end: a blank line ends the paragraph, so it carries no wrap.
func nextLine(live []string, i int) string {
	if i+1 >= len(live) {
		return ""
	}
	return blankCodeSpans(live[i+1])
}

// unclosedText is what an unclosed statement quotes: the construct, not the
// line, whose text may be anything.
func unclosedText(flag uint8) string {
	if flag == mdrecord.MaskComment {
		return "an HTML comment opened here is never closed"
	}
	return "a fenced block opened here is never closed"
}

// insertStatement adds st to out in line order.
func insertStatement(out []Statement, st Statement) []Statement {
	k := len(out)
	for k > 0 && out[k-1].Line > st.Line {
		k--
	}
	out = append(out, Statement{})
	copy(out[k+1:], out[k:])
	out[k] = st
	return out
}

// blankCodeSpans replaces each inline code span on ln with spaces of the same
// length, so a pattern cannot match inside one and the offsets stay put.
func blankCodeSpans(ln string) string {
	ranges := mdrecord.CodeSpanRanges(ln)
	if len(ranges) == 0 {
		return ln
	}
	b := []byte(ln)
	for _, r := range ranges {
		for j := r[0]; j < r[1] && j < len(b); j++ {
			b[j] = ' '
		}
	}
	return string(b)
}

// statementText is a line as a statement quotes it: sanitised, its whitespace
// collapsed, clipped.
func statementText(ln string) string {
	t := strings.Join(strings.Fields(termsafe.Sanitize(ln)), " ")
	if r := []rune(t); len(r) > maxStatementRunes {
		t = string(r[:maxStatementRunes]) + "…"
	}
	return t
}

// Describe renders a statement for a refusal or a finding: its line, what it
// is, and the text it quotes.
func (s Statement) Describe() string {
	what := "a sentence opens \"Not READY\""
	switch s.Kind {
	case UnsettledOpenLabel:
		what = "a bold \"Open:\" label on a line that does not open an open question"
	case UnsettledUnclosed:
		what = "nothing below this line is read"
	case UnsettledLayout:
		what = "the text from this line on is not read one way, so every line from here is read as text"
	}
	return "line " + strconv.Itoa(s.Line) + ": " + what + " (" + s.Text + ")"
}

// UnsettledRemedy is the one remedy for a spec Unsettled reports on, which
// the readiness gate and record-lint both name.
const UnsettledRemedy = "ask the spec's open questions as items under its `## Open Questions` section, where the build reads them, and mark each `resolved:` once it is answered; drop the \"Not READY\" sentence and the bold \"Open:\" label once the question is there, or put the phrase in a code span when it is quoted, not said; close a fence or an HTML comment left open; write a fence at column 0, or at up to three columns with no line inside it nearer the margin than the fence, and not on a list marker's line; and open a comment that runs over several lines at the start of its line, at up to three columns"
