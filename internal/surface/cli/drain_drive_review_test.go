package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/core/update"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// TestAReexecutionCarriesOnlyWhatTheStateDoesNotHold: the boundary runs only after a
// successful move, so the state already holds the cap, the pace and the skips. Each is
// given once: re-applied, it would undo what a person changed since the start.
func TestAReexecutionCarriesOnlyWhatTheStateDoesNotHold(t *testing.T) {
	w := newUpdateWorld(t)
	driveArgs = func() []string {
		return []string{"abcd", "drain", "--drive", "--yes", "--max", "5", "--pace=30/10", "--sub-agents", "2", "--fix-rounds=3",
			"--skip", "iss-1", "--skip=iss-2", "--reason", "held", "--unskip", "iss-3", "--until", "2026-12-01", "--take-over", "--json", "--profile", "walled"}
	}
	w.step(t).atBoundary(context.Background())
	want := []string{"abcd", "drain", "--drive", "--yes", "--until", "2026-12-01", "--take-over", "--json", "--profile", "walled", "--after-update", updFrom + "=" + updTo}
	if len(w.reexecs) != 1 || !slices.Equal(w.reexecs[0].args, want) {
		t.Fatalf("got %v, want %q", w.reexecs, want)
	}
}

// TestAResumeOverADrainTheDriveCappedDoesNotRefuseItself: --max 5 over a live drain of two
// lanes with no cap makes the state's cap seven; a re-execution that carried the 5 would
// refuse itself, and the kept copy would refuse again.
func TestAResumeOverADrainTheDriveCappedDoesNotRefuseItself(t *testing.T) {
	w := newUpdateWorld(t)
	w.passWalls(t)
	state := filepath.Join(w.repo, filepath.FromSlash(loop.DrainStateRel))
	if err := os.MkdirAll(filepath.Dir(state), 0o700); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(loop.DrainState{SchemaVersion: loop.DrainSchemaVersion, Lanes: []loop.DrainLane{
		{Issue: "iss-2609291313276243", RunID: "run-2610160000000001"},
		{Issue: "iss-2609291313276244", RunID: "run-2610160000000002"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(state, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	typed := []string{"abcd", "drain", "--drive", "--yes", "--max", "5"}
	var out, errb strings.Builder
	if code := Run(typed[1:], &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	progress, err := loop.ReadDrainProgress(w.repo)
	if err != nil || progress.Max != 7 {
		t.Fatalf("the cap over two lanes: %+v, %v", progress, err)
	}
	w.useRelease(updTo)
	args := driveReexecArgs(typed, runner.Profile{Dir: w.profile}, updFrom, updTo)
	if code := Run(args[1:], &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if progress, err := loop.ReadDrainProgress(w.repo); err != nil || progress.Max != 7 {
		t.Fatalf("the cap stands: %+v, %v", progress, err)
	}
	if w.terminalCalls != 2 {
		t.Fatalf("the loop ran %d times", w.terminalCalls)
	}
}

// TestAStartThatMayTakeAReleaseSaysWhatItDoesToOtherDrives: a second drive is not refused,
// since drives in two checkouts on one account work today; the start says what a move does
// to it.
func TestAStartThatMayTakeAReleaseSaysWhatItDoesToOtherDrives(t *testing.T) {
	const line = "any other drive on this account ends when this one moves to a new release, whatever flags it has"
	consent := func(t *testing.T, args ...string) string {
		t.Helper()
		var out, errb strings.Builder
		if code := Run(append([]string{"drain", "--drive", "--yes"}, args...), &out, &errb); code != 0 {
			t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
		}
		return strings.Join(strings.Fields(out.String()+errb.String()), " ")
	}
	t.Run("may take one", func(t *testing.T) {
		w := newUpdateWorld(t)
		w.passWalls(t)
		if text := consent(t); !strings.Contains(text, line) {
			t.Fatalf("the start lacks the line:\n%s", text)
		}
	})
	t.Run("takes none", func(t *testing.T) {
		w := newUpdateWorld(t)
		w.passWalls(t)
		if text := consent(t, "--no-self-update"); strings.Contains(text, "ends when this one moves") {
			t.Fatalf("a drive that never moves says it does:\n%s", text)
		}
	})
	t.Run("never can", func(t *testing.T) {
		w := newUpdateWorld(t)
		w.passWalls(t)
		driveSelfUpdateSteps.target = func() update.SelfTarget {
			return update.SelfTarget{Path: w.target, Refusal: &update.Refusal{Shape: update.ShapeSourceBuild, Detail: "a source build"}}
		}
		if text := consent(t); strings.Contains(text, "ends when this one moves") {
			t.Fatalf("a drive that never moves says it does:\n%s", text)
		}
	})
}

// failingStarts has the drive's static check fail its first n times, and counts them all.
func failingStarts(n int) *int {
	calls := 0
	driveWalls.static = func(context.Context, runner.Profile) ([]string, error) {
		calls++
		if calls <= n {
			return nil, errors.New("a file was held for a moment")
		}
		return nil, nil
	}
	return &calls
}

// TestAPassingFailureAtAResumeIsNoVerdictOnTheRelease: a start that fails once and then
// holds goes on, with the release it moved to and one line naming the move.
func TestAPassingFailureAtAResumeIsNoVerdictOnTheRelease(t *testing.T) {
	w := goBackWorld(t)
	w.recordTerminal(t)
	calls := failingStarts(1)
	var out, errb strings.Builder
	if code := Run(resumeArgs(), &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	text := out.String() + errb.String()
	w.restoredNothing(t, "new-bytes")
	if *calls != 2 || w.terminalCalls != 1 || !slices.Equal(w.startWaits, driveStartRetries[:1]) {
		t.Fatalf("%d starts, %d runs, waits %v:\n%s", *calls, w.terminalCalls, w.startWaits, text)
	}
	if !strings.Contains(text, "a file was held for a moment") || !strings.Contains(text, "again") {
		t.Fatalf("the try is said:\n%s", text)
	}
	if got := strings.Count(text, update.UpdatedLine(updFrom, updTo)); got != 1 {
		t.Fatalf("%d lines name the move:\n%s", got, text)
	}
}

// TestAFailureThatRepeatsGoesBackOnceEveryTryIsSpent: only a failure that repeats counts
// against the release.
func TestAFailureThatRepeatsGoesBackOnceEveryTryIsSpent(t *testing.T) {
	w := goBackWorld(t)
	calls := failingStarts(1 << 10)
	var out, errb strings.Builder
	if code := Run(resumeArgs(), &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if *calls != 1+len(driveStartRetries) || !slices.Equal(w.startWaits, driveStartRetries) {
		t.Fatalf("%d starts, waits %v", *calls, w.startWaits)
	}
	if got := w.body(t, w.target); got != "old-bytes" || len(w.reexecs) != 1 {
		t.Fatalf("the installed file holds %q, started %v", got, w.reexecs)
	}
}

// TestTheStartsTriesAreBounded: a resume waits a few minutes at most in all.
func TestTheStartsTriesAreBounded(t *testing.T) {
	var total time.Duration
	for _, d := range driveStartRetries {
		total += d
	}
	if len(driveStartRetries) < 2 || total < time.Minute || total > 5*time.Minute {
		t.Fatalf("%d tries over %v", len(driveStartRetries), total)
	}
}

// TestAStopDuringAResumesWaitEndsItWithNothingRestored: a stop is what was asked for, and
// says nothing of the release. The signal is real, and the wait is the real one.
func TestAStopDuringAResumesWaitEndsItWithNothingRestored(t *testing.T) {
	w := goBackWorld(t)
	calls := failingStarts(1 << 10)
	driveStartWait = func(ctx context.Context, _ time.Duration) error {
		if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
			t.Error(err)
		}
		return waitForStart(ctx, time.Hour)
	}
	var out, errb strings.Builder
	if code := Run(resumeArgs(), &out, &errb); code != driveKilledExit {
		t.Fatalf("exit %d, want %d:\n%s%s", code, driveKilledExit, out.String(), errb.String())
	}
	w.restoredNothing(t, "new-bytes")
	if *calls != 1 {
		t.Fatalf("%d starts", *calls)
	}
}

// TestAKeptCopyTriesAgainToo: a passing failure at the kept copy's start does not end the
// run, and one that repeats ends it with nothing renamed.
func TestAKeptCopyTriesAgainToo(t *testing.T) {
	kept := func(t *testing.T) *updateWorld {
		t.Helper()
		w := newUpdateWorld(t)
		w.passWalls(t)
		w.liveDrain(t, 3)
		return w
	}
	t.Run("passing", func(t *testing.T) {
		w := kept(t)
		w.recordTerminal(t)
		calls := failingStarts(2)
		var out, errb strings.Builder
		if code := Run([]string{"drain", "--drive", "--yes", driveUpdateRefusedFlag, updTo}, &out, &errb); code != 0 {
			t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
		}
		text := out.String() + errb.String()
		if *calls != 3 || w.terminalCalls != 1 {
			t.Fatalf("%d starts, %d runs", *calls, w.terminalCalls)
		}
		if got := strings.Count(text, "going on with "+updFrom); got != 1 {
			t.Fatalf("%d lines say it goes on:\n%s", got, text)
		}
	})
	t.Run("repeating", func(t *testing.T) {
		w := kept(t)
		calls := failingStarts(1 << 10)
		refusedDrive(t, "drain", "--drive", "--yes", driveUpdateRefusedFlag, updTo)
		if *calls != 1+len(driveStartRetries) {
			t.Fatalf("%d starts", *calls)
		}
		w.restoredNothing(t, "old-bytes")
	})
}

// TestAFirstStartIsNotTriedAgain: a start a person typed answers them at once.
func TestAFirstStartIsNotTriedAgain(t *testing.T) {
	w := newUpdateWorld(t)
	w.passWalls(t)
	calls := failingStarts(1 << 10)
	refusedDrive(t, "drain", "--drive", "--yes")
	if *calls != 1 || len(w.startWaits) != 0 {
		t.Fatalf("%d starts, waits %v", *calls, w.startWaits)
	}
}

// TestAStartThatMovedOnceSaysSoOnce: an attempt that reached the loop and failed there
// has said the move, and the next attempt does not say it again.
func TestAStartThatMovedOnceSaysSoOnce(t *testing.T) {
	goBackWorld(t)
	failingStarts(0)
	runs := 0
	term := driveTerminal
	t.Cleanup(func() { driveTerminal = term })
	driveTerminal = func(context.Context, string, loop.TerminalOptions) (loop.TerminalResult, error) {
		if runs++; runs == 1 {
			return loop.TerminalResult{Ended: loop.TerminalRefused}, errors.New("a moment's refusal")
		}
		return loop.TerminalResult{Ended: loop.DrainStoppedEmpty}, nil
	}
	var out, errb strings.Builder
	if code := Run(resumeArgs(), &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	text := out.String() + errb.String()
	if got := strings.Count(text, update.UpdatedLine(updFrom, updTo)); got != 1 || runs != 2 {
		t.Fatalf("%d lines name the move over %d runs:\n%s", got, runs, text)
	}
}

// terminalScript has the loop do one thing on each run.
func terminalScript(t *testing.T, runs ...func(loop.TerminalOptions) (loop.TerminalResult, error)) *int {
	t.Helper()
	term := driveTerminal
	t.Cleanup(func() { driveTerminal = term })
	calls := 0
	driveTerminal = func(_ context.Context, _ string, o loop.TerminalOptions) (loop.TerminalResult, error) {
		run := runs[min(calls, len(runs)-1)]
		calls++
		return run(o)
	}
	return &calls
}

// TestAPanicBeforeTheFirstMoveGoesBack: a release that panics in its start or in its loop
// before a move is a failed start, and goes back through the kept copy's verification.
func TestAPanicBeforeTheFirstMoveGoesBack(t *testing.T) {
	panics := func(loop.TerminalOptions) (loop.TerminalResult, error) { panic("this release reads a nil map") }
	t.Run("in the loop to a copy that verifies", func(t *testing.T) {
		w := goBackWorld(t)
		failingStarts(0)
		calls := terminalScript(t, panics)
		var out, errb strings.Builder
		if code := Run(resumeArgs(), &out, &errb); code != 0 {
			t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
		}
		if got := w.body(t, w.target); got != "old-bytes" || len(w.reexecs) != 1 || *calls != 1+len(driveStartRetries) {
			t.Fatalf("the installed file holds %q, started %v, %d runs", got, w.reexecs, *calls)
		}
		if !slices.Contains(w.reexecs[0].args, driveUpdateRefusedFlag) {
			t.Fatalf("the kept copy starts with the version refused: %v", w.reexecs[0].args)
		}
		if text := out.String() + errb.String(); !strings.Contains(text, "reads a nil map") {
			t.Fatalf("the line says why:\n%s", text)
		}
	})
	t.Run("at the start to a copy that does not", func(t *testing.T) {
		w := goBackWorld(t)
		w.aForeignKeptCopy(t)
		driveWalls.static = func(context.Context, runner.Profile) ([]string, error) { panic("this release reads a nil map") }
		text := refusedDrive(t, resumeArgs()...)
		w.restoredNothing(t, "new-bytes")
		if !strings.Contains(text, "update "+updFrom) {
			t.Fatalf("the exit names the way on: %s", text)
		}
	})
	t.Run("on a first start which is no resume", func(t *testing.T) {
		w := newUpdateWorld(t)
		w.passWalls(t)
		driveWalls.static = func(context.Context, runner.Profile) ([]string, error) { panic("a first start's panic") }
		defer func() {
			if recover() == nil {
				t.Fatal("a panic in a start a person typed reads as a refusal")
			}
		}()
		var out, errb strings.Builder
		Run([]string{"drain", "--drive", "--yes"}, &out, &errb)
	})
}

// TestAPanicAfterTheFirstMoveIsLeftAlone: past a move the drive runs as any does, and the
// process that panicked is not the one to decide what is installed.
func TestAPanicAfterTheFirstMoveIsLeftAlone(t *testing.T) {
	w := goBackWorld(t)
	failingStarts(0)
	terminalScript(t, func(o loop.TerminalOptions) (loop.TerminalResult, error) {
		o.Moved()
		panic("a panic in a later lane")
	})
	defer func() {
		if recover() == nil {
			t.Fatal("the panic was swallowed")
		}
		w.restoredNothing(t, "new-bytes")
	}()
	var out, errb strings.Builder
	Run(resumeArgs(), &out, &errb)
}

// TestAFirstMoveThatRefusesGoesBack: a release that reaches its loop and whose first move
// refuses never moved, so it counts as a failed start.
func TestAFirstMoveThatRefusesGoesBack(t *testing.T) {
	for name, ending := range map[string]func(loop.TerminalOptions) (loop.TerminalResult, error){
		"refused": func(loop.TerminalOptions) (loop.TerminalResult, error) {
			return loop.TerminalResult{Ended: loop.TerminalRefused}, errors.New("this release cannot read the state")
		},
	} {
		t.Run(name, func(t *testing.T) {
			w := goBackWorld(t)
			failingStarts(0)
			calls := terminalScript(t, ending)
			var out, errb strings.Builder
			if code := Run(resumeArgs(), &out, &errb); code != 0 {
				t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
			}
			if got := w.body(t, w.target); got != "old-bytes" || len(w.reexecs) != 1 || *calls != 1+len(driveStartRetries) {
				t.Fatalf("the installed file holds %q, started %v, %d runs", got, w.reexecs, *calls)
			}
		})
	}
}

// TestAnEndingBeforeTheFirstMoveThatIsNoFailureGoesBackToNothing: a stop, a limit or a
// wait that ends the loop before it moves says nothing of the release.
func TestAnEndingBeforeTheFirstMoveThatIsNoFailureGoesBackToNothing(t *testing.T) {
	for _, ended := range []string{loop.TerminalStopped, loop.TerminalLimited, loop.TerminalWaited} {
		t.Run(ended, func(t *testing.T) {
			w := goBackWorld(t)
			failingStarts(0)
			calls := terminalScript(t, func(loop.TerminalOptions) (loop.TerminalResult, error) {
				return loop.TerminalResult{Ended: ended}, nil
			})
			var out, errb strings.Builder
			if code := Run(resumeArgs(), &out, &errb); code != 3 {
				t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
			}
			w.restoredNothing(t, "new-bytes")
			if *calls != 1 {
				t.Fatalf("%d runs", *calls)
			}
		})
	}
}

// TestAHarnessOwedAfterTheProofSwapsNothing: the re-executed drive proves no wall live, on
// the strength of the proof before the swap, so a harness that changed around that proof
// stays with the running drive, whose next wall check proves it. The release is no worse
// for it, and the next check takes it.
func TestAHarnessOwedAfterTheProofSwapsNothing(t *testing.T) {
	w := newUpdateWorld(t)
	u := w.step(t)
	owed := true
	var asked []string
	driveSelfUpdateSteps.harnessOwed = func(p runner.Profile) bool {
		asked = append(asked, p.Dir)
		return owed
	}
	u.atBoundary(context.Background())
	w.nothingSwapped(t)
	if w.proved != 1 || len(w.reexecs) != 0 || len(asked) != 1 || asked[0] != w.profile {
		t.Fatalf("proved %d, started %v, asked %v", w.proved, w.reexecs, asked)
	}
	if u.tried[updTo] {
		t.Fatal("the tag is marked tried")
	}
	if len(w.lines) != 1 || !strings.Contains(w.lines[0], "harness") || !strings.Contains(w.lines[0], "next check") {
		t.Fatalf("the lines: %q", w.lines)
	}
	owed = false
	w.now = w.now.Add(driveCheckEvery)
	u.atBoundary(context.Background())
	if w.staged != 2 || len(w.reexecs) != 1 {
		t.Fatalf("the next check takes it: staged %d, started %v", w.staged, w.reexecs)
	}
}

// TestWhatAnOlderReleaseStartsThisOneWith: an older release, already installed and running
// on a person's machine, starts this one with these arguments, in its rehearsal and at its
// re-execution. They are that release's, fixed when it was cut, so they must not be edited:
// a row changed to make this test pass is a release that cannot be moved onto. Each is a
// literal for that reason. See the spec's limits and the drive's page.
func TestWhatAnOlderReleaseStartsThisOneWith(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
		// ran says the loop is reached; say is text the answer must hold.
		ran bool
		say string
	}{
		{name: "a re-execution", ran: true, say: "abcd updated from v0.17.0 to v0.18.0",
			args: []string{"drain", "--drive", "--after-update", "v0.17.0=v0.18.0", "--yes", "--profile", "walled"}},
		{name: "the same with the value in the flag", ran: true, say: "abcd updated from v0.17.0 to v0.18.0",
			args: []string{"drain", "--drive", "--after-update=v0.17.0=v0.18.0", "--yes", "--profile=walled"}},
		{name: "a kept copy's start", ran: true, say: "v0.18.0 could not start",
			args: []string{"drain", "--drive", "--update-refused", "v0.18.0", "--yes", "--profile", "walled"}},
		{name: "a kept copy's start twice over", ran: true, say: "v0.18.0 could not start",
			args: []string{"drain", "--drive", "--update-refused", "v0.19.0", "--update-refused", "v0.18.0", "--yes"}},
		{name: "the switch", ran: true, say: "none (--no-self-update)",
			args: []string{"drain", "--drive", "--yes", "--no-self-update"}},
		{name: "a bound", ran: true, say: "abcd updated from v0.17.0 to v0.18.0",
			args: []string{"drain", "--drive", "--until", "2099-01-01", "--after-update", "v0.17.0=v0.18.0", "--yes", "--profile", "walled"}},
		{name: "taking a lane over", ran: true, say: "abcd updated from v0.17.0 to v0.18.0",
			args: []string{"drain", "--drive", "--take-over", "--after-update", "v0.17.0=v0.18.0", "--yes", "--profile", "walled"}},
		{name: "an answer in JSON", ran: true, say: `"ended"`,
			args: []string{"drain", "--drive", "--json", "--after-update", "v0.17.0=v0.18.0", "--yes", "--profile", "walled"}},
		{name: "the rehearsal's parse", say: "abcd drain",
			args: []string{"drain", "--drive", "--after-update", "v0.17.0=v0.18.0", "--yes", "--profile", "walled", "--help"}},
		{name: "the rehearsal's dry run", say: `"dry_run": true`,
			args: []string{"drain", "--dry-run", "--json"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newUpdateWorld(t)
			w.passWalls(t)
			w.liveDrain(t, 3)
			if slices.Contains(tt.args, "--after-update") || strings.HasPrefix(strings.Join(tt.args, " "), "drain --drive --after-update=") {
				driveSelfUpdateSteps.target = func() update.SelfTarget {
					return update.SelfTarget{Path: w.target, Version: "v0.18.0"}
				}
			}
			var out, errb strings.Builder
			if code := Run(tt.args, &out, &errb); code != 0 {
				t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
			}
			if text := out.String() + errb.String(); !strings.Contains(text, tt.say) {
				t.Fatalf("no %q in:\n%s", tt.say, text)
			}
			if tt.ran != (w.terminalCalls == 1) {
				t.Fatalf("the loop ran %d times", w.terminalCalls)
			}
		})
	}
}

// TestAResumingDriveTellsTheLoopSo: the loop's report of what a run held back reaches a
// resuming drive only if it knows it is one.
func TestAResumingDriveTellsTheLoopSo(t *testing.T) {
	w := newUpdateWorld(t)
	w.passWalls(t)
	w.liveDrain(t, 3)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 || w.terminal.Resumed {
		t.Fatalf("a first start: exit %d, resumed %v\n%s%s", code, w.terminal.Resumed, out.String(), errb.String())
	}
	if code := Run([]string{"drain", "--drive", "--yes", "--update-refused", updTo}, &out, &errb); code != 0 || !w.terminal.Resumed {
		t.Fatalf("a kept copy: exit %d, resumed %v\n%s%s", code, w.terminal.Resumed, out.String(), errb.String())
	}
	w.useRelease(updTo)
	w.terminal = loop.TerminalOptions{}
	if code := Run(resumeArgs(), &out, &errb); code != 0 || !w.terminal.Resumed {
		t.Fatalf("a new release: exit %d, resumed %v\n%s%s", code, w.terminal.Resumed, out.String(), errb.String())
	}
}

// TestAHeldBackNameIsMadeSafeForTheTerminal: what a run held back can come from the drain's
// state, which is a file on the account.
func TestAHeldBackNameIsMadeSafeForTheTerminal(t *testing.T) {
	err := driveHeldBackExit([]string{"iss-1\x1b[2Jplain"})
	var coded *exitError
	if !errors.As(err, &coded) || coded.Code != 3 || strings.Contains(err.Error(), "\x1b") {
		t.Fatalf("the exit: %q", err.Error())
	}
}

// TestADriveStartedThroughALinkStartsWhatItChecked: the name it runs is the name it held to
// the digest, wherever a link the drive was started through points by then.
func TestADriveStartedThroughALinkStartsWhatItChecked(t *testing.T) {
	w := newUpdateWorld(t)
	u := w.step(t)
	other := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.WriteFile(other, []byte("something else"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "abcd")
	if err := os.Symlink(other, link); err != nil {
		t.Fatal(err)
	}
	u.exe = link
	u.atBoundary(context.Background())
	if len(w.reexecs) != 1 || w.reexecs[0].path != w.target {
		t.Fatalf("started %v, want %s", w.reexecs, w.target)
	}
}

// TestAnInstalledFileThatChangedIsNamedAsThat: the line says which file changed. The
// staging file is gone by then.
func TestAnInstalledFileThatChangedIsNamedAsThat(t *testing.T) {
	t.Run("at the move", func(t *testing.T) {
		w := newUpdateWorld(t)
		u := w.step(t)
		commit := driveSelfUpdateSteps.commit
		driveSelfUpdateSteps.commit = func(target string, st update.Staged, keepAs string) error {
			if err := commit(target, st, keepAs); err != nil {
				return err
			}
			return os.WriteFile(target, []byte("changed since"), 0o755)
		}
		u.atBoundary(context.Background())
		if len(w.reexecs) != 0 {
			t.Fatalf("started %v", w.reexecs)
		}
		text := strings.Join(w.lines, "\n")
		if strings.Contains(text, "staging file") || !strings.Contains(text, "installed binary") || !strings.Contains(text, filepath.Base(w.target)) {
			t.Fatalf("the line: %s", text)
		}
		if got := w.body(t, w.target); got != "old-bytes" {
			t.Fatalf("the recovery stands: the installed file holds %q", got)
		}
	})
	t.Run("going back", func(t *testing.T) {
		w := goBackWorld(t)
		failingStarts(1 << 10)
		driveCheck := driveCheckBeforeExec
		t.Cleanup(func() { driveCheckBeforeExec = driveCheck })
		driveCheckBeforeExec = func(string, string) error { return update.CheckStaged(w.target, sha("not these bytes")) }
		text := refusedDrive(t, resumeArgs()...)
		if strings.Contains(text, "staging file") || !strings.Contains(text, "installed binary") {
			t.Fatalf("the exit: %s", text)
		}
	})
}

// TestAKeptCopyThatVerifiesAndCannotBeRestoredNamesTheVerb: the way back named is the one
// that verifies what it installs.
func TestAKeptCopyThatVerifiesAndCannotBeRestoredNamesTheVerb(t *testing.T) {
	w := goBackWorld(t)
	dir := filepath.Dir(w.target)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	text := refusedDrive(t, resumeArgs()...)
	if strings.Contains(text, "move that copy") || !strings.Contains(text, "abcd update "+updFrom) {
		t.Fatalf("the exit: %s", text)
	}
}

// TestABinaryChangedDuringAResumesWaitEndsTheRun: the attempt after a failed one would pin
// whatever stands there, and a resume proves nothing of its own, so a file changed during
// the wait would become the run's baseline unproved. It ends the run instead.
func TestABinaryChangedDuringAResumesWaitEndsTheRun(t *testing.T) {
	for name, args := range map[string][]string{
		"a new release": resumeArgs(),
		"a kept copy":   {"drain", "--drive", "--yes", "--update-refused", updTo},
	} {
		t.Run(name, func(t *testing.T) {
			w := goBackWorld(t)
			if name == "a kept copy" {
				w.useRelease(updFrom)
			}
			w.recordTerminal(t)
			calls := failingStarts(1)
			driveStartWait = func(context.Context, time.Duration) error {
				return os.WriteFile(w.target, []byte("changed in the wait"), 0o755)
			}
			text := refusedDrive(t, args...)
			w.restoredNothing(t, "changed in the wait")
			if *calls != 1 || w.terminalCalls != 0 {
				t.Fatalf("%d starts, %d runs:\n%s", *calls, w.terminalCalls, text)
			}
			if !strings.Contains(text, "changed") {
				t.Fatalf("the exit says why:\n%s", text)
			}
		})
	}
}

// TestARetriedStartHoldsTheBinariesToThePinBeforeItWritesANewOne: the comparison comes
// first, on every attempt after the first and on no first one.
func TestARetriedStartHoldsTheBinariesToThePinBeforeItWritesANewOne(t *testing.T) {
	w := goBackWorld(t)
	w.recordTerminal(t)
	calls := failingStarts(1)
	held := 0
	driveWalls.binaries = func(runner.Profile) error {
		held++
		return &runner.WallError{Wall: runner.WallSettingsFile, Detail: "git's binary changed while the drive ran"}
	}
	text := refusedDrive(t, resumeArgs()...)
	w.restoredNothing(t, "new-bytes")
	if held != 1 || *calls != 1 || w.terminalCalls != 0 || len(w.startWaits) != 1 {
		t.Fatalf("held %d times, %d starts, %d runs, waits %v:\n%s", held, *calls, w.terminalCalls, w.startWaits, text)
	}
}

// TestARetriedResumeProvesItselfLive: a first attempt goes on the proof the file passed
// before the swap. An attempt after a wait has no such proof of what stands now.
func TestARetriedResumeProvesItselfLive(t *testing.T) {
	proofs := func(t *testing.T, fail error) (*updateWorld, *int) {
		t.Helper()
		w := goBackWorld(t)
		w.recordTerminal(t)
		proved := 0
		driveWalls.live = func(context.Context, runner.Profile) error {
			proved++
			return fail
		}
		return w, &proved
	}
	t.Run("a first attempt proves nothing", func(t *testing.T) {
		w, proved := proofs(t, nil)
		failingStarts(0)
		var out, errb strings.Builder
		if code := Run(resumeArgs(), &out, &errb); code != 0 || *proved != 0 || w.terminalCalls != 1 {
			t.Fatalf("exit %d, proved %d, %d runs", code, *proved, w.terminalCalls)
		}
	})
	t.Run("the second proves itself", func(t *testing.T) {
		w, proved := proofs(t, nil)
		failingStarts(1)
		var out, errb strings.Builder
		if code := Run(resumeArgs(), &out, &errb); code != 0 || *proved != 1 || w.terminalCalls != 1 {
			t.Fatalf("exit %d, proved %d, %d runs", code, *proved, w.terminalCalls)
		}
		w.restoredNothing(t, "new-bytes")
	})
	t.Run("and goes no further on a proof that fails", func(t *testing.T) {
		w, proved := proofs(t, &runner.WallError{Wall: runner.WallTools, Detail: "a tool too many"})
		failingStarts(1)
		var out, errb strings.Builder
		if code := Run(resumeArgs(), &out, &errb); code != 0 || *proved != len(driveStartRetries) || w.terminalCalls != 0 {
			t.Fatalf("exit %d, proved %d, %d runs:\n%s%s", code, *proved, w.terminalCalls, out.String(), errb.String())
		}
		if got := w.body(t, w.target); got != "old-bytes" {
			t.Fatalf("it goes back: the installed file holds %q", got)
		}
	})
}

// TestAWallThatEndsTheLoopIsNoFailedStart: a comparison that catches a change must end the
// run: another attempt would pin the change, and going back would act on a file that may
// not be what this run moved onto. The exit names the way back.
func TestAWallThatEndsTheLoopIsNoFailedStart(t *testing.T) {
	w := goBackWorld(t)
	failingStarts(0)
	calls := terminalScript(t, func(loop.TerminalOptions) (loop.TerminalResult, error) {
		return loop.TerminalResult{Ended: loop.TerminalWall, Why: "abcd's binary changed while the drive ran"}, nil
	})
	text := refusedDrive(t, resumeArgs()...)
	w.restoredNothing(t, "new-bytes")
	if *calls != 1 || len(w.startWaits) != 0 {
		t.Fatalf("%d runs, waits %v:\n%s", *calls, w.startWaits, text)
	}
	if !strings.Contains(text, ".abcd.previous") {
		t.Fatalf("the exit names the way back:\n%s", text)
	}
}

// TestAHarnessThatMovedLeavesTheReleaseForTheNextCheck: a proof on the old file says nothing
// of the new one a re-executed start would pin, so the release waits for the next wall check
// to prove and pin the new one.
func TestAHarnessThatMovedLeavesTheReleaseForTheNextCheck(t *testing.T) {
	t.Run("before the proof", func(t *testing.T) {
		w := newUpdateWorld(t)
		u := w.step(t)
		moved := true
		driveSelfUpdateSteps.harnessMoved = func(runner.Profile) bool { return moved }
		driveSelfUpdateSteps.harnessOwed = func(runner.Profile) bool { return moved }
		u.atBoundary(context.Background())
		w.nothingSwapped(t)
		if w.staged != 0 || w.rehearsed != 0 || w.proved != 0 || len(w.reexecs) != 0 || u.tried[updTo] {
			t.Fatalf("staged %d, rehearsed %d, proved %d, started %v, tried %v", w.staged, w.rehearsed, w.proved, w.reexecs, u.tried)
		}
		if len(w.lines) != 1 || !strings.Contains(w.lines[0], "harness") || !strings.Contains(w.lines[0], "next check") {
			t.Fatalf("the lines: %q", w.lines)
		}
		moved = false
		w.now = w.now.Add(driveCheckEvery)
		u.atBoundary(context.Background())
		if w.proved != 1 || len(w.reexecs) != 1 {
			t.Fatalf("the next check takes it: proved %d, started %v", w.proved, w.reexecs)
		}
	})
	t.Run("during the proof", func(t *testing.T) {
		w := newUpdateWorld(t)
		u := w.step(t)
		moved := false
		driveSelfUpdateSteps.harnessMoved = func(runner.Profile) bool { return moved }
		driveSelfUpdateSteps.prove = func(context.Context, runner.Profile, string) error {
			moved = true
			return nil
		}
		u.atBoundary(context.Background())
		w.nothingSwapped(t)
		if len(w.reexecs) != 0 || u.tried[updTo] {
			t.Fatalf("started %v, tried %v", w.reexecs, u.tried)
		}
	})
}

// TestADriveHoldsItsCheckForOneIntervalAtMost: the loop's ceiling is the check's interval.
func TestADriveHoldsItsCheckForOneIntervalAtMost(t *testing.T) {
	w := newUpdateWorld(t)
	w.passWalls(t)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 || w.terminal.BoundaryCeiling != driveCheckEvery {
		t.Fatalf("exit %d, ceiling %v:\n%s%s", code, w.terminal.BoundaryCeiling, out.String(), errb.String())
	}
}

// signalThisProcess sends a real signal and waits for done to say it arrived.
func signalThisProcess(t *testing.T, done func() bool) {
	t.Helper()
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Error(err)
	}
	for range 400 {
		if done() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Error("the signal was not seen")
}

// TestAStopInsideAFailingTryEndsTheDrive: a stop is what was asked for, however the try it
// came during ended. No further try runs, no agent launches, and the kept copy does not
// start.
func TestAStopInsideAFailingTryEndsTheDrive(t *testing.T) {
	for name, args := range map[string][]string{
		"a new release": resumeArgs(),
		"a kept copy":   {"drain", "--drive", "--yes", "--update-refused", updTo},
	} {
		t.Run(name, func(t *testing.T) {
			w := goBackWorld(t)
			if name == "a kept copy" {
				w.useRelease(updFrom)
			}
			w.recordTerminal(t)
			var errb syncBuffer
			calls := 0
			driveWalls.static = func(context.Context, runner.Profile) ([]string, error) {
				calls++
				signalThisProcess(t, func() bool { return strings.Contains(errb.String(), "stopping:") })
				return nil, errors.New("a file was held for a moment")
			}
			proved := 0
			driveWalls.live = func(context.Context, runner.Profile) error {
				proved++
				return nil
			}
			var out strings.Builder
			if code := Run(args, &out, &errb); code != driveKilledExit {
				t.Fatalf("exit %d, want %d:\n%s%s", code, driveKilledExit, out.String(), errb.String())
			}
			w.restoredNothing(t, "new-bytes")
			if calls != 1 || proved != 0 || w.terminalCalls != 0 || len(w.startWaits) != 0 {
				t.Fatalf("%d tries, %d proofs, %d runs, waits %v", calls, proved, w.terminalCalls, w.startWaits)
			}
		})
	}
}

// TestAStopThatLeftTheTryPassingLaunchesNothing: the static checks passed, and the live
// launch a retried start proves itself with is an agent.
func TestAStopThatLeftTheTryPassingLaunchesNothing(t *testing.T) {
	w := goBackWorld(t)
	w.recordTerminal(t)
	var errb syncBuffer
	calls := 0
	driveWalls.static = func(context.Context, runner.Profile) ([]string, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("a file was held for a moment")
		}
		signalThisProcess(t, func() bool { return strings.Contains(errb.String(), "stopping:") })
		return nil, nil
	}
	proved := 0
	driveWalls.live = func(context.Context, runner.Profile) error {
		proved++
		return nil
	}
	var out strings.Builder
	if code := Run(resumeArgs(), &out, &errb); code != driveKilledExit {
		t.Fatalf("exit %d, want %d:\n%s%s", code, driveKilledExit, out.String(), errb.String())
	}
	if proved != 0 || w.terminalCalls != 0 {
		t.Fatalf("%d proofs, %d runs", proved, w.terminalCalls)
	}
	w.restoredNothing(t, "new-bytes")
}

// TestAStopBeforeTheWatcherEndsAResume: the signal handling the drive starts comes after a
// window in which only the start's own context hears a signal.
func TestAStopBeforeTheWatcherEndsAResume(t *testing.T) {
	w := goBackWorld(t)
	w.recordTerminal(t)
	calls := failingStarts(0)
	at := driveAtYes
	t.Cleanup(func() { driveAtYes = at })
	driveAtYes = func(ctx context.Context) {
		signalThisProcess(t, func() bool { return ctx.Err() != nil })
	}
	var out, errb strings.Builder
	if code := Run(resumeArgs(), &out, &errb); code != driveKilledExit {
		t.Fatalf("exit %d, want %d:\n%s%s", code, driveKilledExit, out.String(), errb.String())
	}
	w.restoredNothing(t, "new-bytes")
	if *calls != 0 || w.terminalCalls != 0 {
		t.Fatalf("%d starts, %d runs", *calls, w.terminalCalls)
	}
}

// offlineKeptCopy has the kept copy's verification come to no answer n times, then to what
// it was, and counts the asking.
func offlineKeptCopy(n int) *int {
	calls, verify := 0, driveSelfUpdateSteps.verifyKept
	driveSelfUpdateSteps.verifyKept = func(keep, tag string) ([]byte, error) {
		calls++
		if calls <= n {
			return nil, &update.NoVerdictError{Err: errors.New("no route to the attestations API")}
		}
		return verify(keep, tag)
	}
	return &calls
}

// TestTheWayBackDoesNotRestOnOneRequest: a verification that came to no answer says nothing
// of the kept copy, so it is asked again, one line each time, and the copy then goes back.
func TestTheWayBackDoesNotRestOnOneRequest(t *testing.T) {
	w := goBackWorld(t)
	calls := offlineKeptCopy(2)
	var out, errb strings.Builder
	if code := Run(resumeArgs(), &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	text := out.String() + errb.String()
	if got := w.body(t, w.target); got != "old-bytes" || len(w.reexecs) != 1 || *calls != 3 {
		t.Fatalf("the installed file holds %q, %d re-executions, asked %d times:\n%s", got, len(w.reexecs), *calls, text)
	}
	if got := w.startWaits[len(driveStartRetries):]; !slices.Equal(got, driveVerifyRetries[:2]) {
		t.Fatalf("waited %v", got)
	}
	if got := strings.Count(text, "no route to the attestations API"); got != 2 {
		t.Fatalf("%d lines say why it asks again:\n%s", got, text)
	}
}

// TestAVerdictOnTheKeptCopyIsNotAskedAgain: a copy found not to be the release is refused
// at once, whatever a later answer might say.
func TestAVerdictOnTheKeptCopyIsNotAskedAgain(t *testing.T) {
	w := goBackWorld(t)
	calls := 0
	driveSelfUpdateSteps.verifyKept = func(string, string) ([]byte, error) {
		calls++
		return nil, errors.New("its build metadata names no release")
	}
	refusedDrive(t, resumeArgs()...)
	w.restoredNothing(t, "new-bytes")
	if calls != 1 || len(w.startWaits) != len(driveStartRetries) {
		t.Fatalf("asked %d times, %v", calls, w.startWaits)
	}
}

// TestAskingAgainOnTheWayBackHasAnEnd: a verification that never answers ends the run with
// nothing renamed, within a bounded time, and the checkout's lock is free between tries.
func TestAskingAgainOnTheWayBackHasAnEnd(t *testing.T) {
	w := goBackWorld(t)
	calls := offlineKeptCopy(1 << 10)
	wait, locked := driveStartWait, 0
	driveStartWait = func(ctx context.Context, d time.Duration) error {
		if err := fsutil.WithFileLock(filepath.Join(w.repo, filepath.FromSlash(driveLockRel)), time.Second, func() error {
			locked++
			return nil
		}); err != nil {
			t.Errorf("the lock during a wait: %v", err)
		}
		return wait(ctx, d)
	}
	text := refusedDrive(t, resumeArgs()...)
	w.restoredNothing(t, "new-bytes")
	if *calls != 1+len(driveVerifyRetries) || locked != len(driveStartRetries)+len(driveVerifyRetries) {
		t.Fatalf("asked %d times, the lock free at %d waits", *calls, locked)
	}
	var total time.Duration
	for _, d := range driveVerifyRetries {
		total += d
	}
	if total < time.Minute || total > 5*time.Minute {
		t.Fatalf("asks over %v", total)
	}
	if !strings.Contains(text, "update "+updFrom) {
		t.Fatalf("the exit names the way back: %s", text)
	}
}

// TestAStopWhileTheWayBackAsksAgainRestoresNothing: the wait is the real one, and so is the
// signal.
func TestAStopWhileTheWayBackAsksAgainRestoresNothing(t *testing.T) {
	w := goBackWorld(t)
	calls := offlineKeptCopy(1 << 10)
	waited := 0
	driveStartWait = func(ctx context.Context, _ time.Duration) error {
		if waited++; waited <= len(driveStartRetries) {
			return nil
		}
		if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
			t.Error(err)
		}
		return waitForStart(ctx, time.Hour)
	}
	var out, errb strings.Builder
	if code := Run(resumeArgs(), &out, &errb); code != driveKilledExit {
		t.Fatalf("exit %d, want %d:\n%s%s", code, driveKilledExit, out.String(), errb.String())
	}
	w.restoredNothing(t, "new-bytes")
	if *calls != 1 {
		t.Fatalf("asked %d times", *calls)
	}
}

// aRefusedMove is a first move the loop refused, the way a real one is.
func aRefusedMove(loop.TerminalOptions) (loop.TerminalResult, error) {
	return loop.TerminalResult{Ended: loop.TerminalRefused}, &loop.Refusal{Stage: "implement", Reason: "the lane could not be opened"}
}

// TestARefusedFirstMoveThatWillBeTriedAgainDoesNotSayItEnded: the attempt is not the
// drive's end, so the end is not said, and the next attempt's line says what comes.
func TestARefusedFirstMoveThatWillBeTriedAgainDoesNotSayItEnded(t *testing.T) {
	w := goBackWorld(t)
	failingStarts(0)
	calls := terminalScript(t, aRefusedMove, func(loop.TerminalOptions) (loop.TerminalResult, error) {
		return loop.TerminalResult{Ended: loop.DrainStoppedCap}, nil
	})
	var out, errb strings.Builder
	if code := Run(resumeArgs(), &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	text := out.String() + errb.String()
	if *calls != 2 || w.body(t, w.target) != "new-bytes" {
		t.Fatalf("%d runs:\n%s", *calls, text)
	}
	if strings.Contains(text, "a move was refused") || !strings.Contains(text, "trying again") {
		t.Fatalf("it says it ended, or not that it tries again:\n%s", text)
	}
}

// TestARefusedFirstMoveIsOneDocument: a reader of the JSON gets one answer, whichever way
// the attempts end, and the end is said once.
func TestARefusedFirstMoveIsOneDocument(t *testing.T) {
	end := func(t *testing.T, args []string) (docs, ends int) {
		t.Helper()
		failingStarts(0)
		terminalScript(t, aRefusedMove)
		var out, errb strings.Builder
		if code := Run(args, &out, &errb); code != 2 {
			t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
		}
		return strings.Count(out.String(), `"refusal"`), strings.Count(errb.String(), "ended: a move was refused")
	}
	t.Run("on the kept copy", func(t *testing.T) {
		w := newUpdateWorld(t)
		w.passWalls(t)
		w.liveDrain(t, 3)
		if docs, ends := end(t, []string{"drain", "--drive", "--yes", driveUpdateRefusedFlag, updTo, "--json"}); docs != 1 || ends != 1 {
			t.Fatalf("%d documents, %d ends", docs, ends)
		}
	})
	t.Run("when the way back cannot be taken", func(t *testing.T) {
		w := goBackWorld(t)
		w.aForeignKeptCopy(t)
		if docs, ends := end(t, append(resumeArgs(), "--json")); docs != 1 || ends != 1 {
			t.Fatalf("%d documents, %d ends", docs, ends)
		}
		w.restoredNothing(t, "new-bytes")
	})
}

// TestARefusedFirstMoveGoingBackLeavesTheAnswerToTheKeptCopy: the kept copy goes on and
// answers for the run, so nothing is said for the release it replaced.
func TestARefusedFirstMoveGoingBackLeavesTheAnswerToTheKeptCopy(t *testing.T) {
	w := goBackWorld(t)
	failingStarts(0)
	terminalScript(t, aRefusedMove)
	var out, errb strings.Builder
	if code := Run(append(resumeArgs(), "--json"), &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if got := w.body(t, w.target); got != "old-bytes" || len(w.reexecs) != 1 || strings.Contains(out.String(), `"refusal"`) {
		t.Fatalf("the installed file holds %q, %d re-executions:\n%s", got, len(w.reexecs), out.String())
	}
}

// aKeptCopyWorld is a kept copy's start with the walls holding.
func aKeptCopyWorld(t *testing.T) *updateWorld {
	t.Helper()
	w := newUpdateWorld(t)
	w.passWalls(t)
	w.liveDrain(t, 3)
	return w
}

var keptCopyArgs = []string{"drain", "--drive", "--yes", driveUpdateRefusedFlag, updTo}

// TestALimitAtAResumesProofIsWaitedOut: nobody is at the terminal to run it again, so a
// limit waits the way one in the drive's loop does, with no bound given.
func TestALimitAtAResumesProofIsWaitedOut(t *testing.T) {
	resets := func(now time.Time) time.Time { return now.Add(3 * time.Hour) }
	t.Run("a kept copy", func(t *testing.T) {
		w := aKeptCopyWorld(t)
		proofs, _ := limitedProof(t, 1, resets)
		var out, errb strings.Builder
		if code := Run(keptCopyArgs, &out, &errb); code != 0 {
			t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
		}
		if *proofs != 2 || w.terminalCalls != 1 {
			t.Fatalf("%d proofs, %d runs", *proofs, w.terminalCalls)
		}
		if !strings.Contains(out.String()+errb.String(), "rate limit") {
			t.Fatalf("the wait names the limit:\n%s%s", out.String(), errb.String())
		}
	})
	t.Run("an attempt after a failed one", func(t *testing.T) {
		w := goBackWorld(t)
		w.recordTerminal(t)
		failingStarts(1)
		proofs, _ := limitedProof(t, 1, resets)
		var out, errb strings.Builder
		if code := Run(resumeArgs(), &out, &errb); code != 0 {
			t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
		}
		if *proofs != 2 || w.terminalCalls != 1 || w.body(t, w.target) != "new-bytes" {
			t.Fatalf("%d proofs, %d runs", *proofs, w.terminalCalls)
		}
	})
}

// TestALimitThatOutlastsAResumesWaitStopsItShort: the wait has the loop's eight hours, and
// ending on it is no clean end and no failed start.
func TestALimitThatOutlastsAResumesWaitStopsItShort(t *testing.T) {
	w := aKeptCopyWorld(t)
	proofs, _ := limitedProof(t, 1<<20, func(time.Time) time.Time { return time.Time{} })
	var out, errb strings.Builder
	if code := Run(keptCopyArgs, &out, &errb); code != 3 {
		t.Fatalf("exit %d, want 3:\n%s%s", code, out.String(), errb.String())
	}
	if *proofs < 2 || w.terminalCalls != 0 || len(w.startWaits) != 0 {
		t.Fatalf("%d proofs, %d runs, waits %v", *proofs, w.terminalCalls, w.startWaits)
	}
}

// TestAStopDuringAResumesProofWaitEndsIt: it ends as a stop during any other part of the
// start does, with nothing restored.
func TestAStopDuringAResumesProofWaitEndsIt(t *testing.T) {
	w := aKeptCopyWorld(t)
	limitedProof(t, 1<<20, func(now time.Time) time.Time { return now.Add(time.Hour) })
	stopped := make(chan struct{})
	driveProofWait.sleep = func(ctx context.Context, _ time.Duration) {
		select {
		case <-stopped:
		default:
			close(stopped)
			if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
				t.Error(err)
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
	var out, errb strings.Builder
	if code := Run(keptCopyArgs, &out, &errb); code != driveKilledExit {
		t.Fatalf("exit %d, want %d:\n%s%s", code, driveKilledExit, out.String(), errb.String())
	}
	if w.terminalCalls != 0 {
		t.Fatalf("%d runs", w.terminalCalls)
	}
}

// TestATeamThatDoesNotHoldEndsARetriedStart: trying again cannot change a signature, and a
// way back is no answer to what stands at the harness's name, so the run ends naming it.
func TestATeamThatDoesNotHoldEndsARetriedStart(t *testing.T) {
	w := goBackWorld(t)
	calls := 0
	driveWalls.static = func(context.Context, runner.Profile) ([]string, error) {
		if calls++; calls == 1 {
			return nil, errors.New("a file was held for a moment")
		}
		return nil, &runner.HarnessTeamError{WallError: &runner.WallError{Wall: runner.WallSettingsFile, Detail: "not verified as signed by the signing team"}}
	}
	text := refusedDrive(t, resumeArgs()...)
	w.restoredNothing(t, "new-bytes")
	if calls != 2 || len(w.startWaits) != 1 {
		t.Fatalf("%d starts, waits %v:\n%s", calls, w.startWaits, text)
	}
	if !strings.Contains(text, "signing team") || !strings.Contains(text, ".abcd.previous") {
		t.Fatalf("the exit says why and the way back:\n%s", text)
	}
}

// TestARetriedStartHoldsWhatTheProcessBeganOn: the likeliest first attempt fails before it
// pins anything, so what is compared is what stood as the process began, with nothing of
// what the second attempt would write in as its baseline.
func TestARetriedStartHoldsWhatTheProcessBeganOn(t *testing.T) {
	type world struct {
		*updateWorld
		tool string
	}
	changes := map[string]func(*testing.T, *world){
		"a machine declaration": func(t *testing.T, w *world) {
			t.Helper()
			path := abcdhome.Path(w.home, "config.json")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"a program": func(t *testing.T, w *world) {
			t.Helper()
			if err := os.WriteFile(w.tool, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
				t.Fatal(err)
			}
		},
	}
	faults := map[string]func(*testing.T, *world) (clear func(), statics *int){
		"a first attempt that failed before it pinned": func(t *testing.T, w *world) (func(), *int) {
			t.Helper()
			statics := failingStarts(0)
			state := filepath.Join(w.repo, filepath.FromSlash(loop.DrainStateRel))
			raw, err := os.ReadFile(state)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(state, []byte("{not json"), 0o600); err != nil {
				t.Fatal(err)
			}
			return func() {
				if err := os.WriteFile(state, raw, 0o600); err != nil {
					t.Error(err)
				}
			}, statics
		},
		"a first attempt that reached the settings": func(t *testing.T, _ *world) (func(), *int) {
			t.Helper()
			return func() {}, failingStarts(1)
		},
	}
	for faultName, fault := range faults {
		for changeName, change := range changes {
			t.Run(faultName+" then "+changeName, func(t *testing.T) {
				w := &world{updateWorld: goBackWorld(t)}
				tools := t.TempDir()
				w.tool = filepath.Join(tools, "make")
				if err := os.WriteFile(w.tool, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
					t.Fatal(err)
				}
				t.Setenv("PATH", tools+string(os.PathListSeparator)+os.Getenv("PATH"))
				w.recordTerminal(t)
				clear, statics := fault(t, w)
				before := *statics
				driveStartWait = func(context.Context, time.Duration) error {
					before = *statics
					change(t, w)
					clear()
					return nil
				}
				text := refusedDrive(t, resumeArgs()...)
				w.restoredNothing(t, "new-bytes")
				if w.terminalCalls != 0 || *statics != before {
					t.Fatalf("%d runs, the settings written again:\n%s", w.terminalCalls, text)
				}
				if !strings.Contains(text, "changed") || !strings.Contains(text, ".abcd.previous") {
					t.Fatalf("the exit says why and the way back:\n%s", text)
				}
			})
		}
	}
}

// useTeams has the drive read each file's signing team from signed.
func useTeams(t *testing.T, signed map[string]string) {
	t.Helper()
	was := runner.Signing
	t.Cleanup(func() { runner.Signing = was })
	runner.Signing = runner.HarnessSigning{
		Team: func(path string) string { return signed[path] },
		Verify: func(path, team string) error {
			if team == "" || signed[path] != team {
				return errors.New("not that team's")
			}
			return nil
		},
	}
}

// onlyTheStandInHarness has PATH reach the world's stand-in harness and git and nothing else
// a person's machine keeps, so a harness that goes is not found elsewhere and a stage's files
// ask no Go toolchain anything under the test's home.
func onlyTheStandInHarness(t *testing.T) string {
	t.Helper()
	harness, err := exec.LookPath("claude")
	if err != nil {
		t.Fatal(err)
	}
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	tools := t.TempDir()
	if err := os.Symlink(git, filepath.Join(tools, "git")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", strings.Join([]string{filepath.Dir(harness), tools, "/usr/bin", "/bin"}, string(os.PathListSeparator)))
	if harness, err = filepath.EvalSymlinks(harness); err != nil {
		t.Fatal(err)
	}
	return harness
}

// aPinnedProfile is a run's profile with the walls written and the comparison the real one,
// and the file its harness pinned.
func (w *updateWorld) aPinnedProfile(t *testing.T) (runner.Profile, string) {
	t.Helper()
	harness := onlyTheStandInHarness(t)
	return w.aProfilePinnedUnder(t, map[string]string{harness: "TEAMAAAAAA"}), harness
}

// aProfilePinnedUnder is that profile with the teams signed in force.
func (w *updateWorld) aProfilePinnedUnder(t *testing.T, signed map[string]string) runner.Profile {
	t.Helper()
	w.passWalls(t)
	useTeams(t, signed)
	p, err := runner.DriveProfile(w.home, "walled")
	if err != nil {
		t.Fatal(err)
	}
	p = p.ForCheckout(w.repo)
	bin := runner.SelfBinary()
	if _, err := runner.WriteWallSettings(p, bin); err != nil {
		t.Fatal(err)
	}
	driveWalls.settings = func(p runner.Profile) error { return runner.CheckWallSettings(p, bin) }
	driveSelfUpdateSteps.harnessOwed = runner.HarnessOwed
	driveSelfUpdateSteps.harnessMoved = runner.HarnessMoved
	return p
}

// TestNothingChangedDuringTheCheckIsSwappedPast: a re-executed start pins what stands, so
// what changed between the check's start and the swap would become its baseline. Nothing is
// swapped, and the loop's own comparison ends the run.
func TestNothingChangedDuringTheCheckIsSwappedPast(t *testing.T) {
	for name, change := range map[string]func(*testing.T, *updateWorld, string){
		"new bytes at the harness's own path": func(t *testing.T, _ *updateWorld, harness string) {
			t.Helper()
			if err := os.WriteFile(harness, []byte("#!/bin/sh\nexit 2\n"), 0o755); err != nil {
				t.Fatal(err)
			}
		},
		"a machine declaration": func(t *testing.T, w *updateWorld, _ string) {
			t.Helper()
			path := abcdhome.Path(w.home, "config.json")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			w := newUpdateWorld(t)
			p, harness := w.aPinnedProfile(t)
			driveSelfUpdateSteps.prove = func(context.Context, runner.Profile, string) error {
				w.proved++
				change(t, w, harness)
				return nil
			}
			commits, commit := 0, driveSelfUpdateSteps.commit
			driveSelfUpdateSteps.commit = func(target string, st update.Staged, keep string) error {
				commits++
				return commit(target, st, keep)
			}
			u := w.stepOn(p, driveFlags{self: driveSelfUpdateSteps.target()})
			u.atBoundary(context.Background())
			if w.proved != 1 || commits != 0 {
				t.Fatalf("proved %d, committed %d", w.proved, commits)
			}
			w.nothingSwapped(t)
			if u.tried[updTo] {
				t.Fatal("the release is not what failed")
			}
		})
	}
}

// TestAPassingProofIsNoChange: the comparison before the swap holds after a proof that
// changed nothing.
func TestAPassingProofIsNoChange(t *testing.T) {
	w := newUpdateWorld(t)
	p, _ := w.aPinnedProfile(t)
	w.stepOn(p, driveFlags{self: driveSelfUpdateSteps.target()}).atBoundary(context.Background())
	if got := w.body(t, w.target); got != "new-bytes" || len(w.reexecs) != 1 {
		t.Fatalf("the installed file holds %q, %d re-executions", got, len(w.reexecs))
	}
}

// TestARetriedStartIsHeldToTheTeamTheProcessBeganOn: a first attempt that pinned nothing
// leaves the team read as the process began to hold what stands after the wait.
func TestARetriedStartIsHeldToTheTeamTheProcessBeganOn(t *testing.T) {
	type world struct {
		*updateWorld
		harness string
		signed  map[string]string
		writes  int
	}
	setUp := func(t *testing.T) *world {
		t.Helper()
		w := &world{updateWorld: goBackWorld(t)}
		w.recordTerminal(t)
		harness := onlyTheStandInHarness(t)
		w.harness, w.signed = harness, map[string]string{harness: "TEAMAAAAAA"}
		useTeams(t, w.signed)
		driveWalls.static = func(_ context.Context, p runner.Profile) ([]string, error) {
			w.writes++
			_, err := runner.WriteWallSettings(p, runner.SelfBinary())
			return nil, err
		}
		return w
	}
	badState := func(t *testing.T, w *world) func() {
		t.Helper()
		state := filepath.Join(w.repo, filepath.FromSlash(loop.DrainStateRel))
		raw, err := os.ReadFile(state)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(state, []byte("{not json"), 0o600); err != nil {
			t.Fatal(err)
		}
		return func() {
			if err := os.WriteFile(state, raw, 0o600); err != nil {
				t.Error(err)
			}
		}
	}
	// standAt puts new bytes at the harness's name, of team, and clears the fault, once.
	standAt := func(t *testing.T, w *world, team string, clear func()) {
		t.Helper()
		did := false
		driveStartWait = func(_ context.Context, d time.Duration) error {
			w.startWaits = append(w.startWaits, d)
			if did {
				return nil
			}
			did = true
			if err := os.WriteFile(w.harness, []byte("#!/bin/sh\nexit 3\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			if team == "" {
				delete(w.signed, w.harness)
			} else {
				w.signed[w.harness] = team
			}
			clear()
			return nil
		}
	}
	t.Run("another team", func(t *testing.T) {
		w := setUp(t)
		standAt(t, w, "TEAMBBBBBB", badState(t, w))
		text := refusedDrive(t, resumeArgs()...)
		w.restoredNothing(t, "new-bytes")
		if w.writes != 1 || w.terminalCalls != 0 || len(w.startWaits) != 1 || !strings.Contains(text, "is not verified as signed") || !strings.Contains(text, ".abcd.previous") {
			t.Fatalf("%d starts, %d runs, waits %v:\n%s", w.writes, w.terminalCalls, w.startWaits, text)
		}
	})
	t.Run("no identity to read", func(t *testing.T) {
		w := setUp(t)
		standAt(t, w, "", badState(t, w))
		text := refusedDrive(t, resumeArgs()...)
		w.restoredNothing(t, "new-bytes")
		if w.writes != 3 || w.terminalCalls != 0 || len(w.startWaits) != len(driveStartRetries) || !strings.Contains(text, "try 2 of 3") {
			t.Fatalf("%d starts, %d runs, waits %v:\n%s", w.writes, w.terminalCalls, w.startWaits, text)
		}
	})
	t.Run("gone as it began and back of another team", func(t *testing.T) {
		w := setUp(t)
		if err := os.Remove(w.harness); err != nil {
			t.Fatal(err)
		}
		standAt(t, w, "TEAMBBBBBB", badState(t, w))
		text := refusedDrive(t, resumeArgs()...)
		w.restoredNothing(t, "new-bytes")
		if w.terminalCalls != 0 || !strings.Contains(text, "signing team") || !strings.Contains(text, ".abcd.previous") {
			t.Fatalf("%d runs:\n%s", w.terminalCalls, text)
		}
	})
	t.Run("gone as it began and back with no identity", func(t *testing.T) {
		w := setUp(t)
		if err := os.Remove(w.harness); err != nil {
			t.Fatal(err)
		}
		standAt(t, w, "", badState(t, w))
		text := refusedDrive(t, resumeArgs()...)
		w.restoredNothing(t, "new-bytes")
		if w.terminalCalls != 0 || !strings.Contains(text, "signing team") {
			t.Fatalf("%d runs:\n%s", w.terminalCalls, text)
		}
	})
}

// TestALinkMovedToAFileTheRunCannotAdmitOnlyHoldsTheRelease: while the file the run proved
// stands, a re-proof could only stop the run for a release that happens to be waiting, so
// the release waits, one line says so, and each check looks again.
func TestALinkMovedToAFileTheRunCannotAdmitOnlyHoldsTheRelease(t *testing.T) {
	for name, team := range map[string]string{"no team held": "", "another team": "TEAMAAAAAA"} {
		t.Run(name, func(t *testing.T) {
			w := newUpdateWorld(t)
			link := onlyTheStandInHarness(t)
			dir, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			first, second := filepath.Join(dir, "first"), filepath.Join(dir, "second")
			for _, path := range []string{first, second} {
				if err := os.WriteFile(path, []byte("#!/bin/sh\n# "+path+"\nexit 1\n"), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			repoint := func(to string) {
				t.Helper()
				if err := os.Remove(link); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(to, link); err != nil {
					t.Fatal(err)
				}
			}
			repoint(first)
			signed := map[string]string{second: "TEAMBBBBBB"}
			if team != "" {
				signed[first] = team
			}
			p := w.aProfilePinnedUnder(t, signed)
			repoint(second)
			u := w.stepOn(p, driveFlags{self: driveSelfUpdateSteps.target()})
			u.atBoundary(context.Background())
			w.now = w.now.Add(driveCheckEvery)
			u.atBoundary(context.Background())
			w.nothingSwapped(t)
			if w.asked != 2 || w.staged != 0 || w.proved != 0 || u.tried[updTo] {
				t.Fatalf("asked %d, staged %d, proved %d, tried %v", w.asked, w.staged, w.proved, u.tried)
			}
			if len(w.lines) != 1 || !strings.Contains(w.lines[0], "carries on") {
				t.Fatalf("the lines: %q", w.lines)
			}
			if runner.HarnessOwed(p) {
				t.Fatal("a file it could not admit is owed no proof")
			}
			if err := driveWalls.settings(p); err != nil {
				t.Fatalf("the run goes on: %v", err)
			}
		})
	}
}

// TestAReleaseCheckIsOnlyHeldForWhereOneCanCome: the loop's word about a release waiting is
// false on a run that takes none.
func TestAReleaseCheckIsOnlyHeldForWhereOneCanCome(t *testing.T) {
	t.Run("the switch", func(t *testing.T) {
		w := newUpdateWorld(t)
		w.passWalls(t)
		var out, errb strings.Builder
		if code := Run([]string{"drain", "--drive", "--yes", "--no-self-update"}, &out, &errb); code != 0 {
			t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
		}
		if w.terminal.BoundaryCeiling != 0 || w.terminal.AtBoundary != nil {
			t.Fatalf("ceiling %v, a boundary set: %v", w.terminal.BoundaryCeiling, w.terminal.AtBoundary != nil)
		}
	})
	t.Run("a build that never self-updates", func(t *testing.T) {
		w := newUpdateWorld(t)
		w.passWalls(t)
		driveSelfUpdateSteps.target = func() update.SelfTarget {
			return update.SelfTarget{Path: w.target, Version: updFrom, Refusal: &update.Refusal{Shape: update.ShapeSourceBuild, Detail: "a source build"}}
		}
		var out, errb strings.Builder
		if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
			t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
		}
		if w.terminal.BoundaryCeiling != 0 || w.terminal.AtBoundary != nil {
			t.Fatalf("ceiling %v, a boundary set: %v", w.terminal.BoundaryCeiling, w.terminal.AtBoundary != nil)
		}
	})
	t.Run("a drive that can take one", func(t *testing.T) {
		w := newUpdateWorld(t)
		w.passWalls(t)
		var out, errb strings.Builder
		if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
			t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
		}
		if w.terminal.BoundaryCeiling == 0 || w.terminal.AtBoundary == nil {
			t.Fatal("it holds its check")
		}
	})
}

// TestAPreflightStartsTheNameReadAtTheStart: after a refused re-execution the name a process
// reports can be a kept copy that has since gone, and every later lane would fail on it.
func TestAPreflightStartsTheNameReadAtTheStart(t *testing.T) {
	was := driveSelfName
	t.Cleanup(func() { driveSelfName = was })
	driveSelfName = func() string { return "/installed/abcd" }
	program, _, err := drivePreflightProgram()
	if err != nil || program != "/installed/abcd" {
		t.Fatalf("%q, %v", program, err)
	}
	driveSelfName = func() string { return "" }
	if _, _, err := drivePreflightProgram(); err == nil {
		t.Fatal("a name that cannot be read is an error")
	}
}
