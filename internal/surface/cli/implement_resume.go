package cli

// implement_resume.go is the front door of the take-over (itd-2611041549461012):
// `abcd implement resume` reads what every run on the machine wrote down, and
// `--take <run-id>` acts on it for one run.

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// newImplementResumeCommand builds `abcd implement resume [--run <id>] [--take <id>]`.
func newImplementResumeCommand(asJSON *bool) *cobra.Command {
	var only, take string
	cmd := &cobra.Command{
		Use: "resume [--run <run-id>] [--take <run-id>]",
		Long: "List every unfinished run this account can read on the machine, in this repository's\n" +
			"worktrees and in each checkout a driver worked in: its driver alive or gone, what a\n" +
			"stop left, and each lane's stage, each agent alive, lost or unknown, each receipt\n" +
			"that arrived uncollected, its pull request, how far origin's default branch as last\n" +
			"fetched has moved past its base, and the files it holds uncommitted. The listing is\n" +
			"all a session that lost its context needs. It writes nothing and uses no network.\n\n" +
			"--take takes a run over from this session, in the checkout that holds it: it claims\n" +
			"the driver, collects every receipt that has arrived before anything else, reserves\n" +
			"again what a stop released, saves aside and names the uncommitted work a gone agent\n" +
			"left, and re-tells its await. It starts no agent: `abcd implement step` does that\n" +
			"next. An agent that may still be in its call is left alone and named.\n\n" +
			"Exit 2 on a refusal. Exit 3 while a stop holds, while another driver's process is\n" +
			"alive or a recorded driver cannot be judged, or when a peer has taken the work a\n" +
			"stop released: back off.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			const prefix = "abcd implement resume"
			w := cmd.OutOrStdout()
			root, err := loopRoot()
			if err != nil {
				return loopFail(w, *asJSON, prefix, err)
			}
			if take != "" {
				res, err := loop.TakeRun(root, take, loop.Options{})
				if err != nil {
					return loopFail(w, *asJSON, prefix, err)
				}
				return render(w, *asJSON, res, func(w io.Writer) { renderTake(w, res) })
			}
			l, err := loop.ListRuns(root, only, loop.Options{})
			if err != nil {
				return loopFail(w, *asJSON, prefix, err)
			}
			return render(w, *asJSON, l, func(w io.Writer) { renderListing(w, l) })
		},
	}
	cmd.Flags().StringVar(&only, "run", "", "list this run alone (run-<16 digits>)")
	cmd.Flags().StringVar(&take, "take", "", "take this run over (run-<16 digits>)")
	cmd.MarkFlagsMutuallyExclusive("run", "take")
	return cmd
}

// renderListing is the listing's text form.
func renderListing(w io.Writer, l loop.Listing) {
	if l.Stop != nil {
		fmt.Fprintf(w, "stop: set by %s at %s holds\n", termsafe.Sanitize(l.Stop.SetBy), l.Stop.SetAt.Format("2006-01-02T15:04:05Z07:00"))
	}
	for _, r := range l.Refused {
		fmt.Fprintf(w, "refused: %s\n", termsafe.Sanitize(r))
	}
	for _, u := range l.Unreadable {
		fmt.Fprintf(w, "unreadable: %s\n", termsafe.Sanitize(u))
	}
	if len(l.Runs) == 0 && len(l.Drains) == 0 {
		fmt.Fprintln(w, "no unfinished run on this machine")
	}
	for _, r := range l.Runs {
		renderListedRun(w, r)
	}
	for _, d := range l.Drains {
		fmt.Fprintf(w, "drain in %s: %s\n", termsafe.Sanitize(d.Checkout), termsafe.Sanitize(d.Next))
	}
}

