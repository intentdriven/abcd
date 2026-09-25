package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/release"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// newLaunchReceiptsCommand builds `abcd launch receipts` (itd-93 AC7, iss-327):
// the release job's receipt gate, run locally on the release branch before the
// merge. A red result here costs an amend; the same result in the release job
// costs a release run.
//
// It is the release job's reader, not a model of it (lint.CheckReleaseReceipts):
// the content commit is derived from the receipts directory the way the job
// derives it, the required gates are read from the committed release workflow
// the job runs, and the verdict is the job's own check. So the two cannot
// disagree about a repository state, and a test holds them to that.
//
// Exit codes:
//
//   - 0 — the release job's receipt gate admits this state, or the release
//     workflow arms no receipt gate (nothing is required).
//   - 1 — it refuses; the report names each missing or non-PROMOTE receipt and
//     the commit it must name.
//   - 2 — a structural fault (the repository or its workflow could not be read).
func newLaunchReceiptsCommand(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "receipts",
		Short: "Run the release job's semantic-receipt gate locally, before the merge (exit 1 when it would refuse)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			check, err := lint.CheckReleaseReceipts(cwd)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd launch receipts: " + scrubPaths(err)}
			}
			if rerr := render(cmd.OutOrStdout(), *asJSON, check, func(w io.Writer) {
				renderReceiptCheck(w, check)
			}); rerr != nil {
				return rerr
			}
			if !check.Pass {
				return &exitError{Code: 1}
			}
			return nil
		},
	}
}

// renderReceiptCheck prints the local verdict: the gate list and where it came
// from, the commit the receipts must name, and every refusal. Receipt contents
// and paths come from the working tree, so every one is sanitised.
func renderReceiptCheck(w io.Writer, c lint.ReceiptCheck) {
	verdict := "PASS"
	if !c.Pass {
		verdict = "REFUSED"
	}
	fmt.Fprintf(w, "abcd launch receipts — %s\n", verdict)
	switch {
	case !c.Gate.Present:
		fmt.Fprintf(w, "  %s is absent, so no receipt gate is armed and no receipt is required.\n", c.Gate.Workflow)
		return
	case !c.Gate.Armed:
		fmt.Fprintf(w, "  %s requires no semantic gate: the deterministic gates alone admit the release.\n", c.Gate.Workflow)
		return
	}
	fmt.Fprintf(w, "  required gates: %s (from %s)\n",
		termsafe.Sanitize(strings.Join(c.Gate.Gates, ", ")), c.Gate.Workflow)
	if c.Commit != "" {
		how := "derived from the receipts directory, as the release job derives it"
		if !c.Derived {
			how = "HEAD, the roll: no receipts directory names a commit on this branch yet"
		}
		fmt.Fprintf(w, "  receipts must name: %s (%s)\n", c.Commit, how)
	}
	if c.DeriveError != "" {
		fmt.Fprintf(w, "  release job's derivation refuses: %s\n", termsafe.Sanitize(scrubMessage(c.DeriveError)))
	}
	for _, p := range c.Problems {
		label := p.Gate
		if label == "" {
			label = "gate"
		}
		fmt.Fprintf(w, "  - %s: %s\n", termsafe.Sanitize(label), termsafe.Sanitize(scrubMessage(p.Message)))
	}
	if len(c.Uncommitted) > 0 {
		fmt.Fprintln(w, "  uncommitted receipt changes (the release job reads the committed tree, so commit them first):")
		for _, u := range c.Uncommitted {
			fmt.Fprintf(w, "    %s\n", termsafe.Sanitize(u))
		}
	}
	if c.Pass {
		fmt.Fprintln(w, "  the release job's receipt gate admits this state.")
	}
}

// scrubMessage redacts the two developer-identity roots — the working
// directory and the home directory — out of a message that is text rather than
// an error, the way scrubPaths does for an error.
func scrubMessage(msg string) string {
	if cwd, e := os.Getwd(); e == nil {
		msg = fsutil.RedactRoot(msg, cwd, ".")
	}
	if home, e := os.UserHomeDir(); e == nil {
		msg = fsutil.RedactRoot(msg, home, "~")
	}
	return msg
}

// renderReceiptsProtocol ends the emit step's render with the receipts protocol
// as a numbered checklist (itd-93 AC8). The steps are composed in core, from
// the release workflow's own required-gate list.
func renderReceiptsProtocol(w io.Writer, p release.ReceiptsProtocol) {
	fmt.Fprintln(w, "  receipts protocol (before you merge the release):")
	for i, step := range p.Steps {
		fmt.Fprintf(w, "  %d. %s\n", i+1, termsafe.Sanitize(step))
	}
}
