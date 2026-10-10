package loop

// outage_test.go is the loop's half of a lost connection (iss-2610080620372731):
// the lanes that reach the network or an agent wait on the run's one shared
// probe while the offline lanes move, the step runs the probe when it is due,
// and a run that gave up stops every step naming what was done and what is
// left. The clock is the fixture's, the network probe is injected, and the one
// real network call (a push to a port nothing listens on) fails at once.

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement"
	"github.com/intentdriven/abcd/internal/gittest"
)

// shared opens the shared run state for the fixture's repository on the
// fixture's clock.
func (f *parFixture) shared(t *testing.T) *implement.Run {
	t.Helper()
	run, err := implement.Open(strings.TrimSpace(f.repo.Git("rev-list", "--max-parents=0", "HEAD")))
	if err != nil {
		t.Fatal(err)
	}
	run.Now = f.clock
	return run
}

// outageNow is the shared run's outage, or the test fails.
func outageNow(t *testing.T, run *implement.Run) *implement.Outage {
	t.Helper()
	cur, err := run.Outage().Current()
	if err != nil {
		t.Fatal(err)
	}
	return cur
}

// toPush takes lane-1 of a fresh fixture to its push: implemented, its round
// passed, its landing prepared and its records committed.
func (f *parFixture) toPush(t *testing.T, laneID string) {
	t.Helper()
	f.roundPassed(t, laneID)
	for range 10 {
		l := f.lane(t, laneID)
		if l.Stage == StageLand && l.Landing != nil && l.Landing.RecordsDone && l.Landing.Pushed == "" {
			return
		}
		f.step(t)
	}
	t.Fatalf("%s never reached its push: %+v", laneID, f.lane(t, laneID))
}

// TestLandPushNetworkFailureHoldsOnlyItsLane: a push the network refuses holds
// only its lane, as a wait on the shared probe; the outage is recorded in the
// shared run (for the session whose claim names the run); a sibling lane's
// offline work moves in the same call; and the next call does not retry the
// push before the probe.
func TestLandPushNetworkFailureHoldsOnlyItsLane(t *testing.T) {
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n", Options{SubAgents: strp("4"), Session: "host-a"})
	f.stepUntil(t, "both implementers are out", func(st State) bool {
		return len(st.Lanes) == 2 && len(st.Lanes[0].Awaits) == 1 && len(st.Lanes[1].Awaits) == 1
	})
	f.implement(t, "lane-1", "one.txt")
	f.toPush(t, "lane-1")
	f.implement(t, "lane-2", "two.txt")
	l1 := f.lane(t, "lane-1")
	preflighted(t, l1, l1.HeadSHA)
	f.repo.Git("remote", "set-url", "--push", "origin", "http://127.0.0.1:1/x.git")

	res := f.step(t)
	if len(res.Blocked) != 1 || res.Blocked[0].Lane != "lane-1" || !res.Blocked[0].Contention ||
		!strings.Contains(res.Blocked[0].Reason, "waits on the shared network probe, next 09:01") {
		t.Fatalf("the push the network refused holds only lane-1, waiting on the shared probe: %+v", res.Blocked)
	}
	if res.Lane != "lane-2" || res.Awaiting == nil || res.Awaiting.Role != RoleRuthless {
		t.Fatalf("lane-2's offline work moves in the same call: %+v", res)
	}
	if res.Outage == nil || !slices.Equal(res.Outage.Down, []string{implement.ServiceNetwork}) {
		t.Fatalf("the result names the outage: %+v", res.Outage)
	}
	if l := f.lane(t, "lane-1"); l.Landing.Pushed != "" || l.Stage != StageLand {
		t.Fatalf("lane-1 stays at its push: %+v", l.Landing)
	}
	run := f.shared(t)
	cur := outageNow(t, run)
	if cur == nil || !slices.Equal(cur.Down, []string{implement.ServiceNetwork}) || len(cur.Reports) != 1 ||
		cur.Reports[0].Lane != "lane-1" || cur.Reports[0].Session != "host-a" || !strings.Contains(cur.Reports[0].What, "git push") {
		t.Fatalf("the outage is recorded for the claim's session: %+v", cur)
	}

	// Before the probe is due, lane-1 waits without trying the push again.
	res = f.step(t)
	if len(res.Blocked) == 0 || res.Blocked[0].Lane != "lane-1" || !strings.Contains(res.Blocked[0].Reason, "waits on the shared network probe") {
		t.Fatalf("lane-1 still waits on the probe: %+v", res)
	}
	if cur := outageNow(t, run); len(cur.Reports) != 1 {
		t.Fatalf("the push was not retried before the probe: %d reports", len(cur.Reports))
	}
}

