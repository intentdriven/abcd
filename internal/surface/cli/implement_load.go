package cli

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/spf13/cobra"
)

// loadReader, loadSelf, loadArgsReader and the clock are the verb's seams onto
// the real machine; tests swap them so no test depends on the machine's actual
// load, and no wait sleeps.
var (
	loadReader     = machineload.Read
	loadSelf       = func() machineload.Self { return machineload.Self{} }
	loadArgsReader = machineload.ReadArgs
	loadNow        = time.Now
	loadSleep      = time.Sleep
	// loadJitter is the waiter's settle jitter; nil is the core's random one.
	loadJitter func() time.Duration
)

// newImplementLoadCommand builds `implement load`, the load check that runs once
// at the start of `make preflight` and of the eval harness
// (itd-2609231434459890). It formats implement.CheckLoad's result and holds no
// logic of its own. It exits 0 on every status: it warns, never refuses.
func newImplementLoadCommand(asJSON *bool) *cobra.Command {
	var site string
	var wait bool
	var timeout time.Duration
	cmd := &cobra.Command{
		Use: "load --site preflight|eval-harness | --wait [--timeout <duration>] [-- <command>...]",
		Long: "Read the machine's load averages and process table once and warn when a program\n" +
			"outside the running work has used nearly all the CPU it could get for longer than\n" +
			"the stray limit, or when the one-minute load average is above the extreme limit.\n" +
			"What a program could get is its fair share: the online cores divided by the\n" +
			"one-minute load, and never more than one core. A lifetime CPU share of at least 0.9\n" +
			"of it makes a stray, so forty busy loops on 16 cores, each at 0.4 of a core, are all\n" +
			"strays, as one loop at a full core of an idle machine is. Your programs and other\n" +
			"accounts' are judged alike. `make preflight` runs it first, and the eval harness runs\n" +
			"it once at its start; it never runs once per test package. Without --wait it never\n" +
			"refuses, never waits and never stops anything, and it exits 0 on every status: ok, warning, skipped\n" +
			"(in CI, where the line says why) and unchecked (a platform other than macOS and\n" +
			"Linux, or a read that failed).\n\n" +
			"Your own strays are named with their pid, process group, age and CPU share, with\n" +
			"commands to stop them that re-check each target first and never match by pattern;\n" +
			"names the private banned-names layer matches are masked. Other accounts' strays\n" +
			"appear only as a count and a total CPU share. The check's own parent chain is never\n" +
			"a stray. Inside an autonomous run (a run state with a joined session) a warning is\n" +
			"also written to the run log as a `load` event.\n\n" +
			"The limits are per machine, in `" + abcdhome.Display("load-limits") + "`, which the check reads and\n" +
			"never creates. `#` starts a comment; every other line is `<key> <value>`:\n\n" +
			"  stray-minutes 30   minutes at nearly all its share before a program is a stray (1 to 10080)\n" +
			"  extreme-load 64    the one-minute load above which the machine is overloaded\n\n" +
			"Either key may be omitted. The defaults are 30 minutes and four times the online\n" +
			"core count. The file must be a regular file you own that nobody else can write, at\n" +
			"most 4 KiB; a file that is not, or that holds an unknown key, a repeated key or a\n" +
			"value out of range, is reported loudly and every default is used.\n\n" +
			"With --wait it waits for capacity instead of warning, and it is the one waiter\n" +
			"concurrent sessions gate a preflight or a race run with, in place of a hand-written\n" +
			"loop; `make preflight` runs its test stages through it. It blocks until the\n" +
			"one-minute load average is below the wait limit (`wait-load <load>` in the same\n" +
			"file; by default twice the online core count), no other `go test -race` run is\n" +
			"live, no other `make preflight` is running and no other waiter is ahead of it in\n" +
			"the queue (a lower pid) or running its command, under any account: all of it is\n" +
			"read from the process table, which shows every account's processes, and nothing\n" +
			"from a file another account could write. A preflight whose own waiter is queued\n" +
			"takes that waiter's place in the queue, and a waiter beneath a waiter running its\n" +
			"command (`--wait -- make preflight`) takes that ancestor's place, ahead of every\n" +
			"queued waiter. Once it finds all of that clear it waits\n" +
			"a random " + implement.WaitJitterMin.String() + " to " + implement.WaitJitterMax.String() + " and checks everything again, and opens only if the\n" +
			"re-check is clear too, so two starters in one quiet minute do not start together.\n" +
			"The load limit is twice the cores because a busy multi-session machine already\n" +
			"idles near one core-count of load, so a limit of the cores never opens; one\n" +
			"waiter at a time is kept by the process table and the hold below, not by the\n" +
			"load number; and one race run adds about a core-count, so admitting below twice\n" +
			"the cores bounds the peak near three times them, under the extreme limit.\n" +
			"A race run is a process executing go whose go command is `test` with -race, a\n" +
			"preflight a make whose goals name preflight, and a waiter an abcd running\n" +
			"`implement load --wait`, each read from its arguments, never from a pattern over a\n" +
			"command line, and the waiter's own process and ancestors are never one: a\n" +
			"hand-written `pgrep -f \"go test -race\"` matched its own shell and never opened.\n" +
			"It reads the machine every " + implement.WaitPollInterval.String() + ", prints one line when it starts waiting and\n" +
			"one when it first finds the machine free, and nothing more until it opens, and\n" +
			"gives up after --timeout (" + implement.WaitDefaultTimeout.String() + " by default), exiting 1 and naming what it\n" +
			"was still waiting on. --json reports what it waited on and for how long.\n\n" +
			"Within one account the waiter that opens also takes a hold,\n" +
			"`" + abcdhome.Display(implement.WaitHoldFileName) + "`, and every other waiter of that account stays shut while\n" +
			"it is in force; a hold taken by the waiter's own ancestor does not shut it.\n" +
			"Without a command the hold lasts " + implement.WaitSettle.String() + ", long enough for the load average to show\n" +
			"the work admitted. With a command after `--` (`abcd implement load --wait -- go\n" +
			"test -race ./...`) the waiter runs it once the gate opens and exits with its exit code,\n" +
			"or with 128+N, as a shell does, when signal N killed it;\n" +
			"the hold lasts until the command has started its own race run, the command exits,\n" +
			"or " + strconv.Itoa(int(implement.WaitCommandCap.Minutes())) + " minutes pass, and a hold whose waiter has died, or whose pid now\n" +
			"belongs to another process (a different start time), is reclaimed by the next\n" +
			"waiter. A SIGINT or SIGTERM to the waiter is passed on to the command, and the\n" +
			"waiter waits for it to exit, exits with its code and releases the hold. A\n" +
			"symlinked " + abcdhome.Display() + " is refused. On a CI runner it opens at once.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return nil
			}
			if !wait {
				return &exitError{Code: 2, Msg: "abcd implement load: a command to run is given only with --wait, after `--` (nothing checked)"}
			}
			if cmd.ArgsLenAtDash() != 0 {
				return &exitError{Code: 2, Msg: "abcd implement load: the command to run goes after `--`: abcd implement load --wait -- <command> (nothing waited for)"}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if wait {
				if cmd.Flags().Changed("site") {
					return &exitError{Code: 2, Msg: "abcd implement load: --wait takes no --site; the site names where the warning runs (nothing waited for)"}
				}
				if timeout < 0 {
					return &exitError{Code: 2, Msg: "abcd implement load: --timeout must not be negative (nothing waited for)"}
				}
				return runLoadWait(cmd, *asJSON, timeout, args)
			}
			if cmd.Flags().Changed("timeout") {
				return &exitError{Code: 2, Msg: "abcd implement load: --timeout is for --wait (nothing checked)"}
			}
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
	cmd.Flags().BoolVar(&wait, "wait", false, "wait until the load is below the wait limit and no other go test -race run is live")
	cmd.Flags().DurationVar(&timeout, "timeout", implement.WaitDefaultTimeout, "with --wait, give up after this long and exit 1")
	return cmd
}

// runLoadWait is `implement load --wait`: it formats implement.WaitLoad's
// result, and in the command form runs the command once the gate opens and
// releases the hold when it exits.
func runLoadWait(cmd *cobra.Command, asJSON bool, timeout time.Duration, command []string) error {
	out := cmd.OutOrStdout()
	req := implement.WaitRequest{
		Read: loadReader, ReadArgs: loadArgsReader, Self: loadSelf(),
		Now: loadNow, Sleep: loadSleep, Timeout: timeout, Command: len(command) > 0, Jitter: loadJitter,
	}
	if !asJSON {
		req.OnWait = func(r implement.WaitResult) { fmt.Fprintf(out, "waiting for capacity: %s\n", waitConditions(r)) }
		req.OnSettle = func(j time.Duration) {
			fmt.Fprintf(out, "load gate: the machine looks free; checking everything again in %s before opening, so two starters in one quiet minute do not start together\n",
				j.Round(time.Second))
		}
	}
	res, err := implement.WaitLoad(req)
	if err != nil {
		return &exitError{Code: 1, Msg: "abcd implement load --wait: " + err.Error()}
	}
	if err := render(out, asJSON, res, func(w io.Writer) { renderLoadWait(w, res) }); err != nil {
		return err
	}
	if res.Status == implement.WaitTimedOut {
		return &exitError{Code: 1}
	}
	if len(command) == 0 {
		return nil
	}
	code, runErr := runGated(cmd, command)
	if res.Hold != nil && res.Hold.Form == implement.HoldCommand {
		home, _ := os.UserHomeDir()
		if err := implement.ReleaseWaitHold(home, *res.Hold); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "abcd implement load --wait: the hold %s could not be released (%v); it lapses once this process is gone\n",
				abcdhome.Display(implement.WaitHoldFileName), err)
		}
	}
	if runErr != nil {
		return &exitError{Code: 127, Msg: "abcd implement load --wait: the command could not be started: " + runErr.Error()}
	}
	if code != 0 {
		return &exitError{Code: code}
	}
	return nil
}

