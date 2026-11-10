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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/core/update"
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
	// settings checks the settings abcd wrote still stand.
	settings func(p runner.Profile) error
	// binaries holds the binaries to what this process pinned, the harness aside.
	binaries func(p runner.Profile) error
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
			lines = append(lines, "the profile directory's settings were edited, and were replaced with abcd's: a drive's agents hold abcd's settings and nothing else")
		}
		moved, err := runner.SetAsideSyncedSkills(p)
		if moved != "" {
			lines = append(lines, "the harness had synced the account's skills into the profile directory, where every launch would load them; abcd moved them to "+
				moved+" and closed the skills folder so the harness cannot sync them back")
		}
		if err != nil {
			return lines, err
		}
		if note := runner.StageCachesNote(p); note != "" {
			lines = append(lines, note)
		}
		if err := runner.CheckWallSettings(p, bin); err != nil {
			return lines, err
		}
		return lines, runner.CheckLogin(ctx, p)
	},
	live: func(ctx context.Context, p runner.Profile) error {
		return runner.ProveWall(ctx, p, runner.SelfBinary())
	},
	settings: func(p runner.Profile) error { return runner.CheckWallSettings(p, runner.SelfBinary()) },
	binaries: runner.CheckPinnedBinaries,
}

// driveLockRel is the lock one drive holds on a checkout from its yes to its end,
// and driveLockWait how long a second drive tries for it.
const (
	driveLockRel  = ".abcd/.work.local/run/drive.lock"
	driveLockWait = 2 * time.Second
)

// driveTerminal is loop.RunTerminal, a seam the tests replace.
var driveTerminal = loop.RunTerminal

// driveLanesElsewhere is loop.LanesOnAnotherProfile, a seam the tests replace.
var driveLanesElsewhere = loop.LanesOnAnotherProfile

// driveProofWait are the clock and the sleep a wait at the live proof runs on, nil
// for the real ones: seams the tests replace.
var driveProofWait struct {
	now   func() time.Time
	sleep func(ctx context.Context, d time.Duration)
}

// driveSelfName names the running binary, a variable so a test can say what it reads.
var driveSelfName = runner.SelfBinary

// drivePreflightProgram is what a lane's preflight starts; a variable so a test
// can read the environment it is given.
var drivePreflightProgram = func() (string, []string, error) {
	self := driveSelfName()
	if self == "" {
		return "", nil, errors.New("abcd cannot name its own binary")
	}
	return self, []string{"implement", "load", "--wait", "--", "make", "preflight"}, nil
}

// driveGates are what a lane's preflight and the driver's own git run, as paths: a
// change to one is a change to what judges the lane.
var driveGates = []string{"Makefile", "GNUmakefile", "makefile", "scripts/", ".githooks/", "hooks/"}

// driveGate reports whether path, relative to the checkout, is one of those, or
// under the hooks path the repository configures. A directory counts with or
// without its trailing slash.
func driveGate(path, hooksPath string) bool {
	path = filepath.ToSlash(filepath.Clean(path))
	for _, gate := range driveGates {
		dir := strings.TrimSuffix(gate, "/")
		if path == dir || (dir != gate && strings.HasPrefix(path, gate)) {
			return true
		}
	}
	if hooksPath = strings.Trim(hooksPath, "/"); hooksPath != "" {
		return path == hooksPath || strings.HasPrefix(path, hooksPath+"/")
	}
	return false
}

// driveHooksPath is the hooks path the repository at dir configures, relative to
// dir, "" for none or one outside it.
func driveHooksPath(dir string) string {
	paths, err := gitutil.HooksPaths(dir)
	if err != nil || len(paths) == 0 {
		return ""
	}
	path := paths[len(paths)-1]
	if filepath.IsAbs(path) {
		rel, err := filepath.Rel(dir, path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, "../") {
			return ""
		}
		path = rel
	}
	return filepath.ToSlash(filepath.Clean(path))
}

