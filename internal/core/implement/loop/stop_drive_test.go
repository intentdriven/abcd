package loop

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/claim"
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

// TestAHaltedRunsMergedLaneWaitsForItsTakeOver: the forge merged it, and the finish is the
// run's own move, which the take-over gives it back.
func TestAHaltedRunsMergedLaneWaitsForItsTakeOver(t *testing.T) {
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
	if i := slices.IndexFunc(res.Finished, func(d DrainFinished) bool { return d.RunID == f.runID }); i >= 0 {
		t.Fatalf("the move took a halted run's last step: %+v", res.Finished)
	}
	if ld := currentLane(t, f.repo, f.runID); ld.Stage != StageLand {
		t.Fatalf("the lane left its landing: %+v", ld)
	}
	if p := parkedOf(t, res, f.runID); p.State != ParkedStopHolds || !strings.Contains(p.Note, "implement resume --take "+f.runID) {
		t.Fatalf("the move does not name the take-over: %+v", p)
	}
	if !strings.Contains(res.Next, f.runID) {
		t.Fatalf("next does not name the run: %s", res.Next)
	}
}

// markUnsettled leaves the run a person's stop's mark with work a save could not take, so it
// has not let its claims go.
func markUnsettled(t *testing.T, repoRoot, runID string) {
	t.Helper()
	now := time.Now().UTC()
	if err := mutate(repoRoot, runID, func(root *os.Root, _ *State) (bool, error) {
		c, err := readControl(root, runID)
		if err != nil {
			return false, err
		}
		c.Stop = &RunStop{SetBy: "someone", SetAt: now, NotedAt: now, FromFile: true,
			Left: []StopLeft{{Lane: "lane-1", Stage: StageLand, Note: "its work could not be saved aside"}}}
		return false, writeControl(root, runID, c)
	}); err != nil {
		t.Fatal(err)
	}
}

// aReadThatRefusesAStopFile is the options of a move whose read of a stop refused what stands
// there, which is no stop and no lift, so a mark stays.
func aReadThatRefusesAStopFile(t *testing.T) Options {
	t.Helper()
	home := t.TempDir()
	p := abcdhome.Path(home, runstop.HomeStopFile)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(home, "elsewhere"), p); err != nil {
		t.Fatal(err)
	}
	return Options{stopHome: home}
}

