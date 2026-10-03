package question

import (
	"strings"

	"github.com/intentdriven/abcd/internal/textwidth"
)

// estimateRows is the rows a question takes in the host's frame at l.Columns
// (rule 13): one row for the header, the question text wrapped at
// l.HostTextColumns, each option's label row and its description wrapped
// beneath it, and l.HostChromeRows for the frame. A preview sits beside the
// options and is not counted here; it is held to the rows the frame leaves it.
//
// The estimate wraps greedily, as a terminal fills a line, and its two host
// figures are provisional until step 5 of spc-2610030944505997 calibrates them
// against a dated screenshot.
func estimateRows(t Tab, l Limits) int {
	rows := 1 + blockRows(t.Text, l.HostTextColumns) + l.HostChromeRows
	for _, o := range t.Options {
		rows += max(1, blockRows(o.Label, l.HostTextColumns)) + blockRows(o.Description, l.HostTextColumns)
	}
	return rows
}

// previewBudget is the rows the frame leaves a preview: the window less the
// frame, the header and the question text above the options.
func previewBudget(t Tab, l Limits) int {
	return l.Rows - l.HostChromeRows - 1 - blockRows(t.Text, l.HostTextColumns)
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