// TestStepProbesOnceWhenDue: a step before the probe is due runs none and
// waits; the step that finds it due runs it once; a step right after runs no
// second; and the probe that proves the network back ends the outage, and the
// held push goes out in that same call.
func TestStepProbesOnceWhenDue(t *testing.T) {
	f := newParFixture(t, "", Options{Session: "host-a"})
	f.stepUntil(t, "the implementer is out", func(st State) bool { return len(st.Lanes[0].Awaits) == 1 })
	f.implement(t, "lane-1", "one.txt")
	f.toPush(t, "lane-1")
	l := f.lane(t, "lane-1")
	preflighted(t, l, l.HeadSHA)
	run := f.shared(t)
	if _, err := run.Outage().Record("host-a", implement.ServiceNetwork, implement.OutageTool, "lane-1", "git push"); err != nil {
		t.Fatal(err)
	}

	calls, up := 0, false
	o := f.opts()
	o.NetworkProbe = func() (bool, string) {
		calls++
		if up {
			return true, "origin answered"
		}
		return false, "Could not resolve host: github.com"
	}
	step := func() (StepResult, error) { return advance(f.repo.Root(), f.runID, f.stages, o) }

	_, err := step()
	r := mustRefusal(t, err)
	if !r.Contention || r.Lane != "lane-1" || !strings.Contains(r.Reason, "next 09:01") || r.Outage == nil || calls != 0 {
		t.Fatalf("before the probe is due the step waits and probes nothing (calls %d): %+v", calls, r)
	}

	f.now = f.now.Add(time.Minute)
	_, err = step()
	r = mustRefusal(t, err)
	if calls != 1 || !r.Contention || !strings.Contains(r.Reason, "next 09:06") {
		t.Fatalf("the due probe runs once and fails, and the next falls five minutes on (calls %d): %+v", calls, r)
	}
	if _, err := step(); err == nil || calls != 1 {
		t.Fatalf("a step right after runs no second probe: calls %d", calls)
	}
	if cur := outageNow(t, run); cur == nil || len(cur.Probes) != 1 {
		t.Fatalf("one probe recorded: %+v", cur)
	}

	f.now = f.now.Add(5 * time.Minute)
	up = true
	res, err := step()
	if err != nil {
		t.Fatalf("the probe that proves the network back lets the push go: %v", err)
	}
	if calls != 2 || outageNow(t, run) != nil {
		t.Fatalf("the second due probe ran and ended the outage (calls %d)", calls)
	}
	if f.lane(t, "lane-1").Landing.Pushed == "" || res.Outage != nil {
		t.Fatalf("the held push went out in the same call: %+v %+v", f.lane(t, "lane-1").Landing, res)
	}
}

// giveUp opens a network outage on the shared run and drives it to the
// give-up on a short schedule.
func giveUp(t *testing.T, run *implement.Run, session string, advance func(time.Duration)) {
	t.Helper()
	h := run.Outage()
	h.Schedule = implement.Schedule{Waits: []time.Duration{time.Minute}, Hourly: time.Minute, HourlyFor: time.Minute}
	if _, err := h.Record(session, implement.ServiceNetwork, implement.OutageTool, "lane-2", "gh pr create"); err != nil {
		t.Fatal(err)
	}
	down := implement.Prober{Network: func() (bool, string) { return false, "connection refused" }}
	for range 4 {
		advance(time.Minute)
		out, err := h.ProbeIfDue(session, down)
		if err != nil {
			t.Fatal(err)
		}
		if out.GaveUp {
			return
		}
	}
	t.Fatal("the outage never gave up")
}

