package cli

import (
	"fmt"
	"io"

	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/board"
	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/report"
	"github.com/intentdriven/abcd/internal/core/reviews"
	"github.com/intentdriven/abcd/internal/core/statusblock"
	"github.com/intentdriven/abcd/internal/core/statusline"
	"github.com/intentdriven/abcd/internal/fsutil"
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
	// View names the view asked for (spc-2610031844142274): product-thinker
	// or facilitator. --json carries every field whichever view it names.
	View string `json:"view"`
	// Version is the installed version, the one `abcd --version` reports
	// (spc-2610100613109045, decision 4): read from core.NewVersion, so it
	// reads no network and runs no subprocess (adr-38).
	Version string `json:"version"`
	// Statusline is present in a managed checkout and omitted — not null —
	// everywhere else, the collection convention of every --json envelope.
	Statusline *boardStatusline `json:"statusline,omitempty"`
	// Inbox is the count of reports waiting in the inbox (itd-2609221656361680),
	// omitted when none wait.
	Inbox *report.Tally `json:"inbox,omitempty"`
	// Peers is present only when a live peer holds a record that differs here
	// (itd-2609091416295622); omitted, not null, otherwise.
	Peers *boardPeersLine `json:"peers,omitempty"`
	// Worktrees counts the worktrees this repository's lane in the worktree
	// store holds and the ones it could reclaim (spc-2609301811532881),
	// from the peers scan above; omitted while the lane holds none. It is
	// the board's alone: the status block the site renders does not carry it.
	Worktrees *boardWorktreesLine `json:"worktrees,omitempty"`
	// Oracle is the model-tier routing, one row per agent, present only once a
	// routing table is accepted (itd-2609170822093401); omitted otherwise.
	Oracle []oracle.BoardRow `json:"oracle,omitempty"`
	// Reviews is the staleness of every review folder under
	// .abcd/work/reviews/, stalest first, with the threshold past which a row
	// is flagged (itd-28); omitted when the tree holds none. The text render
	// lists the dated reviews and folds the release receipts into one line.
	Reviews *reviews.Board `json:"reviews,omitempty"`
	// Facilitator is facilitator.mode as the merge gate resolves it (the
	// default branch's committed configuration beside the machine layer), with
	// the layer and origin that set it, or the read's refusal as Error
	// (spc-2610161609350616); present in a managed checkout only.
	Facilitator *boardFacilitator `json:"facilitator,omitempty"`
	// Status is the Now / Next / Later block (itd-2609212103568351), present
	// in a checkout abcd manages and omitted elsewhere.
	Status *statusblock.Block `json:"status,omitempty"`
}

// boardStatusline is the board's view of the row: the state the badge
// reports, the plain form the text render prints, and the elements for a
// consumer that wants them apart.
type boardStatusline struct {
	State    statusline.State     `json:"state"`
	Plain    string               `json:"plain"`
	Elements []statusline.Element `json:"elements"`
}

// boardFacilitator is the board's facilitator.mode in --json: the resolved
// reading, or the refusal alone (key and error, no mode, layer or origin), so
// a consumer testing for a mode finds none on a refused read.
type boardFacilitator struct {
	Key         string `json:"key"`
	Mode        string `json:"mode,omitempty"`
	Layer       string `json:"layer,omitempty"`
	Origin      string `json:"origin,omitempty"`
	HomeRefused bool   `json:"home_refused,omitempty"`
	Error       string `json:"error,omitempty"`
}

// readBoardFacilitator reads facilitator.mode for a managed checkout, nil
// elsewhere, through the merge gate's own resolver
// (landing.ReadDefaultFacilitator): the default branch's committed
// .abcd/config.json beside the machine layer, never the checkout's working
// tree, so the board says what the gate does. A repository with no default
// branch yet is drawn as unreadable, with the reason. The machine layer's
// notes are not printed here: a refused home's is the note layered.RootsFor
// writes for cwd, which the oracle member (boardOracle) already prints on
// every board run. A refusal is masked with the home redacted, since it can
// carry an absolute path and a key a configuration file wrote.
func readBoardFacilitator(cwd string) *boardFacilitator {
	root, err := gitutil.CheckoutRoot(cwd, statuslineStore)
	if err != nil || !ahoy.Managed(root) {
		return nil
	}
	f, err := landing.ReadDefaultFacilitator(cwd)
	if err != nil {
		return &boardFacilitator{Key: landing.KeyFacilitatorMode, Error: termsafe.Sanitize(fsutil.RedactHome(err.Error()))}
	}
	return &boardFacilitator{Key: f.Key, Mode: f.Mode, Layer: f.Layer, Origin: f.Origin, HomeRefused: f.HomeRefused}
}

// merges is the reading as the board draws it; nil draws no line.
func (b *boardFacilitator) merges() *board.Merges {
	if b == nil {
		return nil
	}
	return &board.Merges{Mode: b.Mode, Layer: b.Layer, Origin: b.Origin, Refused: b.Error, HomeRefused: b.HomeRefused}
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
