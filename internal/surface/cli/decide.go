package cli

// decide.go is the front door onto the two decision stores
// (spc-2610110931381316, decision 17). `abcd decide "<text>"` records a decision
// through internal/core/decisions, one file in `.abcd/work/decisions/`; bare
// `abcd decide` lists those records newest first; and `abcd decide --adr
// "<title>"` mints an architecture decision record through internal/core/decide
// (the 2026-09-01 ruling dcn-2609010000000007). Decisions outnumber
// ADRs about eight to one, so the common act takes the short form.
//
// The verb is deliberately SMALL, for the reason ideate's is: everything an ADR
// is actually made of — the context, the decision, the alternatives, the
// consequences — is a human's to write, and `commands/decide.md` says so. The
// binary's whole job is the part a human does badly across branches: allocating
// an id that cannot collide, and laying the file in the right place.
//
// Exit codes follow the house shape: 0 when the record landed or the listing
// rendered, 2 for every operand refusal (a missing title for --adr, a text with
// nothing slug-able in it, a repository without the shared tier, no checkout),
// and 1 when the folder could not be read.

import (
	"fmt"
	"io"
	"os"

	"github.com/intentdriven/abcd/internal/core/decide"
	"github.com/intentdriven/abcd/internal/core/decisions"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// newDecideCommand builds the `decide` verb. It takes at most one operand and
// registers no sub-commands: with the operand it records a decision, with
// --adr it mints an ADR from the operand as the title, and with neither it
// lists the decisions folder, which is the verb's bare render.
func newDecideCommand(asJSON *bool) *cobra.Command {
	var adr bool
	cmd := &cobra.Command{
		Use: `decide ["<text>" | --adr "<title>"]`,
		Long: "Record a decision, list the decisions recorded, or mint an architecture decision record.\n\n" +
			"abcd decide \"<text>\" records a decision: one new file, .abcd/work/decisions/dcn-<id>-<slug>.md,\n" +
			"whose id is minted through the shared record-id seam and whose body is `<date> — <text>`.\n" +
			"Nothing is appended anywhere, so two branches that each record a decision merge with nothing\n" +
			"to reconcile. It needs the shared tier .abcd/work/, which `abcd ahoy install` lays.\n\n" +
			"abcd decide with no operand lists every decision recorded, newest first, read from the folder\n" +
			"each time: the id, then the decision. Where a frozen .abcd/work/DECISIONS.md also exists, the\n" +
			"listing ends by naming it as earlier history it does not list. It writes nothing.\n\n" +
			"abcd decide --adr \"<title>\" mints an architecture decision record: its id is\n" +
			"`adr-<yymmddHHMMSS><rrrr>` and the store's skeleton lands under\n" +
			".abcd/development/decisions/adrs/, with the filename ordered by the stamp, so two branches\n" +
			"deciding on the same day cannot allocate the same number. The hand-numbered records\n" +
			"0001-0058 keep their ids and their filenames. The ADR is EMPTY: the verb owns the id, the\n" +
			"date, the filename and the four sections, and states nothing; it lands `proposed` until the\n" +
			"author sets `accepted`.",
		Args: func(_ *cobra.Command, args []string) error {
			switch {
			case adr && len(args) != 1:
				return &exitError{Code: 2, Msg: `decide --adr: one quoted title is required — abcd decide --adr "<title>"`}
			case len(args) > 1:
				return &exitError{Code: 2, Msg: `decide: one quoted operand at most — abcd decide "<text>" records a decision, abcd decide lists them`}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if adr {
				return mintADR(cmd, *asJSON, args[0])
			}
			if len(args) == 0 {
				return listDecisions(cmd, *asJSON)
			}
			return recordDecision(cmd, *asJSON, args[0])
		},
	}
	cmd.Flags().BoolVar(&adr, "adr", false, "mint an architecture decision record from the operand as its title, instead of recording a decision")
	return cmd
}

// mintADR is `abcd decide --adr "<title>"`: the ADR mint, unchanged from the
// bare form it had before decision 17.
func mintADR(cmd *cobra.Command, asJSON bool, title string) error {
	repoRoot, err := decideStoreRoot(cmd, decide.ADRsRelDir, "decision store")
	if err != nil {
		return err
	}
	d, err := decide.Create(repoRoot, title)
	if err != nil {
		return &exitError{Code: 2, Msg: "decide: " + scrubPaths(err)}
	}
	return render(cmd.OutOrStdout(), asJSON, d, func(w io.Writer) {
		fmt.Fprint(w, renderDecision(d))
	})
}

// recordDecision is `abcd decide "<text>"`: one decision record, written by
// the folder's one writer.
func recordDecision(cmd *cobra.Command, asJSON bool, text string) error {
	repoRoot, err := decideStoreRoot(cmd, decisions.RelDir, "decisions folder")
	if err != nil {
		return err
	}
	d, err := decisions.Record(repoRoot, text)
	if err != nil {
		return &exitError{Code: 2, Msg: "decide: " + scrubPaths(err)}
	}
	return render(cmd.OutOrStdout(), asJSON, d, func(w io.Writer) {
		fmt.Fprint(w, renderRecorded(d))
	})
}

// listDecisions is bare `abcd decide`: the folder, newest first. A file the
// reader skipped is named on stderr, so a malformed record is never dropped in
// silence.
func listDecisions(cmd *cobra.Command, asJSON bool) error {
	repoRoot, err := decideStoreRoot(cmd, decisions.RelDir, "decisions folder")
	if err != nil {
		return err
	}
	l, err := decisions.ReadListing(repoRoot)
	if err != nil {
		return &exitError{Code: 1, Msg: "decide: " + scrubPaths(err)}
	}
	for _, s := range l.Skipped {
		fmt.Fprintf(cmd.ErrOrStderr(), "abcd decide: skipped %s: %s\n", termsafe.Sanitize(s.Path), termsafe.Sanitize(s.Reason))
	}
	return render(cmd.OutOrStdout(), asJSON, l.Entries(), func(w io.Writer) {
		fmt.Fprint(w, termsafe.SanitizeBlock(l.Text()))
	})
}

// renderRecorded is the human view of a recorded decision.
func renderRecorded(d decisions.Decision) string {
	out := fmt.Sprintf("decision recorded — %s\n", d.ID)
	out += fmt.Sprintf("  date:   %s\n", d.Date)
	out += fmt.Sprintf("  record: %s\n", d.Path)
	out += fmt.Sprintf("  %s\n", termsafe.Sanitize(d.Text))
	return out
}

// decideStoreRoot is the front door's first step: the checkout whose decision
// store the mint writes into, resolved from the working directory rather than
// taken to BE it.
//
// The verb used to hand os.Getwd() straight to decide.Create, which joins the
// store's relative directory onto whatever it is given and creates the tree. Run
// from a subdirectory that laid a SECOND store beneath it; run outside every
// repository it exited 0 and laid a complete ADR store in a plain directory,
// reporting a repo-relative path that reads exactly like the checkout store's
// (iss-2609091707224329). A decision filed either way reaches no gate, no
// release cut and no reader, and the durable record is the family where that
// costs most: the decision the record holds was never written anywhere else.
//
// gitutil.CheckoutRoot owns the resolution and both refusals — the same one the
// capture verbs resolve their ledger through, with only the store's noun
// differing. Refusing is the whole point outside a checkout: there is no
// decision store to address, and laying one where the caller stood is the defect
// rather than a lenient fallback. Nothing is written on a refusal, because the
// core is never reached.
//
// The stray-store note rides the same step, on stderr, exactly as the ledger's
// does: a resolution that silently steps over a store the defect already laid
// would leave those decisions where nothing will ever look again. It REPORTS and
// moves nothing.
func decideStoreRoot(cmd *cobra.Command, storeRel, noun string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root, err := gitutil.CheckoutRoot(cwd, "the "+noun)
	if err != nil {
		return "", &exitError{Code: 2, Msg: "abcd decide: " + err.Error() + " (nothing written)"}
	}
	for _, note := range strayStoreNotes(cwd, root, storeRel, noun) {
		fmt.Fprintf(cmd.ErrOrStderr(), "abcd decide: %s\n", termsafe.Sanitize(note))
	}
	return root, nil
}

// renderDecision is the human view of a minted decision. Every value in it is
// core-owned — a validated slug, a minted id, a date derived from that id, and a
// path this binary built — except the title, which the core already passed
// through the canonical redactor before it reached the file.
func renderDecision(d decide.Decision) string {
	out := fmt.Sprintf("decision record minted — %s\n", d.ID)
	out += fmt.Sprintf("  title:  %s\n", d.Title)
	out += fmt.Sprintf("  date:   %s\n", d.Date)
	out += fmt.Sprintf("  record: %s\n", d.Path)
	out += "  status: proposed — write the four sections, then set `accepted` when the decision is in force\n"
	return out
}
