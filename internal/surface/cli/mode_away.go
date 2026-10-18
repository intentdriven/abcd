package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/autonomy"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/spf13/cobra"
)

// awayOutput is the --json envelope of `abcd mode autonomous`.
type awayOutput struct {
	Away       bool      `json:"away"`
	SetBy      string    `json:"set_by"`
	SetFrom    string    `json:"set_from"`
	Start      time.Time `json:"start"`
	Until      time.Time `json:"until"`
	OpenEnded  bool      `json:"open_ended"`
	HoldPushes bool      `json:"hold_pushes"`
	Queued     int       `json:"queued"`
	HandedBack int       `json:"handed_back"`
	Stops      int       `json:"stops"`
	Terms      []string  `json:"terms"`
}

// backOutput is the --json envelope of `abcd mode back`.
type backOutput struct {
	State     string   `json:"state"`
	Lines     []string `json:"lines"`
	EndedBy   string   `json:"ended_by,omitempty"`
	EndedFrom string   `json:"ended_from,omitempty"`
}

// queueOutput is the --json envelope of `abcd mode queue`.
type queueOutput struct {
	Queued int `json:"queued"`
}

// newModeAutonomousCommand builds `abcd mode autonomous`, the front door onto
// autonomy.SetAway (itd-2610180912189125, step 1).
func newModeAutonomousCommand(asJSON *bool) *cobra.Command {
	var until string
	var holdPushes bool
	cmd := &cobra.Command{
		Use: "autonomous",
		Long: "Record that the person is away, for every working copy of this repository on\n" +
			"this machine: who set it (the holder name reservations resolve without a\n" +
			"network read, else git's user.name), from which working copy, and when it ends.\n\n" +
			"--until takes a clock time (its next occurrence, local time, as 09:00) or a\n" +
			"timestamp (2026-10-19T09:00, or RFC 3339). Without it the mode is open-ended,\n" +
			"and every record ends seven days after it was set at most: an --until past that,\n" +
			"or one already past, is refused. --hold-pushes records the person's word that\n" +
			"pushes wait for their return; it narrows what sessions do and grants nothing.\n\n" +
			"The away mode grants no rights: the person's own standing instruction decides\n" +
			"what is committed, pushed, and opened or merged as a pull request. It ends at its\n" +
			"time or with `abcd mode back`; a prompt never ends it. Setting it again keeps\n" +
			"whatever is queued or recorded and not yet read.\n\n" +
			"The record lives in " + abcdhome.Display(autonomy.StoreDir, "<root-sha>", autonomy.RecordFile) + ", keyed on the repository's\n" +
			"root commit, and is changed only under its lock. Exit 2 on a refusal (an unreadable\n" +
			"or past --until, one past seven days, no local tier, no checkout, or a record or\n" +
			"lock that is not safely this account's) with nothing written. No network request.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			rec, err := autonomy.SetAway(cwd, until, holdPushes, autonomy.Options{})
			if err != nil {
				return awayRefusal("autonomous", err)
			}
			out := awayOutput{
				Away: true, SetBy: rec.SetBy, SetFrom: fsutil.DisplayPath(rec.SetFrom),
				Start: rec.Start, Until: rec.Ends(), OpenEnded: rec.OpenEnded, HoldPushes: rec.HoldPushes,
				Queued: len(rec.Queue), HandedBack: len(rec.HandedBack), Stops: len(rec.Stops), Terms: rec.Terms(),
			}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				for _, l := range out.Terms {
					fmt.Fprintln(w, l)
				}
				if n := out.Queued + out.HandedBack + out.Stops; n > 0 {
					fmt.Fprintf(w, "Kept from before, not yet read: %d queued, %d handed back, %d stops recorded.\n",
						out.Queued, out.HandedBack, out.Stops)
				}
			})
		},
	}
	cmd.Flags().StringVar(&until, "until", "", "when the mode ends: a clock time (09:00) or a timestamp; open-ended (seven days at most) when omitted")
	cmd.Flags().BoolVar(&holdPushes, "hold-pushes", false, "the person's word that pushes wait for their return (grants nothing)")
	return cmd
}

