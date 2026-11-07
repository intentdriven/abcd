package cli

// drain_drive_update.go is how a running drive takes a new release of abcd
// (itd-2611062149308723, spc-2611062151392235; adr-2611062149324491, proposed):
// at a lane boundary, at most about once an hour, on the yes the consent screen
// named it in. The decision reads the flag and the clock, in memory; the running
// binary's build metadata and its kind, read once at the start; and the release
// origin. No file on disk brings a binary in: even the
// release that could not start is carried in the arguments of the process that
// comes back from it. A failed rehearsal keeps a version out and never brings
// one in.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/core/update"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/procgroup"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// driveCheckEvery is how often a drive asks the release origin, at most.
const driveCheckEvery = time.Hour

// driveAfterUpdateFlag and driveUpdateRefusedFlag are what a drive passes when it
// re-executes itself. Neither is for a person: each is hidden, and refused with no
// drain in progress.
const (
	driveAfterUpdateFlag   = "--after-update"
	driveUpdateRefusedFlag = "--update-refused"
)

// driveLine is the opening of every line a drive says.
const driveLine = "abcd drain --drive: "

// driveUpdateSteps are the step's seams, as driveWallChecks are the walls'.
type driveUpdateSteps struct {
	// target judges the running file; a drive reads it once, at its start.
	target func() update.SelfTarget
	// candidate asks the origin for its latest release: one request.
	candidate func(current string) (tag string, newer bool, err error)
	// stage writes tag's verified bytes to the staging name beside target.
	stage func(target, tag string) (update.Staged, error)
	// rehearse runs the staged binary read-only against this checkout; args are
	// what its re-execution would be started with.
	rehearse func(ctx context.Context, c driveCandidate, args []string) error
	// prove is the live launch with the staged binary as its approver and hooks.
	prove func(ctx context.Context, p runner.Profile, staged string) error
	// harnessOwed reports whether the run owes a proof of the harness it would start on.
	harnessOwed func(p runner.Profile) bool
	// harnessMoved reports whether a start would resolve the harness to another file.
	harnessMoved func(p runner.Profile) bool
	// commit swaps the staged file in, keeping the previous one.
	commit func(target string, st update.Staged, keepAs string) error
	// verifyKept reads the kept copy and holds it to the release it was; what it
	// returns are the bytes it verified.
	verifyKept func(keepAs, tag string) ([]byte, error)
	// pinned is the digest the run holds for the binary at path, read at its start.
	pinned func(p runner.Profile, path string) ([sha256.Size]byte, bool)
}

// driveCandidate is a staged binary as a rehearsal runs it: where, the digest it was
// verified at, and the environment it runs in.
type driveCandidate struct {
	repoRoot, path, sha256 string
	env                    []string
}

// driveCheckBeforeExec holds a file to the digest it was verified at, and to being a
// regular file, immediately before it is run.
var driveCheckBeforeExec = update.CheckStaged

// driveSelfUpdateSteps are the steps as they ship. The release client is built only
// when a step runs, so a drive that never reaches one constructs nothing that can
// reach the network.
var driveSelfUpdateSteps = driveUpdateSteps{
	target: update.ResolveSelfTarget,
	candidate: func(current string) (string, bool, error) {
		return update.NewSelfUpdater().Candidate(current)
	},
	stage: func(target, tag string) (update.Staged, error) {
		return update.NewSelfUpdater().StageRelease(target, tag, nil)
	},
	rehearse: driveRehearse,
	prove:    runner.ProveCandidate,
	commit:   update.CommitStaged,

	harnessOwed:  runner.HarnessOwed,
	harnessMoved: runner.HarnessMoved,

	verifyKept: update.VerifyKept,
	pinned:     runner.PinnedBinary,
}

// driveReexec replaces this process. It returns only on a failure. Every descriptor
// abcd opens is close-on-exec, the drive's lock and log among them, and a caught signal
// goes back to its default across exec, so nothing needs closing first and a refused
// exec leaves the drive as it was. The process group, the terminal and the standard
// streams carry over.
var driveReexec = func(path string, args, env []string) error {
	// #nosec G204 -- path is the running binary's own, read before the swap.
	return syscall.Exec(path, args, env)
}

// driveArgs are the arguments this process was started with.
var driveArgs = func() []string { return os.Args }

