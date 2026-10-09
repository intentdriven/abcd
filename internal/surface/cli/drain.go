package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// drainOutput is the --json envelope of `drain --dry-run`: the core's plan,
// marked as a dry run so a reader never takes it for a run's summary.
type drainOutput struct {
	DryRun bool `json:"dry_run"`
	capture.DrainPlan
}

// newDrainCommand builds the `drain` verb (itd-82, spc-2609212015054359):
// `--dry-run` classifies every open issue by the eligibility rule and writes
// nothing; the bare verb performs one move of the drain run (loop.Drain).
func newDrainCommand(asJSON *bool) *cobra.Command {
	var dryRun bool
	var maxLanes int
	var pace, subAgents, fixRounds string
	cmd := &cobra.Command{
		Use: "drain [--dry-run] [--max <n>] [--pace <work-minutes>/<pause-minutes>] [--sub-agents <n>] [--fix-rounds <n>]",
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
			"live, or names a release tag this checkout lacks, is always handed back. Every\n" +
			"other open issue is handed back, listed as ineligible, or skipped naming its\n" +
			"blocker, by the rule that excluded it.\n\n" +
			"--dry-run shows every open issue's disposition, the eligible ones first in the\n" +
			"order a drain takes them (by category, then severity, then oldest first), and\n" +
			"writes nothing. The host judgement over each eligible remedy does not run; it can\n" +
			"only ever hand an issue back.\n\n" +
			"Without --dry-run, each invocation performs one move of the drain and exits. It\n" +
			"hands the next eligible issue, in that order, to the implement loop's issue-keyed\n" +
			"lane (the run `abcd build <iss-N>` starts), one lane at a time, and names the run to\n" +
			"drive with `abcd implement step`. Run it again once that lane is handed back or its\n" +
			"pull request is open, and it routes the lane's outcome and opens the next. A lane\n" +
			"that finds a decision in its issue hands it back by kind, its work discarded: a\n" +
			"user-visible change is promoted to an intent draft (`capture promote`, which\n" +
			"stamps the issue's related_intents and nothing else); a trust or safety rule is\n" +
			"flagged as needing a decision record, with the question; a design finding or a\n" +
			"second package is flagged with the home the lane names. Every issue the rule\n" +
			"hands back is flagged naming the rule. Nothing but the promotion is written to\n" +
			"the ledger, and every hand-back is in the summary.\n\n" +
			"The drain is paced as a run is: its window and pause are --pace, --sub-agents and\n" +
			"--fix-rounds as `abcd build` reads them, set when the drain begins. At the window's\n" +
			"end the drain's state (.abcd/.work.local/run/drain.json) takes next_eligible_at and\n" +
			"the call opens nothing; before that time a drain opens nothing, and after it the\n" +
			"next invocation continues. --max <n> caps the lanes the drain opens (the default\n" +
			"is all); at the cap, or when nothing eligible is left, the drain reports and ends,\n" +
			"and the next `abcd drain` begins a new one. A cap or pace named while a drain is in\n" +
			"progress that differs from the one it began with is refused.\n\n" +
			"Without the repository's record, the dry run and the run both refuse (exit 2),\n" +
			"naming how to add it; `abcd ahoy install` offers it. A run that opens nothing or\n" +
			"merges nothing exits 0 and says why. Exit 2 on a refusal, exit 3 when another\n" +
			"drain or run holds the state lock.",
		Example: "  abcd drain --dry-run\n  abcd drain --dry-run --json\n  abcd drain --max 3\n  abcd drain --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			repoRoot, err := ledgerRootFor(cmd, "abcd drain")
			if err != nil {
				return err
			}
			if !dryRun {
				return runDrain(cmd, *asJSON, repoRoot, maxLanes, pace, subAgents, fixRounds)
			}
			if cmd.Flags().Changed("max") || cmd.Flags().Changed("pace") || cmd.Flags().Changed("sub-agents") || cmd.Flags().Changed("fix-rounds") {
				return &exitError{Code: 2, Msg: "abcd drain: --dry-run takes no --max, --pace, --sub-agents or --fix-rounds: it opens no lane (nothing written)"}
			}
			plan, err := capture.PlanDrain(capture.DrainPlanRequest{RepoRoot: repoRoot})
			// Every refusal of the rule exits 2, as the bare verb's does: a rule
			// unrecorded, ambiguous, malformed, or unreadable (a link, a record
			// past the size cap).
			if isDrainRuleRefusal(err) {
				return &exitError{Code: 2, Msg: "abcd drain: " + termsafe.Sanitize(err.Error()) + " (nothing written)"}
			}
			if err != nil {
				return fmt.Errorf("abcd drain: %w", err)
			}
			warnLoosened(cmd, plan.Record, plan.Loosened)
			out := drainOutput{DryRun: true, DrainPlan: plan}
			return renderLedger(cmd.OutOrStdout(), *asJSON, repoRoot, out, func(w io.Writer) { renderDrainPlan(w, plan) })
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show every open issue's disposition and the order a drain takes them; writes nothing")
	cmd.Flags().IntVar(&maxLanes, "max", 0, "cap the lanes this drain opens; the default is all")
	cmd.Flags().StringVar(&pace, "pace", "", "the drain's working window and pause, <work-minutes>/<pause-minutes>; wins over every configured layer")
	cmd.Flags().StringVar(&subAgents, "sub-agents", "", "the ceiling on lanes and validators alive at once; wins over every configured layer")
	cmd.Flags().StringVar(&fixRounds, "fix-rounds", "", "the fix rounds a lane may take before it is handed back; wins over every configured layer")
	return cmd
}

// warnLoosened names a loosened floor on stderr whatever the output mode
// (ruling H11), so a reader of --json sees it too.
func warnLoosened(cmd *cobra.Command, record string, loosened []string) {
	if len(loosened) > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "abcd drain: warning: this repository's rule (%s) loosens abcd's floors: a drain may take %s\n",
			termsafe.Sanitize(record), termsafe.Sanitize(strings.Join(loosened, ", ")))
	}
}

