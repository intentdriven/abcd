package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// outageNow and outageNetworkProbe are the outage verbs' two seams onto the
// real world: the clock the run reads (nil is the wall clock) and the network
// probe, given the checkout root. Tests swap them, so no test waits or touches
// the network.
var (
	outageNow          func() time.Time
	outageNetworkProbe = implement.RemoteProbe
)

// implementOutageOutput is the bare render's --json shape: the outage in
// force, null when there is none.
type implementOutageOutput struct {
	Dir    string            `json:"dir"`
	Outage *implement.Outage `json:"outage"`
}

// implementOutageWait is the probe's --json shape when it must wait: not due,
// or another session's probe running.
type implementOutageWait struct {
	Waiting bool `json:"waiting"`
	*implement.OutageWaitError
}

// newImplementOutageCommand builds `implement outage`, the front door onto the
// run's shared outage (iss-2610080620372731). It formats what
// implement.Outages returns and holds no logic of its own.
func newImplementOutageCommand(asJSON *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use: "outage",
		Long: "The run's shared lost connection. A lane that loses the network (git, gh, a\n" +
			"download) or the model service (an agent back with an API error, overloaded, a\n" +
			"5xx, a timed-out request) records it (`record`); the run keeps one outage record\n" +
			"in the run state, and every lane keeps to offline work and waits on one shared\n" +
			"probe (`probe`) instead of retrying alone. The probe runs a minute after the\n" +
			"outage opens, then five minutes and ten minutes after each failed probe, then\n" +
			"hourly; the failed probe eight hours into the hourly stage gives up, stops the\n" +
			"run and raises one notification, held until a session acknowledges it (`ack`).\n" +
			"A usage or rate limit is not an outage.\n\n" +
			"The network is proven back by `git ls-remote origin HEAD`; the model service only\n" +
			"by a canary agent the lead runs and reports with `probe --model ok|fail` — the\n" +
			"lead's own turn running is not proof. The outage ends when every service down is\n" +
			"proven back, and `report` lists each outage with its minutes and what was retried.\n\n" +
			"Bare `abcd implement outage` is read-only: the outage in force, the services down,\n" +
			"when the next probe is due and who holds it. It creates nothing.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			run, err := implementRun(runPeek, "")
			if err != nil {
				return implementRefusal("outage", err)
			}
			cur, err := run.Outage().Current()
			if err != nil {
				return implementRefusal("outage", err)
			}
			st := implementOutageOutput{Dir: fsutil.RedactHome(run.Dir), Outage: cur}
			return render(cmd.OutOrStdout(), *asJSON, st, func(w io.Writer) { renderOutage(w, cur) })
		},
	}
	cmd.AddCommand(
		newImplementOutageRecordCommand(asJSON),
		newImplementOutageProbeCommand(asJSON),
		newImplementOutageAckCommand(asJSON),
		newImplementOutageClearCommand(asJSON),
	)
	return cmd
}

// withOutageRun is withRun for an outage sub-verb, with the verb's clock.
func withOutageRun(sub, session string, fn func(*implement.Run) error) error {
	return withRun("outage "+sub, session, func(run *implement.Run) error {
		if outageNow != nil {
			run.Now = outageNow
		}
		return fn(run)
	})
}

// renderOutage writes the outage in force.
func renderOutage(w io.Writer, o *implement.Outage) {
	if o == nil {
		fmt.Fprintln(w, "no outage: nothing to wait on")
		return
	}
	switch o.Status {
	case implement.OutageGaveUp:
		fmt.Fprintf(w, "outage open since %s; the run gave up at %s after %d probes and has stopped\n",
			o.StartedAt.Format(time.RFC3339), o.GaveUpAt.Format(time.RFC3339), len(o.Probes))
		if o.NotifyPending {
			fmt.Fprintln(w, "  notification pending: tell the product thinker, then `abcd implement outage ack`")
		}
	default:
		fmt.Fprintf(w, "outage open since %s\n", o.StartedAt.Format(time.RFC3339))
		fmt.Fprintf(w, "  next probe: %s", o.NextProbeAt.Format(time.RFC3339))
		if o.HourlySince != nil {
			fmt.Fprintf(w, " (hourly since %s)", o.HourlySince.Format(time.RFC3339))
		}
		fmt.Fprintln(w)
		if o.Lease != nil {
			fmt.Fprintf(w, "  probing: session %s, lease until %s\n", termsafe.Sanitize(o.Lease.Holder), o.Lease.Until.Format(time.RFC3339))
		}
	}
	fmt.Fprintf(w, "  down:     %s (of %s)\n", strings.Join(o.Down, ", "), strings.Join(o.Services, ", "))
	fmt.Fprintf(w, "  noticed:  %s\n", strings.Join(o.Kinds, ", "))
	fmt.Fprintf(w, "  probes:   %d, %d failed\n", len(o.Probes), o.Failures())
	fmt.Fprintf(w, "  retrying: %s\n", termsafe.Sanitize(strings.Join(o.Retried(), "; ")))
}

