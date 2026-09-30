package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// build.go is the front door onto internal/core/implement/loop
// (itd-2609201916151817, spc-2609202134338445): `abcd build <itd-N>`, the verb a
// person types, and the step interface a driving host calls under `implement`
// (decision 8: `build` for people, `implement` for the machinery) — `implement
// status`, `implement step`, `implement receipt` and `implement record`.

// loopStore is the noun the checkout resolution names in its refusal.
const loopStore = "the run state file"

// loopRoot resolves the checkout the caller stands in.
func loopRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return gitutil.CheckoutRoot(cwd, loopStore)
}

// loopRefusalDoc is the --json document a refusal renders before the error
// envelope: the stage, the reason and the remedy as fields (criterion 13), and
// every pre-start check's row when the refusal is a check's.
type loopRefusalDoc struct {
	Refusal loop.Refusal `json:"refusal"`
}

// loopFail maps an error from the loop to the process outcome. A refusal exits
// 2, and contention (a peer holds the record, the run is paused or locked) exits
// 3: back off and take other work, as `abcd implement` does. Under --json the
// refusal is rendered as its own document first, so a machine reads the stage,
// the reason and the remedy as fields; the envelope follows it, last on the
// stream as every refusal is.
func loopFail(w io.Writer, asJSON bool, prefix string, err error) error {
	if r, ok := loop.AsRefusal(err); ok {
		red := *r
		red.Reason = fsutil.RedactHome(red.Reason)
		red.Remedy = fsutil.RedactHome(red.Remedy)
		if len(red.Excluded) > 0 {
			ex := make([]loop.Excluded, len(red.Excluded))
			for i, e := range red.Excluded {
				e.Reason = fsutil.RedactHome(e.Reason)
				ex[i] = e
			}
			red.Excluded = ex
		}
		code := 2
		if red.Contention {
			code = 3
		}
		if asJSON {
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			if encErr := enc.Encode(loopRefusalDoc{Refusal: red}); encErr != nil {
				return encErr
			}
		}
		return &exitError{Code: code, Msg: prefix + ": " + termsafe.Sanitize(red.Error()) + " (nothing written)"}
	}
	if errorsIsNoCheckout(err) {
		return &exitError{Code: 2, Msg: prefix + ": " + err.Error() + " (nothing written)"}
	}
	return fmt.Errorf("%s: %w", prefix, err)
}

// errorsIsNoCheckout reports the checkout-root refusal.
func errorsIsNoCheckout(err error) bool { return errors.Is(err, gitutil.ErrNoCheckoutRoot) }

