package cli

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// The bound a person sets on a drive (iss-2611050753557564): --until reaches the
// loop, and the yes names it.

func TestUntilReachesTheLoopAsAnInstantOrAsTheEndOfADate(t *testing.T) {
	at := time.Now().Add(14 * 24 * time.Hour).UTC().Truncate(time.Second)
	day := time.Now().AddDate(0, 0, 14)
	endOfDay := time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 0, time.Local)
	for _, c := range []struct {
		name, given string
		want        time.Time
	}{
		{"instant", at.Format(time.RFC3339), at},
		{"date", day.Format(time.DateOnly), endOfDay},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := newDriveWorld(t)
			w.passWalls(t)
			var out, errb strings.Builder
			if code := Run([]string{"drain", "--drive", "--yes", "--until", c.given}, &out, &errb); code != 0 {
				t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
			}
			if !w.terminal.Until.Equal(c.want) {
				t.Fatalf("the loop's bound is %v, want %v", w.terminal.Until, c.want)
			}
		})
	}
}

func TestUntilRefusesWhatCannotBound(t *testing.T) {
	newDriveWorld(t)
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	for _, c := range []struct {
		args []string
		says string
	}{
		{[]string{"drain", "--until", future}, "--drive"},
		{[]string{"drain", "--drive", "--yes", "--until", "soon-ish"}, "RFC 3339"},
		{[]string{"drain", "--drive", "--yes", "--until", "2020-01-01"}, "already past"},
	} {
		text := refusedDrive(t, c.args...)
		if !strings.Contains(text, c.says) {
			t.Errorf("%v: the refusal does not say %q: %s", c.args, c.says, text)
		}
		if strings.Contains(text, "soon-ish") {
			t.Errorf("a refusal never repeats the value given: %s", text)
		}
	}
}

func TestTheYesNamesTheBound(t *testing.T) {
	at := time.Now().Add(14 * 24 * time.Hour).UTC().Truncate(time.Second)
	w := newDriveWorld(t)
	w.passWalls(t)
	withTerminal(t)
	out, err := runCLIPipedStdin(t, "n\n", "drain", "--drive", "--until", at.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("a no is no failure: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "bound:") || !strings.Contains(string(out), at.Format(time.RFC3339)) {
		t.Fatalf("the consent names the bound:\n%s", out)
	}
	out, err = runCLIPipedStdin(t, "n\n", "drain", "--drive")
	if err != nil {
		t.Fatalf("a no is no failure: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "bound:") || !strings.Contains(string(out), "eight hours") {
		t.Fatalf("with none given the consent names the eight hours:\n%s", out)
	}
}

func TestWithNoUntilTheLoopHasNoBound(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if !w.terminal.Until.IsZero() {
		t.Fatalf("no flag sets no bound: %v", w.terminal.Until)
	}
}

// TestADriveCanAskForHundredsOfLanes: --max has no ceiling of its own.
func TestADriveCanAskForHundredsOfLanes(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes", "--max", "500"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if w.terminal.Drain.Max != 500 {
		t.Fatalf("the cap reaches the drain: %d", w.terminal.Drain.Max)
	}
}

// TestADriveHandsTheLoopItsProbeOnlyUnderABound: the yes of a drive with no bound
// never mentioned a probe.
func TestADriveHandsTheLoopItsProbeOnlyUnderABound(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	at := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes", "--until", at}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if w.terminal.Probe == nil {
		t.Fatal("under a bound the loop has its probe")
	}
	if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if w.terminal.Probe != nil {
		t.Fatal("with no bound the loop has no probe")
	}
}

// TestAnEmptyUntilIsRefused: not read as no bound.
func TestAnEmptyUntilIsRefused(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	text := refusedDrive(t, "drain", "--drive", "--yes", "--until", "")
	if !strings.Contains(text, "--until") || w.terminalCalls != 0 {
		t.Fatalf("the refusal names the flag and nothing runs: %d calls: %s", w.terminalCalls, text)
	}
}