func newImplementOutageRecordCommand(asJSON *bool) *cobra.Command {
	var session, service, kind, lane, what string
	cmd := &cobra.Command{
		Use: "record --session <id> --service network|model --kind host|agent|tool --lane <lane> --what <text>",
		Long: "Report a lost connection: the service lost (network or model), what noticed it\n" +
			"(host: the host's own model calls; agent: a sub-agent back failed; tool: a tool's\n" +
			"network call), the lane and what it was doing. The first report opens the run's\n" +
			"outage and logs outage_start; a later one joins it. A report to an outage that\n" +
			"gave up joins it and raises no second notification.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withOutageRun("record", session, func(run *implement.Run) error {
				o, err := run.Outage().Record(session, service, kind, lane, what)
				if err != nil {
					return err
				}
				return render(cmd.OutOrStdout(), *asJSON, o, func(w io.Writer) {
					fmt.Fprintf(w, "outage recorded (%s down); next probe at %s\n", strings.Join(o.Down, ", "),
						o.NextProbeAt.Format(time.RFC3339))
				})
			})
		},
	}
	cmd.Flags().StringVar(&session, "session", "", "this session's id")
	cmd.Flags().StringVar(&service, "service", "", "the service lost: "+enumHelp(implement.OutageServices()))
	cmd.Flags().StringVar(&kind, "kind", "", "what noticed it: "+enumHelp(implement.OutageKinds()))
	cmd.Flags().StringVar(&lane, "lane", "", "the lane that lost it")
	cmd.Flags().StringVar(&what, "what", "", "what the lane was doing (the step it will retry)")
	return cmd
}

func newImplementOutageProbeCommand(asJSON *bool) *cobra.Command {
	var session, model string
	cmd := &cobra.Command{
		Use: "probe --session <id> [--model ok|fail]",
		Long: "Run the shared probe if it is due and no other session is running it. The network,\n" +
			"when down, is probed with `git ls-remote --exit-code origin HEAD` (20s at most); the\n" +
			"model service, when down, takes --model, the verdict of the canary agent the lead\n" +
			"ran — without it the model side stays down. Read the bare verb's next probe time\n" +
			"before running the canary, so it runs once per due probe.\n\n" +
			"Exit 0: no outage, or every service proven back — the network step may go ahead.\n" +
			"Exit 3: wait — the probe is not due, another session holds it, or a service is\n" +
			"still down; --json carries next_probe_at. Exit 2: the run gave up, or a refusal.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var verdict func() (bool, string)
			switch model {
			case "":
			case "ok":
				verdict = func() (bool, string) { return true, "the canary agent answered" }
			case "fail":
				verdict = func() (bool, string) { return false, "the canary agent failed" }
			default:
				return &exitError{Code: 2, Msg: fmt.Sprintf("abcd implement outage probe: --model %q is not ok or fail (nothing probed)", model)}
			}
			// The exit a rendered outcome carries is returned after withRun, which
			// would otherwise prefix it with a message the rendered output already
			// says.
			exit := 0
			err := withOutageRun("probe", session, func(run *implement.Run) error {
				p := implement.Prober{Network: outageNetworkProbe(run.RepoRoot), Model: verdict}
				out, err := run.Outage().ProbeIfDue(session, p)
				var wait *implement.OutageWaitError
				if errors.As(err, &wait) && *asJSON {
					exit = 3
					return render(cmd.OutOrStdout(), true, implementOutageWait{Waiting: true, OutageWaitError: wait}, nil)
				}
				if err != nil {
					return err
				}
				if rerr := render(cmd.OutOrStdout(), *asJSON, out, func(w io.Writer) { renderProbe(w, out) }); rerr != nil {
					return rerr
				}
				switch {
				case out.GaveUp:
					exit = 2
				case out.Probed && !out.Ended && out.Outage != nil:
					exit = 3
				}
				return nil
			})
			if err == nil && exit != 0 {
				return &exitError{Code: exit}
			}
			return err
		},
	}
	cmd.Flags().StringVar(&session, "session", "", "this session's id")
	cmd.Flags().StringVar(&model, "model", "", "the canary agent's verdict on the model service: ok | fail")
	return cmd
}

