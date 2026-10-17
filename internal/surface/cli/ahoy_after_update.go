package cli

// ahoy_after_update.go is the front door of the moment after an abcd update
// (spc-2610170635097509, step 7): `abcd ahoy after-update` prints what the
// agent relays at the start of its first reply, the one line per release
// saying what changed and the offer to run setup again; --full prints the
// full summary; --answer records the person's answer to the offer and --seen
// that the news was read. Bare and --full write nothing. The words are
// core's (internal/core/afterupdate): numbers, checked release tags, core's
// titles and the digest inside the binary, never the repository's text.
//
// It also holds `ahoy install --only-waiting`, the setup the yes runs: the
// flag is registered here and read by the install's RunE.

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/afterupdate"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// onlyWaitingFlag is the install's flag for the setup the yes runs.
const onlyWaitingFlag = "only-waiting"

// addOnlyWaitingFlag registers --only-waiting on the install command.
func addOnlyWaitingFlag(installCmd *cobra.Command) {
	installCmd.Flags().Bool(onlyWaitingFlag, false, "ask only the setup that waits after an abcd update: "+
		"the questions this repository never answered and the settings whose default changed, never one answered or deferred with decide later; "+
		"every kind of change is approved up front, as the yes to the after-update offer approves them; "+
		"with the approve-everything flag no question is answered, so each still waits")
}

// onlyWaiting reports whether the install was asked for --only-waiting.
func onlyWaiting(cmd *cobra.Command) bool {
	v, _ := cmd.Flags().GetBool(onlyWaitingFlag)
	return v
}

// afterUpdateRecorded is what --answer and --seen render.
type afterUpdateRecorded struct {
	Answer   string `json:"answer,omitempty"`
	Release  string `json:"release"`
	NewsSeen bool   `json:"news_seen"`
	Next     string `json:"next,omitempty"`
}

// onlyWaitingCommand is the setup a yes runs.
const onlyWaitingCommand = "abcd ahoy install --only-waiting"

func newAhoyAfterUpdateCommand(asJSON *bool) *cobra.Command {
	var full, seen bool
	var answer string
	cmd := &cobra.Command{
		Use:  "after-update",
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
				return &exitError{Code: 2, Msg: "abcd ahoy after-update: --answer records the reply to the offer and takes neither --full nor --seen; nothing was written"}
			case full && seen:
				return &exitError{Code: 2, Msg: "abcd ahoy after-update: --full shows the summary and writes nothing, so it takes no --seen; nothing was written"}
			case answered && answer != afterupdate.AnswerYes && answer != afterupdate.AnswerNotNow:
				return &exitError{Code: 2, Msg: "abcd ahoy after-update: --answer is yes or not-now; nothing was written"}
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			switch {
			case answered:
				return runAfterUpdateAnswer(cmd, cwd, answer, *asJSON)
			case seen:
				return runAfterUpdateSeen(cmd, *asJSON)
			case full:
				f, err := afterupdate.Full(cwd, "")
				if err != nil {
					return afterUpdateRefusal(err)
				}
				return render(cmd.OutOrStdout(), *asJSON, f, func(w io.Writer) { printAfterUpdateLines(w, f.Lines) })
			}
			m, err := afterupdate.Moment(cwd)
			if err != nil {
				return err
			}
			return render(cmd.OutOrStdout(), *asJSON, m, func(w io.Writer) { printAfterUpdateLines(w, m.Lines) })
		},
	}
	cmd.Flags().BoolVar(&full, "full", false, "print the full summary of the releases since the last update and what waits here; writes nothing")
	cmd.Flags().StringVar(&answer, "answer", "", "record the person's reply to the offer, yes or not-now, once for this repository and release; run it only with their reply")
	cmd.Flags().BoolVar(&seen, "seen", false, "record that the person read the news where no offer was made; run it only after they were shown it")
	return cmd
}

// runAfterUpdateAnswer records the person's answer and says what follows.
func runAfterUpdateAnswer(cmd *cobra.Command, cwd, answer string, asJSON bool) error {
	a, err := afterupdate.Record(cwd, answer)
	if err != nil && a.Release == "" {
		return afterUpdateRefusal(err)
	}
	out := afterUpdateRecorded{Answer: a.Answer, Release: a.Release, NewsSeen: err == nil}
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

// runAfterUpdateSeen records that the news up to the running release was read.
func runAfterUpdateSeen(cmd *cobra.Command, asJSON bool) error {
	if err := afterupdate.MarkSeen(); err != nil {
		return afterUpdateRefusal(err)
	}
	m := afterUpdateRecorded{Release: runningRelease(), NewsSeen: true}
	return render(cmd.OutOrStdout(), asJSON, m, func(w io.Writer) {
		fmt.Fprintf(w, "abcd: recorded that this computer was shown abcd's news up to %s.\n", m.Release)
	})
}

// runningRelease is the running release after MarkSeen accepted it as one.
func runningRelease() string { return termsafe.Sanitize(core.Version) }

// afterUpdateRefusal turns core's refusals into exit 2, naming that nothing
// was written; any other error is returned as it is.
func afterUpdateRefusal(err error) error {
	for _, refusal := range []error{afterupdate.ErrNotRelease, afterupdate.ErrNoOffer, afterupdate.ErrAlreadyAnswered, afterupdate.ErrBadAnswer} {
		if errors.Is(err, refusal) {
			return &exitError{Code: 2, Msg: "abcd ahoy after-update: " + err.Error() + "; nothing was written"}
		}
	}
	return err
}

// printAfterUpdateLines writes each line, sanitised for the terminal.
func printAfterUpdateLines(w io.Writer, lines []string) {
	for _, l := range lines {
		fmt.Fprintln(w, termsafe.Sanitize(l))
	}
}