// driveRehearsalTimeout bounds the staged binary's whole read-only run.
var driveRehearsalTimeout = 2 * time.Minute

// driveUpdate is the step's state for one process: the check's clock and the releases
// not to take again.
type driveUpdate struct {
	repoRoot string
	p        runner.Profile
	f        driveFlags
	stop     <-chan struct{}
	say      func(string)
	detail   io.Writer
	now      func() time.Time
	// exe is the path the run pinned its binary under, read before any swap: on Linux the
	// name this process reports for itself follows its file to the kept copy.
	exe   string
	last  time.Time
	tried map[string]bool
	// saidHeld is the release last said to wait on a harness the run cannot admit.
	saidHeld string
	// rehearsalSaid is each reason a rehearsal failed that has been said.
	rehearsalSaid map[string]bool
}

func newDriveUpdate(repoRoot string, p runner.Profile, f driveFlags, stop <-chan struct{}, say func(string), detail io.Writer) *driveUpdate {
	u := &driveUpdate{repoRoot: repoRoot, p: p, f: f, stop: stop, say: say, detail: detail, now: time.Now, exe: runner.SelfBinary(), tried: map[string]bool{}, rehearsalSaid: map[string]bool{}}
	for _, tag := range f.refused {
		u.tried[tag] = true
	}
	// A drive that has just moved has just asked: the clock goes on across the move.
	if f.resuming() {
		u.last = u.now()
	}
	return u
}

// resuming reports whether this is a drive's own re-execution: the yes was given once
// and the walls proved by the process that started it.
func (f driveFlags) resuming() bool { return f.afterUpdate != "" || len(f.refused) > 0 }

func (u *driveUpdate) halted(ctx context.Context) bool {
	if ctx.Err() != nil {
		return true
	}
	select {
	case <-u.stop:
		return true
	default:
		return false
	}
}

// note writes one line to the log alone.
func (u *driveUpdate) note(format string, args ...any) {
	line := termsafe.Sanitize(fsutil.RedactHome(fmt.Sprintf(format, args...)))
	fmt.Fprintln(u.detail, time.Now().UTC().Format(time.RFC3339)+" "+driveLine+line)
}

// tell says one line.
func (u *driveUpdate) tell(format string, args ...any) {
	u.say(driveLine + termsafe.Sanitize(fsutil.RedactHome(fmt.Sprintf(format, args...))))
}

// notTaken says once that tag was not taken, and takes it off the table for the run.
func (u *driveUpdate) notTaken(tag, why string) {
	u.tried[tag] = true
	u.tell("a newer release %s was not taken: %s; going on with %s", tag, why, u.f.self.Version)
}

// atBoundary is the step, run with no lane in hand. It reports whether it asked the
// origin, so that the loop checks everything again before anything starts.
func (u *driveUpdate) atBoundary(ctx context.Context) bool {
	if u.f.noSelfUpdate || u.f.self.Refusal != nil || u.halted(ctx) {
		return false
	}
	now := u.now()
	if !u.last.IsZero() && now.Sub(u.last) < driveCheckEvery {
		return false
	}
	u.last = now
	// A failure here is the loop's to end, on its own next check.
	if driveWalls.settings(u.p) != nil {
		return true
	}
	tag, newer, err := driveSelfUpdateSteps.candidate(u.f.self.Version)
	switch {
	case err != nil:
		u.tell("the release origin gave no latest release: %s; going on with %s, and the next check asks again", err.Error(), u.f.self.Version)
	case !newer:
		u.note("the latest release is %s, nothing newer than %s to take", tag, u.f.self.Version)
	case u.tried[tag]:
		u.note("the latest release is %s, which this run does not take again", tag)
	default:
		u.take(ctx, tag)
	}
	return true
}

