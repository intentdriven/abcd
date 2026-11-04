package loop

// receipt_standing_stall_test.go holds what a step says and does around a
// receipt it read: a lane stopped on one that does not verify names only the
// real ways out and comes back from a drain by name; a retry's agent answers
// for its own attempt; and what stood before an agent was started is never its
// answer, whichever way that agent is run.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/gittest"
)

// duringRun is a transcript store that runs fn when a runner's run has ended
// and before its answer is handed in, the moment a peer's step can act on it.
type duringRun struct {
	memTranscripts
	fn func()
}

func (d *duringRun) Store(name string, req runner.Request, a runner.Answer, raw []byte) error {
	if d.fn != nil {
		d.fn()
	}
	return d.memTranscripts.Store(name, req, a, raw)
}

// asideHolds reports whether a file under the lane's aside directory has body.
func asideHolds(t *testing.T, root, id, lane, body string) bool {
	t.Helper()
	dir := filepath.Join(root, filepath.FromSlash(RunRelDir), id, lane, AsideDirName)
	found := false
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if b, rerr := os.ReadFile(p); rerr == nil && string(b) == body {
				found = true
			}
		}
		return nil
	})
	return found
}

// stalledOnModel takes a routed implementer whose hand-in was refused and whose
// receipt names no model to the state it stops in: the receipt stands, finished,
// and the runner's report that would have named its model is gone.
func stalledOnModel(t *testing.T) (*gittest.Repo, string, Lane, string, Stages, Runners, driveEnv, LaneReceipt) {
	t.Helper()
	repo, id, l, dir := awaitingLane(t)
	c := laneCommit(t, repo, l, "one.txt")
	rc := goodReceipt(t, id, l, dir, c)
	body := map[string]any{"schema_version": rc.SchemaVersion, "run_id": rc.RunID, "lane": rc.Lane, "branch": rc.Branch,
		"commits": rc.Commits, "definition_of_done": rc.DefinitionOfDone, "report": rc.Report}
	t.Setenv(loopFakeReceiptEnv, writeReceipt(t, t.TempDir(), body))
	env := newDriveEnv(t, "receipt", "claude")
	stages := refuseVerifyOnce(DefaultStages(), nil)
	rs := Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}}
	res, err := Drive(context.Background(), repo.Root(), id, stages, Options{}, rs)
	if err != nil {
		t.Fatal(err)
	}
	if res.Paused == nil {
		t.Fatalf("the refused hand-in pauses the role: %+v", res)
	}
	return repo, id, l, dir, stages, rs, env, rc
}

// wantOnlyRealWaysOut fails on a sentence that is false of a stopped lane, and
// unless next names what is: correcting the receipt, and starting over.
func wantOnlyRealWaysOut(t *testing.T, next string, lane string) {
	t.Helper()
	for _, false1 := range []string{"retries it on the runner", "--route", "start a fresh", "did not start it"} {
		if strings.Contains(next, false1) {
			t.Fatalf("next says %q, which is false of this lane: %s", false1, next)
		}
	}
	for _, want := range []string{"does not verify", "the model your harness runs", "abcd implement receipt", "abcd implement step --restart " + lane} {
		if !strings.Contains(next, want) {
			t.Fatalf("next names %q: %s", want, next)
		}
	}
}