// runDrain is the bare verb: one move of the drain run.
func runDrain(cmd *cobra.Command, asJSON bool, repoRoot string, maxLanes int, pace, subAgents, fixRounds string) error {
	const prefix = "abcd drain"
	roots, notes := layered.RootsFor(repoRoot)
	for _, n := range notes {
		fmt.Fprintln(cmd.ErrOrStderr(), termsafe.Sanitize(n))
	}
	o := loop.Options{Roots: &roots}
	if cmd.Flags().Changed("pace") {
		o.Pace = &pace
	}
	if cmd.Flags().Changed("sub-agents") {
		o.SubAgents = &subAgents
	}
	if cmd.Flags().Changed("fix-rounds") {
		o.FixRounds = &fixRounds
	}
	res, err := loop.Drain(repoRoot, o, loop.DrainOptions{Max: maxLanes})
	if isDrainRuleRefusal(err) {
		return &exitError{Code: 2, Msg: "abcd drain: " + termsafe.Sanitize(err.Error()) + " (nothing written)"}
	}
	if err != nil {
		return loopFail(cmd.OutOrStdout(), asJSON, prefix, err)
	}
	warnLoosened(cmd, res.Rule, res.Loosened)
	for i := range res.Passed {
		res.Passed[i].Reason = fsutil.RedactHome(res.Passed[i].Reason)
	}
	res.Next = fsutil.RedactHome(res.Next)
	if res.Start != nil {
		res.Start.Next = fsutil.RedactHome(res.Start.Next)
	}
	return render(cmd.OutOrStdout(), asJSON, res, func(w io.Writer) { renderDrainRun(w, res) })
}

