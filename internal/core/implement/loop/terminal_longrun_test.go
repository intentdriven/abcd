package loop

// terminal_longrun_test.go is a drive with a bound of the person's own
// (iss-2611050753557564): each shape a limit can take, a long outage and a stuck
// queue, all waited through until the bound, with the claim kept or backed off.
// The clock and the sleep are injected, so a wait of days takes no time.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/core/landing"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
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
	o.Probe = func(_ context.Context, role, _ string) (bool, *runner.Failure) {
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
	o.Probe = func(context.Context, string, string) (bool, *runner.Failure) {
		probes++
		fl := limitedNoReset()
		fl.ResetsAt = reset
		return false, fl
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != DrainStoppedCap {
		t.Fatalf("the lane lands after the reset: %+v, %d probes\n%s", res, probes, w.said())
	}
	if got := slept(w.sleeps); got < 48*time.Hour-time.Minute {
		t.Fatalf("the run waits for the reset the probe named: %v", got)
	}
	if probes > 1+int(48*time.Hour/terminalNamedProbeEvery) {
		t.Fatalf("a named reset is asked about at a low rate: %d probes", probes)
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
	o.Probe = func(context.Context, string, string) (bool, *runner.Failure) {
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

// TestWithNoBoundNoProbeRuns: the yes of a drive with no bound never mentioned a
// probe, so none runs: the lane's own launch finds out, as it did, the waits grow
// as they did, and the run ends at eight hours.
func TestWithNoBoundNoProbeRuns(t *testing.T) {
	w := newTerminalWorld(t, append([]string{"shape"}, repeated("limited", 60)...)...)
	w.realWorktree()
	o := w.options(t)
	o.Drain.Max = 1
	w.guardSleeps(t, &o)
	probes := 0
	o.Probe = func(context.Context, string, string) (bool, *runner.Failure) {
		probes++
		return false, limitedNoReset()
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalLimited || probes != 0 || w.launches(t) < 4 {
		t.Fatalf("the lane's launches find out, and no probe runs: %+v, %d probes, %d launches\n%s", res, probes, w.launches(t), w.said())
	}
	want := []time.Duration{time.Minute, 5 * time.Minute, 10 * time.Minute, time.Hour}
	if !slices.Equal(w.sleeps[:len(want)], want) {
		t.Fatalf("the waits are as they were: %v", w.sleeps)
	}
}

// TestAnImplausibleResetIsTreatedAsNone: a reset a year off is a misread field,
// not a wait, so it is probed like a limit that named none.
func TestAnImplausibleResetIsTreatedAsNone(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	w.resetIn(t, 400*24*time.Hour)
	o := w.bounded(t, 30*24*time.Hour)
	o.Probe = func(context.Context, string, string) (bool, *runner.Failure) { return true, nil }
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
	o.Options.NetworkProbe = func() (bool, string) { return false, "no answer" }
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
		wallSleep(context.Background(), time.Hour, nil, nil, now, looks)
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
	o.Probe = func(context.Context, string, string) (bool, *runner.Failure) { return true, nil }
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
	o.Probe = func(context.Context, string, string) (bool, *runner.Failure) {
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

// TestAHundredsOfLanesDrainFitsItsState: a drain of five hundred lanes, each judged
// and handed back at length, goes through the state's own writer and reader, inside
// what a read allows.
func TestAHundredsOfLanesDrainFitsItsState(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	drainToLane(t, repo, Options{}, DrainOptions{})
	st, _, err := readDrain(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	now := st.UpdatedAt
	st.Max = 500
	st.Lanes, st.HandBacks, st.Judgements = nil, nil, nil
	reason := strings.Repeat("a reason of some length ", 40)
	for i := range 500 {
		id := "iss-" + strconv.Itoa(2611050000000000+i)
		st.Lanes = append(st.Lanes, DrainLane{Issue: id, RunID: "run-" + strconv.Itoa(2611050000000000+i), OpenedAt: now, Outcome: DrainLaneHandedBack})
		st.HandBacks = append(st.HandBacks, DrainRoute{Issue: id, From: "lane", Kind: HandBackStuck, Route: RouteHome, Home: "a home", Reason: reason, Wrote: "nothing", At: &now})
		st.Judgements = append(st.Judgements, DrainJudgement{Issue: id, RemedySHA256: strings.Repeat("a", 64), Answer: JudgementNo, Reason: reason, Applied: true, At: now})
	}
	if err := writeDrain(repo.Root(), st); err != nil {
		t.Fatal(err)
	}
	got, _, err := readDrain(repo.Root())
	if err != nil {
		t.Fatalf("five hundred lanes do not read back: %v", err)
	}
	if len(got.Lanes) != 500 || len(got.Judgements) != 500 || len(got.HandBacks) != 500 {
		t.Fatalf("what was written is what is read: %d lanes, %d judgements, %d hand-backs", len(got.Lanes), len(got.Judgements), len(got.HandBacks))
	}
	raw, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) > maxDrainStateBytes/2 {
		t.Fatalf("five hundred lanes take %d bytes of the %d a read allows", len(raw), maxDrainStateBytes)
	}
}

// TestAJudgeThatCannotRunHoldsNothingBackUnderABound: a judge whose launch failed,
// or was killed, says nothing of the issue, so under a bound nothing is held back,
// the ledger is not emptied, and the run waits on the outage's schedule.
func TestAJudgeThatCannotRunHoldsNothingBackUnderABound(t *testing.T) {
	for _, bad := range []string{"none", "late"} {
		t.Run(bad, func(t *testing.T) {
			w := newTerminalWorld(t, repeated(bad, 60)...)
			o := w.bounded(t, 10*time.Hour)
			o.Drain.JudgeTimeout = 200 * time.Millisecond
			res, err := RunTerminal(context.Background(), w.repo.Root(), o)
			if err != nil {
				t.Fatal(err)
			}
			if res.Ended != TerminalUntil || strings.Contains(w.said(), "holding") {
				t.Fatalf("the outage waits to the bound and holds nothing back: %+v\n%s", res, w.said())
			}
			st, _, _ := readDrain(w.repo.Root())
			if len(st.Skips) != 0 || st.Stopped != "" {
				t.Fatalf("the drain is left as it was: %+v", st)
			}
		})
	}
}

// TestAJudgeThatRanAndAnsweredBadlyIsStillHeldBackUnderABound: it ran, so the fault
// is the issue's, and the run goes on to other work.
func TestAJudgeThatRanAndAnsweredBadlyIsStillHeldBackUnderABound(t *testing.T) {
	w := newTerminalWorld(t, repeated("noshape", terminalJudgeAsks)...)
	o := w.bounded(t, 10*time.Hour)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != DrainStoppedCap || !strings.Contains(w.said(), "holding") {
		t.Fatalf("the issue is held back and the run goes on: %+v\n%s", res, w.said())
	}
}

// TestAFailedPushWhileReservingWaitsUnderABound: origin refusing a push mid-move may
// pass, so under a bound the run waits and carries on when it does; with no bound it
// refuses, as it did.
func TestAFailedPushWhileReservingWaitsUnderABound(t *testing.T) {
	gone := func(w *terminalWorld) {
		w.repo.Git("remote", "set-url", "--push", "origin", filepath.Join(t.TempDir(), "gone"))
	}
	w := newTerminalWorld(t)
	gone(w)
	o := w.bounded(t, 30*24*time.Hour)
	waited := time.Duration(0)
	w.onSleep = func(d time.Duration) {
		if waited += d; waited >= 2*time.Hour {
			w.repo.Git("remote", "set-url", "--push", "origin", w.bare)
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatalf("a failed push is waited out: %v\n%s", err, w.said())
	}
	if res.Ended != DrainStoppedCap || waited < 2*time.Hour {
		t.Fatalf("the run carries on once the push goes through: %+v, waited %v\n%s", res, waited, w.said())
	}

	w = newTerminalWorld(t)
	gone(w)
	o = w.options(t)
	res, err = RunTerminal(context.Background(), w.repo.Root(), o)
	if err == nil || res.Ended != TerminalRefused {
		t.Fatalf("with no bound the move refuses, as it did: %+v %v", res, err)
	}
}

// noProbeAnswers is a probe that gets nothing back.
func noProbeAnswers(n *int) func(context.Context, string, string) (bool, *runner.Failure) {
	return func(context.Context, string, string) (bool, *runner.Failure) {
		*n++
		return false, &runner.Failure{Runner: runner.Claude, Reason: runner.ReasonFailed, Detail: "no answer"}
	}
}

// probeAnswers is a probe that shows the account is alive.
func probeAnswers(n *int) func(context.Context, string, string) (bool, *runner.Failure) {
	return func(context.Context, string, string) (bool, *runner.Failure) {
		*n++
		return true, nil
	}
}

// poisonRun drives one lane through bad launches of turn, then good ones, under a
// bound, and answers what came of it.
func poisonRun(t *testing.T, turn string, n int, probe func(*int) func(context.Context, string, string) (bool, *runner.Failure)) (*terminalWorld, []DrainRoute, int) {
	t.Helper()
	w := newTerminalWorld(t, append([]string{"shape"}, repeated(turn, n)...)...)
	o := w.bounded(t, 30*24*time.Hour)
	o.Drain.Max = 2
	o.Runners.Timeout = time.Second
	probes := 0
	if probe != nil {
		o.Probe = probe(&probes)
	}
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatal(err)
	}
	st, _, _ := readDrain(w.repo.Root())
	return w, st.HandBacks, probes
}

// TestAPoisonLaneGoesBackAndTheRunGoesOn: launches that spend and fail, with the
// account alive to answer a probe, are the work's; after a few the lane goes back as
// stuck, and the next lane lands.
func TestAPoisonLaneGoesBackAndTheRunGoesOn(t *testing.T) {
	w, back, probes := poisonRun(t, "spends", terminalPoisonLaunches, probeAnswers)
	if len(back) != 1 || back[0].Kind != HandBackStuck || probes != terminalPoisonLaunches {
		t.Fatalf("the lane goes back as stuck: %+v, %d probes\n%s", back, probes, w.said())
	}
	if len(lanesDone(t, w.repo)) != 1 {
		t.Fatalf("the next lane lands:\n%s", w.said())
	}
}

// TestSpendWithNoProbeAnsweredHasACeiling: launches that keep spending with no probe
// answering would otherwise relaunch until the bound, so at six in a row the lane goes
// back; a launch started with the service unreachable was seen to spend nothing on 2.1.318,
// so an outage of that kind does not get there.
func TestSpendWithNoProbeAnsweredHasACeiling(t *testing.T) {
	w, back, probes := poisonRun(t, "spends", terminalSpentUnanswered, noProbeAnswers)
	if len(back) != 1 || back[0].Kind != HandBackStuck || probes != terminalSpentUnanswered {
		t.Fatalf("the lane goes back at the ceiling: %+v, %d probes\n%s", back, probes, w.said())
	}
	if len(lanesDone(t, w.repo)) != 1 {
		t.Fatalf("the next lane lands:\n%s", w.said())
	}
	w, back, _ = poisonRun(t, "spends", terminalSpentUnanswered-1, noProbeAnswers)
	if len(back) != 0 {
		t.Fatalf("below the ceiling nothing goes back: %+v\n%s", back, w.said())
	}
	w, back, _ = poisonRun(t, "retries", 2*terminalSpentUnanswered, noProbeAnswers)
	if len(back) != 0 {
		t.Fatalf("an outage spends nothing and never gets there: %+v\n%s", back, w.said())
	}
}

// TestACrashAfterInitCountsOnlyIfAProbeAnswers: the init event comes before any
// contact with the model, so a launch that got that far and failed says nothing; a
// probe answering shows the account is alive, which leaves the work.
func TestACrashAfterInitCountsOnlyIfAProbeAnswers(t *testing.T) {
	w, back, probes := poisonRun(t, "crashes", terminalPoisonLaunches, probeAnswers)
	if len(back) != 1 || back[0].Kind != HandBackStuck || probes != terminalPoisonLaunches {
		t.Fatalf("a probe answering counts each: %+v, %d probes\n%s", back, probes, w.said())
	}
	w, back, probes = poisonRun(t, "crashes", 2*terminalPoisonLaunches, noProbeAnswers)
	if len(back) != 0 || probes == 0 {
		t.Fatalf("a probe unanswered counts nothing: %+v, %d probes\n%s", back, probes, w.said())
	}
	if w, back, _ = poisonRun(t, "crashes", 2*terminalPoisonLaunches, nil); len(back) != 0 {
		t.Fatalf("with no probe nothing is counted: %+v\n%s", back, w.said())
	}
}

// TestARejectedProbeCountsNothing: the same limit that may have failed the launch.
func TestARejectedProbeCountsNothing(t *testing.T) {
	w, back, probes := poisonRun(t, "crashes", 2*terminalPoisonLaunches, func(n *int) func(context.Context, string, string) (bool, *runner.Failure) {
		return func(context.Context, string, string) (bool, *runner.Failure) {
			*n++
			return false, limitedNoReset()
		}
	})
	if len(back) != 0 || probes == 0 {
		t.Fatalf("%+v, %d probes\n%s", back, probes, w.said())
	}
}

// TestALaunchStuckRetryingIsAnOutage: it says so, costs no probe, and is never counted
// against its lane, a probe that would answer included.
func TestALaunchStuckRetryingIsAnOutage(t *testing.T) {
	w, back, probes := poisonRun(t, "retries", 2*terminalPoisonLaunches, probeAnswers)
	if len(back) != 0 || probes != 0 {
		t.Fatalf("nothing counts, and nothing is asked: %+v, %d probes\n%s", back, probes, w.said())
	}
	if !strings.Contains(w.said(), "retrying the model service") {
		t.Fatalf("the wait names it:\n%s", w.said())
	}
}

// TestAnAPIStatusThatIsTheAccountsCountsNothing: a result naming the service's own
// failure says nothing of the work.
func TestAnAPIStatusThatIsTheAccountsCountsNothing(t *testing.T) {
	w, back, probes := poisonRun(t, "apifails", 2*terminalPoisonLaunches, probeAnswers)
	if len(back) != 0 || probes != 0 {
		t.Fatalf("%+v, %d probes\n%s", back, probes, w.said())
	}
}

// TestLaunchesThatNeverReachTheModelStayOnTheOutageWait: no stream at all, and no
// account alive to say otherwise.
func TestLaunchesThatNeverReachTheModelStayOnTheOutageWait(t *testing.T) {
	w, back, _ := poisonRun(t, "none", 2*terminalPoisonLaunches, noProbeAnswers)
	if len(back) != 0 || len(lanesDone(t, w.repo)) == 0 {
		t.Fatalf("the lane stays and lands: %+v\n%s", back, w.said())
	}
}

// TestASilentLaunchCountsOnlyIfAProbeAnswers: a kill with nothing spent may be a hang
// or a limit waited out inside it: while the account answers nothing it is the
// outage's, and when the account answers the work is what hangs.
func TestASilentLaunchCountsOnlyIfAProbeAnswers(t *testing.T) {
	w, back, probes := poisonRun(t, "late", 2*terminalPoisonLaunches, noProbeAnswers)
	if len(back) != 0 || probes == 0 {
		t.Fatalf("a silent launch with no answer counts nothing: %+v, %d probes\n%s", back, probes, w.said())
	}
	w, back, probes = poisonRun(t, "late", terminalPoisonLaunches, probeAnswers)
	if len(back) != 1 || probes != terminalPoisonLaunches {
		t.Fatalf("with the account alive it goes back: %+v, %d probes\n%s", back, probes, w.said())
	}
}

// judgeRun drives judges of turn under a bound, and answers what came of it.
func judgeRun(t *testing.T, turn string, probe func(*int) func(context.Context, string, string) (bool, *runner.Failure)) (*terminalWorld, TerminalResult, int) {
	t.Helper()
	w := newTerminalWorld(t, repeated(turn, 40)...)
	o := w.bounded(t, 10*time.Hour)
	o.Drain.JudgeTimeout = 500 * time.Millisecond
	probes := 0
	if probe != nil {
		o.Probe = probe(&probes)
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	return w, res, probes
}

// TestAJudgeThatSpentCountsIfAProbeAnswersOrAtTheCeiling: as a lane's launch does.
func TestAJudgeThatSpentCountsIfAProbeAnswersOrAtTheCeiling(t *testing.T) {
	w, _, probes := judgeRun(t, "spends", probeAnswers)
	if !strings.Contains(w.said(), "holding") || probes < terminalJudgeAsks {
		t.Fatalf("it holds the issue back as the engine counts: %d probes\n%s", probes, w.said())
	}
	w, _, probes = judgeRun(t, "spends", noProbeAnswers)
	if !strings.Contains(w.said(), "holding") || probes < terminalSpentUnanswered {
		t.Fatalf("with no probe answered it holds the issue back at the ceiling: %d probes\n%s", probes, w.said())
	}
}

// TestAnAccountWideJudgeFailureDoesNotEmptyTheDrain: exit 0 and nothing spent: while
// no probe answers nothing is held back, and when one does the fault is the issue's.
func TestAnAccountWideJudgeFailureDoesNotEmptyTheDrain(t *testing.T) {
	w, res, _ := judgeRun(t, "accountfails", noProbeAnswers)
	if res.Ended != TerminalUntil || strings.Contains(w.said(), "holding") {
		t.Fatalf("the drain waits and holds nothing back: %+v\n%s", res, w.said())
	}
	w, _, probes := judgeRun(t, "accountfails", probeAnswers)
	if !strings.Contains(w.said(), "holding") || probes == 0 {
		t.Fatalf("a probe answering holds it back: %d probes\n%s", probes, w.said())
	}
}

// TestAJudgeStuckRetryingHoldsNothingBack: an outage, with nothing to ask.
func TestAJudgeStuckRetryingHoldsNothingBack(t *testing.T) {
	w, res, probes := judgeRun(t, "retries", probeAnswers)
	if res.Ended != TerminalUntil || strings.Contains(w.said(), "holding") || probes != 0 {
		t.Fatalf("%+v, %d probes\n%s", res, probes, w.said())
	}
}

// TestARunsFirstLaneStepChecksTheClaim: a restart has waited for as long as it was
// down, so the check is owed from the start.
func TestARunsFirstLaneStepChecksTheClaim(t *testing.T) {
	root, bare, runID, c := laneForPeerCheck(t)
	c.t = c.t.Add(3 * time.Hour)
	q := otherCopy(t, bare, quinnEmail)
	run, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := claim.Reserve(q, claim.Request{Record: run.Key, Footprint: []string{"elsewhere.md"}, Lease: 24 * time.Hour}, as(t, q, claim.Options{Now: c.now})); err != nil {
		t.Fatal(err)
	}
	if tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}}); !tm.peerTook(runID) {
		t.Fatal("the first step finds the peer")
	}
}

// TestAnOutageAtTheProofWaitsOnTheScheduleAndProvesAgain: a launch that came to no
// verdict on any check is waited through, not refused.
func TestAnOutageAtTheProofWaitsOnTheScheduleAndProvesAgain(t *testing.T) {
	c := &clock{t: time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)}
	var slept []time.Duration
	o := TerminalOptions{Options: Options{Now: c.now}, Until: c.t.Add(30 * 24 * time.Hour), Stop: make(chan struct{}),
		Sleep: func(_ context.Context, d time.Duration) { slept = append(slept, d); c.t = c.t.Add(d) }}
	proofs := 0
	ended, err := WaitOutProof(context.Background(), t.TempDir(), o, &runner.NoVerdictError{Detail: "no verdict"}, func(context.Context) error {
		if proofs++; proofs < 4 {
			return &runner.NoVerdictError{Detail: "no verdict"}
		}
		return nil
	})
	if err != nil || ended != "" || proofs != 4 {
		t.Fatalf("it proves again until it can: %q %v, %d proofs", ended, err, proofs)
	}
	want := []time.Duration{TerminalWaitFloor, TerminalWaitFloor, time.Minute, 5 * time.Minute}
	if !slices.Equal(slept, want) {
		t.Fatalf("the waits are the outage's: %v", slept)
	}
}

// TestAVerdictAtTheProofEndsTheWait: a check that does not hold is refused at once.
func TestAVerdictAtTheProofEndsTheWait(t *testing.T) {
	c := &clock{t: time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)}
	o := TerminalOptions{Options: Options{Now: c.now}, Until: c.t.Add(30 * 24 * time.Hour), Stop: make(chan struct{}),
		Sleep: func(_ context.Context, d time.Duration) { c.t = c.t.Add(d) }}
	verdict := errors.New("a check does not hold")
	ended, err := WaitOutProof(context.Background(), t.TempDir(), o, &runner.NoVerdictError{Detail: "no verdict"}, func(context.Context) error { return verdict })
	if !errors.Is(err, verdict) || ended != "" {
		t.Fatalf("%q %v", ended, err)
	}
}

// TestAProbeAtTheProofHoldsToTheSettings: probing during a wait at the proof is a
// launch like any other, so the check before a stage comes first.
func TestAProbeAtTheProofHoldsToTheSettings(t *testing.T) {
	c := &clock{t: time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)}
	o := TerminalOptions{Options: Options{Now: c.now}, Until: c.t.Add(30 * 24 * time.Hour), Stop: make(chan struct{}),
		Sleep:       func(_ context.Context, d time.Duration) { c.t = c.t.Add(d) },
		BeforeStage: func() error { return errors.New("the settings differ") },
		Probe: func(context.Context, string, string) (bool, *runner.Failure) {
			t.Fatal("no probe starts on settings that differ")
			return false, nil
		}}
	ended, err := WaitOutProof(context.Background(), t.TempDir(), o, &runner.LimitError{}, func(context.Context) error {
		t.Fatal("no proof runs either")
		return nil
	})
	if err != nil || ended != TerminalWall {
		t.Fatalf("the settings end it: %q %v", ended, err)
	}
}

// TestWithNoBoundNothingIsCountedAgainstALane: an attended run behaves as it did.
func TestWithNoBoundNothingIsCountedAgainstALane(t *testing.T) {
	w := newTerminalWorld(t, append([]string{"shape"}, repeated("crashes", 2*terminalPoisonLaunches)...)...)
	o := w.options(t)
	o.Drain.Max = 1
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	st, _, _ := readDrain(w.repo.Root())
	if res.Ended != DrainStoppedCap || len(st.HandBacks) != 0 {
		t.Fatalf("the lane stays and lands: %+v %+v\n%s", res, st.HandBacks, w.said())
	}
}

// TestAParkedLaneIsRenewedThroughALongWait: the wait renews a lane parked at its open
// pull request at half its lease, having read the forge; one whose pull request is
// closed it leaves alone; and one not yet due costs no forge call.
func TestAParkedLaneIsRenewedThroughALongWait(t *testing.T) {
	f, l := parkDrainLane(t)
	ref := claim.RefPrefix + secondIssue
	fenced := originReservation(t, f.bare, ref)
	at := fenced.claimed
	o := TerminalOptions{Options: Options{Now: func() time.Time { return at }}}
	tm := newTerminal(f.repo.Root(), o)

	listings := f.parkedListings(t)
	tm.renewHeld()
	if f.parkedListings(t) != listings {
		t.Fatal("a lane not yet due costs no forge call")
	}

	f.listPRs(t, stubPR{Number: 7, HeadRefName: l.Branch, State: "CLOSED"})
	at = fenced.claimed.Add(12*time.Hour + 30*time.Minute)
	tm.renewHeld()
	if tip := originReservation(t, f.bare, ref); tip.commit != fenced.commit {
		t.Fatalf("a closed pull request's lane is not renewed: %+v", tip)
	}

	f.listPRs(t, stubPR{Number: 7, HeadRefName: l.Branch, State: "OPEN", Mergeable: "MERGEABLE", MergeStateStatus: "BLOCKED"})
	tm.renewHeld()
	if tip := originReservation(t, f.bare, ref); tip.parent != fenced.commit || tip.released {
		t.Fatalf("an open pull request's lane is renewed: fenced %+v, now %+v", fenced, tip)
	}
}

// laneForPeerCheck opens one lane on a real remote, for the check after a wait.
func laneForPeerCheck(t *testing.T) (root, bare, runID string, c *clock) {
	t.Helper()
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	c = &clock{t: time.Now().UTC().Truncate(time.Second)}
	first := drainToLane(t, repo, Options{Now: c.now}, DrainOptions{})
	if first.Lane == nil {
		t.Fatalf("the drain opens a lane: %+v", first)
	}
	return repo.Root(), bare, first.Lane.RunID, c
}

// TestAPeerCheckNotMadeIsStillOwed: a failed fetch leaves it owed; one that reads the
// view clears it.
func TestAPeerCheckNotMadeIsStillOwed(t *testing.T) {
	root, bare, runID, c := laneForPeerCheck(t)
	tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}})
	tm.waited = 2 * time.Hour
	gitIn(t, root, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "gone"))
	if tm.peerTook(runID) || tm.waited != 2*time.Hour {
		t.Fatalf("a check that could not be made stays owed: %v", tm.waited)
	}
	gitIn(t, root, "remote", "set-url", "origin", bare)
	if tm.peerTook(runID) || tm.waited != 0 {
		t.Fatalf("a check that was made is settled: %v", tm.waited)
	}
}