// TestAStoppedLaneNamesOnlyItsRealWaysOutAndEachOneWorks: every answer about the
// lane says only what is true, and each way it names moves the lane.
func TestAStoppedLaneNamesOnlyItsRealWaysOutAndEachOneWorks(t *testing.T) {
	correct := func(t *testing.T, dir string, rc LaneReceipt) {
		t.Helper()
		rc.Model = "a-model"
		writeReceipt(t, dir, rc)
	}
	t.Run("what it says", func(t *testing.T) {
		repo, id, l, _, stages, rs, env, _ := stalledOnModel(t)
		for range 2 {
			res, err := Drive(context.Background(), repo.Root(), id, stages, Options{}, rs)
			if err != nil {
				t.Fatal(err)
			}
			wantOnlyRealWaysOut(t, res.Next, l.ID)
		}
		if n := env.launches(t, "claude"); n != 1 {
			t.Fatalf("no second runner: %d launches", n)
		}
		st, err := ReadState(repo.Root(), id)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range st.Record {
			if e.Stage == StagePaused && strings.Contains(e.Note, "did not start it") {
				t.Fatalf("the record says the runner did not start it: %s", e.Note)
			}
		}
	})
	t.Run("the refused hand-in's own answer", func(t *testing.T) {
		repo, id, l, dir := awaitingLane(t)
		c := laneCommit(t, repo, l, "one.txt")
		_ = goodReceipt(t, id, l, dir, c)
		t.Setenv(loopFakeReceiptEnv, writeReceipt(t, t.TempDir(), "half a receipt"))
		newDriveEnv(t, "receipt", "claude")
		res, err := Drive(context.Background(), repo.Root(), id, DefaultStages(), Options{},
			Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}})
		if err != nil {
			t.Fatal(err)
		}
		if res.Paused == nil {
			t.Fatalf("the role pauses: %+v", res)
		}
		if strings.Contains(res.Next, "retries it on the runner") || strings.Contains(res.Next, "did not start it") {
			t.Fatalf("the step says what is false of this lane: %s", res.Next)
		}
		if !strings.Contains(res.Next, "abcd implement receipt") || !strings.Contains(res.Next, "--restart "+l.ID) {
			t.Fatalf("the step names the ways out: %s", res.Next)
		}
	})
	t.Run("a corrected receipt handed back", func(t *testing.T) {
		repo, id, _, dir, stages, _, _, rc := stalledOnModel(t)
		correct(t, dir, rc)
		res, err := Receipt(repo.Root(), id, filepath.Join(dir, ReceiptFileName), stages, Options{})
		if err != nil || res.Stage != StageValidate {
			t.Fatalf("the hand-in moves the lane on: %+v %v", res, err)
		}
	})
	t.Run("a corrected receipt a plain step reads", func(t *testing.T) {
		repo, id, _, dir, stages, rs, env, rc := stalledOnModel(t)
		correct(t, dir, rc)
		res, err := Drive(context.Background(), repo.Root(), id, stages, Options{}, rs)
		if err != nil || res.Stage != StageValidate {
			t.Fatalf("the step collects it: %+v %v", res, err)
		}
		if n := env.launches(t, "claude"); n != 1 {
			t.Fatalf("no second runner: %d launches", n)
		}
	})
	t.Run("starting over", func(t *testing.T) {
		repo, id, l, _, stages, rs, env, _ := stalledOnModel(t)
		res, err := RestartDrive(context.Background(), repo.Root(), id, l.ID, "", stages, Options{}, rs)
		if err != nil {
			t.Fatalf("the restart the step names is taken: %v", err)
		}
		if n := env.launches(t, "claude"); n != 2 {
			t.Fatalf("a restart starts the fresh agent: %d launches, %+v", n, res)
		}
		if res.Aside.Receipt != ReceiptFileName {
			t.Fatalf("the old receipt is set aside: %+v", res.Aside)
		}
	})
	t.Run("a route says what a plain step does", func(t *testing.T) {
		repo, id, _, dir, stages, _, _, _ := stalledOnModel(t)
		cfg := runnerConfig(t, claudeImplementer, "")
		overrideRoute(t, cfg, "implementer=runner:claude")
		_, err := Drive(context.Background(), repo.Root(), id, stages, Options{}, Runners{Config: cfg, Transcripts: &memTranscripts{}})
		r := mustRefusal(t, err)
		if !strings.Contains(r.Remedy, filepath.ToSlash(relTo(t, repo.Root(), filepath.Join(dir, ReceiptFileName)))) {
			t.Fatalf("the refusal names the receipt that stands: %+v", r)
		}
	})
}

