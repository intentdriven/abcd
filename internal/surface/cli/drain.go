package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/drainrule"
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
			"hand the rest back by kind. Which issues need no decision is this repository's own\n" +
			"recorded decision: an accepted decision record whose frontmatter carries the four\n" +
			"fields drain_categories, drain_severities, drain_security and drain_remedy. The\n" +
			"rule reads the record's fields alone: nothing open in blocked_by; a category the\n" +
			"rule takes; a severity it takes; and a remedy: field. abcd's strict baseline takes\n" +
			"tech-debt, documentation, inconsistency, drift, bug and ux at nitpick or minor, and\n" +
			"hands every security issue to a person. A repository's record may loosen those\n" +
			"floors (major, critical, security), and every floor it loosens is named. An issue\n" +
			"whose remedy opens \"Waits on\", or whose deferral past the current release tag is\n" +
			"live, is always handed back. Every other open issue is handed back, listed as\n" +
			"ineligible, or skipped naming its blocker, by the rule that excluded it.\n\n" +
			"--dry-run shows every open issue's disposition, the eligible ones first in the\n" +
			"order a drain takes them (by category, then severity, then oldest first), and\n" +
			"writes nothing. The host judgement over each eligible remedy does not run in a dry\n" +
			"run; it can only ever hand an issue back.\n\n" +
			"Without the repository's record, the dry run and the run both refuse (exit 2),\n" +
			"naming how to add it; `abcd ahoy install` offers it. The run itself is not built:\n" +
			"without --dry-run the verb refuses to start, and exits 2 with nothing written.",
		Example: "  abcd drain --dry-run\n  abcd drain --dry-run --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			repoRoot, err := ledgerRootFor(cmd, "abcd drain")
			if err != nil {
				return err
			}
			if !dryRun {
				err := capture.DrainStart(repoRoot)
				return &exitError{Code: 2, Msg: "abcd drain: refused to start: " + termsafe.Sanitize(err.Error()) + " (nothing written)"}
			}
			plan, err := capture.PlanDrain(capture.DrainPlanRequest{RepoRoot: repoRoot})
			if errors.Is(err, drainrule.ErrUnrecorded) || errors.Is(err, drainrule.ErrMalformed) || errors.Is(err, drainrule.ErrAmbiguous) {
				return &exitError{Code: 2, Msg: "abcd drain: " + termsafe.Sanitize(err.Error()) + " (nothing written)"}
			}
			if err != nil {
				return fmt.Errorf("abcd drain: %w", err)
			}
			// A loosened floor is loud (ruling H11): named on stderr whatever the
			// output mode, so a reader of --json sees it too.
			if len(plan.Loosened) > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd drain: warning: this repository's rule (%s) loosens abcd's floors: a drain may take %s\n",
					termsafe.Sanitize(plan.Record), termsafe.Sanitize(strings.Join(plan.Loosened, ", ")))
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
	fmt.Fprintf(w, "  rule:  %s, this repository's own record (which issues need no decision)\n", termsafe.Sanitize(plan.Record))
	if len(plan.Loosened) == 0 {
		fmt.Fprintln(w, "  floors: the rule loosens none of abcd's floors")
	} else {
		fmt.Fprintln(w, "  LOOSENED: this repository's rule lets a drain take what abcd's baseline hands to a person:")
		for _, f := range plan.Loosened {
			fmt.Fprintf(w, "    - %s\n", termsafe.Sanitize(f))
		}
	}
	if plan.Anchor != "" {
		fmt.Fprintf(w, "  anchor: %s (a deferral past it is live)\n", termsafe.Sanitize(plan.Anchor))
	}
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
