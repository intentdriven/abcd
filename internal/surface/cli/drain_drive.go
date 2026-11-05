package cli

// drain_drive.go is `abcd drain --drive` (itd-2610222215485265,
// spc-2610230813348813; adr-2610222222326104): its preconditions, each refused
// before anything is reserved and naming what is missing; its consent; and the
// terminal loop started on the walled login. The loop itself is core
// (loop.RunTerminal).

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/procgroup"
	"github.com/intentdriven/abcd/internal/termsafe"
	"github.com/spf13/cobra"
)

// driveDefaultCap is the lanes a drive opens when --max does not say.
const driveDefaultCap = 8

// driveSessionMarkers are the variables a harness sets for what runs inside its
// session. A drive starts only from a command the person typed in a terminal.
var driveSessionMarkers = []string{"CLAUDECODE", "CLAUDE_CODE_SESSION_ID", "CLAUDE_CODE_ENTRYPOINT"}

// driveRoles are the roles a drive starts, each of which must have somewhere to
// run with no host.
var driveRoles = []string{loop.RoleDrainJudge, loop.RoleImplementer, loop.RoleRuthless, loop.RoleSecurity}

// driveWallChecks are the walls a drive proves. static writes nothing it cannot
// write again and spends nothing, and returns the lines to say; live starts one
// short launch, so it runs after the yes.
type driveWallChecks struct {
	static func(ctx context.Context, p runner.Profile) ([]string, error)
	live   func(ctx context.Context, p runner.Profile) error
}

// driveWalls is the one precondition the wall checks enter through
// (runner/wall.go). The settings are written at each start, so a new binary
// re-wires them, and a file a person edited is replaced and said.
var driveWalls = driveWallChecks{
	static: func(ctx context.Context, p runner.Profile) ([]string, error) {
		bin := runner.SelfBinary()
		replaced, err := runner.WriteWallSettings(p, bin)
		if err != nil {
			return nil, err
		}
		var lines []string
		if replaced {
			lines = append(lines, "the profile directory's settings were edited, and were replaced with abcd's: a drive's agents hold abcd's hooks and nothing else")
		}
		if err := runner.CheckWallSettings(p, bin); err != nil {
			return lines, err
		}
		return lines, runner.CheckLogin(ctx, p)
	},
	live: func(ctx context.Context, p runner.Profile) error {
		return runner.ProveWall(ctx, p, runner.SelfBinary())
	},
}

// driveLockRel is the lock one drive holds on a checkout from its yes to its end,
// and driveLockWait how long a second drive tries for it.
const (
	driveLockRel  = ".abcd/.work.local/run/drive.lock"
	driveLockWait = 2 * time.Second
)

// driveTerminal is loop.RunTerminal, a seam the tests replace.
var driveTerminal = loop.RunTerminal

// drivePreflightProgram is what a lane's preflight starts; a variable so a test
// can read the environment it is given.
var drivePreflightProgram = func() (string, []string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", nil, fmt.Errorf("abcd cannot name its own binary: %w", err)
	}
	return self, []string{"implement", "load", "--wait", "--", "make", "preflight"}, nil
}

// driveGateFiles are what a lane's preflight runs, as paths its diff may touch:
// a change to one is a change to what judges the lane.
var driveGateFiles = []string{"Makefile", "GNUmakefile", "makefile", "scripts/"}

// driveGateCheck says whether a lane's preflight may run: not over a lane that
// changes what the preflight runs, unless the person agreed with
// --trust-lane-gates, and not when what the lane changed cannot be read.
func driveGateCheck(trust bool) func(worktree, base string) error {
	return func(worktree, base string) error {
		if trust {
			return nil
		}
		out, err := gitutil.Run(worktree, "diff", "--name-only", base, "HEAD", "--")
		if err != nil {
			return &loop.PreflightRefused{Reason: "what the lane changed could not be read, so its preflight is not run"}
		}
		for _, path := range strings.Split(out, "\n") {
			for _, gate := range driveGateFiles {
				if path == gate || (strings.HasSuffix(gate, "/") && strings.HasPrefix(path, gate)) {
					return &loop.PreflightRefused{Reason: "the lane changes " + termsafe.Sanitize(path) + ", which its preflight runs, so it is not run; a drive started with --trust-lane-gates runs it"}
				}
			}
		}
		return nil
	}
}

