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
			lines = append(lines, "the walled directory's settings were edited, and were replaced with abcd's: a drive's agents hold abcd's hooks and nothing else")
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

// driveTerminal is loop.RunTerminal, a seam the tests replace.
var driveTerminal = loop.RunTerminal

// driveFlags are what `abcd drain --drive` was given.
type driveFlags struct {
	yes      bool
	maxLanes int
	takeOver bool
	skips    drainSkips
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
func driveConsent(cmd *cobra.Command, f driveFlags, lanes, runs int, p runner.Profile) (bool, error) {
	if !f.yes && !connectIsTerminal(cmd.InOrStdin()) {
		return false, driveRefusal("there is no terminal to ask at; `abcd drain --drive --yes` is the scripted yes")
	}
	w := cmd.OutOrStdout()
	fmt.Fprintln(w, "abcd drain --drive works through this repository's issues with no further input:")
	fmt.Fprintf(w, "  cap:      %d lane(s)\n", lanes)
	fmt.Fprintf(w, "  estimate: about %d agent run(s) at most\n", runs)
	fmt.Fprintf(w, "  runner: claude\n")
	fmt.Fprintf(w, "  route:    this account's subscription, on the walled login %s\n", abcdhome.Display("profiles", filepath.Base(p.Dir)))
	fmt.Fprintln(w, "  first:    one short live launch proves the walls, on the same subscription")
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
	o := loop.Options{Roots: &roots, Runners: walled, TakeOver: f.takeOver}
	progress, err := loop.ReadDrainProgress(repoRoot)
	if err != nil {
		return loopFail(cmd.OutOrStdout(), asJSON, "abcd drain", err)
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
	yes, err := driveConsent(cmd, f, lanes, runs, p)
	if err != nil {
		return err
	}
	if !yes {
		fmt.Fprintln(cmd.OutOrStdout(), "abcd drain --drive: nothing started, nothing reserved")
		return nil
	}
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
	return driveRun(sig, cmd, asJSON, repoRoot, roots, p, o, d)
}

// driveRun runs the terminal loop and says how it ended.
func driveRun(sig *driveSignals, cmd *cobra.Command, asJSON bool, repoRoot string, roots layered.Roots, p runner.Profile, o loop.Options, d loop.DrainOptions) error {
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
		Options:   o,
		Drain:     d,
		Runners:   loop.Runners{Config: o.Runners, Transcripts: d.Transcripts},
		Review:    review,
		Preflight: drivePreflight(detail),
		Say:       say,
		Stop:      sig.stop,
		BeforeStage: func() error {
			return runner.CheckWallSettings(p, runner.SelfBinary())
		},
	})
	if err != nil {
		return loopFail(cmd.OutOrStdout(), asJSON, "abcd drain", err)
	}
	if rerr := render(cmd.OutOrStdout(), asJSON, res, func(w io.Writer) { renderDriveEnd(w, res) }); rerr != nil {
		return rerr
	}
	if res.Ended == loop.TerminalKilled {
		return &exitError{Code: driveKilledExit, Msg: "abcd drain --drive: killed"}
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
// does not reach it.
func drivePreflight(w io.Writer) func(ctx context.Context, worktree string) error {
	return func(ctx context.Context, worktree string) error {
		self, err := os.Executable()
		if err != nil {
			return fmt.Errorf("abcd cannot name its own binary: %w", err)
		}
		// #nosec G204 -- self is the running binary; the arguments are fixed.
		cmd := exec.CommandContext(ctx, self, "implement", "load", "--wait", "--", "make", "preflight")
		cmd.Dir = worktree
		cmd.Env = gitutil.ScrubbedEnv()
		cmd.Stdout, cmd.Stderr = w, w
		procgroup.Lead(cmd)
		cmd.Cancel = func() error { return procgroup.Kill(cmd) }
		return cmd.Run()
	}
}

// renderDriveEnd is how the drive ended and what it left.
func renderDriveEnd(w io.Writer, res loop.TerminalResult) {
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
		loop.TerminalWall:       "a wall no longer held",
		loop.TerminalRefused:    "a move was refused",
		loop.TerminalKilled:     "killed",
	}[res.Ended]
	if why == "" {
		why = res.Ended
	}
	fmt.Fprintf(w, "abcd drain --drive: ended: %s\n", why)
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