// runGated runs the gated command with the verb's own streams and returns its
// exit code; the error is a command that could not be started. A SIGINT or
// SIGTERM to the waiter is forwarded to the command, and the waiter goes on
// waiting for it: a waiter that died first would orphan the command, whose
// hold then reads as a dead holder's, and the next waiter would be admitted
// beside the orphan. A command killed by signal N returns 128+N, as a shell
// reports it, so a command stopped by SIGTERM never reads as one that failed
// with 1. The caller releases the hold once this returns.
func runGated(cmd *cobra.Command, command []string) (int, error) {
	c := exec.Command(command[0], command[1:]...)
	c.Stdin, c.Stdout, c.Stderr = cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr()
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigs)
	if err := c.Start(); err != nil {
		return 0, err
	}
	done := make(chan error, 1)
	go func() { done <- c.Wait() }()
	var err error
wait:
	for {
		select {
		case sig := <-sigs:
			_ = c.Process.Signal(sig)
		case err = <-done:
			break wait
		}
	}
	var exit *exec.ExitError
	switch {
	case err == nil:
		return 0, nil
	case errors.As(err, &exit):
		if code := exit.ExitCode(); code > 0 {
			return code, nil
		}
		if sig, ok := signalledBy(exit.ProcessState); ok {
			return 128 + sig, nil
		}
		return 1, nil
	default:
		return 0, err
	}
}