// TestARunningClaimIsWaitedOutBeforeARestartIsOffered: while a runner's claim
// stands its agent may still be working, so the step says to wait, offers no
// restart, and counts nothing against the lane; once it lapses the ways out are
// named.
func TestARunningClaimIsWaitedOutBeforeARestartIsOffered(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	res, err := advance(root, id, steps, o)
	if err != nil || res.Awaiting == nil {
		t.Fatalf("the implementer is handed out: %+v %v", res, err)
	}
	path := res.Awaiting.Receipt
	until := o.now().Add(time.Hour)
	claimAwait(t, root, id, path, until)
	if err := os.WriteFile(path, []byte("half a receipt"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err = advance(root, id, steps, o)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(res.Next, "--restart") || !strings.Contains(res.Next, "may still be working") || !strings.Contains(res.Next, stamp(until)) {
		t.Fatalf("the step says to wait for the claim: %s", res.Next)
	}
	later := Options{Now: func() time.Time { return until.Add(time.Minute) }}
	res, err = advance(root, id, steps, later)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Next, "--restart lane-1") || strings.Contains(res.Next, "may still be working") {
		t.Fatalf("a lapsed claim leaves the real ways out: %s", res.Next)
	}
	if strings.Contains(res.Next, "start a fresh") {
		t.Fatalf("the step does not ask for a fresh agent: %s", res.Next)
	}
}

// TestADrainHandsAStoppedLaneBackByName: a stopped lane is no wait. A drain
// counts it as it counts a refused step, hands the lane back as stuck at the
// bound, and takes up the next issue. A claim that may still be working counts
// for nothing.
func TestADrainHandsAStoppedLaneBackByName(t *testing.T) {
	c := &clock{t: time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)}
	o := Options{Now: c.now}
	repo, bare, runID := stuckDrain(t, c)
	stepTo(t, repo, runID, DefaultStages(), StageImplement)
	res, err := advance(repo.Root(), runID, DefaultStages(), o)
	if err != nil || res.Awaiting == nil {
		t.Fatalf("the implementer is handed out: %+v %v", res, err)
	}
	path := res.Awaiting.Receipt
	claimAwait(t, repo.Root(), runID, path, c.now().Add(time.Hour))
	if err := os.WriteFile(absIn(repo.Root(), path), []byte("half a receipt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := advance(repo.Root(), runID, DefaultStages(), o); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Lanes[0].Refused != nil {
		t.Fatalf("a claim that may still be working counts for nothing: %+v", st.Lanes[0].Refused)
	}
	if err := mutate(repo.Root(), runID, func(_ *os.Root, st *State) (bool, error) {
		st.Lanes[0].Awaits[0].Running = nil
		st.Lanes[0].Awaits[0].Paused = &RunnerPause{Runner: runner.Claude, Reason: runner.ReasonInvalid, Detail: "refused", Since: c.now(), At: c.now(), Tries: 1}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := advance(repo.Root(), runID, DefaultStages(), o); err != nil {
			t.Fatal(err)
		}
	}
	dr, err := Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if dr.Lane == nil || dr.Lane.RunID != runID || len(dr.Routed) != 0 {
		t.Fatalf("two steps leave the lane in progress: %+v", dr)
	}
	if _, err := advance(repo.Root(), runID, DefaultStages(), o); err != nil {
		t.Fatal(err)
	}
	dr, err = Drain(repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	route := wantStuck(t, dr, bare, runID, ReceiptFileName)
	if route.Kind != claim.HandBackStuck || !strings.Contains(route.Reason, "refused") {
		t.Fatalf("the hand-back names the lane's receipt and why: %+v", route)
	}
}

// TestARunnersRewrittenReceiptIsNamedAndNoPauseIsRecorded is the path a routed
// implementer's rewrite takes: a peer's step collects its receipt, the agent
// changes the file, and its runner's hand-in is refused. Nothing is paused or
// retried, so the record says neither: it says what happened, and so does the
// step's answer.
func TestARunnersRewrittenReceiptIsNamedAndNoPauseIsRecorded(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	env := newDriveEnv(t, "ok", "claude")
	var path string
	store := &duringRun{}
	store.fn = func() {
		res, err := advance(root, id, steps, o)
		if err != nil || res.Stage != StageValidate {
			t.Errorf("the peer's step collects the receipt: %+v %v", res, err)
		}
		st, err := ReadState(root, id)
		if err != nil {
			t.Fatal(err)
		}
		path = st.Lanes[0].Receipt
		if err := os.WriteFile(path, []byte(`{"rewritten": true}`+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	res, err := Drive(context.Background(), root, id, steps, o, Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: store})
	if err != nil {
		t.Fatalf("nothing about the step itself is refused: %v", err)
	}
	if res.Paused != nil || res.Stage != StageValidate {
		t.Fatalf("nothing is paused, and the lane stays where the collection took it: %+v", res)
	}
	if !strings.Contains(res.Next, "changed since") {
		t.Fatalf("the step's answer carries the refusal: %s", res.Next)
	}
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Fallbacks) != 0 {
		t.Fatalf("no failed start is counted: %+v", st.Fallbacks)
	}
	for _, e := range st.Record {
		if e.Stage == StagePaused {
			t.Fatalf("no pause is recorded: %s", e.Note)
		}
	}
	if !recordSays(t, root, id, StageRunner, "changed since") {
		t.Fatalf("the record says what happened: %+v", st.Record)
	}
	if n := env.launches(t, "claude"); n != 1 {
		t.Fatalf("nothing is started again: %d launches", n)
	}
}

// TestARetrysClaimDoesNotAdmitThePreviousAttemptsReceipt: a claim speaks for its
// own attempt. What the last attempt left at the path, when no step took it, is
// set aside as the claim is taken, so a peer's step during the retry finds nothing
// of it to collect.
func TestARetrysClaimDoesNotAdmitThePreviousAttemptsReceipt(t *testing.T) {
	for name, c := range map[string]struct {
		reason runner.Reason
		body   string
	}{
		"a runner that never ran, and a file that would verify": {runner.ReasonAbsent, "{}\n"},
		"a runner that exited badly, and a file that does not":  {runner.ReasonFailed, "half a receipt"},
	} {
		t.Run(name, func(t *testing.T) {
			root, id, steps, o := startToImplement(t)
			res, err := advance(root, id, steps, o)
			if err != nil || res.Awaiting == nil {
				t.Fatalf("the implementer is handed out: %+v %v", res, err)
			}
			path := res.Awaiting.Receipt
			if err := mutate(root, id, func(_ *os.Root, st *State) (bool, error) {
				st.Lanes[0].Awaits[0].Paused = &RunnerPause{Runner: runner.Claude, Reason: c.reason, Detail: "the last attempt", Since: o.now(), At: o.now(), Tries: 1}
				return true, nil
			}); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(c.body), 0o600); err != nil {
				t.Fatal(err)
			}
			env := newDriveEnv(t, "noreceipt", "claude")
			store := &duringRun{}
			store.fn = func() {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Errorf("the last attempt's file is off the path during the retry: %v", err)
				}
				if res, err := advance(root, id, steps, o); err != nil || res.Stage != StageImplement {
					t.Errorf("a peer's step during the retry collects nothing: %+v %v", res, err)
				}
			}
			if _, err := Drive(context.Background(), root, id, steps, o, Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: store}); err != nil {
				t.Fatal(err)
			}
			if n := env.launches(t, "claude"); n != 1 {
				t.Fatalf("the retry starts its runner: %d launches", n)
			}
			if !asideHolds(t, root, id, "lane-1", c.body) {
				t.Fatal("the last attempt's file is kept aside")
			}
		})
	}
}

// TestAFinishedAgentWhoseRunnerExitedBadlyIsCollected: an agent can commit and
// write its receipt, and its harness then exit non-zero or print what cannot be
// parsed, so no hand-in ever ran. The attempt's claim cleared the path, so the
// file is that attempt's: the next step reads it, and a receipt that verifies
// ends the pause with no second agent, naming the runner. A --route for the role
// reads nothing and starts nothing, and the plain step after it collects.
func TestAFinishedAgentWhoseRunnerExitedBadlyIsCollected(t *testing.T) {
	for _, reason := range []runner.Reason{runner.ReasonFailed, runner.ReasonUnparsable, runner.ReasonRefused} {
		t.Run(string(reason), func(t *testing.T) {
			setUp := func(t *testing.T) (string, string, Stages, Options, string) {
				t.Helper()
				root, id, steps, o := startToImplement(t)
				res, err := advance(root, id, steps, o)
				if err != nil || res.Awaiting == nil {
					t.Fatalf("the implementer is handed out: %+v %v", res, err)
				}
				path := res.Awaiting.Receipt
				if err := mutate(root, id, func(_ *os.Root, st *State) (bool, error) {
					st.Lanes[0].Awaits[0].Paused = &RunnerPause{Runner: runner.Claude, Reason: reason, Detail: "exited 1", Since: o.now(), At: o.now(), Tries: 1}
					return true, nil
				}); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				return root, id, steps, o, path
			}
			t.Run("a plain step", func(t *testing.T) {
				root, id, steps, o, path := setUp(t)
				env := newDriveEnv(t, "ok", "claude")
				res, err := Drive(context.Background(), root, id, steps, o, Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}})
				if err != nil {
					t.Fatal(err)
				}
				if res.Stage != StageValidate || res.PerformedStage != StageImplement || res.Retried != nil {
					t.Fatalf("the step collects the receipt: %+v", res)
				}
				if n := env.launches(t, "claude"); n != 0 {
					t.Fatalf("no second agent is paid for: %d launches", n)
				}
				if body, err := os.ReadFile(path); err != nil || string(body) != "{}\n" {
					t.Fatalf("the receipt stays where it stood: %q %v", body, err)
				}
				st, err := ReadState(root, id)
				if err != nil {
					t.Fatal(err)
				}
				if r := st.Lanes[0].Receipts[0].Route; r == nil || r.Asked != runner.Claude || r.Ran != "" {
					t.Fatalf("the receipt names the runner its pause recorded: %+v", r)
				}
			})
			t.Run("a route first", func(t *testing.T) {
				root, id, steps, o, _ := setUp(t)
				env := newDriveEnv(t, "ok", "claude")
				cfg := runnerConfig(t, claudeImplementer, "")
				overrideRoute(t, cfg, "implementer=runner:claude")
				if _, err := Drive(context.Background(), root, id, steps, o, Runners{Config: cfg, Transcripts: &memTranscripts{}}); err == nil {
					t.Fatal("a --route the step will not use is refused")
				}
				res, err := Drive(context.Background(), root, id, steps, o, Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}})
				if err != nil || res.Stage != StageValidate {
					t.Fatalf("the plain step collects: %+v %v", res, err)
				}
				if n := env.launches(t, "claude"); n != 0 {
					t.Fatalf("no second agent is paid for: %d launches", n)
				}
			})
		})
	}
}