// take stages tag, rehearses and proves it, swaps it in and re-executes. Whatever goes
// wrong before the swap leaves the drive where it was.
func (u *driveUpdate) take(ctx context.Context, tag string) {
	steps := driveSelfUpdateSteps
	target, from := u.f.self.Path, u.f.self.Version
	// A proof on the file the run holds says nothing of another a re-executed start would
	// pin, so a check whose harness moved fetches nothing and rehearses nothing.
	if steps.harnessMoved(u.p) {
		switch {
		case steps.harnessOwed(u.p):
			u.tell("a newer release %s is waiting, and the harness has moved to another file, which the next check before a stage proves; going on with %s, and the next check tries it again", tag, from)
		case u.saidHeld != tag:
			u.saidHeld = tag
			u.tell("a newer release %s is waiting, and the harness's name does not lead to a file this run's signing team admits, so the release waits and the run carries on with the file it proved, on %s; each check looks again", tag, from)
		}
		return
	}
	st, err := steps.stage(target, tag)
	if err != nil {
		// A fault that says nothing of the release leaves it to the next check, as a
		// limit at the proof does: a run lasts weeks, and a passing fault must not cost it
		// a release.
		var none *update.NoVerdictError
		if errors.As(err, &none) {
			u.tell("a newer release %s could not be fetched just now: %s; going on with %s, and the next check tries it again", tag, none.Error(), from)
			return
		}
		u.notTaken(tag, err.Error())
		return
	}
	swapped := false
	defer func() {
		if !swapped {
			_ = os.Remove(st.Path)
		}
	}()
	args := driveReexecArgs(driveArgs(), u.p, from, tag)
	if u.halted(ctx) {
		return
	}
	// The read-only run comes first: it spends nothing, and a release that cannot plan
	// a drain has no use for a live launch.
	candidate := driveCandidate{repoRoot: u.repoRoot, path: st.Path, sha256: st.SHA256, env: runner.HookedEnv(u.p)(os.Environ())}
	if err := steps.rehearse(ctx, candidate, args); err != nil {
		switch {
		case u.halted(ctx):
		case u.fault(tag, err):
		default:
			u.rehearsalFailed(tag, err.Error())
		}
		return
	}
	if u.halted(ctx) {
		return
	}
	if err := driveCheckBeforeExec(st.Path, st.SHA256); err != nil {
		u.fault(tag, err)
		return
	}
	if err := steps.prove(ctx, u.p, st.Path); err != nil {
		var limit *runner.LimitError
		var none *runner.NoVerdictError
		switch {
		case u.halted(ctx):
		case errors.As(err, &limit), errors.As(err, &none):
			// No verdict on any wall is no verdict on the release: the next check asks again.
			u.tell("a newer release %s could not be proved just now: %s; going on with %s, and the next check tries it again", tag, err.Error(), from)
		default:
			u.notTaken(tag, err.Error())
		}
		return
	}
	if u.halted(ctx) {
		return
	}
	// The re-executed drive proves no wall live, on the strength of the proof just passed.
	// A harness moved or owed around it stays with this drive, whose next wall check proves
	// it, and says nothing of the release.
	if steps.harnessOwed(u.p) || steps.harnessMoved(u.p) {
		u.tell("a newer release %s was proved, and the harness changed around that proof; going on with %s, and the next check tries it again", tag, from)
		return
	}
	// A re-executed start pins what stands, so anything else that changed since this check
	// began, new bytes at the harness's own path, a program or a declaration, would become
	// its baseline. The proof has put the settings back, so a difference is a real one, and
	// the loop's own comparison ends the run on it.
	if driveWalls.settings(u.p) != nil {
		return
	}
	keep := update.KeptPath(target)
	if err := steps.commit(target, st, keep); err != nil {
		if !u.fault(tag, err) {
			u.notTaken(tag, err.Error())
		}
		return
	}
	swapped = true
	// As abcd update does, after the swap and never as an input.
	ahoy.RefreshPathEntryDigest(target, st.SHA256)
	if err := ahoy.RecordReleaseMark(tag, false); err != nil {
		u.note("the newest-release mark could not be recorded: %s", err.Error())
	}
	// A re-execution would lose a stop that has just arrived, so the run ends as asked
	// and the next start runs the new binary.
	if u.halted(ctx) {
		u.tell("%s is installed, and a stop came before this drive moved onto it; the next drive starts on it", tag)
		return
	}
	// The installed name is held to the verified digest once more. A file that changed
	// here is no verdict on the release, and what the run holds goes back in place.
	err = driveCheckInstalled(target, st.SHA256)
	if err == nil {
		err = driveReexec(target, args, os.Environ())
		if err == nil {
			return // only a stand-in returns with nothing to say
		}
		u.tried[tag] = true
	}
	// The run already holds the digest of the binary it started on, so nothing else goes
	// back in place, and no request is needed to know it.
	data, rerr := update.ReadKept(keep)
	if pinned, ok := steps.pinned(u.p, u.exe); rerr == nil && (!ok || sha256.Sum256(data) != pinned) {
		rerr = errors.New("the file there does not match the binary this run started on")
	}
	digest := ""
	if rerr == nil {
		digest, rerr = update.RestoreKept(target, keep, data)
	}
	if rerr != nil {
		u.tell("%s could not be started (%s), and the previous abcd could not be put back: %s; it is kept at %s", tag, err.Error(), rerr.Error(), keep)
		return
	}
	ahoy.RefreshPathEntryDigest(target, digest)
	u.tell("%s could not be started: %s; the previous abcd is back in place, and the drive goes on with %s", tag, err.Error(), from)
}