// newBuildCommand builds `abcd build <itd-N>`.
func newBuildCommand(asJSON *bool) *cobra.Command {
	var session, pace, subAgents, fixRounds string
	cmd := &cobra.Command{
		Use: "build <itd-N> [--session <id>] [--pace <work-minutes>/<pause-minutes>] [--sub-agents <n>] [--fix-rounds <n>]",
		Long: "Start the implement loop for one intent, or resume the run already in progress for it.\n" +
			"A new run's checks run first, and every one must pass:\n" +
			"the intent is READY (planned, criteria written, its spec linked and written), asks no\n" +
			"open question, has no unanswered claim section, is not held, names no unshipped intent\n" +
			"in `blocked_by`, its spec leaves a step to build, and no peer holds it (no sibling\n" +
			"worktree or local branch holds it in another bucket, and no session holds a live claim\n" +
			"on it; a peer or claim that cannot be read counts as holding it). A refusal names the\n" +
			"check, the reason and the remedy, and writes nothing. `abcd build next` picks the intent\n" +
			"instead of taking one named.\n\n" +
			"When the checks pass, the run is created in this checkout's local tier,\n" +
			"`.abcd/.work.local/run/<run-id>/state.json`: one lane for the spec's first unlanded step,\n" +
			"the other unlanded steps pending, and the run record's first line. The tier itself is\n" +
			"never created: only a repository abcd manages has one. Starting again while the run is\n" +
			"in progress creates nothing and names the run without judging the checks again (the\n" +
			"run's own lanes change what they read), so a killed process resumes where it stopped.\n\n" +
			"--session names the host session's id in the shared run state (`abcd implement join`):\n" +
			"a new run then claims the intent there for that session, the run id as its lane, so a\n" +
			"build of the same intent from any other checkout of the repository is refused as held\n" +
			"before this run's lane has moved or claimed anything, and the session's own claim on the\n" +
			"intent is not counted as a peer's. A session that has not joined is refused. Without it\n" +
			"the run holds no claim, and the result says so.\n\n" +
			"A new run is paced: a working window, a pause after it, a ceiling on the run's lanes and\n" +
			"validators alive at once, and the fix rounds a lane may take before it is handed back. The\n" +
			"four numbers are read once, when the run starts: --pace <work-minutes>/<pause-minutes>,\n" +
			"--sub-agents <n> and --fix-rounds <n> for this run, else pace.work_minutes, pace.pause_minutes,\n" +
			"pace.sub_agents and pace.fix_rounds in the repository's .abcd/config.json, else in\n" +
			"~/.abcd/config.json, else the bundled 120/300 with 2 sub-agents and 3 fix rounds. The result and the run\n" +
			"record name each number's layer. A malformed pace or ceiling, typed or configured, is\n" +
			"refused naming the value and the accepted form, and writes nothing. Starting again keeps\n" +
			"the run's pace; a flag naming another is refused. The window and the pause bind through\n" +
			"`abcd implement step`; the ceiling is recorded with the run, and this build does not\n" +
			"count lanes against it. A lane whose validators still do not pass after its fix rounds is\n" +
			"handed back: it stops as unachievable with the last round's findings, the run starts nothing\n" +
			"further for it, and `abcd implement step` refuses naming the hand-back.\n\n" +
			"The run then moves one step per `abcd implement step`, driven by the host session.\n\n" +
			"Exit 2 on a refusal, exit 3 when a peer holds the intent or the run state is locked\n" +
			"(back off and take other work).",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			const prefix = "abcd build"
			root, err := loopRoot()
			if err != nil {
				return loopFail(cmd.OutOrStdout(), *asJSON, prefix, err)
			}
			roots, notes := layered.RootsFor(root)
			for _, n := range notes {
				fmt.Fprintln(cmd.ErrOrStderr(), termsafe.Sanitize(n))
			}
			o := loop.Options{Session: session, Roots: &roots}
			if cmd.Flags().Changed("pace") {
				o.Pace = &pace
			}
			if cmd.Flags().Changed("sub-agents") {
				o.SubAgents = &subAgents
			}
			if cmd.Flags().Changed("fix-rounds") {
				o.FixRounds = &fixRounds
			}
			res, err := loop.Start(root, args[0], o)
			if err != nil {
				return loopFail(cmd.OutOrStdout(), *asJSON, prefix, err)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				verb := "started"
				if res.Resumed {
					verb = "resumed"
				}
				fmt.Fprintf(w, "build %s: %s run %s\n", termsafe.Sanitize(args[0]), verb, res.RunID)
				fmt.Fprintf(w, "  state:   %s\n", res.State)
				renderPace(w, res.Pace)
				renderLaneLine(w, res.Lane)
				renderPending(w, res.Pending)
				switch {
				case res.Claim != nil:
					fmt.Fprintf(w, "  claim:   %s for session %s until %s\n", res.Claim.Claim.Record,
						termsafe.Sanitize(res.Claim.Claim.Session), res.Claim.Claim.ExpiresAt.Format(time.RFC3339))
				case !res.Resumed:
					fmt.Fprintln(w, "  claim:   none (no --session): a build in another checkout cannot see this run until its lane shows")
				}
				fmt.Fprintf(w, "next: %s\n", termsafe.Sanitize(fsutil.RedactHome(res.Next)))
			})
		},
	}
	cmd.Flags().StringVar(&session, "session", "", "the host session's id in the shared run state; a new run claims the intent for it")
	cmd.Flags().StringVar(&pace, "pace", "", "this run's working window and pause, <work-minutes>/<pause-minutes> (e.g. 90/240); wins over every configured layer")
	cmd.Flags().StringVar(&subAgents, "sub-agents", "", "this run's ceiling on lanes and validators alive at once, a whole number; wins over every configured layer")
	cmd.Flags().StringVar(&fixRounds, "fix-rounds", "", "the fix rounds a lane of this run may take before it is handed back, a whole number from 0 (bundled: 3); wins over every configured layer")
	cmd.AddCommand(newBuildNextCommand(asJSON))
	return cmd
}