// TestAReceiptAnExitedRunnerLeftThatDoesNotVerifyIsRetried: a run that ended badly
// may have stopped anywhere, so its unfinished receipt is no finished work to stop
// for. The role is retried as it always was, the step's answer says what became of
// the file, and nothing counts against the lane.
func TestAReceiptAnExitedRunnerLeftThatDoesNotVerifyIsRetried(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	res, err := advance(root, id, steps, o)
	if err != nil || res.Awaiting == nil {
		t.Fatalf("the implementer is handed out: %+v %v", res, err)
	}
	path := res.Awaiting.Receipt
	if err := mutate(root, id, func(_ *os.Root, st *State) (bool, error) {
		st.Lanes[0].Awaits[0].Paused = &RunnerPause{Runner: runner.Claude, Reason: runner.ReasonFailed, Detail: "exited 1", Since: o.now(), At: o.now(), Tries: 1}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("half a receipt"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := newDriveEnv(t, "ok", "claude")
	res, err = Drive(context.Background(), root, id, steps, o, Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatal(err)
	}
	if n := env.launches(t, "claude"); n != 1 {
		t.Fatalf("the role is retried: %d launches", n)
	}
	if res.Stage != StageValidate {
		t.Fatalf("the fresh agent's receipt is taken: %+v", res)
	}
	if !strings.Contains(res.Next, "set aside if") {
		t.Fatalf("the answer says what becomes of the file: %s", res.Next)
	}
	if !asideHolds(t, root, id, "lane-1", "half a receipt") {
		t.Fatal("the file is kept aside")
	}
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if st.Lanes[0].Refused != nil {
		t.Fatalf("nothing counts against the lane: %+v", st.Lanes[0].Refused)
	}
}

// TestAnImplementerKilledWithItsReceiptWrittenIsCollected: an agent killed at its
// bound after it wrote everything, its receipt included, is not one whose receipt
// is missing. The kill pauses the role, and the next step reads the receipt.
func TestAnImplementerKilledWithItsReceiptWrittenIsCollected(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	id := start.RunID
	if err := os.WriteFile(filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), "brief.md"), []byte("# brief\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	wt := gittest.NewRepo(t)
	wt.Write("base.txt", "base\n")
	wt.Commit("base")
	base := strings.TrimSpace(wt.Git("rev-parse", "HEAD"))
	wt.Write("work.txt", "work\n")
	wt.Commit("work")
	steps := driveSteps(wt.Root())
	cut := steps[0].Run
	steps[0].Run = func(c Context, l *Lane) (Outcome, error) {
		out, err := cut(c, l)
		l.BaseSHA = base
		return out, err
	}
	// driveSteps' receipt is at the checkout's root, the report beside it.
	if err := os.WriteFile(filepath.Join(repo.Root(), ReportFileName), []byte("# report\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	clock := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	o := Options{Now: func() time.Time { return clock }}
	for range 2 {
		if _, err := advance(repo.Root(), id, steps, o); err != nil {
			t.Fatal(err)
		}
	}
	env := newDriveEnv(t, "hang", "claude")
	// The agent writes its receipt while its harness is still up, as the claim
	// already stands.
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-time.After(20 * time.Millisecond):
			}
			if _, err := os.Stat(filepath.Join(env.log, "claude.pid")); err == nil {
				_ = os.WriteFile(filepath.Join(repo.Root(), "receipt-lane-1.json"), []byte("{}\n"), 0o600)
				return
			}
		}
	}()
	rs := Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}, Timeout: 2 * time.Second}
	res, err := Drive(context.Background(), repo.Root(), id, steps, o, rs)
	if err != nil {
		t.Fatal(err)
	}
	if res.Paused == nil {
		t.Fatalf("the kill pauses the role for the next step to read: %+v", res)
	}
	if strings.Contains(res.Next, "only its receipt is missing") {
		t.Fatalf("the step says what is false: %s", res.Next)
	}
	res, err = Drive(context.Background(), repo.Root(), id, steps, o, rs)
	if err != nil {
		t.Fatal(err)
	}
	if res.Stage != StageValidate || res.PerformedStage != StageImplement {
		t.Fatalf("the next step collects the receipt: %+v", res)
	}
	if n := env.launches(t, "claude"); n != 1 {
		t.Fatalf("no second agent is paid for: %d launches", n)
	}
}

