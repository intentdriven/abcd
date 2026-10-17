package cli

// update_news.go is the front door of the moment after an abcd update
// (spc-2610170635097509, step 7): `abcd update news` prints what the agent
// relays at the start of its first reply, the one line per release saying
// what changed and the offer to run setup again; --full prints the full
// summary; --answer records the person's answer to the offer and --seen that
// the news was read. Bare and --full write nothing. The words are core's
// (internal/core/afterupdate): numbers, checked release tags, core's titles
// and the digest inside the binary, never the repository's text.
//
// The setup a yes runs, `ahoy install --only-waiting`, is ahoy's
// (ahoy_only_waiting.go).

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/afterupdate"
	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// newsRecorded is what --answer and --seen render.
type newsRecorded struct {
	Answer   string `json:"answer,omitempty"`
	Release  string `json:"release"`
	NewsSeen bool   `json:"news_seen"`
	Next     string `json:"next,omitempty"`
}

func newUpdateNewsCommand(asJSON *bool) *cobra.Command {
	var full, seen bool
	var answer string
	cmd := &cobra.Command{
		Use:  "news",
		Args: cobra.NoArgs,
		Long: "After an abcd update, print what the agent relays at the start of its first reply: " +
			"one line per release this computer has not been shown, saying how many changes it made and which need the person to act, " +
			"and, where the update brings this repository setup it never answered, the offer to run setup again. " +
			"It prints nothing when there is nothing to say. --full prints every change of the releases since the last update, " +
			"those that need action first, each with the record it cites, and what waits here. " +
			"Neither writes anything. Only a person's reply is recorded: --answer yes|not-now records the answer to the offer " +
			"for this repository and release, once, and --seen records that the news was read where no offer was made, both in " +
			"abcd's home folder. On yes, run " + onlyWaitingCommand + ". The text comes only from the binary's own digest and " +
			"checked release numbers, and nothing is fetched.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			answered := cmd.Flags().Changed("answer")
			switch {
			case answered && (full || seen):
				return &exitError{Code: 2, Msg: "abcd update news: --answer records the reply to the offer and takes neither --full nor --seen; nothing was written"}
			case full && seen:
				return &exitError{Code: 2, Msg: "abcd update news: --full shows the summary and writes nothing, so it takes no --seen; nothing was written"}
			case answered && answer != afterupdate.AnswerYes && answer != afterupdate.AnswerNotNow:
				return &exitError{Code: 2, Msg: "abcd update news: --answer is yes or not-now; nothing was written"}
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			switch {
			case answered:
				return runUpdateNewsAnswer(cmd, cwd, answer, *asJSON)
			case seen:
				return runUpdateNewsSeen(cmd, *asJSON)
			case full:
				f, err := afterupdate.Full(cwd, "")
				if err != nil {
					return updateNewsRefusal(err)
				}
				return render(cmd.OutOrStdout(), *asJSON, f, func(w io.Writer) { printUpdateNewsLines(w, f.Lines) })
			}
			m, err := afterupdate.Moment(cwd)
			if err != nil {
				return err
			}
			return render(cmd.OutOrStdout(), *asJSON, m, func(w io.Writer) { printUpdateNewsLines(w, m.Lines) })
		},
	}
	cmd.Flags().BoolVar(&full, "full", false, "print the full summary of the releases since the last update and what waits here; writes nothing")
	cmd.Flags().StringVar(&answer, "answer", "", "record the person's reply to the offer, yes or not-now, once for this repository and release; run it only with their reply")
	cmd.Flags().BoolVar(&seen, "seen", false, "record that the person read the news where no offer was made; run it only after they were shown it")
	return cmd
}

