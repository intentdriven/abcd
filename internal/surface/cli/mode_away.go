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
	"github.com/intentdriven/abcd/internal/termsafe"
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
	// Notes say what could not be read or done as usual on the way to
	// ending the mode.
	Notes []string `json:"notes,omitempty"`
}

// queueOutput is the --json envelope of `abcd mode queue "<step>"`.
type queueOutput struct {
	Queued int `json:"queued"`
}

// queueListOutput is the --json envelope of bare `abcd mode queue`, which
// lists the queue and writes nothing.
type queueListOutput struct {
	Queued int      `json:"queued"`
	Steps  []string `json:"steps"`
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
			"Setting it from a session whose setting reads as off a profile prints one line on stderr opening\n" +
			"WARNING:, after the set, naming what abcd cannot check and addressed to the person. The mode is set and\n" +
			"the exit code is the same either way: it warns and never refuses. " + offProfileHelp + "\n\n" +
			"The record lives in " + abcdhome.Display(autonomy.StoreDir, "<root-sha>", autonomy.RecordFile) + ", keyed on the repository's\n" +
			"root commit, and is changed under its lock, which only `abcd mode back` goes\n" +
			"around. A record merely past its 64 KiB is carried, its oldest lines let go.\n" +
			"Exit 2 on a refusal (an unreadable or past --until, one past seven days, no local\n" +
			"tier, no checkout, or a record or lock that is not safely this account's) with\n" +
			"nothing written. No network request.",
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
			warnOffProfile(cmd, "abcd mode autonomous")
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
			"A session runs it only on the person's word in that session. It ends the mode\n" +
			"whatever state the record is in: one past its 64 KiB is printed whole and kept\n" +
			"with its oldest lines let go and counted; one that does not parse, or is not\n" +
			"safely this account's, is replaced by a closed record, printing nothing from\n" +
			"it; and a lock not had in time is gone around. Each is named on its own line.\n" +
			"No network request.",
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
			out := backOutput{State: res.State, Lines: res.Lines, Notes: res.Notes}
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
				for _, n := range out.Notes {
					fmt.Fprintln(w, "abcd: "+termsafe.CleanProseLine(n, 512))
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
// autonomy.QueueStep, and bare `abcd mode queue`, which lists the queue.
func newModeQueueCommand(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use: "queue [<step>]",
		Long: "Queue one step for the person's return, while the away mode holds: an act only\n" +
			"the person may approve, which a session queues instead of asking, then carries on\n" +
			"with other work. `abcd mode back` prints each queued step on one line.\n\n" +
			"Bare `abcd mode queue` lists the steps queued in a record not yet ended, one per\n" +
			"line, oldest first, and writes nothing: the away mode holds as before. With no\n" +
			"record, or one already ended, it prints nothing. With --json it carries the count\n" +
			"and the steps.\n\n" +
			"Queuing a step exits 2 when no away record holds (ask the person instead), on an\n" +
			"empty step or one over 500 characters, and when the queue is full (it is never\n" +
			"cut); either form exits 2 when the record or its lock is not safely this account's.\n" +
			"Nothing is written on any refusal. No network request.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			if len(args) == 0 {
				return listQueue(cmd.OutOrStdout(), *asJSON, cwd)
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

// listQueue is bare `abcd mode queue`: the steps queued in a record not yet
// ended (one that holds, or one whose time passed before `abcd mode back`
// read it), one per line, with nothing written and no lock taken.
func listQueue(w io.Writer, asJSON bool, cwd string) error {
	_, st, err := autonomy.Resolve(cwd, autonomy.Options{})
	if err != nil {
		return awayRefusal("queue", err)
	}
	rec, err := st.Read()
	if err != nil {
		return awayRefusal("queue", err)
	}
	out := queueListOutput{Steps: []string{}}
	if rec != nil && rec.Closed == nil {
		for _, q := range rec.Queue {
			out.Steps = append(out.Steps, termsafe.CleanProseLine(q.Step, autonomy.MaxStepLen*4))
		}
	}
	out.Queued = len(out.Steps)
	return render(w, asJSON, out, func(w io.Writer) {
		for _, s := range out.Steps {
			fmt.Fprintln(w, s)
		}
	})
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
		autonomy.ErrStepEmpty, autonomy.ErrStepTooLong, autonomy.ErrQueueFull, autonomy.ErrNotAway,
		autonomy.ErrNoLocalTier, autonomy.ErrUnsafe, autonomy.ErrLockBusy, autonomy.ErrRecordFull,
		autonomy.ErrNoRootCommit, autonomy.ErrNoHome, gitutil.ErrNoCheckoutRoot,
	} {
		if errors.Is(err, sentinel) {
			return &exitError{Code: 2, Msg: "abcd mode " + verb + ": " + err.Error() + " (nothing written)"}
		}
	}
	return fmt.Errorf("abcd mode %s: %w", verb, err)
}