// handBackReceipt is a receipt for lane l carrying a hand-back, with its report
// in dir.
func handBackReceipt(t *testing.T, id string, l Lane, dir string) LaneReceipt {
	t.Helper()
	rc := goodReceipt(t, id, l, dir)
	rc.Commits, rc.DefinitionOfDone = nil, nil
	rc.HandBack = &LaneHandBack{Kind: HandBackTrustRule, Reason: "the fix turns on a rule"}
	return rc
}

// wantLaneKept fails unless lane l of the run keeps its worktree and branch and
// is not handed back.
func wantLaneKept(t *testing.T, repo *gittest.Repo, id string, l Lane) {
	t.Helper()
	if _, err := os.Stat(l.Worktree); err != nil {
		t.Fatalf("the lane's worktree stands: %v", err)
	}
	if out := repo.Git("branch", "--list", l.Branch); !strings.Contains(out, l.Branch) {
		t.Fatalf("the lane's branch stands: %q", out)
	}
	if got := currentLane(t, repo, id); got.HandBack != nil || got.Stage == StageHandedBack {
		t.Fatalf("the lane is not handed back: %+v", got)
	}
}

// TestAStepTakesNoHandBackFromAFileAtAPath: a hand-back discards the lane's
// worktree and branch, so it is an act, made through `implement receipt` or by the
// runner's own hand-in as its agent ends. A step leaves such a receipt to the path
// any receipt it cannot take has, and the lane stands.
func TestAStepTakesNoHandBackFromAFileAtAPath(t *testing.T) {
	pauseAs := func(t *testing.T, repo *gittest.Repo, id string, reason runner.Reason) {
		t.Helper()
		if err := mutate(repo.Root(), id, func(_ *os.Root, st *State) (bool, error) {
			now := time.Now()
			st.Lanes[0].Awaits[0].Paused = &RunnerPause{Runner: runner.Claude, Reason: reason, Detail: "the last attempt", Since: now, At: now, Tries: 1}
			return true, nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("after a bad exit it is retried over", func(t *testing.T) {
		repo, id, l, dir := awaitingLane(t)
		pauseAs(t, repo, id, runner.ReasonFailed)
		rc := handBackReceipt(t, id, l, dir)
		writeReceipt(t, dir, rc)
		env := newDriveEnv(t, "noreceipt", "claude")
		if _, err := Drive(context.Background(), repo.Root(), id, DefaultStages(), Options{},
			Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}}); err != nil {
			t.Fatal(err)
		}
		wantLaneKept(t, repo, id, l)
		if n := env.launches(t, "claude"); n != 1 {
			t.Fatalf("the role is retried: %d launches", n)
		}
		if _, err := os.Stat(filepath.Join(dir, ReceiptFileName)); !os.IsNotExist(err) {
			t.Fatalf("the file is set aside as the retry's claim is taken: %v", err)
		}
	})
	t.Run("under a claim it waits", func(t *testing.T) {
		repo, id, l, dir := awaitingLane(t)
		claimAwait(t, repo.Root(), id, l.awaiting().Receipt, time.Now().Add(time.Hour))
		writeReceipt(t, dir, handBackReceipt(t, id, l, dir))
		res, err := advance(repo.Root(), id, DefaultStages(), Options{})
		if err != nil {
			t.Fatal(err)
		}
		wantLaneKept(t, repo, id, l)
		if !strings.Contains(res.Next, "may still be working") {
			t.Fatalf("the step says to wait: %s", res.Next)
		}
	})
	t.Run("after a refused answer implement receipt takes it", func(t *testing.T) {
		repo, id, l, dir := awaitingLane(t)
		pauseAs(t, repo, id, runner.ReasonInvalid)
		path := writeReceipt(t, dir, handBackReceipt(t, id, l, dir))
		res, err := advance(repo.Root(), id, DefaultStages(), Options{})
		if err != nil {
			t.Fatal(err)
		}
		wantLaneKept(t, repo, id, l)
		if !strings.Contains(res.Next, "abcd implement receipt") {
			t.Fatalf("the step names the hand-in: %s", res.Next)
		}
		res, err = Receipt(repo.Root(), id, path, DefaultStages(), Options{})
		if err != nil || res.HandBack == nil {
			t.Fatalf("the hand-in hands the lane back: %+v %v", res, err)
		}
	})
	t.Run("at a sync's receipt", func(t *testing.T) {
		repo, id, stages := reviewedRun(t)
		l := currentLane(t, repo, id)
		rel := RunRelDir + "/" + id + "/" + l.ID + "/sync-1/" + ReceiptFileName
		if err := mutate(repo.Root(), id, func(_ *os.Root, st *State) (bool, error) {
			st.Lanes[0].Syncs = []Sync{{Merged: l.BaseSHA, Conflicted: true, Receipt: rel, Brief: l.Brief}}
			st.Lanes[0].Awaits = []Await{{Role: RoleImplementer, Brief: l.Brief, Receipt: rel, Since: time.Now(),
				Running: &RunnerClaim{Runner: runner.Claude, Session: "the-claim", Until: time.Now().Add(-time.Hour)}}}
			return true, nil
		}); err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), id, l.ID)
		rc := handBackReceipt(t, id, l, dir)
		body, err := json.Marshal(rc)
		if err != nil {
			t.Fatal(err)
		}
		standAt(t, repo.Root(), rel, string(body))
		if _, err := advance(repo.Root(), id, stages, Options{}); err != nil {
			t.Fatal(err)
		}
		wantLaneKept(t, repo, id, l)
	})
}