// newBuildNextCommand builds `abcd build next` (itd-2609211116005482).
func newBuildNextCommand(asJSON *bool) *cobra.Command {
	var session, pace, subAgents, fixRounds string
	var maxPicks int
	var untilEmpty bool
	cmd := &cobra.Command{
		Use: "next [--session <id>] [--pace <work-minutes>/<pause-minutes>] [--sub-agents <n>] [--fix-rounds <n>] [--max <n>] [--until-empty]",
		Long: "Pick the readiest planned intent, write down why, and start its run.\n\n" +
			"The candidates are the planned intents that pass every check `abcd build <itd-N>` runs\n" +
			"(READY, no open question, no unanswered claim section, not held, no unshipped intent in\n" +
			"`blocked_by`, a step left to build, no peer holding it), less one this checkout already has\n" +
			"a run in progress for. Each is scored from its record, three parts at equal weight, each 0\n" +
			"to 100: criteria clarity (the share of its acceptance criteria in Given-When-Then form), a\n" +
			"test path (its spec's `## Footprint` names tests) and the expected footprint (100 divided by\n" +
			"the packages that section names). A part whose section is absent reads zero, and the\n" +
			"reason says the spec carries no footprint. The readiest is taken; the oldest among equals,\n" +
			"and the reason then says the tie was broken by age.\n\n" +
			"The pick starts the run `abcd build <itd-N>` would start for that intent, with the pick in\n" +
			"the run's state. The reason is one `pursued:` grounds entry opening `picked by run <run-id>\n" +
			"on <date>`: every candidate with its score, the rule, the runner-up and why it lost, and the\n" +
			"falsifier. The lane's worktree stage appends it to the intent in the lane's own worktree and\n" +
			"commits it there as the lane branch's first commit, record-only, before the brief; the\n" +
			"receipt verifier does not count that commit as the implementer's. The checkout you run this\n" +
			"in is never written but for the run state. `abcd intent ready` keeps reporting the person's\n" +
			"entry as the most recent conjecture.\n\n" +
			"One pick per invocation. --max <n> above 1 and --until-empty, which continue under the pace\n" +
			"rule, are refused: that half of the verb is not built in this abcd. --session, --pace,\n" +
			"--sub-agents and --fix-rounds are `abcd build`'s own. A lane handed back after its fix\n" +
			"rounds falsifies the pick: the run record says so, and the intent's entry is not edited.\n\n" +
			"No candidate is refused, naming each excluded intent and the check that excluded it, and\n" +
			"nothing is written. Exit 2 on a refusal, exit 3 when the chosen intent's run is already in\n" +
			"progress or the run state is locked.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			const prefix = "abcd build next"
			root, err := loopRoot()
			if err != nil {
				return loopFail(cmd.OutOrStdout(), *asJSON, prefix, err)
			}
			roots, notes := layered.RootsFor(root)
			for _, n := range notes {
				fmt.Fprintln(cmd.ErrOrStderr(), termsafe.Sanitize(n))
			}
			o := loop.Options{Session: session, Roots: &roots}
			if cmd.Flags().Changed("pace") {
				o.Pace = &pace
			}
			if cmd.Flags().Changed("sub-agents") {
				o.SubAgents = &subAgents
			}
			if cmd.Flags().Changed("fix-rounds") {
				o.FixRounds = &fixRounds
			}
			res, err := loop.Next(root, o, loop.NextOptions{Max: maxPicks, UntilEmpty: untilEmpty})
			if err != nil {
				return loopFail(cmd.OutOrStdout(), *asJSON, prefix, err)
			}
			for i := range res.Excluded {
				res.Excluded[i].Reason = fsutil.RedactHome(res.Excluded[i].Reason)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) { renderNext(w, res) })
		},
	}
	cmd.Flags().StringVar(&session, "session", "", "the host session's id in the shared run state; the new run claims the picked intent for it")
	cmd.Flags().StringVar(&pace, "pace", "", "the new run's working window and pause, <work-minutes>/<pause-minutes>; wins over every configured layer")
	cmd.Flags().StringVar(&subAgents, "sub-agents", "", "the new run's ceiling on lanes and validators alive at once; wins over every configured layer")
	cmd.Flags().StringVar(&fixRounds, "fix-rounds", "", "the fix rounds a lane of the new run may take before it is handed back; wins over every configured layer")
	cmd.Flags().IntVar(&maxPicks, "max", 0, "how many picks to make; only 1 is built, and more is refused")
	cmd.Flags().BoolVar(&untilEmpty, "until-empty", false, "pick until no candidate is left; not built, and refused")
	return cmd
}

