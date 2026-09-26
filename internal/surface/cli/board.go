package cli

import (
	"fmt"
	"io"

	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/report"
	"github.com/intentdriven/abcd/internal/core/reviews"
	"github.com/intentdriven/abcd/internal/core/statusline"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// boardOutput is the bare board's --json envelope: core's StatusInfo, whose
// fields are promoted unchanged, plus the one-line state a managed checkout
// carries (spc-70). It wraps StatusInfo here, the way ahoyOutput wraps its
// result, so package core never imports statusline: the board is the second
// consumer of the render, not a third owner of its words.
type boardOutput struct {
	core.StatusInfo
	// Statusline is present in a managed checkout and omitted — not null —
	// everywhere else, the collection convention of every --json envelope.
	Statusline *boardStatusline `json:"statusline,omitempty"`
	// Inbox is the count of reports waiting in the inbox (itd-2609221656361680),
	// omitted when none wait.
	Inbox *report.Tally `json:"inbox,omitempty"`
	// Peers is present only when a live peer holds a record that differs here
	// (itd-2609091416295622); omitted, not null, otherwise.
	Peers *boardPeersLine `json:"peers,omitempty"`
	// Oracle is the model-tier routing, one row per agent, present only once a
	// routing table is accepted (itd-2609170822093401); omitted otherwise.
	Oracle []oracle.BoardRow `json:"oracle,omitempty"`
	// Reviews is the staleness of every review folder under
	// .abcd/work/reviews/, stalest first, with the threshold past which a row
	// is flagged for a re-run (itd-28); omitted when the tree holds none.
	Reviews *reviews.Board `json:"reviews,omitempty"`
}

// boardStatusline is the board's view of the row: the state the badge
// reports, the plain form the text render prints, and the elements for a
// consumer that wants them apart.
type boardStatusline struct {
	State    statusline.State     `json:"state"`
	Plain    string               `json:"plain"`
	Elements []statusline.Element `json:"elements"`
}

// boardPresence composes the presence line for the checkout containing cwd,
// or nil where there is none to compose: no checkout, or one abcd does not
// manage. It is the board half of ac-7 — the fallback where a host has no
// status surface — so it ignores the setting's off switch: the switch
// silences the status LINE, and the board is the on-demand answer for
// exactly the machines that have none. The board never runs the previous
// command and never writes.
//
// The render's notes and any failure go to stderr under the board's own
// prefix, and the line is omitted rather than invented on a failure.
func boardPresence(cwd string, stderr io.Writer) *boardStatusline {
	root, err := gitutil.CheckoutRoot(cwd, statuslineStore)
	if err != nil || !ahoy.Managed(root) {
		return nil
	}
	set, notes, err := statusline.Load()
	if err != nil {
		fmt.Fprintf(stderr, "abcd: %s; the bundled defaults render\n", termsafe.Sanitize(err.Error()))
		set = statusline.Defaults()
	}
	for _, n := range notes {
		fmt.Fprintf(stderr, "abcd: %s\n", termsafe.Sanitize(n))
	}
	set.Disabled = false
	res, err := statusline.Compose(root, statusline.Payload{}, set)
	if err != nil {
		fmt.Fprintf(stderr, "abcd: the presence line is omitted — %s\n", termsafe.Sanitize(err.Error()))
		return nil
	}
	for _, n := range res.Notes {
		fmt.Fprintf(stderr, "abcd: %s\n", termsafe.Sanitize(n))
	}
	return &boardStatusline{State: res.State, Plain: res.Row.Plain(), Elements: res.Row.Elements}
}
