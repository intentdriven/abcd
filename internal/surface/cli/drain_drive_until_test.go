package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/layered"
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
	if !strings.Contains(string(out), "bound:") || !strings.Contains(string(out), at.Format(time.RFC3339)+", --until;") {
		t.Fatalf("the consent names the bound and its source:\n%s", out)
	}
}

// sevenDaysOn reports whether end is seven days after a start made between
// before and after, in whole seconds.
func sevenDaysOn(end, before, after time.Time) bool {
	week := 7 * 24 * time.Hour
	return !end.Before(before.Add(week).Truncate(time.Second)) && !end.After(after.Add(week))
}

// TestADriveWithNoUntilPassesTheLoopABound: a drive the person starts with no
// --until ends seven days on, abcd's default, so the loop always has a bound
// and the eight-hour endings never end a drive a person started.
func TestADriveWithNoUntilPassesTheLoopABound(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	var out, errb strings.Builder
	before := time.Now()
	if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if !sevenDaysOn(w.terminal.Until, before, time.Now()) {
		t.Fatalf("with no --until the loop's bound is seven days on: %v", w.terminal.Until)
	}
}

// TestAStartWithNoUntilShowsAnEndSevenDaysOnBeforeTheQuestion: the consent
// names the end as an instant with its source, why no repository figure was
// read, the command that carries the drive on and what it holds, all before
// the question; the estimate carries the retry wording, since the drive waits.
func TestAStartWithNoUntilShowsAnEndSevenDaysOnBeforeTheQuestion(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	withTerminal(t)
	before := time.Now()
	out, err := runCLIPipedStdin(t, "n\n", "drain", "--drive")
	after := time.Now()
	if err != nil {
		t.Fatalf("a no is no failure: %v\n%s", err, out)
	}
	plan, _, asked := strings.Cut(string(out), "Start? [y/N]")
	if !asked {
		t.Fatalf("the question was not asked:\n%s", out)
	}
	m := regexp.MustCompile(`(?m)^  bound:    ends at (\S+), seven days on, abcd's default;`).FindStringSubmatch(plan)
	if m == nil {
		t.Fatalf("the bound line names the end and its source before the question:\n%s", out)
	}
	end, err := time.Parse(time.RFC3339, m[1])
	if err != nil || !sevenDaysOn(end, before, after) {
		t.Fatalf("the end shown is seven days on: %q (%v)", m[1], err)
	}
	for _, want := range []string{
		"no repository figure for drive.until_days was read",
		"  restart:  a reboot, a logout or a crash ends the drive",
		loop.TerminalResumeCommand + " --until " + m[1],
		"  holds:    1 lane in work;",
		"retries through failures",
	} {
		if !strings.Contains(plan, want) {
			t.Errorf("before the question the start says %q:\n%s", want, plan)
		}
	}
	if strings.Contains(plan, "eight hours") || strings.Contains(plan, "at most") {
		t.Errorf("the start no longer names an eight-hour wait or a ceiling on runs:\n%s", plan)
	}
}

// TestUntilWinsAndIsNotHeldToTheDefault: the person's --until is their own
// bound: it reaches the loop as given, past the default's 31 days, and the
// default is not read at all, so a setting that would refuse the default
// refuses nothing.
func TestUntilWinsAndIsNotHeldToTheDefault(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	w.machineConfig(t, `{"runner":{"claude":{}},"drive":{"until_days":3}}`)
	at := time.Now().Add(60 * 24 * time.Hour).UTC().Truncate(time.Second)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes", "--until", at.Format(time.RFC3339)}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if !w.terminal.Until.Equal(at) {
		t.Fatalf("the loop's bound is %v, want the --until %v", w.terminal.Until, at)
	}
	text := refusedDrive(t, "drain", "--drive", "--yes")
	if !strings.Contains(text, "drive.until_days") || w.terminalCalls != 1 {
		t.Fatalf("with no --until the default is read, and refused in the machine's file: %d calls: %s", w.terminalCalls, text)
	}
}

// TestTheRestartCommandIsPrintedOnceAfterTheYes: after the yes the drive
// prints the one command that carries it on to the same end, once, with the
// --profile the person typed.
func TestTheRestartCommandIsPrintedOnceAfterTheYes(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	withTerminal(t)
	out, err := runCLIPipedStdin(t, "y\n", "drain", "--drive", "--profile", "walled")
	if err != nil {
		t.Fatalf("exit: %v\n%s", err, out)
	}
	_, after, asked := strings.Cut(string(out), "Start? [y/N]")
	if !asked || w.terminalCalls != 1 {
		t.Fatalf("asked %v, %d calls:\n%s", asked, w.terminalCalls, out)
	}
	cmd := loop.TerminalResumeCommand + " --until " + w.terminal.Until.UTC().Format(time.RFC3339) + " --profile walled"
	if got := strings.Count(after, cmd); got != 1 {
		t.Fatalf("after the yes the command %q is printed %d times:\n%s", cmd, got, after)
	}
}

