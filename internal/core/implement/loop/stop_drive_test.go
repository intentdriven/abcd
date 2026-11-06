package loop

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/core/runstop"
)

// TestAStopEndsAPreflightAtItsNextLook: one queued at the load gate or running does not
// outlast a stop set from outside the terminal.
func TestAStopEndsAPreflightAtItsNextLook(t *testing.T) {
	old := wallLookEvery
	wallLookEvery = 10 * time.Millisecond
	t.Cleanup(func() { wallLookEvery = old })
	w := newTerminalWorld(t, "shape")
	home, _ := stopWorld(t, false)
	o := w.options(t)
	o.Options.stopHome = home
	started := make(chan struct{})
	o.Preflight = func(ctx context.Context, _ string) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}
	tm := newTerminal(w.repo.Root(), o)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() { done <- tm.preflight(ctx, "the worktree") }()
	<-started
	if _, err := runstop.Set(home, time.Now()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("the preflight went on past the stop")
	}
}

// TestAPreflightAStopEndedIsNoFailure: it is not logged against the lane or waited on.
func TestAPreflightAStopEndedIsNoFailure(t *testing.T) {
	repo, runID, _, _, _ := deadLane(t)
	home, _ := stopWorld(t, false)
	if _, err := runstop.Set(home, time.Now()); err != nil {
		t.Fatal(err)
	}
	var said, slept []string
	tm := newTerminal(repo.Root(), TerminalOptions{
		Options:   Options{stopHome: home},
		Stages:    DefaultStages(),
		Say:       func(s string) { said = append(said, s) },
		Preflight: func(context.Context, string) error { return context.Canceled },
		Sleep:     func(_ context.Context, d time.Duration) { slept = append(slept, d.String()) },
	})
	again, ended, err := tm.stepRefused(context.Background(), runID, &Refusal{checkWait: time.Now()})
	if err != nil || again || ended != "" {
		t.Fatalf("the step: %v %q %v", again, ended, err)
	}
	if len(slept) != 0 || strings.Contains(strings.Join(said, "\n"), "its preflight failed") {
		t.Fatalf("it was read as a failure: %v\n%s", slept, strings.Join(said, "\n"))
	}
}

// TestAProbeStartsNothingUnderAStop: a probe is a launch.
func TestAProbeStartsNothingUnderAStop(t *testing.T) {
	w := newTerminalWorld(t, "shape")
	home, _ := stopWorld(t, false)
	if _, err := runstop.Set(home, time.Now()); err != nil {
		t.Fatal(err)
	}
	o := w.bounded(t, 30*24*time.Hour)
	o.Options.stopHome = home
	probes := 0
	o.Probe = func(context.Context, string, string) (bool, *runner.Failure) {
		probes++
		return true, nil
	}
	tm := newTerminal(w.repo.Root(), o)
	tm.blames = []launchBlame{{role: RoleImplementer, model: "a-model", blame: runner.BlameAsk}}
	tm.settleBlames(context.Background())
	if probes != 0 {
		t.Fatalf("a probe was launched under the stop: %d", probes)
	}
}

