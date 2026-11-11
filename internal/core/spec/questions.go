package spec

import (
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/mdrecord"
)

// questions.go reads what a record's `## Open Questions` section still asks:
// the count the build's pre-start check refuses on (itd-2609201916151817,
// criterion 1: a READY intent with an open question does not start). An
// intent's section and a spec's are the same section, read by this one reader
// (iss-2611101318042398), and Unsettled exempts a bold "Open:" label only on a
// line this reader counts as a question.

var (
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
	// settledBoldRe is an item's explicit disposition as a bold span that opens
	// with it (`**Resolved — …**`, `**Deferred**`, `**explicitly deferred**`,
	// `**explicit deferral**`), wherever in the item the span sits.
	settledBoldRe = regexp.MustCompile(`(?i)\*\*(resolved|deferred|explicitly deferred|explicit deferral)\b`)
	// settledLabelRe is the disposition as a LABEL (`resolved:`, `RESOLVED:`,
	// `Deferred:`), and a label only where a label is written: opening a line of
	// the item — a nested sub-bullet continuation (`  - Resolved: …`) opens one
	// too — after a closing bold, with or without a colon after it
	// (`**Which surface?** RESOLVED:`, `**Which id?**: Resolved:`), or after a
	// dash (`**Refusal breadth** — resolved:`). Mid-sentence the same word and
	// colon are prose — "once the split is resolved: the old or the new?" is a
	// question — and reading them as a marker let build start past it
	// (iss-2609260932374727).
	settledLabelRe = regexp.MustCompile(`(?i)(^|^[-*][ \t]+|\*\*:?[ \t]*|[—–][ \t]*|[ \t]-[ \t]+)(resolved|deferred)[ \t]*:`)
)

// Question is one question a record's `## Open Questions` section still asks.
type Question struct {
	// Line is the 1-based line its item opens on.
	Line int
	// Text is the item's first line, its list marker dropped.
	Text string
}

// OpenQuestions returns the questions a record's `## Open Questions` section
// still asks: one per top-level list item, bulleted or numbered, in document
// order, each as its first line's text, under a sub-heading of the section as
// well as directly under it. A settled record says so in prose — the minted
// `_None recorded yet._`, or `_None open; …_` naming the decisions that
// answered them — and prose, a blockquote and an indented continuation are not
// questions. A record with no such section asks none. The section's bounds are
// taken from mdrecord.ReadPage, the reading Unsettled takes.
//
// It reads fail-closed, recognising only the two settled markers the record
// uses (the 2026-09-25 decision dcn-2609250000000028):
//
//   - a section whose first line is an italic `_All resolved …_` (or `_All
//     four resolved …_`) is settled whole: every item below it is an answer
//     kept for the reader;
//   - an item explicitly marked resolved or deferred — a bold span opening
//     with the word (`**Resolved — …**`, `**Deferred**`, `**explicitly
//     deferred**`, `**explicit deferral**`) anywhere in the item, continuation
//     lines included, or the word as a label (`resolved:`, `Deferred:`)
//     opening a line of the item (a nested sub-bullet included), after a
//     closing bold (and an optional colon), or after a dash — is not a
//     question. The same word and colon mid-sentence are prose.
//
// Everything else under the heading that is a list item is a question
// whatever it says: an item led `**Open`, an item that only points elsewhere,
// and a question that merely mentions deferral all count. The remedy is the
// record's own convention — the answer moves to `## Decisions` and the item or
// the section says it is settled.
func OpenQuestions(content string) []Question {
	lines := strings.Split(content, "\n")
	return openQuestionsIn(lines, mdrecord.ReadPage(lines).Mask)
}

// openQuestionsIn is OpenQuestions over a page mask the caller already read.
func openQuestionsIn(lines []string, mask []uint8) []Question {
	start, end, ok := mdrecord.SectionLineRangeNestedIn(lines, mask, OpenQuestionsHeadingRe)
	if !ok {
		return nil
	}
	var out []Question
	var item []string // the current item's first line, then its continuation lines
	itemLine := 0
	flush := func() {
		if item == nil {
			return
		}
		if !settledItem(item) {
			out = append(out, Question{Line: itemLine, Text: item[0]})
		}
		item = nil
	}
	first := true
	for i := start; i < end; i++ {
		ln := strings.TrimRight(lines[i], "\r")
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
			item, itemLine = []string{strings.TrimSpace(mdrecord.TrimBulletPrefix(ln))}, i+1
		case numberedItemRe.MatchString(ln):
			flush()
			item, itemLine = []string{strings.TrimSpace(numberedItemRe.FindStringSubmatch(ln)[1])}, i+1
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

// settledItem reports whether an item — its first line's text, then its
// continuation lines, each trimmed — carries an explicit resolved or deferred
// marker and is not led "Open". The lines are judged apart for the label, whose
// place is the start of a line, and joined for the bold span, which may wrap.
func settledItem(lines []string) bool {
	if openLeadRe.MatchString(lines[0]) {
		return false
	}
	if settledBoldRe.MatchString(strings.Join(lines, " ")) {
		return true
	}
	for _, ln := range lines {
		if settledLabelRe.MatchString(ln) {
			return true
		}
	}
	return false
}
