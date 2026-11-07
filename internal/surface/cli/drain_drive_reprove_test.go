package cli

// drain_drive_reprove_test.go holds the drive's check before a stage on a harness that
// went: a new one proved, one line, and the run goes on; one that fails ends it.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// reproveDrive runs a drive whose walls hold on one stand-in harness, and in the loop's
// place replaces the harness with stream and calls the check before a stage twice.
func reproveDrive(t *testing.T, newStream string) (first, second error, out string) {
	t.Helper()
	standInSigned(t)
	return reproveDriveAsSigned(t, newStream)
}

// reproveDriveAsSigned is reproveDrive under whatever runner.Signing the test set.
func reproveDriveAsSigned(t *testing.T, newStream string) (first, second error, out string) {
	t.Helper()
	return reproveDriveRun(t, newStream, nil, nil)
}

// outputWatch is a writer that shows each write to a watcher as it comes.
type outputWatch struct {
	strings.Builder
	watch func(string)
}

func (o *outputWatch) Write(b []byte) (int, error) {
	if o.watch != nil {
		o.watch(string(b))
	}
	return o.Builder.Write(b)
}

// reproveDriveRun drives to the first two checks on the new harness: during runs just
// before them, with the new file in place, and watch sees the run's output.
func reproveDriveRun(t *testing.T, newStream string, during func(), watch func(string)) (first, second error, out string) {
	t.Helper()
	newDriveWorld(t)
	clock := time.Now().UTC().Truncate(time.Second)
	wait := driveProofWait
	t.Cleanup(func() { driveProofWait = wait })
	driveProofWait.now = func() time.Time { return clock }
	driveProofWait.sleep = func(_ context.Context, d time.Duration) { clock = clock.Add(d) }
	standInHarness(t, true, wallInit+wallRefused+wallResult)
	path := os.Getenv("PATH")
	was := filepath.Join(strings.SplitN(path, string(os.PathListSeparator), 2)[0], "claude")
	term := driveTerminal
	t.Cleanup(func() { driveTerminal = term })
	driveTerminal = func(_ context.Context, _ string, o loop.TerminalOptions) (loop.TerminalResult, error) {
		if err := os.Remove(was); err != nil {
			t.Fatal(err)
		}
		standInHarness(t, true, newStream)
		if during != nil {
			during()
		}
		first, second = o.BeforeStage(), o.BeforeStage()
		return loop.TerminalResult{Ended: loop.DrainStoppedEmpty}, nil
	}
	stdout := &outputWatch{watch: watch}
	var stderr strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes"}, stdout, &stderr); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, stdout.String(), stderr.String())
	}
	return first, second, stdout.String() + stderr.String()
}

// standInSigned has runner.Signing say every file is the test team's.
func standInSigned(t *testing.T) {
	t.Helper()
	signing := runner.Signing
	t.Cleanup(func() { runner.Signing = signing })
	runner.Signing = runner.HarnessSigning{
		Team:   func(string) string { return "TEAMAAAAAA" },
		Verify: func(string, string) error { return nil },
	}
}

// TestADriveProvesTheNewHarnessOnceAndSaysSo: one line naming both versions, and the
// check holds on the new file with no further launch.
func TestADriveProvesTheNewHarnessOnceAndSaysSo(t *testing.T) {
	first, second, out := reproveDrive(t, strings.ReplaceAll(wallInit, "2.1.319", "2.1.320")+wallRefused+wallResult)
	if first != nil || second != nil {
		t.Fatalf("the check holds: %v, %v", first, second)
	}
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, "2.1.320") {
			lines = append(lines, l)
		}
	}
	if len(lines) != 1 || !strings.Contains(lines[0], "2.1.319") {
		t.Fatalf("one line naming both versions:\n%s", out)
	}
}

// TestADriveEndsOnANewHarnessThatFails: the check names the wall and the version.
func TestADriveEndsOnANewHarnessThatFails(t *testing.T) {
	first, second, _ := reproveDrive(t, strings.ReplaceAll(wallInit, "2.1.319", "2.1.320")+wallResult)
	var we *runner.WallError
	if !errors.As(first, &we) || we.Wall != runner.WallGuard || !strings.Contains(first.Error(), "2.1.320") {
		t.Fatalf("the refusal names the check and the version: %v", first)
	}
	if second == nil {
		t.Fatal("the next check does not hold either")
	}
}

// TestADriveStopsOnANewHarnessItCannotVerify: one line says why and what to do, and the
// new file is not run: its stream would pass, so a launch would hold the check.
func TestADriveStopsOnANewHarnessItCannotVerify(t *testing.T) {
	signing := runner.Signing
	t.Cleanup(func() { runner.Signing = signing })
	runner.Signing = runner.HarnessSigning{
		Team:   func(string) string { return "" },
		Verify: func(string, string) error { return errors.New("no") },
	}
	first, second, out := reproveDriveAsSigned(t, strings.ReplaceAll(wallInit, "2.1.319", "2.1.320")+wallRefused+wallResult)
	var we *runner.WallError
	if !errors.As(first, &we) || !strings.Contains(first.Error(), "signing team") || !strings.Contains(first.Error(), "start the drive again") {
		t.Fatalf("the stop says why and what to do: %v", first)
	}
	if second == nil || strings.Contains(out, "2.1.320") {
		t.Fatalf("the next check does not hold, and no line names a version it never asked: %v\n%s", second, out)
	}
}