// renderNext is the text form of a pick.
func renderNext(w io.Writer, res loop.NextResult) {
	p := res.Pick
	fmt.Fprintf(w, "build next: picked %s (score %d) from %d candidate(s); run %s started\n",
		p.Chosen.ID, p.Chosen.Score.Total, len(res.Candidates), res.Start.RunID)
	fmt.Fprintln(w, "  candidates, in the pick order:")
	for _, c := range res.Candidates {
		s := c.Score
		line := fmt.Sprintf("    %s  %d  criteria %d (%s), test path %d, footprint %d", c.ID, s.Total,
			s.Criteria.Points, s.Criteria.Detail, s.TestPath.Points, s.Footprint.Points)
		if s.NoFootprint {
			line += "; its spec carries no footprint"
		}
		fmt.Fprintln(w, termsafe.Sanitize(line))
	}
	if len(res.Excluded) > 0 {
		fmt.Fprintln(w, "  excluded:")
		for _, e := range res.Excluded {
			fmt.Fprintf(w, "    %s  %s: %s\n", e.ID, e.Check, termsafe.Sanitize(e.Reason))
		}
	}
	fmt.Fprintf(w, "  rule:    %s\n", p.Rule)
	switch {
	case p.RunnerUp == nil:
		fmt.Fprintln(w, "  runner-up: none (the only candidate)")
	case p.TieBrokenByAge:
		fmt.Fprintf(w, "  runner-up: %s, tied at %d; the tie was broken by age\n", p.RunnerUp.ID, p.RunnerUp.Score.Total)
	default:
		fmt.Fprintf(w, "  runner-up: %s at %d, lost on score\n", p.RunnerUp.ID, p.RunnerUp.Score.Total)
	}
	fmt.Fprintf(w, "  entry:   pursued: %s\n", termsafe.Sanitize(res.Entry))
	fmt.Fprintf(w, "           (the lane's worktree stage commits it as %s's first commit)\n", res.Start.Lane.ID)
	fmt.Fprintf(w, "  state:   %s\n", res.Start.State)
	renderPace(w, res.Start.Pace)
	renderLaneLine(w, res.Start.Lane)
	renderPending(w, res.Start.Pending)
	fmt.Fprintf(w, "next: %s\n", termsafe.Sanitize(fsutil.RedactHome(res.Start.Next)))
}

// renderPace renders a run's pace and the layer each number came from; a run
// started before the loop paced a run says so.
func renderPace(w io.Writer, p *loop.Pace) {
	if p == nil {
		fmt.Fprintln(w, "  pace:    none (the run started before the loop paced a run)")
		return
	}
	fmt.Fprintf(w, "  pace:    %s\n", termsafe.Sanitize(p.String()))
}

// renderLaneLine renders one lane as a line, and its footprint once its stages
// have made one.
func renderLaneLine(w io.Writer, l loop.Lane) {
	fmt.Fprintf(w, "  %s:  spec step %d, %q — next stage: %s\n", l.ID, l.SpecStep, termsafe.Sanitize(l.StepTitle), l.Stage)
	if l.Branch != "" {
		fmt.Fprintf(w, "    branch %s (%s..%s), worktree %s\n", termsafe.Sanitize(l.Branch), shortSHA(l.BaseSHA), shortSHA(l.HeadSHA),
			termsafe.Sanitize(fsutil.RedactHome(l.Worktree)))
	}
	if n := len(l.Validation); n > 0 {
		r := l.Validation[n-1]
		parts := make([]string, 0, len(r.Validators))
		for _, v := range r.Validators {
			verdict := v.Verdict
			if verdict == "" {
				verdict = "pending"
			}
			parts = append(parts, v.Role+" "+verdict)
		}
		fmt.Fprintf(w, "    validation round %d at %s: %s\n", r.Round, shortSHA(r.HeadSHA), termsafe.Sanitize(strings.Join(parts, ", ")))
	}
	renderLanding(w, l.PR, l.Landing)
	if l.Awaiting != nil {
		fmt.Fprintf(w, "    awaiting the %s's receipt at %s (brief %s)\n", termsafe.Sanitize(l.Awaiting.Role),
			termsafe.Sanitize(fsutil.RedactHome(l.Awaiting.Receipt)), termsafe.Sanitize(fsutil.RedactHome(l.Awaiting.Brief)))
	}
}

// renderPending renders the spec steps waiting for a lane.
func renderPending(w io.Writer, pending []loop.PendingStep) {
	if len(pending) == 0 {
		return
	}
	parts := make([]string, 0, len(pending))
	for _, p := range pending {
		parts = append(parts, fmt.Sprintf("%d %q", p.Number, termsafe.Sanitize(p.Title)))
	}
	fmt.Fprintf(w, "  pending: spec step %s\n", strings.Join(parts, ", "))
}

// redactAwait home-redacts the paths an await carries, for a stream. They keep
// RedactHome rather than fsutil.DisplayPath: the brief is the file the agent is
// handed and the receipt the path it writes and passes to `implement receipt`,
// so a base name would leave it unable to act on either.
func redactAwait(a *loop.Await) *loop.Await {
	if a == nil {
		return nil
	}
	c := *a
	c.Brief = fsutil.RedactHome(c.Brief)
	c.Receipt = fsutil.RedactHome(c.Receipt)
	return &c
}

// implementStatusRuns is `implement status`'s --json shape.
type implementStatusRuns struct {
	Runs []loop.State `json:"runs"`
}