// markHalted marks the run as a stop leaves it: halted, its reservation released.
func markHalted(t *testing.T, repoRoot, runID string, fromFile bool) {
	t.Helper()
	now := time.Now().UTC()
	if err := mutate(repoRoot, runID, func(root *os.Root, st *State) (bool, error) {
		c, err := readControl(root, runID)
		if err != nil {
			return false, err
		}
		c.Stop = &RunStop{SetBy: "someone", SetAt: now, NotedAt: now, HaltedAt: &now, FromFile: fromFile, Released: "released"}
		if err := writeControl(root, runID, c); err != nil {
			return false, err
		}
		st.ReservedAt = nil
		st.Record = append(st.Record, Entry{At: now, Stage: StageReservationReleased, Note: "released"})
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
}

// armedConflict is a lane armed at land whose pull request the forge lists as conflicting
// only in a generated file, which a drain's move would take back through its sync.
func armedConflict(t *testing.T) (*landFixture, Lane) {
	t.Helper()
	f := syncFixture(t)
	f.passToLandOnce(t, "fix.txt", "gen.txt")
	f.step(t)
	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	preflighted(t, l, l.HeadSHA)
	f.step(t)
	f.step(t)
	f.step(t)
	l = currentLane(t, f.repo, f.runID)
	if !l.Landing.Armed || l.PR != 7 {
		t.Fatalf("armed: %+v", l.Landing)
	}
	f.moveMain(t, "generated again on main\n", "gen.txt")
	if err := os.WriteFile(filepath.Join(f.gh, "mergeable.json"), []byte(`{"mergeable":"CONFLICTING","mergeStateStatus":"DIRTY"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	f.listPRs(t, stubPR{Number: 7, HeadRefName: l.Branch, State: "OPEN", Mergeable: "CONFLICTING", MergeStateStatus: "DIRTY"})
	return f, l
}

// TestAParkedRunADriveStoppedOnIsLeftAlone: what its state says is not known to be its
// driver's, so a drain's move does nothing on its word and names it.
func TestAParkedRunADriveStoppedOnIsLeftAlone(t *testing.T) {
	f, _ := armedConflict(t)
	if err := recordDriveStop(f.runID); err != nil {
		t.Fatal(err)
	}
	p, err := driveStopPath(f.runID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(p) })
	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(f.ghLog(t), "--disable-auto") {
		t.Fatalf("the move disarmed a run a drive stopped on:\n%s", f.ghLog(t))
	}
	i := slices.IndexFunc(res.Finished, func(d DrainFinished) bool { return d.RunID == f.runID })
	if i < 0 || res.Finished[i].Done || !strings.Contains(res.Finished[i].Note, "a drive stopped on") {
		t.Fatalf("the move does not name it: %+v", res.Finished)
	}
}

// TestAHaltedParkedRunIsNotSyncedBeforeItsTakeOver: a drain's move neither disarms the pull
// request nor merges into the lane, and names what the run waits for.
func TestAHaltedParkedRunIsNotSyncedBeforeItsTakeOver(t *testing.T) {
	for _, tc := range []struct {
		name     string
		fromFile bool
	}{{"a person's stop", true}, {"a signal", false}} {
		t.Run(tc.name, func(t *testing.T) {
			f, l := armedConflict(t)
			markHalted(t, f.repo.Root(), f.runID, tc.fromFile)
			res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(f.ghLog(t), "--disable-auto") {
				t.Fatalf("the move disarmed a halted run:\n%s", f.ghLog(t))
			}
			if ld := currentLane(t, f.repo, f.runID).Landing; !ld.Armed || ld.Pushed != l.Landing.Pushed {
				t.Fatalf("the move changed the lane: %+v", ld)
			}
			if p := parkedOf(t, res, f.runID); !strings.Contains(p.Note, "implement resume --take "+f.runID) {
				t.Fatalf("the move does not name the take-over: %+v", p)
			}
		})
	}
}

// TestAHaltedParkedRunIsNotQueuedBeforeItsTakeOver: nor does it queue one the forge shows
// armed and clean.
func TestAHaltedParkedRunIsNotQueuedBeforeItsTakeOver(t *testing.T) {
	f := drainLandFixture(t)
	first := drainToLane(t, f.repo, Options{}, DrainOptions{})
	f.runID, f.issue = first.Start.RunID, secondIssue
	l := f.armIssueLane(t)
	f.writeGH(t, "pr.json", []any{map[string]any{"number": 7, "headRefName": l.Branch, "state": "OPEN", "mergeable": "MERGEABLE", "mergeStateStatus": "CLEAN",
		"autoMergeRequest": map[string]any{"mergeMethod": "MERGE"}}})
	f.queueAnswer(t, stubQueue{head: l.Landing.Pushed, enabledAt: "2026-10-15T07:00:00Z", status: "CLEAN"})
	markHalted(t, f.repo.Root(), f.runID, true)
	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(f.gh, "enqueued")); err == nil {
		t.Fatalf("the move queued a halted run's pull request:\n%s", f.ghLog(t))
	}
	if p := parkedOf(t, res, f.runID); !strings.Contains(p.Note, "implement resume --take "+f.runID) {
		t.Fatalf("the move does not name the take-over: %+v", p)
	}
}

// TestAHaltedRunsMergedLaneStillFinishes: the forge merged it, and only the finish is left.
func TestAHaltedRunsMergedLaneStillFinishes(t *testing.T) {
	f := drainLandFixture(t)
	first := f.firstMove(t)
	f.runID, f.issue = first.Start.RunID, secondIssue
	l := f.armIssueLaneOnce(t)
	markHalted(t, f.repo.Root(), f.runID, true)
	f.merged(t, l.Landing.Pushed)
	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Finished) != 1 || !res.Finished[0].Done || res.Finished[0].RunID != f.runID {
		t.Fatalf("the merged lane is not finished: %+v", res.Finished)
	}
}

// TestTheFenceRefusesAReservationAStopReleased: a run with no reservation because a stop let
// it go is not a run that never held one.
func TestTheFenceRefusesAReservationAStopReleased(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.toArmOnce(t)
	now := time.Now().UTC()
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		st.ReservedAt = nil
		st.Record = append(st.Record, Entry{At: now, Stage: StageReservationReleased, Note: "released"})
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	r := f.fenced(t)
	if !strings.Contains(r.Reason, "let its reservation") || !strings.Contains(r.Remedy, "implement resume --take "+f.runID) {
		t.Fatalf("the fence does not say so: %+v", r)
	}
}