// renderProbe writes what a probe did.
func renderProbe(w io.Writer, out implement.ProbeOutcome) {
	switch {
	case !out.Probed:
		fmt.Fprintln(w, "no outage: nothing to wait on")
		return
	case out.Ended:
		fmt.Fprintf(w, "every service is back; the outage ended after %.1f min\n", out.Minutes)
		return
	}
	for _, r := range out.Probe.Results {
		state := "still down"
		if r.OK {
			state = "back"
		}
		fmt.Fprintf(w, "  %s: %s (%s)\n", r.Service, state, termsafe.Sanitize(r.Detail))
	}
	if out.GaveUp {
		fmt.Fprintln(w, "the run gave up on the outage and stops here; tell the product thinker, then `abcd implement outage ack`")
		return
	}
	fmt.Fprintf(w, "still down (%s); next probe at %s\n", strings.Join(out.Outage.Down, ", "), out.Outage.NextProbeAt.Format(time.RFC3339))
}

func newImplementOutageAckCommand(asJSON *bool) *cobra.Command {
	var session string
	cmd := &cobra.Command{
		Use: "ack --session <id>",
		Long: "Acknowledge the give-up's notification once the product thinker has been told.\n" +
			"Refused when no notification is pending, so it is never raised twice.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withOutageRun("ack", session, func(run *implement.Run) error {
				o, err := run.Outage().Ack(session)
				if err != nil {
					return err
				}
				return render(cmd.OutOrStdout(), *asJSON, o, func(w io.Writer) {
					fmt.Fprintln(w, "notification acknowledged")
				})
			})
		},
	}
	cmd.Flags().StringVar(&session, "session", "", "this session's id")
	return cmd
}

func newImplementOutageClearCommand(asJSON *bool) *cobra.Command {
	var session, reason string
	cmd := &cobra.Command{
		Use: "clear --session <id> --reason <why>",
		Long: "Close the outage by hand, open or given up. It logs outage_end with the reason and\n" +
			"an intervention: the shared probe did not prove the connection back on its own.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withOutageRun("clear", session, func(run *implement.Run) error {
				end, err := run.Outage().Clear(session, reason)
				if err != nil {
					return err
				}
				return render(cmd.OutOrStdout(), *asJSON, end, func(w io.Writer) {
					fmt.Fprintf(w, "outage cleared after %.1f min\n", end.Minutes)
				})
			})
		},
	}
	cmd.Flags().StringVar(&session, "session", "", "this session's id")
	cmd.Flags().StringVar(&reason, "reason", "", "why it is cleared by hand")
	return cmd
}

// renderOutageSpans writes the report's outages.
func renderOutageSpans(w io.Writer, spans []implement.OutageSpan) {
	for _, s := range spans {
		outcome := s.Outcome
		if outcome == "gave_up" {
			outcome = "gave up"
		}
		fmt.Fprintf(w, "outage: %s, %.1f min, %s; %s down, noticed by %s; %d probe(s); retried: %s\n",
			s.Start.Format(time.RFC3339), s.Minutes, outcome, strings.Join(s.Services, "+"), strings.Join(s.Kinds, ", "),
			s.Probes, termsafe.Sanitize(strings.Join(s.Retried, "; ")))
	}
}
