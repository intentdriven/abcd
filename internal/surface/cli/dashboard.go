package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/adapter/tailscale"
	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/surface/dashboard"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// dashboardDefaultPort is the port start listens on unless --port names
// another (decision 23 of itd-2610032150577708).
const dashboardDefaultPort = 8080

// newDashboardCommand builds the `dashboard` verb — the front door onto
// internal/surface/dashboard (itd-2610032150577708, spc-2610040741034208 step
// 1): start, stop and status of the one server abcd may run, reachable only
// from devices on the person's own Tailscale network.
func newDashboardCommand(asJSON *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use: "dashboard",
		Long: "Start, stop and report the product thinker's dashboard: one server on this\n" +
			"computer, listening only on this computer's own Tailscale addresses, over plain\n" +
			"HTTP with no certificate, never through Serve or Funnel. Every connection is let\n" +
			"in only when Tailscale's own lookup of its address names a device of a person,\n" +
			"never from a header; any device on the Tailscale network can open it but a\n" +
			"tagged machine and this computer itself, which start checks once and then\n" +
			"refuses, so the dashboard is opened from another device. Until the product\n" +
			"thinker decides, a device shared in from another account and a request\n" +
			"another device's Serve or Funnel relays are refused too.\n\n" +
			"Bare `abcd dashboard` is `abcd dashboard status`. The server runs until `abcd\n" +
			"dashboard stop`, and never starts by itself. Its run file and the devices that\n" +
			"opened it live in `" + abcdhome.Display("dashboard/") + "`.\n\n" +
			"Exit 2 on a refusal, with nothing started or stopped.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return runDashboardStatus(cmd, *asJSON) },
	}
	cmd.AddCommand(newDashboardStartCommand(asJSON), newDashboardStopCommand(asJSON), &cobra.Command{
		Use: "status",
		Long: "Report whether the dashboard runs, where, since when, and the devices that\n" +
			"opened it in this run, each by its Tailscale device name and person. Writes\n" +
			"nothing.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return runDashboardStatus(cmd, *asJSON) },
	}, newDashboardServeCommand())
	return cmd
}

func newDashboardStartCommand(asJSON *bool) *cobra.Command {
	var port int
	c := &cobra.Command{
		Use: "start",
		Long: "Start the dashboard on this computer's own Tailscale addresses and print one\n" +
			"line saying where to open it and who can. It refuses outside a checkout abcd\n" +
			"manages, when a dashboard already runs on this computer, when Tailscale is not\n" +
			"running, and when the port is one Tailscale's own Serve or Funnel configuration\n" +
			"uses. It returns once a fetch of its own address through Tailscale answers. An\n" +
			"address the fetch cannot connect to at all is dropped, no longer listened on,\n" +
			"and named with why in the line; if no address answers, or one that connects\n" +
			"does not, the server is stopped and the failure named.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			const prefix = "abcd dashboard start: "
			if port < 1 || port > 65535 {
				return &exitError{Code: 2, Msg: fmt.Sprintf("%s--port %d is not a port (1 to 65535)", prefix, port)}
			}
			home, err := dashboardHome(prefix)
			if err != nil {
				return err
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			root, err := gitutil.CheckoutRoot(cwd, "the dashboard's checkout")
			if err != nil {
				return &exitError{Code: 2, Msg: prefix + err.Error() + " (nothing started)"}
			}
			if !ahoy.Managed(root) {
				return &exitError{Code: 2, Msg: prefix + "this checkout is not one abcd manages; run `abcd ahoy install` first (nothing started)"}
			}
			bin, err := tailscale.Resolve(exec.LookPath, tailscale.BundledCommands, root)
			if err != nil {
				return &exitError{Code: 2, Msg: prefix + err.Error() + " (nothing started)"}
			}
			launch, err := dashboard.DefaultLauncher()
			if err != nil {
				return fmt.Errorf("%s%w", prefix, err)
			}
			res, err := dashboard.Start(cmd.Context(), dashboard.StartOptions{
				Home: home, Root: root, Port: port, Tailscale: tailscale.New(bin), Launch: launch,
			})
			if err != nil {
				return dashboardError(prefix, home, err, "nothing started")
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				fmt.Fprintln(w, termsafe.Sanitize(res.Line))
			})
		},
	}
	c.Flags().IntVar(&port, "port", dashboardDefaultPort, "the port to listen on, on each of this computer's Tailscale addresses")
	return c
}