// rehearsalFailed says a failed read-only run once per reason, and leaves the tag open: it
// reads this checkout as it stands, so it may pass at a later check.
func (u *driveUpdate) rehearsalFailed(tag, why string) {
	key := tag + " " + why
	if u.rehearsalSaid[key] {
		u.note("%s still does not pass its read-only run: %s", tag, why)
		return
	}
	u.rehearsalSaid[key] = true
	u.tell("a newer release %s did not pass its read-only run: %s; going on with %s, and later checks try it again", tag, why, u.f.self.Version)
}

// fault says once that err says nothing of the release, and reports whether it was such a
// fault. The tag stays open, and the next check tries it again.
func (u *driveUpdate) fault(tag string, err error) bool {
	var none *update.NoVerdictError
	if !errors.As(err, &none) {
		return false
	}
	u.tell("a newer release %s could not be taken just now: %s; going on with %s, and the next check tries it again", tag, none.Error(), u.f.self.Version)
	return true
}

// flagGiven reports whether args carry flag, in either of its spellings.
func flagGiven(args []string, flag string) bool {
	return slices.ContainsFunc(args, func(a string) bool { return a == flag || strings.HasPrefix(a, flag+"=") })
}

// withoutFlag is args with each occurrence of a flag taking a value taken out.
func withoutFlag(args []string, flag string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == flag:
			i++
		case strings.HasPrefix(args[i], flag+"="):
		default:
			out = append(out, args[i])
		}
	}
	return out
}

// driveReexecArgs are what a drive re-executes itself with: its own arguments, this
// move's flag in place of any earlier one, the yes it was given once, and the profile
// it runs on by name, so a second directory appearing there since is no reason to stop.
func driveReexecArgs(args []string, p runner.Profile, from, to string) []string {
	kept := args
	for _, name := range driveStateHolds {
		kept = withoutFlag(kept, name)
	}
	out := append(withoutFlag(kept, driveAfterUpdateFlag), driveAfterUpdateFlag, from+"="+to)
	if !flagGiven(out, "--yes") {
		out = append(out, "--yes")
	}
	if !flagGiven(out, "--profile") {
		out = append(out, "--profile", p.Name())
	}
	return out
}

// driveStartRetries is what a resuming start waits before each further attempt.
var driveStartRetries = []time.Duration{15 * time.Second, 45 * time.Second, 2 * time.Minute}

// driveStartWait waits out one of them, a seam the tests replace.
var driveStartWait = waitForStart

