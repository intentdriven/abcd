package loop

import (
	"context"
	"errors"
	"slices"
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

// TestAPreflightThatFailsCountsTowardTheOutage: a preflight that cannot mint
// its receipt is bounded as a failing runner is.
func TestAPreflightThatFailsCountsTowardTheOutage(t *testing.T) {
	w := newTerminalWorld(t)
	w.needsPreflight = true
	w.preflight = errors.New("the preflight failed")
	o := w.options(t)
	o.Drain.Max = 1
	calls := 0
	inner := o.Preflight
	o.Preflight = func(ctx context.Context, wt string) error {
		calls++
		if calls > 3 {
			w.preflight = nil
		}
		err := inner(ctx, wt)
		if err != nil {
			w.preflights = nil
		}
		return err
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != DrainStoppedCap || calls != 4 {
		t.Fatalf("the failing preflight is retried on the schedule: %+v, %d calls, waits %v", res, calls, w.sleeps)
	}
	if !slices.Contains(w.sleeps, time.Minute) {
		t.Fatalf("the third failure enters the schedule: %v", w.sleeps)
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