func newDashboardStopCommand(asJSON *bool) *cobra.Command {
	return &cobra.Command{
		Use: "stop",
		Long: "Stop the dashboard this computer runs. It signals only the process start\n" +
			"launched, checked by its process id, start time and executable just before the\n" +
			"signal, waits until its addresses answer nothing, and removes the run file. A\n" +
			"run file naming a process that is gone or is another program is removed with\n" +
			"nothing signalled. It never stops anything by name.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			const prefix = "abcd dashboard stop: "
			home, err := dashboardHome(prefix)
			if err != nil {
				return err
			}
			res, err := dashboard.Stop(home)
			if err != nil {
				return dashboardError(prefix, home, err, "nothing signalled")
			}
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) {
				switch {
				case res.Stopped:
					fmt.Fprintf(w, "abcd dashboard: stopped; %s answers nothing until `abcd dashboard start`\n", termsafe.Sanitize(res.URL))
				case res.Stale:
					fmt.Fprintln(w, "abcd dashboard: was not running (its run file named a process that is gone or is another program; removed, nothing signalled)")
				default:
					fmt.Fprintln(w, "abcd dashboard: not running; nothing to stop")
				}
			})
		},
	}
}

// newDashboardServeCommand is the hidden server process start launches. It
// refuses unless start's pipes are open, so nothing but start starts it.
func newDashboardServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:    "serve",
		Hidden: true,
		Long: "The dashboard server process. Only `abcd dashboard start` runs it, handing it\n" +
			"the pipes it reads its configuration from and reports on; run any other way it\n" +
			"refuses.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := dashboard.Serve(cmd.Context()); err != nil {
				var r *dashboard.Refusal
				if errors.As(err, &r) {
					return &exitError{Code: 2, Msg: "abcd dashboard serve: " + err.Error()}
				}
				return fmt.Errorf("abcd dashboard serve: %w", err)
			}
			return nil
		},
	}
}

func runDashboardStatus(cmd *cobra.Command, asJSON bool) error {
	const prefix = "abcd dashboard status: "
	home, err := dashboardHome(prefix)
	if err != nil {
		return err
	}
	st, err := dashboard.ReadStatus(home)
	if err != nil {
		return dashboardError(prefix, home, err, "nothing read")
	}
	return render(cmd.OutOrStdout(), asJSON, st, func(w io.Writer) { renderDashboardStatus(w, st) })
}

// renderDashboardStatus is the text form. Device and person names come from
// Tailscale and are sanitised before they reach the terminal.
func renderDashboardStatus(w io.Writer, st dashboard.Status) {
	if !st.Running {
		line := "abcd dashboard — not running; `abcd dashboard start` starts it"
		if st.Stale {
			line += " (a run file names a process that is gone; `abcd dashboard stop` removes it)"
		}
		fmt.Fprintln(w, line)
		return
	}
	fmt.Fprintf(w, "abcd dashboard — running at %s since %s, listening on %s\n",
		termsafe.Sanitize(st.URL), st.Since.Format("2006-01-02 15:04 MST"), termsafe.Sanitize(strings.Join(st.Addrs, " and ")))
	if len(st.Devices) == 0 {
		fmt.Fprintln(w, "  no device has opened it in this run")
		return
	}
	fmt.Fprintf(w, "  opened in this run by %s:\n", countOf(len(st.Devices), "device"))
	for _, d := range st.Devices {
		person := d.Person
		if person == "" {
			person = d.Login
		}
		fmt.Fprintf(w, "  %s — %s (%s), last %s\n", termsafe.Sanitize(d.Device), termsafe.Sanitize(person),
			termsafe.Sanitize(d.Login), d.Last.Format("2006-01-02 15:04 MST"))
	}
	fmt.Fprintln(w, "  a device is removed from the dashboard by removing it from your Tailscale network")
}

// dashboardHome is the person's home directory, or the refusal naming why
// there is none.
func dashboardHome(prefix string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", &exitError{Code: 2, Msg: prefix + "no home directory to keep the dashboard's run file in"}
	}
	return home, nil
}

// dashboardError maps the surface's errors to the front door's exits: a
// refusal is exit 2, anything else exit 1, each with the home redacted.
func dashboardError(prefix, home string, err error, nothing string) error {
	msg := fsutil.RedactHome(strings.ReplaceAll(err.Error(), home, "~"))
	var r *dashboard.Refusal
	if errors.As(err, &r) {
		return &exitError{Code: 2, Msg: prefix + msg + " (" + nothing + ")"}
	}
	return &exitError{Code: 1, Msg: prefix + msg}
}
