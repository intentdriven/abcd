package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/core/mode"
	"github.com/intentdriven/abcd/internal/core/statusline"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// modeOutput is the --json envelope of the mode verb. Notice is present only
// on a set that owes somebody an answer where the host has no status surface;
// the print form never carries it.
type modeOutput struct {
	State  mode.State `json:"state"`
	Notice string     `json:"notice,omitempty"`
	// Away is present on the bare print only while an away record holds.
	Away *modeAway `json:"away,omitempty"`
}

// modeAway is what bare `abcd mode` says of an away record that holds: when
// it ends and how many steps wait for the person's return.
type modeAway struct {
	Until     time.Time `json:"until"`
	OpenEnded bool      `json:"open_ended"`
	Queued    int       `json:"queued"`
}

// newModeCommand builds the `mode` verb — the front door onto
// internal/core/mode, the waiting-on state behind the status line's badge
// (spc-70, itd-200). Bare `abcd mode` prints the stored state; `abcd mode
// <state>` sets it. Two writers share the one verb: the agent, which calls it
// when it stops for a verdict and names whom it is addressing, and the human,
// who calls it to say which hat they wear. Both read the same file, so
// neither the render nor the agent can invent an answer of its own.
func newModeCommand(asJSON *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use: "mode [<state>]",
		Long: "Print or set the waiting-on state behind the status line's badge.\n\n" +
			"Bare `abcd mode` prints the stored state: `managed` (abcd is here and nobody\n" +
			"is waiting), `facilitator` (the loop is parked on the facilitator, the person\n" +
			"at the terminal running the agents), or `product-thinker` (the loop is parked\n" +
			"on the product thinker, who answers on a surface of their own). An absent\n" +
			"store reads as `managed`. While an away record holds, a second line says when\n" +
			"the away mode ends and how many steps are queued; --json carries both under\n" +
			"`away`, absent otherwise.\n\n" +
			"`abcd mode <state>` sets it. Two writers share the verb: the agent runs it\n" +
			"when it stops for a verdict other than a question, naming whom it is\n" +
			"addressing, and the human runs it by hand to say which hat they wear. The state lives per checkout at\n" +
			"`.abcd/.work.local/mode`, so only a repository abcd manages — one that has\n" +
			"the local-ephemeral tier — can hold it; elsewhere the set refuses and creates\n" +
			"nothing. The next status-line refresh and the bare `abcd` board read the\n" +
			"same file.\n\n" +
			"A question needs no setting: when the guard admits abcd's question through\n" +
			"the host's question tool, it sets the state from the question's chip\n" +
			"(Product to product-thinker, Tech and Setup to facilitator) and marks the\n" +
			"question open. The answer resets the state to `managed` and says so once on\n" +
			"stderr: the question tool's PostToolUse hook resets it when the answer comes\n" +
			"back, and the prompt hook on the next human message where the host runs no\n" +
			"PostToolUse. A state set by hand with no question open is left as it is.\n\n" +
			"Where this machine has no status surface — no `" + abcdhome.Display("statusline.json") + "`, or\n" +
			"one with `disabled` set — the set form prints one line naming whose answer\n" +
			"is owed, once, because the verb call is the stop. Setting `managed` owes\n" +
			"nobody and prints nothing; with a surface installed nothing is printed at\n" +
			"all. With --json the notice is a field. Both forms make no network request.\n\n" +
			"Exit 2 on a refusal — an unknown state, no local tier, or no checkout —\n" +
			"and nothing is written on any of them.\n\n" +
			"The away mode sits beside the badge, not as a fourth word: `abcd mode\n" +
			"autonomous [--until <time>] [--hold-pushes]` records that the person is away,\n" +
			"`abcd mode queue \"<step>\"` keeps a step for their return, bare `abcd mode\n" +
			"queue` lists what is queued and writes nothing, and `abcd mode back` ends it\n" +
			"and prints what was queued and recorded. While it holds, setting\n" +
			"`product-thinker` or `facilitator` is refused (exit 2), naming `abcd decide`:\n" +
			"a stop parked on a person who is away is a question nobody answers. The away\n" +
			"mode grants no rights; the person's standing instruction decides what is pushed.",
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: stateWords(),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			if len(args) == 0 {
				st, err := mode.Read(cwd)
				if err != nil {
					return modeRefusal(err)
				}
				out := modeOutput{State: st, Away: awayHeld(cmd.ErrOrStderr(), cwd)}
				return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
					fmt.Fprintln(w, st)
					if a := out.Away; a != nil {
						end := a.Until.Local().Format("2006-01-02 15:04 MST")
						if a.OpenEnded {
							end += " (open-ended, seven days at most)"
						}
						fmt.Fprintf(w, "away until %s, %d queued\n", end, a.Queued)
					}
				})
			}
			st, err := mode.ParseState(args[0])
			if err != nil {
				return modeRefusal(err)
			}
			if err := refuseParkWhileAway(cmd.ErrOrStderr(), cwd, st); err != nil {
				return modeRefusal(err)
			}
			if err := mode.Set(cwd, st); err != nil {
				return modeRefusal(err)
			}
			// An explicit set supersedes a question still marked open under
			// another state, so the next message's reset never undoes it; a
			// marker naming the state just set is kept, so a retry that sets
			// the mode again beside the question still resets on the answer.
			// The state is already set; a marker that cannot be cleared is
			// said, never a refusal.
			if root, err := mode.Root(cwd); err == nil {
				if err := mode.ClearQuestionOpenOtherThan(root, st); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "abcd mode: the state is set, but %s\n", termsafe.Sanitize(scrubPaths(err)))
				}
			}
			out := modeOutput{State: st}
			if !statusSurfaceInstalled() {
				out.Notice = answerOwedNotice(st)
			}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				if out.Notice != "" {
					fmt.Fprintln(w, out.Notice)
				}
			})
		},
	}
	cmd.AddCommand(newModeAutonomousCommand(asJSON), newModeBackCommand(asJSON), newModeQueueCommand(asJSON))
	return cmd
}

