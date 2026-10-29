package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
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
	var reviews []string
	cmd := &cobra.Command{
		Use: "land [<branch>]",
		Long: "abcd land [<branch>] [--no-push] [--no-enqueue]\n" +
			"abcd land [<branch>] --record-review <return> --record-review <return> [...]\n\n" +
			"Land a branch whose work is committed: the checkout's current branch, or the one\n" +
			"named, from the worktree that has it checked out. Each step can be resumed by\n" +
			"running abcd land again.\n\n" +
			"land runs the branch's own `make preflight` with your environment and\n" +
			"credentials, so run it only on a branch whose content you trust.\n\n" +
			"  1. Origin: every fetch and push URL of origin must be the repository the forge\n" +
			"     client resolves, and the forge's default branch must be the one origin\n" +
			"     itself names (git ls-remote --symref origin HEAD) and not the branch;\n" +
			"     checked before anything runs. A stale or missing local origin/HEAD is\n" +
			"     reported and passed over, never written.\n" +
			"  2. Landed: origin's default branch is fetched; a head already on it has landed,\n" +
			"     and so has one the forge lists a merged pull request of this repository's\n" +
			"     branch at (a squash or rebase merge): land says so and stops, pushing,\n" +
			"     gating and queueing nothing. A forge that cannot be asked gets no push.\n" +
			"  3. Gate: when a preflight receipt already names the branch's head (in any\n" +
			"     worktree, as the pre-push hook looks), the gate is skipped; otherwise\n" +
			"     `make preflight` runs in the branch's checkout, its output on stderr, and the\n" +
			"     head must be unchanged, a receipt present and origin's URLs as checked after\n" +
			"     it. A repository whose make file declares no preflight target gets no push.\n" +
			"  4. Announce (with land.push on): before the push, \"merging <item>\" is posted on\n" +
			"     your status branch, as abcd announce posts it, for each issue, intent or pull\n" +
			"     request the branch's own commits name in a Refs:, Resolves: or Delivers:\n" +
			"     trailer, and the push waits until each is on origin. One you already\n" +
			"     announced, standing a minute more, is not sent again; a record not yet on\n" +
			"     the default branch is not announced, and the step says so. One that cannot\n" +
			"     be published refuses the landing, nothing pushed. A push typed by hand with\n" +
			"     git push posts no announcement.\n" +
			"  5. Push (land.push): a plain push of the gated commit to the branch's name on\n" +
			"     origin, with the repository's hooks running, never forced, never the default\n" +
			"     branch.\n" +
			"  6. Enqueue (land.enqueue): the branch's open pull request, armed for the merge\n" +
			"     queue where the merge gate allows it (today: only where a merge queue gates the\n" +
			"     default branch and a ruleset requires a person's approval), pinned to the\n" +
			"     head, then confirmed in the queue. An arming someone else made is left as it\n" +
			"     is. With no pull request it composes the body from the branch's commits (each\n" +
			"     subject, their Refs:, Resolves: and Delivers: lines, abcd's own Assisted-by label\n" +
			"     and the models their Assisted-by: lines name) into the\n" +
			"     checkout's .abcd/.work.local/ where it has one, judges it as the shell guard\n" +
			"     judges a posted body, prints the `gh pr create` command naming it with\n" +
			"     --body-file, and stops; it never opens one.\n\n" +
			"land.push and land.enqueue are read from .abcd/config.json (in the working\n" +
			"directory's checkout and the branch's own) and " + abcdhome.Display("config.json") + ",\n" +
			"default true, and off wins from any layer: any file can turn an act off and none\n" +
			"can turn it back on. --no-push and --no-enqueue turn one off for this run. An act\n" +
			"that is off says which layer turned it off; with both off nothing runs. With\n" +
			"land.push off and land.enqueue on the gate still runs: arming says the head is\n" +
			"ready, and readiness is the gate.\n\n" +
			"Refused on the default branch, a detached HEAD, a tree with uncommitted changes, an\n" +
			"origin that is not the repository the forge client resolves (an SSH host alias\n" +
			"included), and a pull request only another repository opened.\n\n" +
			"Before the push, the reservations and origin's release tags are fetched, and the\n" +
			"release hold is judged for the checkout land runs in: while another session cuts\n" +
			"a release (a live rel-vX.Y.Z reservation, or an unreadable one within its hold),\n" +
			"nothing is pushed, armed or queued. The copy cutting the release lands through\n" +
			"its own hold. A release reserved with a date more than five minutes ahead of\n" +
			"this clock holds from when this checkout first saw it; one this checkout cannot\n" +
			"keep that sighting for (no .abcd/.work.local/) is not counted, and a notice\n" +
			"line on stderr names it.\n\n" +
			"Exit 0 when every step did its act or said why it stopped. Exit 2 on a refusal,\n" +
			"which names what was done before it; reservations that cannot be fetched are\n" +
			"one, and so is an announcement that cannot be published, each made before\n" +
			"anything is pushed. Exit 3 when a release being cut holds the\n" +
			"landing: it names the release, its holder and until when.\n\n" +
			"--record-review records the review pair's verdicts for the branch's head and\n" +
			"lands nothing: given once per reviewer's return (at least two, no two the\n" +
			"same), each naming the head's first nine characters under one ### Verdict\n" +
			"heading, it writes the head's review receipt in\n" +
			".abcd/.work.local/review-receipts/ when every verdict passes the stop rule\n" +
			"(SHIP or APPROVE: no critical or major finding). A return that does not pass is\n" +
			"refused naming it, exit 2, and nothing is written. It takes no --no-push or\n" +
			"--no-enqueue.",
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
			if len(reviews) > 0 {
				if noPush || noEnqueue {
					return &exitError{Code: 2, Msg: "abcd land: --record-review lands nothing, so it takes no --no-push or --no-enqueue; nothing was written"}
				}
				return recordReview(cmd, *asJSON, cwd, args, reviews)
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
			for _, n := range res.Notices {
				fmt.Fprintf(cmd.ErrOrStderr(), "abcd land: %s\n", termsafe.Sanitize(fsutil.RedactHome(n)))
			}
			if err != nil {
				return landRefusal(res, err)
			}
			return render(cmd.OutOrStdout(), *asJSON, landView(res), func(w io.Writer) { renderLand(w, landView(res)) })
		},
	}
	cmd.Flags().BoolVar(&noPush, "no-push", false, "do not push the branch this run; print the push command instead")
	cmd.Flags().BoolVar(&noEnqueue, "no-enqueue", false, "do not arm or queue the pull request this run")
	cmd.Flags().StringArrayVar(&reviews, "record-review", nil, "record the review pair's verdicts for the branch's head from this reviewer's return (give it once per return, at least two); lands nothing")
	return cmd
}