// TestAReleasedTipStopsTheLane: someone released the lane's reservation during the
// wait, so what it was held for has changed, and the lane goes no further.
func TestAReleasedTipStopsTheLane(t *testing.T) {
	root, _, runID, c := laneForPeerCheck(t)
	run, err := ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReleaseReservation(root, run.Key, Options{Now: c.now}); err != nil {
		t.Fatal(err)
	}
	tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}})
	tm.waited = 2 * time.Hour
	if !tm.peerTook(runID) {
		t.Fatal("a released tip stops the lane")
	}
	run, err = ReadState(root, runID)
	if err != nil {
		t.Fatal(err)
	}
	if i := run.current(); i < 0 || run.Lanes[i].Refused == nil || run.Lanes[i].Refused.Final == "" {
		t.Fatalf("the refusal cannot change, so the next move hands it back: %+v", run.Lanes)
	}
}

// TestShortWaitsAddUpToAPeerCheck: many short waits are as long as one long one.
func TestShortWaitsAddUpToAPeerCheck(t *testing.T) {
	c := &clock{t: time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)}
	tm, _, _ := bareTerminal(t, c, time.Time{})
	tm.waited = 0
	for range 120 {
		tm.sleep(context.Background(), TerminalWaitFloor, "a short wait")
	}
	if tm.waited != time.Hour {
		t.Fatalf("an hour of short waits is an hour: %v", tm.waited)
	}
}