func newImplementStatusCommand(asJSON *bool) *cobra.Command {
	var runID string
	cmd := &cobra.Command{
		Use: "status [--run <run-id>]",
		Long: "Render the runs `abcd build` started in this checkout, or the one --run names: the\n" +
			"intent and spec, each lane with its spec step and next stage, what an awaiting lane\n" +
			"waits on, the pending spec steps, and the run record. Read-only: it writes nothing\n" +
			"and creates nothing. Exit 2 when --run names no run.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			const prefix = "abcd implement status"
			root, err := loopRoot()
			if err != nil {
				return loopFail(cmd.OutOrStdout(), *asJSON, prefix, err)
			}
			var runs []loop.State
			if runID != "" {
				st, err := loop.ReadState(root, runID)
				if err != nil {
					return loopFail(cmd.OutOrStdout(), *asJSON, prefix, err)
				}
				runs = []loop.State{st}
			} else if runs, err = loop.Runs(root); err != nil {
				return loopFail(cmd.OutOrStdout(), *asJSON, prefix, err)
			}
			for i := range runs {
				for j := range runs[i].Lanes {
					runs[i].Lanes[j].Awaiting = redactAwait(runs[i].Lanes[j].Awaiting)
					runs[i].Lanes[j].Worktree = fsutil.DisplayPath(runs[i].Lanes[j].Worktree)
				}
			}
			return render(cmd.OutOrStdout(), *asJSON, implementStatusRuns{Runs: runs}, func(w io.Writer) {
				if len(runs) == 0 {
					fmt.Fprintln(w, "no run in this checkout — start one with `abcd build <itd-N>`")
					return
				}
				for _, st := range runs {
					state := "in progress"
					if st.Complete() {
						state = "complete"
					}
					for _, l := range st.Lanes {
						if l.HandBack != nil {
							state = "handed back (" + l.ID + ", " + l.HandBack.Verdict + ")"
						}
					}
					fmt.Fprintf(w, "run %s  %s (%s)  %s, driven by the %s\n", st.RunID, st.Key, st.Spec, state, st.Driver)
					fmt.Fprintf(w, "  state:   %s\n", loop.StateRelPath(st.RunID))
					renderPace(w, st.Pace)
					if st.NextEligibleAt != nil {
						fmt.Fprintf(w, "  paused until %s\n", st.NextEligibleAt.UTC().Format("2006-01-02T15:04:05Z07:00"))
					}
					for _, l := range st.Lanes {
						renderLaneLine(w, l)
					}
					renderPending(w, st.Pending)
					fmt.Fprintf(w, "  record:  %d line(s)\n", len(st.Record))
					for _, e := range st.Record {
						fmt.Fprintf(w, "    %s  %-9s %s  %s\n", e.At.Format("2006-01-02T15:04:05Z"), termsafe.Sanitize(e.Stage),
							termsafe.Sanitize(e.Lane), termsafe.Sanitize(fsutil.RedactHome(e.Note)))
					}
				}
			})
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "the run to render (run-<16 digits>); every run in this checkout when omitted")
	return cmd
}

// renderStepResult is the text form of an `implement step` or `implement receipt` result.
func renderStepResult(w io.Writer, verb string, res loop.StepResult) {
	switch {
	case res.HandBack != nil:
		fmt.Fprintf(w, "%s: HANDED BACK: %s's %s stopped as %s after %d fix round(s); the run starts nothing further for it\n",
			verb, res.RunID, res.Lane, res.HandBack.Verdict, res.HandBack.FixRounds)
	case res.NextEligibleAt != nil:
		fmt.Fprintf(w, "%s: %s's window has elapsed; paused until %s\n", verb, res.RunID, res.NextEligibleAt.UTC().Format(time.RFC3339))
	case res.PerformedStage != "":
		fmt.Fprintf(w, "%s: %s completed %s's %s stage\n", verb, res.RunID, res.Lane, res.PerformedStage)
	case res.Awaiting != nil:
		fmt.Fprintf(w, "%s: %s's %s stage awaits the %s's receipt\n", verb, res.Lane, res.Stage, termsafe.Sanitize(res.Awaiting.Role))
		fmt.Fprintf(w, "  brief:   %s\n  receipt: %s\n", termsafe.Sanitize(res.Awaiting.Brief), termsafe.Sanitize(res.Awaiting.Receipt))
	case res.Complete:
		fmt.Fprintf(w, "%s: %s is complete\n", verb, res.RunID)
	}
	fmt.Fprintf(w, "next: %s\n", termsafe.Sanitize(fsutil.RedactHome(res.Next)))
}