// driveLogLines are the lines of the drive's one log, each with its stamp cut.
func driveLogLines(t *testing.T, repo string) []string {
	t.Helper()
	logs, err := filepath.Glob(filepath.Join(repo, ".abcd", ".work.local", "logs", "drain-drive-*.log"))
	if err != nil || len(logs) != 1 {
		t.Fatalf("want one drive log, found %v (%v)", logs, err)
	}
	raw, err := os.ReadFile(logs[0])
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, l := range strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n") {
		_, rest, ok := strings.Cut(l, " ")
		if !ok {
			t.Fatalf("a log line with no stamp: %q", l)
		}
		lines = append(lines, rest)
	}
	return lines
}

// TestWithYesTheFullPlanIsTheLogsFirstLines: a drive started with --yes,
// which nobody may be watching, writes the full plan it showed, line for line,
// at the head of its log, the note on the default among them.
func TestWithYesTheFullPlanIsTheLogsFirstLines(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	shown := strings.Split(out.String(), "\n")
	n := 1
	for n < len(shown) && strings.HasPrefix(shown[n], "  ") {
		n++
	}
	plan := shown[:n]
	if !strings.HasPrefix(plan[0], "abcd drain --drive works through") || !strings.Contains(strings.Join(plan, "\n"), "  restart:  ") {
		t.Fatalf("the plan is the output's first lines:\n%s", out.String())
	}
	logged := driveLogLines(t, w.repo)
	if len(logged) < n || !slices.Equal(logged[:n], plan) {
		t.Fatalf("the log's first lines are the plan:\nplan:\n%s\nlog:\n%s", strings.Join(plan, "\n"), strings.Join(logged, "\n"))
	}
	if !strings.Contains(strings.Join(logged[:n], "\n"), "no repository figure for drive.until_days was read") {
		t.Fatalf("the log names why no repository figure was read:\n%s", strings.Join(logged, "\n"))
	}
}