// renderLoadWait writes the waiter's closing line.
func renderLoadWait(w io.Writer, res implement.WaitResult) {
	waited := (time.Duration(res.WaitedS * float64(time.Second))).Round(time.Second)
	switch res.Status {
	case implement.WaitSkipped, implement.WaitUnchecked:
		fmt.Fprintf(w, "load gate open without waiting: %s\n", res.Reason)
	case implement.WaitTimedOut:
		fmt.Fprintf(w, "load gate still shut after %s: %s\n", waited, waitConditions(res))
	case implement.WaitOpen:
		hold := "the hold its own caller took stands"
		if h := res.Hold; h != nil {
			if h.Form == implement.HoldCommand {
				hold = fmt.Sprintf("held until the command starts its own race run or exits, at most %s", h.Until.Sub(h.Opened).Round(time.Second))
			} else {
				hold = fmt.Sprintf("held for %s", h.Until.Sub(h.Opened).Round(time.Second))
			}
		}
		fmt.Fprintf(w, "load gate open after %s: load %.1f is below %s (%s), and no other go test -race run, make preflight or waiter ahead is live; %s, so one waiter starts at a time\n",
			waited, res.Load, num(res.WaitLoad), waitLimitText(res), hold)
	}
	if res.Malformed != "" {
		fmt.Fprintf(w, "LOAD CHECK SETTINGS UNUSABLE: %s: %s; the wait limit took its default\n", implement.LimitsFileDisplay, res.Malformed)
	}
}

