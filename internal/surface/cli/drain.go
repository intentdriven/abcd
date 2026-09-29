package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// drainOutput is the --json envelope of `drain --dry-run`: the core's plan,
// marked as a dry run so a reader never takes it for a run's summary.
type drainOutput struct {
	DryRun bool `json:"dry_run"`
	capture.DrainPlan
}

// newDrainCommand builds the `drain` verb (itd-82, spc-2609212015054359): the
// field-only slice. `--dry-run` classifies every open issue by the eligibility
// rule and writes nothing; the run itself is not built, so a bare `drain`
// refuses to start and says why.
func newDrainCommand(asJSON *bool) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use: "drain",
		Long: "Work the open issue ledger unattended: fix the issues that need no decision, and\n" +
			"hand the rest back by kind. The rule for which issues need no decision is a\n" +
			"recorded decision, and it reads the record's fields alone: nothing open in\n" +
			"blocked_by; a category in the fixable set (tech-debt, documentation,\n" +
			"inconsistency, drift, bug, ux); severity nitpick or minor; and a remedy: field.\n" +
			"A security issue is always a person's. Every other open issue is handed back,\n" +
			"listed as ineligible, or skipped naming its blocker, by the rule that excluded it.\n\n" +
			"--dry-run shows every open issue's disposition, the eligible ones first in the\n" +
			"order a drain takes them (category tech-debt, documentation, inconsistency,\n" +
			"drift, bug, ux; then nitpick before minor; then oldest first), and writes\n" +
			"nothing. The host judgement over each eligible remedy does not run in a dry\n" +
			"run; it can only ever hand an issue back.\n\n" +
			"The run itself is not built: without --dry-run the verb refuses to start, and\n" +
			"exits 2 with nothing read or written.",
		Example: "  abcd drain --dry-run\n  abcd drain --dry-run --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !dryRun {
				if err := capture.DrainStart(); err != nil {
					return &exitError{Code: 2, Msg: "abcd drain: refused to start: " + err.Error() + " (nothing read, nothing written)"}
				}
			}
			repoRoot, err := ledgerRootFor(cmd, "abcd drain")
			if err != nil {
				return err
			}
			plan, err := capture.PlanDrain(capture.DrainPlanRequest{RepoRoot: repoRoot})
			if err != nil {
				return fmt.Errorf("abcd drain: %w", err)
			}
			out := drainOutput{DryRun: true, DrainPlan: plan}
			return renderLedger(cmd.OutOrStdout(), *asJSON, repoRoot, out, func(w io.Writer) { renderDrainPlan(w, plan) })
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show every open issue's disposition and the order a drain takes them; writes nothing")
	return cmd
}

// drainOutcomes is the order the counts line names the dispositions in.
var drainOutcomes = []capture.DrainOutcome{
	capture.DrainEligible, capture.DrainHandBack, capture.DrainIneligible,
	capture.DrainSkipped, capture.DrainUnreadable,
}

// renderDrainPlan is the dry run's text: the rule's record, the order, one
// line per open issue, and the counts. Every runtime string is sanitised.
func renderDrainPlan(w io.Writer, plan capture.DrainPlan) {
	fmt.Fprintf(w, "abcd drain --dry-run: %d open issue(s) classified by field; writes nothing\n", len(plan.Dispositions))
	fmt.Fprintf(w, "  rule:  %s (which issues need no decision)\n", termsafe.Sanitize(plan.Record))
	fmt.Fprintf(w, "  order: %s\n", termsafe.Sanitize(plan.Order))
	for _, v := range plan.Dispositions {
		kind := strings.TrimSpace(string(v.Severity) + " " + string(v.Category))
		if kind != "" {
			kind = " (" + kind + ")"
		}
		fmt.Fprintf(w, "  %-10s %s%s — %s\n", v.Outcome, termsafe.Sanitize(v.ID), termsafe.Sanitize(kind), termsafe.Sanitize(v.Reason))
	}
	counts := make([]string, 0, len(drainOutcomes))
	for _, o := range drainOutcomes {
		counts = append(counts, fmt.Sprintf("%d %s", plan.Counts[o], o))
	}
	fmt.Fprintf(w, "  counts: %s\n", strings.Join(counts, ", "))
	if plan.Counts[capture.DrainEligible] > 0 {
		fmt.Fprintln(w, "  the host judgement over each eligible remedy has not run; it can only hand an issue back")
	}
}