// waitForStart returns when d has passed, or with ctx's error when ctx ends first.
func waitForStart(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// driveVerifyRetries is what the way back waits before each further asking of the kept
// copy's verification, when the last came to no answer.
var driveVerifyRetries = []time.Duration{5 * time.Second, 15 * time.Second, 30 * time.Second, time.Minute}

// driveStartFailed reports whether err is a start that failed, rather than another
// drive's hold, a stop or a limit, which end a resuming drive as they end any.
func driveStartFailed(err error) bool {
	var coded *exitError
	return !errors.As(err, &coded) || (coded.Code == 2 && !coded.wall)
}

// driveStoppedStarting is the end of a resuming start a stop was asked for during: nothing
// goes back, since a stop is what was asked for and says nothing of the release.
func driveStoppedStarting() error {
	return &exitError{Code: driveKilledExit, Msg: driveLine + "a stop was asked for during the start, so it ends here with nothing restored"}
}

// driveEndedOnWall reports whether a comparison ended the drive.
func driveEndedOnWall(err error) bool {
	var coded *exitError
	return errors.As(err, &coded) && coded.wall
}

// driveFileDigest is a file's content by hash, "" when it cannot be read.
func driveFileDigest(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// driveStillOnItsBinaries holds an attempt after a failed one to what this process began
// on and to anything it pinned, before it pins anything new: another attempt would write a
// change made during the wait in as the baseline, and the likeliest failed attempt pinned
// nothing to compare with. It ends the run.
func driveStillOnItsBinaries(p runner.Profile, f driveFlags) error {
	if driveFileDigest(f.self.Path) != f.startedOn {
		return &exitError{Code: 2, wall: true, Msg: driveLine + termsafe.Sanitize(fsutil.RedactHome("the installed binary "+f.self.Path+" changed while the start waited to try again"))}
	}
	if err := f.began.Check(); err != nil {
		return &exitError{Code: 2, wall: true, Msg: driveLine + termsafe.Sanitize(fsutil.RedactHome(err.Error()))}
	}
	if err := driveWalls.binaries(p); err != nil {
		return &exitError{Code: 2, wall: true, Msg: driveLine + termsafe.Sanitize(fsutil.RedactHome(err.Error()))}
	}
	return nil
}

// driveWaitToStartAgain says the failed start and waits out wait, returning the killed
// exit if a signal comes first. Nothing is restored then: a stop is what was asked for.
func driveWaitToStartAgain(w io.Writer, repoRoot string, f driveFlags, cause error, wait time.Duration, try int) error {
	// The drive's own handling has ended, and a closed terminal is no reason to die here.
	pipes := make(chan os.Signal, 1)
	signal.Notify(pipes, syscall.SIGPIPE)
	defer signal.Stop(pipes)
	who := f.self.Version
	if f.afterUpdate != "" {
		_, who, _ = parseAfterUpdate(f.afterUpdate)
	}
	ctx, stop := drainSignalContext()
	defer stop()
	say, _, closeLog := driveLog(repoRoot, w)
	say(fmt.Sprintf("%s%s could not start: %s; trying again in %s, try %d of %d", driveLine, who,
		termsafe.Sanitize(fsutil.RedactHome(cause.Error())), wait, try+2, len(driveStartRetries)+1))
	closeLog()
	if driveStartWait(ctx, wait) != nil {
		return &exitError{Code: driveKilledExit, Msg: driveLine + "stopped before the start was tried again"}
	}
	return nil
}

// driveWaitToVerifyAgain says why the kept copy has no answer yet and waits out wait,
// returning the killed exit if a signal comes first, with nothing restored.
func driveWaitToVerifyAgain(w io.Writer, repoRoot, from string, cause error, wait time.Duration, try int) error {
	ctx, stop := drainSignalContext()
	defer stop()
	say, _, closeLog := driveLog(repoRoot, w)
	say(fmt.Sprintf("%sthe kept copy of %s has no answer yet: %s; asking again in %s, try %d of %d", driveLine, from,
		termsafe.Sanitize(fsutil.RedactHome(cause.Error())), wait, try+2, len(driveVerifyRetries)+1))
	closeLog()
	if driveStartWait(ctx, wait) != nil {
		return &exitError{Code: driveKilledExit, Msg: driveLine + "stopped before the kept copy was verified again, so nothing was restored"}
	}
	return nil
}

// driveCheckInstalled holds the installed name to the digest it was verified at, saying
// which file it was that changed.
func driveCheckInstalled(target, sum string) error {
	if err := driveCheckBeforeExec(target, sum); err != nil {
		return fmt.Errorf("the installed binary %s changed after it was verified", target)
	}
	return nil
}

// driveStateHolds are the flags a boundary's re-execution leaves out. The boundary runs
// only after a move, so the drain's state holds what they gave: given again, a cap could
// refuse the start it was set for, and a pace or a skip could undo what a person changed
// since.
var driveStateHolds = []string{"--max", "--pace", "--sub-agents", "--fix-rounds", "--skip", "--reason", "--unskip"}

// driveRunCandidate runs the staged binary in dir as a process group of its own, killed
// whole if ctx ends, and returns what it wrote to its standard output.
func driveRunCandidate(ctx context.Context, c driveCandidate, args ...string) ([]byte, error) {
	// #nosec G204 -- the file has passed the update verb's verification; args are fixed.
	if err := driveCheckBeforeExec(c.path, c.sha256); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, c.path, args...)
	cmd.Dir, cmd.Env = c.repoRoot, c.env
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	procgroup.Lead(cmd)
	cmd.Cancel = func() error { return procgroup.Kill(cmd) }
	cmd.WaitDelay = 5 * time.Second
	err := cmd.Run()
	procgroup.KillLeft(cmd)
	switch {
	case ctx.Err() != nil:
		return nil, fmt.Errorf("it did not finish within %s", driveRehearsalTimeout)
	case err != nil:
		last := strings.TrimSpace(stderr.String())
		if i := strings.LastIndexByte(last, '\n'); i >= 0 {
			last = last[i+1:]
		}
		if len(last) > 200 {
			last = last[:200]
		}
		return nil, fmt.Errorf("%v: %s", err, last)
	}
	return stdout.Bytes(), nil
}

// driveRehearse holds the staged binary to what the wall proof does not exercise, the
// drain verbs, before the installed binary is touched: it must accept every argument
// its re-execution would take, and answer `drain --dry-run --json`, which writes
// nothing and fetches nothing, with a plan.
func driveRehearse(ctx context.Context, c driveCandidate, args []string) error {
	ctx, cancel := context.WithTimeout(ctx, driveRehearsalTimeout)
	defer cancel()
	if _, err := driveRunCandidate(ctx, c, append(slices.Clone(args[1:]), "--help")...); err != nil {
		return fmt.Errorf("it did not take the arguments its re-execution would be given: %w", err)
	}
	out, err := driveRunCandidate(ctx, c, "drain", "--dry-run", "--json")
	if err != nil {
		return fmt.Errorf("`drain --dry-run` did not answer: %w", err)
	}
	var plan struct {
		DryRun bool `json:"dry_run"`
	}
	if err := json.Unmarshal(out, &plan); err != nil || !plan.DryRun {
		return errors.New("`drain --dry-run --json` did not answer with a plan")
	}
	return nil
}

// parseAfterUpdate reads --after-update <from>=<to>.
func parseAfterUpdate(value string) (from, to string, ok bool) {
	from, to, ok = strings.Cut(value, "=")
	return from, to, ok && ahoy.ReleaseTagShape.MatchString(from) && ahoy.ReleaseTagShape.MatchString(to)
}

// driveResumeRefused refuses the two hidden flags when they cannot be a drive's own
// re-execution, naming the flag and repeating no value. A state that cannot be read is
// left to the start, which falls back from it.
func driveResumeRefused(repoRoot string, f driveFlags) error {
	refuse := func(flag, why string) error {
		return &exitError{Code: 2, Msg: "abcd drain: " + flag + " is a drive's own, passed when it re-executes itself: " + why + " (nothing started, nothing reserved)"}
	}
	flag := driveUpdateRefusedFlag
	if f.afterUpdate != "" {
		flag = driveAfterUpdateFlag
		_, to, ok := parseAfterUpdate(f.afterUpdate)
		if !ok {
			return refuse(flag, "its value is not <from>=<to>, two release tags")
		}
		if to != f.self.Version {
			return refuse(flag, "it names a release this binary's own build does not claim")
		}
	}
	for _, tag := range f.refused {
		if !ahoy.ReleaseTagShape.MatchString(tag) {
			return refuse(driveUpdateRefusedFlag, "its value is not a release tag")
		}
	}
	if !f.resuming() {
		return nil
	}
	if f.self.Refusal != nil {
		return refuse(flag, "this build never self-updates")
	}
	for _, k := range driveSessionMarkers {
		if os.Getenv(k) != "" {
			return refuse(flag, k+" is set, so this is an agent's own session")
		}
	}
	if progress, err := loop.ReadDrainProgress(repoRoot); err == nil && !progress.Live {
		return refuse(flag, "no drain is in progress")
	}
	return nil
}

// driveReleaseLine is the consent screen's line on releases: the check named, so the yes
// is the explicit ask adr-38's second tier requires, or why there is none.
func driveReleaseLine(f driveFlags) string {
	switch {
	case f.noSelfUpdate:
		return "none (--no-self-update)"
	case f.self.Refusal != nil:
		return "none: " + termsafe.Sanitize(f.self.Refusal.Detail)
	}
	return "asks the release origin about once an hour, at a lane boundary, and takes a newer stable release after the update verb's verification and the live check, going on from the drain's state (--no-self-update turns it off)"
}

// driveKeptNote says how the previous binary comes back, which past the start is the
// person's act: the kept copy when it verifies as that release, and otherwise the update
// verb with its tag.
func driveKeptNote(w io.Writer, f driveFlags) {
	target := f.self.Path
	keep := update.KeptPath(target)
	from, _, _ := parseAfterUpdate(f.afterUpdate)
	line := "to go back to " + from + ", run abcd update " + from + ", then abcd drain --drive again"
	if _, err := driveSelfUpdateSteps.verifyKept(keep, from); err == nil {
		line = "the abcd this run moved from is kept at " + keep + "; to go back to it, move it over " + target + ", then run abcd drain --drive again"
	}
	fmt.Fprintln(w, driveLine+termsafe.Sanitize(fsutil.RedactHome(line)))
}

// driveGoBack is a re-executed start that kept failing before its first move: the kept copy goes
// back in place, and the drive carries on on it with the release named as refused, in
// its arguments, where no file decides it. Only a kept copy verified as the release it
// replaced goes back: what cannot be verified renames nothing, and a verification that
// came to no answer is asked again over about two minutes, the lock free between tries,
// before it counts as that. It takes the checkout's drive lock for the rename, so another
// drive's checkout is left alone. A failure from here ends the run, naming the kept copy.
func driveGoBack(w io.Writer, repoRoot string, f driveFlags, cause error) error {
	from, to, _ := parseAfterUpdate(f.afterUpdate)
	target := f.self.Path
	keep := update.KeptPath(target)
	why := termsafe.Sanitize(fsutil.RedactHome(cause.Error()))
	// The drive's own handling has ended, and a closed terminal is no reason to die here.
	pipes := make(chan os.Signal, 1)
	signal.Notify(pipes, syscall.SIGPIPE)
	defer signal.Stop(pipes)
	var digest string
	var unverified error
	var err error
	for try := 0; ; try++ {
		unverified = nil
		err = fsutil.WithFileLock(filepath.Join(repoRoot, filepath.FromSlash(driveLockRel)), driveLockWait, func() error {
			data, verr := driveSelfUpdateSteps.verifyKept(keep, from)
			if verr != nil {
				unverified = verr
				return nil
			}
			var rerr error
			digest, rerr = update.RestoreKept(target, keep, data)
			return rerr
		})
		var none *update.NoVerdictError
		if err != nil || !errors.As(unverified, &none) || try == len(driveVerifyRetries) {
			break
		}
		if werr := driveWaitToVerifyAgain(w, repoRoot, from, unverified, driveVerifyRetries[try], try); werr != nil {
			return werr
		}
	}
	if errors.Is(err, fsutil.ErrLockContention) {
		return cause
	}
	if unverified != nil {
		return &exitError{Code: 2, Msg: driveLine + termsafe.Sanitize(fsutil.RedactHome(to+" could not start ("+why+"), and the copy at "+keep+" did not verify as "+from+": "+unverified.Error()+"; nothing was restored: run abcd update "+from+", then abcd drain --drive again"))}
	}
	if err != nil {
		return &exitError{Code: 2, Msg: driveLine + termsafe.Sanitize(fsutil.RedactHome(to+" could not start ("+why+"), and the abcd it replaced could not be restored from its kept copy at "+keep+": "+err.Error()+"; run abcd update "+from+", then abcd drain --drive again"))}
	}
	ahoy.RefreshPathEntryDigest(target, digest)
	say, _, closeLog := driveLog(repoRoot, w)
	say(driveLine + to + " could not start: " + why + "; " + from + " is restored from the kept copy and the drive goes on with it, and " + to + " is not tried again in this run")
	closeLog()
	args := append(withoutFlag(driveArgs(), driveAfterUpdateFlag), driveUpdateRefusedFlag, to)
	err = driveCheckInstalled(target, digest)
	if err == nil {
		err = driveReexec(target, args, os.Environ())
	}
	if err != nil {
		return &exitError{Code: 2, Msg: driveLine + termsafe.Sanitize(fsutil.RedactHome("the kept copy of "+from+" is back at "+target+", and it could not be started: "+err.Error()+"; run abcd drain --drive again"))}
	}
	return nil
}
