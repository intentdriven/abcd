package question

import (
	"strings"

	"github.com/intentdriven/abcd/internal/textwidth"
)

// estimateRows is the rows a question takes in the host's frame at l.Columns
// (rule 13): l.HostChromeRows for the frame, the chip's row included, the
// question text wrapped at l.HostTextColumns, and each option's label row and
// its description wrapped beneath it at l.HostOptionColumns. A preview is
// refused (rule 14), so its layout is not estimated.
//
// The estimate wraps greedily, as a terminal fills a line. Its three host
// figures were calibrated on 2026-10-03 against Claude Code's question view at
// 80 by 24 (step 5 of spc-2610030944505997): the estimate of the screenshot's
// question equals the 24 rows the host drew.
func estimateRows(t Tab, l Limits) int { return tabRows(t, l).Total }

// RowSplit is the row estimate by part: the host's frame with the header's
// chip row, the question text, and the options with their descriptions, and
// their sum. The rows finding reports it, so the asker sees which part to cut
// (iss-2610071538055431), and `abcd guard question` reports it for every tab
// (spc-2610301917245442), so the drafter copies the check's figures instead of
// counting by hand.
type RowSplit struct {
	HeaderAndFrame int `json:"header_and_frame"`
	QuestionText   int `json:"question_text"`
	Options        int `json:"options"`
	Total          int `json:"total"`
}

// tabRows is estimateRows by part: the one computation the rows rule and every
// caller that reports a tab's rows read.
func tabRows(t Tab, l Limits) RowSplit {
	s := RowSplit{HeaderAndFrame: l.HostChromeRows, QuestionText: blockRows(t.Text, l.HostTextColumns)}
	for _, o := range t.Options {
		s.Options += max(1, blockRows(o.Label, l.HostOptionColumns)) + blockRows(o.Description, l.HostOptionColumns)
	}
	s.Total = s.HeaderAndFrame + s.QuestionText + s.Options
	return s
}

// blockRows is the rows text takes wrapped at width: each line wrapped on its
// own, a blank line one row, leading and trailing blank lines dropped.
//
// A word wider than width stands alone on its wrapped line (textwidth.Greedy),
// and the host hard-wraps it there, so that line counts the rows it fills,
// ceil(its width / width), not one. The count lives here, not in Greedy:
// standing alone is right for the banner Greedy lays out, and only an
// estimate of the host's frame needs the hard wrap counted.
func blockRows(s string, width int) int {
	s = strings.Trim(s, "\n")
	if strings.TrimSpace(s) == "" {
		return 0
	}
	width = max(width, 1)
	rows := 0
	for _, ln := range strings.Split(s, "\n") {
		wrapped := textwidth.Greedy(strings.Fields(ln), width)
		if len(wrapped) == 0 {
			rows++
			continue
		}
		for _, w := range wrapped {
			rows += max(1, (textwidth.Columns(w)+width-1)/width)
		}
	}
	return rows
}
