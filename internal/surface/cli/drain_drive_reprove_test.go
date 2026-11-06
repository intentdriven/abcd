package cli

// drain_drive_reprove_test.go holds the drive's check before a stage on a harness that
// went: a new one proved, one line, and the run goes on; one that fails ends it.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// reproveDrive runs a drive whose walls hold on one stand-in harness, and in the loop's
// place replaces the harness with stream and calls the check before a stage twice.
func reproveDrive(t *testing.T, newStream string) (first, second error, out string) {
	t.Helper()
	newDriveWorld(t)
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
		first, second = o.BeforeStage(), o.BeforeStage()
		return loop.TerminalResult{Ended: loop.DrainStoppedEmpty}, nil
	}
	var stdout, stderr strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, stdout.String(), stderr.String())
	}
	return first, second, stdout.String() + stderr.String()
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

// TestTheConsentSaysAnUpdateIsProvedInTheRun: it adds no prompt, and one line says so.
func TestTheConsentSaysAnUpdateIsProvedInTheRun(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d: %s%s", code, out.String(), errb.String())
	}
	if !strings.Contains(out.String(), "a harness update during the run is proved again") {
		t.Fatalf("the consent:\n%s", out.String())
	}
}