// driveGateCheck says whether a lane's landing may go on: not over a lane that
// changes what judges it, which a person's own session handles, and not when what
// the lane changed cannot be read.
func driveGateCheck() func(worktree, base string) error {
	return func(worktree, base string) error {
		out, err := gitutil.Run(worktree, "diff", "--name-only", "-z", "--no-renames", base, "HEAD", "--")
		if err != nil {
			return &loop.PreflightRefused{Reason: "what the lane changed could not be read, so its landing does not go on"}
		}
		hooks := driveHooksPath(worktree)
		for _, path := range strings.Split(out, "\x00") {
			if driveGate(path, hooks) {
				return &loop.PreflightRefused{Reason: "the lane changes " + strconv.Quote(termsafe.Sanitize(path)) + ", a gate file, so it needs a person's own session and its landing does not go on: review the branch there"}
			}
		}
		return nil
	}
}

// drivePassOver says why an issue is passed over rather than opened: its own path
// is one of the gates, so a lane on it could only be handed back.
func drivePassOver(repoRoot string) func(id, foundAt string) string {
	hooks := driveHooksPath(repoRoot)
	return func(_, foundAt string) string {
		path, ok := capture.FoundAtPath(foundAt)
		if !ok {
			path, _, _ = strings.Cut(foundAt, ":")
		}
		if !driveGate(path, hooks) {
			return ""
		}
		return "it needs a person's own session, since it changes a gate file: " + strconv.Quote(termsafe.Sanitize(path))
	}
}