// waitConditions names what holds the gate shut.
func waitConditions(res implement.WaitResult) string {
	var parts []string
	for _, c := range res.Still {
		switch c {
		case implement.WaitOnLoad:
			parts = append(parts, fmt.Sprintf("load %.1f is not below %s (%s)", res.Load, num(res.WaitLoad), waitLimitText(res)))
		case implement.WaitOnRaceRun:
			pids := make([]string, len(res.RaceRuns))
			for i, p := range res.RaceRuns {
				pids[i] = strconv.Itoa(p)
			}
			what := "a go test -race run is live (pid "
			if len(pids) > 1 {
				what = fmt.Sprintf("%d go test -race runs are live (pids ", len(pids))
			}
			parts = append(parts, what+strings.Join(pids, ", ")+")")
		case implement.WaitOnPreflight:
			parts = append(parts, pidsText(res.Preflights, "another make preflight is running", "make preflights are running"))
		case implement.WaitOnWaiter:
			parts = append(parts, pidsText(res.Waiters, "another waiter is ahead in the queue or running its command",
				"waiters are ahead in the queue or running their commands"))
		case implement.WaitOnLock:
			if h := res.Holder; h != nil {
				parts = append(parts, fmt.Sprintf("another waiter holds the gate (pid %d, until %s at the latest)", h.PID, h.Until.Local().Format("15:04:05")))
			} else {
				parts = append(parts, "another waiter is taking the gate")
			}
		case implement.WaitOnUnreadable:
			why := "the machine's load or process table could not be read"
			if res.Reason != "" {
				why += " (" + res.Reason + ")"
			}
			parts = append(parts, why)
		}
	}
	return strings.Join(parts, "; ")
}

// pidsText names pids after one or, for several, a count and many.
func pidsText(pids []int, one, many string) string {
	list := make([]string, len(pids))
	for i, p := range pids {
		list[i] = strconv.Itoa(p)
	}
	if len(pids) > 1 {
		return fmt.Sprintf("%d %s (pids %s)", len(pids), many, strings.Join(list, ", "))
	}
	return one + " (pid " + strings.Join(list, ", ") + ")"
}

// waitLimitText says where the wait limit came from.
func waitLimitText(res implement.WaitResult) string {
	if res.LimitSource == implement.LimitsFile {
		return "set in " + implement.LimitsFileDisplay
	}
	return fmt.Sprintf("%d x %d online cores", machineload.DefaultWaitFactor, res.Cores)
}

// renderLoad writes the check's report.
func renderLoad(w io.Writer, res implement.LoadResult) {
	if res.Limits != nil && res.Limits.Malformed != "" {
		sep := ": "
		if strings.HasPrefix(res.Limits.Malformed, "line ") {
			sep = " "
		}
		fmt.Fprintf(w, "LOAD CHECK SETTINGS UNUSABLE: %s%s%s; using the defaults for every limit: %d min, load %s (%d x %s)\n",
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
		fmt.Fprintf(w, "  Your programs at nearly all the CPU they can get, for over %s:\n", strayLimit)
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
		fmt.Fprintf(w, "  Other accounts: %d programs at nearly all the CPU they can get, for over %s, using about %.1f cores.\n",
			res.OtherStrays.Count, strayLimit, res.OtherStrays.Cores)
	}
	if res.Load != nil {
		// What one program can get at this load, said only when it is under a
		// core, so a stray at 40% of a core reads as the whole of its share.
		fair := machineload.FairShare(machineload.Snapshot{Load1: res.Load.One, HasLoad: true, Cores: res.Cores})
		share := ""
		if stray && fair < 1 {
			share = fmt.Sprintf("a program can get about %d%% of a core", int(math.Round(fair*100)))
		}
		if extreme {
			derivation := fmt.Sprintf("%d x %s", machineload.DefaultExtremeFactor, coresText(res))
			if lim.ExtremeFromFile {
				derivation = "set in " + implement.LimitsFileDisplay
			}
			fmt.Fprintf(w, "  Load %.1f over 1 min (%.1f over 5, %.1f over 15) is above the extreme limit of %s (%s).\n",
				res.Load.One, res.Load.Five, res.Load.Fifteen, num(lim.ExtremeLoad), derivation)
			if share != "" {
				fmt.Fprintf(w, "  At that load %s.\n", share)
			}
		} else if share != "" {
			fmt.Fprintf(w, "  Load %.1f on %s: %s.\n", res.Load.One, coresText(res), share)
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