// driveFlags are what `abcd drain --drive` was given.
type driveFlags struct {
	yes      bool
	maxLanes int
	takeOver bool
	skips    drainSkips
	// pace, subAgents and fixRounds are the flags as given, nil for one not.
	pace, subAgents, fixRounds *string
	// trustGates is --trust-lane-gates.
	trustGates bool
	// until is the run's bound; zero leaves each wait its eight hours.
	until time.Time
}

// parseDriveUntil reads --until: an RFC 3339 instant, or a date, read as the end of
// that day on this machine's clock. "" is no bound. A refusal never repeats the
// value given.
func parseDriveUntil(given string, now time.Time) (time.Time, error) {
	if given == "" {
		return time.Time{}, nil
	}
	at, err := time.Parse(time.RFC3339, given)
	if err != nil {
		day, derr := time.ParseInLocation(time.DateOnly, given, time.Local)
		if derr != nil {
			return time.Time{}, driveRefusal("--until is not a date (YYYY-MM-DD) or an RFC 3339 instant (the value given is not repeated here)")
		}
		at = time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 0, time.Local)
	}
	if !at.After(now) {
		return time.Time{}, driveRefusal("--until names a time already past, so the run would end before its first move")
	}
	return at, nil
}

// driveRefusal is a precondition's refusal, naming that nothing was started.
func driveRefusal(format string, args ...any) error {
	return &exitError{Code: 2, Msg: "abcd drain --drive: " + termsafe.Sanitize(fsutil.RedactHome(fmt.Sprintf(format, args...))) + " (nothing started, nothing reserved)"}
}

// drivePreconditions refuses what a drive cannot start on, in the order the
// person can act on it, and answers the configuration and the profile.
func drivePreconditions(ctx context.Context, cmd *cobra.Command, repoRoot string, roots layered.Roots) (*runner.Config, runner.Profile, error) {
	var p runner.Profile
	for _, k := range driveSessionMarkers {
		if os.Getenv(k) != "" {
			return nil, p, driveRefusal("%s is set, so this is an agent's own session; a drive starts only from a command the person types in a plain terminal", k)
		}
	}
	if held := runner.HeldCredentials(); len(held) > 0 {
		return nil, p, driveRefusal("%s is set in the environment, and a drive runs on this account's subscription alone until %s covers how a key reaches the harness and what it may spend; unset it",
			strings.Join(held, ", "), runner.BudgetIntent)
	}
	cfg, err := loadRunners(cmd, roots)
	if err != nil {
		return nil, p, driveRefusal("%s", err.Error())
	}
	if _, ok := cfg.Runner(runner.Claude); !ok {
		return nil, p, driveRefusal("the claude runner is not enabled on this machine; set runner.claude in %s", layered.Config.MachineOrigin())
	}
	if err := runner.AdmitClaude(repoRoot); err != nil {
		return nil, p, driveRefusal("%s", err.Error())
	}
	if cfg.JudgeOnOpenCode() {
		return nil, p, driveRefusal("the drain's judge is routed to opencode, which cannot be held to the judge's tools; route roles.%s.runner to claude, or remove it", loop.RoleDrainJudge)
	}
	if pinned := cfg.PinnedToHost(driveRoles); len(pinned) > 0 {
		return nil, p, driveRefusal("%s pinned to the host, which has nowhere to run with no host session; remove the pin", strings.Join(pinned, ", "))
	}
	if p, err = runner.DriveProfile(roots.Home); err != nil {
		return nil, p, driveRefusal("%s", err.Error())
	}
	p.Repo, p.Ledger = repoRoot, cfg.Ledger()
	lines, err := driveWalls.static(ctx, p)
	if err != nil {
		return nil, p, driveRefusal("%s", err.Error())
	}
	for _, l := range lines {
		fmt.Fprintln(cmd.ErrOrStderr(), "abcd drain --drive: "+termsafe.Sanitize(fsutil.RedactHome(l)))
	}
	return cfg, p, nil
}