func newImplementStepCommand(asJSON *bool) *cobra.Command {
	var runID string
	cmd := &cobra.Command{
		Use: "step [--run <run-id>]",
		Long: "Perform the next stage of the run's current lane, write the state, and exit. At a stage\n" +
			"that hands work to an agent, the result names the agent to start, the brief it is handed\n" +
			"and the path its receipt goes to; the lane then advances only on\n" +
			"`abcd implement receipt`, and running `implement step` again re-tells the same thing and\n" +
			"moves nothing. A lane lands one step of the spec; its stages are how it gets there, and\n" +
			"when a lane is done the spec's next pending step opens the next lane, and the run\n" +
			"record names it. A complete run says so.\n\n" +
			"The lane's stages, in order: worktree makes the lane's worktree in the machine-scoped\n" +
			"store, ~/.abcd/worktrees/<root-sha>/<run-id>-<lane-id>, on a branch build/<run-id>-<lane-id>\n" +
			"cut from the default branch; brief renders the lane's brief from that base (the intent,\n" +
			"the spec, the conventions of AGENTS.md, the decisions the intent cites, and the spec\n" +
			"steps before the lane's with what landed each) into the lane's directory of the run;\n" +
			"implement hands the lane to a fresh implementer and awaits\n" +
			"its receipt; validate hands the lane's head to validators that did not implement it, one\n" +
			"fresh agent at a time — a ruthless-reviewer, a security-reviewer and, on the lane whose\n" +
			"landing closes the spec and ships the intent, an intent-auditor over the whole delivery,\n" +
			"from the base of the run's first lane to that lane's head (a lane that does not close the\n" +
			"spec takes no audit) — and records each verdict itself, parsed from the validator's own\n" +
			"return. A round one of them did not pass goes to a fresh implementer, who applies each\n" +
			"finding or rejects it in writing in its report, and the next round judges the new head\n" +
			"afresh; a round that passes completes the stage, unless a lane report states a verdict,\n" +
			"which is refused naming the report. The audit passes only when every criterion is met: a\n" +
			"criterion it could not decide (INCONCLUSIVE) fails the round as a not-met one does, and\n" +
			"goes to the fresh implementer with the finding. A round that does not pass once the lane\n" +
			"has taken the run's fix rounds (--fix-rounds, bundled 3) hands the lane back instead: it\n" +
			"stops as unachievable, the result and the run record name the last round's findings, the\n" +
			"run starts nothing further for it, and every later step is refused naming the hand-back.\n" +
			"land follows a passing round, one step per call: it checks the lane's worktree is clean\n" +
			"at the judged head; on the lane that closes the spec it runs `spec close` in the lane's\n" +
			"worktree and ingests the audit that lane took, and for every capture the lane's receipts\n" +
			"declared fixed it runs `capture resolve` with the lane's commit, committing them on the\n" +
			"lane's branch with Delivers: and Resolves: trailers and an Assisted-by: naming the model\n" +
			"the lane's receipts reported (refused when one reported none), the repository's hooks\n" +
			"running; it pushes the branch only once the\n" +
			"repository's preflight receipt names its head (the pre-push hook runs; nothing is\n" +
			"skipped or forced); it opens the pull request through gh, with a body built from the\n" +
			"records and passed through the outbound scrub, then re-reads the body the forge holds and\n" +
			"strips a session URL or tool footer; it arms auto-merge with the merge-queue method the\n" +
			"ruleset mirror (.abcd/work/rulesets/) names at the lane's base, or leaves the pull request\n" +
			"open where no merge queue gates the default branch, and pushes nothing after that; and\n" +
			"once the pushed head is an ancestor of the default branch on origin it removes the lane's\n" +
			"worktree and branch and the lane is done. Until then the call exits 3 and waits.\n\n" +
			"A stage whose body this abcd does not carry is refused naming the spec piece that\n" +
			"delivers it, and the run is unchanged. A stage that fails leaves the state as it was,\n" +
			"so the next invocation performs it again; a completed stage is never repeated.\n\n" +
			"The run's window clock: once the run's working window has elapsed, the call starts\n" +
			"nothing, writes next_eligible_at (now plus the run's pause) and exits 0 naming it; an\n" +
			"agent already started may still hand back its receipt. Before next_eligible_at the call\n" +
			"is refused as a pause and nothing changes; at or after it, a new window opens.\n\n" +
			"--run names the run; without it, the one run in progress in this checkout. Exit 2 on a\n" +
			"refusal, exit 3 on a pause or a locked run state.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			const prefix = "abcd implement step"
			root, err := loopRoot()
			if err != nil {
				return loopFail(cmd.OutOrStdout(), *asJSON, prefix, err)
			}
			id, err := loop.Resolve(root, runID)
			if err != nil {
				return loopFail(cmd.OutOrStdout(), *asJSON, prefix, err)
			}
			res, err := loop.Advance(root, id, loop.DefaultStages(), loop.Options{})
			if err != nil {
				return loopFail(cmd.OutOrStdout(), *asJSON, prefix, err)
			}
			res.Awaiting = redactAwait(res.Awaiting)
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) { renderStepResult(w, "step", res) })
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "the run to step (run-<16 digits>); the one run in progress when omitted")
	return cmd
}