// TestUnderABoundTheEstimateSaysItIsNoCeiling: retries through limits and outages can
// pass it, and the bound is a time, not a limit on what is spent.
func TestUnderABoundTheEstimateSaysItIsNoCeiling(t *testing.T) {
	at := time.Now().Add(14 * 24 * time.Hour).UTC().Format(time.RFC3339)
	w := newDriveWorld(t)
	w.passWalls(t)
	withTerminal(t)
	out, err := runCLIPipedStdin(t, "n\n", "drain", "--drive", "--until", at)
	if err != nil {
		t.Fatalf("a no is no failure: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "at most") || !strings.Contains(string(out), "not a limit on what it spends") {
		t.Fatalf("the consent says the estimate holds no longer:\n%s", out)
	}
	out, err = runCLIPipedStdin(t, "n\n", "drain", "--drive")
	if err != nil {
		t.Fatalf("a no is no failure: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "at most") {
		t.Fatalf("with no bound the estimate reads as it did:\n%s", out)
	}
}

// limitedProof has the live proof meet a limit calls times, then hold, on an injected
// clock, and reports how many proofs ran.
func limitedProof(t *testing.T, calls int, reset func(now time.Time) time.Time) (*int, *time.Time) {
	t.Helper()
	clock := time.Now().UTC().Truncate(time.Second)
	wait := driveProofWait
	t.Cleanup(func() { driveProofWait = wait })
	driveProofWait.now = func() time.Time { return clock }
	driveProofWait.sleep = func(_ context.Context, d time.Duration) { clock = clock.Add(d) }
	proofs := 0
	driveWalls.live = func(context.Context, runner.Profile) error {
		proofs++
		if proofs <= calls {
			return &runner.LimitError{ResetsAt: reset(clock)}
		}
		return nil
	}
	return &proofs, &clock
}

// TestALimitAtTheLiveProofIsWaitedOutUnderABound: a restart during a limit waits for
// its reset, proves again, and goes on.
func TestALimitAtTheLiveProofIsWaitedOutUnderABound(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	proofs, _ := limitedProof(t, 1, func(now time.Time) time.Time { return now.Add(3 * time.Hour) })
	at := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes", "--until", at}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if *proofs != 2 || w.terminalCalls != 1 {
		t.Fatalf("the proof runs again and the loop starts: %d proofs, %d calls", *proofs, w.terminalCalls)
	}
	if !strings.Contains(out.String()+errb.String(), "rate limit") {
		t.Fatalf("the wait names the limit:\n%s%s", out.String(), errb.String())
	}
}

// TestALimitAtTheLiveProofWithNoBoundIsRefusedAsALimit: it says so, and blames no
// wall.
func TestALimitAtTheLiveProofWithNoBoundIsRefusedAsALimit(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	limitedProof(t, 1, func(now time.Time) time.Time { return now.Add(3 * time.Hour) })
	var out, errb strings.Builder
	code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb)
	text := out.String() + errb.String()
	if code != 3 || !strings.Contains(text, "rate limit") || strings.Contains(text, "does not hold") {
		t.Fatalf("exit %d, naming the limit:\n%s", code, text)
	}
	if !strings.Contains(text, "--until") || w.terminalCalls != 0 {
		t.Fatalf("it names the way to wait, and nothing runs: %d calls:\n%s", w.terminalCalls, text)
	}
	noDrainBegun(t, w.repo)
}

// TestAnOutageAtTheLiveProofIsWaitedOutUnderABound: a restart during an outage waits,
// proves again, and goes on.
func TestAnOutageAtTheLiveProofIsWaitedOutUnderABound(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	proofs, _ := limitedProof(t, 0, nil)
	driveWalls.live = func(context.Context, runner.Profile) error {
		if *proofs++; *proofs < 3 {
			return &runner.NoVerdictError{Detail: "the launch came to no verdict"}
		}
		return nil
	}
	at := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes", "--until", at}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if *proofs != 3 || w.terminalCalls != 1 {
		t.Fatalf("the proof runs again and the loop starts: %d proofs, %d calls", *proofs, w.terminalCalls)
	}
}

// TestAnOutageAtTheLiveProofWithNoBoundIsRefused: nothing there to wait with.
func TestAnOutageAtTheLiveProofWithNoBoundIsRefused(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	driveWalls.live = func(context.Context, runner.Profile) error {
		return &runner.NoVerdictError{Detail: "the launch came to no verdict"}
	}
	text := refusedDrive(t, "drain", "--drive", "--yes")
	if !strings.Contains(text, "no verdict") || w.terminalCalls != 0 {
		t.Fatalf("%d calls: %s", w.terminalCalls, text)
	}
}

// TestAVerdictAtTheLiveProofIsRefusedUnderABound: a check that does not hold is not
// waited out.
func TestAVerdictAtTheLiveProofIsRefusedUnderABound(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	driveWalls.live = func(context.Context, runner.Profile) error {
		return &runner.WallError{Wall: runner.WallTools, Detail: "it held a tool it was not given"}
	}
	at := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	text := refusedDrive(t, "drain", "--drive", "--yes", "--until", at)
	if !strings.Contains(text, "held a tool") || w.terminalCalls != 0 {
		t.Fatalf("%d calls: %s", w.terminalCalls, text)
	}
}

// TestAWaitAtTheLiveProofHoldsToTheSettings: settings that changed during the wait stop
// the probes it would make, and the drive.
func TestAWaitAtTheLiveProofHoldsToTheSettings(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	limitedProof(t, 1000, func(now time.Time) time.Time { return now.Add(24 * time.Hour) })
	driveWalls.settings = func(runner.Profile) error { return errors.New("the settings differ") }
	at := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes", "--until", at}, &out, &errb); code != 2 || w.terminalCalls != 0 {
		t.Fatalf("exit %d, %d calls:\n%s%s", code, w.terminalCalls, out.String(), errb.String())
	}
}

// TestABoundPassingAtTheLiveProofEndsCleanly: nothing starts, and that is no failure.
func TestABoundPassingAtTheLiveProofEndsCleanly(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	limitedProof(t, 1000, func(now time.Time) time.Time { return now.Add(30 * 24 * time.Hour) })
	at := time.Now().Add(6 * time.Hour).UTC().Format(time.RFC3339)
	var out, errb strings.Builder
	code := Run([]string{"drain", "--drive", "--yes", "--until", at}, &out, &errb)
	text := out.String() + errb.String()
	if code != 0 || w.terminalCalls != 0 || !strings.Contains(text, "nothing started") {
		t.Fatalf("exit %d, %d calls:\n%s", code, w.terminalCalls, text)
	}
	noDrainBegun(t, w.repo)
}

func TestTheEndSaysTheBoundEndedIt(t *testing.T) {
	var b strings.Builder
	renderDriveEnd(&b, loop.TerminalResult{Ended: loop.TerminalUntil})
	if !strings.Contains(b.String(), "--until") {
		t.Fatalf("the end names the bound: %s", b.String())
	}
}