// TestAMarkedParkedRunIsLeftAloneBeforeItSettles: a run with the mark and no halt yet is not
// synced, queued or finished by a drain's move, which names it and what takes it up.
func TestAMarkedParkedRunIsLeftAloneBeforeItSettles(t *testing.T) {
	named := func(t *testing.T, res DrainResult, runID string) {
		t.Helper()
		if i := slices.IndexFunc(res.Finished, func(d DrainFinished) bool { return d.RunID == runID }); i >= 0 {
			t.Fatalf("the move finished a marked run: %+v", res.Finished)
		}
		if p := parkedOf(t, res, runID); p.State != ParkedStopHolds || !strings.Contains(p.Note, runID) || !strings.Contains(p.Note, "implement step --run "+runID) {
			t.Fatalf("the move does not name what takes it up: %+v", p)
		}
	}
	t.Run("a conflict it would sync", func(t *testing.T) {
		f, l := armedConflict(t)
		markUnsettled(t, f.repo.Root(), f.runID)
		res, err := Drain(f.repo.Root(), aReadThatRefusesAStopFile(t), DrainOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(f.ghLog(t), "--disable-auto") {
			t.Fatalf("the move disarmed a marked run:\n%s", f.ghLog(t))
		}
		if ld := currentLane(t, f.repo, f.runID).Landing; !ld.Armed || ld.Pushed != l.Landing.Pushed {
			t.Fatalf("the move changed the lane: %+v", ld)
		}
		named(t, res, f.runID)
	})
	t.Run("a pull request it would queue", func(t *testing.T) {
		f := drainLandFixture(t)
		first := drainToLane(t, f.repo, Options{}, DrainOptions{})
		f.runID, f.issue = first.Start.RunID, secondIssue
		l := f.armIssueLane(t)
		f.writeGH(t, "pr.json", []any{map[string]any{"number": 7, "headRefName": l.Branch, "state": "OPEN", "mergeable": "MERGEABLE", "mergeStateStatus": "CLEAN",
			"autoMergeRequest": map[string]any{"mergeMethod": "MERGE"}}})
		f.queueAnswer(t, stubQueue{head: l.Landing.Pushed, enabledAt: "2026-10-15T07:00:00Z", status: "CLEAN"})
		markUnsettled(t, f.repo.Root(), f.runID)
		res, err := Drain(f.repo.Root(), aReadThatRefusesAStopFile(t), DrainOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(f.gh, "enqueued")); err == nil {
			t.Fatalf("the move queued a marked run's pull request:\n%s", f.ghLog(t))
		}
		named(t, res, f.runID)
	})
	t.Run("a lane that merged", func(t *testing.T) {
		f := drainLandFixture(t)
		first := f.firstMove(t)
		f.runID, f.issue = first.Start.RunID, secondIssue
		l := f.armIssueLaneOnce(t)
		markUnsettled(t, f.repo.Root(), f.runID)
		f.merged(t, l.Landing.Pushed)
		res, err := Drain(f.repo.Root(), aReadThatRefusesAStopFile(t), DrainOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if ld := currentLane(t, f.repo, f.runID); ld.Stage != StageLand {
			t.Fatalf("the lane left its landing: %+v", ld)
		}
		named(t, res, f.runID)
	})
}

// TestAMarkedParkedRunIsRenewedWhileTheReadRefusesAStopFile: with no stop seen and no lift
// either, the move renews the reservation of a run with work a save could not take, whatever
// else it leaves undone for the run.
func TestAMarkedParkedRunIsRenewedWhileTheReadRefusesAStopFile(t *testing.T) {
	f, l := parkDrainLane(t)
	f.listPRs(t, stubPR{Number: 7, HeadRefName: l.Branch, State: "OPEN", Mergeable: "MERGEABLE", MergeStateStatus: "BLOCKED"})
	ref := claim.RefPrefix + secondIssue
	fenced := originReservation(t, f.bare, ref)
	at := fenced.claimed.Add(12*time.Hour + 30*time.Minute)
	markUnsettled(t, f.repo.Root(), f.runID)
	o := aReadThatRefusesAStopFile(t)
	o.Now = func() time.Time { return at }
	res, err := Drain(f.repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if tip := originReservation(t, f.bare, ref); tip.parent != fenced.commit || tip.released || tip.claimed.Before(at) {
		t.Fatalf("the claim is not renewed: fenced %+v, now %+v", fenced, tip)
	}
	if p := parkedOf(t, res, f.runID); p.State != ParkedStopHolds {
		t.Fatalf("the lane is not named: %+v", p)
	}
}

// TestAReadThatHoldsTheRunRenewsNoParkedClaim: a read that holds the run refuses the move
// before any renewal.
func TestAReadThatHoldsTheRunRenewsNoParkedClaim(t *testing.T) {
	f, l := parkDrainLane(t)
	f.listPRs(t, stubPR{Number: 7, HeadRefName: l.Branch, State: "OPEN", Mergeable: "MERGEABLE", MergeStateStatus: "BLOCKED"})
	ref := claim.RefPrefix + secondIssue
	fenced := originReservation(t, f.bare, ref)
	home, _ := stopWorld(t, true)
	at := fenced.claimed.Add(12*time.Hour + 30*time.Minute)
	o := Options{stopHome: home, Now: func() time.Time { return at }}
	if err := checkOnce(f.repo.Root(), f.runID, o); err != nil {
		t.Fatal(err)
	}
	markUnsettled(t, f.repo.Root(), f.runID)
	if err := os.Remove(settingOf(home)); err != nil {
		t.Fatal(err)
	}
	_, err := Drain(f.repo.Root(), o, DrainOptions{})
	wantStopRefusal(t, err)
	if tip := originReservation(t, f.bare, ref); tip.commit != fenced.commit || tip.released {
		t.Fatalf("the claim moved: fenced %+v, now %+v", fenced, tip)
	}
}

// TestAStopInForceRenewsNoParkedClaim: under a stop the move refuses before any renewal and
// no process stays to renew: a settled run releases its reservation, and one with work a save
// could not take leaves it to lapse at its lease. The lane's work stays on its branch and in
// its worktree either way.
func TestAStopInForceRenewsNoParkedClaim(t *testing.T) {
	stands := func(t *testing.T, f *landFixture, l Lane) {
		t.Helper()
		if ld := currentLane(t, f.repo, f.runID); ld.Stage != StageLand || ld.Landing.Pushed != l.Landing.Pushed {
			t.Fatalf("the lane moved: %+v", ld)
		}
		if _, err := os.Stat(l.Worktree); err != nil {
			t.Fatalf("the lane's worktree: %v", err)
		}
		if got := f.repo.Git("rev-parse", "refs/heads/"+l.Branch); strings.TrimSpace(got) != l.Landing.Pushed {
			t.Fatalf("the lane's branch is at %s, want %s", got, l.Landing.Pushed)
		}
	}
	t.Run("work a save could not take", func(t *testing.T) {
		f, l := parkDrainLane(t)
		f.listPRs(t, stubPR{Number: 7, HeadRefName: l.Branch, State: "OPEN", Mergeable: "MERGEABLE", MergeStateStatus: "BLOCKED"})
		ref := claim.RefPrefix + secondIssue
		fenced := originReservation(t, f.bare, ref)
		left := filepath.Join(l.Worktree, "unsaved.txt")
		if err := os.WriteFile(left, []byte("work in hand\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		old := checkAsidePatch
		checkAsidePatch = func(string, string) error { return errors.New("will not apply") }
		t.Cleanup(func() { checkAsidePatch = old })
		o, _, _ := stopped(t)
		for _, after := range []time.Duration{12*time.Hour + 30*time.Minute, 23 * time.Hour} {
			at := fenced.claimed.Add(after)
			o.Now = func() time.Time { return at }
			_, err := Drain(f.repo.Root(), o, DrainOptions{})
			wantStopRefusal(t, err)
			if tip := originReservation(t, f.bare, ref); tip.commit != fenced.commit || tip.released {
				t.Fatalf("%s on: the claim moved: fenced %+v, now %+v", after, fenced, tip)
			}
		}
		c := controlOf(t, f.repo.Root(), f.runID)
		if c.Stop == nil || c.Stop.HaltedAt != nil {
			t.Fatalf("the run has settled: %+v", c.Stop)
		}
		if _, err := os.Stat(left); err != nil {
			t.Fatalf("the work in hand: %v", err)
		}
		stands(t, f, l)
	})
	t.Run("a run that settles", func(t *testing.T) {
		f, l := parkDrainLane(t)
		f.listPRs(t, stubPR{Number: 7, HeadRefName: l.Branch, State: "OPEN", Mergeable: "MERGEABLE", MergeStateStatus: "BLOCKED"})
		ref := claim.RefPrefix + secondIssue
		fenced := originReservation(t, f.bare, ref)
		o, _, _ := stopped(t)
		at := fenced.claimed.Add(12*time.Hour + 30*time.Minute)
		o.Now = func() time.Time { return at }
		_, err := Drain(f.repo.Root(), o, DrainOptions{})
		wantStopRefusal(t, err)
		if tip := originReservation(t, f.bare, ref); !tip.released {
			t.Fatalf("the claim is not released: fenced %+v, now %+v", fenced, tip)
		}
		stands(t, f, l)
	})
}

// TestAMarkArrivingDuringTheFetchLeavesTheRunAlone: the mark is read again once the forge has
// been listed and the default branch fetched, which take time.
func TestAMarkArrivingDuringTheFetchLeavesTheRunAlone(t *testing.T) {
	f := drainLandFixture(t)
	first := f.firstMove(t)
	f.runID, f.issue = first.Start.RunID, secondIssue
	l := f.armIssueLaneOnce(t)
	f.merged(t, l.Landing.Pushed)
	old := afterParkedFetch
	afterParkedFetch = func() { markHalted(t, f.repo.Root(), f.runID, true) }
	t.Cleanup(func() { afterParkedFetch = old })
	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if i := slices.IndexFunc(res.Finished, func(d DrainFinished) bool { return d.RunID == f.runID }); i >= 0 {
		t.Fatalf("the move took the run's last step: %+v", res.Finished)
	}
	if ld := currentLane(t, f.repo, f.runID); ld.Stage != StageLand {
		t.Fatalf("the lane left its landing: %+v", ld)
	}
	if p := parkedOf(t, res, f.runID); p.State != ParkedStopHolds || !strings.Contains(p.Note, "implement resume --take "+f.runID) {
		t.Fatalf("the move does not name it: %+v", p)
	}
}

// TestATakeOverFinishesAParkedRunsMergedLane: a stop halts the parked run, the forge merges it
// while it waits, a drain's move leaves it after the lift, and after the take-over the next
// move takes the last step, its worktree and branch going with it.
func TestATakeOverFinishesAParkedRunsMergedLane(t *testing.T) {
	f := drainLandFixture(t)
	first := f.firstMove(t)
	f.runID, f.issue = first.Start.RunID, secondIssue
	l := f.armIssueLaneOnce(t)
	o, home, _ := stopped(t)
	_, err := Drain(f.repo.Root(), o, DrainOptions{})
	wantStopRefusal(t, err)
	if c := controlOf(t, f.repo.Root(), f.runID); c.Stop == nil || c.Stop.HaltedAt == nil {
		t.Fatalf("the run did not halt: %+v", c.Stop)
	}
	f.merged(t, l.Landing.Pushed)
	if lifted, err := runstop.Lift(home, time.Now()); err != nil || !lifted {
		t.Fatalf("the lift: %v, %v", lifted, err)
	}
	waited, err := Drain(f.repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if p := parkedOf(t, waited, f.runID); p.State != ParkedStopHolds || !strings.Contains(p.Note, "implement resume --take "+f.runID) {
		t.Fatalf("the move does not name the take-over: %+v", p)
	}
	if _, err := TakeRun(f.repo.Root(), f.runID, o); err != nil {
		t.Fatalf("the take-over: %v", err)
	}
	res, err := Drain(f.repo.Root(), o, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(res.Finished, func(d DrainFinished) bool { return d.RunID == f.runID })
	if i < 0 || !res.Finished[i].Done {
		t.Fatalf("the merged lane is not finished after the take-over: %+v, %s", res.Finished, res.Next)
	}
	if _, err := os.Stat(l.Worktree); !os.IsNotExist(err) {
		t.Fatalf("the lane's worktree stands: %v", err)
	}
	if got := f.repo.Git("for-each-ref", "refs/heads/"+l.Branch); strings.TrimSpace(got) != "" {
		t.Fatalf("the lane's branch stands: %s", got)
	}
}

// TestAnUnsettledMarkNamesWhatTakesItUp: a person's stop lifts and a step goes on; a signal has
// nothing to lift, so its mark is taken over.
func TestAnUnsettledMarkNamesWhatTakesItUp(t *testing.T) {
	f, _ := armedConflict(t)
	mark := func(fromFile bool) {
		now := time.Now().UTC()
		if err := mutate(f.repo.Root(), f.runID, func(root *os.Root, _ *State) (bool, error) {
			c, err := readControl(root, f.runID)
			if err != nil {
				return false, err
			}
			c.Stop = &RunStop{SetBy: "someone", SetAt: now, NotedAt: now, FromFile: fromFile}
			return false, writeControl(root, f.runID, c)
		}); err != nil {
			t.Fatal(err)
		}
	}
	mark(true)
	if note := stopMarkNote(f.repo.Root(), f.runID); !strings.Contains(note, "once the stop lifts") || !strings.Contains(note, "implement step --run "+f.runID) {
		t.Fatalf("a person's stop: %s", note)
	}
	mark(false)
	note := stopMarkNote(f.repo.Root(), f.runID)
	if !strings.Contains(note, "implement resume --take "+f.runID) || strings.Contains(note, "lifts") || strings.Contains(note, "implement step") {
		t.Fatalf("a signal: %s", note)
	}
}

// TestParkedNoteSaysWhatItRead: a lane left to a stop, or one whose pull request could not be
// read, is not among those read.
func TestParkedNoteSaysWhatItRead(t *testing.T) {
	note := parkedNote([]DrainParked{
		{Issue: "iss-1", RunID: "run-1", State: ParkedWaiting, Note: "waiting"},
		{Issue: "iss-2", RunID: "run-2", State: ParkedStopHolds, Note: "a stop"},
		{Issue: "iss-3", RunID: "run-3", State: ParkedUnread, Note: "no answer"},
	})
	for _, want := range []string{
		"read 1 parked lane(s) from their pull requests: iss-1's run run-1: waiting",
		"could not read the pull requests of 1 parked lane(s): iss-3's run run-3: no answer",
		"left 1 parked lane(s) as a stop's mark holds them: iss-2's run run-2: a stop",
	} {
		if !strings.Contains(note, want) {
			t.Errorf("the note does not say %q: %s", want, note)
		}
	}
	if only := parkedNote([]DrainParked{{Issue: "iss-2", RunID: "run-2", State: ParkedStopHolds, Note: "a stop"}}); strings.Contains(only, "from their pull requests") {
		t.Errorf("a lane it did not read is said to be read: %s", only)
	}
}

// TestAParkedRunGoesOnFromAStopLiftedBeforeItSettled: a read that saw clearly and found no
// stop is the lift, as at every move, and the run is then as any other.
func TestAParkedRunGoesOnFromAStopLiftedBeforeItSettled(t *testing.T) {
	f := drainLandFixture(t)
	first := f.firstMove(t)
	f.runID, f.issue = first.Start.RunID, secondIssue
	l := f.armIssueLaneOnce(t)
	markUnsettled(t, f.repo.Root(), f.runID)
	f.merged(t, l.Landing.Pushed)
	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Finished) != 1 || !res.Finished[0].Done || res.Finished[0].RunID != f.runID {
		t.Fatalf("the merged lane is not finished: %+v", res.Finished)
	}
	if c, err := runControl(f.repo.Root(), f.runID); err != nil || c.Stop != nil {
		t.Fatalf("the mark stands: %+v, %v", c.Stop, err)
	}
}

// TestTheFenceRefusesAReservationAStopReleased: a run a stop halted has let its reservation
// go, and the take-over the fence names reserves it again.
func TestTheFenceRefusesAReservationAStopReleased(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.toArmOnce(t)
	markHalted(t, f.repo.Root(), f.runID, true)
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = fence(Context{RepoRoot: f.repo.Root(), State: st, Now: time.Now().UTC()}, st.Lanes[0], fenceLease, fenceQueue)
	r, ok := AsRefusal(err)
	if !ok || !r.Contention || !strings.Contains(r.Reason, "let its reservation") || !strings.Contains(r.Remedy, "implement resume --take "+f.runID) {
		t.Fatalf("the fence does not say so: %v", err)
	}
}

// letGoWithNoMark has the run let its reservation go with no stop's mark standing.
func letGoWithNoMark(t *testing.T, f *landFixture) time.Time {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-30*time.Minute), "internal/widget")
	if _, err := claim.Reserve(f.repo.Root(), claim.Request{Record: "itd-10", Release: true}, clockAt(now.Add(-time.Minute))); err != nil {
		t.Fatal(err)
	}
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		st.ReservedAt = nil
		st.Record = append(st.Record, Entry{At: now, Stage: StageReservationReleased, Note: "released"})
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	return now
}

// TestTheFenceSendsAReservationLetGoToReserve: with no mark standing a take-over reserves
// nothing, so the fence refuses and names what does.
func TestTheFenceSendsAReservationLetGoToReserve(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.toArmOnce(t)
	letGoWithNoMark(t, f)
	r := f.fenced(t)
	for _, want := range []string{"abcd reserve itd-10 --footprint", "--discard " + l.ID} {
		if !strings.Contains(r.Remedy, want) {
			t.Errorf("the remedy lacks %q: %+v", want, r)
		}
	}
	if strings.Contains(r.Remedy, "--take") {
		t.Fatalf("the remedy would come back to the fence: %+v", r)
	}
}

// TestTheRemedyForAReservationLetGoWorks: each thing the refusal names gets the run past it.
func TestTheRemedyForAReservationLetGoWorks(t *testing.T) {
	t.Run("reserve", func(t *testing.T) {
		f := newLandFixture(t, queueRuleset("MERGE"))
		f.toArmOnce(t)
		now := letGoWithNoMark(t, f)
		f.fenced(t)
		if _, err := claim.Reserve(f.repo.Root(), claim.Request{Record: "itd-10", Footprint: []string{"internal/widget"}}, clockAt(now)); err != nil {
			t.Fatal(err)
		}
		if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(f.ghLog(t), "pr merge") {
			t.Fatalf("the reserved run did not arm:\n%s", f.ghLog(t))
		}
	})
	t.Run("discard", func(t *testing.T) {
		f := newLandFixture(t, queueRuleset("MERGE"))
		l := f.toArmOnce(t)
		letGoWithNoMark(t, f)
		f.fenced(t)
		if _, err := Discard(f.repo.Root(), f.runID, l.ID, Options{}); err != nil {
			t.Fatal(err)
		}
		if got := onlyLaneStage(t, f); got != StageDiscarded {
			t.Fatalf("the lane is at %s", got)
		}
	})
}

// onlyLaneStage is the fixture's one lane's stage.
func onlyLaneStage(t *testing.T, f *landFixture) Stage {
	t.Helper()
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	return st.Lanes[0].Stage
}

// TestAMarkedRunIsNotDiscardedAsLetGo: a stop's mark keeps its own remedy.
func TestAMarkedRunIsNotDiscardedAsLetGo(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.toArmOnce(t)
	letGoWithNoMark(t, f)
	markHalted(t, f.repo.Root(), f.runID, true)
	if _, err := Discard(f.repo.Root(), f.runID, l.ID, Options{}); err == nil {
		t.Fatal("a lane a stop's mark holds was discarded as let go")
	}
}

// TestARunThatNeverReservedPassesTheFence: nothing was let go, so nothing is asked for.
func TestARunThatNeverReservedPassesTheFence(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.toArmOnce(t)
	if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.ghLog(t), "pr merge") {
		t.Fatalf("the run did not arm:\n%s", f.ghLog(t))
	}
}

// TestTheFenceRefusesOnAnUnreadableControlRecord: what cannot be read is said, and does not
// arm on the chance that no mark stands.
func TestTheFenceRefusesOnAnUnreadableControlRecord(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.toArmOnce(t)
	if err := os.WriteFile(filepath.Join(f.repo.Root(), filepath.FromSlash(controlRel(f.runID))), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = fence(Context{RepoRoot: f.repo.Root(), State: st, Now: time.Now().UTC()}, st.Lanes[0], fenceLease, fenceQueue)
	if r, ok := AsRefusal(err); !ok || !r.Contention || !strings.Contains(r.Reason, "cannot be read") {
		t.Fatalf("the fence does not say so: %v", err)
	}
}

// signalAtTheArmingStep ends a drive on a signal the moment its first lane arms.
func signalAtTheArmingStep(t *testing.T) (*terminalWorld, TerminalResult) {
	t.Helper()
	w := newTerminalWorld(t, "shape")
	w.realWorktree()
	armed := false
	for i := range w.stages {
		if w.stages[i].Name == StageLand {
			w.stages[i].Run = func(_ Context, l *Lane) (Outcome, error) {
				if armed {
					return Outcome{Note: "land done"}, nil
				}
				armed = true
				l.PR, l.Landing = 7, &Landing{Pushed: strings.Repeat("a", 40), Merge: "armed", Armed: true}
				close(w.stop)
				return Outcome{Stay: true, Note: "pull request #7 is armed"}, nil
			}
		}
	}
	res, err := RunTerminal(context.Background(), w.repo.Root(), w.options(t))
	if err != nil {
		t.Fatal(err)
	}
	if res.Ended != TerminalStopped || res.Run == "" {
		t.Fatalf("the run ends on the signal: %+v\n%s", res, strings.Join(w.lines, "\n"))
	}
	if c := controlOf(t, w.repo.Root(), res.Run); c.Stop == nil || c.Stop.FromFile || c.Stop.HaltedAt == nil {
		t.Fatalf("the signal's stop: %+v", c.Stop)
	}
	return w, res
}

// laterDrivePassesOver wants a later drive to name the run as a person's to take over.
func laterDrivePassesOver(t *testing.T, w *terminalWorld, res TerminalResult) {
	t.Helper()
	next := w.options(t)
	next.Stop = make(chan struct{})
	later, err := RunTerminal(context.Background(), w.repo.Root(), next)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(later.PassedOver, func(p Excluded) bool { return p.ID == res.Issue })
	if i < 0 || later.PassedOver[i].Check != CheckNeedsPerson || !strings.Contains(later.PassedOver[i].Reason, "implement resume --take "+res.Run) {
		t.Fatalf("a later drive does not name it: %+v\n%s", later.PassedOver, strings.Join(w.lines, "\n"))
	}
}

// TestASignalAtTheArmingStepWaitsForItsTakeOver: a later drive takes up no parked lane, so
// the end names no command that would, and that drive names the run and goes on.
func TestASignalAtTheArmingStepWaitsForItsTakeOver(t *testing.T) {
	w, res := signalAtTheArmingStep(t)
	if res.Resume != "" {
		t.Fatalf("the end names a command that takes up no parked lane: %+v", res)
	}
	if !strings.Contains(strings.Join(w.lines, "\n"), "implement resume --take "+res.Run) {
		t.Fatalf("the end does not name the take-over:\n%s", strings.Join(w.lines, "\n"))
	}
	laterDrivePassesOver(t, w, res)
}

// TestANewDrainNamesARunASignalParked: a drain begun since has no lane of that issue in hand,
// and is told the same.
func TestANewDrainNamesARunASignalParked(t *testing.T) {
	w, res := signalAtTheArmingStep(t)
	if err := os.Remove(filepath.Join(w.repo.Root(), filepath.FromSlash(DrainStateRel))); err != nil {
		t.Fatal(err)
	}
	laterDrivePassesOver(t, w, res)
}

// TestADriveNamesAParkedRunItLeftAlone: what is not finished is not said to be, says what it
// is once, and is no movement.
func TestADriveNamesAParkedRunItLeftAlone(t *testing.T) {
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
	var lines []string
	tm := newTerminal(f.repo.Root(), TerminalOptions{Stages: DefaultStages(), Say: func(s string) { lines = append(lines, s) }})
	tm.sayMove(res)
	tm.sayMove(res)
	said := strings.Join(lines, "\n")
	if strings.Contains(said, "finished ") {
		t.Fatalf("a run it left alone is said to be finished:\n%s", said)
	}
	if strings.Count(said, "a drive stopped on") != 1 {
		t.Fatalf("the run is not named once:\n%s", said)
	}
	if drainMoved(res) {
		t.Fatalf("a run it left alone reads as movement: %+v", res.Finished)
	}
	done := res
	done.Finished = []DrainFinished{{Issue: "iss-1", RunID: f.runID, Done: true}}
	if !drainMoved(done) {
		t.Fatal("a finished lane is movement")
	}
}