// refuseParkWhileAway refuses parking the badge on a person while the away
// mode holds for this checkout's repository. Reading the away record fails
// open: one that cannot be read safely reads as not away, and says so in one
// line on errw; a repository with no root commit has no away record at all.
func refuseParkWhileAway(errw io.Writer, cwd string, st mode.State) error {
	if st == mode.Managed {
		return nil
	}
	root, err := mode.Root(cwd)
	if err != nil {
		return nil // mode.Set reports the root refusal itself.
	}
	_, away, err := autonomy.AwayAt(root, autonomy.Options{})
	if err != nil && errors.Is(err, autonomy.ErrUnsafe) {
		fmt.Fprintf(errw, "abcd mode: %v; read as not away\n", err)
	}
	return mode.RefuseParkWhileAway(st, away)
}

// awayHeld reads the away record for cwd's repository for the bare print:
// nil unless one holds. Like refuseParkWhileAway it fails open, saying in one
// line on errw when the record is not safely this account's; a directory
// outside a checkout, or a repository with no root commit, has no record.
func awayHeld(errw io.Writer, cwd string) *modeAway {
	root, err := mode.Root(cwd)
	if err != nil {
		return nil
	}
	rec, away, err := autonomy.AwayAt(root, autonomy.Options{})
	if err != nil {
		if errors.Is(err, autonomy.ErrUnsafe) {
			fmt.Fprintf(errw, "abcd mode: %v; read as not away\n", err)
		}
		return nil
	}
	if !away {
		return nil
	}
	return &modeAway{Until: rec.Ends(), OpenEnded: rec.OpenEnded, Queued: len(rec.Queue)}
}

// stateWords is the closed vocabulary as strings, for shell completion.
func stateWords() []string {
	var words []string
	for _, s := range mode.States() {
		words = append(words, string(s))
	}
	return words
}

// modeRefusal maps the store's refusals to the operand-refusal exit code the
// other stores reserve (decide, intent, capture): an unknown word, an absent
// local tier and an absent checkout are each "this invocation cannot be
// honoured, and nothing was written". Anything else is an I/O failure at the
// default exit 1. The store's own wording is kept: it names the three words,
// or the tier, or the checkout, and it carries no local path.
func modeRefusal(err error) error {
	if errors.Is(err, mode.ErrUnknownState) || errors.Is(err, mode.ErrNoLocalTier) || errors.Is(err, mode.ErrAway) || errors.Is(err, gitutil.ErrNoCheckoutRoot) {
		return &exitError{Code: 2, Msg: "abcd mode: " + err.Error() + " (nothing written)"}
	}
	return fmt.Errorf("abcd mode: %w", err)
}

// statusSurfaceInstalled reports whether this machine renders abcd's status
// line: the user-level setting is present and its off switch is not thrown.
// It is the ac-7 gate — the one-line notice is the fallback for a host with
// no surface, and a load that cannot even resolve the home reads as no
// surface, because a line printed once too often costs a line and a stop that
// goes unnoticed costs the stop.
func statusSurfaceInstalled() bool {
	set, _, err := statusline.Load()
	return err == nil && set.Installed && !set.Disabled
}

// answerOwedNotice is the one line the set form prints where there is no
// status surface: "" for a state that owes nobody an answer.
func answerOwedNotice(st mode.State) string {
	waiting := st.WaitingOn()
	if waiting == "" {
		return ""
	}
	return "abcd: " + waiting + " — an answer is owed"
}