// TestANamedResetIsAskedAboutAtALowRate: the limit holds, so each probe is rejected;
// the run asks every few hours, starts no lane's agent, and lands after the reset.
func TestANamedResetIsAskedAboutAtALowRate(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	reset := w.resetIn(t, 72*time.Hour)
	o := w.bounded(t, 30*24*time.Hour)
	probes := 0
	o.Probe = func(context.Context, string, string) (bool, *runner.Failure) {
		probes++
		fl := limitedNoReset()
		fl.ResetsAt = reset
		return false, fl
	}
	w.onSleep = func(time.Duration) {
		if n := w.launches(t); w.clock.t.Before(reset) && n != 2 {
			t.Fatalf("an agent started during the wait: %d", n)
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != DrainStoppedCap {
		t.Fatalf("the lane lands after the reset: %+v %v\n%s", res, err, w.said())
	}
	if want := int(72*time.Hour/terminalNamedProbeEvery) - 1; probes < want-1 || probes > want+1 {
		t.Fatalf("about %d probes, got %d", want, probes)
	}
	if got := slept(w.sleeps); got < 72*time.Hour-time.Minute {
		t.Fatalf("the run waits for the reset: %v", got)
	}
}

// TestAFalseLimitEndsAtTheFirstAnswer: a limit a probe gets past was no limit, so the
// run does not sit out the reset it named.
func TestAFalseLimitEndsAtTheFirstAnswer(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	reset := w.resetIn(t, 72*time.Hour)
	o := w.bounded(t, 30*24*time.Hour)
	probes := 0
	o.Probe = func(context.Context, string, string) (bool, *runner.Failure) {
		probes++
		if probes == 1 {
			fl := limitedNoReset()
			fl.ResetsAt = reset
			return false, fl
		}
		return true, nil
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != DrainStoppedCap {
		t.Fatalf("the lane lands: %+v %v\n%s", res, err, w.said())
	}
	if got := slept(w.sleeps); got > 2*terminalNamedProbeEvery+time.Hour {
		t.Fatalf("the run does not sit the reset out: %v", got)
	}
}

// TestAFarResetIsDoubtedSooner: the first look at a reset this far off comes early.
func TestAFarResetIsDoubtedSooner(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.realWorktree()
	w.resetIn(t, terminalFarReset+24*time.Hour)
	o := w.bounded(t, 30*24*time.Hour)
	o.Probe = func(context.Context, string, string) (bool, *runner.Failure) { return true, nil }
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != DrainStoppedCap {
		t.Fatalf("the lane lands: %+v %v\n%s", res, err, w.said())
	}
	if got := slept(w.sleeps); got > terminalFarResetFirstLook+time.Minute {
		t.Fatalf("the first look comes early: %v", got)
	}
}

// TestAProbeThatSaysLittleLeavesTheNamedResetStanding: a rejection naming no reset,
// and a probe with no answer, each leave the run waiting for the time it was told,
// with no lane launched to find out.
func TestAProbeThatSaysLittleLeavesTheNamedResetStanding(t *testing.T) {
	for name, fl := range map[string]*runner.Failure{
		"rejected":  limitedNoReset(),
		"no answer": {Runner: runner.Claude, Reason: runner.ReasonFailed, Detail: "no answer"},
	} {
		t.Run(name, func(t *testing.T) {
			w := newTerminalWorld(t, "shape", "limited")
			w.realWorktree()
			reset := w.resetIn(t, 24*time.Hour)
			o := w.bounded(t, 30*24*time.Hour)
			o.Probe = func(context.Context, string, string) (bool, *runner.Failure) { return false, fl }
			w.onSleep = func(time.Duration) {
				if n := w.launches(t); w.clock.t.Before(reset) && n != 2 {
					t.Fatalf("an agent started during the wait: %d", n)
				}
			}
			res, err := RunTerminal(context.Background(), w.repo.Root(), o)
			if err != nil || res.Ended != DrainStoppedCap {
				t.Fatalf("the lane lands: %+v %v\n%s", res, err, w.said())
			}
			if got := slept(w.sleeps); got < 24*time.Hour-time.Minute {
				t.Fatalf("the named reset stands: %v", got)
			}
		})
	}
}

// TestTheProbeAsksWithTheLaunchesModel: a limit may be one model's, so the probe asks
// on the model of the launch that met it.
func TestTheProbeAsksWithTheLaunchesModel(t *testing.T) {
	c := &clock{t: time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)}
	tm, _, _ := bareTerminal(t, c, c.t.Add(48*time.Hour))
	asked := ""
	tm.o.Probe = func(_ context.Context, _, model string) (bool, *runner.Failure) {
		asked = model
		return true, nil
	}
	fl := limitedNoReset()
	fl.Model = "anthropic/a-model"
	tm.noteFailure(RoleImplementer, fl)
	if ended := tm.limited(context.Background(), RoleImplementer, time.Time{}); ended != "" {
		t.Fatalf("the run goes on: %q", ended)
	}
	if asked != "anthropic/a-model" {
		t.Fatalf("the probe asked on %q", asked)
	}
}

// TestALimitAtTheProofIsWaitedOutAndProvedAgain: the proof's own limit waits as any
// agent's does, the run proving again when it has passed.
func TestALimitAtTheProofIsWaitedOutAndProvedAgain(t *testing.T) {
	c := &clock{t: time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)}
	var slept []time.Duration
	o := TerminalOptions{Options: Options{Now: c.now}, Until: c.t.Add(30 * 24 * time.Hour), Stop: make(chan struct{}),
		Sleep: func(_ context.Context, d time.Duration) { slept = append(slept, d); c.t = c.t.Add(d) }}
	root := t.TempDir()
	reset := c.t.Add(3 * time.Hour)
	proofs := 0
	ended, err := WaitOutProof(context.Background(), root, o, &runner.LimitError{ResetsAt: reset}, func(context.Context) error {
		proofs++
		if c.t.Before(reset) {
			t.Fatalf("a proof ran before the reset: %v", c.t)
		}
		return nil
	})
	if err != nil || ended != "" || proofs != 1 {
		t.Fatalf("one proof after the reset: %q %v, %d proofs", ended, err, proofs)
	}
}

// TestABoundPassingAtTheProofEndsTheWait: nothing starts.
func TestABoundPassingAtTheProofEndsTheWait(t *testing.T) {
	c := &clock{t: time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)}
	o := TerminalOptions{Options: Options{Now: c.now}, Until: c.t.Add(time.Hour), Stop: make(chan struct{}),
		Sleep: func(_ context.Context, d time.Duration) { c.t = c.t.Add(d) }}
	ended, err := WaitOutProof(context.Background(), t.TempDir(), o, &runner.LimitError{ResetsAt: c.t.Add(3 * time.Hour)}, func(context.Context) error {
		t.Fatal("no proof runs past the bound")
		return nil
	})
	if err != nil || ended != TerminalUntil {
		t.Fatalf("the bound ends it: %q %v", ended, err)
	}
}