// renderDrainRun is the text summary of one drain move: the rule, the order,
// the cap and the pace; every lane the drain opened with its outcome; every
// hand-back with its route and the record change it made; the rule's flags;
// and the next move. Every runtime string is sanitised.
func renderDrainRun(w io.Writer, res loop.DrainResult) {
	verb := "continues"
	if res.Started {
		verb = "begins"
	}
	fmt.Fprintf(w, "abcd drain: the drain %s; state %s\n", verb, res.State)
	fmt.Fprintf(w, "  rule:  %s, this repository's own record\n", termsafe.Sanitize(res.Rule))
	for _, f := range res.Loosened {
		fmt.Fprintf(w, "  LOOSENED: %s\n", termsafe.Sanitize(f))
	}
	fmt.Fprintf(w, "  order: %s\n", termsafe.Sanitize(res.Order))
	if res.Max == 0 {
		fmt.Fprintln(w, "  cap:   none (--max not given: all)")
	} else {
		fmt.Fprintf(w, "  cap:   --max %d lane(s); %d opened\n", res.Max, len(res.Lanes))
	}
	renderPace(w, res.Pace)
	fmt.Fprintf(w, "  lanes: %d opened, one at a time\n", len(res.Lanes))
	for _, l := range res.Lanes {
		line := fmt.Sprintf("    %s  %s  %s", l.Issue, l.RunID, l.Outcome)
		if l.PR > 0 {
			line += fmt.Sprintf(" (pull request #%d)", l.PR)
		}
		fmt.Fprintln(w, line)
	}
	if len(res.HandBacks) > 0 {
		fmt.Fprintln(w, "  handed back by their lanes:")
		for _, r := range res.HandBacks {
			fmt.Fprintf(w, "    %s\n", termsafe.Sanitize(loop.DrainSummaryLine(r)))
		}
	}
	if len(res.Flags) > 0 {
		fmt.Fprintln(w, "  handed back by the rule (flags; written nowhere):")
		for _, r := range res.Flags {
			fmt.Fprintf(w, "    %s\n", termsafe.Sanitize(loop.DrainSummaryLine(r)))
		}
	}
	for _, p := range res.Passed {
		fmt.Fprintf(w, "  passed: %s (%s): %s\n", p.ID, p.Check, termsafe.Sanitize(p.Reason))
	}
	counts := map[capture.DrainOutcome]int{}
	for _, v := range res.Dispositions {
		counts[v.Outcome]++
	}
	parts := make([]string, 0, len(drainOutcomes))
	for _, o := range drainOutcomes {
		parts = append(parts, fmt.Sprintf("%d %s", counts[o], o))
	}
	fmt.Fprintf(w, "  ledger: %s (`abcd drain --dry-run` lists each)\n", strings.Join(parts, ", "))
	if res.NextEligibleAt != nil {
		fmt.Fprintf(w, "  next_eligible_at: %s\n", res.NextEligibleAt.UTC().Format(time.RFC3339))
	}
	switch res.Stopped {
	case loop.DrainStoppedCap:
		fmt.Fprintln(w, "  ended: the cap is reached")
	case loop.DrainStoppedEmpty:
		fmt.Fprintln(w, "  ended: nothing eligible is left")
	case loop.DrainStoppedOutage:
		fmt.Fprintln(w, "  ended: the run gave up on a lost connection")
	}
	fmt.Fprintln(w, "  the host judgement over each remedy is not built; a lane may still hand its issue back")
	fmt.Fprintf(w, "next: %s\n", termsafe.Sanitize(res.Next))
}

// isDrainRuleRefusal reports whether err is the rule load refusing: every one
// of drainrule's sentinels.
func isDrainRuleRefusal(err error) bool {
	for _, s := range []error{drainrule.ErrUnrecorded, drainrule.ErrMalformed, drainrule.ErrAmbiguous, drainrule.ErrUnreadable} {
		if errors.Is(err, s) {
			return true
		}
	}
	return false
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
	switch {
	case plan.Anchor != "" && plan.AnchorStale != "":
		fmt.Fprintf(w, "  anchor: %s is stale: an open record is deferred past %s, a release tag this checkout lacks, so every record deferred past a tag it lacks is handed back; `git fetch --tags` and drain again\n",
			termsafe.Sanitize(plan.Anchor), termsafe.Sanitize(plan.AnchorStale))
	case plan.Anchor != "":
		fmt.Fprintf(w, "  anchor: %s (a deferral past it is live)\n", termsafe.Sanitize(plan.Anchor))
	}
	if plan.AnchorUnknown {
		fmt.Fprintln(w, "  anchor: unknown: this checkout holds no release tag (a shallow clone fetches none), so every record carrying a deferral is handed back; `git fetch --tags` and drain again")
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
