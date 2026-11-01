package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// newImplementWidenCommand builds `implement widen`: an issue lane's
// implementer widens the run's reservation to a further path before its first
// edit there, backing off another copy's live hold (iss-2610311224444576;
// dcn-2610311544325397).
func newImplementWidenCommand(asJSON *bool) *cobra.Command {
	var runID, laneID string
	cmd := &cobra.Command{
		Use: "widen <path>... --lane <lane-id> [--run <run-id>]",
		Long: "Widen an issue run's reservation to each path named before the lane's implementer\n" +
			"edits it: the issue is reserved again on origin with the footprint its reservation holds\n" +
			"and the paths it does not cover yet, judged for overlap and against the cap like any\n" +
			"reservation. A path already covered adds nothing, and when every one is, nothing is\n" +
			"pushed. The run record says what was widened. Only the run's current lane, at its\n" +
			"implement or validate stage, widens; an intent's run holds its spec's footprint and is\n" +
			"refused. A run holding no reservation has nothing to widen and says so.\n\n" +
			"An implementer's receipt is refused while the lane's branch changes a path the\n" +
			"reservation does not cover, naming this command, so a widening forgotten or an edit made\n" +
			"through the shell is caught before the lane moves on.\n\n" +
			"Run it from the checkout the run lives in; the lane's brief names the command in full.\n" +
			"--run names the run; without it, the one run in progress in this checkout. Exit 2 on a\n" +
			"refusal; exit 3 when another copy holds a live reservation on one of the paths, with\n" +
			"nothing pushed (edit none of them, and stop without a receipt: the lane is handed to an\n" +
			"implementer again on a later step), or on a locked run state.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			const prefix = "abcd implement widen"
			if laneID == "" {
				return &exitError{Code: 2, Msg: prefix + ": --lane is required: name the lane the brief names"}
			}
			root, err := loopRoot()
			if err != nil {
				return loopFail(cmd.OutOrStdout(), *asJSON, prefix, err)
			}
			id, err := loop.Resolve(root, runID)
			if err != nil {
				return loopFail(cmd.OutOrStdout(), *asJSON, prefix, err)
			}
			res, err := loop.Widen(root, id, laneID, args, loop.Options{})
			if err != nil {
				return loopFail(cmd.OutOrStdout(), *asJSON, prefix, err)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) { renderWiden(w, res) })
		},
	}
	cmd.Flags().StringVar(&laneID, "lane", "", "the lane whose implementer is about to edit the paths (lane-<n>)")
	cmd.Flags().StringVar(&runID, "run", "", "the run the lane belongs to (run-<16 digits>); the one run in progress when omitted")
	return cmd
}

// renderWiden is a widening's text form.
func renderWiden(w io.Writer, res loop.WidenResult) {
	if len(res.Added) == 0 {
		fmt.Fprintf(w, "widen: %s\n", termsafe.Sanitize(res.Note))
		return
	}
	fmt.Fprintf(w, "widen: %s's %s now holds %s on %s\n", res.RunID, res.Lane,
		termsafe.Sanitize(strings.Join(res.Added, ", ")), res.Record)
	fmt.Fprintf(w, "  footprint: %s\n", termsafe.Sanitize(strings.Join(res.Footprint, ", ")))
}