// TestStepAfterGiveUpRefusesWithDoneAndLeft: once the run gave up, every step
// is refused at the stage `outage`, naming since when, the probes, what the run
// had done (the lanes landed), what is left (each lane with its stage, the
// pending steps) and the pending notification, which an ack lowers.
func TestStepAfterGiveUpRefusesWithDoneAndLeft(t *testing.T) {
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n3. Three\n   - needs: 2\n", Options{SubAgents: strp("4"), Session: "host-a"})
	f.stepUntil(t, "both implementers are out", func(st State) bool {
		return len(st.Lanes) == 2 && len(st.Lanes[0].Awaits) == 1 && len(st.Lanes[1].Awaits) == 1
	})
	f.implement(t, "lane-1", "one.txt")
	f.roundPassed(t, "lane-1")
	f.landed(t, "lane-1")
	run := f.shared(t)
	giveUp(t, run, "host-a", func(d time.Duration) { f.now = f.now.Add(d) })
	before := stateBytes(t, f.repo.Root(), f.runID)

	_, err := advance(f.repo.Root(), f.runID, f.stages, f.opts())
	r := mustRefusal(t, err)
	if r.Stage != StageOutage || r.Contention || r.Outage == nil {
		t.Fatalf("a step after the give-up is refused at the stage outage: %+v", r)
	}
	in := r.Outage
	if in.GaveUpAt == nil || in.Probes < 2 || !in.Notify || in.Since.IsZero() {
		t.Fatalf("the refusal names since when, the probes and the pending notification: %+v", in)
	}
	if in.Done == nil || !slices.Equal(in.Done.Landed, []string{"lane-1"}) {
		t.Fatalf("done names the landed lane: %+v", in.Done)
	}
	if in.Left == nil || len(in.Left.Lanes) != 1 || in.Left.Lanes[0].Lane != "lane-2" || in.Left.Lanes[0].Stage != StageImplement ||
		len(in.Left.Pending) != 1 || in.Left.Pending[0].Number != 3 {
		t.Fatalf("left names lane-2 at its stage and step 3 pending: %+v", in.Left)
	}
	for _, want := range []string{"landed lane-1", "lane-2 at implement", "step 3 pending", "product thinker", "outage ack"} {
		if !strings.Contains(r.Error(), want) {
			t.Fatalf("the refusal's text names %q: %s", want, r.Error())
		}
	}
	if !slices.Equal(before, stateBytes(t, f.repo.Root(), f.runID)) {
		t.Fatal("the refused step wrote the run's state")
	}

	if _, err := run.Outage().Ack("host-a"); err != nil {
		t.Fatal(err)
	}
	_, err = advance(f.repo.Root(), f.runID, f.stages, f.opts())
	if r := mustRefusal(t, err); r.Stage != StageOutage || r.Outage.Notify || strings.Contains(r.Remedy, "outage ack") {
		t.Fatalf("after the ack the step is still refused, with no notification pending: %+v", r)
	}
}

// joinShared joins session to the shared run state of the checkout at root.
func joinShared(t *testing.T, root string, now func() time.Time) *implement.Run {
	t.Helper()
	sha := strings.TrimSpace(gitOut(t, root, "rev-list", "--max-parents=0", "HEAD"))
	run := sharedRun(t, root, sha, "host-a")
	run.Now = now
	return run
}