// reviewedRun is a run whose lane has its implementer's receipt, at validate,
// with one reviewer per role asking for no model.
func reviewedRun(t *testing.T) (*gittest.Repo, string, Stages) {
	t.Helper()
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := mutate(repo.Root(), start.RunID, func(_ *os.Root, st *State) (bool, error) {
		st.Review = nil
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	implemented(t, repo, start.RunID, DefaultStages(), "one.txt")
	return repo, start.RunID, DefaultStages()
}

// standAt writes body at a path of the run, as whatever stood there before.
func standAt(t *testing.T, root, rel, body string) {
	t.Helper()
	abs := absIn(root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// firstReturn is round one's first reviewer's return path.
func firstReturn(t *testing.T, id, lane string) string {
	t.Helper()
	dir, err := reviewerDir(id, lane, 1, RoleRuthless, 0)
	if err != nil {
		t.Fatal(err)
	}
	return dir + "/" + ReturnFileName
}

// TestWhatStoodBeforeAnAgentWasStartedIsNotItsAnswer: at each way an agent is
// run, a file at its path from before it was handed out is set aside unread as
// it is handed out, and an agent that then writes nothing has answered nothing.
func TestWhatStoodBeforeAnAgentWasStartedIsNotItsAnswer(t *testing.T) {
	writesNothing := func(t *testing.T) driveEnv {
		t.Helper()
		t.Setenv(loopFakeReceiptEnv, filepath.Join(t.TempDir(), "no-such-file"))
		return newDriveEnv(t, "receipt", "claude")
	}
	t.Run("a runner's reviewer", func(t *testing.T) {
		repo, id, stages := reviewedRun(t)
		l := currentLane(t, repo, id)
		standAt(t, repo.Root(), firstReturn(t, id, l.ID), reviewerReturn("SHIP"))
		env := writesNothing(t)
		res, err := Drive(context.Background(), repo.Root(), id, stages, Options{},
			Runners{Config: runnerConfig(t, routeOnClaude(RoleRuthless), ""), Transcripts: &memTranscripts{}})
		if err != nil {
			t.Fatal(err)
		}
		if n := env.launches(t, "claude"); n != 1 {
			t.Fatalf("the reviewer is started: %d launches", n)
		}
		st, err := ReadState(repo.Root(), id)
		if err != nil {
			t.Fatal(err)
		}
		if v := st.Lanes[0].Validation[0].Validators[0]; v.Verdict != "" || v.Route != nil {
			t.Fatalf("no verdict is recorded for a reviewer that wrote nothing: %+v", v)
		}
		if res.Paused == nil {
			t.Fatalf("the role pauses for a reviewer that wrote nothing: %+v", res)
		}
		if !asideHolds(t, repo.Root(), id, l.ID, reviewerReturn("SHIP")) {
			t.Fatal("what stood is kept aside")
		}
		if !recordSays(t, repo.Root(), id, "", "set aside", firstReturn(t, id, l.ID)) {
			t.Fatalf("the record says so: %+v", st.Record)
		}
	})
	t.Run("a runner's fix round", func(t *testing.T) {
		repo, id, stages := reviewedRun(t)
		l := currentLane(t, repo, id)
		handBack(t, repo, id, stages, RoleRuthless, reviewerReturn("FIX FIRST"))
		passRound(t, repo, id, stages, RoleSecurity, RoleAuditor)
		fixDir, err := roundDir(id, l.ID, 1, FixDirName)
		if err != nil {
			t.Fatal(err)
		}
		first, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), id, l.ID, ReceiptFileName))
		if err != nil {
			t.Fatal(err)
		}
		standAt(t, repo.Root(), fixDir+"/"+ReceiptFileName, string(first))
		env := writesNothing(t)
		if _, err := Drive(context.Background(), repo.Root(), id, stages, Options{},
			Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}}); err != nil {
			t.Fatal(err)
		}
		if n := env.launches(t, "claude"); n != 1 {
			t.Fatalf("the fix round's implementer is started: %d launches", n)
		}
		st, err := ReadState(repo.Root(), id)
		if err != nil {
			t.Fatal(err)
		}
		if st.Lanes[0].Validation[0].Fix != "" {
			t.Fatalf("the round stays open: %+v", st.Lanes[0].Validation[0])
		}
	})
	t.Run("the review route", func(t *testing.T) {
		repo := briefRepo(t, agentsMarked)
		oneReviewerPerRole(t)
		start, err := Start(repo.Root(), "itd-10", Options{})
		if err != nil {
			t.Fatal(err)
		}
		id, stages := start.RunID, DefaultStages()
		l := implemented(t, repo, id, stages, "one.txt")
		standAt(t, repo.Root(), firstReturn(t, id, l.ID), reviewerReturn("SHIP"))
		res := reviewOut(t, repo.Root(), id, stages, RoleRuthless)
		srv := rpServer(func(string) string { return reviewerReturn("FIX FIRST") })
		got, err := DriveReview(context.Background(), repo.Root(), id, res, stages, Options{}, rpRoute(t, srv))
		if err != nil {
			t.Fatal(err)
		}
		if got.Route == nil || got.Route.Ran != "rp" {
			t.Fatalf("the route's own answer is taken: %+v", got)
		}
		st, err := ReadState(repo.Root(), id)
		if err != nil {
			t.Fatal(err)
		}
		if v := st.Lanes[0].Validation[0].Validators[0]; v.Verdict != "FIX FIRST" {
			t.Fatalf("the verdict is the route's: %+v", v)
		}
	})
	t.Run("a host's reviewer", func(t *testing.T) {
		repo, id, stages := reviewedRun(t)
		l := currentLane(t, repo, id)
		rel := firstReturn(t, id, l.ID)
		standAt(t, repo.Root(), rel, reviewerReturn("SHIP"))
		reviewOut(t, repo.Root(), id, stages, RoleRuthless)
		if _, err := Receipt(repo.Root(), id, absIn(repo.Root(), rel), stages, Options{}); err == nil {
			t.Fatal("nothing the host's agent wrote stands to hand back")
		}
		st, err := ReadState(repo.Root(), id)
		if err != nil {
			t.Fatal(err)
		}
		if v := st.Lanes[0].Validation[0].Validators[0]; v.Verdict != "" {
			t.Fatalf("no verdict is recorded: %+v", v)
		}
	})
	t.Run("a host's implementer", func(t *testing.T) {
		root, id, steps, o := startToImplement(t)
		rel := "receipt-lane-1.json"
		standAt(t, root, rel, "{}\n")
		res, err := advance(root, id, steps, o)
		if err != nil || res.Awaiting == nil {
			t.Fatalf("the implementer is handed out: %+v %v", res, err)
		}
		if _, err := os.Stat(absIn(root, rel)); !os.IsNotExist(err) {
			t.Fatalf("the file leaves the path as the await is handed out: %v", err)
		}
	})
}