// driveFlags are what `abcd drain --drive` was given.
type driveFlags struct {
	yes      bool
	maxLanes int
	takeOver bool
	// profile is the --profile name, "" for none given.
	profile string
	skips   drainSkips
	// pace, subAgents and fixRounds are the flags as given, nil for one not.
	pace, subAgents, fixRounds *string
	// until is the run's bound; zero leaves each wait its eight hours.
	until time.Time
	// noSelfUpdate is --no-self-update.
	noSelfUpdate bool
	// afterUpdate is the hidden --after-update <from>=<to> a drive's own re-execution
	// passes, "" for none; refused the hidden --update-refused tags.
	afterUpdate string
	refused     []string
	// self is the running file, read once at the start.
	self update.SelfTarget
	// reached is set once the loop has made a move.
	reached *bool
	// said is set once a resuming drive has said what it resumes from; a failed start
	// tried again does not say it twice.
	said *bool
	// stopped is set once a stop was asked for during a resuming start, which outlives a
	// try that fails.
	stopped *bool
	// began is what a resuming process began on beside the installed file, which each
	// attempt after a failed one is held to before it pins anything.
	began runner.StartPins
	// startedOn is the installed file's digest as this resuming process began, and retried
	// says this is an attempt after a failed one.
	startedOn string
	retried   bool
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
func drivePreconditions(ctx context.Context, cmd *cobra.Command, repoRoot string, roots layered.Roots, named string) (*runner.Config, runner.Profile, error) {
	var p runner.Profile
	for _, k := range driveSessionMarkers {
		if os.Getenv(k) != "" {
			return nil, p, driveRefusal("%s is set, so this is an agent's own session; a drive starts only from a command the person types in a plain terminal", k)
		}
	}
	if held := runner.HeldCredentials(); len(held) > 0 {
		return nil, p, driveRefusal("%s is set in the environment, and a drive runs on a subscription alone until %s covers how a key reaches the harness and what it may spend; unset it",
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
	if p, err = runner.DriveProfile(roots.Home, named); err != nil {
		return nil, p, driveRefusal("%s", err.Error())
	}
	p = p.ForCheckout(repoRoot)
	p.Ledger = cfg.Ledger()
	// Every reviewer the review set names must be one a route here can launch,
	// asked of the walled configuration the drive's lanes launch on, never the
	// unwalled one read above; one that cannot would wait at review for ever
	// (iss-2611080057001367).
	walled, err := driveWalled(cfg, p, roots)
	if err != nil {
		return nil, p, driveRefusal("%s", err.Error())
	}
	if err := loop.CheckReviewersLaunch(loop.Options{Roots: &roots, Runners: walled}, repoRoot); err != nil {
		return nil, p, driveRefusal("%s", err.Error())
	}
	return cfg, p, nil
}

// driveWalled is cfg for this drive: every role on the claude harness on p
// (runner.Config.Walled), holding the routes the machine's class map resolves
// as the only ones the launch takes by name (iss-2611080057001367).
func driveWalled(cfg *runner.Config, p runner.Profile, roots layered.Roots) (*runner.Config, error) {
	routes, err := loop.ClassRoutes(roots)
	if err != nil {
		return nil, err
	}
	return cfg.Walled(p).WithClassRoutes(routes), nil
}

// driveHoldsToProfile refuses a drive that would go on with a lane on another profile
// than the one it started on, which is another subscription: before the question, so
// nothing is spent or reserved. The loop holds to the same rule for a lane that comes
// back later (drive_profile.go). A state that cannot be read is the first move's to
// refuse, in its own words.
func driveHoldsToProfile(repoRoot, profile string) error {
	lanes, err := driveLanesElsewhere(repoRoot, profile)
	if err != nil || len(lanes) == 0 {
		return nil
	}
	return driveRefusal("%s", loop.OnAnotherProfile(lanes[0], profile))
}

// driveBeforeStage is the check before each stage: the walls, and where the file the run
// proved is gone, a new file held to the run's signing team and the one proof of it. A limit,
// a launch with no verdict, or a file that has gone for now waits as the first proof's does,
// renewing what the run holds, with the walls beside the harness checked at each turn; that
// wait's own end is what the run ends on, and a wall that fails in it is the reason given. A
// file that goes after this check is the launch's own full check to refuse, which starts
// nothing, and the next turn finds the proof owed.
func driveBeforeStage(sig *driveSignals, repoRoot string, p runner.Profile, o loop.Options, until time.Time, say func(string)) func() error {
	return func() error {
		bin := runner.SelfBinary()
		if runner.HarnessOwed(p) {
			line, err := runner.ReproveHarness(sig.ctx, p, bin)
			var limit *runner.LimitError
			var none *runner.NoVerdictError
			if sig.ctx.Err() == nil && (errors.As(err, &limit) || errors.As(err, &none)) {
				o.Now = driveProofWait.now
				var wall error
				ended, werr := loop.WaitOutProof(sig.ctx, repoRoot, loop.TerminalOptions{Options: o, Until: until, Stop: sig.stop, Say: say, Sleep: driveProofWait.sleep,
					BeforeStage: func() error {
						if wall = runner.CheckWallSettingsBesideHarness(p, bin); wall != nil {
							return &loop.WallEnded{Ended: loop.TerminalWall}
						}
						return nil
					}},
					err, func(ctx context.Context) error {
						var perr error
						line, perr = runner.ReproveHarness(ctx, p, bin)
						return perr
					})
				if wall != nil {
					return wall
				}
				if ended != "" {
					return &loop.WallEnded{Ended: ended}
				}
				err = werr
			}
			if sig.ctx.Err() != nil {
				return &loop.WallEnded{Ended: loop.TerminalKilled}
			}
			if err != nil {
				return err
			}
			if line != "" {
				say(line)
			}
		}
		return runner.CheckWallSettingsBesideHarness(p, bin)
	}
}

// driveConsent shows what the drive will do and asks once; --yes is the
// scripted answer, and with no terminal nothing else answers.
func driveConsent(cmd *cobra.Command, w io.Writer, f driveFlags, lanes, runs int, p runner.Profile) (bool, error) {
	if !f.yes && !connectIsTerminal(cmd.InOrStdin()) {
		return false, driveRefusal("there is no terminal to ask at; `abcd drain --drive --yes` is the scripted yes")
	}
	fmt.Fprintln(w, "abcd drain --drive works through this repository's issues with no further input:")
	fmt.Fprintf(w, "  cap:      %d lane(s)\n", lanes)
	if f.until.IsZero() {
		fmt.Fprintf(w, "  estimate: about %d agent run(s) at most\n", runs)
		fmt.Fprintln(w, "  bound:    eight hours on any one wait (--until sets one of your own)")
	} else {
		fmt.Fprintf(w, "  estimate: about %d agent run(s) if nothing fails; retries through failures and short probes of a limit add to it\n", runs)
		fmt.Fprintf(w, "  bound:    ends at %s, and waits through limits and outages until then; a time, not a limit on what it spends\n", f.until.UTC().Format(time.RFC3339))
	}
	fmt.Fprintf(w, "  runner: claude\n")
	name := termsafe.Sanitize(p.Name())
	fmt.Fprintf(w, "  route:    the subscription logged in there, on the separate login %s\n", abcdhome.Display("profiles", name))
	fmt.Fprintf(w, "  profile:  %s, a separate login and settings on this one account; a stage's shell reads what this account can\n", name)
	fmt.Fprintln(w, "  gates:    a change to the Makefile, scripts/ or the hooks is not a drive's: it needs a person's own session")
	fmt.Fprintln(w, "  first:    one short live launch checks the tools, the guard and the shell, on the same subscription")
	fmt.Fprintln(w, "  updates:  a harness update during the run carries on only where the new file has the same signing team and passes one more such launch; elsewhere the run stops")
	fmt.Fprintln(w, "  releases: "+fsutil.RedactHome(driveReleaseLine(f)))
	if !f.noSelfUpdate && f.self.Refusal == nil {
		fmt.Fprintln(w, "  others:   any other drive on this account ends when this one moves to a new release, whatever flags it has")
	}
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

// errAnotherDrive is what a drive that lost the checkout's lock ends on. Holding it is
// no fault of a release, so a re-executed start does not go back over it.
var errAnotherDrive = &exitError{Code: 3, Msg: "abcd drain --drive: another drive holds this checkout; nothing started, nothing reserved"}

// runDrive is the verb, and what a re-executed drive does when it cannot go on: a failed
// start is tried again on a short schedule, since a passing failure is no verdict on the
// release; one that repeats before its first move goes back to the kept copy, and past
// that move names the copy.
func runDrive(cmd *cobra.Command, asJSON bool, repoRoot string, f driveFlags) error {
	reached, said, stopped := false, false, false
	f.reached, f.said, f.stopped = &reached, &said, &stopped
	if f.resuming() {
		f.startedOn = driveFileDigest(f.self.Path)
		runner.PinPrograms(drivePrograms...)
		roots, _ := layered.RootsFor(repoRoot)
		f.began = runner.PinStart(roots.Home)
	}
	err := startGuarded(cmd, asJSON, repoRoot, f)
	if f.resuming() {
		for i, wait := range driveStartRetries {
			if err == nil || reached || stopped || !driveStartFailed(err) {
				break
			}
			if killed := driveWaitToStartAgain(cmd.ErrOrStderr(), repoRoot, f, err, wait, i); killed != nil {
				return killed
			}
			f.retried = true
			err = startGuarded(cmd, asJSON, repoRoot, f)
		}
	}
	if err != nil && stopped && !reached && driveStartFailed(err) {
		return driveStoppedStarting()
	}
	if err == nil || f.afterUpdate == "" {
		return driveSayRefused(err)
	}
	switch failed := driveStartFailed(err); {
	case driveEndedOnWall(err), failed && reached:
		driveKeptNote(cmd.ErrOrStderr(), f)
		return err
	case !failed:
		return err
	}
	back := driveGoBack(cmd.ErrOrStderr(), repoRoot, f, err)
	if back == nil {
		return nil
	}
	// The kept copy did not take over, so the answer for this attempt is still owed.
	answer := driveSayRefused(err)
	if back == err {
		return answer
	}
	return back
}

// driveHeldRefusal is a failure on a resuming start before its first move: another attempt
// or the way back may follow, so what it says to its reader waits until none does.
type driveHeldRefusal struct {
	end error
	say func() error
}

func (e *driveHeldRefusal) Error() string { return e.end.Error() }

// driveFail says how the loop's err ended the run, unless a further attempt may yet answer
// for it; res is the run's result, when it got as far as having one.
func driveFail(cmd *cobra.Command, asJSON bool, f driveFlags, res *loop.TerminalResult, err error) error {
	say := func() error {
		if res != nil && res.Ended != "" {
			renderDriveEnd(cmd.ErrOrStderr(), *res)
		}
		return loopFail(cmd.OutOrStdout(), asJSON, "abcd drain", err)
	}
	if f.resuming() && !*f.reached {
		if end := loopFail(io.Discard, false, "abcd drain", err); driveStartFailed(end) {
			return &driveHeldRefusal{end: end, say: say}
		}
	}
	return say()
}

// driveSayRefused says what a held refusal held, once, when no attempt comes after it.
func driveSayRefused(err error) error {
	var held *driveHeldRefusal
	if errors.As(err, &held) {
		return held.say()
	}
	return err
}

// startGuarded is startDrive, with a panic before a resuming drive's first move read as
// a failed start, so it reaches the same tries and the same verified way back. Past that
// move, and on a start a person typed, a panic stays a panic.
func startGuarded(cmd *cobra.Command, asJSON bool, repoRoot string, f driveFlags) (err error) {
	defer func() {
		if !f.resuming() || *f.reached {
			return
		}
		if r := recover(); r != nil {
			err = fmt.Errorf("it panicked: %v", r)
		}
	}()
	return startDrive(cmd, asJSON, repoRoot, f)
}

// startDrive takes the checkout and starts the run. Before the yes a signal ends the
// process, with nothing started; from the yes on it is a stop (drain_drive_signal.go).
func startDrive(cmd *cobra.Command, asJSON bool, repoRoot string, f driveFlags) error {
	ctx, stopAsking := drainSignalContext()
	defer stopAsking()
	roots, notes := layered.RootsFor(repoRoot)
	for _, n := range notes {
		fmt.Fprintln(cmd.ErrOrStderr(), noteLine(n))
	}
	cfg, p, err := drivePreconditions(ctx, cmd, repoRoot, roots, f.profile)
	if err != nil {
		return err
	}
	if err := fsutil.EnsureRealDirAll(repoRoot, filepath.ToSlash(filepath.Dir(driveLockRel)), 0o700); err != nil {
		return driveRefusal("%s", err.Error())
	}
	// One drive per checkout, from before it writes anything to its end: one
	// stopping after a hang-up must not release a claim under the next.
	err = fsutil.WithFileLock(filepath.Join(repoRoot, filepath.FromSlash(driveLockRel)), driveLockWait, func() error {
		return runDriveLocked(ctx, cmd, asJSON, repoRoot, roots, cfg, p, f, stopAsking)
	})
	if errors.Is(err, fsutil.ErrLockContention) {
		return errAnotherDrive
	}
	return err
}

// drivePrograms are the program names a drive fixes the meaning of.
var drivePrograms = []string{"git", "make", "gh", "claude"}

// driveAtYes is called in the window before the drive's own signal handling starts, a seam
// the tests stand a signal in with.
var driveAtYes = func(context.Context) {}

// runDriveLocked is the drive holding the checkout's drive lock, up to its yes.
func runDriveLocked(starting context.Context, cmd *cobra.Command, asJSON bool, repoRoot string, roots layered.Roots, cfg *runner.Config, p runner.Profile, f driveFlags, stopAsking func()) error {
	if f.resuming() {
		p.Began = &f.began
	}
	walled, err := driveWalled(cfg, p, roots)
	if err != nil {
		return driveFail(cmd, asJSON, f, nil, err)
	}
	o := loop.Options{Roots: &roots, Runners: walled, TakeOver: f.takeOver, Pace: f.pace, SubAgents: f.subAgents, FixRounds: f.fixRounds, Profile: p.Name()}
	if err := driveHoldsToProfile(repoRoot, p.Name()); err != nil {
		return err
	}
	progress, err := loop.ReadDrainProgress(repoRoot)
	if err != nil {
		return driveFail(cmd, asJSON, f, nil, err)
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
	// A re-executed drive asks nothing: the yes was given once, and covered this.
	if !f.resuming() {
		runs, err := loop.EstimateRuns(o, repoRoot, lanes)
		if err != nil {
			return driveFail(cmd, asJSON, f, nil, err)
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
	}
	driveAtYes(starting)
	// Ending the start's own handling ends its context too, so what it heard is read first.
	heard := starting.Err() != nil
	stopAsking()
	// What a program name means is fixed now, hooks the driver's git runs get only
	// what a stage carries, and only the push keeps the person's credentials.
	runner.PinPrograms(drivePrograms...)
	gitutil.SetHookedEnv(runner.HookedEnv(p))
	defer gitutil.SetHookedEnv(nil)
	// The signal handling says what each signal did while the drive writes.
	errw := sharedStderr(cmd.ErrOrStderr())
	cmd.SetErr(errw)
	sig := watchDriveSignals(func() {
		fmt.Fprintln(errw, "abcd drain --drive: stopping: the agent in a call finishes that call, then the lane's reservation is released; press again to kill it and exit at once")
	}, func() {
		fmt.Fprintln(errw, "abcd drain --drive: killing the running agent's process group and exiting")
	})
	defer sig.close()
	// A stop outlives a try that fails, and the window before this handling began counts.
	if f.resuming() {
		defer func() {
			if sig.stopAsked() {
				*f.stopped = true
			}
		}()
		if heard {
			*f.stopped = true
			return driveStoppedStarting()
		}
	}
	if f.retried {
		if err := driveStillOnItsBinaries(p, f); err != nil {
			return err
		}
	}
	// Only a yes writes anything: another drive on this login reads these files.
	lines, err := driveWalls.static(sig.ctx, p)
	for _, l := range lines {
		fmt.Fprintln(errw, "abcd drain --drive: "+termsafe.Sanitize(fsutil.RedactHome(l)))
	}
	if err != nil {
		if sig.ctx.Err() != nil {
			return &exitError{Code: driveKilledExit, Msg: "abcd drain --drive: killed during the proof (nothing started, nothing reserved)"}
		}
		var team *runner.HarnessTeamError
		if errors.As(err, &team) {
			return &exitError{Code: 2, wall: true, Msg: driveLine + termsafe.Sanitize(fsutil.RedactHome(err.Error()))}
		}
		return driveRefusal("%s", err.Error())
	}
	// A release moved onto passed the live proof moments ago, from the file now installed.
	// A kept copy going on after a refused start did not, and proves itself: it is rare and
	// costs one launch. Nor did an attempt after a wait, whose harness may have changed in it.
	if f.afterUpdate == "" || f.retried {
		// The proof is an agent, and none starts after a stop.
		if f.resuming() && sig.stopAsked() {
			return driveStoppedStarting()
		}
		ended, err := driveProve(sig, cmd, asJSON, repoRoot, p, o, f)
		if err != nil {
			return err
		}
		if ended {
			return nil
		}
	}
	d := loop.DrainOptions{Max: lanes, Transcripts: &lazyHistoryStore{cmd: cmd}, PassOver: drivePassOver(repoRoot),
		Skip: f.skips.skip, SkipReason: f.skips.reason, Unskip: f.skips.unskip}
	if progress.Live {
		d.Max = f.maxLanes
		if progress.Max == 0 {
			if err := loop.CapDrainInProgress(repoRoot, lanes); err != nil {
				return driveFail(cmd, asJSON, f, nil, err)
			}
			d.Max = 0
		}
	}
	return driveRun(sig, cmd, asJSON, repoRoot, roots, p, o, d, f, lines)
}

// driveProve runs the live proof. A limit it meets is no failed check, and nor is a
// launch that came to no verdict on any: with no bound a start someone typed refuses,
// naming which, and under one it waits on the schedule any limit or outage does and proves
// again. A resuming start has nobody at the terminal to run it again, so a limit waits
// there with no bound too. Only a verdict, or the harness's absence, is refused then. It
// reports true when the wait ended the drive cleanly, before anything started.
func driveProve(sig *driveSignals, cmd *cobra.Command, asJSON bool, repoRoot string, p runner.Profile, o loop.Options, f driveFlags) (bool, error) {
	killed := &exitError{Code: driveKilledExit, Msg: "abcd drain --drive: killed during the proof (nothing started, nothing reserved)"}
	err := driveWalls.live(sig.ctx, p)
	var limit *runner.LimitError
	var none *runner.NoVerdictError
	isLimit := errors.As(err, &limit)
	if isLimit && f.until.IsZero() && !f.resuming() && sig.ctx.Err() == nil {
		resets := ""
		if !limit.ResetsAt.IsZero() {
			resets = ", resetting at " + limit.ResetsAt.UTC().Format(time.RFC3339)
		}
		return false, &exitError{Code: 3, Msg: "abcd drain --drive: profile " + termsafe.Sanitize(p.Name()) + " has met its rate limit" + resets +
			", so the live proof could not run; run this again then, or give --until <date> to wait through it (nothing started, nothing reserved)"}
	}
	if ((isLimit && f.resuming()) || ((isLimit || errors.As(err, &none)) && !f.until.IsZero())) && sig.ctx.Err() == nil {
		lines := cmd.OutOrStdout()
		if asJSON {
			lines = cmd.ErrOrStderr()
		}
		say, _, closeLog := driveLog(repoRoot, lines)
		defer closeLog()
		o.Now = driveProofWait.now
		var ended string
		ended, err = loop.WaitOutProof(sig.ctx, repoRoot, loop.TerminalOptions{Options: o, Until: f.until, Stop: sig.stop, Say: say,
			Sleep: driveProofWait.sleep, Probe: o.Runners.ProbeLimit, BeforeStage: func() error { return driveWalls.settings(p) }},
			err, func(ctx context.Context) error { return driveWalls.live(ctx, p) })
		switch ended {
		case "":
		case loop.TerminalKilled:
			return false, killed
		case loop.TerminalWall:
			return false, &exitError{Code: 2, Msg: "abcd drain --drive: a check no longer held (nothing started, nothing reserved)", wall: true}
		default:
			if f.resuming() && ended == loop.TerminalStopped {
				return false, driveStoppedStarting()
			}
			if f.resuming() && ended != loop.TerminalUntil {
				return false, &exitError{Code: 3, Msg: "abcd drain --drive: stopped short: " + driveEndWords(ended) + " (nothing started on this binary)"}
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "abcd drain --drive: ended: %s (nothing started, nothing reserved)\n", driveEndWords(ended))
			return true, nil
		}
	}
	if err != nil {
		if sig.ctx.Err() != nil {
			// The second signal killed the proof's launch, which says nothing
			// about the wall it was proving.
			return false, killed
		}
		return false, driveRefusal("%s", err.Error())
	}
	return false, nil
}

// driveRun runs the terminal loop and says how it ended.
func driveRun(sig *driveSignals, cmd *cobra.Command, asJSON bool, repoRoot string, roots layered.Roots, p runner.Profile, o loop.Options, d loop.DrainOptions, f driveFlags, startLines []string) error {
	until := f.until
	review, err := loop.LoadReviewRoute(roots, agentPrompt)
	if err != nil {
		return driveFail(cmd, asJSON, f, nil, err)
	}
	tiers, diags, err := loop.LoadTiering(roots, review)
	if err != nil {
		return driveFail(cmd, asJSON, f, nil, err)
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
	// What the start said came before the log opened; someone reading it later needs it.
	for _, l := range startLines {
		fmt.Fprintln(detail, time.Now().UTC().Format(time.RFC3339)+" abcd drain --drive: "+termsafe.Sanitize(fsutil.RedactHome(l)))
	}
	noticeUnattended(cmd, "abcd drain")
	switch {
	case *f.said:
	case f.afterUpdate != "":
		from, to, _ := parseAfterUpdate(f.afterUpdate)
		say(driveLine + update.UpdatedLine(from, to))
	case f.resuming():
		say(driveLine + f.refused[len(f.refused)-1] + " could not start, so the drive is going on with " + f.self.Version)
	}
	*f.said = true
	updates := newDriveUpdate(repoRoot, p, f, sig.stop, say, detail)
	// A yes with no bound never mentioned a probe.
	var probe func(ctx context.Context, role, model string) (bool, *runner.Failure)
	if !until.IsZero() {
		probe = o.Runners.ProbeLimit
	}
	terminal := loop.TerminalOptions{
		Options:          o,
		Drain:            d,
		Runners:          loop.Runners{Config: o.Runners, Transcripts: d.Transcripts},
		Review:           review,
		Preflight:        drivePreflight(detail, p),
		PreflightAllowed: driveGateCheck(),
		Say:              say,
		Stop:             sig.stop,
		Until:            until,
		Probe:            probe,
		BeforeStage:      driveBeforeStage(sig, repoRoot, p, o, until, say),
		Resumed:          f.resuming(),
		Moved:            func() { *f.reached = true },
	}
	// A run that takes no release has no check to hold, and nothing to say waits for one.
	if !f.noSelfUpdate && f.self.Refusal == nil {
		terminal.BoundaryCeiling, terminal.AtBoundary = driveCheckEvery, updates.atBoundary
	}
	res, err := driveTerminal(sig.ctx, repoRoot, terminal)
	// What was typed to choose the profile is what going on takes.
	if res.Resume != "" && f.profile != "" {
		res.Resume += " --profile " + f.profile
	}
	if err != nil {
		return driveFail(cmd, asJSON, f, &res, err)
	}
	if rerr := render(cmd.OutOrStdout(), asJSON, res, func(w io.Writer) { renderDriveEnd(w, res) }); rerr != nil {
		return rerr
	}
	if len(res.HeldBack) > 0 && res.Ended != loop.TerminalKilled && res.Ended != loop.TerminalWall {
		return driveHeldBackExit(res.HeldBack)
	}
	switch res.Ended {
	case loop.TerminalKilled:
		return &exitError{Code: driveKilledExit, Msg: "abcd drain --drive: killed"}
	case loop.TerminalWall:
		return &exitError{Code: 2, Msg: "abcd drain --drive: " + driveEndWords(res.Ended) + driveWhy(res), wall: true}
	case loop.TerminalStopped, loop.TerminalOutage, loop.TerminalLimited, loop.TerminalWaited, loop.TerminalHalted, loop.DrainStoppedOutage, loop.DrainStoppedOrigin:
		return &exitError{Code: 3, Msg: "abcd drain --drive: stopped short: " + driveEndWords(res.Ended) + driveWhy(res)}
	}
	return nil
}

// driveHeldBackExit is the exit of a run that held issues back.
func driveHeldBackExit(held []string) error {
	return &exitError{Code: 3, Msg: "abcd drain --drive: stopped short: it held back " + termsafe.Sanitize(strings.Join(held, ", "))}
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
		err = cmd.Run()
		procgroup.KillLeft(cmd)
		return err
	}
}

// driveWhy is what stopped a drive, as a clause to follow its end words.
func driveWhy(res loop.TerminalResult) string {
	if res.Why == "" {
		return ""
	}
	return ": " + termsafe.Sanitize(fsutil.RedactHome(res.Why))
}

// renderDriveEnd is how the drive ended and what it left.
func renderDriveEnd(w io.Writer, res loop.TerminalResult) {
	fmt.Fprintf(w, "abcd drain --drive: ended: %s%s\n", driveEndWords(res.Ended), driveWhy(res))
	c := res.Last.Counts
	fmt.Fprintf(w, "  lanes: %d opened, %d pull request(s), %d handed back; about %d agent run(s) spent\n",
		c.LanesOpened, c.PullRequests, c.LanesHandedBack, c.Spend.AgentRuns)
	for _, id := range res.HeldBack {
		fmt.Fprintf(w, "  held back: %s: its judge ran and did not answer (kept in the main checkout's %s)\n", termsafe.Sanitize(id), loop.HeldBackRel)
	}
	for _, p := range res.PassedOver {
		fmt.Fprintf(w, "  passed over: %s: %s\n", termsafe.Sanitize(p.ID), termsafe.Sanitize(p.Reason))
	}
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
		loop.TerminalHalted:     "a stop holds the run",
		loop.TerminalLimited:    "the profile's rate limit outlasts what a drive waits",
		loop.TerminalWaited:     "nothing moved for eight hours",
		loop.TerminalUntil:      "the time --until named has passed",
		loop.TerminalWall:       "a check before a stage no longer holds",
		loop.TerminalRefused:    "a move was refused",
		loop.TerminalKilled:     "killed",
	}[ended]
	if why == "" {
		return ended
	}
	return why
}