// driveConsent shows what the drive will do and asks once; --yes is the
// scripted answer, and with no terminal nothing else answers.
func driveConsent(cmd *cobra.Command, w io.Writer, f driveFlags, lanes, runs int, p runner.Profile) (bool, error) {
	if !f.yes && !connectIsTerminal(cmd.InOrStdin()) {
		return false, driveRefusal("there is no terminal to ask at; `abcd drain --drive --yes` is the scripted yes")
	}
	fmt.Fprintln(w, "abcd drain --drive works through this repository's issues with no further input:")
	fmt.Fprintf(w, "  cap:      %d lane(s)\n", lanes)
	fmt.Fprintf(w, "  estimate: about %d agent run(s) at most\n", runs)
	if f.until.IsZero() {
		fmt.Fprintln(w, "  bound:    eight hours on any one wait (--until sets one of your own)")
	} else {
		fmt.Fprintf(w, "  bound:    ends at %s, and waits through limits and outages until then\n", f.until.UTC().Format(time.RFC3339))
	}
	fmt.Fprintf(w, "  runner: claude\n")
	fmt.Fprintf(w, "  route:    this account's subscription, on the separate login %s\n", abcdhome.Display("profiles", filepath.Base(p.Dir)))
	fmt.Fprintln(w, "  profile:  a separate login and settings on this one account; a stage's shell reads what this account can")
	fmt.Fprintln(w, "  first:    one short live launch checks the tools, the guard and the shell, on the same subscription")
	if f.yes {
		return true, nil
	}
	fmt.Fprint(w, "Start? [y/N] ")
	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	a := strings.ToLower(strings.TrimSpace(line))
	return a == "y" || a == "yes", nil
}

// runDrive is the verb. Before the yes a signal ends the process, with nothing
// started; from the yes on it is a stop (drain_drive_signal.go).
func runDrive(cmd *cobra.Command, asJSON bool, repoRoot string, f driveFlags) error {
	ctx, stopAsking := drainSignalContext()
	defer stopAsking()
	roots, notes := layered.RootsFor(repoRoot)
	for _, n := range notes {
		fmt.Fprintln(cmd.ErrOrStderr(), noteLine(n))
	}
	cfg, p, err := drivePreconditions(ctx, cmd, repoRoot, roots)
	if err != nil {
		return err
	}
	walled := cfg.Walled(p)
	o := loop.Options{Roots: &roots, Runners: walled, TakeOver: f.takeOver, Pace: f.pace, SubAgents: f.subAgents, FixRounds: f.fixRounds}
	progress, err := loop.ReadDrainProgress(repoRoot)
	if err != nil {
		return loopFail(cmd.OutOrStdout(), asJSON, "abcd drain", err)
	}
	if progress.Live && progress.Max > 0 && f.maxLanes > 0 && f.maxLanes != progress.Max {
		return driveRefusal("--max %d differs from the cap of %d the drain in progress began with; leave --max out", f.maxLanes, progress.Max)
	}
	lanes := driveDefaultCap
	if f.maxLanes > 0 {
		lanes = f.maxLanes
	}
	if progress.Live && progress.Max > 0 {
		lanes = max(progress.Max-progress.Lanes, 0)
	}
	runs, err := loop.EstimateRuns(o, repoRoot, lanes)
	if err != nil {
		return loopFail(cmd.OutOrStdout(), asJSON, "abcd drain", err)
	}
	asked := cmd.OutOrStdout()
	if asJSON {
		asked = cmd.ErrOrStderr()
	}
	yes, err := driveConsent(cmd, asked, f, lanes, runs, p)
	if err != nil {
		return err
	}
	if !yes {
		fmt.Fprintln(asked, "abcd drain --drive: nothing started, nothing reserved")
		return nil
	}
	if err := fsutil.EnsureRealDirAll(repoRoot, filepath.ToSlash(filepath.Dir(driveLockRel)), 0o700); err != nil {
		return driveRefusal("%s", err.Error())
	}
	// One drive per checkout, from its yes to its end: one stopping after a
	// hang-up must not release a claim under the next.
	err = fsutil.WithFileLock(filepath.Join(repoRoot, filepath.FromSlash(driveLockRel)), driveLockWait, func() error {
		return runDriveHeld(cmd, asJSON, repoRoot, roots, p, o, f, lanes, progress, stopAsking)
	})
	if errors.Is(err, fsutil.ErrLockContention) {
		return &exitError{Code: 3, Msg: "abcd drain --drive: another drive holds this checkout; nothing started, nothing reserved"}
	}
	return err
}

