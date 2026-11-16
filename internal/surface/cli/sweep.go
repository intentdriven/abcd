package cli

// sweep.go — `abcd sweep [--apply]`, the front door of internal/core/sweep
// (itd-2610222214476887, spc-2610230730269295 step 5). Bare, it is the dry
// run: one row per worktree, local branch, branch on origin, reservation and
// waiting inbox report, each with what --apply would do or why it would leave
// the item, and nothing changed but the fetch. --apply acts on what is
// provably spent. The core fetches, judges and acts; this file finds the
// checkout, hands the core its readers and renders the rows.

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/record/match"
	"github.com/intentdriven/abcd/internal/core/sweep"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// sweepOptions is the options the verb hands the core beside the forge,
// the build loop's reader and the working directory: nil in a release, and
// set by a test to fix the probe's clock or process listing. The automatic
// sweep takes them too.
var sweepOptions func(*sweep.Options)

// installAutoSweep installs the automatic sweep (iss-2611150649557869,
// dcn-2611161331159806) in the loop, which runs it after a landing sees its
// pull request merged and at the end of a drain. Each verb that can land a
// lane or end a drain installs it just before it moves the loop; it is not
// installed at init, so no hook, whose git calls must run under the hook's
// deadline, ever reaches the sweep.
func installAutoSweep() { loop.SetAutoSweeper(sweepOnOwn) }

// sweepOnOwn is the automatic sweep of the checkout at repoRoot, with the
// readers `abcd sweep --apply` is handed, from the process's own working
// directory, so the worktree the process is in is kept; it answers the line
// the act it followed reports. It reads sweep.automatic itself: off, or
// outside a repository abcd manages, it removes nothing.
func sweepOnOwn(ctx context.Context, repoRoot string) string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = repoRoot
	}
	o := sweep.Options{Forge: worktreeForge, Held: loop.HeldWorktrees, Cwd: cwd}
	if sweepOptions != nil {
		sweepOptions(&o)
	}
	return sweep.Automatic(ctx, repoRoot, sweep.AutoOptions{Options: o}).Text()
}

// newSweepCommand builds `abcd sweep [--apply]`.
func newSweepCommand(asJSON *bool) *cobra.Command {
	var apply bool
	cmd := &cobra.Command{
		Use: "sweep",
		Long: "List everything this repository has left lying around, and with --apply clear\n" +
			"what is provably spent. It fetches origin first, its branches and the\n" +
			"reservations, then lists one row per worktree of the repository (in the\n" +
			"worktree store and outside it, the store's strays and the embargo store among\n" +
			"them), per local branch, per branch on origin, per reservation branch and per\n" +
			"report waiting in abcd's inbox, each with what --apply would do (remove,\n" +
			"save-patch, delete, dispatch or promote) or why it would keep the item.\n\n" +
			"--apply removes a merged, clean worktree through the store's prune, which\n" +
			"archives its notes first, saves a dirty one's uncommitted work as a patch in\n" +
			"the notes archive and keeps it, deletes a merged local branch with git branch\n" +
			"-d and a merged branch on origin under a lease once nothing still needs it,\n" +
			"dispatches the reservation sweep's workflow when a reservation is spent (it\n" +
			"never deletes one itself), and files each waiting inbox report into the\n" +
			"ledger. Nothing is forced and no process is stopped. Anything dirty,\n" +
			"unmerged, locked, held by a live session, recently written, or that the sweep\n" +
			"cannot classify stays, named with the reason; a check that could not be made\n" +
			"holds every item it could have cleared, and is said once at the top.\n\n" +
			"Exit 0 when the run acted on something (a dry run: when --apply would), 1 when\n" +
			"it completed and acted on nothing, and 2 on a refusal or a fault, outside a\n" +
			"git checkout or when the fetch fails among them.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSweepVerb(cmd, *asJSON, apply)
		},
	}
	cmd.Flags().BoolVar(&apply, "apply", false, "act on what is provably spent; without it the sweep changes nothing but the fetch")
	return cmd
}

// runSweepVerb runs one sweep from the working directory's checkout and
// renders it, exiting with the result's code.
func runSweepVerb(cmd *cobra.Command, asJSON, apply bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return &exitError{Code: 2, Msg: "abcd sweep: " + termsafe.Sanitize(err.Error()) + "; nothing was swept"}
	}
	root, err := gitutil.Toplevel(cwd)
	if err != nil {
		return &exitError{Code: 2, Msg: "abcd sweep: not inside a git checkout; nothing was swept"}
	}
	o := sweep.Options{Forge: worktreeForge, Held: loop.HeldWorktrees, Cwd: cwd}
	if apply {
		o.Match = sweepMatch(cmd.ErrOrStderr(), cwd)
	}
	if sweepOptions != nil {
		sweepOptions(&o)
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	res, runErr := sweep.Run(ctx, root, o, apply)
	if err := render(cmd.OutOrStdout(), asJSON, res, func(w io.Writer) { renderSweepRows(w, res) }); err != nil {
		return err
	}
	switch code := res.ExitCode(); {
	case code == sweep.ExitRefused || runErr != nil:
		msg := res.Refusal
		if msg == "" && runErr != nil {
			msg = runErr.Error()
		}
		return &exitError{Code: sweep.ExitRefused, Msg: "abcd sweep: " + termsafe.Sanitize(msg)}
	case code == sweep.ExitNothing:
		return &exitError{Code: sweep.ExitNothing}
	}
	return nil
}

// sweepMatch is the filing-time match each promoted inbox report runs,
// configured as `inbox promote` configures it; nil when the checkout has no
// ledger, which files each report unmatched.
func sweepMatch(stderr io.Writer, cwd string) *match.Config {
	ledger, err := capture.LedgerRoot(cwd)
	if err != nil {
		return nil
	}
	mc, _ := resolveMatch(stderr, "sweep", ledger)
	return mc
}

// renderSweepRows lays out the notes once, then one line a row: what --apply
// would do (or, applied, what it did), the kind, the name, the path where
// there is one, the state read, and the reason.
func renderSweepRows(w io.Writer, res sweep.Result) {
	head := "abcd sweep"
	if res.Applied {
		head += " --apply"
	} else {
		head += " (dry run: nothing changed but the fetch; --apply acts on the rows that do not say keep)"
	}
	if res.Lane != "" {
		head += ", lane " + res.Lane
	}
	fmt.Fprintln(w, termsafe.Sanitize(head))
	for _, n := range res.Notes() {
		fmt.Fprintln(w, "note: "+termsafe.Sanitize(n))
	}
	if len(res.Rows) == 0 {
		fmt.Fprintln(w, "  (nothing to sweep)")
	}
	for _, r := range res.Rows {
		fmt.Fprintln(w, termsafe.Sanitize("  "+sweep.RowLine(r, res.Applied)))
	}
	if res.Refusal != "" {
		fmt.Fprintln(w, "refused: "+termsafe.Sanitize(res.Refusal))
	}
}
