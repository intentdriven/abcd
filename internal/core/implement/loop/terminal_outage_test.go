package loop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
)

// repeated is the script line, n times.
func repeated(line string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = line
	}
	return out
}

// TestThreeFailuresInARowEnterTheOutageSchedule: two failures retry at the
// floor; the third enters one, five and ten minutes, then hourly, saying so;
// at eight hours the run ends, and nothing is handed back.
func TestThreeFailuresInARowEnterTheOutageSchedule(t *testing.T) {
	w := newTerminalWorld(t, append([]string{"shape"}, repeated("none", 40)...)...)
	o := w.options(t)
	o.Drain.Max = 1
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalOutage {
		t.Fatalf("eight hours of failures end the run: %+v", res)
	}
	want := []time.Duration{TerminalWaitFloor, TerminalWaitFloor, time.Minute, 5 * time.Minute, 10 * time.Minute, time.Hour}
	if len(w.sleeps) < len(want) || !slices.Equal(w.sleeps[:len(want)], want) {
		t.Fatalf("the waits are %v", w.sleeps)
	}
	var total time.Duration
	for _, d := range w.sleeps[2:] {
		total += d
	}
	if total < 8*time.Hour || total > 9*time.Hour {
		t.Fatalf("the run ends at eight hours of outage, after %v", total)
	}
	if !strings.Contains(strings.Join(w.lines, "\n"), "outage") {
		t.Fatalf("each wait says so: %q", w.lines)
	}
	st, _, _ := readDrain(w.repo.Root())
	if len(st.HandBacks) != 0 {
		t.Fatalf("an outage hands nothing back: %+v", st.HandBacks)
	}
	run, err := ReadState(w.repo.Root(), st.Lanes[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Lanes[0].HandBack != nil {
		t.Fatalf("the lane is not handed back: %+v", run.Lanes[0].HandBack)
	}
}

// TestAJudgeFailureEntersTheOutageAtOnceAndIsAskedAgain: one judge failure
// waits a minute; the same issue's judge is asked again, and its answer ends
// the outage.
func TestAJudgeFailureEntersTheOutageAtOnceAndIsAskedAgain(t *testing.T) {
	w := newTerminalWorld(t, "none", "none")
	o := w.options(t)
	o.Drain.Max = 1
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != DrainStoppedCap || len(lanesDone(t, w.repo)) != 1 {
		t.Fatalf("the judge's later answer lets the run go on: %+v\n%s", res, strings.Join(w.lines, "\n"))
	}
	if len(w.sleeps) < 2 || w.sleeps[0] != time.Minute || w.sleeps[1] != 5*time.Minute {
		t.Fatalf("a judge failure enters the schedule at once: %v", w.sleeps)
	}
}

// TestASuccessEndsTheCount: failures broken by a success never reach the
// schedule.
func TestASuccessEndsTheCount(t *testing.T) {
	var c terminalOutageClock
	now := time.Now()
	for range 2 {
		if d, up := c.fail(now, false); d != TerminalWaitFloor || up {
			t.Fatalf("below three a failure waits at the floor: %v %v", d, up)
		}
	}
	c.succeed()
	if d, _ := c.fail(now, false); d != TerminalWaitFloor {
		t.Fatalf("a success ends the count: %v", d)
	}
}

// TestAJudgedMoveThatThenWaitsEndsTheCount: the judge answers, so the outage is
// over, though the same move goes on to wait; the next failure starts the
// schedule afresh.
func TestAJudgedMoveThatThenWaitsEndsTheCount(t *testing.T) {
	w := newTerminalWorld(t, "none", "yes", "none")
	q := otherCopy(t, w.bare, quinnEmail)
	held := map[string]string{secondIssue: "commands/drain.md", thirdIssue: "commands/other.md"}
	for id, path := range held {
		if _, err := claim.Reserve(q, claim.Request{Record: id, Footprint: []string{path}}, as(t, q, claim.Options{})); err != nil {
			t.Fatal(err)
		}
	}
	w.onSleep = func(d time.Duration) {
		if d < drainWaitMinutes*time.Minute {
			return
		}
		for id := range held {
			delete(held, id)
			if _, err := claim.Reserve(q, claim.Request{Record: id, Release: true}, as(t, q, claim.Options{})); err != nil {
				t.Fatal(err)
			}
		}
	}
	o := w.options(t)
	o.Drain.Max = 1
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatal(err)
	}
	want := []time.Duration{time.Minute, drainWaitMinutes * time.Minute, time.Minute}
	if len(w.sleeps) < len(want) || !slices.Equal(w.sleeps[:len(want)], want) {
		t.Fatalf("the waits are %v, want %v first\n%s", w.sleeps, want, strings.Join(w.lines, "\n"))
	}
}

// sleptBefore fails a run that waits on and on, which would otherwise hang.
func (w *terminalWorld) guardSleeps(t *testing.T, o *TerminalOptions) {
	t.Helper()
	sleep := o.Sleep
	o.Sleep = func(ctx context.Context, d time.Duration) {
		if len(w.sleeps) > 300 {
			t.Fatalf("the run waits with no deadline: first waits %v\n%s", w.sleeps[:8], strings.Join(w.lines, "\n"))
		}
		sleep(ctx, d)
	}
}

// realWorktree has lanes cut a real worktree and write their receipts in their
// own directories, so a rate limit can save the work aside.
func (w *terminalWorld) realWorktree() {
	for _, s := range DefaultStages() {
		if s.Name == StageWorktree {
			w.stages[0] = s
		}
	}
	n := 0
	for i := range w.stages {
		if w.stages[i].Name == StageImplement {
			w.stages[i].Run = func(c Context, l *Lane) (Outcome, error) {
				n++
				dir := filepath.Join(c.RepoRoot, c.RunDir, l.ID)
				if err := os.MkdirAll(dir, 0o700); err != nil {
					return Outcome{}, err
				}
				return Outcome{Await: &Await{Role: RoleImplementer, Brief: filepath.Join(c.RepoRoot, l.Brief),
					Receipt: filepath.Join(c.RunDir, l.ID, fmt.Sprintf("receipt-%d.json", n))}}, nil
			}
			verify := w.stages[i].Verify
			w.stages[i].Verify = func(c Context, l *Lane, receipt string) error {
				return verify(c, l, absIn(c.RepoRoot, receipt))
			}
		}
	}
}

// slept adds the waits up.
func slept(ds []time.Duration) time.Duration {
	var total time.Duration
	for _, d := range ds {
		total += d
	}
	return total
}

// TestARateLimitWaitsForTheResetItReports: no agent starts until the time the
// harness said the limit resets.
func TestARateLimitWaitsForTheResetItReports(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	reset := w.clock.t.Add(3 * time.Hour)
	t.Setenv(terminalResetEnv, strconv.FormatInt(reset.Unix(), 10))
	o := w.options(t)
	o.Drain.Max = 1
	w.guardSleeps(t, &o)
	launchesAtFirstWait := -1
	w.onSleep = func(time.Duration) {
		if launchesAtFirstWait < 0 {
			launchesAtFirstWait = w.launches(t)
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != DrainStoppedCap {
		t.Fatalf("the lane lands after the reset: %+v\n%s", res, strings.Join(w.lines, "\n"))
	}
	if got := slept(w.sleeps); got < 3*time.Hour-time.Minute || got > 4*time.Hour {
		t.Fatalf("the run waits for the reset: %v, waits %v\n%s", got, w.sleeps, strings.Join(w.lines, "\n"))
	}
	if launchesAtFirstWait != 2 {
		t.Fatalf("a judge and the limited implementer start before the wait, %d launches", launchesAtFirstWait)
	}
	if w.launches(t) != 3 {
		t.Fatalf("one agent starts after the reset, %d launches\n%s", w.launches(t), strings.Join(w.lines, "\n"))
	}
}

// TestARateLimitThatReportsNoResetWaitsLongerEachTimeThenStops: every launch
// meets the limit and nothing says when it ends: the waits grow, few agents
// start, and the run ends saying so, handing nothing back.
func TestARateLimitThatReportsNoResetWaitsLongerEachTimeThenStops(t *testing.T) {
	w := newTerminalWorld(t, append([]string{"shape"}, repeated("limited", 60)...)...)
	w.realWorktree()
	o := w.options(t)
	o.Drain.Max = 1
	w.guardSleeps(t, &o)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalLimited {
		t.Fatalf("the run ends on the limit: %+v\n%s", res, strings.Join(w.lines, "\n"))
	}
	want := []time.Duration{time.Minute, 5 * time.Minute, 10 * time.Minute, time.Hour}
	if len(w.sleeps) < len(want) || !slices.Equal(w.sleeps[:len(want)], want) {
		t.Fatalf("the waits grow: %v", w.sleeps)
	}
	if n := w.launches(t); n > 15 {
		t.Fatalf("few agents start against a limit: %d", n)
	}
	if !strings.Contains(strings.Join(w.lines, "\n"), "rate limit") {
		t.Fatalf("the run says why it ended: %q", w.lines)
	}
	st, _, _ := readDrain(w.repo.Root())
	if len(st.HandBacks) != 0 {
		t.Fatalf("a limit hands nothing back: %+v", st.HandBacks)
	}
}

// TestAResetPastTheDeadlineEndsTheRunNamingIt: a reset days away is not slept
// towards; the run ends at once and says when it is.
func TestAResetPastTheDeadlineEndsTheRunNamingIt(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	reset := w.clock.t.Add(72 * time.Hour).UTC().Truncate(time.Second)
	t.Setenv(terminalResetEnv, strconv.FormatInt(reset.Unix(), 10))
	o := w.options(t)
	o.Drain.Max = 1
	w.guardSleeps(t, &o)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalLimited || len(w.sleeps) != 0 {
		t.Fatalf("the run ends at once: %+v, waits %v", res, w.sleeps)
	}
	if !strings.Contains(strings.Join(w.lines, "\n"), reset.Format(time.RFC3339)) {
		t.Fatalf("the run names the reset: %q", w.lines)
	}
}

// TestAPreflightThatMintsNoReceiptCountsTowardStuck: a preflight that exits 0
// and leaves no receipt runs once for a head; the lane then goes to the stuck
// check rather than round again for ever.
func TestAPreflightThatMintsNoReceiptCountsTowardStuck(t *testing.T) {
	w := newTerminalWorld(t)
	w.noReceipt = true
	o := w.options(t)
	o.Drain.Max = 1
	o.Drain.Skip, o.Drain.SkipReason = []string{secondIssue, thirdIssue}, "the one lane's alone"
	w.guardSleeps(t, &o)
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatalf("%v\n%s", err, strings.Join(w.lines, "\n"))
	}
	if len(w.preflights) != 1 {
		t.Fatalf("the preflight runs once for the head: %d\n%s", len(w.preflights), strings.Join(w.lines, "\n"))
	}
	st, _, _ := readDrain(w.repo.Root())
	if len(st.HandBacks) != 1 || st.HandBacks[0].Kind != HandBackStuck {
		t.Fatalf("the lane is handed back as stuck: %+v\n%s", st.HandBacks, strings.Join(w.lines, "\n"))
	}
}

// heldByAPeer has another copy reserve all three issues and keep them for as long as
// the run waits, renewing at half the lease as a drive does. Each renewal is a commit
// on origin, so renewing at every wait would be much of what a long wait's test does.
// At every look a hold has more of its lease left than the longest wait, so the run
// waiting sees only that it is held.
func (w *terminalWorld) heldByAPeer(t *testing.T) {
	t.Helper()
	q := otherCopy(t, w.bare, quinnEmail)
	held := map[string]string{eligibleIssue: "AGENTS.md", secondIssue: "commands/drain.md", thirdIssue: "commands/other.md"}
	opts := func() claim.Options { return as(t, q, claim.Options{Now: w.clock.now}) }
	for id, path := range held {
		if _, err := claim.Reserve(q, claim.Request{Record: id, Footprint: []string{path}}, opts()); err != nil {
			t.Fatal(err)
		}
	}
	renewed := w.clock.t
	w.onSleep = func(d time.Duration) {
		now := w.clock.t
		wakes := now.Add(d)
		if wakes.Sub(renewed) < terminalRenewEvery {
			return
		}
		w.clock.t = wakes
		for id := range held {
			if _, err := claim.Reserve(q, claim.Request{Record: id, Renew: true}, opts()); err != nil {
				t.Fatal(err)
			}
		}
		w.clock.t = now
		renewed = wakes
	}
}

// TestAWaitOnHeldWorkHasADeadline: everything stays held by someone else, so the
// move names one wait after another; the run ends at the deadline, saying so,
// and hands nothing back.
func TestAWaitOnHeldWorkHasADeadline(t *testing.T) {
	w := newTerminalWorld(t)
	w.heldByAPeer(t)
	o := w.options(t)
	w.guardSleeps(t, &o)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalWaited {
		t.Fatalf("the wait ends the run: %+v\n%s", res, strings.Join(w.lines, "\n"))
	}
	if got := slept(w.sleeps); got < terminalOutageLimit || got > terminalOutageLimit+time.Hour {
		t.Fatalf("the run waits as long as a drive waits: %v", got)
	}
	if !strings.Contains(w.lines[len(w.lines)-1], "stopping") {
		t.Fatalf("the last line says so: %q", w.lines[len(w.lines)-1])
	}
}