// TestAnAgentsCallSaysItStartedAndStillRuns: a long stage and a hang look alike from
// outside, so a call says it began and, at least hourly, that it is still going.
func TestAnAgentsCallSaysItStartedAndStillRuns(t *testing.T) {
	c := &clock{t: time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)}
	tm, _, lines := bareTerminal(t, c, time.Time{})
	tm.startCall("the implementer")
	if !strings.Contains(strings.Join(*lines, "\n"), "the implementer") {
		t.Fatalf("the start says so: %q", *lines)
	}
	said := len(*lines)
	tm.sayCallStill()
	if len(*lines) != said {
		t.Fatalf("nothing new to say yet: %q", *lines)
	}
	c.t = c.t.Add(terminalCallSayEvery)
	tm.sayCallStill()
	if len(*lines) != said+1 || !strings.Contains((*lines)[said], "30m") {
		t.Fatalf("it says it is still going, and for how long: %q", *lines)
	}
	tm.endCall()
	c.t = c.t.Add(2 * terminalCallSayEvery)
	tm.sayCallStill()
	if len(*lines) != said+1 {
		t.Fatalf("a call over says nothing more: %q", *lines)
	}
}

// TestARunSaysWhenAnAgentStarts: through a real drive, the judge's and the
// implementer's both.
func TestARunSaysWhenAnAgentStarts(t *testing.T) {
	w := newTerminalWorld(t)
	o := w.options(t)
	o.Drain.Max = 1
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{RoleDrainJudge, RoleImplementer} {
		if !strings.Contains(w.said(), "starting the "+role) {
			t.Errorf("no line says the %s started:\n%s", role, w.said())
		}
	}
}

