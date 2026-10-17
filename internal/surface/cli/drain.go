package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// drainOutput is the --json envelope of `drain --dry-run`: the core's plan
// and the view of origin it was read from, marked as a dry run so a reader
// never takes it for a run's summary.
type drainOutput struct {
	DryRun bool `json:"dry_run"`
	loop.DrainPreview
}

// newDrainCommand builds the `drain` verb (itd-82, spc-2609212015054359):
// `--dry-run` classifies every open issue by the eligibility rule and writes
// nothing; the bare verb performs one move of the drain run (loop.Drain).
func newDrainCommand(asJSON *bool) *cobra.Command {
	var dryRun bool
	var maxLanes int
	var pace, subAgents, fixRounds, judgement string
	cmd := &cobra.Command{
		Use: "drain [--dry-run] [--max <n>] [--pace <work-minutes>/<pause-minutes>] [--sub-agents <n>] [--fix-rounds <n>] [--judgement <file>]",
		Long: "Work the open issue ledger unattended: fix the issues that need no decision, and\n" +
			"hand the rest back by kind. Which issues need no decision is this repository's own\n" +
			"recorded decision: an accepted decision record whose frontmatter carries the four\n" +
			"fields drain_categories, drain_severities, drain_security and drain_remedy. The\n" +
			"rule reads the issue's fields alone: nothing open in blocked_by; no intent in\n" +
			"related_intents; a category the rule takes; a severity it takes; and a remedy:\n" +
			"field. abcd's strict baseline takes tech-debt, documentation, inconsistency, drift,\n" +
			"bug and ux at nitpick or minor, and hands every security issue to a person. A\n" +
			"repository's record may loosen those floors (major, critical, security), and every\n" +
			"floor it loosens is named. Three hand-backs hold whatever the record says. An issue\n" +
			"whose related_intents names an intent is handed back as promoted, naming the\n" +
			"intents: it goes through its intent (`abcd build <itd-N>`, or `abcd intent audit\n" +
			"<itd-N>` for an audit's owed check), and a person's own `abcd build <iss-N>` still\n" +
			"takes it; a record the consistency pass files is the exception, since the intents\n" +
			"it names are the two it found contradicting: once a person has written its remedy,\n" +
			"it is judged by its other fields as if it named no intent, and an intent it was\n" +
			"promoted into still owns it. An issue whose remedy opens \"Waits on\", or whose\n" +
			"deferral past the current release tag is live, or names a release tag this checkout\n" +
			"lacks, is handed back too. Every other open issue is handed back, listed as\n" +
			"ineligible, or skipped naming its blocker, by the rule that excluded it.\n\n" +
			"In a checkout with a remote named origin, the ledger is read from origin's default\n" +
			"branch, never from the checkout, which may be behind it: each move fetches the\n" +
			"reservations and that branch first, under the drain's lock and before any judgement\n" +
			"or reservation, and plans from what it fetched. A move that cannot read origin (the\n" +
			"fetch fails, the default branch cannot be read, or no origin/HEAD names one) plans\n" +
			"nothing: it records the failure in the drain's state and waits as for an outage,\n" +
			"looking again after one minute, five, ten, then hourly, and exits 0 saying when.\n" +
			"A probe that fails eight hours into the hourly stage gives up, as the outage gate\n" +
			"does: the drain ends (stopped \"origin\"), and the next abcd drain begins a new\n" +
			"drain, waiting again if origin still cannot be read.\n\n" +
			"--dry-run shows every open issue's disposition, the eligible ones first in the\n" +
			"order a drain takes them (by category, then severity, then the issues naming a\n" +
			"file before the rest, then each account's own order on origin, drawn from the\n" +
			"holder git's identity names, when it names one, and the copy's key), and\n" +
			"writes nothing; on a copy with no key yet it takes each group oldest first and\n" +
			"says the copy's own order is drawn at its first move. It fetches nothing either: it plans from origin's default branch as\n" +
			"last fetched, and says how old that view is, how many commits the checkout is\n" +
			"behind it, and which issues open in the checkout it has already finished. The host\n" +
			"judgement over each eligible remedy does not run; it can only ever hand an issue\n" +
			"back.\n\n" +
			"Without --dry-run, each invocation performs one move of the drain and exits. Before\n" +
			"the next eligible issue's lane opens, the host judges its remedy: on origin, the\n" +
			"drain first reserves the issue (passing over, with no push, any issue or footprint\n" +
			"another copy holds), then writes\n" +
			"a request (.abcd/.work.local/run/drain-judgement.request.md) asking whether the\n" +
			"remedy changes what a user sees or a trust boundary, and opens nothing. The host\n" +
			"writes its answer where the request says and hands it back with --judgement <file>,\n" +
			"which is validated strictly; a refused answer changes nothing. When your machine's\n" +
			"config routes the drain-judge role to a runner (roles.drain-judge.runner), the same\n" +
			"move asks that runner instead, on a brief of its own, and takes its answer as\n" +
			"--judgement would, abcd computing the remedy's digest itself; a runner that does not\n" +
			"answer leaves the request to the host, recorded as a fallback. A yes hands the\n" +
			"issue back before any lane opens, routed as a lane's hand-back of the same kind\n" +
			"(below); a no lets the lane open on the reservation. The drain releases that\n" +
			"reservation, and only it, with a fetch and one tombstone push whenever no lane will\n" +
			"open on it: on a yes, on an answer over an issue that has left the eligible set,\n" +
			"when the judgement is superseded (the order asks another issue, nothing eligible is\n" +
			"left, the drain stops, or a no comes once the window has elapsed), when the lane's\n" +
			"start is refused or fails, and when a request lapses unanswered. A yes's tombstone\n" +
			"publishes the hand-back on origin, its kind and the blob id of the issue's record\n" +
			"there, as a lane's hand-back does, when the drain's reservation is still the\n" +
			"issue's tip: every account's drain passes the issue over, pushing and asking\n" +
			"nothing, until a person changes that record, and no drain takes over or renews a\n" +
			"reservation a person made over it, lapsed or not. A yes whose reservation has\n" +
			"moved on (renewed, reserved again or taken over since, or released by a person),\n" +
			"or that took none because you held the issue live, publishes nothing yet: the\n" +
			"drain keeps the hand-back pending (pending_hand_backs in --json) and publishes it\n" +
			"over the issue's next release on origin. A record changed or finished on origin's\n" +
			"default branch since, one origin holds in no one file, or a tombstone the hand-back\n" +
			"would take past its cap drops it, and the move says it was dropped and why; a\n" +
			"default branch that could not be read keeps it pending. A promotion\n" +
			"this checkout cannot make, not holding the record as origin does, is left to a\n" +
			"checkout that can. The reservation is taken once per request: one unanswered when it lapses is\n" +
			"dropped and the issue asked again only after the drain's other eligible work. The\n" +
			"judgement can never make an issue eligible:\n" +
			"an answer over an issue that has left the eligible set decides nothing, and one over\n" +
			"a remedy rewritten since the request is refused and asked again. Once judged, the\n" +
			"drain hands the next eligible issue, in that order, to the implement loop's issue-keyed\n" +
			"lane (the run `abcd build <iss-N>` starts), one lane at a time, and names the run to\n" +
			"drive with `abcd implement step`. Run it again once that lane is handed back or its\n" +
			"pull request is open, and it routes the lane's outcome and opens the next. A lane\n" +
			"that cannot finish does not hold the drain: each refused step of its run is logged\n" +
			"on the lane (refused in the run's state: the stage, the check, the reason, a count\n" +
			"of the same refusal in a row, since when and when last it refused there, and why it\n" +
			"cannot change), and once the step has been refused the same way three times, or has\n" +
			"kept refusing at one stage for sixty minutes, from its first refusal there to its\n" +
			"last, a quiet of more than fifteen minutes starting the time afresh\n" +
			"(pace.stuck_refusals and pace.stuck_minutes in the config layers), the next move\n" +
			"hands the lane back as stuck, releases its reservation with a tombstone marking it\n" +
			"stuck, and takes up the next issue. A refusal that cannot change (the issue\n" +
			"finished at the lane's base or on origin's default branch, the reservation now\n" +
			"another person's) hands it back at once; the wait for a preflight receipt, an\n" +
			"outage's wait, and a land fence that cannot read origin are never counted, and a\n" +
			"step that succeeds as the move hands the lane back keeps it. `abcd implement step\n" +
			"--discard` removes a stuck lane's worktree and branch and ends its run. Every move\n" +
			"renews the reservation of the lane in progress at half its lease, waiting or not\n" +
			"(a lane handed back is renewed by none), and the landing checks it before its push,\n" +
			"so a lane whose reservation became another person's pushes nothing and is handed\n" +
			"back as stuck. A lane\n" +
			"that finds a decision in its issue hands it back by kind, its work discarded: a\n" +
			"user-visible change is promoted to an intent draft (`capture promote`, which\n" +
			"stamps the issue's related_intents and nothing else); a trust or safety rule is\n" +
			"flagged as needing a decision record, with the question; a design finding or a\n" +
			"second package is flagged with the home the lane names. Every issue the rule\n" +
			"hands back is flagged naming the rule, less one this drain promoted itself, which\n" +
			"is counted once, as its promotion. Nothing but the promotion is written to the\n" +
			"ledger, and every hand-back is in the summary.\n\n" +
			"The drain is paced as a run is: its window and pause are --pace, --sub-agents and\n" +
			"--fix-rounds as `abcd build` reads them, set when the drain begins. At the window's\n" +
			"end the drain's state (.abcd/.work.local/run/drain.json) takes next_eligible_at and\n" +
			"the call opens nothing; before that time a drain opens nothing, and after it the\n" +
			"next invocation continues. --max <n> caps the lanes the drain opens (the default\n" +
			"is all); at the cap, or when nothing eligible is left, held or not, the drain reports\n" +
			"and ends. When you hold ten live reservations, the most one person may hold, or every\n" +
			"eligible issue left is held by another, a move neither refuses nor ends: it reserves\n" +
			"and asks nothing, writes waiting_until fifteen minutes on (apart from the pace\n" +
			"window) and waiting, \"cap\" or \"held\", for which of the two it waits on, names the\n" +
			"reservations and pull requests or the holds it waits on, and keeps its judgements;\n" +
			"run it again at or after that time. Once a drain has ended, the\n" +
			"next `abcd drain` begins a new one. A cap or pace named while a drain is in\n" +
			"progress that differs from the one it began with is refused. Once a run gives up on a\n" +
			"lost connection (`abcd implement outage`), the drain opens nothing and ends with\n" +
			"stopped \"outage\", naming the services down, since when, the probes and the lanes it\n" +
			"opened; the next `abcd drain`, once the outage is cleared, begins a new one.\n\n" +
			"Without the repository's record, the dry run and the run both refuse (exit 2),\n" +
			"naming how to add it; `abcd ahoy install` offers it. A run that opens nothing or\n" +
			"merges nothing exits 0 and says why. Exit 2 on a refusal, exit 3 when another\n" +
			"drain or run holds the state lock.",
		Example: "  abcd drain --dry-run\n  abcd drain --dry-run --json\n  abcd drain --max 3\n  abcd drain --json\n  abcd drain --judgement .abcd/.work.local/run/drain-judgement.json --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			repoRoot, err := ledgerRootFor(cmd, "abcd drain")
			if err != nil {
				return err
			}
			if !dryRun {
				return runDrain(cmd, *asJSON, repoRoot, maxLanes, pace, subAgents, fixRounds, judgement)
			}
			if cmd.Flags().Changed("max") || cmd.Flags().Changed("pace") || cmd.Flags().Changed("sub-agents") || cmd.Flags().Changed("fix-rounds") || cmd.Flags().Changed("judgement") {
				return &exitError{Code: 2, Msg: "abcd drain: --dry-run takes no --max, --pace, --sub-agents, --fix-rounds or --judgement: it opens no lane and asks no judgement (nothing written)"}
			}
			pv, err := loop.DrainDryRun(repoRoot, time.Now())
			// Every refusal of the rule exits 2, as the bare verb's does: a rule
			// unrecorded, ambiguous, malformed, or unreadable (a link, a record
			// past the size cap).
			if isDrainRuleRefusal(err) {
				return &exitError{Code: 2, Msg: "abcd drain: " + termsafe.Sanitize(err.Error()) + " (nothing written)"}
			}
			if err != nil {
				return fmt.Errorf("abcd drain: %w", err)
			}
			warnLoosened(cmd, pv.Record, pv.Loosened)
			pv.OriginUnread = fsutil.RedactHome(pv.OriginUnread)
			out := drainOutput{DryRun: true, DrainPreview: pv}
			return renderLedger(cmd.OutOrStdout(), *asJSON, repoRoot, out, func(w io.Writer) {
				renderDrainPlan(w, pv.DrainPlan)
				renderDrainOrigin(w, pv.Origin, pv.OriginUnread)
			})
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show every open issue's disposition and the order a drain takes them; writes nothing")
	cmd.Flags().IntVar(&maxLanes, "max", 0, "cap the lanes this drain opens; the default is all")
	cmd.Flags().StringVar(&pace, "pace", "", "the drain's working window and pause, <work-minutes>/<pause-minutes>; wins over every configured layer")
	cmd.Flags().StringVar(&subAgents, "sub-agents", "", "the ceiling on lanes and validators alive at once; wins over every configured layer")
	cmd.Flags().StringVar(&fixRounds, "fix-rounds", "", "the fix rounds a lane may take before it is handed back; wins over every configured layer")
	cmd.Flags().StringVar(&judgement, "judgement", "", "the host's answer to the judgement the drain awaits, at the path its request names")
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
func runDrain(cmd *cobra.Command, asJSON bool, repoRoot string, maxLanes int, pace, subAgents, fixRounds, judgement string) error {
	const prefix = "abcd drain"
	roots, notes := layered.RootsFor(repoRoot)
	for _, n := range notes {
		fmt.Fprintln(cmd.ErrOrStderr(), termsafe.Sanitize(n))
	}
	cfg, err := loadRunners(cmd, roots)
	if err != nil {
		return loopFail(cmd.OutOrStdout(), asJSON, prefix, err)
	}
	o := loop.Options{Roots: &roots, Runners: cfg}
	if cmd.Flags().Changed("pace") {
		o.Pace = &pace
	}
	if cmd.Flags().Changed("sub-agents") {
		o.SubAgents = &subAgents
	}
	if cmd.Flags().Changed("fix-rounds") {
		o.FixRounds = &fixRounds
	}
	if cmd.Flags().Changed("judgement") && strings.TrimSpace(judgement) == "" {
		return &exitError{Code: 2, Msg: "abcd drain: --judgement names no file; give the answer's path the judgement request names (nothing written)"}
	}
	res, err := loop.Drain(repoRoot, o, loop.DrainOptions{Max: maxLanes, Judgement: judgement, Transcripts: &lazyHistoryStore{cmd: cmd}})
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
		res.Start.ForkUnchecked = fsutil.RedactHome(res.Start.ForkUnchecked)
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
	if res.Start != nil {
		renderBudget(w, res.Start.Checks)
		// A lane's reservation that could not ask the forge whether origin is
		// a fork, or was let through on the fork the repository allows, says
		// so, as reserve does (spc-2610161044461448, step 10).
		if res.Start.ForkUnchecked != "" {
			fmt.Fprintf(w, "  %s\n", termsafe.Sanitize(claim.ForkUncheckedLine(res.Start.ForkUnchecked)))
		}
		if res.Start.ForkAllowed != "" {
			fmt.Fprintf(w, "  %s\n", termsafe.Sanitize(claim.ForkAllowedLine(res.Start.ForkAllowed)))
		}
	}
	fmt.Fprintf(w, "  lanes: %d opened, one at a time\n", len(res.Lanes))
	for _, l := range res.Lanes {
		line := fmt.Sprintf("    %s  %s  %s", l.Issue, l.RunID, l.Outcome)
		if l.PR > 0 {
			line += fmt.Sprintf(" (pull request #%d)", l.PR)
		}
		fmt.Fprintln(w, line)
	}
	if res.Judged != nil {
		fmt.Fprintf(w, "  judged: %s\n", termsafe.Sanitize(loop.JudgementSummary(*res.Judged)))
	}
	if len(res.Judgements) > 0 {
		fmt.Fprintf(w, "  host judgements: %d taken, each over the remedy as it stood\n", len(res.Judgements))
	}
	if res.Judging != nil {
		fmt.Fprintf(w, "  awaiting the host judgement on %s: request %s, answer %s\n",
			termsafe.Sanitize(res.Judging.Issue), termsafe.Sanitize(res.Judging.Request), termsafe.Sanitize(res.Judging.Answer))
	}
	if len(res.HandBacks) > 0 {
		fmt.Fprintln(w, "  handed back by their lanes or the host judgement:")
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
	for _, p := range res.PendingHandBacks {
		fmt.Fprintf(w, "  hand-back pending: %s (%s), routed %s, not yet on origin: %s\n", p.Issue, p.Kind,
			p.At.UTC().Format(time.RFC3339), termsafe.Sanitize(p.Why))
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
	if res.Origin != nil {
		renderDrainOrigin(w, res.Origin, "")
	}
	if wt := res.OriginWait; wt != nil {
		fmt.Fprintf(w, "  origin: waiting (%s) since %s, %d probe(s); next at %s\n", termsafe.Sanitize(wt.Kind),
			wt.Since.UTC().Format(time.RFC3339), wt.Probes, wt.NextProbeAt.UTC().Format(time.RFC3339))
	}
	renderDrainCounts(w, res.Counts)
	if res.NextEligibleAt != nil {
		fmt.Fprintf(w, "  next_eligible_at: %s\n", res.NextEligibleAt.UTC().Format(time.RFC3339))
	}
	if res.WaitingUntil != nil {
		fmt.Fprintf(w, "  waiting_until: %s (%s)\n", res.WaitingUntil.UTC().Format(time.RFC3339), termsafe.Sanitize(res.Waiting))
	}
	switch res.Stopped {
	case loop.DrainStoppedCap:
		fmt.Fprintln(w, "  ended: the cap is reached")
	case loop.DrainStoppedEmpty:
		fmt.Fprintln(w, "  ended: nothing eligible is left")
	case loop.DrainStoppedOutage:
		fmt.Fprintln(w, "  ended: the run gave up on a lost connection")
	case loop.DrainStoppedOrigin:
		fmt.Fprintln(w, "  ended: the drain gave up waiting on origin")
	}
	fmt.Fprintf(w, "next: %s\n", termsafe.Sanitize(res.Next))
}

// renderDrainCounts is the summary's counts (itd-82 scope 8): the lanes by
// outcome and the pull requests they opened, the hand-backs by route, the
// rule's flags, the judgements, the passed issues, and the spend in agent
// runs, named a floor when a lane's run could not be read.
func renderDrainCounts(w io.Writer, c loop.DrainCounts) {
	fmt.Fprintf(w, "  counts: %d lane(s) opened (%d in progress, %d pull request(s) opened, %d handed back); "+
		"hand-backs routed: %d promoted, %d flagged for a decision record, %d flagged with a home; "+
		"%d flagged by the rule; %d host judgement(s); %d passed\n",
		c.LanesOpened, c.InProgress, c.PullRequests, c.LanesHandedBack, c.Promotions, c.DecisionRecords, c.Homes,
		c.RuleFlags, c.Judgements, c.Passed)
	floor := ""
	if len(c.Spend.Unread) > 0 {
		floor = "at least "
	}
	fmt.Fprintf(w, "  spend:  %s%d agent run(s) (%d implementer(s), %d validator(s)) across the drain's lanes, the unit the budget check counts in",
		floor, c.Spend.AgentRuns, c.Spend.Implementers, c.Spend.Validators)
	if floor != "" {
		fmt.Fprintf(w, "; the state of %s could not be read", termsafe.Sanitize(strings.Join(c.Spend.Unread, ", ")))
	}
	fmt.Fprintln(w)
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

// renderDrainOrigin is the view of origin's default branch a plan was read
// from: the branch and commit, the view's age, the commits this checkout is
// behind, and the issues open here that it has finished; or why there is no
// such view.
func renderDrainOrigin(w io.Writer, o *loop.DrainOrigin, unread string) {
	if o == nil {
		if unread != "" {
			fmt.Fprintf(w, "  origin: %s\n", termsafe.Sanitize(unread))
		}
		return
	}
	age := "at a time nothing here recorded"
	if o.FetchedAt != nil {
		age = fmt.Sprintf("%d minute(s) before, at %s", o.AgeMinutes, o.FetchedAt.UTC().Format(time.RFC3339))
	}
	fmt.Fprintf(w, "  origin: planned from %s at %s, as last fetched %s; this checkout is %d commit(s) behind it\n",
		termsafe.Sanitize(o.Ref), termsafe.Sanitize(shortCommit(o.Commit)), age, o.Behind)
	if len(o.Finished) > 0 {
		fmt.Fprintf(w, "  finished on %s, still open in this checkout: %s\n", termsafe.Sanitize(o.Ref), termsafe.Sanitize(strings.Join(o.Finished, ", ")))
	}
}