func newImplementReceiptCommand(asJSON *bool) *cobra.Command {
	var runID string
	cmd := &cobra.Command{
		Use: "receipt <path> [--run <run-id>]",
		Long: "Hand back the receipt the run's awaiting lane named when its stage handed work to an\n" +
			"agent. The path must be the one the stage named. The stage's verifier checks it; a\n" +
			"receipt that verifies completes the stage and the lane moves to its next stage, and one\n" +
			"that does not is refused naming what is missing, with the lane left where it was. A\n" +
			"stage whose verifier this abcd does not carry is refused naming the spec piece that\n" +
			"delivers it.\n\n" +
			"An implementer's receipt is read strictly (one JSON object, no field the brief does not\n" +
			"name, within its size cap, never through a symlink) and verifies only when every commit\n" +
			"it names is on the lane's branch past its base, the definition of done's output exists\n" +
			"in the lane's directory with a zero exit code, and the report exists there. A receipt\n" +
			"that verifies moves the lane's head to its branch's tip. Its optional resolves list names\n" +
			"each capture the lane fixed, with the commit that fixed it (one the receipt names), the\n" +
			"note, the impact and the grounds; the landing resolves each.\n\n" +
			"At the validate stage the receipt is the validator's return: a reviewer's is refused\n" +
			"unless it has one Verdict section stating one verdict of its role (SHIP or FIX FIRST;\n" +
			"APPROVE, BLOCK or NEEDS-INPUT), and the intent-auditor's unless it is the fidelity verdict\n" +
			"the request asked for, echoing its receipt and both provenance hashes. The loop records\n" +
			"the verdict and the lane stays at validate for the next validator. A fresh implementer's\n" +
			"receipt after a round is verified as an implementer's is.\n\n" +
			"--run names the run; without it, the one run in progress in this checkout. Exit 2 on a\n" +
			"refusal, exit 3 on a locked run state.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			const prefix = "abcd implement receipt"
			root, err := loopRoot()
			if err != nil {
				return loopFail(cmd.OutOrStdout(), *asJSON, prefix, err)
			}
			id, err := loop.Resolve(root, runID)
			if err != nil {
				return loopFail(cmd.OutOrStdout(), *asJSON, prefix, err)
			}
			path := args[0]
			if cwd, err := os.Getwd(); err == nil && !filepath.IsAbs(path) {
				path = filepath.Join(cwd, path)
			}
			res, err := loop.Receipt(root, id, path, loop.DefaultStages(), loop.Options{})
			if err != nil {
				return loopFail(cmd.OutOrStdout(), *asJSON, prefix, err)
			}
			res.Awaiting = redactAwait(res.Awaiting)
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) { renderStepResult(w, "receipt", res) })
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "the run the receipt belongs to (run-<16 digits>); the one run in progress when omitted")
	return cmd
}

// newImplementRecordCommand builds `implement record`: the run record read back
// at the end (spc-2609202134338445 piece 10), and the run's transcripts
// captured into the history store, one capture per path.
func newImplementRecordCommand(asJSON *bool) *cobra.Command {
	var runID string
	var transcripts []string
	cmd := &cobra.Command{
		Use: "record [--run <run-id>] [--transcript <path>]...",
		Long: "Render a run's record: every lane with its spec step, branch and head, the implementers'\n" +
			"receipts the loop verified with the model each runner reported, every verdict the loop\n" +
			"recorded from a validator's return, the captures each lane fixed, its pull request and\n" +
			"what its landing did, the transcripts captured into the history store, and the record's\n" +
			"lines. Read-only unless --transcript is given.\n\n" +
			"--transcript <path>, repeatable, captures each transcript into the history store as\n" +
			"`abcd history capture <path>` does, one capture per path, and records it in the run's\n" +
			"state; it is refused on a run that is not complete, since the record's transcripts are\n" +
			"the run's, captured at its end. A capture that fails stops the call: the transcripts\n" +
			"before it are recorded, and the refusal names the failure.\n\n" +
			"--run names the run; without it, the one run in progress, or else the most recently\n" +
			"started run. Exit 2 on a refusal, exit 3 on a locked run state.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			const prefix = "abcd implement record"
			root, err := loopRoot()
			if err != nil {
				return loopFail(cmd.OutOrStdout(), *asJSON, prefix, err)
			}
			id := runID
			if id == "" {
				if id, err = loop.LatestRun(root); err != nil {
					return loopFail(cmd.OutOrStdout(), *asJSON, prefix, err)
				}
			}
			var rec loop.RunRecord
			if len(transcripts) > 0 {
				repoRoot, rootSHA, err := historyStore(cmd)
				if err != nil {
					return loopFail(cmd.OutOrStdout(), *asJSON, prefix, err)
				}
				capture := func(path string) (loop.Transcript, error) {
					res, err := captureTranscriptSource(cmd, repoRoot, rootSHA, path, "", "", "")
					if err != nil {
						return loop.Transcript{}, errors.New(fsutil.RedactHome(err.Error()))
					}
					return loop.Transcript{Path: fsutil.RedactHome(path), Session: res.Record.SessionID, Stored: res.Record.Path, Wrote: res.Wrote}, nil
				}
				rec, err = loop.CaptureTranscripts(root, id, transcripts, capture, loop.Options{})
				if err != nil {
					return loopFail(cmd.OutOrStdout(), *asJSON, prefix, err)
				}
			} else if rec, err = loop.ReadRecord(root, id); err != nil {
				return loopFail(cmd.OutOrStdout(), *asJSON, prefix, err)
			}
			return render(cmd.OutOrStdout(), *asJSON, rec, func(w io.Writer) { renderRunRecord(w, rec) })
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "the run to render (run-<16 digits>); the one in progress, else the latest, when omitted")
	cmd.Flags().StringArrayVar(&transcripts, "transcript", nil, "a transcript to capture into the history store for a complete run (repeatable; one capture per path)")
	return cmd
}