// TestAFixRoundsRoutedImplementerIsCollectedUnderTheSameProof: a fix round's
// runner that finished and whose hand-in was lost is collected as the first
// implementer's is, once its branch has moved past the round's head. A receipt
// with no commit past that head is not a fix a step can take; it is said, and
// `implement receipt` still hands it back.
func TestAFixRoundsRoutedImplementerIsCollectedUnderTheSameProof(t *testing.T) {
	out := func(t *testing.T) (*gittest.Repo, string, Stages, Lane, string, string) {
		t.Helper()
		repo, id, stages := reviewedRun(t)
		l := currentLane(t, repo, id)
		handBack(t, repo, id, stages, RoleRuthless, reviewerReturn("FIX FIRST"))
		passRound(t, repo, id, stages, RoleSecurity, RoleAuditor)
		res, err := advance(repo.Root(), id, stages, Options{})
		if err != nil || res.Awaiting == nil || res.Awaiting.Role != RoleImplementer {
			t.Fatalf("the fix round's implementer is handed out: %+v %v", res, err)
		}
		claimAwait(t, repo.Root(), id, res.Awaiting.Receipt, time.Now().Add(time.Hour))
		dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), id, l.ID)
		return repo, id, stages, l, dir, res.Awaiting.Receipt
	}
	writeFix := func(t *testing.T, repo *gittest.Repo, id string, l Lane, dir, rel string, commits ...string) {
		t.Helper()
		rc := goodReceipt(t, id, l, dir, commits...)
		// The receipt's report and output are named relative to the lane's
		// directory, as a fix round's brief tells its implementer to.
		rcBody, err := json.Marshal(rc)
		if err != nil {
			t.Fatal(err)
		}
		standAt(t, repo.Root(), rel, string(rcBody))
	}
	t.Run("with a commit past the round's head", func(t *testing.T) {
		repo, id, stages, l, dir, rel := out(t)
		fix := laneCommit(t, repo, l, "fix.txt")
		writeFix(t, repo, id, l, dir, rel, fix)
		if _, err := advance(repo.Root(), id, stages, Options{}); err != nil {
			t.Fatal(err)
		}
		st, err := ReadState(repo.Root(), id)
		if err != nil {
			t.Fatal(err)
		}
		lane := st.Lanes[0]
		if lane.Validation[0].Fix != rel || lane.HeadSHA != fix || len(lane.Awaits) != 0 {
			t.Fatalf("the step closes the round on the fix: %+v", lane)
		}
		// A later hand-in of the same bytes is answered, and a changed file is
		// not, as for the first implementer.
		if _, err := Receipt(repo.Root(), id, absIn(repo.Root(), rel), stages, Options{}); err != nil {
			t.Fatalf("the same bytes are answered: %v", err)
		}
		standAt(t, repo.Root(), rel, "{}\n")
		if _, err := Receipt(repo.Root(), id, absIn(repo.Root(), rel), stages, Options{}); err == nil || !strings.Contains(err.Error(), "changed since") {
			t.Fatalf("a changed file is refused: %v", err)
		}
	})
	t.Run("with none", func(t *testing.T) {
		repo, id, stages, l, dir, rel := out(t)
		writeFix(t, repo, id, l, dir, rel, l.HeadSHA)
		res, err := advance(repo.Root(), id, stages, Options{})
		if err != nil {
			t.Fatal(err)
		}
		st, err := ReadState(repo.Root(), id)
		if err != nil {
			t.Fatal(err)
		}
		if lane := st.Lanes[0]; lane.Validation[0].Fix != "" || len(lane.Awaits) != 1 {
			t.Fatalf("the step does not take it: %+v", lane.Validation[0])
		}
		if !strings.Contains(res.Next, "no commit past") {
			t.Fatalf("the step says why: %s", res.Next)
		}
		if _, err := Receipt(repo.Root(), id, absIn(repo.Root(), rel), stages, Options{}); err != nil {
			t.Fatalf("a hand-in is still the implementer's to make: %v", err)
		}
	})
}

