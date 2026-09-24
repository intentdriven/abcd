package cli

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/spf13/cobra"
)

// loadReader and loadSelf are the verb's two seams onto the real machine; tests
// swap them so no test depends on the machine's actual load.
var (
	loadReader = machineload.Read
	loadSelf   = func() machineload.Self { return machineload.Self{} }
)

// newImplementLoadCommand builds `implement load`, the load check that runs once
// at the start of `make preflight` and of the eval harness
// (itd-2609231434459890). It formats implement.CheckLoad's result and holds no
// logic of its own. It exits 0 on every status: it warns, never refuses.
func newImplementLoadCommand(asJSON *bool) *cobra.Command {
	var site string
	cmd := &cobra.Command{
		Use:   "load --site preflight|eval-harness",
		Short: "Check the machine's load before abcd's own tests start; warns, never refuses (exit 0)",
		Long: "Read the machine's load averages and process table once and warn when a program\n" +
			"outside the running work has held a near-full core (a lifetime CPU share of 0.9 or\n" +
			"more) for longer than the stray limit, or when the one-minute load average is above\n" +
			"the extreme limit. `make preflight` runs it first, and the eval harness runs it once\n" +
			"at its start; it never runs once per test package. It never refuses, never waits and\n" +
			"never stops anything, and it exits 0 on every status: ok, warning, skipped (in CI,\n" +
			"where the line says why) and unchecked (a platform other than macOS and\n" +
			"Linux, or a read that failed).\n\n" +
			"Your own strays are named with their pid, process group, age and CPU share, with\n" +
			"commands to stop them that re-check each target first and never match by pattern;\n" +
			"names the private banned-names layer matches are masked. Other accounts' strays\n" +
			"appear only as a count and a total CPU share. The check's own parent chain is never\n" +
			"a stray. Inside an autonomous run (a run state with a joined session) a warning is\n" +
			"also written to the run log as a `load` event.\n\n" +
			"The limits are per machine, in `~/.abcd/load-limits`, which the check reads and\n" +
			"never creates. `#` starts a comment; every other line is `<key> <value>`:\n\n" +
			"  stray-minutes 30   minutes at a near-full core before a program is a stray (1 to 10080)\n" +
			"  extreme-load 64    the one-minute load above which the machine is overloaded\n\n" +
			"Either key may be omitted. The defaults are 30 minutes and four times the online\n" +
			"core count. The file must be a regular file you own that nobody else can write, at\n" +
			"most 4 KiB; a file that is not, or that holds an unknown key, a repeated key or a\n" +
			"value out of range, is reported loudly and both defaults are used.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !slices.Contains(implement.LoadSites(), site) {
				return &exitError{Code: 2, Msg: fmt.Sprintf("abcd implement load: --site %q is not one of: %s (nothing checked)",
					site, strings.Join(implement.LoadSites(), ", "))}
			}
			req := implement.LoadRequest{Site: site, Read: loadReader, Self: loadSelf()}
			if cwd, err := os.Getwd(); err == nil {
				if root, err := gitutil.CheckoutRoot(cwd, "the run state"); err == nil {
					req.RepoRoot = root
				}
			}
			res := implement.CheckLoad(req)
			return render(cmd.OutOrStdout(), *asJSON, res, func(w io.Writer) { renderLoad(w, res) })
		},
	}
	cmd.Flags().StringVar(&site, "site", "", "where the check runs: "+enumHelp(implement.LoadSites()))
	return cmd
}

// renderLoad writes the check's report.
func renderLoad(w io.Writer, res implement.LoadResult) {
	if res.Limits != nil && res.Limits.Malformed != "" {
		sep := ": "
		if strings.HasPrefix(res.Limits.Malformed, "line ") {
			sep = " "
		}
		fmt.Fprintf(w, "LOAD CHECK SETTINGS UNUSABLE: %s%s%s; using the defaults for both limits: %d min, load %s (%d x %s)\n",
			implement.LimitsFileDisplay, sep, res.Limits.Malformed, res.Limits.StrayMinutes,
			num(res.Limits.ExtremeLoad), machineload.DefaultExtremeFactor, coresText(res))
	}
	switch res.Status {
	case implement.LoadSkipped:
		fmt.Fprintf(w, "load check (%s): %s\n", res.Site, res.Reason)
	case implement.LoadOK:
		fmt.Fprintf(w, "load check (%s): nothing to warn about; load %.1f on %s\n", res.Site, res.Load.One, coresText(res))
	case implement.LoadUnchecked:
		if res.Load == nil {
			fmt.Fprintf(w, "LOAD CHECK UNAVAILABLE (%s): %s; carrying on without checking the machine's load\n", res.Site, res.Reason)
			break
		}
		fmt.Fprintf(w, "LOAD CHECK UNAVAILABLE (%s): %s; carrying on without checking for stray programs; load %.1f on %s\n",
			res.Site, res.Reason, res.Load.One, coresText(res))
	case implement.LoadWarning:
		renderLoadWarning(w, res)
	}
	if res.RunLog.Error != "" {
		fmt.Fprintf(w, "could not write the load warning to the run log: %s; the warning above stands\n", res.RunLog.Error)
	}
}

