package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/embargo"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// announceStore is the noun the checkout resolution names in its refusal.
const announceStore = "the announcement"

// announceAct is the core act the verb drives; a test stands in for it to
// reach a failure no real checkout produces.
var announceAct = claim.Announce

// announceOutput is the --json document of one announcement that went out.
type announceOutput struct {
	Kind      string `json:"kind"`
	Item      string `json:"item"`
	Sender    string `json:"sender"`
	Branch    string `json:"branch"`
	SentAt    string `json:"sent_at"`
	ExpiresAt string `json:"expires_at"`
	Commit    string `json:"commit"`
	Attempts  int    `json:"attempts"`
}

// newAnnounceCommand builds the `announce` verb — the front door onto
// claim.Announce (itd-2609151838312703, spc-2610150728396006): a session posts
// a short fixed-form announcement on the shared remote, which every other
// session, under any account, on any machine or in a hosted session, can read.
// The verb is an agent's, beside `reserve` (the product thinker's ruling of
// 2026-10-15, decision 21).
func newAnnounceCommand(asJSON *bool) *cobra.Command {
	var life time.Duration
	cmd := &cobra.Command{
		Use: "announce <kind> <item>",
		Long: "abcd announce <kind> <item> [--expires-in <duration>]\n\n" +
			"Post a short fixed-form announcement on the shared remote, so every other\n" +
			"session sees what this one is doing. The kind is one of " + strings.Join(claim.AnnouncementKinds, ", ") + ";\n" +
			"the item is the record it concerns (iss-N or itd-N) or a pull request (#N, or\n" +
			"the bare number, since a shell reads #N as a comment). --expires-in sets how\n" +
			"long it stands (default 30m, from 1m to 24h). An announcement carries no free text: a\n" +
			"question points at the item, and the conversation happens there.\n\n" +
			"The announcement is one commit that changes no file on your own status branch,\n" +
			"abcd-status/<your GitHub username in lower case> on the remote named origin,\n" +
			"carrying the kind, the item, you as sender, this working copy, the time sent\n" +
			"and the expiry. The verb lists the status branches on origin (git ls-remote),\n" +
			"fetches each one named by a lower-case GitHub username, with origin's default\n" +
			"branch, then makes one plain fast-forward push with the repository's hooks\n" +
			"running. A push that lost a race to another of your sessions is rebuilt on the\n" +
			"new tip and pushed again, up to three times; then it says \"not sent\" and why.\n" +
			"Nothing is ever forced.\n\n" +
			"A record must already be on origin's default branch, since an announcement is\n" +
			"public; a record the embargo store on this machine holds is refused, and so is\n" +
			"every announcement when that store cannot be checked. The sender is worked out\n" +
			"as reserve's holder is: from user.email when it is a GitHub no-reply address,\n" +
			"then reservation.github_username in the machine configuration, then the\n" +
			"session's forge token, asked whose account it is. When git's identity is not a\n" +
			"person's, the token's account sends and authors the announcement, and the\n" +
			"identity git had is shown nowhere.\n\n" +
			"Exit 0 when the announcement reached origin; exit 2 on every refusal, when it\n" +
			"was not sent, and when a push was made but whether it landed cannot be told.\n" +
			"Nothing is pushed on a refusal except \"not sent\", which follows pushes origin\n" +
			"refused; the unknown outcome says a refused push may still have reached origin.",
		Args: func(_ *cobra.Command, args []string) error {
			switch {
			case len(args) < 2:
				return &exitError{Code: 2, Msg: "abcd announce: name a kind and an item (abcd announce <kind> <item>), the kind one of " +
					strings.Join(claim.AnnouncementKinds, ", ") + "; nothing was pushed"}
			case len(args) > 2:
				return &exitError{Code: 2, Msg: "abcd announce: takes a kind and an item, and no free text; a question points at the item, " +
					"and the conversation happens there; nothing was pushed"}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := gitutil.CheckoutRoot(cwd, announceStore)
			if err != nil {
				return &exitError{Code: 2, Msg: "abcd announce: " + err.Error() + "; nothing was pushed"}
			}
			res, err := announceAct(root, claim.AnnounceRequest{Kind: args[0], Item: args[1], For: life},
				claim.Options{Embargo: announceEmbargoCheck, ForgeUser: reserveForgeUser})
			if err != nil {
				return announceRefusal(err)
			}
			out := announceOutput{Kind: res.Kind, Item: res.Item, Sender: res.Sender, Branch: res.Branch,
				SentAt: rfc3339(res.SentAt), ExpiresAt: rfc3339(res.ExpiresAt), Commit: res.Commit, Attempts: res.Attempts}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) { renderAnnounce(w, out, res.ExpiresAt.Sub(res.SentAt)) })
		},
	}
	cmd.Flags().DurationVar(&life, "expires-in", 0, "how long the announcement stands, from 1m to 24h (default 30m)")
	return cmd
}

// announceRefusal maps a failed announcement to exit 2, as reserveRefusal
// does. A refusal already says what became of the push (nothing pushed, no
// announcement sent, or not sent and why), so it is passed through; any other
// failure is told what the core counted: nothing was pushed when no push was
// tried, and otherwise that the pushes were refused but one may still have
// reached origin, so the branch is looked at before announcing again.
func announceRefusal(err error) error {
	msg := "abcd announce: " + termsafe.Sanitize(fsutil.RedactHome(err.Error()))
	if errors.Is(err, claim.ErrRefused) {
		return &exitError{Code: 2, Msg: msg}
	}
	var ae *claim.AnnounceError
	if errors.As(err, &ae) && ae.Pushes > 0 {
		branch := termsafe.Sanitize(ae.Branch)
		pushes := fmt.Sprintf("%d earlier pushes", ae.Pushes)
		if ae.Pushes == 1 {
			pushes = "1 earlier push"
		}
		return &exitError{Code: 2, Msg: fmt.Sprintf("%s; not sent as far as git reported, but origin refused %s to %s "+
			"and a refused push may still have reached it: fetch origin and look at %s before announcing again", msg, pushes, branch, branch)}
	}
	return &exitError{Code: 2, Msg: msg + "; not sent, and nothing was pushed"}
}

// announceEmbargoCheck is the embargo check an announcement is held to (C5):
// `abcd embargo check` on its whole message and the record it names. Any
// finding refuses it, and a check that cannot run is an error, which refuses
// it too.
func announceEmbargoCheck(root, home, text string, records []string) (bool, error) {
	if home == "" {
		return false, fmt.Errorf("the home directory cannot be named, so the embargo store cannot be found")
	}
	res, err := embargo.Check(embargo.CheckRequest{Home: home, RepoRoot: root, Files: []string{"-"},
		Stdin: strings.NewReader(text), Records: records})
	if err != nil {
		return false, err
	}
	return !res.Clean(), nil
}

// renderAnnounce is the text form: one line saying what went out, as whom, on
// which branch, and until when.
func renderAnnounce(w io.Writer, out announceOutput, life time.Duration) {
	fmt.Fprintf(w, "abcd announce: sent %q as %s on %s, standing until %s (%s)\n",
		out.Kind+" "+out.Item, out.Sender, out.Branch, out.ExpiresAt, life)
}
