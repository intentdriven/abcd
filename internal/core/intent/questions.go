package intent

import (
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/mdrecord"
)

// questions.go reads what an intent's `## Open Questions` section still asks —
// the count the build's pre-start check refuses on (itd-2609201916151817,
// criterion 1: a READY intent with an open question does not start).

var (
	// openQuestionsHeadingRe matches the `## Open Questions` heading (any depth).
	openQuestionsHeadingRe = regexp.MustCompile(`^#{1,6}\s+Open Questions\s*$`)
	// numberedItemRe is a column-0 numbered list item, `1. text` or `1) text` —
	// the other spelling of a list a question is written in.
	numberedItemRe = regexp.MustCompile(`^[0-9]{1,4}[.)][ \t]+(\S.*)$`)
	// allSettledRe is the italic line that opens a settled section and says
	// every item below it is settled: `_All resolved …_`, or with one count
	// word, `_All four resolved …_`. "All but one resolved" does not match.
	allSettledRe = regexp.MustCompile(`(?i)^_all(\s+\w+)?\s+resolved\b`)
	// openLeadRe is an item led by a bold "Open": a question, whatever else
	// the item says.
	openLeadRe = regexp.MustCompile(`(?i)^\*\*open\b`)
	// settledMarkRe is an item's explicit disposition: a bold span that opens
	// with it (`**Resolved — …**`, `**Deferred**`, `**explicitly deferred**`,
	// `**explicit deferral**`), or the word as a label (`resolved:`,
	// `RESOLVED:`, `Deferred:`).
	settledMarkRe = regexp.MustCompile(`(?i)\*\*(resolved|deferred|explicitly deferred|explicit deferral)\b|\b(resolved|deferred)\s*:`)
)

// OpenQuestions returns the questions an intent's `## Open Questions` section
// still asks: one per top-level list item, bulleted or numbered, in document
// order, each as its first line's text. A settled record says so in prose — the
// minted `_None recorded yet._`, or `_None open; …_` naming the decisions that
// answered them — and prose, a blockquote and an indented continuation are not
// questions. A record with no such section asks none.
//
// It reads fail-closed, recognising only the two settled markers the record
// uses (the 2026-09-25 DECISIONS entry):
//
//   - a section whose first line is an italic `_All resolved …_` (or `_All
//     four resolved …_`) is settled whole: every item below it is an answer
//     kept for the reader;
//   - an item explicitly marked resolved or deferred — a bold span opening
//     with the word (`**Resolved — …**`, `**Deferred**`, `**explicitly
//     deferred**`, `**explicit deferral**`) or the word as a label
//     (`resolved:`, `Deferred:`) anywhere in the item, continuation lines
//     included — is not a question.
//
// Everything else under the heading that is a list item is a question
// whatever it says: an item led `**Open`, an item that only points elsewhere,
// and a question that merely mentions deferral all count. The remedy is the
// record's own convention — the answer moves to `## Decisions` and the item or
// the section says it is settled.
func OpenQuestions(content string) []string {
	var out []string
	var item []string // the current item's first line, then its continuation lines
	flush := func() {
		if item == nil {
			return
		}
		if !settledItem(strings.Join(item, " ")) {
			out = append(out, item[0])
		}
		item = nil
	}
	first := true
	for _, ln := range strings.Split(sectionBody(content, openQuestionsHeadingRe), "\n") {
		ln = strings.TrimRight(ln, "\r")
		trimmed := strings.TrimSpace(ln)
		if first && trimmed != "" {
			first = false
			if allSettledRe.MatchString(trimmed) {
				return nil
			}
		}
		switch {
		case mdrecord.IsTopLevelBullet(ln):
			flush()
			item = []string{strings.TrimSpace(mdrecord.TrimBulletPrefix(ln))}
		case numberedItemRe.MatchString(ln):
			flush()
			item = []string{strings.TrimSpace(numberedItemRe.FindStringSubmatch(ln)[1])}
		case trimmed == "":
		case ln[0] == ' ' || ln[0] == '\t':
			if item != nil {
				item = append(item, trimmed)
			}
		default:
			flush()
		}
	}
	flush()
	return out
}

// settledItem reports whether an item's text, its continuation lines joined,
// carries an explicit resolved or deferred marker and is not led "Open".
func settledItem(text string) bool {
	return !openLeadRe.MatchString(text) && settledMarkRe.MatchString(text)
}