// renderRunRecord is the text form of a run's record.
func renderRunRecord(w io.Writer, rec loop.RunRecord) {
	state := "in progress"
	if rec.Complete {
		state = "complete"
	}
	fmt.Fprintf(w, "run %s  %s (%s)  %s, driven by the %s\n", rec.RunID, termsafe.Sanitize(rec.Key), termsafe.Sanitize(rec.Spec), state, rec.Driver)
	renderPace(w, rec.Pace)
	for _, l := range rec.Lanes {
		fmt.Fprintf(w, "  %s:  spec step %d, %q — %s\n", l.ID, l.SpecStep, termsafe.Sanitize(l.StepTitle), l.Stage)
		if l.Branch != "" {
			fmt.Fprintf(w, "    branch %s (%s..%s)\n", termsafe.Sanitize(l.Branch), shortSHA(l.BaseSHA), shortSHA(l.HeadSHA))
		}
		for _, r := range l.Receipts {
			model := r.Model
			if model == "" {
				model = "none reported"
			}
			fmt.Fprintf(w, "    receipt %s (%s; model %s)\n", termsafe.Sanitize(r.Receipt), termsafe.Sanitize(r.Role), termsafe.Sanitize(model))
		}
		for _, v := range l.Verdicts {
			fmt.Fprintf(w, "    round %d at %s: %s %s\n", v.Round, shortSHA(v.HeadSHA), termsafe.Sanitize(v.Role), termsafe.Sanitize(v.Verdict))
		}
		if len(l.Resolves) > 0 {
			fmt.Fprintf(w, "    resolves %s\n", termsafe.Sanitize(strings.Join(l.Resolves, ", ")))
		}
		renderLanding(w, l.PR, l.Landing)
		if l.HandBack != nil {
			fmt.Fprintf(w, "    handed back as %s after %d fix round(s)\n", termsafe.Sanitize(l.HandBack.Verdict), l.HandBack.FixRounds)
		}
	}
	renderPending(w, rec.Pending)
	fmt.Fprintf(w, "  transcripts: %d captured into the history store\n", len(rec.Transcripts))
	for _, t := range rec.Transcripts {
		how := "stored"
		if !t.Wrote {
			how = "already stored"
		}
		fmt.Fprintf(w, "    %s -> session %s (%s)\n", termsafe.Sanitize(fsutil.RedactHome(t.Path)), termsafe.Sanitize(t.Session), how)
	}
	fmt.Fprintf(w, "  record:  %d line(s)\n", len(rec.Record))
	for _, e := range rec.Record {
		fmt.Fprintf(w, "    %s  %-10s %s  %s\n", e.At.Format("2006-01-02T15:04:05Z"), termsafe.Sanitize(e.Stage),
			termsafe.Sanitize(e.Lane), termsafe.Sanitize(fsutil.RedactHome(e.Note)))
	}
}

// renderLanding renders what a lane's landing has done so far.
func renderLanding(w io.Writer, pr int, ld *loop.Landing) {
	if ld == nil {
		return
	}
	var parts []string
	if ld.Closes {
		parts = append(parts, "closes the spec")
	}
	if ld.Records != "" {
		parts = append(parts, "records "+shortSHA(ld.Records))
	}
	if ld.Pushed != "" {
		parts = append(parts, "pushed "+shortSHA(ld.Pushed))
	}
	if pr > 0 {
		parts = append(parts, fmt.Sprintf("pull request #%d", pr))
	}
	if ld.Merge != "" {
		parts = append(parts, ld.Merge)
	}
	if ld.Merged != "" {
		parts = append(parts, "landed at "+shortSHA(ld.Merged))
	}
	if len(parts) == 0 {
		parts = append(parts, "prepared")
	}
	fmt.Fprintf(w, "    landing: %s\n", termsafe.Sanitize(strings.Join(parts, "; ")))
}
