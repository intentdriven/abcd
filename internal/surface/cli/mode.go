package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/intentdriven/abcd/internal/core/mode"
	"github.com/intentdriven/abcd/internal/core/statusline"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/spf13/cobra"
)

// modeOutput is the --json envelope of the mode verb. Notice is present only
// on a set that owes somebody an answer where the host has no status surface;
// the print form never carries it.
type modeOutput struct {
	State  mode.State `json:"state"`
	Notice string     `json:"notice,omitempty"`
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
			"store reads as `managed`.\n\n" +
			"`abcd mode <state>` sets it. Two writers share the verb: the agent runs it\n" +
			"when it stops for a verdict, naming whom it is addressing, and the human runs\n" +
			"it by hand to say which hat they wear. The state lives per checkout at\n" +
			"`.abcd/.work.local/mode`, so only a repository abcd manages — one that has\n" +
			"the local-ephemeral tier — can hold it; elsewhere the set refuses and creates\n" +
			"nothing. The next status-line refresh and the bare `abcd` board read the\n" +
			"same file.\n\n" +
			"Where this machine has no status surface — no `~/.abcd/statusline.json`, or\n" +
			"one with `disabled` set — the set form prints one line naming whose answer\n" +
			"is owed, once, because the verb call is the stop. Setting `managed` owes\n" +
			"nobody and prints nothing; with a surface installed nothing is printed at\n" +
			"all. With --json the notice is a field. Both forms make no network request.\n\n" +
			"Exit 2 on a refusal — an unknown state, no local tier, or no checkout —\n" +
			"and nothing is written on any of them.",
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
				return render(cmd.OutOrStdout(), *asJSON, modeOutput{State: st}, func(w io.Writer) {
					fmt.Fprintln(w, st)
				})
			}
			st, err := mode.ParseState(args[0])
			if err != nil {
				return modeRefusal(err)
			}
			if err := mode.Set(cwd, st); err != nil {
				return modeRefusal(err)
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
	return cmd
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
	if errors.Is(err, mode.ErrUnknownState) || errors.Is(err, mode.ErrNoLocalTier) || errors.Is(err, gitutil.ErrNoCheckoutRoot) {
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
	who := st.Addressee()
	if who == "" {
		return ""
	}
	return "abcd: waiting on the " + who + " — an answer is owed"
}