// gitOut runs one git command in root and returns its output.
func gitOut(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Env = gittest.Env(t)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// withGit puts git beside the fake harnesses on the drive environment's PATH:
// the loop reads the shared run's outage through the repository's root commit.
func withGit(t *testing.T, harnesses ...string) driveEnv {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	env := newDriveEnv(t, "ok", harnesses...)
	if err := os.Symlink(git, filepath.Join(env.bin, "git")); err != nil {
		t.Fatal(err)
	}
	return env
}

// TestDriveStartsNoRunnerDuringModelOutage: while the model service is down
// the driven step hands no agent work and launches no runner: the lane waits
// on the shared model probe, which only the lead's canary proves.
func TestDriveStartsNoRunnerDuringModelOutage(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	run := joinShared(t, root, o.Now)
	if _, err := run.Outage().Record("host-a", implement.ServiceModel, implement.OutageAgent, "lane-1", "the implementer"); err != nil {
		t.Fatal(err)
	}
	o.Session = "host-a"
	env := withGit(t, "claude")
	cfg := runnerConfig(t, `{"roles":{"implementer":{"runner":"claude"}},"runner":{"claude":{}}}`, "")
	before := stateBytes(t, root, id)
	_, err := Drive(context.Background(), root, id, steps, o, Runners{Config: cfg, Transcripts: &memTranscripts{}})
	r := mustRefusal(t, err)
	if !r.Contention || !strings.Contains(r.Reason, "model service is down") || !strings.Contains(r.Reason, "shared model probe") ||
		!strings.Contains(r.Remedy, "--model ok|fail") || r.Outage == nil {
		t.Fatalf("the implementer waits on the shared model probe: %+v", r)
	}
	if _, err := os.Stat(filepath.Join(env.log, "claude.argv.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("no runner was launched during the model outage (%v)", err)
	}
	if !slices.Equal(before, stateBytes(t, root, id)) {
		t.Fatal("the held start wrote the run's state")
	}
}

// TestNetworkOnlyOutageLetsOfflineAgentStart: a network outage does not hold
// an agent the host starts — its work is offline — and the result names the
// outage; it does hold one a runner starts, which reaches the model from this
// machine over the network that is down.
func TestNetworkOnlyOutageLetsOfflineAgentStart(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	run := joinShared(t, root, o.Now)
	if _, err := run.Outage().Record("host-a", implement.ServiceNetwork, implement.OutageTool, "lane-9", "git fetch"); err != nil {
		t.Fatal(err)
	}
	o.Session = "host-a"
	o.NetworkProbe = func() (bool, string) { t.Fatal("no probe is due"); return false, "" }

	env := withGit(t, "claude")
	cfg := runnerConfig(t, `{"roles":{"implementer":{"runner":"claude"}},"runner":{"claude":{}}}`, "")
	before := stateBytes(t, root, id)
	_, err := Drive(context.Background(), root, id, steps, o, Runners{Config: cfg, Transcripts: &memTranscripts{}})
	if r := mustRefusal(t, err); !r.Contention || !strings.Contains(r.Reason, "runner") || !strings.Contains(r.Reason, "shared network probe") {
		t.Fatalf("a runner-started implementer waits on the network probe: %+v", r)
	}
	if _, err := os.Stat(filepath.Join(env.log, "claude.argv.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("no runner was launched during the network outage (%v)", err)
	}
	if !slices.Equal(before, stateBytes(t, root, id)) {
		t.Fatal("the held start wrote the run's state")
	}

	res, err := advance(root, id, steps, o)
	if err != nil {
		t.Fatalf("a host-started agent is not held by a network outage: %v", err)
	}
	if res.Awaiting == nil || res.Awaiting.Role != RoleImplementer {
		t.Fatalf("the implementer is handed out: %+v", res)
	}
	if res.Outage == nil || !slices.Equal(res.Outage.Down, []string{implement.ServiceNetwork}) {
		t.Fatalf("the result names the open outage: %+v", res.Outage)
	}
}

// TestDrainStopsOnOutage: a drain whose run gave up on an outage stops with
// stopped "outage", naming the give-up.
func TestDrainStopsOnOutage(t *testing.T) {
	repo := drainRepo(t)
	c := &clock{t: time.Date(2026, 10, 9, 9, 0, 0, 0, time.UTC)}
	o := Options{Now: c.now}
	first := drainToLane(t, repo, o, DrainOptions{})
	if first.Start == nil {
		t.Fatalf("the drain opens its first lane: %+v", first)
	}
	run := joinShared(t, repo.Root(), c.now)
	giveUp(t, run, "host-a", func(d time.Duration) { c.t = c.t.Add(d) })

	res, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Stopped != DrainStoppedOutage || !res.Complete || !strings.Contains(res.Next, "gave up") {
		t.Fatalf("the drain stops on the outage: %+v", res)
	}
	st, live, err := readDrain(repo.Root())
	if err != nil || live || st.Stopped != DrainStoppedOutage || st.EndedAt == nil {
		t.Fatalf("the drain's state records the stop: %+v live=%v %v", st, live, err)
	}
}

// TestRecordCarriesOutages: the run record lists every outage the shared log
// records over the run's lifetime, with its minutes, outcome and what was
// retried, and none from before the run began.
func TestRecordCarriesOutages(t *testing.T) {
	root, id, _, _ := startToImplement(t)
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	now := st.CreatedAt.Add(-2 * time.Hour)
	run := joinShared(t, root, func() time.Time { return now })
	if _, err := run.Outage().Record("host-a", implement.ServiceNetwork, implement.OutageTool, "lane-0", "an earlier push"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Hour)
	if _, err := run.Outage().Clear("host-a", "back before the run"); err != nil {
		t.Fatal(err)
	}

	now = st.CreatedAt.Add(time.Minute)
	if _, err := run.Outage().Record("host-a", implement.ServiceNetwork, implement.OutageTool, "lane-1", "git push lane-1"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(3 * time.Minute)
	out, err := run.Outage().ProbeIfDue("host-a", implement.Prober{Network: func() (bool, string) { return true, "origin answered" }})
	if err != nil || !out.Ended {
		t.Fatalf("the probe ends the outage: %+v %v", out, err)
	}

	rec, err := ReadRecord(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.Outages) != 1 {
		t.Fatalf("the record carries the run's one outage, not the earlier one: %+v", rec.Outages)
	}
	s := rec.Outages[0]
	if s.Outcome != "ended" || s.Minutes != 3 || !slices.Equal(s.Retried, []string{"git push lane-1"}) || s.Probes != 1 {
		t.Fatalf("the outage carries its minutes, outcome, probes and what was retried: %+v", s)
	}
}