// TestEvidenceIsRedactedBeforeItIsCut: a token across the place a line is cut is seen
// whole by the redactor, and what is kept stays within its bounds.
func TestEvidenceIsRedactedBeforeItIsCut(t *testing.T) {
	root := t.TempDir()
	token := "ghp_" + strings.Repeat("a1B2c3D4e5", 4)[:36]
	long := strings.Repeat("x", evidenceMaxLine-10) + " " + token + " " + strings.Repeat("y", 500)
	e := newDriveEvidence(root, func(string) {})
	e.begin("limit", evidenceEntry{Event: "limit", Seen: &runner.HarnessSeen{ExitCode: 1, Shape: []string{long},
		Stderr: strings.Repeat("z", evidenceMaxStderr-10) + " " + token + " " + strings.Repeat("w", 500)}}, time.Unix(1791000000, 0).UTC())
	files := evidenceFiles(t, root, "limit")
	if len(files) != 1 {
		t.Fatalf("one file: %v", files)
	}
	lines := evidenceLines(t, files[0])
	shape, _ := lines[0]["shape"].([]any)
	if len(shape) != 1 || len(shape[0].(string)) > evidenceMaxLine+3 {
		t.Fatalf("the line is cut: %d", len(shape[0].(string)))
	}
	if stderr, _ := lines[0]["stderr"].(string); len(stderr) > evidenceMaxStderr+3 {
		t.Fatalf("the stderr line is cut: %d", len(stderr))
	}
	raw, _ := os.ReadFile(files[0])
	if strings.Contains(string(raw), token[:12]) {
		t.Fatalf("part of the token survived the cut:\n%s", raw)
	}
}

