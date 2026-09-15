package cli

// decide.go is the front door onto internal/core/decide — the mint that files a
// decision record (the 2026-09-01 ruling in `.abcd/work/DECISIONS.md`).
//
// The verb is deliberately SMALL, for the reason ideate's is: everything an ADR
// is actually made of — the context, the decision, the alternatives, the
// consequences — is a human's to write, and `commands/decide.md` says so. The
// binary's whole job is the part a human does badly across branches: allocating
// an id that cannot collide, and laying the skeleton in the right place.
//
// Exit codes follow the house shape: 0 when the record landed, 2 for every
// operand refusal (no title, a title with nothing slug-able in it, an unwritable
// store). There is no exit 1 — a mint either lands or it does not.

import (
	"fmt"
	"io"
	"os"

	"github.com/intentdriven/abcd/internal/core/decide"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// newDecideCommand builds the `decide` verb. It takes one operand — the
// decision's title — and registers no sub-commands: there is no read-only board
// to render here, because `abcd <adr-id>` already answers "what is this
// decision" and the site's decisions index already answers "what has been
// decided".
func newDecideCommand(asJSON *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   `decide "<title>"`,
		Short: "Mint a decision record (ADR) and lay its skeleton",
		Long: "Mint an architecture decision record: allocate its id through the shared record-id\n" +
			"seam and write the store's skeleton under .abcd/development/decisions/adrs/.\n\n" +
			"The id is `adr-<yymmddHHMMSS><rrrr>` and the filename is ordered by that stamp, so two\n" +
			"branches deciding on the same day cannot allocate the same number — the collision a\n" +
			"hand-numbered ordinal has by construction. The hand-numbered records 0001-0058 keep\n" +
			"their ids and their filenames; nothing is renumbered, and every reader admits both.\n\n" +
			"The verb writes an EMPTY record: it owns the id, the date, the filename and the four\n" +
			"sections, and states nothing. The decision is the author's to write, and the status it\n" +
			"lands with is `proposed` until the author sets `accepted`.",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &exitError{Code: 2, Msg: `decide: one quoted title is required — abcd decide "<title>"`}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := decideStoreRoot(cmd)
			if err != nil {
				return err
			}
			d, err := decide.Create(repoRoot, args[0])
			if err != nil {
				return &exitError{Code: 2, Msg: "decide: " + scrubPaths(err)}
			}
			return render(cmd.OutOrStdout(), *asJSON, d, func(w io.Writer) {
				fmt.Fprint(w, renderDecision(d))
			})
		},
	}
	return cmd
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
func decideStoreRoot(cmd *cobra.Command) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root, err := gitutil.CheckoutRoot(cwd, "the decision store")
	if err != nil {
		return "", &exitError{Code: 2, Msg: "abcd decide: " + err.Error() + " (nothing written)"}
	}
	for _, note := range strayStoreNotes(cwd, root, decide.ADRsRelDir, "decision store") {
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
