package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/board"
	"github.com/intentdriven/abcd/internal/term"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The bare board's two axes (spc-2610031844142274, open question 1): whose view
// is drawn, and in which form.
const (
	viewProduct     = "product"
	viewFacilitator = "facilitator"
	formText        = "text"
	formMarkdown    = "markdown"
)

// boardView reads --view, the product thinker's when it is omitted, always:
// never the stored status's role and never the last one chosen (decision 8).
func boardView(v string) (board.View, error) {
	switch v {
	case "", viewProduct:
		return board.Product, nil
	case viewFacilitator:
		return board.Facilitator, nil
	}
	return 0, &exitError{Code: 2, Msg: fmt.Sprintf("--view takes %s or %s, not %q", viewProduct, viewFacilitator, termsafe.Sanitize(v))}
}

// boardForm reads --format, text when it is omitted.
func boardForm(f string) (board.Form, error) {
	switch f {
	case "", formText:
		return board.Text, nil
	case formMarkdown:
		return board.Markdown, nil
	}
	return 0, &exitError{Code: 2, Msg: fmt.Sprintf("--format takes %s or %s, not %q", formText, formMarkdown, termsafe.Sanitize(f))}
}

// viewName is the view as --json names it: the role whose view it is.
func viewName(v board.View) string {
	if v == board.Facilitator {
		return "facilitator"
	}
	return "product-thinker"
}

// boardRung is how much colour the board is drawn with: none off a Terminal,
// and on one the colour ladder's rung, which NO_COLOR and --no-color bring to
// Mono, in the core's own terms.
func boardRung(out io.Writer, noColor bool) board.Rung {
	if !bannerTTY(out) {
		return board.Mono
	}
	switch term.ResolveColorMode(os.Getenv, noColor) {
	case term.TrueColor:
		return board.TrueColor
	case term.Ansi256:
		return board.Ansi256
	case term.Ansi16:
		return board.Ansi16
	}
	return board.Mono
}

// boardRows are the facilitator's labelled rows, worded here from the readers
// the bare board has always called, in the full board's order.
func boardRows(st core.StatusInfo, b boardOutput) []board.Row {
	rows := []board.Row{
		{Label: "git repo", Text: yesNo(st.IsGitRepo)},
		{Label: "record", Text: yesNo(st.HasRecord)},
		{Label: "work tiers", Text: termsafe.Sanitize(tierList(st.WorkTiers))},
	}
	if b.Statusline != nil {
		rows = append(rows, board.Row{Label: "presence", Text: b.Statusline.Plain})
	}
	if b.Peers != nil {
		rows = append(rows, board.Row{Label: "peers", Text: fmt.Sprintf("%s differing here across %s — abcd peers",
			countOf(b.Peers.IDs, "record"), countOf(b.Peers.Live, "live peer"))})
	}
	if b.Inbox != nil {
		rows = append(rows, board.Row{Label: "inbox", Text: inboxTallyText(*b.Inbox) + " — `abcd inbox`"})
	}
	if r := oracleRow(b.Oracle); r != nil {
		rows = append(rows, *r)
	}
	if r := reviewsRow(b.Reviews); r != nil {
		rows = append(rows, *r)
	}
	return rows
}
