package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/intentdriven/abcd/internal/core/matrix"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// newLabMatrixLaunchCommand builds `lab matrix launch`: a consented run's
// runner cells' copies launched in the foreground, each by its cell's runner
// in its own clone (spc-2610161949559050 step 11).
func newLabMatrixLaunchCommand(asJSON *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use: "launch <lab-id> <run-id> [<task>/<cell>/<repeat>]",
		Long: "abcd lab matrix launch <lab-id> <run-id> [<task>/<cell>/<repeat>]\n\n" +
			"Launch, in the foreground, the copies of a consented run that wait for their\n" +
			"cell's runner, at most as many at once as the run's declared concurrency; with a\n" +
			"key, that one copy alone. Each copy's runner is started in its own clone, in the\n" +
			"runners' cell mode, on the blind brief a host cell's copy is made from, with the\n" +
			"cell's route for this launch alone. Nothing stands in for the runner: one that\n" +
			"fails, refuses or answers unparsably is the cell's failure, never handed to the\n" +
			"host; a provider down before the copy began is tried again on the same pin and\n" +
			"recorded as not started, never scored. A provider-bound clone passes the secret\n" +
			"scan again first. The brief and the receipt sit in the hand-off store, outside\n" +
			"the clone, the clone store and the lab's state. A made copy is checked for\n" +
			"compliance and written as its trial through the secret screen, as a host copy's\n" +
			"hand-back is; a made code copy waits for its hidden tests' result, handed back\n" +
			"with `abcd lab matrix copy --receipt`. A metered copy whose declared cost the\n" +
			"spending cap's remainder does not cover is left waiting and named.\n\n" +
			"An interrupt, a termination or a hangup stops the launch: every runner at work\n" +
			"is killed with its process group, and each copy it was making is left\n" +
			"interrupted, which is the run's error and never the copy's score; such a copy is\n" +
			"not launched again. A launch killed outright leaves its copies interrupted too,\n" +
			"once its run's launch lock is free.\n\n" +
			"Refused, exit 2, with nothing launched: a key that is not one, a copy the run\n" +
			"did not hand off, a host cell's copy, a refused cell's, one already launched,\n" +
			"and a cell whose route, source or pin on this machine is not the one the run\n" +
			"recorded. A launch stopped part-way exits 2 and says what it wrote. One launch\n" +
			"per run runs at a time: a second, while one holds the run's launch lock, is\n" +
			"refused at exit 3, naming who holds it. A halted lab exits 1.",
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			const verb = "abcd lab matrix launch"
			root, err := labRoot(verb)
			if err != nil {
				return err
			}
			home, err := matrixHome(verb)
			if err != nil {
				return err
			}
			var only *matrix.TrialKey
			if len(args) == 3 {
				key, err := matrix.ParseTrialKey(args[2])
				if err != nil {
					return verbFailure(verb, err)
				}
				only = &key
			}
			parent := cmd.Context()
			if parent == nil {
				parent = context.Background()
			}
			// A hangup (the terminal closed, the session dropped) stops the
			// launch as an interrupt does: the runners lead process groups of
			// their own, outside the terminal's, so nothing else would end them.
			ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
			defer stop()
			res, err := matrix.LaunchCopies(ctx, root, home, args[0], args[1], only)
			for i := range res.Copies {
				res.Copies[i].Reason = fsutil.RedactHome(res.Copies[i].Reason)
			}
			if errors.Is(err, matrix.ErrLaunchHeld) {
				return matrixHeld(verb, err)
			}
			if err != nil {
				if len(res.Copies) > 0 {
					renderLaunch(cmd.ErrOrStderr(), verb, res)
				}
				return verbFailure(verb, err)
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) { renderLaunch(w, verb, res) })
		},
	}
	return cmd
}

// renderLaunch prints each copy's launch, every field read back from the
// lab's state escaped.
func renderLaunch(w io.Writer, verb string, res matrix.LaunchResult) {
	if len(res.Copies) == 0 {
		fmt.Fprintf(w, "%s %s: no copy waits for its cell's runner\n", verb, termsafe.Sanitize(res.Run))
		return
	}
	fmt.Fprintf(w, "%s %s: %d copies\n", verb, termsafe.Sanitize(res.Run), len(res.Copies))
	for _, c := range res.Copies {
		status := string(c.Status)
		if status == "" {
			status = "-"
		}
		fmt.Fprintf(w, "  %-28s %-12s %-11s %s\n", termsafe.Sanitize(c.Key.String()), termsafe.Sanitize(string(c.State)),
			termsafe.Sanitize(status), termsafe.Sanitize(c.Reason))
	}
}