// newModeBackCommand builds `abcd mode back`, the front door onto
// autonomy.Back.
func newModeBackCommand(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use: "back",
		Long: "End the away mode: print one plain line per queued step, per item handed back\n" +
			"and per recorded stop (a stop at the host's cap is labelled \"possibly forced by\n" +
			"the cap\"), record who ended it and from which working copy, and close the record.\n" +
			"On a record whose time has already passed it prints the same and closes it. With\n" +
			"no record, or one already closed, it says there is nothing to end.\n\n" +
			"A session runs it only on the person's word in that session. Exit 2 when the\n" +
			"record or its lock is not safely this account's, naming the file, with nothing\n" +
			"written. No network request.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			res, err := autonomy.Back(cwd, autonomy.Options{})
			if err != nil {
				return awayRefusal("back", err)
			}
			out := backOutput{State: res.State, Lines: res.Lines}
			if out.Lines == nil {
				out.Lines = []string{}
			}
			if res.State != "none" && res.Record != nil && res.Record.Closed != nil {
				out.EndedBy = res.Record.Closed.By
				out.EndedFrom = fsutil.DisplayPath(res.Record.Closed.From)
			}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				for _, l := range out.Lines {
					fmt.Fprintln(w, l)
				}
				switch res.State {
				case "none":
					fmt.Fprintln(w, "abcd: not in the away mode; nothing to end")
				case "lapsed":
					fmt.Fprintf(w, "abcd: the away window ended at %s; closed by %s from %s\n",
						res.Record.Ends().Local().Format("2006-01-02 15:04 MST"), whoWords(out.EndedBy), out.EndedFrom)
				default:
					fmt.Fprintf(w, "abcd: the away mode is over, ended by %s from %s\n", whoWords(out.EndedBy), out.EndedFrom)
				}
			})
		},
	}
}

// newModeQueueCommand builds `abcd mode queue "<step>"`, the front door onto
// autonomy.QueueStep.
func newModeQueueCommand(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use: "queue <step>",
		Long: "Queue one step for the person's return, while the away mode holds: an act only\n" +
			"the person may approve, which a session queues instead of asking, then carries on\n" +
			"with other work. `abcd mode back` prints each queued step on one line.\n\n" +
			"Exit 2 when no away record holds (ask the person instead), on an empty step or one\n" +
			"over 500 characters, and when the record or its lock is not safely this account's,\n" +
			"with nothing written. No network request.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			rec, err := autonomy.QueueStep(cwd, args[0], autonomy.Options{})
			if err != nil {
				return awayRefusal("queue", err)
			}
			out := queueOutput{Queued: len(rec.Queue)}
			return render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) {
				fmt.Fprintf(w, "abcd: queued for the person's return (%d queued)\n", out.Queued)
			})
		},
	}
}

func whoWords(name string) string {
	if name == "" {
		return "someone git does not name"
	}
	return name
}

// awayRefusal maps the away verbs' refusals to exit 2 with nothing written:
// each is "this invocation cannot be honoured". Anything else is an I/O
// failure at the default exit 1.
func awayRefusal(verb string, err error) error {
	for _, sentinel := range []error{
		autonomy.ErrUntilUnreadable, autonomy.ErrUntilPast, autonomy.ErrUntilBeyondCap,
		autonomy.ErrStepEmpty, autonomy.ErrStepTooLong, autonomy.ErrNotAway,
		autonomy.ErrNoLocalTier, autonomy.ErrUnsafe, autonomy.ErrLockBusy, autonomy.ErrRecordFull,
		autonomy.ErrNoRootCommit, autonomy.ErrNoHome, gitutil.ErrNoCheckoutRoot,
	} {
		if errors.Is(err, sentinel) {
			return &exitError{Code: 2, Msg: "abcd mode " + verb + ": " + err.Error() + " (nothing written)"}
		}
	}
	return fmt.Errorf("abcd mode %s: %w", verb, err)
}