func renderListedRun(w io.Writer, r loop.ListedRun) {
	fmt.Fprintf(w, "%s  %s  in %s\n", termsafe.Sanitize(r.RunID), termsafe.Sanitize(r.Key), termsafe.Sanitize(r.Checkout))
	if d := r.Driver; d != nil {
		name := fmt.Sprintf("process %d", d.PID)
		if d.Session != "" {
			name = "session " + d.Session + " (" + name + ")"
		}
		fmt.Fprintf(w, "  driver: %s, %s\n", termsafe.Sanitize(name), d.State)
	} else {
		fmt.Fprintln(w, "  driver: none recorded")
	}
	if !r.Reserved {
		fmt.Fprintln(w, "  reservation: not held")
	}
	if s := r.Stop; s != nil {
		state := "an agent is still in its call"
		if s.HaltedAt != nil {
			state = "halted: " + s.Released
		}
		fmt.Fprintf(w, "  stop: set by %s; %s\n", termsafe.Sanitize(s.SetBy), termsafe.Sanitize(state))
		for _, l := range s.Left {
			line := fmt.Sprintf("%s at %s", l.Lane, l.Stage)
			if l.Aside != "" {
				line += fmt.Sprintf(": %d file(s) saved aside at %s", l.Files, l.Aside)
			}
			if l.Note != "" {
				line += ": " + l.Note
			}
			fmt.Fprintf(w, "    left %s\n", termsafe.Sanitize(line))
		}
	}
	for _, l := range r.Lanes {
		var bits []string
		if l.PR != 0 {
			bits = append(bits, fmt.Sprintf("pull request #%d", l.PR))
		}
		if l.BaseMoved != nil {
			bits = append(bits, fmt.Sprintf("base moved %d commit(s)", *l.BaseMoved))
		}
		switch {
		case l.Uncommitted != nil:
			bits = append(bits, fmt.Sprintf("%d file(s) uncommitted", *l.Uncommitted))
		case l.UnreadWhy != "":
			bits = append(bits, l.UnreadWhy)
		}
		fmt.Fprintf(w, "  %s at %s: %s\n", termsafe.Sanitize(l.ID), termsafe.Sanitize(string(l.Stage)), termsafe.Sanitize(strings.Join(bits, ", ")))
		for _, a := range l.Agents {
			arrived := ""
			if a.ReceiptArrived {
				arrived = ", its receipt has arrived uncollected"
			}
			fmt.Fprintf(w, "    %s: %s%s\n", termsafe.Sanitize(a.Role), a.State, arrived)
		}
		for _, a := range l.Asides {
			fmt.Fprintf(w, "    aside: %s\n", termsafe.Sanitize(a))
		}
	}
	fmt.Fprintf(w, "  next: %s\n", termsafe.Sanitize(r.Next))
}

// renderTake is the take-over's text form.
func renderTake(w io.Writer, res loop.TakeResult) {
	fmt.Fprintf(w, "took over %s\n", termsafe.Sanitize(res.RunID))
	for lane, note := range res.Collected {
		fmt.Fprintf(w, "  %s: %s\n", termsafe.Sanitize(lane), termsafe.Sanitize(note))
	}
	if res.Reservation != "" {
		fmt.Fprintf(w, "  reservation: %s\n", termsafe.Sanitize(res.Reservation))
	}
	for _, a := range res.Saved {
		fmt.Fprintf(w, "  %s: %d file(s) saved aside at %s\n", termsafe.Sanitize(a.Lane), len(a.Files), termsafe.Sanitize(a.Path))
	}
	for _, r := range res.Restarted {
		fmt.Fprintf(w, "  %s: restarted from %s; %d file(s) saved aside at %s; %s\n", termsafe.Sanitize(r.Lane), shortSHA(r.Aside.Head),
			len(r.Aside.Files), termsafe.Sanitize(r.Aside.Path), termsafe.Sanitize(r.Next))
	}
	for _, l := range res.Left {
		fmt.Fprintf(w, "  left: %s\n", termsafe.Sanitize(l))
	}
	if res.Run != nil {
		renderListedRun(w, *res.Run)
	}
	fmt.Fprintf(w, "next: %s\n", termsafe.Sanitize(res.Next))
}