// TestEvidenceStopsItsDetailPastItsSizeAndKeepsItsAnswers: a long wait's probes stay
// small, and each still says what it got.
func TestEvidenceStopsItsDetailPastItsSizeAndKeepsItsAnswers(t *testing.T) {
	root := t.TempDir()
	now := time.Unix(1791000000, 0).UTC()
	e := newDriveEvidence(root, func(string) {})
	seen := &runner.HarnessSeen{ExitCode: 1, Shape: []string{strings.Repeat("s", evidenceMaxLine)}}
	e.begin("limit", evidenceEntry{Event: "limit", Seen: seen}, now)
	const probes = 700
	for range probes {
		e.add(evidenceEntry{Event: "probe", Answer: "limited", Seen: seen}, now)
	}
	files := evidenceFiles(t, root, "limit")
	fi, err := os.Stat(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() > 2*evidenceMaxBytes {
		t.Fatalf("the file grew to %d bytes", fi.Size())
	}
	lines := evidenceLines(t, files[0])
	if len(lines) != probes+1 || lines[len(lines)-1]["answer"] != "limited" || lines[len(lines)-1]["shape"] != nil {
		t.Fatalf("every answer stays, without its detail: %d lines, last %+v", len(lines), lines[len(lines)-1])
	}
}

// TestEvidenceIsPrunedAcrossRuns: the newest are kept, of each kind, and nothing else
// in the folder is touched.
func TestEvidenceIsPrunedAcrossRuns(t *testing.T) {
	root := t.TempDir()
	if err := fsutil.EnsureRealDirAll(root, evidenceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, filepath.FromSlash(evidenceDir))
	old := time.Unix(1700000000, 0).UTC()
	for i := range evidenceKeepFiles + 5 {
		name := "drive-limit-" + old.Add(time.Duration(i)*time.Hour).Format("20060102T150405Z") + ".jsonl"
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	other := filepath.Join(dir, "drain-drive-20200101T000000Z.log")
	if err := os.WriteFile(other, []byte("log\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	e := newDriveEvidence(root, func(string) {})
	e.begin("limit", evidenceEntry{Event: "limit"}, time.Unix(1791000000, 0).UTC())
	files := evidenceFiles(t, root, "limit")
	if len(files) != evidenceKeepFiles {
		t.Fatalf("%d files kept", len(files))
	}
	if !strings.Contains(files[len(files)-1], time.Unix(1791000000, 0).UTC().Format("20060102T150405Z")) {
		t.Fatalf("the newest stays: %v", files)
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatalf("nothing else is touched: %v", err)
	}
}

// TestEvidenceWritesThroughNoLinkOutOfItsFolder: a file swapped for a link does not
// carry a write elsewhere.
func TestEvidenceWritesThroughNoLinkOutOfItsFolder(t *testing.T) {
	root := t.TempDir()
	e := newDriveEvidence(root, func(string) {})
	now := time.Unix(1791000000, 0).UTC()
	e.begin("limit", evidenceEntry{Event: "limit"}, now)
	files := evidenceFiles(t, root, "limit")
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("mine\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(files[0]); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, files[0]); err != nil {
		t.Fatal(err)
	}
	e.add(evidenceEntry{Event: "probe", Answer: "limited"}, now)
	if raw, _ := os.ReadFile(outside); string(raw) != "mine\n" {
		t.Fatalf("a write went through the link: %s", raw)
	}
}

// stalledStages has the first lane's implement stage hand out await, and every later
// lane's the ordinary one.
func (w *terminalWorld) stalledStages(await func(c Context, l *Lane) *Await) {
	for i := range w.stages {
		if w.stages[i].Name != StageImplement {
			continue
		}
		run := w.stages[i].Run
		first := true
		w.stages[i].Run = func(c Context, l *Lane) (Outcome, error) {
			if first {
				first = false
				return Outcome{Await: await(c, l)}, nil
			}
			return run(c, l)
		}
	}
}

// TestALaneThatCannotMoveGoesBackUnderABound: a step that starts no agent and moves no
// stage, for ever, would hold every other issue until the bound, so after a few the
// lane goes back as stuck, and the next lands.
func TestALaneThatCannotMoveGoesBackUnderABound(t *testing.T) {
	w := newTerminalWorld(t)
	w.stalledStages(func(c Context, l *Lane) *Await {
		return &Await{Role: RoleAuditor, Brief: filepath.Join(c.RepoRoot, l.Brief), Receipt: filepath.Join(c.RepoRoot, RunRelDir, "receipt-stalled.json")}
	})
	w.assertStallGoesBack(t)
}

// TestAKeptAwaitGoesBackUnderABound: a limit whose checkpoint could not save the agent's
// work aside keeps the await for a person, here with a worktree that is no checkout, and
// no step takes it up, so the run goes on to other issues.
func TestAKeptAwaitGoesBackUnderABound(t *testing.T) {
	w := newTerminalWorld(t, "shape", "limited")
	w.assertStallGoesBack(t)
	if !strings.Contains(w.said(), "did not move") {
		t.Fatalf("the steps say they did not move:\n%s", w.said())
	}
}

// assertStallGoesBack runs two lanes under a bound: the first goes back as stuck, soon,
// and the next lands.
func (w *terminalWorld) assertStallGoesBack(t *testing.T) {
	t.Helper()
	o := w.bounded(t, 30*24*time.Hour)
	o.Drain.Max = 2
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	st, _, _ := readDrain(w.repo.Root())
	if len(st.HandBacks) != 1 || st.HandBacks[0].Kind != HandBackStuck {
		t.Fatalf("the lane goes back as stuck: %+v %+v\n%s", res, st.HandBacks, w.said())
	}
	if len(lanesDone(t, w.repo)) != 1 {
		t.Fatalf("the next lane lands: %+v\n%s", res, w.said())
	}
	if got := slept(w.sleeps); got > 4*time.Hour {
		t.Fatalf("it holds the rest up for %v", got)
	}
}

// TestALaneWhoseAgentIsStillClaimedIsNoStall: a claim left by a killed run lapses on
// its own, and a restart waits for it rather than count it against the lane.
func TestALaneWhoseAgentIsStillClaimedIsNoStall(t *testing.T) {
	w := newTerminalWorld(t)
	until := w.clock.t.Add(100 * time.Hour)
	w.stalledStages(func(c Context, l *Lane) *Await {
		return &Await{Role: RoleImplementer, Brief: filepath.Join(c.RepoRoot, l.Brief), Receipt: filepath.Join(c.RepoRoot, RunRelDir, "receipt-claimed.json"),
			Running: &RunnerClaim{Runner: runner.Claude, Session: "a-killed-run", Until: until}}
	})
	o := w.bounded(t, 10*time.Hour)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	st, _, _ := readDrain(w.repo.Root())
	if res.Ended != TerminalUntil || len(st.HandBacks) != 0 {
		t.Fatalf("the wait lasts to the bound and hands nothing back: %+v %+v\n%s", res, st.HandBacks, w.said())
	}
}

// TestWithNoBoundALaneThatCannotMoveStaysOnTheOutagesDeadline: an attended run behaves
// as it did.
func TestWithNoBoundALaneThatCannotMoveStaysOnTheOutagesDeadline(t *testing.T) {
	w := newTerminalWorld(t)
	w.stalledStages(func(c Context, l *Lane) *Await {
		return &Await{Role: RoleAuditor, Brief: filepath.Join(c.RepoRoot, l.Brief), Receipt: filepath.Join(c.RepoRoot, RunRelDir, "receipt-held.json")}
	})
	o := w.options(t)
	o.Drain.Max = 1
	w.guardSleeps(t, &o)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	st, _, _ := readDrain(w.repo.Root())
	if res.Ended != TerminalOutage || len(st.HandBacks) != 0 {
		t.Fatalf("%+v %+v", res, st.HandBacks)
	}
}

// TestAMoveEndsTheStallCount: only no-moves in a row count.
func TestAMoveEndsTheStallCount(t *testing.T) {
	c := &clock{t: time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)}
	tm, _, _ := bareTerminal(t, c, c.t.Add(time.Hour))
	tm.stalled = terminalNoMoveLimit - 1
	tm.moved()
	if tm.stalled != 0 {
		t.Fatalf("a move ends the count: %d", tm.stalled)
	}
}

// TestAForge5xxWaitsAsALostNetwork: the landing's reading of gh's and git's words: a
// contention naming the shared probe, not a refusal three of which hand it back.
func TestAForge5xxWaitsAsALostNetwork(t *testing.T) {
	for _, text := range []string{
		"HTTP 502: Bad Gateway (https://api.github.com/repos/o/r/pulls)",
		"fatal: unable to access 'https://github.com/o/r.git/': The requested URL returned error: 503",
	} {
		r := netFailure(StageLand, "lane-1", "gh pr create", errors.New(text))
		if r == nil || !r.Contention || r.outage == nil {
			t.Errorf("%q reads as %+v", text, r)
		}
	}
}

// TestEachKindOfFailureLeavesItsEvidence: the first of each, not only the run's first.
func TestEachKindOfFailureLeavesItsEvidence(t *testing.T) {
	w := newTerminalWorld(t, "shape", "none", "retries", "apifails", "none")
	o := w.bounded(t, 30*24*time.Hour)
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatal(err)
	}
	files := evidenceFiles(t, w.repo.Root(), "suspect")
	if len(files) != 3 {
		t.Fatalf("one file for each of the three kinds: %v\n%s", files, w.said())
	}
	joined := strings.Join(files, "\n")
	for _, want := range []string{"suspect-failed-", "suspect-outage-", "suspect-api-503-"} {
		if !strings.Contains(joined, want) {
			t.Errorf("no file names %s: %v", want, files)
		}
	}
}

// TestAStaleNetworkOutageIsProbedUnderABound: a drive has no session of a person's, so
// an open record would hold every agent for the whole bound; it joins under its own, and
// the probe that falls due ends it.
func TestAStaleNetworkOutageIsProbedUnderABound(t *testing.T) {
	w := newTerminalWorld(t)
	run := joinShared(t, w.repo.Root(), w.clock.now)
	if _, err := run.Outage().Record("host-a", implement.ServiceNetwork, implement.OutageTool, "lane-9", "git push"); err != nil {
		t.Fatal(err)
	}
	o := w.bounded(t, 30*24*time.Hour)
	o.Options.NetworkProbe = func() (bool, string) { return true, "origin answered" }
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != DrainStoppedCap {
		t.Fatalf("the probe ends the outage and the lane lands: %+v %v\n%s", res, err, w.said())
	}
	if cur, _ := run.Outage().Current(); cur != nil {
		t.Fatalf("the record is closed: %+v", cur)
	}
}

// TestANetworkOutageUnderABoundIsNotGivenUpOn: probes that keep failing past eight
// hours keep the record open, to be proved back, not stopped for a person.
func TestANetworkOutageUnderABoundIsNotGivenUpOn(t *testing.T) {
	w := newTerminalWorld(t)
	run := joinShared(t, w.repo.Root(), w.clock.now)
	if _, err := run.Outage().Record("host-a", implement.ServiceNetwork, implement.OutageTool, "lane-9", "git push"); err != nil {
		t.Fatal(err)
	}
	o := w.bounded(t, 24*time.Hour)
	o.Options.NetworkProbe = func() (bool, string) { return false, "no answer" }
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != TerminalUntil {
		t.Fatalf("the bound ends it: %+v %v\n%s", res, err, w.said())
	}
	cur, err := run.Outage().Current()
	if err != nil || cur == nil || cur.Status != implement.OutageOpen || len(cur.Probes) < 8 {
		t.Fatalf("the record stays open and is still probed: %+v %v", cur, err)
	}
}

// TestAStaleModelOutageIsClearedByAnAnsweredProbe: the probe is the canary, since
// nothing else would ever start an agent to find out.
func TestAStaleModelOutageIsClearedByAnAnsweredProbe(t *testing.T) {
	w := newTerminalWorld(t)
	run := joinShared(t, w.repo.Root(), w.clock.now)
	if _, err := run.Outage().Record("host-a", implement.ServiceModel, implement.OutageHost, "lane-9", "a turn"); err != nil {
		t.Fatal(err)
	}
	o := w.bounded(t, 30*24*time.Hour)
	probes := 0
	o.Probe = probeAnswers(&probes)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != DrainStoppedCap || probes == 0 {
		t.Fatalf("the probe ends the outage and the lane lands: %+v %v, %d probes\n%s", res, err, probes, w.said())
	}
}

// TestADriveLeavesTheSharedRunItJoined: its session does not outlast it.
func TestADriveLeavesTheSharedRunItJoined(t *testing.T) {
	w := newTerminalWorld(t)
	o := w.bounded(t, 30*24*time.Hour)
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatal(err)
	}
	sha, err := gitutil.RootCommit(w.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	run, err := implement.Peek(sha)
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := run.Sessions()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range sessions {
		if strings.HasPrefix(s.Session, "drive-") {
			t.Fatalf("the drive's session is still open: %+v", s)
		}
	}
}

// TestAnOutageDiscountsANoMoveOnlyIfItHoldsTheAgentThisLaneWants: a kept await wants no
// agent, so the outage is not why it cannot move.
func TestAnOutageDiscountsANoMoveOnlyIfItHoldsTheAgentThisLaneWants(t *testing.T) {
	outage := StepResult{Outage: &OutageInfo{Down: []string{implement.ServiceNetwork}}}
	wants := State{Lanes: []Lane{{ID: "lane-1", Stage: StageImplement, Awaits: []Await{{Role: RoleImplementer}}}}}
	kept := State{Lanes: []Lane{{ID: "lane-1", Stage: StageImplement, Awaits: []Await{{Role: RoleImplementer, Kept: "not saved"}}}}}
	if !outageHoldsLane(outage, wants) {
		t.Error("an agent it wants is held")
	}
	if outageHoldsLane(outage, kept) {
		t.Error("a kept await wants no agent")
	}
	if outageHoldsLane(StepResult{}, wants) {
		t.Error("no outage holds nothing")
	}
}

// TestAStepThatStartedAnAgentIsNoStall: a silent launch, killed with its await kept,
// moves no stage; it did start an agent, so finished work is not sent back for that.
func TestAStepThatStartedAnAgentIsNoStall(t *testing.T) {
	root, _, runID, c := laneForPeerCheck(t)
	tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}, Until: c.t.Add(time.Hour)})
	tm.stalled = terminalNoMoveLimit - 1
	tm.o.Options.RunnerStarting(RoleImplementer)
	if tm.cannotMove(runID, StepResult{}) || tm.stalled != 0 {
		t.Fatalf("a step that started an agent ends the count: %d", tm.stalled)
	}
	tm.startedAgent = false
	tm.stalled = terminalNoMoveLimit - 1
	if !tm.cannotMove(runID, StepResult{}) {
		t.Fatal("one that started none reaches the limit")
	}
}