// noIdentityOnTheNewFile has every file the test team's until the new one is in place, then
// that file readable only once ready says so.
func noIdentityOnTheNewFile(t *testing.T, ready func() bool) func() {
	t.Helper()
	standInSigned(t)
	return func() {
		runner.Signing = runner.HarnessSigning{
			Team: func(string) string {
				if ready() {
					return "TEAMAAAAAA"
				}
				return ""
			},
			Verify: func(string, string) error {
				if ready() {
					return nil
				}
				return errors.New("no")
			},
		}
	}
}

// TestADriveWaitsForANewHarnessWithNoIdentityYet: each wait says which, the file read later
// passes, and the run goes on.
func TestADriveWaitsForANewHarnessWithNoIdentityYet(t *testing.T) {
	readable := false
	during := noIdentityOnTheNewFile(t, func() bool { return readable })
	first, second, out := reproveDriveRun(t, strings.ReplaceAll(wallInit, "2.1.319", "2.1.320")+wallRefused+wallResult, func() {
		during()
		sleep := driveProofWait.sleep
		driveProofWait.sleep = func(ctx context.Context, d time.Duration) {
			readable = true
			sleep(ctx, d)
		}
	}, nil)
	if first != nil || second != nil {
		t.Fatalf("the run goes on: %v, %v\n%s", first, second, out)
	}
	if strings.Count(out, "no signing identity") != 1 || !strings.Contains(out, "waiting 3m0s") || !strings.Contains(out, "from 2.1.319 to 2.1.320") {
		t.Fatalf("one wait saying which, then the update's line:\n%s", out)
	}
}

// TestADriveStopsOnceANewHarnessHasSpentItsTries: the wait lines say which, and the stop is
// the one a file that cannot be verified always got.
func TestADriveStopsOnceANewHarnessHasSpentItsTries(t *testing.T) {
	first, _, out := reproveDriveRun(t, strings.ReplaceAll(wallInit, "2.1.319", "2.1.320")+wallRefused+wallResult,
		noIdentityOnTheNewFile(t, func() bool { return false }), nil)
	var we *runner.WallError
	var none *runner.NoVerdictError
	if errors.As(first, &none) || !errors.As(first, &we) || !strings.Contains(first.Error(), "start the drive again") {
		t.Fatalf("the stop says why and what to do: %v", first)
	}
	if strings.Count(out, "no signing identity") != 2 || strings.Contains(out, "2.1.320") {
		t.Fatalf("two waits saying which, and nothing ran:\n%s", out)
	}
}

// TestAWallInTheProofsWaitIsTheReasonTheRunGives: a wall that fails while a re-proof waits
// reaches the run as itself, so its end line says which.
func TestAWallInTheProofsWaitIsTheReasonTheRunGives(t *testing.T) {
	standInSigned(t)
	first, _, _ := reproveDriveRun(t, strings.ReplaceAll(wallInit, "2.1.319", "2.1.320")+wallResult, func() {
		settings := abcdhome.Path(os.Getenv("HOME"), "profiles", "walled", "settings.json")
		sleep := driveProofWait.sleep
		driveProofWait.sleep = func(ctx context.Context, d time.Duration) {
			if err := os.WriteFile(settings, []byte("{}"), 0o600); err != nil {
				t.Error(err)
			}
			sleep(ctx, d)
		}
	}, nil)
	var ended *loop.WallEnded
	if first == nil || errors.As(first, &ended) || !strings.Contains(first.Error(), "settings") {
		t.Fatalf("the wall itself: %v", first)
	}
}

// TestAHarnessGoingAfterItsProofLeavesTheNextCheckToFindIt: a file that goes after the check
// and before the launch is no reason to stop. The launch's own check starts nothing on it,
// the next check finds the proof owed, and a file that stays gone ends at the outage's limit.
func TestAHarnessGoingAfterItsProofLeavesTheNextCheckToFindIt(t *testing.T) {
	standInSigned(t)
	var once sync.Once
	first, second, out := reproveDriveRun(t, strings.ReplaceAll(wallInit, "2.1.319", "2.1.320")+wallRefused+wallResult, nil, func(written string) {
		if strings.Contains(written, "2.1.320") {
			once.Do(func() {
				harness := filepath.Join(strings.SplitN(os.Getenv("PATH"), string(os.PathListSeparator), 2)[0], "claude")
				if err := os.Remove(harness); err != nil {
					t.Error(err)
				}
				t.Setenv("PATH", "/usr/bin:/bin")
			})
		}
	})
	if first != nil {
		t.Fatalf("it goes on:\n%v\n%s", first, out)
	}
	var ended *loop.WallEnded
	if !errors.As(second, &ended) || ended.Ended != loop.TerminalOutage {
		t.Fatalf("a file that stays gone ends at the outage's limit: %v\n%s", second, out)
	}
	if !strings.Contains(out, "no harness") {
		t.Fatalf("the wait says why:\n%s", out)
	}
}

// TestTheConsentSaysAnUpdateIsProvedInTheRun: it adds no prompt, and one line says so.
func TestTheConsentSaysAnUpdateIsProvedInTheRun(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d: %s%s", code, out.String(), errb.String())
	}
	if !strings.Contains(out.String(), "a harness update during the run carries on only where the new file has the same signing team") {
		t.Fatalf("the consent:\n%s", out.String())
	}
}