// runDriveHeld is the drive from its yes on, holding the checkout's drive lock.
func runDriveHeld(cmd *cobra.Command, asJSON bool, repoRoot string, roots layered.Roots, p runner.Profile, o loop.Options, f driveFlags, lanes int, progress loop.DrainProgress, stopAsking func()) error {
	stopAsking()
	errw := cmd.ErrOrStderr()
	sig := watchDriveSignals(func() {
		fmt.Fprintln(errw, "abcd drain --drive: stopping: the agent in a call finishes that call, then the lane's reservation is released; press again to kill it and exit at once")
	}, func() {
		fmt.Fprintln(errw, "abcd drain --drive: killing the running agent's process group and exiting")
	})
	defer sig.close()
	if err := driveWalls.live(sig.ctx, p); err != nil {
		if sig.ctx.Err() != nil {
			// The second signal killed the proof's launch, which says nothing
			// about the wall it was proving.
			return &exitError{Code: driveKilledExit, Msg: "abcd drain --drive: killed during the proof (nothing started, nothing reserved)"}
		}
		return driveRefusal("%s", err.Error())
	}
	d := loop.DrainOptions{Max: lanes, Transcripts: &lazyHistoryStore{cmd: cmd},
		Skip: f.skips.skip, SkipReason: f.skips.reason, Unskip: f.skips.unskip}
	if progress.Live {
		d.Max = f.maxLanes
		if progress.Max == 0 {
			if err := loop.CapDrainInProgress(repoRoot, lanes); err != nil {
				return loopFail(cmd.OutOrStdout(), asJSON, "abcd drain", err)
			}
			d.Max = 0
		}
	}
	return driveRun(sig, cmd, asJSON, repoRoot, roots, p, o, d, f.trustGates, f.until)
}

// driveRun runs the terminal loop and says how it ended.
func driveRun(sig *driveSignals, cmd *cobra.Command, asJSON bool, repoRoot string, roots layered.Roots, p runner.Profile, o loop.Options, d loop.DrainOptions, trustGates bool, until time.Time) error {
	review, err := loop.LoadReviewRoute(roots, agentPrompt)
	if err != nil {
		return loopFail(cmd.OutOrStdout(), asJSON, "abcd drain", err)
	}
	tiers, diags, err := loop.LoadTiering(roots, review)
	if err != nil {
		return loopFail(cmd.OutOrStdout(), asJSON, "abcd drain", err)
	}
	for _, line := range diags {
		fmt.Fprintln(cmd.ErrOrStderr(), termsafe.Sanitize(fsutil.RedactHome(line)))
	}
	o.Tiers = tiers
	lines := cmd.OutOrStdout()
	if asJSON {
		lines = cmd.ErrOrStderr()
	}
	say, detail, closeLog := driveLog(repoRoot, lines)
	defer closeLog()
	noticeUnattended(cmd, "abcd drain")
	res, err := driveTerminal(sig.ctx, repoRoot, loop.TerminalOptions{
		Options:          o,
		Drain:            d,
		Runners:          loop.Runners{Config: o.Runners, Transcripts: d.Transcripts},
		Review:           review,
		Preflight:        drivePreflight(detail, p),
		PreflightAllowed: driveGateCheck(trustGates),
		Say:              say,
		Stop:             sig.stop,
		Until:            until,
		Probe:            o.Runners.ProbeLimit,
		BeforeStage: func() error {
			return runner.CheckWallSettings(p, runner.SelfBinary())
		},
	})
	if err != nil {
		if res.Ended != "" {
			renderDriveEnd(cmd.ErrOrStderr(), res)
		}
		return loopFail(cmd.OutOrStdout(), asJSON, "abcd drain", err)
	}
	if rerr := render(cmd.OutOrStdout(), asJSON, res, func(w io.Writer) { renderDriveEnd(w, res) }); rerr != nil {
		return rerr
	}
	switch res.Ended {
	case loop.TerminalKilled:
		return &exitError{Code: driveKilledExit, Msg: "abcd drain --drive: killed"}
	case loop.TerminalWall:
		return &exitError{Code: 2, Msg: "abcd drain --drive: a wall no longer held"}
	case loop.TerminalOutage, loop.TerminalLimited, loop.TerminalWaited, loop.TerminalHalted, loop.DrainStoppedOutage, loop.DrainStoppedOrigin:
		return &exitError{Code: 3, Msg: "abcd drain --drive: stopped short: " + driveEndWords(res.Ended)}
	}
	return nil
}