// landFailsAt makes the land stage fail on the act and the words fail answers, each
// read as a lost network.
func (w *terminalWorld) landFailsAt(fail func(l *Lane) (string, error)) {
	for i := range w.stages {
		if w.stages[i].Name == StageLand {
			w.stages[i].Run = func(c Context, l *Lane) (Outcome, error) {
				what, err := fail(l)
				if r := netFailure(StageLand, l.ID, what, err); r != nil {
					return Outcome{}, r
				}
				return Outcome{}, refuse(string(StageLand), "", l.ID, err.Error(), "settle it")
			}
		}
	}
}

// landFails makes the land stage fail on text, a network reading as a stage meets it.
func (w *terminalWorld) landFails(text string, fails *int) {
	w.landFailsAt(func(l *Lane) (string, error) {
		*fails++
		return "git push " + l.Branch, errors.New(text)
	})
}

// TestANetworkReadingThatOriginAnswersGoesBack: the same act read as a lost network for
// two hours while origin answers is no lost network, so the lane goes back; not sooner,
// since on a poor link or in a partial incident origin answers while the act cannot work.
func TestANetworkReadingThatOriginAnswersGoesBack(t *testing.T) {
	for name, text := range map[string]string{
		"git": "error: RPC failed; HTTP 500 curl 22 The requested URL returned error: 500",
		"eof": "fatal: early EOF",
	} {
		t.Run(name, func(t *testing.T) {
			w := newTerminalWorld(t)
			fails := 0
			var first, last time.Time
			w.landFailsAt(func(l *Lane) (string, error) {
				fails++
				if first.IsZero() {
					first = w.clock.t
				}
				last = w.clock.t
				return "git push " + l.Branch, errors.New(text)
			})
			o := w.bounded(t, 30*24*time.Hour)
			o.Options.NetworkProbe = func() (bool, string) { return true, "origin answered" }
			if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
				t.Fatal(err)
			}
			st, _, _ := readDrain(w.repo.Root())
			if len(st.HandBacks) == 0 || st.HandBacks[0].Kind != HandBackStuck {
				t.Fatalf("the lane goes back as stuck: %+v after %d fails\n%s", st.HandBacks, fails, w.said())
			}
			if got := last.Sub(first); got < terminalNetGoesBack-terminalIdleCap {
				t.Fatalf("it went back after %v and %d fails", got, fails)
			}
		})
	}
}

// TestANetworkReadingWithOriginDownKeepsWaiting: and with origin down it is what it reads
// as, waited out until the bound.
func TestANetworkReadingWithOriginDownKeepsWaiting(t *testing.T) {
	w := newTerminalWorld(t)
	fails := 0
	w.landFails("ssh: connect to host github.com port 22: Connection refused", &fails)
	o := w.bounded(t, 10*time.Hour)
	o.Options.NetworkProbe = func() (bool, string) { return false, "no answer" }
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil {
		t.Fatal(err)
	}
	st, _, _ := readDrain(w.repo.Root())
	if res.Ended != TerminalUntil || len(st.HandBacks) != 0 {
		t.Fatalf("%+v %+v\n%s", res, st.HandBacks, w.said())
	}
}