// TestThePrintedCommandCarriesOnToTheFirstEndTime: a drive that ends leaving
// work names, as the way on, the command with its own end, and that command,
// typed, starts a drive that ends then and not seven days after it.
func TestThePrintedCommandCarriesOnToTheFirstEndTime(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	driveTerminal = func(_ context.Context, _ string, o loop.TerminalOptions) (loop.TerminalResult, error) {
		w.terminalCalls++
		w.terminal = o
		return loop.TerminalResult{Ended: loop.TerminalStopped, Resume: loop.TerminalResumeCommand}, nil
	}
	var out, errb strings.Builder
	Run([]string{"drain", "--drive", "--yes", "--profile", "walled"}, &out, &errb)
	first := w.terminal.Until
	m := regexp.MustCompile(`(?m)^  to resume: (.+)$`).FindStringSubmatch(out.String() + errb.String())
	if m == nil {
		t.Fatalf("the end names no way on:\n%s%s", out.String(), errb.String())
	}
	want := loop.TerminalResumeCommand + " --until " + first.UTC().Format(time.RFC3339) + " --profile walled"
	if m[1] != want {
		t.Fatalf("the way on is %q, want %q", m[1], want)
	}
	args := strings.Fields(strings.TrimPrefix(m[1], "abcd "))
	Run(append(args, "--yes"), &out, &errb)
	if w.terminalCalls != 2 || !w.terminal.Until.Equal(first) {
		t.Fatalf("the command carries the drive on to %v; it ended at %v (%d calls)", first, w.terminal.Until, w.terminalCalls)
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

// TestADriveAlwaysHandsTheLoopItsProbe: every drive a person starts has a
// bound, so the loop always has its probe of a limit.
func TestADriveAlwaysHandsTheLoopItsProbe(t *testing.T) {
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
	if w.terminal.Probe == nil {
		t.Fatal("with no --until the loop has its probe too")
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
	if strings.Contains(string(out), "at most") || !strings.Contains(string(out), "not a limit on what it spends") {
		t.Fatalf("with no --until the estimate holds no longer either:\n%s", out)
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

// TestALimitAtTheLiveProofIsWaitedOutWithNoUntil: a drive started with no
// --until has its default end, so a limit or an outage at the live proof is
// waited out and the drive goes on, where it used to be refused.
func TestALimitAtTheLiveProofIsWaitedOutWithNoUntil(t *testing.T) {
	t.Run("limit", func(t *testing.T) {
		w := newDriveWorld(t)
		w.passWalls(t)
		proofs, _ := limitedProof(t, 1, func(now time.Time) time.Time { return now.Add(3 * time.Hour) })
		var out, errb strings.Builder
		if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
			t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
		}
		if *proofs != 2 || w.terminalCalls != 1 {
			t.Fatalf("the proof runs again and the loop starts: %d proofs, %d calls", *proofs, w.terminalCalls)
		}
	})
	t.Run("outage", func(t *testing.T) {
		w := newDriveWorld(t)
		w.passWalls(t)
		proofs, _ := limitedProof(t, 0, nil)
		driveWalls.live = func(context.Context, runner.Profile) error {
			if *proofs++; *proofs < 3 {
				return &runner.NoVerdictError{Detail: "the launch came to no verdict"}
			}
			return nil
		}
		var out, errb strings.Builder
		if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
			t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
		}
		if *proofs != 3 || w.terminalCalls != 1 {
			t.Fatalf("the proof runs again and the loop starts: %d proofs, %d calls", *proofs, w.terminalCalls)
		}
	})
}

// TestADriveIsStillWaitingNineHoursIntoALimitThatNamesNoReset: the eight
// hours a limit with no reset used to end a wait on are not a drive's end,
// which is seven days on: nine hours in, the drive is still waiting, and
// goes on when the limit lifts.
func TestADriveIsStillWaitingNineHoursIntoALimitThatNamesNoReset(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	_, clock := limitedProof(t, 0, nil)
	start := *clock
	waitedPastNine := false
	driveWalls.live = func(context.Context, runner.Profile) error {
		if clock.Sub(start) >= 9*time.Hour {
			waitedPastNine = true
		}
		if clock.Sub(start) < 10*time.Hour {
			return &runner.LimitError{}
		}
		return nil
	}
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if !waitedPastNine || w.terminalCalls != 1 {
		t.Fatalf("nine hours in the drive still waits, then goes on: past nine %v, %d calls, waited %s",
			waitedPastNine, w.terminalCalls, clock.Sub(start))
	}
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

// TestAnEndAlreadyPastNamesAWayOnThatStarts: a drive that reaches its end with
// a lane on its branch, the way every default drive with work left ends, names
// a way on with no --until, since the start refuses an end already past and a
// new one fixes its own; the command it names starts.
func TestAnEndAlreadyPastNamesAWayOnThatStarts(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	clock := driveClock
	t.Cleanup(func() { driveClock = clock })
	driveTerminal = func(_ context.Context, _ string, o loop.TerminalOptions) (loop.TerminalResult, error) {
		w.terminalCalls++
		w.terminal = o
		// The drive ran to its end: the end has passed by the time it says how it ended.
		driveClock = func() time.Time { return o.Until.Add(time.Second) }
		return loop.TerminalResult{Ended: loop.TerminalUntil, Run: "run-1", Issue: "iss-1", Lane: "lane-1", Resume: loop.TerminalResumeCommand}, nil
	}
	at := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	var out, errb strings.Builder
	Run([]string{"drain", "--drive", "--yes", "--until", at, "--profile", "walled"}, &out, &errb)
	m := regexp.MustCompile(`(?m)^  to resume: (.+)$`).FindStringSubmatch(out.String() + errb.String())
	if m == nil {
		t.Fatalf("the end names no way on:\n%s%s", out.String(), errb.String())
	}
	if want := loop.TerminalResumeCommand + " --profile walled"; m[1] != want {
		t.Fatalf("past its end the way on is %q, want %q", m[1], want)
	}
	driveClock = clock
	driveTerminal = func(_ context.Context, _ string, o loop.TerminalOptions) (loop.TerminalResult, error) {
		w.terminalCalls++
		w.terminal = o
		return loop.TerminalResult{Ended: loop.DrainStoppedEmpty}, nil
	}
	var out2, errb2 strings.Builder
	if code := Run(append(strings.Fields(strings.TrimPrefix(m[1], "abcd ")), "--yes"), &out2, &errb2); code != 0 || w.terminalCalls != 2 {
		t.Fatalf("the way on starts: exit %d, %d calls:\n%s%s", code, w.terminalCalls, out2.String(), errb2.String())
	}
}

// TestAStopDuringTheReadOfTheDefaultEndsTheStart: the first interrupt while the
// start asks origin for the default ends a typed start before its question,
// with nothing started, even when the read it cut short fell back to the
// bundled seven days and a scripted yes would answer.
func TestAStopDuringTheReadOfTheDefaultEndsTheStart(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	resolve := driveResolveDays
	t.Cleanup(func() { driveResolveDays = resolve })
	driveResolveDays = func(ctx context.Context, _ string, _ layered.Roots) (loop.PaceValue, string, error) {
		signalThisProcess(t, func() bool { return ctx.Err() != nil })
		// The git child died on the same signal, so origin could not be asked.
		return loop.PaceValue{Value: loop.BundledDriveDays, Layer: "bundled", Origin: "bundled"},
			"origin could not be asked which branch is its default (signal: interrupt), so no repository figure for drive.until_days was read", nil
	}
	var out, errb strings.Builder
	code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb)
	text := out.String() + errb.String()
	if code != driveKilledExit || w.terminalCalls != 0 || !strings.Contains(text, "nothing started") {
		t.Fatalf("exit %d, %d calls:\n%s", code, w.terminalCalls, text)
	}
	if strings.Contains(text, "works through this repository") {
		t.Fatalf("the plan was shown after the stop:\n%s", text)
	}
	noDrainBegun(t, w.repo)
}