// renderLoadWarning writes the warning block. It is a pure function of the facts
// a `load` event carries, so a logged event renders exactly what was printed.
func renderLoadWarning(w io.Writer, res implement.LoadResult) {
	stray := slices.Contains(res.Triggers, machineload.TriggerStray)
	extreme := slices.Contains(res.Triggers, machineload.TriggerExtreme)
	lim := res.Limits
	if lim == nil {
		lim = &implement.LoadLimits{}
	}
	strayLimit := fmt.Sprintf("%d min", lim.StrayMinutes)
	if lim.StrayFromFile {
		strayLimit += " (set in " + implement.LimitsFileDisplay + ")"
	}

	if stray {
		fmt.Fprintf(w, "LOAD WARNING (%s): programs that are not abcd's tests are keeping this machine busy.\n", res.Site)
	} else {
		fmt.Fprintf(w, "LOAD WARNING (%s): this machine's load is above the extreme limit.\n", res.Site)
	}
	if res.Reason != "" {
		fmt.Fprintf(w, "  Only the load was checked: %s.\n", res.Reason)
	}
	if len(res.OwnStrays) > 0 {
		fmt.Fprintf(w, "  Your programs at a near-full core for over %s:\n", strayLimit)
		width := 0
		for _, s := range res.OwnStrays {
			width = max(width, len(s.Name))
		}
		for _, s := range res.OwnStrays {
			fmt.Fprintf(w, "    %-*s    pid %d  group %d  running %s  CPU %d%% of a core\n",
				width, s.Name, s.PID, s.PGID, ageText(s.AgeS), s.CPUPct)
		}
		if res.OwnStraysMore > 0 {
			fmt.Fprintf(w, "    and %d more\n", res.OwnStraysMore)
		}
		if res.NamesWithheld {
			fmt.Fprintln(w, "  Their names are withheld: the private banned-names layer could not be read.")
		}
	}
	if len(res.Remedy) > 0 {
		fmt.Fprintln(w, "  To stop them, if you did not mean to leave them running, check each target first:")
		for _, r := range res.Remedy {
			if r.Form == machineload.RemedyGroup {
				pids := make([]string, len(r.PIDs))
				for i, p := range r.PIDs {
					pids[i] = strconv.Itoa(p)
				}
				fmt.Fprintf(w, "    pgrep -g %d        lists only %s, then:  kill -- -%d\n", r.PGID, strings.Join(pids, " "), r.PGID)
				continue
			}
			why := ""
			switch r.Why {
			case machineload.WhyMixedGroup:
				why = "   (its group holds other programs)"
			case machineload.WhyCallersGroup:
				why = "   (its group is the one this check runs in)"
			}
			fmt.Fprintf(w, "    ps -o pid=,comm= -p %d   still shows it, then:  kill %d%s\n", r.PID, r.PID, why)
		}
		fmt.Fprintln(w, "  Never by pattern (pkill -f, killall): a pattern also matches other sessions' programs.")
	}
	if res.OtherStrays.Count > 0 {
		fmt.Fprintf(w, "  Other accounts: %d programs at a near-full core for over %s, using about %.1f cores.\n",
			res.OtherStrays.Count, strayLimit, res.OtherStrays.Cores)
	}
	if res.Load != nil {
		if extreme {
			derivation := fmt.Sprintf("%d x %s", machineload.DefaultExtremeFactor, coresText(res))
			if lim.ExtremeFromFile {
				derivation = "set in " + implement.LimitsFileDisplay
			}
			fmt.Fprintf(w, "  Load %.1f over 1 min (%.1f over 5, %.1f over 15) is above the extreme limit of %s (%s).\n",
				res.Load.One, res.Load.Five, res.Load.Fifteen, num(lim.ExtremeLoad), derivation)
		} else {
			fmt.Fprintf(w, "  Load %.1f on %s.\n", res.Load.One, coresText(res))
		}
	}
	if len(res.OwnStrays) > 0 {
		fmt.Fprintln(w, "abcd carries on; stopping them is your call.")
	} else {
		fmt.Fprintln(w, "abcd carries on.")
	}
}

// coresText names the core count and where it came from.
func coresText(res implement.LoadResult) string {
	if res.CoresSource == implement.CoresRuntime {
		return fmt.Sprintf("%d cores (the online count could not be read)", res.Cores)
	}
	return fmt.Sprintf("%d online cores", res.Cores)
}

// ageText renders an age as the warning prints it: days and hours, hours and
// minutes, or minutes.
func ageText(secs int64) string {
	d := time.Duration(secs) * time.Second
	switch {
	case d >= 24*time.Hour:
		return fmt.Sprintf("%dd %02dh", int(d/(24*time.Hour)), int(d%(24*time.Hour)/time.Hour))
	case d >= time.Hour:
		return fmt.Sprintf("%dh %02dm", int(d/time.Hour), int(d%time.Hour/time.Minute))
	default:
		return fmt.Sprintf("%dm", int(d/time.Minute))
	}
}

// num prints a limit without trailing zeros: 64, 80.5.
func num(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