// TestAHookThatPrintsAStatusIsNoLostNetwork: a pre-push hook's own words, through a real
// push, read as the refusal they are.
func TestAHookThatPrintsAStatusIsNoLostNetwork(t *testing.T) {
	repo, _ := reservingDrainRepo(t, "commands/drain.md")
	hook := filepath.Join(repo.Root(), ".git", "hooks", "pre-push")
	if err := os.MkdirAll(filepath.Dir(hook), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hook, []byte("#!/bin/sh\necho 'HTTP 500 from fixture' >&2\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	repo.Write("a-file.md", "a\n")
	repo.Commit("a change")
	_, err := gitutil.Net(repo.Root(), "push", "--porcelain", "origin", "HEAD:refs/heads/a-branch")
	if err == nil {
		t.Fatal("the hook refuses the push")
	}
	if !strings.Contains(err.Error(), "HTTP 500 from fixture") {
		t.Fatalf("the hook's words are in the error: %v", err)
	}
	if r := netFailure(StageLand, "lane-1", "git push", err); r != nil {
		t.Fatalf("it reads as a lost network: %+v", r)
	}
}

// TestARepeatCountsTheActNotWhatItPrinted: a line that changes each time, a port or a
// timestamp from a hook, is still the same act failing, so the clock is not restarted.
func TestARepeatCountsTheActNotWhatItPrinted(t *testing.T) {
	w := newTerminalWorld(t)
	fails := 0
	w.landFailsAt(func(l *Lane) (string, error) {
		fails++
		return "git push " + l.Branch, fmt.Errorf("dial tcp 127.0.0.1:%d: connect: connection refused", 54000+fails)
	})
	o := w.bounded(t, 30*24*time.Hour)
	o.Options.NetworkProbe = func() (bool, string) { return true, "origin answered" }
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatal(err)
	}
	st, _, _ := readDrain(w.repo.Root())
	if len(st.HandBacks) == 0 || st.HandBacks[0].Kind != HandBackStuck {
		t.Fatalf("the lane goes back as stuck: %+v after %d fails\n%s", st.HandBacks, fails, w.said())
	}
}

// TestAForgeActFollowsTheSameRule: a forge call and a push share one clock rule.
func TestAForgeActFollowsTheSameRule(t *testing.T) {
	w := newTerminalWorld(t)
	var first, last time.Time
	w.landFailsAt(func(*Lane) (string, error) {
		if first.IsZero() {
			first = w.clock.t
		}
		last = w.clock.t
		return "gh pr create", errors.New("gh pr create: exit status 1 (HTTP 502: Bad Gateway (https://api.github.com/repos/o/r/pulls))")
	})
	o := w.bounded(t, 30*24*time.Hour)
	o.Options.NetworkProbe = func() (bool, string) { return true, "origin answered" }
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatal(err)
	}
	st, _, _ := readDrain(w.repo.Root())
	if len(st.HandBacks) == 0 || st.HandBacks[0].Kind != HandBackStuck {
		t.Fatalf("after that long it goes back: %+v\n%s", st.HandBacks, w.said())
	}
	if got := last.Sub(first); got < terminalNetGoesBack-terminalIdleCap {
		t.Fatalf("it went back after %v", got)
	}
}

// TestADifferentActRestartsTheClock: the failure has moved on, so it has not lasted.
func TestADifferentActRestartsTheClock(t *testing.T) {
	w := newTerminalWorld(t)
	start := w.clock.t
	var last time.Time
	w.landFailsAt(func(*Lane) (string, error) {
		last = w.clock.t
		what := "git push a"
		if w.clock.t.Sub(start) > terminalNetGoesBack-30*time.Minute {
			what = "gh pr create"
		}
		return what, errors.New("fatal: early EOF")
	})
	o := w.bounded(t, 30*24*time.Hour)
	o.Options.NetworkProbe = func() (bool, string) { return true, "origin answered" }
	if _, err := RunTerminal(context.Background(), w.repo.Root(), o); err != nil {
		t.Fatal(err)
	}
	if got := last.Sub(start); got < 2*terminalNetGoesBack-time.Hour {
		t.Fatalf("the second act has a clock of its own: %v", got)
	}
}

// TestAMoveRestartsTheNetworkClock: a step that worked ends it.
func TestAMoveRestartsTheNetworkClock(t *testing.T) {
	c := &clock{t: time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)}
	tm, _, _ := bareTerminal(t, c, c.t.Add(time.Hour))
	tm.netKey, tm.netSince = "git push a", c.t
	tm.moved()
	if tm.netKey != "" || !tm.netSince.IsZero() {
		t.Fatalf("a move ends the clock: %q %v", tm.netKey, tm.netSince)
	}
}

// TestAForge5xxThroughTheClientWaitsAsALostNetwork: what the landing really reads, with
// the client's own wrapper around a one-line stderr.
func TestAForge5xxThroughTheClientWaitsAsALostNetwork(t *testing.T) {
	bin := t.TempDir()
	script := "#!/bin/sh\necho 'HTTP 502: Bad Gateway (https://api.github.com/repos/o/r/pulls)' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	_, err := landing.GH(t.TempDir(), "pr", "create")
	if err == nil {
		t.Fatal("the stand-in fails")
	}
	r := netFailure(StageLand, "lane-1", "gh pr create", err)
	if r == nil || !r.Contention || r.outage == nil {
		t.Fatalf("%q reads as %+v", err, r)
	}
}

// TestACanaryThatCannotLandSpendsNothing: a probe's result that could not be written
// would otherwise cost one short launch per turn of the loop until the bound.
func TestACanaryThatCannotLandSpendsNothing(t *testing.T) {
	root, _, _, c := laneForPeerCheck(t)
	run := joinShared(t, root, c.now)
	if _, err := run.Outage().Record("host-a", implement.ServiceModel, implement.OutageHost, "lane-9", "a turn"); err != nil {
		t.Fatal(err)
	}
	c.t = c.t.Add(time.Hour)
	probes := 0
	tm := newTerminal(root, TerminalOptions{Options: Options{Now: c.now}, Until: c.t.Add(time.Hour), Probe: probeAnswers(&probes)})
	tm.shared, tm.session = run, "host-a"
	if err := os.Chmod(run.Dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(run.Dir, 0o700) })
	tm.settleOutage(context.Background())
	if probes != 0 {
		t.Fatalf("a probe ran with nowhere to put its answer: %d", probes)
	}
}

// TestAGaveUpRecordIsClearedWhenItsProbesAnswer: another session on the default schedule
// can give up during a long outage, and the drive cannot probe a record in that state, so
// with no person there its own probes clear it when they answer, and the run goes on.
func TestAGaveUpRecordIsClearedWhenItsProbesAnswer(t *testing.T) {
	w := newTerminalWorld(t)
	run := joinShared(t, w.repo.Root(), w.clock.now)
	giveUp(t, run, "host-a", func(d time.Duration) { w.clock.t = w.clock.t.Add(d) })
	start := w.clock.t
	o := w.bounded(t, 30*24*time.Hour)
	o.Options.NetworkProbe = func() (bool, string) {
		return w.clock.t.Sub(start) >= 5*time.Hour, "origin"
	}
	probes := 0
	o.Probe = probeAnswers(&probes)
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != DrainStoppedCap {
		t.Fatalf("the run goes on once the probes answer: %+v %v\n%s", res, err, w.said())
	}
	if cur, _ := run.Outage().Current(); cur != nil {
		t.Fatalf("the record is cleared: %+v", cur)
	}
	if got := slept(w.sleeps); got < 5*time.Hour {
		t.Fatalf("it waits for the answer: %v", got)
	}
}

// TestAGaveUpRecordWithOriginDownStaysHeld: while the network is still out nothing is
// cleared, nothing starts, and the bound ends it.
func TestAGaveUpRecordWithOriginDownStaysHeld(t *testing.T) {
	w := newTerminalWorld(t)
	run := joinShared(t, w.repo.Root(), w.clock.now)
	giveUp(t, run, "host-a", func(d time.Duration) { w.clock.t = w.clock.t.Add(d) })
	o := w.bounded(t, 10*time.Hour)
	o.Options.NetworkProbe = func() (bool, string) { return false, "no answer" }
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != TerminalUntil || w.launches(t) != 0 {
		t.Fatalf("%+v %v, %d launches\n%s", res, err, w.launches(t), w.said())
	}
	if cur, _ := run.Outage().Current(); cur == nil || cur.Status != implement.OutageGaveUp {
		t.Fatalf("the record stands: %+v", cur)
	}
}

// TestWithNoBoundAGaveUpRecordStopsTheDrain: an attended run behaves as it did.
func TestWithNoBoundAGaveUpRecordStopsTheDrain(t *testing.T) {
	w := newTerminalWorld(t)
	run := joinShared(t, w.repo.Root(), w.clock.now)
	giveUp(t, run, "host-a", func(d time.Duration) { w.clock.t = w.clock.t.Add(d) })
	o := w.options(t)
	o.Options.NetworkProbe = func() (bool, string) { return true, "origin answered" }
	res, err := RunTerminal(context.Background(), w.repo.Root(), o)
	if err != nil || res.Ended != DrainStoppedOutage {
		t.Fatalf("%+v %v\n%s", res, err, w.said())
	}
	if cur, _ := run.Outage().Current(); cur == nil {
		t.Fatal("it clears nothing")
	}
}