// driveLog is where a drive's lines go: the terminal, one line per step, and a
// log under the local tier, stamped, which also takes the detail, a preflight's
// output. A log that cannot be opened is said and the lines still print.
func driveLog(repoRoot string, w io.Writer) (say func(string), detail io.Writer, closeLog func()) {
	const dir = ".abcd/.work.local/logs"
	var mu sync.Mutex
	var log *os.File
	if err := fsutil.EnsureRealDirAll(repoRoot, dir, 0o700); err == nil {
		name := "drain-drive-" + time.Now().UTC().Format("20060102T150405Z") + ".log"
		log, _ = os.OpenFile(filepath.Join(repoRoot, filepath.FromSlash(dir), name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	}
	if log == nil {
		fmt.Fprintln(w, "abcd drain --drive: the drive's log could not be opened under "+dir+"; its lines print here alone")
		detail = io.Discard
	} else {
		detail = log
	}
	return func(line string) {
			mu.Lock()
			defer mu.Unlock()
			fmt.Fprintln(w, line)
			if log != nil {
				fmt.Fprintln(log, time.Now().UTC().Format(time.RFC3339)+" "+line)
			}
		}, detail, func() {
			if log != nil {
				_ = log.Close()
			}
		}
}

// drivePreflight runs the repository's preflight in a lane's worktree behind
// the load gate, as a process group of its own, so the terminal's interrupt
// does not reach it, and with only what a stage carries of the environment,
// since the lane's own files decide what it runs.
func drivePreflight(w io.Writer, p runner.Profile) func(ctx context.Context, worktree string) error {
	return func(ctx context.Context, worktree string) error {
		program, args, err := drivePreflightProgram()
		if err != nil {
			return err
		}
		if err := runner.ResetStageHome(p); err != nil {
			return err
		}
		// #nosec G204 -- program is the running binary; the arguments are fixed.
		cmd := exec.CommandContext(ctx, program, args...)
		cmd.Dir = worktree
		cmd.Env = runner.PreflightEnv(p)
		cmd.Stdout, cmd.Stderr = w, w
		procgroup.Lead(cmd)
		cmd.Cancel = func() error { return procgroup.Kill(cmd) }
		return cmd.Run()
	}
}

// renderDriveEnd is how the drive ended and what it left.
func renderDriveEnd(w io.Writer, res loop.TerminalResult) {
	fmt.Fprintf(w, "abcd drain --drive: ended: %s\n", driveEndWords(res.Ended))
	c := res.Last.Counts
	fmt.Fprintf(w, "  lanes: %d opened, %d pull request(s), %d handed back; about %d agent run(s) spent\n",
		c.LanesOpened, c.PullRequests, c.LanesHandedBack, c.Spend.AgentRuns)
	if res.Run != "" {
		fmt.Fprintf(w, "  left on its branch: %s, run %s, %s\n", termsafe.Sanitize(res.Issue), termsafe.Sanitize(res.Run), termsafe.Sanitize(res.Lane))
		if res.Resume != "" {
			fmt.Fprintf(w, "  to resume: %s\n", res.Resume)
		}
		if res.Lapses != nil {
			fmt.Fprintf(w, "  its reservation lapses at %s\n", res.Lapses.UTC().Format(time.RFC3339))
		}
	}
}

// driveEndWords says why a drive ended.
func driveEndWords(ended string) string {
	why := map[string]string{
		loop.DrainStoppedCap:    "the cap is reached",
		loop.DrainStoppedEmpty:  "nothing eligible is left",
		loop.DrainStoppedOutage: "a run gave up on a lost connection",
		loop.DrainStoppedOrigin: "the drain gave up waiting on origin",
		loop.TerminalOutage:     "eight hours of runner failures",
		loop.TerminalStopped:    "stopped",
		loop.TerminalHalted:     "a stop set from elsewhere holds",
		loop.TerminalLimited:    "the account's rate limit outlasts what a drive waits",
		loop.TerminalWaited:     "nothing moved for eight hours",
		loop.TerminalUntil:      "the time --until named has passed",
		loop.TerminalWall:       "a wall no longer held",
		loop.TerminalRefused:    "a move was refused",
		loop.TerminalKilled:     "killed",
	}[ended]
	if why == "" {
		return ended
	}
	return why
}
