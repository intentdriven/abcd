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
	// questionItemRe is a list item's marker line at up to three columns: a
	// bullet (`-`, `*`, `+`) or a number (`1.`, `1)`), then blanks and the
	// item's text, or nothing for an item whose text starts on the next line.
	questionItemRe = regexp.MustCompile(`^( {0,3})([-*+]|[0-9]{1,9}[.)])(?:([ \t]+)(.*))?$`)
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
// still asks: one per list item opening at up to three columns, bulleted or
// numbered, in document order, each as its first line's text, under a
// sub-heading of the section as well as directly under it. A settled record
// says so in prose — the minted `_None recorded yet._`, or `_None open; …_`
// naming the decisions that answered them — and prose, a blockquote and an
// indented continuation are not questions. A record with no such section asks
// none. It reads the page as mdrecord.ReadPage reads it, the reading Unsettled
// takes: the section's bounds, its items and their settled markers are what
// the page shows, so a line in a fenced block or an HTML comment neither opens
// an item nor settles one.
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
	return openQuestionsIn(mdrecord.ReadPage(strings.Split(content, "\n")))
}

// openQuestionsIn is OpenQuestions over a page already read. An item runs
// from its marker line over the lines indented to its content column, and,
// with no blank line between, over any indented line; a marker nearer the
// margin than the content column opens the next item.
func openQuestionsIn(page mdrecord.Page) []Question {
	text := page.Text
	start, end, ok := mdrecord.SectionLineRangeNestedIn(text, page.Mask, OpenQuestionsHeadingRe)
	if !ok {
		return nil
	}
	var out []Question
	var item []string // the item's text lines, its first line's marker dropped
	itemLine, content, blank := 0, 0, false
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
		ln := text[i]
		trimmed := strings.TrimSpace(ln)
		if first && trimmed != "" {
			first = false
			if allSettledRe.MatchString(trimmed) {
				return nil
			}
		}
		w := indentCols(ln)
		m := questionItemRe.FindStringSubmatch(ln)
		switch {
		case trimmed == "":
			blank = item != nil
		case m != nil && (item == nil || w < content):
			flush()
			item, itemLine, content, blank = []string{strings.TrimSpace(m[4])}, i+1, itemContent(m), false
		case item != nil && (w >= content || (w > 0 && !blank)):
			if len(item) == 1 && item[0] == "" {
				item[0] = trimmed // an item whose text starts on the line after its marker
			} else {
				item = append(item, trimmed)
			}
			blank = false
		default:
			flush()
		}
	}
	flush()
	return out
}

// itemContent is the content column of a question item's marker line
// (questionItemRe's match): where its text starts, or one past the marker
// when the text is indented five columns or more or there is none.
func itemContent(m []string) int {
	end := len(m[1]) + len(m[2])
	col := end
	for _, c := range m[3] {
		if c == '\t' {
			col += 4 - col%4
		} else {
			col++
		}
	}
	if m[4] == "" || col-end >= 5 {
		return end + 1
	}
	return col
}

// indentCols is the column a line's first non-blank character sits at, a tab
// advancing to the next multiple of four.
func indentCols(ln string) int {
	w := 0
	for _, c := range ln {
		switch c {
		case ' ':
			w++
		case '\t':
			w += 4 - w%4
		default:
			return w
		}
	}
	return w
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
