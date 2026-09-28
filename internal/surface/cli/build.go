package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
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
// envelope: the step, the reason and the remedy as fields (criterion 13), and
// every pre-start check's row when the refusal is a check's.
type loopRefusalDoc struct {
	Refusal loop.Refusal `json:"refusal"`
}

// loopFail maps an error from the loop to the process outcome. A refusal exits
// 2, and contention (a peer holds the record, the run is paused or locked) exits
// 3: back off and take other work, as `abcd implement` does. Under --json the
// refusal is rendered as its own document first, so a machine reads the step,
// the reason and the remedy as fields; the envelope follows it, last on the
// stream as every refusal is.
func loopFail(w io.Writer, asJSON bool, prefix string, err error) error {
	if r, ok := loop.AsRefusal(err); ok {
		red := *r
		red.Reason = fsutil.RedactHome(red.Reason)
		red.Remedy = fsutil.RedactHome(red.Remedy)
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
	return &cobra.Command{
		Use: "build <itd-N>",
		Long: "Start the implement loop for one intent, or resume the run already in progress for it.\n" +
			"A new run's checks run first, and every one must pass:\n" +
			"the intent is READY (planned, criteria written, its spec linked and written), asks no\n" +
			"open question, has no unanswered claim section, is not held, its spec leaves a step to\n" +
			"build, and no peer holds it (no sibling worktree or local branch holds it in another\n" +
			"bucket, and no session holds a live claim on it; a peer or claim that cannot be read\n" +
			"counts as holding it). A refusal names the check, the reason\n" +
			"and the remedy, and writes nothing.\n\n" +
			"When the checks pass, the run is created in this checkout's local tier,\n" +
			"`.abcd/.work.local/run/<run-id>/state.json`: one lane for the spec's first unlanded step,\n" +
			"the other unlanded steps pending, and the run record's first line. The tier itself is\n" +
			"never created: only a repository abcd manages has one. Starting again while the run is\n" +
			"in progress creates nothing and names the run without judging the checks again (the\n" +
			"run's own lanes change what they read), so a killed process resumes where it stopped.\n\n" +
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
			res, err := loop.Start(root, args[0], loop.Options{})
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
				renderLaneLine(w, res.Lane)
				renderPending(w, res.Pending)
				fmt.Fprintf(w, "next: %s\n", termsafe.Sanitize(fsutil.RedactHome(res.Next)))
			})
		},
	}
}

// renderLaneLine renders one lane as a line, and its footprint once its steps
// have made one.
func renderLaneLine(w io.Writer, l loop.Lane) {
	fmt.Fprintf(w, "  %s:  spec step %d, %q — next: %s\n", l.ID, l.SpecStep, termsafe.Sanitize(l.StepTitle), l.Step)
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
			"intent and spec, each lane with its spec step and next step, what an awaiting lane\n" +
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
					if st.NextEligibleAt != nil {
						fmt.Fprintf(w, "  paused until %s\n", st.NextEligibleAt.UTC().Format("2006-01-02T15:04:05Z07:00"))
					}
					for _, l := range st.Lanes {
						renderLaneLine(w, l)
					}
					renderPending(w, st.Pending)
					fmt.Fprintf(w, "  record:  %d line(s)\n", len(st.Record))
					for _, e := range st.Record {
						fmt.Fprintf(w, "    %s  %-9s %s  %s\n", e.At.Format("2006-01-02T15:04:05Z"), termsafe.Sanitize(e.Step),
							termsafe.Sanitize(e.Lane), termsafe.Sanitize(fsutil.RedactHome(e.Note)))
					}
				}
			})
		},
	}
	cmd.Flags().StringVar(&runID, "run", "", "the run to render (run-<16 digits>); every run in this checkout when omitted")
	return cmd
}

// renderStepResult is the text form of a step or receipt result.
func renderStepResult(w io.Writer, verb string, res loop.StepResult) {
	switch {
	case res.Performed != "":
		fmt.Fprintf(w, "%s: %s completed %s's %s step\n", verb, res.RunID, res.Lane, res.Performed)
	case res.Awaiting != nil:
		fmt.Fprintf(w, "%s: %s's %s step awaits the %s's receipt\n", verb, res.Lane, res.Step, termsafe.Sanitize(res.Awaiting.Role))
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
		Long: "Perform one step of the run's current lane, write the state, and exit. At a step that\n" +
			"hands work to an agent, the result names the agent to start, the brief it is handed\n" +
			"and the path its receipt goes to; the lane then advances only on\n" +
			"`abcd implement receipt`, and asking for a step again re-tells the same thing and\n" +
			"moves nothing. When a lane is done the spec's next pending step opens the next lane.\n" +
			"A complete run says so.\n\n" +
			"The lane's steps, in order: worktree makes the lane's worktree in the machine-scoped\n" +
			"store, ~/.abcd/worktrees/<root-sha>/<run-id>-<lane-id>, on a branch build/<run-id>-<lane-id>\n" +
			"cut from the default branch; brief renders the lane's brief from that base (the intent,\n" +
			"the spec, the conventions of AGENTS.md and the decisions the intent cites) into the\n" +
			"lane's directory of the run; implement hands the lane to a fresh implementer and awaits\n" +
			"its receipt; validate and land follow.\n\n" +
			"A step whose body this abcd does not carry is refused naming the spec piece that\n" +
			"delivers it, and the run is unchanged. A step that fails leaves the state as it was,\n" +
			"so the next invocation performs it again; a completed step is never repeated. Before\n" +
			"the run's next_eligible_at the step is refused as a pause.\n\n" +
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
			res, err := loop.Advance(root, id, loop.DefaultSteps(), loop.Options{})
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
		Long: "Hand back the receipt the run's awaiting lane named when its step handed work to an\n" +
			"agent. The path must be the one the step named. The step's verifier checks it; a\n" +
			"receipt that verifies completes the step and the lane moves to its next step, and one\n" +
			"that does not is refused naming what is missing, with the lane left where it was. A\n" +
			"step whose verifier this abcd does not carry is refused naming the spec piece that\n" +
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
			res, err := loop.Receipt(root, id, path, loop.DefaultSteps(), loop.Options{})
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
