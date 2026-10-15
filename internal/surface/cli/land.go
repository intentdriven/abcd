package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// landSeams are the core's seams the surface tests replace; nil is the real
// forge client, networked git and `make preflight`.
var landSeams landing.Options

// newLandCommand builds the `land` verb — the front door onto
// internal/core/landing (iss-2610150524235929, iss-2610150524246898): one
// landing path for any branch whose work is committed. It gates the branch's
// head on a preflight receipt, pushes the branch to origin, and arms its open
// pull request's merge where the merge gate allows, each step resumable by
// running the verb again.
func newLandCommand(asJSON *bool) *cobra.Command {
	var noPush, noEnqueue bool
	cmd := &cobra.Command{
		Use: "land [<branch>]",
		Long: "abcd land [<branch>] [--no-push] [--no-enqueue]\n\n" +
			"Land a branch whose work is committed: the checkout's current branch, or the one\n" +
			"named, from the worktree that has it checked out. Each step can be resumed by\n" +
			"running abcd land again.\n\n" +
			"  1. Gate: when a preflight receipt already names the branch's head (in any\n" +
			"     worktree, as the pre-push hook looks), the gate is skipped; otherwise\n" +
			"     `make preflight` runs in the branch's checkout, its output on stderr, and the\n" +
			"     head must be unchanged and a receipt present after it. A repository whose\n" +
			"     make file declares no preflight target gets no push.\n" +
			"  2. Push (land.push): a plain push of the branch to the same name on origin,\n" +
			"     with the repository's hooks running, never forced, never the default branch.\n" +
			"  3. Enqueue (land.enqueue): the branch's open pull request, armed for the merge\n" +
			"     queue where the merge gate allows it (today: only where a merge queue gates the\n" +
			"     default branch and a ruleset requires a person's approval), then confirmed in\n" +
			"     the queue. With no pull request it prints the `gh pr create` command and stops;\n" +
			"     it never opens one.\n\n" +
			"land.push and land.enqueue are read from .abcd/config.json and\n" +
			abcdhome.Display("config.json") + ", default true, and off wins from any layer: either file\n" +
			"can turn an act off and neither can turn it back on. --no-push and --no-enqueue turn\n" +
			"one off for this run. An act that is off says which layer turned it off.\n\n" +
			"Refused on the default branch, a detached HEAD, a tree with uncommitted changes, an\n" +
			"origin that is not the repository the forge client resolves, and a pull request\n" +
			"from another repository. Exit 0 when every step did its act or said why it\n" +
			"stopped; exit 2 on a refusal, which names what was done before it.",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 1 {
				return &exitError{Code: 2, Msg: "abcd land: takes at most one branch, not " + fmt.Sprint(len(args)) + "; nothing was pushed"}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			roots, notes := layered.RootsFor(cwd)
			for _, n := range notes {
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd land: %s\n", termsafe.Sanitize(fsutil.RedactHome(n)))
			}
			o := landSeams
			o.Roots, o.NoPush, o.NoEnqueue = roots, noPush, noEnqueue
			o.GateOutput = cmd.ErrOrStderr()
			if len(args) == 1 {
				o.Branch = args[0]
			}
			res, err := landing.Land(cwd, o)
			if err != nil {
				return landRefusal(res, err)
			}
			return render(cmd.OutOrStdout(), *asJSON, landView(res), func(w io.Writer) { renderLand(w, landView(res)) })
		},
	}
	cmd.Flags().BoolVar(&noPush, "no-push", false, "do not push the branch this run; print the push command instead")
	cmd.Flags().BoolVar(&noEnqueue, "no-enqueue", false, "do not arm or queue the pull request this run")
	return cmd
}

// landRefusal is a refused landing as exit 2, naming the steps done before it.
func landRefusal(res landing.Result, err error) error {
	msg := "abcd land: " + termsafe.Sanitize(fsutil.RedactHome(err.Error()))
	var r *landing.Refusal
	if !errors.As(err, &r) {
		msg += "; nothing more was done"
	}
	var done []string
	for _, s := range res.Steps {
		done = append(done, s.Name+": "+termsafe.Sanitize(s.Note))
	}
	if len(done) > 0 {
		msg += " (before it, " + strings.Join(done, "; ") + ")"
	}
	return &exitError{Code: 2, Msg: msg}
}

// landView is a result as a surface shows it, every value a repository or the
// forge supplied sanitised for the terminal.
func landView(res landing.Result) landing.Result {
	out := res
	out.Branch = termsafe.Sanitize(res.Branch)
	out.Checkout = termsafe.Sanitize(res.Checkout)
	out.Default = termsafe.Sanitize(res.Default)
	out.PRURL = termsafe.Sanitize(res.PRURL)
	out.Steps = make([]landing.Step, len(res.Steps))
	for i, s := range res.Steps {
		out.Steps[i] = landing.Step{Name: s.Name, Status: s.Status, Note: termsafe.Sanitize(s.Note), Command: termsafe.Sanitize(s.Command)}
	}
	return out
}

// renderLand is the text form: the branch and head, then one line per step,
// with the command to run under a step that prints one.
func renderLand(w io.Writer, res landing.Result) {
	fmt.Fprintf(w, "abcd land: %s at %s (into %s)\n", res.Branch, shortCommit(res.Head), res.Default)
	for _, s := range res.Steps {
		fmt.Fprintf(w, "  %-8s %-8s %s\n", s.Name, s.Status, s.Note)
		if s.Command != "" {
			fmt.Fprintf(w, "           run: %s\n", s.Command)
		}
	}
}