// runUpdateNewsAnswer records the person's answer and says what follows.
func runUpdateNewsAnswer(cmd *cobra.Command, cwd, answer string, asJSON bool) error {
	a, err := afterupdate.Record(cwd, answer)
	if err != nil && a.Release == "" {
		return updateNewsRefusal(err)
	}
	out := newsRecorded{Answer: a.Answer, Release: a.Release, NewsSeen: err == nil}
	if a.Answer == afterupdate.AnswerYes {
		out.Next = onlyWaitingCommand
	}
	if rerr := render(cmd.OutOrStdout(), asJSON, out, func(w io.Writer) {
		if a.Answer == afterupdate.AnswerYes {
			fmt.Fprintf(w, "abcd: recorded yes to setting up %s here; now run: %s\n", a.Release, onlyWaitingCommand)
			return
		}
		fmt.Fprintf(w, "abcd: recorded not now for %s here; the offer is not repeated until the next release, and abcd ahoy still lists what waits.\n", a.Release)
	}); rerr != nil {
		return rerr
	}
	// The answer stands; only the news mark failed, so the next moment shows
	// the news again rather than losing it.
	return err
}

// runUpdateNewsSeen records that the news up to the running release was read.
func runUpdateNewsSeen(cmd *cobra.Command, asJSON bool) error {
	if err := afterupdate.MarkSeen(); err != nil {
		return updateNewsRefusal(err)
	}
	m := newsRecorded{Release: runningRelease(), NewsSeen: true}
	return render(cmd.OutOrStdout(), asJSON, m, func(w io.Writer) {
		fmt.Fprintf(w, "abcd: recorded that this computer was shown abcd's news up to %s.\n", m.Release)
	})
}

// runningRelease is the running release after MarkSeen accepted it as one.
func runningRelease() string { return termsafe.Sanitize(core.Version) }

// updateNewsSessionLine is the session start's one line for the moment
// after an update. It is a constant, with no version and no count: stdout of
// session start reaches the session's context, so nothing a repository's
// tracked files hold may ride on it (iss-2608241115201044). The moment's own
// words reach context by the agent's run of the verb. It sends the agent to
// the plugin's command page rather than to a bare abcd: the page runs the
// binary this plugin ships, through the plugin root the host writes into it,
// where the abcd first on PATH can be an older release that has no news verb
// and takes the word for a release tag. Naming the binary's own path instead
// would put a machine path on the line, and a plugin directory or marketplace
// name can be text a repository chose.
const updateNewsSessionLine = "abcd: an abcd update has news for this session. After the person's first message, " +
	"run /abcd:update news, whose page runs the abcd this plugin ships (never a bare abcd: the one on PATH can be an older release), " +
	"then begin your first reply with the lines it prints, word for word, before anything else."

// updateNewsSessionStart prints updateNewsSessionLine to w where the moment
// holds anything for the repository at cwd, whose detection is det, and
// writes nothing. A failure to work it out is a notice for stderr, never a
// line on stdout.
func updateNewsSessionStart(w io.Writer, cwd string, det ahoy.DetectionResult) []string {
	st, err := afterupdate.Pending(cwd, det)
	if err != nil {
		return []string{"abcd: could not work out what the abcd update brings (" +
			termsafe.Sanitize(fsutil.RedactHome(err.Error())) + "); `abcd update news` says it when asked."}
	}
	if st.Any() {
		fmt.Fprintln(w, updateNewsSessionLine)
	}
	return nil
}

// updateNewsRefusal turns core's refusals into exit 2, naming that nothing
// was written; any other error is returned as it is.
func updateNewsRefusal(err error) error {
	for _, refusal := range []error{afterupdate.ErrNotRelease, afterupdate.ErrNoOffer, afterupdate.ErrAlreadyAnswered, afterupdate.ErrBadAnswer} {
		if errors.Is(err, refusal) {
			return &exitError{Code: 2, Msg: "abcd update news: " + err.Error() + "; nothing was written"}
		}
	}
	return err
}

// printUpdateNewsLines writes each line, sanitised for the terminal.
func printUpdateNewsLines(w io.Writer, lines []string) {
	for _, l := range lines {
		fmt.Fprintln(w, termsafe.Sanitize(l))
	}
}