// recordReview is `abcd land --record-review`: it reads each return, bounded,
// and records the review receipt for the head of the branch named (or the
// checkout's current branch), refusing with exit 2 when the pair does not pass.
func recordReview(cmd *cobra.Command, asJSON bool, cwd string, args, paths []string) error {
	rev := "HEAD"
	if len(args) == 1 {
		rev = "refs/heads/" + args[0]
	}
	root, err := gitutil.CheckoutRoot(cwd, "the review receipts")
	if err != nil {
		return &exitError{Code: 2, Msg: "abcd land: " + termsafe.Sanitize(err.Error()) + "; nothing was written"}
	}
	sha, err := gitutil.Run(root, "rev-parse", "--verify", "--quiet", rev+"^{commit}")
	if err != nil || sha == "" {
		return &exitError{Code: 2, Msg: "abcd land: the branch's head cannot be resolved; nothing was written"}
	}
	var returns []landing.ReviewReturn
	for _, p := range paths {
		data, err := fsutil.ReadGuarded(p, landing.MaxReviewReturnBytes)
		if err != nil {
			return &exitError{Code: 2, Msg: "abcd land: the return " + termsafe.Sanitize(fsutil.RedactHome(p)) + " cannot be read: " + termsafe.Sanitize(fsutil.RedactHome(err.Error())) + "; nothing was written"}
		}
		returns = append(returns, landing.ReviewReturn{Name: filepath.Base(p), Text: string(data)})
	}
	receipt, err := landing.RecordReview(root, sha, returns)
	if err != nil {
		return &exitError{Code: 2, Msg: "abcd land: " + termsafe.Sanitize(fsutil.RedactHome(err.Error())) + "; nothing was written"}
	}
	view := struct {
		Commit  string `json:"commit"`
		Receipt string `json:"receipt"`
	}{sha, receipt}
	return render(cmd.OutOrStdout(), asJSON, view, func(w io.Writer) {
		fmt.Fprintf(w, "abcd land: recorded the review receipt for %s (%s)\n", sha, termsafe.Sanitize(fsutil.RedactHome(receipt)))
	})
}

// landRefusal is a refused landing as exit 2, naming the steps done before it;
// one a release being cut holds is contention, exit 3. Reservations that
// cannot be fetched are a refusal, exit 2: nobody is known to hold anything.
func landRefusal(res landing.Result, err error) error {
	msg := "abcd land: " + termsafe.Sanitize(gitutil.ScrubURLCredentials(fsutil.RedactHome(err.Error())))
	code := 2
	var r *landing.Refusal
	if !errors.As(err, &r) {
		msg += "; nothing more was done"
	} else if r.Held {
		code = 3
	}
	var done []string
	for _, s := range res.Steps {
		done = append(done, s.Name+": "+termsafe.Sanitize(s.Note))
	}
	if len(done) > 0 {
		msg += " (before it, " + strings.Join(done, "; ") + ")"
	}
	return &exitError{Code: code, Msg: msg}
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
