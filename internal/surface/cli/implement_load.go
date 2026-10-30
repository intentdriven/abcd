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
	"sync"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
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
	// loadMark is the command form's mark; nil is the core's /bin/sleep.
	loadMark func() (func(), error)
	// loadRenew renews the caller's own reservations half through their
	// lease, run every loadRenewEvery while the waiter waits and while its
	// command runs; loadRenewRound, when set, is called after each round has
	// been said.
	loadRenew      = func(root string) ([]claim.Renewal, error) { return claim.RenewDue(root, claim.Options{}) }
	loadRenewEvery = 5 * time.Minute
	loadRenewRound func()
	// loadRenewRuns renews the due reservations of the runs the waiter's
	// checkout serves (loop.RenewRuns), in each round beside loadRenew.
	loadRenewRuns = func(root string) []loop.RunRenewal { return loop.RenewRuns(root, time.Now()) }
	// loadRenewStopWait bounds how long the waiter, its command exited, waits
	// for a renewal round in flight to finish (dcn-2610292238191496): one
	// networked git command's own cap (gitutil.NetTimeout), so a push under
	// way ends, pushed or failed, and a minute for the local reads and the
	// run record's write that follow it.
	loadRenewStopWait = gitutil.NetTimeout + time.Minute
)

// newImplementLoadCommand builds `implement load`, the load check that runs once
// at the start of `make preflight` and of the eval harness
// (itd-2609231434459890). It formats implement.CheckLoad's result and holds no
// logic of its own. It exits 0 on every status: it warns, never refuses.
func newImplementLoadCommand(asJSON *bool) *cobra.Command {
	var site string
	var wait, race bool
	var timeout time.Duration
	cmd := &cobra.Command{
		Use: "load --site preflight|eval-harness | --wait [--timeout <duration>] [-- <command>...] | --race-gate [--timeout <duration>]",
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
			"the queue (one that started earlier, whatever its pid, or within two seconds of\n" +
			"it with the lower pid) or running its command, under any account: all of it is\n" +
			"read from the process table, which shows every account's processes: no file\n" +
			"another account can write is read; the gate reads only the process table (and,\n" +
			"for the wait limit, your own load-limits). A waiter ahead of yours still delays\n" +
			"yours until it opens or reaches its own --timeout, whatever account it runs\n" +
			"under and whatever its own account's wait limit is, with one exception: a\n" +
			"waiter ahead with a higher pid, as after the process ids wrap, may run an abcd\n" +
			"built before this order, which waits on every lower pid and so on yours, so once\n" +
			"such waiters alone have kept yours shut for " + implement.WaitCrossedCap.String() + " on end, and a step more\n" +
			"for each such waiter ahead beyond the first, yours stops counting them. A\n" +
			"preflight whose own\n" +
			"waiter is queued\n" +
			"takes that waiter's place in the queue, and a waiter beneath a waiter running its\n" +
			"command (`--wait -- make preflight`) takes that ancestor's place, ahead of every\n" +
			"queued waiter. Once it finds all of that clear it waits\n" +
			"a random " + implement.WaitJitterMin.String() + " to " + implement.WaitJitterMax.String() + " and checks everything again, and opens only if the\n" +
			"re-check is clear too, so two starters in one quiet minute do not start together.\n" +
			"With a command it then shows itself as running it and looks once more " + implement.WaitOpenPause.String() + "\n" +
			"later, before starting it: across accounts, which share no hold, two waiters can\n" +
			"open in one moment, and each stands back from the other unless it started earlier\n" +
			"and has the lower pid too: it gives up its hold, re-draws its jitter and waits\n" +
			"again (at most " + strconv.Itoa(implement.WaitBackoffCap) + " times), while the other runs its command. So the later\n" +
			"start stands back whatever the pids, and the earlier one from a lower pid as well,\n" +
			"which an older abcd (v0.15.1 and earlier) lets run.\n" +
			"The load limit is twice the cores because a busy multi-session machine already\n" +
			"idles near one core-count of load, so a limit of the cores never opens; one\n" +
			"waiter at a time is kept by the process table, the hold below and that last\n" +
			"look, not by the load number; and one race run adds about a core-count, so admitting below twice\n" +
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
			"With --race-gate in place of --wait it waits at the race gate, which `make\n" +
			"preflight` runs before its race stage. It takes no command after `--`: it gates\n" +
			"the race stage that follows it, and a race run of your own goes through\n" +
			"`--wait -- go test -race`. It is spelt without --wait because an\n" +
			"older abcd reads any `implement load --wait` as a waiter at its first gate, and\n" +
			"would count a preflight at its race gate as queued, not in its tests, so the two\n" +
			"are refused together. At one preflight at a time, the default, it opens at once,\n" +
			"without reading the machine, taking a hold or waiting a jitter, so a preflight's\n" +
			"race stage follows its tests exactly as before. Above one it is shut only while\n" +
			"another `go test -race` run is live, a race waiter is ahead of it (by the same\n" +
			"rule) or running its command, the load is at the wait limit, or the race gate's\n" +
			"own hold is in force (each gate is shut only by its own gate's hold). That keeps\n" +
			"race stages apart among the preflights whose checkouts have the race gate: one\n" +
			"at a time within an account, and across accounts all but a narrow window (a\n" +
			"race waiter gone before its race run shows, or, after a pid wrap, two waiters\n" +
			"whose start and pid disagree: one running v0.15.1, which counts a waiter ahead\n" +
			"only with the lower pid as well, or two of the start order whose whole-second\n" +
			"ages read a tie differently or the later of which stopped counting the earlier\n" +
			"past its " + implement.WaitCrossedCap.String() + " cap). An account at one opens its race gate at once\n" +
			"whatever another account set, and a checkout older than the gate runs its race\n" +
			"stage ungated, so set more than 1 only once no such checkout runs a preflight. A\n" +
			"first gate's waiter never queues behind a race waiter. A preflight whose race\n" +
			"waiter is queued runs nothing: at 1 it still counts as in its test stages, and\n" +
			"above 1 it takes no slot until its race run starts, which the race gate keeps\n" +
			"to one at a time. A waiter that has opened stands back only from a waiter that\n" +
			"opened in the same moment at its own gate, never from one at the other gate.\n\n" +
			"How many preflights the first gate admits to their test stages at once is your\n" +
			"own machine's setting: `" + implement.PreflightSlotsKey + "` in `" + abcdhome.Display("config.json") + "`, or " + implement.PreflightSlotsEnv + " in the\n" +
			"environment, which wins, for a trial; a whole number from 1 to " + strconv.Itoa(implement.MaxPreflightSlots) + ", 1 by default,\n" +
			"and never read from a repository file, so a checkout cannot raise the machine's\n" +
			"load. At 1 both gates read as before the setting existed. Above 1 the\n" +
			"first gate opens while fewer than that many other preflights or waiters running\n" +
			"their command are in their test stages (a `--wait -- make preflight` counts\n" +
			"once), a waiter queued ahead still shuts it, and a live race run does not shut\n" +
			"it by itself; the load limit still does, and the race gate keeps race stages\n" +
			"apart, so one race stage may run beside that many unit stages. A value it\n" +
			"cannot use refuses the wait, naming where it was set. A `" + abcdhome.Display("config.json") + "`\n" +
			"that cannot be read for it (writable by others, not valid JSON, or a `load`\n" +
			"that is not an object) is read past instead: one preflight at a time applies,\n" +
			"and one line on stderr names the file, the cause and how to fix it; so does a\n" +
			"key under `load` that nothing reads.\n\n" +
			"Within one account the waiter that opens also takes a hold,\n" +
			"`" + abcdhome.Display(implement.WaitHoldFileName) + "` at the first gate and `" + abcdhome.Display(implement.WaitRaceHoldFileName) + "` at\n" +
			"the race gate, and every other waiter of that account at the same gate stays\n" +
			"shut while it is in force; a hold taken by the waiter's own ancestor does not\n" +
			"shut it.\n" +
			"Without a command the hold lasts " + implement.WaitSettle.String() + ", long enough for the load average to show\n" +
			"the work admitted. With a command after `--` (`abcd implement load --wait -- go\n" +
			"test -race ./...`) the waiter runs it once the gate opens and exits with its exit code,\n" +
			"or with 128+N, as a shell does, when signal N killed it;\n" +
			"the hold lasts until the command has started its own race run, the command exits,\n" +
			"or " + strconv.Itoa(int(implement.WaitCommandCap.Minutes())) + " minutes pass, and a hold whose waiter has died, or whose pid now\n" +
			"belongs to another process (a different start time), is reclaimed by the next\n" +
			"waiter. A SIGINT or SIGTERM to the waiter is passed on to the command, and the\n" +
			"waiter waits for it to exit, exits with its code and releases the hold. A\n" +
			"command that is a race run (go, then `test` with -race) is gated as one for both\n" +
			"gates: above one preflight at a time it stays shut while another race run is\n" +
			"live, judges and takes the race gate's hold as well as its own, and from the\n" +
			"moment it opens every race gate and every other such command counts it as a\n" +
			"race run, so a second one waits for the first. A\n" +
			"symlinked " + abcdhome.Display() + " is refused. On a CI runner it opens at once.\n\n" +
			"While it waits, and while its command runs, in a checkout, the waiter keeps\n" +
			"your own reservations from lapsing: at once and then every five minutes it\n" +
			"renews each of your live\n" +
			"reservations that is half through its lease and whose latest commit this copy\n" +
			"pushed, for its own term or\n" +
			"two hours, whichever is longer, and says each renewal on standard error. Your\n" +
			"own reservations it only renews: it reserves nothing new, takes nothing over and\n" +
			"revives none that has lapsed, so a lease shorter than ten minutes can still lapse between two rounds;\n" +
			"a renewal that fails, or a reservation whose latest commit this copy did not push (or pushed before it kept a note of it), is said once, and the wait goes on.\n" +
			"On the same rounds it renews, as implement step does, the due reservation of\n" +
			"each run in progress that lives in this checkout, or in another worktree of\n" +
			"the repository with a lane whose worktree is this checkout, reviving one that\n" +
			"has lapsed and logging each in the run record, but not a run parked at land,\n" +
			"which the drain renews; while another invocation holds the checkout's run\n" +
			"lock the round skips that checkout's runs, and the next round renews them. Each is said\n" +
			"on standard error, a failure once until what is said of it changes. The\n" +
			"renewal runs beside the gate, never in it, so\n" +
			"the gate's order and its open never wait on one; it stops when the gated\n" +
			"command exits, letting a round in flight finish and be said first, for at most\n" +
			"as long as one networked git command may run and a minute more. Since it runs through git and gh, no waiter counts another's git or gh\n" +
			"child as its command: gate make or go, never git or gh itself.",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return nil
			}
			if race {
				// The race gate admits the race stage that follows it in the
				// preflight's own recipe; it never runs a command, so its
				// hold is always the settle hold.
				return &exitError{Code: 2, Msg: "abcd implement load: --race-gate takes no command: it gates the race stage that follows it, so a command to run goes after `--` with --wait (nothing waited for)"}
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
			if wait && race {
				return &exitError{Code: 2, Msg: "abcd implement load: --race-gate waits on its own, so it takes no --wait: an older abcd reads any --wait as a waiter at its first gate (nothing waited for)"}
			}
			if wait || race {
				if cmd.Flags().Changed("site") {
					flag := "--wait"
					if race {
						flag = "--race-gate"
					}
					return &exitError{Code: 2, Msg: "abcd implement load: " + flag + " takes no --site; the site names where the warning runs (nothing waited for)"}
				}
				if timeout < 0 {
					return &exitError{Code: 2, Msg: "abcd implement load: --timeout must not be negative (nothing waited for)"}
				}
				return runLoadWait(cmd, *asJSON, timeout, race, args)
			}
			if cmd.Flags().Changed("timeout") {
				return &exitError{Code: 2, Msg: "abcd implement load: --timeout is for --wait or --race-gate (nothing checked)"}
			}
			if !slices.Contains(implement.LoadSites(), site) {
				return &exitError{Code: 2, Msg: fmt.Sprintf("abcd implement load: --site is not one of: %s (the value given is not repeated here; nothing checked)",
					strings.Join(implement.LoadSites(), ", "))}
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
	cmd.Flags().BoolVar(&race, "race-gate", false, "wait at the race gate, before a race stage, instead of the first gate: open at once at one preflight at a time, and above that shut only by another go test -race run, a race waiter ahead, the load and the race gate's hold")
	cmd.Flags().DurationVar(&timeout, "timeout", implement.WaitDefaultTimeout, "with --wait or --race-gate, give up after this long and exit 1")
	return cmd
}

// runLoadWait is `implement load --wait`: it formats implement.WaitLoad's
// result, and in the command form runs the command once the gate opens and
// releases the hold when it exits.
func runLoadWait(cmd *cobra.Command, asJSON bool, timeout time.Duration, race bool, command []string) error {
	out := cmd.OutOrStdout()
	req := implement.WaitRequest{
		Read: loadReader, ReadArgs: loadArgsReader, Self: loadSelf(),
		Now: loadNow, Sleep: loadSleep, Timeout: timeout, Command: len(command) > 0, Race: race, Jitter: loadJitter,
		Mark: loadMark, RaceCommand: machineload.IsRaceCommand(command),
	}
	if !asJSON {
		req.OnWait = func(r implement.WaitResult) { fmt.Fprintf(out, "waiting for capacity: %s\n", waitConditions(r)) }
		req.OnSettle = func(j time.Duration) {
			fmt.Fprintf(out, "load gate: the machine looks free; checking everything again in %s before opening, so two starters in one quiet minute do not start together\n",
				j.Round(time.Second))
		}
		req.OnBackoff = func(first []int, j time.Duration) {
			fmt.Fprintf(out, "load gate: %s in the same moment, ahead of this one by start or by pid; standing back, and checking again in %s\n",
				pidsText(first, "another waiter opened", "waiters opened"), j.Round(time.Second))
		}
	}
	flag := "--wait"
	if race {
		flag = "--race-gate"
	}
	// The renewal goes on until the command exits, not only while the gate is
	// shut: an agent's preflight or go test runs for hours under the gate, and
	// nothing else renews a build's reservation meanwhile
	// (iss-2610261837062463). Every line the waiter itself says on standard
	// error goes through errOut too, so none is written beside a renewal's.
	errOut := sharedStderr(cmd.ErrOrStderr())
	stopRenewal := startHoldRenewal(errOut)
	defer stopRenewal()
	res, err := implement.WaitLoad(req)
	if err != nil {
		return &exitError{Code: 1, Msg: "abcd implement load " + flag + ": " + err.Error()}
	}
	if res.SlotsNote != "" {
		fmt.Fprintf(errOut, "abcd implement load %s: %s\n", flag, res.SlotsNote)
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
	unmark := func() {}
	if res.Unmark != nil {
		unmark = res.Unmark
	}
	code, runErr := runGated(cmd, command, errOut, unmark)
	unmark()
	if res.Hold != nil && res.Hold.Form == implement.HoldCommand {
		home, _ := os.UserHomeDir()
		if err := implement.ReleaseWaitHold(home, *res.Hold); err != nil {
			fmt.Fprintf(errOut, "abcd implement load --wait: the hold %s could not be released (%v); it lapses once this process is gone\n",
				abcdhome.Display(implement.HoldFileName(res.Hold.Gate)), err)
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

// startHoldRenewal renews the caller's own reservations while the waiter
// waits and while its command runs, so a reservation made by hand does not
// lapse while its preflight waits its turn (iss-2610170926295770): from the
// checkout the waiter runs in, at once and then every loadRenewEvery, it
// renews each of this person's live reservations that is half through its
// lease and whose latest commit this copy pushed (claim.RenewDue), and says
// each renewal on w. A renewal that fails, or a reservation left unrenewed,
// is said once until it succeeds, and the wait goes on.
//
// On the same rounds it renews the due reservation of each run the checkout
// serves (loop.RenewRuns; iss-2610261837062463, dcn-2610291352512058): a run
// in progress living in it, or in a sibling worktree with a lane whose
// worktree it is, so a build whose implementer holds its stage for hours,
// its go test and make preflight queued and running here, keeps its
// reservation with no step to renew it. Each run's renewal is said on w, and
// one that fails, or finds the reservation another's, is said once until
// what is said of it changes.
//
// It runs beside the gate, never in it: the gate's looks, its order and its
// open neither wait on a renewal nor read one. Stop starts no further round
// and waits, for at most loadRenewStopWait, for a round in flight to finish
// and be said, so a renewal it pushed is not left unsaid
// (dcn-2610292238191496); past that bound whatever the round says is
// dropped. Stop may be called more than once. Outside a checkout there is
// nothing to renew.
func startHoldRenewal(w io.Writer) (stop func()) {
	cwd, err := os.Getwd()
	if err != nil {
		return func() {}
	}
	root, err := gitutil.CheckoutRoot(cwd, "the reservations")
	if err != nil {
		return func() {}
	}
	var mu sync.Mutex
	stopped := false
	say := func(format string, a ...any) {
		mu.Lock()
		defer mu.Unlock()
		if !stopped {
			fmt.Fprintf(w, format, a...)
		}
	}
	// The seams are read here, once, so the goroutine reads no shared state.
	renew, renewRuns, every, round, stopWait := loadRenew, loadRenewRuns, loadRenewEvery, loadRenewRound, loadRenewStopWait
	done, finished := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(finished)
		tick := time.NewTicker(every)
		defer tick.Stop()
		failed := map[string]bool{}
		runSaid := map[string]string{}
		viewFailed := false
		for {
			got, err := renew(root)
			if err != nil && !viewFailed {
				say("abcd implement load --wait: could not read the reservations to renew them (%v); the wait goes on\n", err)
			}
			viewFailed = err != nil
			for _, r := range got {
				switch {
				case r.Err != nil && !failed[r.Record]:
					say("abcd implement load --wait: could not renew your reservation on %s (%v); the wait goes on\n", r.Record, r.Err)
				case r.Err == nil:
					say("abcd implement load --wait: renewed your reservation on %s, half its lease having passed while this waits, until %s\n",
						r.Record, r.Until.UTC().Format(time.RFC3339))
				}
				failed[r.Record] = r.Err != nil
			}
			for _, r := range renewRuns(root) {
				note := termsafe.Sanitize(fsutil.RedactHome(r.Note))
				key := r.Root + "\x00" + r.RunID
				if r.Renewed || runSaid[key] != note {
					say("abcd implement load --wait: run %s: %s\n", termsafe.Sanitize(r.RunID), note)
				}
				runSaid[key] = note
				if r.Renewed {
					delete(runSaid, key)
				}
			}
			if round != nil {
				round()
			}
			select {
			case <-done:
				return
			case <-tick.C:
			}
			// A tick and the stop can be ready at once, and select picks
			// either: the stop wins, so no round starts after it.
			select {
			case <-done:
				return
			default:
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			close(done)
			wait := time.NewTimer(stopWait)
			defer wait.Stop()
			select {
			case <-finished:
			case <-wait.C:
			}
			mu.Lock()
			stopped = true
			mu.Unlock()
		})
	}
}

// sharedStderr is the waiter's standard error as the gated command and the
// renewal's messages share it while the command runs. A file is passed to the
// command as it is, so the command writes to it directly, a terminal staying
// one. Any other writer is wrapped so each write holds a lock: the command's
// output is then copied into it by a goroutine of exec's, and a
// bytes.Buffer's own copy (ReadFrom) would drop every renewal message written
// while the command ran.
func sharedStderr(w io.Writer) io.Writer {
	if _, ok := w.(*os.File); ok {
		return w
	}
	return &lockedWriter{w: w}
}

// lockedWriter serializes the writes to w.
type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

// runGated runs the gated command with the verb's own streams, standard
// error as stderr gives it (sharedStderr), and returns its exit code; the error is a command that could not be started. A SIGINT or
// SIGTERM to the waiter is forwarded to the command, and the waiter goes on
// waiting for it: a waiter that died first would orphan the command, whose
// hold then reads as a dead holder's, and the next waiter would be admitted
// beside the orphan. A command killed by signal N returns 128+N, as a shell
// reports it, so a command stopped by SIGTERM never reads as one that failed
// with 1. started is called once the command has started, to stop the
// waiter's mark (implement.WaitResult.Unmark). The caller releases the hold
// once this returns.
func runGated(cmd *cobra.Command, command []string, stderr io.Writer, started func()) (int, error) {
	c := exec.Command(command[0], command[1:]...)
	c.Stdin, c.Stdout, c.Stderr = cmd.InOrStdin(), cmd.OutOrStdout(), stderr
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigs)
	if err := c.Start(); err != nil {
		return 0, err
	}
	started()
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
		if res.Gate == implement.GateRace && res.Slots <= 1 {
			fmt.Fprintf(w, "race gate open at once: %s\n", res.Reason)
			break
		}
		hold := "the hold its own caller took stands"
		if h := res.Hold; h != nil {
			if h.Form == implement.HoldCommand {
				hold = fmt.Sprintf("held until the command starts its own race run or exits, at most %s", h.Until.Sub(h.Opened).Round(time.Second))
			} else {
				hold = fmt.Sprintf("held for %s", h.Until.Sub(h.Opened).Round(time.Second))
			}
		}
		gate, clear := "load gate", "no other go test -race run, make preflight or waiter ahead is live"
		switch {
		case res.Gate == implement.GateRace:
			gate, clear = "race gate", "no other go test -race run or race waiter ahead is live"
		case res.Slots > 1:
			clear = fmt.Sprintf("fewer than %d other preflights or gated commands are in their test stages (%s) and no waiter is queued ahead", res.Slots, slotsText(res))
		}
		fmt.Fprintf(w, "%s open after %s: load %.1f is below %s (%s), and %s; %s, so one waiter starts at a time\n",
			gate, waited, res.Load, num(res.WaitLoad), waitLimitText(res), clear, hold)
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
			part := fmt.Sprintf("load %.1f is not below %s (%s)", res.Load, num(res.WaitLoad), waitLimitText(res))
			if len(res.UngatedTests) > 0 {
				part += ", with " + pidsText(res.UngatedTests, "a go test run outside the gate adding to it",
					"go test runs outside the gate adding to it")
			}
			parts = append(parts, part)
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

// slotsText says how many preflights the first gate admits to their test
// stages at once, and where that was set.
func slotsText(res implement.WaitResult) string {
	switch res.SlotsSource {
	case implement.SlotsEnv:
		return fmt.Sprintf("%d at once, set by %s", res.Slots, implement.PreflightSlotsEnv)
	case implement.SlotsMachine:
		return fmt.Sprintf("%d at once, set as %s in %s", res.Slots, implement.PreflightSlotsKey, abcdhome.Display("config.json"))
	}
	return fmt.Sprintf("%d at once", res.Slots)
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
