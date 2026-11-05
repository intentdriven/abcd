package loop

// terminal_longrun_test.go is a drive with a bound of the person's own
// (iss-2611050753557564): each shape a limit can take, a long outage and a stuck
// queue, all waited through until the bound, with the claim kept or backed off.
// The clock and the sleep are injected, so a wait of days takes no time.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// bounded is the options with the bound d away.
func (w *terminalWorld) bounded(t *testing.T, d time.Duration) TerminalOptions {
	t.Helper()
	w.clock.t = w.clock.t.Truncate(time.Second)
	o := w.options(t)
	o.Until = w.clock.t.Add(d)
	o.Drain.Max = 1
	w.guardSleeps(t, &o)
	return o
}

// resetIn has the fake harness's limit name a reset d away.
func (w *terminalWorld) resetIn(t *testing.T, d time.Duration) time.Time {
	t.Helper()
	at := w.clock.t.Add(d)
	t.Setenv(terminalResetEnv, strconv.FormatInt(at.Unix(), 10))
	return at.UTC().Truncate(time.Second)
}

func (w *terminalWorld) said() string { return strings.Join(w.lines, "\n") }

// letGoAtTheEnd fails unless the lane's run let its reservation go.
func letGoAtTheEnd(t *testing.T, w *terminalWorld) State {
	t.Helper()
	st, _, _ := readDrain(w.repo.Root())
	if len(st.Lanes) != 1 {
		t.Fatalf("one lane: %+v", st.Lanes)
	}
	run, err := ReadState(w.repo.Root(), st.Lanes[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	if !letGoByDrive(run) {
		t.Fatalf("the reservation is released as the run ends:\n%s", w.said())
	}
	return run
}

// TestAResetDaysAwayIsWaitedOutUnderABound: with a bound of its own a drive waits
// for a reset however far off, starts nothing before it, keeps renewing what it
// holds, says so as it goes, and lands the lane afterwards.
func TestAResetDaysAwayIsWaitedOutUnderABound(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	reset := w.resetIn(t, 72*time.Hour)
	o := w.bounded(t, 30*24*time.Hour)
	w.onSleep = func(time.Duration) {
		if n := w.launches(t); w.clock.t.Before(reset) && n != 2 {
			t.Fatalf("an agent started during the wait: %d launches", n)
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != DrainStoppedCap || w.launches(t) != 3 {
		t.Fatalf("the lane lands after the reset: %+v, %d launches\n%s", res, w.launches(t), w.said())
	}
	if got := slept(w.sleeps); got < 72*time.Hour-time.Minute || got > 73*time.Hour {
		t.Fatalf("the run waits for the reset: %v", got)
	}
	st, _, _ := readDrain(w.repo.Root())
	run, err := ReadState(w.repo.Root(), st.Lanes[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	renewed := 0
	for _, e := range run.Record {
		if e.Stage == StageReservationRenewed {
			renewed++
		}
	}
	if renewed < 30 {
		t.Fatalf("the wait keeps its claim: %d renewals in three days", renewed)
	}
	if n := strings.Count(w.said(), "rate limit"); n < 60 {
		t.Fatalf("the wait says so as it goes: %d lines\n%s", n, w.said())
	}
}

// limitedNoReset is a probe's answer: a limit with no reset named.
func limitedNoReset() *runner.Failure {
	return &runner.Failure{Runner: runner.Claude, Reason: runner.ReasonRateLimited, Detail: "a limit"}
}

// TestALimitWithNoResetIsProbedOnACappedBackoff: nothing says when it ends, so
// the run probes, no further apart than the cap, past eight hours, starting no
// lane's agent until a probe says the limit has lifted.
func TestALimitWithNoResetIsProbedOnACappedBackoff(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	o := w.bounded(t, 30*24*time.Hour)
	probes := 0
	o.Probe = func(_ context.Context, role string) (bool, *runner.Failure) {
		if role != RoleImplementer {
			t.Errorf("the probe asks for the role that met the limit, got %q", role)
		}
		probes++
		if probes < 30 {
			return false, limitedNoReset()
		}
		return true, nil
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != DrainStoppedCap || probes != 30 {
		t.Fatalf("the run goes on once a probe says so: %+v, %d probes\n%s", res, probes, w.said())
	}
	if w.launches(t) != 3 {
		t.Fatalf("a probe stands in for a lane's launch: %d launches\n%s", w.launches(t), w.said())
	}
	if max := slices.Max(w.sleeps); max > terminalProbeCap {
		t.Fatalf("no two probes are further apart than the cap: %v", w.sleeps)
	}
	want := []time.Duration{time.Minute, 5 * time.Minute, 10 * time.Minute, 30 * time.Minute}
	if !slices.Equal(w.sleeps[:len(want)], want) {
		t.Fatalf("the backoff grows to its cap: %v", w.sleeps)
	}
	if got := slept(w.sleeps); got <= 8*time.Hour {
		t.Fatalf("the run outlasts eight hours: %v", got)
	}
}

// TestAProbeThatNamesAResetWaitsForIt: the probe learns what the limit did not
// say, and the run stops probing and waits for that time.
func TestAProbeThatNamesAResetWaitsForIt(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	o := w.bounded(t, 30*24*time.Hour)
	reset := w.clock.t.Add(48 * time.Hour)
	probes := 0
	o.Probe = func(context.Context, string) (bool, *runner.Failure) {
		probes++
		fl := limitedNoReset()
		fl.ResetsAt = reset
		return false, fl
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != DrainStoppedCap || probes != 1 {
		t.Fatalf("one probe, then the wait: %+v, %d probes\n%s", res, probes, w.said())
	}
	if got := slept(w.sleeps); got < 48*time.Hour-time.Minute {
		t.Fatalf("the run waits for the reset the probe named: %v", got)
	}
}

// TestAProbeThatGetsNoAnswerLeavesItToTheLane: a probe that fails some other way, a
// harness that cannot answer it or the network gone, says nothing about the limit,
// so a few in a row hand the question to the lane's own launch rather than wait on
// a probe that may never answer.
func TestAProbeThatGetsNoAnswerLeavesItToTheLane(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	o := w.bounded(t, 30*24*time.Hour)
	probes := 0
	o.Probe = func(context.Context, string) (bool, *runner.Failure) {
		probes++
		return false, &runner.Failure{Runner: runner.Claude, Reason: runner.ReasonFailed, Detail: "no answer"}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != DrainStoppedCap {
		t.Fatalf("the lane's own launch finds out: %+v %v, %d probes\n%s", res, err, probes, w.said())
	}
	if probes != terminalProbeMisses || w.launches(t) != 3 {
		t.Fatalf("%d probes, %d launches\n%s", probes, w.launches(t), w.said())
	}
}

// TestWithNoBoundAProbedLimitStillEndsAtEightHours: an attended run behaves as
// it did, and its waits grow as they did.
func TestWithNoBoundAProbedLimitStillEndsAtEightHours(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	o := w.options(t)
	o.Drain.Max = 1
	w.guardSleeps(t, &o)
	probes := 0
	o.Probe = func(context.Context, string) (bool, *runner.Failure) {
		probes++
		return false, limitedNoReset()
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalLimited || w.launches(t) != 2 {
		t.Fatalf("the run ends on the limit: %+v, %d launches\n%s", res, w.launches(t), w.said())
	}
	want := []time.Duration{time.Minute, 5 * time.Minute, 10 * time.Minute, time.Hour}
	if !slices.Equal(w.sleeps[:len(want)], want) {
		t.Fatalf("the waits are as they were: %v", w.sleeps)
	}
	if got := slept(w.sleeps); got < 8*time.Hour || got > 9*time.Hour {
		t.Fatalf("eight hours: %v", got)
	}
	if probes == 0 {
		t.Fatal("the probe stands in for the lane's launch there too")
	}
}

// TestAnImplausibleResetIsTreatedAsNone: a reset a year off is a misread field,
// not a wait, so it is probed like a limit that named none.
func TestAnImplausibleResetIsTreatedAsNone(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	w.resetIn(t, 400*24*time.Hour)
	o := w.bounded(t, 30*24*time.Hour)
	o.Probe = func(context.Context, string) (bool, *runner.Failure) { return true, nil }
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != DrainStoppedCap {
		t.Fatalf("the run goes on: %+v %v\n%s", res, err, w.said())
	}
	if got := slept(w.sleeps); got > time.Hour {
		t.Fatalf("it never waits for that reset: %v", got)
	}
}

// TestARateLimitedJudgeWaitsForTheReset: the judge is an agent like any other:
// none starts before the reset.
func TestARateLimitedJudgeWaitsForTheReset(t *testing.T) {
	w := newTerminalWorld(t, "limited")
	w.resetIn(t, 3*time.Hour)
	o := w.options(t)
	o.Drain.Max = 1
	w.guardSleeps(t, &o)
	w.onSleep = func(time.Duration) {
		if n := w.launches(t); n != 1 {
			t.Fatalf("an agent started during the wait: %d launches", n)
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != DrainStoppedCap {
		t.Fatalf("the judge is asked again after the reset: %+v\n%s", res, w.said())
	}
	if got := slept(w.sleeps); got < 3*time.Hour-time.Minute || got > 4*time.Hour {
		t.Fatalf("the run waits for the reset: %v, %v\n%s", got, w.sleeps, w.said())
	}
	if !strings.Contains(w.said(), "rate limit") {
		t.Fatalf("the wait names the limit:\n%s", w.said())
	}
}

// TestAFarResetOnAJudgeWithNoBoundEndsAsALimit: it ends the run as a limit, never
// as runner failures, and at once.
func TestAFarResetOnAJudgeWithNoBoundEndsAsALimit(t *testing.T) {
	w := newTerminalWorld(t, "limited")
	reset := w.resetIn(t, 72*time.Hour)
	o := w.options(t)
	o.Drain.Max = 1
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalLimited || len(w.sleeps) != 0 {
		t.Fatalf("the run ends on the limit, at once: %+v, %v\n%s", res, w.sleeps, w.said())
	}
	if !strings.Contains(w.said(), reset.Format(time.RFC3339)) {
		t.Fatalf("the run names the reset:\n%s", w.said())
	}
}

// TestAFarResetOnAJudgeUnderABoundIsWaitedOut: and under a bound it waits.
func TestAFarResetOnAJudgeUnderABoundIsWaitedOut(t *testing.T) {
	w := newTerminalWorld(t, "limited")
	w.resetIn(t, 72*time.Hour)
	o := w.bounded(t, 30*24*time.Hour)
	first := 0
	w.onSleep = func(time.Duration) {
		if first == 0 {
			first = w.launches(t)
		}
		if n := w.launches(t); n != first {
			t.Fatalf("an agent started during the wait: %d, then %d", first, n)
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != DrainStoppedCap {
		t.Fatalf("the run goes on: %+v %v\n%s", res, err, w.said())
	}
	if got := slept(w.sleeps); got < 72*time.Hour-time.Minute {
		t.Fatalf("the run waits for the reset: %v", got)
	}
}

// TestAnOutageUnderABoundOutlastsEightHours: failures in a row are waited through
// on the schedule until the bound, which ends the run with the lane saved and
// nothing handed back.
func TestAnOutageUnderABoundOutlastsEightHours(t *testing.T) {
	w := newTerminalWorld(t, append([]string{"shape"}, repeated("none", 60)...)...)
	o := w.bounded(t, 24*time.Hour)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalUntil || res.Run == "" || res.Resume != TerminalResumeCommand {
		t.Fatalf("the bound ends the run, naming the lane: %+v\n%s", res, w.said())
	}
	if got := slept(w.sleeps); got != 24*time.Hour {
		t.Fatalf("the run waits to the bound and no further: %v, %v", got, w.sleeps)
	}
	if max := slices.Max(w.sleeps); max > time.Hour {
		t.Fatalf("the wait is capped: %v", w.sleeps)
	}
	st, _, _ := readDrain(w.repo.Root())
	if len(st.HandBacks) != 0 {
		t.Fatalf("an outage hands nothing back: %+v", st.HandBacks)
	}
	run := letGoAtTheEnd(t, w)
	if run.Lanes[0].HandBack != nil {
		t.Fatalf("the lane is not handed back: %+v", run.Lanes[0].HandBack)
	}
}

// TestAStuckQueueUnderABoundOutlastsEightHours: everything held by someone else
// is waited on until the bound, at a capped interval, each wait saying who holds
// it, and the run renews nothing it does not hold.
func TestAStuckQueueUnderABoundOutlastsEightHours(t *testing.T) {
	w := newTerminalWorld(t)
	q := otherCopy(t, w.bare, quinnEmail)
	held := map[string]string{eligibleIssue: "AGENTS.md", secondIssue: "commands/drain.md", thirdIssue: "commands/other.md"}
	opts := func() claim.Options { return as(t, q, claim.Options{Now: w.clock.now}) }
	for id, path := range held {
		if _, err := claim.Reserve(q, claim.Request{Record: id, Footprint: []string{path}}, opts()); err != nil {
			t.Fatal(err)
		}
	}
	w.onSleep = func(d time.Duration) {
		now := w.clock.t
		w.clock.t = now.Add(d)
		for id := range held {
			if _, err := claim.Reserve(q, claim.Request{Record: id, Renew: true}, opts()); err != nil {
				t.Fatal(err)
			}
		}
		w.clock.t = now
	}
	o := w.bounded(t, 10*time.Hour)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalUntil {
		t.Fatalf("the bound ends the wait: %+v\n%s", res, w.said())
	}
	if got := slept(w.sleeps); got != 10*time.Hour {
		t.Fatalf("the run waits to the bound and no further: %v", got)
	}
	if !strings.Contains(w.lines[len(w.lines)-1], "--until") {
		t.Fatalf("the last line says why it ended: %q", w.lines[len(w.lines)-1])
	}
}

// TestLosingOriginUnderABoundOutlastsEightHours: the drain's own give-up on origin
// stands down under a bound: the run keeps looking until it ends.
func TestLosingOriginUnderABoundOutlastsEightHours(t *testing.T) {
	w := newTerminalWorld(t)
	w.repo.Git("remote", "set-url", "origin", filepath.Join(t.TempDir(), "gone"))
	o := w.bounded(t, 24*time.Hour)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalUntil {
		t.Fatalf("the bound ends the wait, not origin: %+v\n%s", res, w.said())
	}
	if got := slept(w.sleeps); got != 24*time.Hour {
		t.Fatalf("the run waits to the bound and no further: %v, %v", got, w.sleeps)
	}
}

// TestLosingOriginWithNoBoundStillGivesUp: an attended run behaves as it did.
func TestLosingOriginWithNoBoundStillGivesUp(t *testing.T) {
	w := newTerminalWorld(t)
	w.repo.Git("remote", "set-url", "origin", filepath.Join(t.TempDir(), "gone"))
	o := w.options(t)
	w.guardSleeps(t, &o)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != DrainStoppedOrigin {
		t.Fatalf("the drain gives up on origin: %+v\n%s", res, w.said())
	}
}

// TestAnOutageTheSharedRunGaveUpOnWaitsUnderABound: a record another session gave
// up on holds the drive until a person clears it or the bound passes: it opens
// nothing, says so, and does not end the run.
func TestAnOutageTheSharedRunGaveUpOnWaitsUnderABound(t *testing.T) {
	w := newTerminalWorld(t)
	run := joinShared(t, w.repo.Root(), w.clock.now)
	giveUp(t, run, "host-a", func(d time.Duration) { w.clock.t = w.clock.t.Add(d) })
	o := w.bounded(t, 10*time.Hour)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalUntil || w.launches(t) != 0 {
		t.Fatalf("the bound ends the wait, and no agent starts: %+v, %d launches\n%s", res, w.launches(t), w.said())
	}
	if !strings.Contains(w.said(), "outage clear") {
		t.Fatalf("the wait names what lifts it:\n%s", w.said())
	}
}

// TestABoundEndsTheRunWithTheLaneSavedAndResumable: a reset past the bound is
// waited for only as far as the bound, which ends the run with the lane on its
// branch and its reservation released; a later drive takes the lane up.
func TestABoundEndsTheRunWithTheLaneSavedAndResumable(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	w.resetIn(t, 72*time.Hour)
	o := w.bounded(t, 24*time.Hour)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalUntil || res.Run == "" || res.Resume != TerminalResumeCommand || w.launches(t) != 2 {
		t.Fatalf("the bound ends the run, naming the lane: %+v, %d launches\n%s", res, w.launches(t), w.said())
	}
	if got := slept(w.sleeps); got != 24*time.Hour {
		t.Fatalf("the run waits to the bound and no further: %v", got)
	}
	letGoAtTheEnd(t, w)

	w.clock.t = w.clock.t.Add(72 * time.Hour)
	again := w.bounded(t, 24*time.Hour)
	res, err = RunTerminal(context.Background(), w.repo.Root(), again)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != DrainStoppedCap || len(lanesDone(t, w.repo)) != 1 {
		t.Fatalf("a later drive lands the lane: %+v\n%s", res, w.said())
	}
}

// TestABoundThatHasPassedStartsNothing: checked before every move.
func TestABoundThatHasPassedStartsNothing(t *testing.T) {
	w := newTerminalWorld(t)
	o := w.bounded(t, 0)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != TerminalUntil || w.launches(t) != 0 {
		t.Fatalf("nothing starts: %+v %v, %d launches", res, err, w.launches(t))
	}
}

// TestAMachineAsleepCutsTheWaitAndRevivesTheClaim: the machine sleeps through two
// days of a three-day wait and a lease with them. The wait counts what the wall
// clock says, so it ends at the reset, and the claim is held again on waking.
func TestAMachineAsleepCutsTheWaitAndRevivesTheClaim(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	reset := w.resetIn(t, 72*time.Hour)
	o := w.bounded(t, 30*24*time.Hour)
	once := false
	w.onSleep = func(time.Duration) {
		if !once {
			once = true
			w.clock.t = w.clock.t.Add(48 * time.Hour)
		}
		if n := w.launches(t); w.clock.t.Before(reset) && n != 2 {
			t.Fatalf("an agent started during the wait: %d", n)
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != DrainStoppedCap {
		t.Fatalf("the lane lands: %+v\n%s", res, w.said())
	}
	if got := slept(w.sleeps); got > 25*time.Hour {
		t.Fatalf("the time asleep counts: the run slept %v more", got)
	}
	if w.clock.t.Before(reset) {
		t.Fatalf("the wait ended before the reset: %v", w.clock.t)
	}
	st, _, _ := readDrain(w.repo.Root())
	run, err := ReadState(w.repo.Root(), st.Lanes[0].RunID)
	if err != nil {
		t.Fatal(err)
	}
	revived := slices.ContainsFunc(run.Record, func(e Entry) bool {
		return e.Stage == StageReservationRenewed && strings.Contains(e.Note, "lapsed")
	})
	if !revived {
		t.Fatalf("the lapsed claim is held again on waking:\n%s", w.said())
	}
}

// TestAPeerWhoTookALapsedIssueBacksTheLaneOff: while the machine slept the lease
// lapsed and a peer reserved the issue, so the lane starts no further agent, goes
// back as stuck, and the run moves on; the peer keeps what it took.
func TestAPeerWhoTookALapsedIssueBacksTheLaneOff(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	w.resetIn(t, 5*time.Hour)
	o := w.bounded(t, 30*24*time.Hour)
	o.Drain.Max = 2
	q := otherCopy(t, w.bare, quinnEmail)
	took := ""
	launchesAtWake := 0
	w.onSleep = func(time.Duration) {
		if took != "" {
			return
		}
		st, _, _ := readDrain(w.repo.Root())
		took = st.Lanes[0].Issue
		launchesAtWake = w.launches(t)
		w.clock.t = w.clock.t.Add(3 * time.Hour)
		q2 := as(t, q, claim.Options{Now: w.clock.now})
		if _, err := claim.Reserve(q, claim.Request{Record: took, Footprint: []string{"elsewhere.md"}, Lease: 24 * time.Hour}, q2); err != nil {
			t.Fatal(err)
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	st, _, _ := readDrain(w.repo.Root())
	if len(st.HandBacks) != 1 || st.HandBacks[0].Kind != HandBackStuck || st.HandBacks[0].Issue != took {
		t.Fatalf("the lane goes back as stuck: %+v %+v\n%s", res, st.HandBacks, w.said())
	}
	if _, err := claim.Fetch(w.repo.Root(), claim.Options{Now: w.clock.now}); err != nil {
		t.Fatal(err)
	}
	snap, err := claim.View(w.repo.Root(), w.clock.t)
	if err != nil {
		t.Fatal(err)
	}
	if r, ok := snap.Lookup(took); !ok || r.State != claim.StateLive || r.Copy == claim.CopyThis {
		t.Fatalf("the peer keeps what it took: %+v", r)
	}
	// One judge and one implementer for the next lane, if it opened.
	if n := w.launches(t); n > launchesAtWake+2 {
		t.Fatalf("no further agent starts on the lane: %d, then %d\n%s", launchesAtWake, n, w.said())
	}
}

// bareTerminal is a run with nothing to renew, for the wait's own rules.
func bareTerminal(t *testing.T, c *clock, until time.Time) (*terminal, *[]time.Duration, *[]string) {
	t.Helper()
	var sleeps []time.Duration
	var lines []string
	stop := make(chan struct{})
	o := TerminalOptions{Options: Options{Now: c.now}, Until: until, Stop: stop,
		Say: func(s string) { lines = append(lines, s) },
		Sleep: func(_ context.Context, d time.Duration) {
			sleeps = append(sleeps, d)
			c.t = c.t.Add(d)
		}}
	return newTerminal(t.TempDir(), o), &sleeps, &lines
}

// TestAClockSetBackDoesNotStretchAWait: the wall clock can only shorten a wait:
// the time actually slept still ends it.
func TestAClockSetBackDoesNotStretchAWait(t *testing.T) {
	c := &clock{t: time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)}
	tm, sleeps, _ := bareTerminal(t, c, time.Time{})
	inner := tm.o.Sleep
	first := true
	tm.o.Sleep = func(ctx context.Context, d time.Duration) {
		inner(ctx, d)
		if first {
			first = false
			c.t = c.t.Add(-5 * time.Hour)
		}
	}
	tm.sleep(context.Background(), 3*time.Hour, "a wait")
	if got := slept(*sleeps); got != 3*time.Hour {
		t.Fatalf("the wait lasts what it asked: %v", got)
	}
}

// TestAWaitSaysItsLineOnChangeAndPeriodically: the same wait repeated is one line
// per period, and a change of reason is a line at once.
func TestAWaitSaysItsLineOnChangeAndPeriodically(t *testing.T) {
	c := &clock{t: time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)}
	tm, _, lines := bareTerminal(t, c, time.Time{})
	for range 240 { // two hours of thirty seconds
		tm.sleep(context.Background(), TerminalWaitFloor, "held by a peer")
	}
	if n := len(*lines); n < 4 || n > 5 {
		t.Fatalf("one line each half hour, %d: %q", n, *lines)
	}
	tm.sleep(context.Background(), TerminalWaitFloor, "something else")
	if !strings.Contains((*lines)[len(*lines)-1], "something else") {
		t.Fatalf("a change says so at once: %q", *lines)
	}
}

// TestAContendedWaitBacksOffToItsCap: a wait with nothing to go on grows, under a
// bound, up to its cap, and with no bound stays at the floor.
func TestAContendedWaitBacksOffToItsCap(t *testing.T) {
	c := &clock{t: time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)}
	tm, sleeps, _ := bareTerminal(t, c, c.t.Add(48*time.Hour))
	for range 200 {
		if tm.idle(context.Background(), TerminalWaitFloor, "a contended lock") {
			t.Fatal("a bound's idle wait never ends the run")
		}
	}
	if (*sleeps)[0] != TerminalWaitFloor || slices.Max(*sleeps) != terminalIdleCap {
		t.Fatalf("the waits grow to the cap: %v", (*sleeps)[:12])
	}
	c = &clock{t: time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)}
	tm, sleeps, _ = bareTerminal(t, c, time.Time{})
	for range 100 {
		tm.idle(context.Background(), TerminalWaitFloor, "a contended lock")
	}
	if slices.Max(*sleeps) != TerminalWaitFloor {
		t.Fatalf("with no bound the wait stays at the floor: %v", *sleeps)
	}
}

// TestASleepThatOverrunsEndsOnTheWallClock: the production sleep looks at the wall
// clock, so a machine that slept past the end wakes the wait at once.
func TestASleepThatOverrunsEndsOnTheWallClock(t *testing.T) {
	start := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	var offset atomic.Int64
	now := func() time.Time { return start.Add(time.Duration(offset.Load())) }
	looks := make(chan time.Time)
	done := make(chan struct{})
	go func() {
		wallSleep(context.Background(), time.Hour, nil, now, looks)
		close(done)
	}()
	looks <- start // not yet: a second send proves it is still looking
	looks <- start
	offset.Store(int64(2 * time.Hour))
	for {
		select {
		case looks <- start:
		case <-done:
			return
		case <-time.After(10 * time.Second):
			t.Fatal("the sleep did not end when the wall clock passed its end")
		}
	}
}

// evidenceFiles are the evidence files a run left.
func evidenceFiles(t *testing.T, root, kind string) []string {
	t.Helper()
	got, err := filepath.Glob(filepath.Join(root, ".abcd", ".work.local", "logs", "drive-"+kind+"-*.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// evidenceLines reads one evidence file's entries.
func evidenceLines(t *testing.T, path string) []map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("an evidence line is not JSON: %q", line)
		}
		out = append(out, m)
	}
	return out
}

// TestTheFirstLimitLeavesItsEvidence: what the harness showed, the exit, the reset
// read, what the loop decided, and when it resumed; and no second file for a
// second limit in the same run.
func TestTheFirstLimitLeavesItsEvidence(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited", "limited")
	w.realWorktree()
	reset := w.resetIn(t, 3*time.Hour)
	o := w.bounded(t, 30*24*time.Hour)
	o.Probe = func(context.Context, string) (bool, *runner.Failure) { return true, nil }
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatal(err)
	}
	files := evidenceFiles(t, w.repo.Root(), "limit")
	if len(files) != 1 {
		t.Fatalf("one evidence file for the run's first limit: %v\n%s", files, w.said())
	}
	fi, err := os.Stat(files[0])
	if err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("the evidence is the account's alone: %v %v", fi, err)
	}
	lines := evidenceLines(t, files[0])
	if lines[0]["event"] != "limit" || lines[0]["exit_code"] != float64(1) || lines[0]["reset_read"] != reset.Format(time.RFC3339) {
		t.Fatalf("the first line says what was read: %+v", lines[0])
	}
	shape, _ := json.Marshal(lines[0]["shape"])
	if !strings.Contains(string(shape), "rejected") || strings.Contains(string(shape), "fake-session") {
		t.Fatalf("the shape shows the event and no identifier: %s", shape)
	}
	if lines[1]["event"] != "decided" || !strings.Contains(lines[1]["decided"].(string), "reset") || lines[1]["next_look"] != reset.Format(time.RFC3339) {
		t.Fatalf("the second line says what the loop decided: %+v", lines[1])
	}
	if last := lines[len(lines)-1]; last["event"] != "resumed" {
		t.Fatalf("the last line says when it resumed: %+v", lines)
	}
	if !strings.Contains(w.said(), filepath.Base(files[0])) {
		t.Fatalf("the run names the file:\n%s", w.said())
	}
}

// TestAProbeIsAddedToTheEvidence: each probe's answer is on the record.
func TestAProbeIsAddedToTheEvidence(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	o := w.bounded(t, 30*24*time.Hour)
	probes := 0
	o.Probe = func(context.Context, string) (bool, *runner.Failure) {
		probes++
		return probes > 1, limitedNoReset()
	}
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatal(err)
	}
	files := evidenceFiles(t, w.repo.Root(), "limit")
	if len(files) != 1 {
		t.Fatalf("one evidence file: %v", files)
	}
	var answers []string
	for _, l := range evidenceLines(t, files[0]) {
		if l["event"] == "probe" {
			answers = append(answers, l["answer"].(string))
		}
	}
	if !slices.Equal(answers, []string{"limited", "renewed"}) {
		t.Fatalf("each probe is on the record: %v", answers)
	}
}

// TestASilentAgentIsAHangWithRecovery: an agent that writes nothing until its
// deadline cannot be told from a limit waited out inside its process, so it is
// killed, its work stays in the lane's worktree, the run retries, the lane lands,
// and what it showed is kept as a suspect, never called a limit.
func TestASilentAgentIsAHangWithRecovery(t *testing.T) {
	w := newTerminalWorld(t, "shape", "late")
	for i := range w.stages {
		if w.stages[i].Name == StageImplement {
			run := w.stages[i].Run
			w.stages[i].Run = func(c Context, l *Lane) (Outcome, error) {
				if err := os.WriteFile(filepath.Join(l.Worktree, "half-done.txt"), []byte("half\n"), 0o600); err != nil {
					return Outcome{}, err
				}
				return run(c, l)
			}
		}
	}
	o := w.bounded(t, 30*24*time.Hour)
	o.Runners.Timeout = time.Second
	var worktree string
	o.BeforeStage = func() error {
		if st, _, err := readDrain(w.repo.Root()); err == nil && len(st.Lanes) > 0 {
			if run, err := ReadState(w.repo.Root(), st.Lanes[0].RunID); err == nil && run.Lanes[0].Worktree != "" {
				worktree = run.Lanes[0].Worktree
			}
		}
		return nil
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != DrainStoppedCap || len(lanesDone(t, w.repo)) != 1 {
		t.Fatalf("the lane lands on the retry: %+v\n%s", res, w.said())
	}
	if _, err := os.Stat(filepath.Join(worktree, "half-done.txt")); err != nil {
		t.Fatalf("the kill leaves the lane's work: %v", err)
	}
	if got := evidenceFiles(t, w.repo.Root(), "limit"); len(got) != 0 {
		t.Fatalf("silence alone is never called a limit: %v", got)
	}
	files := evidenceFiles(t, w.repo.Root(), "suspect")
	if len(files) != 1 {
		t.Fatalf("the silence is kept as a suspect: %v\n%s", files, w.said())
	}
	first := evidenceLines(t, files[0])[0]
	if first["exit_code"] != float64(-1) || first["silent_for_seconds"] == nil {
		t.Fatalf("the suspect says what was seen: %+v", first)
	}
}

// TestEvidenceHoldsNoSecretOrPath: what the harness wrote goes through the
// scanner's redactor, so neither a token nor the home directory survives.
func TestEvidenceHoldsNoSecretOrPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	token := "ghp_" + strings.Repeat("a1B2c3D4e5", 4)[:36]
	var lines []string
	e := newDriveEvidence(root, func(s string) { lines = append(lines, s) })
	e.begin("limit", evidenceEntry{Event: "limit", Seen: &runner.HarnessSeen{ExitCode: 1,
		Shape:  []string{`{"result":"token ` + token + ` under ` + home + `/project"}`},
		Stderr: "mail pat@example.invalid token " + token}}, time.Unix(1791000000, 0).UTC())
	files := evidenceFiles(t, root, "limit")
	if len(files) != 1 {
		t.Fatalf("one file: %v", files)
	}
	raw, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, never := range []string{token, home, "pat@example.invalid"} {
		if strings.Contains(string(raw), never) {
			t.Errorf("the evidence holds %q:\n%s", never, raw)
		}
	}
}

// TestAHundredsOfLanesDrainFitsItsState: the state a drain keeps stays well within
// what it may read back, at a size a long run can ask for.
func TestAHundredsOfLanesDrainFitsItsState(t *testing.T) {
	st := DrainState{SchemaVersion: 1, Max: 500}
	for i := range 500 {
		id := "iss-26110500000" + strconv.Itoa(10000+i)
		st.Lanes = append(st.Lanes, DrainLane{Issue: id, RunID: "run-" + id, Outcome: DrainLaneHandedBack})
		st.HandBacks = append(st.HandBacks, DrainRoute{Issue: id, Kind: HandBackStuck, Reason: strings.Repeat("a reason of some length ", 40)})
	}
	raw, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) > maxDrainStateBytes/2 {
		t.Fatalf("five hundred lanes take %d bytes of the %d a read allows", len(raw), maxDrainStateBytes)
	}
}
