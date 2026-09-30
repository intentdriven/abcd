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
// status`, `implement step` and `implement receipt`.

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
	var session, pace, subAgents string
	cmd := &cobra.Command{
		Use: "build <itd-N> [--session <id>] [--pace <work-minutes>/<pause-minutes>] [--sub-agents <n>]",
		Long: "Start the implement loop for one intent, or resume the run already in progress for it.\n" +
			"A new run's checks run first, and every one must pass:\n" +
			"the intent is READY (planned, criteria written, its spec linked and written), asks no\n" +
			"open question, has no unanswered claim section, is not held, names no unsettled blocker\n" +
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
			"A new run is paced: a working window, a pause after it, and a ceiling on the run's lanes\n" +
			"and validators alive at once. The three numbers are read once, when the run starts:\n" +
			"--pace <work-minutes>/<pause-minutes> and --sub-agents <n> for this run, else pace.work_minutes,\n" +
			"pace.pause_minutes and pace.sub_agents in the repository's .abcd/config.json, else in\n" +
			"~/.abcd/config.json, else the bundled 120/300 with 2 sub-agents. The result and the run\n" +
			"record name each number's layer. A malformed pace or ceiling, typed or configured, is\n" +
			"refused naming the value and the accepted form, and writes nothing. Starting again keeps\n" +
			"the run's pace; a flag naming another is refused. The window and the pause bind through\n" +
			"`abcd implement step`; the ceiling is recorded with the run, and this build does not\n" +
			"count lanes against it.\n\n" +
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
	cmd.AddCommand(newBuildNextCommand(asJSON))
	return cmd
}

// newBuildNextCommand builds `abcd build next` (itd-2609211116005482).
func newBuildNextCommand(asJSON *bool) *cobra.Command {
	var session, pace, subAgents string
	var maxPicks int
	var untilEmpty bool
	cmd := &cobra.Command{
		Use: "next [--session <id>] [--pace <work-minutes>/<pause-minutes>] [--sub-agents <n>] [--max <n>] [--until-empty]",
		Long: "Pick the readiest planned intent, write down why, and start its run.\n\n" +
			"The candidates are the planned intents that pass every check `abcd build <itd-N>` runs\n" +
			"(READY, no open question, no unanswered claim section, not held, no unsettled blocker in\n" +
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
			"rule, are refused: that half of the verb is not built in this abcd. --session, --pace and\n" +
			"--sub-agents are `abcd build`'s own.\n\n" +
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
			"its receipt; validate and land follow.\n\n" +
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
			"that verifies moves the lane's head to its branch's tip.\n\n" +
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
