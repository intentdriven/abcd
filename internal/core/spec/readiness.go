package spec

// readiness.go reads what a spec's own text says against its being built
// (iss-2611101318042398): the readiness gate and record-lint's
// spec_unsettled rule both ask Unsettled, so the two never disagree about
// what a spec states. A spec says it is not ready in one place, its
// `## Open Questions` section, which the readiness gate reads as it reads an
// intent's; Unsettled names the two other ways a spec has said it, which no
// reader saw: a sentence opening "Not READY", and a bold "Open:" label outside
// that section.
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
// spec alike.
var OpenQuestionsHeadingRe = regexp.MustCompile(`^#{1,6}\s+Open Questions\s*$`)

// The kinds of statement Unsettled reports.
const (
	// UnsettledNotReady is a sentence that opens "Not READY" (or "NOT READY").
	UnsettledNotReady = "not_ready"
	// UnsettledOpenLabel is a bold "Open:" label (`**Open:**`, `**Open**:`,
	// `**Open question:**`) outside the `## Open Questions` section.
	UnsettledOpenLabel = "open_label"
)

// Statement is one place a spec's text says it is not ready to build.
type Statement struct {
	// Line is the 1-based line it is on.
	Line int
	// Kind is UnsettledNotReady or UnsettledOpenLabel.
	Kind string
	// Text is the line, sanitised for a terminal, its whitespace collapsed and
	// clipped to maxStatementRunes.
	Text string
}

// maxStatementRunes caps the text a statement quotes.
const maxStatementRunes = 160

var (
	// lineLeadRe is what may open a line before its first word: a blockquote,
	// a list marker (bulleted or numbered) and emphasis.
	lineLeadRe = regexp.MustCompile(`^[ \t>]*(?:(?:[-*+]|[0-9]{1,4}[.)])[ \t]+)?[*_]*`)
	// notReadyLeadRe is "Not READY" opening the line's text.
	notReadyLeadRe = regexp.MustCompile(`^(?:Not|NOT) READY\b`)
	// notReadyAfterRe is "Not READY" opening a sentence inside the line.
	notReadyAfterRe = regexp.MustCompile(`[.!?][*_)]*[ \t]+[*_(]*(?:Not|NOT) READY\b`)
	// openLabelRe is a bold "Open" label: the word, optionally "question(s)",
	// then a colon inside or straight after the bold. A bold span that merely
	// opens with the word ("**Open pull requests**", "**Open, and not gating**")
	// is not one.
	openLabelRe = regexp.MustCompile(`(?i)\*\*open(?: questions?)?(?::|\*\*[ \t]*:)`)
)

// Unsettled returns every statement in content, a spec's whole text, that
// says the spec is not ready to build, in document order:
//
//   - a sentence that opens "Not READY" or "NOT READY" — at the start of a
//     line's text (after a blockquote, a list marker or emphasis) or after a
//     sentence's closing ".", "!" or "?" — anywhere in the body;
//   - a bold "Open:" label outside the `## Open Questions` section, where no
//     reader of open questions looks.
//
// The frontmatter, fenced blocks, HTML comments and inline code spans are not
// read: a spec quoting the phrase in code says nothing about itself. A line
// carrying both is reported once, as not ready.
func Unsettled(content string) []Statement {
	lines := strings.Split(content, "\n")
	mask := mdrecord.Mask(lines)
	body := 0
	if c := frontmatter.Close(lines); c >= 0 {
		body = c + 1
	}
	qStart, qEnd, hasQ := mdrecord.SectionLineRangeIn(lines, mask, OpenQuestionsHeadingRe)
	var out []Statement
	for i := body; i < len(lines); i++ {
		if i < len(mask) && mask[i] != 0 {
			continue
		}
		ln := blankCodeSpans(strings.TrimRight(lines[i], "\r"))
		kind := ""
		switch {
		case notReadyLeadRe.MatchString(ln[len(lineLeadRe.FindString(ln)):]), notReadyAfterRe.MatchString(ln):
			kind = UnsettledNotReady
		case openLabelRe.MatchString(ln) && !(hasQ && i >= qStart && i < qEnd):
			kind = UnsettledOpenLabel
		}
		if kind != "" {
			out = append(out, Statement{Line: i + 1, Kind: kind, Text: statementText(lines[i])})
		}
	}
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
	if s.Kind == UnsettledOpenLabel {
		what = "a bold \"Open:\" label outside ## Open Questions"
	}
	return "line " + strconv.Itoa(s.Line) + ": " + what + " (" + s.Text + ")"
}

// UnsettledRemedy is the one remedy for a spec Unsettled reports on, which
// the readiness gate and record-lint both name.
const UnsettledRemedy = "ask the spec's open questions as items under its `## Open Questions` section, where the build reads them, and mark each `resolved:` once it is answered; drop the \"Not READY\" sentence and the bold \"Open:\" label once the question is there"
