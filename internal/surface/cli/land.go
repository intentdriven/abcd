package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
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
	var noPush, noEnqueue, approve bool
	var reviews []string
	var restsOn string
	cmd := &cobra.Command{
		Use: "land [<branch>]",
		Long: "abcd land [<branch>] [--no-push] [--no-enqueue]\n" +
			"abcd land [<branch>] --record-review <return> --record-review <return> [...] [--rests-on <base>]\n" +
			"abcd land [<branch>] --approve\n\n" +
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
			"  6. Enqueue (land.enqueue): the branch's open pull request, armed where the merge\n" +
			"     gate allows it, pinned to the head: into the merge queue by its method where\n" +
			"     one gates the default branch (then confirmed in the queue), otherwise as the\n" +
			"     forge's auto-merge (a merge commit; a forge that refuses it is said). An\n" +
			"     arming someone else made at a head the gate arms is left as it is. Before it\n" +
			"     arms, a pull request whose body has a line Depends-On: #N (or #N, #N) is read\n" +
			"     with the pull requests it names, and theirs while open, at most 25 in all: one\n" +
			"     not yet merged stops the step, no new arming made, naming it and when the\n" +
			"     wait is refused (24 hours after the pull request was opened); one closed\n" +
			"     unmerged, one that is no pull request or merges into another branch, a\n" +
			"     cycle, a wait at or past its 24 hours, a line that looks like one in another\n" +
			"     shape, and a forge answer land cannot read whole refuse, exit 2, no new\n" +
			"     arming made, saying what to reword or settle. This account's arming\n" +
			"     already in place is withdrawn on such a stop or refusal. Rebuilding it on\n" +
			"     the default branch once they merge is not done yet: run abcd land again.\n" +
			"     Where a merge queue gates the\n" +
			"     default branch, a branch that shares a commit not on it with another pull\n" +
			"     request queued or armed into it (a branch cut from that one's) is not armed:\n" +
			"     the step stops naming it, since the queue would remove this one; run abcd land\n" +
			"     again after that one merges, or rebase onto the default branch without its\n" +
			"     commits. One whose head is not in the checkout is fetched from origin by\n" +
			"     refs/pull/<n>/head to compare, and a listing or head land cannot read arms\n" +
			"     nothing. With no pull request it composes\n" +
			"     the body from the branch's commits (each subject, their Refs:, Resolves: and\n" +
			"     Delivers: lines, abcd's own Assisted-by label and the models their\n" +
			"     Assisted-by: lines name) into the checkout's .abcd/.work.local/ where it has\n" +
			"     one, judges it as the shell guard judges a posted body, prints the `gh pr\n" +
			"     create` command naming it with --body-file, and stops; it never opens one.\n\n" +
			"The merge gate reads facilitator.mode from the default branch's committed\n" +
			".abcd/config.json and " + abcdhome.Display("config.json") + " (human from either wins;\n" +
			"human by default), never from the branch being landed. A pull request whose\n" +
			"author has Read or Triage permission, or none, is armed only once an approval of\n" +
			"its head counts, in either mode. An approval counts when it is the latest review\n" +
			"by someone with Write, Maintain or Admin permission, approves the head commit\n" +
			"and is not the author's, or when the technical facilitator recorded it with\n" +
			"--approve. Then, and for an author with one of those: human arms only once an\n" +
			"approval of the head counts, and otherwise says it waits for the technical\n" +
			"facilitator's; auto arms only a head a review receipt names (--record-review),\n" +
			"and otherwise names the commit. An author whose permission cannot be read is\n" +
			"unverified, and nothing is armed. human stops an agent merging by mistake, not\n" +
			"one set on getting round it: the recorded approval is a file anything running\n" +
			"under your account can write, so it is a rule your agents keep, not a proof\n" +
			"that you approved.\n\n" +
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
			"A person's stop (abcd implement stop) holds a landing: it is read before anything\n" +
			"runs, again after the gate and again at the arming, and while one holds, or a read\n" +
			"cannot tell whether one does, nothing more is pushed, armed or queued. --approve and\n" +
			"--record-review land nothing and are not held, and a landing with both acts off does\n" +
			"nothing for a stop to hold.\n\n" +
			"Before a push that will move the head of the branch's armed or queued pull request,\n" +
			"land withdraws the arming, reads it back and says so. Where the merge gate refuses\n" +
			"the local head, land withdraws this account's arming at it and refuses over another\n" +
			"account's. Where it cannot make a withdrawal, it refuses, nothing pushed.\n\n" +
			"Exit 0 when every step did its act or said why it stopped. Exit 2 on a refusal,\n" +
			"which names what was done before it; reservations that cannot be fetched are\n" +
			"one, and so is an announcement that cannot be published, each made before\n" +
			"anything is pushed. Exit 3 when a release being cut holds the\n" +
			"landing: it names the release, its holder and until when; and when a stop holds it,\n" +
			"naming who set it and when, or a read cannot tell whether one does, saying why.\n\n" +
			"--record-review records the review pair's verdicts for the branch's head and\n" +
			"lands nothing: given once per reviewer's return (at least two, no two the\n" +
			"same), each with a first line reading Reviewed head: <name>, naming the head by at\n" +
			"least its first nine characters, and one ### Verdict heading and no other line opening a verdict, qualified or not, fenced or not,\n" +
			"where the first line under the heading opens with the verdict (SHIP, APPROVE, FIX\n" +
			"FIRST, BLOCK or NEEDS-INPUT) and names the head by\n" +
			"its full object name (**SHIP** <full name>, or a Head: <full name> line directly\n" +
			"under the heading), it writes the head's review receipt in\n" +
			".abcd/.work.local/review-receipts/ when every verdict passes the stop rule\n" +
			"(SHIP or APPROVE: no critical or major finding). A return that does not pass is\n" +
			"refused naming it, exit 2, and nothing is written. It takes no --no-push or\n" +
			"--no-enqueue. A receipt recorded before a verdict named its head does not count at\n" +
			"the merge gate or as a delta's base: its returns are not kept, so it cannot be read\n" +
			"again, and the head records its pair again.\n\n" +
			"--rests-on <base> makes that a delta review, of a wording fix after a full pair: the\n" +
			"head's receipt rests on the full pair recorded for <base>, a\n" +
			"full object name, and each return's first line reads exactly Reviewed delta:\n" +
			"<base>..<head> in place of the head's, with no Reviewed head: line, its verdict still\n" +
			"naming the head in full. It is refused, and the head needs a full pair, unless the base\n" +
			"has a full pair (a delta's receipt is not one), the head descends from it, and the\n" +
			"branch's own commits since it change only plain text markdown files of mode 100644\n" +
			"under docs/explanation/, docs/how-to/, docs/tutorials/, .abcd/development/plans/,\n" +
			".abcd/development/roadmap/, .abcd/work/issues/resolved/ or .abcd/work/issues/wontfix/,\n" +
			"at most " + strconv.Itoa(landing.MaxDeltaLines) + "\n" +
			"lines, " + strconv.Itoa(landing.MaxDeltaFiles) + " files and 64 KiB of patch. Every changed line is wording alone: letters,\n" +
			"digits and a sentence's punctuation, naming no record and no file, holding no code\n" +
			"or tool's marker and no invisible character, outside any code fence and any ledger\n" +
			"record's grounds. No part of such a path may start with a dot or begin as an instruction file's\n" +
			"name does, every file keeps its frontmatter byte for byte, and no file is added,\n" +
			"removed or moved. Any other path needs a full pair, as code, open issues, decisions,\n" +
			"intents, specs, the brief and the ADRs always do, and so does a corrected id. A merge\n" +
			"of a default-branch commit counts for nothing when it is exactly\n" +
			"the merge git makes of its parents with no conflict; any other merge is\n" +
			"refused. The merge gate asks all of this again at landing. Without --rests-on, a\n" +
			"return carrying a Reviewed delta line is refused.\n\n" +
			"--approve records the technical facilitator's approval of the branch's head and\n" +
			"lands nothing: it writes .abcd/.work.local/approvals/<head>, which the merge\n" +
			"gate counts as an approval of that commit, and of no other. It is the\n" +
			"facilitator's own act, for a pull request they authored (the forge refuses an\n" +
			"author's approval of their own): the shell guard refuses it to an agent.",
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
			if cmd.Flags().Changed("rests-on") && len(reviews) == 0 {
				return &exitError{Code: 2, Msg: "abcd land: --rests-on names the full pair a delta review rests on, so it takes --record-review; nothing was written"}
			}
			if approve {
				if noPush || noEnqueue || len(reviews) > 0 {
					return &exitError{Code: 2, Msg: "abcd land: --approve lands nothing, so it takes no --no-push, --no-enqueue or --record-review; nothing was written"}
				}
				return approveHead(cmd, *asJSON, cwd, args)
			}
			if len(reviews) > 0 {
				if noPush || noEnqueue {
					return &exitError{Code: 2, Msg: "abcd land: --record-review lands nothing, so it takes no --no-push or --no-enqueue; nothing was written"}
				}
				return recordReview(cmd, *asJSON, cwd, args, reviews, restsOn)
			}
			roots, notes := layered.RootsFor(cwd)
			for _, n := range notes {
				fmt.Fprintln(cmd.ErrOrStderr(), noteLine(n))
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
	cmd.Flags().BoolVar(&approve, "approve", false, "record the technical facilitator's approval of the branch's head, a person's act the shell guard refuses to an agent; lands nothing")
	cmd.Flags().StringArrayVar(&reviews, "record-review", nil, "record the review pair's verdicts for the branch's head from this reviewer's return (give it once per return, at least two); lands nothing")
	cmd.Flags().StringVar(&restsOn, "rests-on", "", "make --record-review a delta review resting on the full pair recorded for this commit, given by its full object name; each return also states Reviewed delta: <base>..<head>")
	return cmd
}

// recordReview is `abcd land --record-review`: it reads each return, bounded,
// and records the review receipt for the head of the branch named (or the
// checkout's current branch), refusing with exit 2 when the pair does not pass.
// With restsOn, a base head, the receipt is a delta's.
func recordReview(cmd *cobra.Command, asJSON bool, cwd string, args, paths []string, restsOn string) error {
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
	var receipt string
	if cmd.Flags().Changed("rests-on") {
		receipt, err = landing.RecordDeltaReview(root, sha, restsOn, returns)
	} else {
		receipt, err = landing.RecordReview(root, sha, returns)
	}
	if err != nil {
		return &exitError{Code: 2, Msg: "abcd land: " + termsafe.Sanitize(fsutil.RedactHome(err.Error())) + "; nothing was written"}
	}
	view := struct {
		Commit  string `json:"commit"`
		Receipt string `json:"receipt"`
		RestsOn string `json:"rests_on,omitempty"`
	}{sha, receipt, restsOn}
	return render(cmd.OutOrStdout(), asJSON, view, func(w io.Writer) {
		if restsOn != "" {
			fmt.Fprintf(w, "abcd land: recorded the delta review receipt for %s, resting on the full pair at %s (%s)\n", sha, restsOn, termsafe.Sanitize(fsutil.RedactHome(receipt)))
			return
		}
		fmt.Fprintf(w, "abcd land: recorded the review receipt for %s (%s)\n", sha, termsafe.Sanitize(fsutil.RedactHome(receipt)))
	})
}

// approveHead is `abcd land --approve`: it records the technical
// facilitator's approval of the head of the branch named (or the checkout's
// current branch), which the merge gate counts as an approval of that commit
// (spc-2610161609350616 step 4).
func approveHead(cmd *cobra.Command, asJSON bool, cwd string, args []string) error {
	rev := "HEAD"
	if len(args) == 1 {
		rev = "refs/heads/" + args[0]
	}
	root, err := gitutil.CheckoutRoot(cwd, "the approvals")
	if err != nil {
		return &exitError{Code: 2, Msg: "abcd land: " + termsafe.Sanitize(err.Error()) + "; nothing was written"}
	}
	sha, err := gitutil.Run(root, "rev-parse", "--verify", "--quiet", rev+"^{commit}")
	if err != nil || sha == "" {
		return &exitError{Code: 2, Msg: "abcd land: the branch's head cannot be resolved; nothing was written"}
	}
	rec, err := landing.RecordApproval(root, sha)
	if err != nil {
		return &exitError{Code: 2, Msg: "abcd land: " + termsafe.Sanitize(fsutil.RedactHome(err.Error())) + "; nothing was written"}
	}
	view := struct {
		Commit   string `json:"commit"`
		Approval string `json:"approval"`
	}{sha, rec}
	return render(cmd.OutOrStdout(), asJSON, view, func(w io.Writer) {
		fmt.Fprintf(w, "abcd land: recorded the technical facilitator's approval of %s (%s)\n", sha, termsafe.Sanitize(fsutil.RedactHome(rec)))
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