// TestACollectedDigestSurvivesAReservationSave: the digest a later hand-in is
// read against lives in the record, and a reservation-only save keeps it.
func TestACollectedDigestSurvivesAReservationSave(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	res, err := advance(root, id, steps, o)
	if err != nil || res.Awaiting == nil {
		t.Fatalf("the implementer is handed out: %+v %v", res, err)
	}
	path := res.Awaiting.Receipt
	claimAwait(t, root, id, path, o.now().Add(time.Hour))
	if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if res, err = advance(root, id, steps, o); err != nil || res.Stage != StageValidate {
		t.Fatalf("the step collects the receipt: %+v %v", res, err)
	}
	if err := mutateReservation(root, id, func(st *State) (bool, error) {
		st.Record = append(st.Record, Entry{At: o.now(), Stage: StageReservationRenewed, Note: "renewed"})
		st.UpdatedAt = o.now()
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := Receipt(root, id, path, steps, o); err != nil {
		t.Fatalf("the same bytes are still answered: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"changed": true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Receipt(root, id, path, steps, o); err == nil || !strings.Contains(err.Error(), "changed since") {
		t.Fatalf("a changed file is still refused: %v", err)
	}
}

// overrideRoute puts a step's --route on cfg.
func overrideRoute(t *testing.T, cfg *runner.Config, text string) {
	t.Helper()
	role, _, _ := strings.Cut(text, "=")
	routes, err := oracle.ParseRunnerRoutes([]string{text}, []string{role})
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Override(routes); err != nil {
		t.Fatal(err)
	}
}

// relTo is path relative to root.
func relTo(t *testing.T, root, path string) string {
	t.Helper()
	rel, err := filepath.Rel(root, path)
	if err != nil {
		t.Fatal(err)
	}
	return rel
}
