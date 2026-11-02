package loop

// runner_killed_test.go proves the part of iss-2611021751406429 the loop
// owns: an implementer its runner killed after it committed its work and
// wrote its report keeps the lane at its stage for its receipt, rather than
// counting a failed start and pausing the lane; one killed with no work
// standing pauses as before; and either way what the killed launch spent is
// kept, never dropped.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/gittest"
)

// killOnClaude drives the implement stage to the claude runner, which hangs
// past its bound and is killed. The lane's worktree is a repository whose
// branch carries commits commits past the lane's base, and the implementer's
// report stands beside its receipt when report is set.
func killOnClaude(t *testing.T, commits int, report bool) (root, id string, res StepResult) {
	t.Helper()
	repo := loopRepo(t, readyIntent("", settledQuestions), specWithSteps(""))
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), "brief.md"), []byte("# brief\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	wt := gittest.NewRepo(t)
	wt.Write("base.txt", "base\n")
	wt.Commit("base")
	base := strings.TrimSpace(wt.Git("rev-parse", "HEAD"))
	for i := range commits {
		wt.Write("work.txt", strings.Repeat("work\n", i+1))
		wt.Commit("work")
	}
	steps := driveSteps(wt.Root())
	cut := steps[0].Run
	steps[0].Run = func(c Context, l *Lane) (Outcome, error) {
		out, err := cut(c, l)
		l.BaseSHA = base
		return out, err
	}
	if report {
		// driveSteps' receipt is at the checkout's root: the report stands
		// beside it, as a lane's stands beside its receipt.
		if err := os.WriteFile(filepath.Join(repo.Root(), ReportFileName), []byte("# report\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	clock := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	o := Options{Now: func() time.Time { return clock }}
	for range 2 {
		if _, err := advance(repo.Root(), start.RunID, steps, o); err != nil {
			t.Fatal(err)
		}
	}
	newDriveEnv(t, "hang", "claude")
	res, err = Drive(context.Background(), repo.Root(), start.RunID, steps, o,
		Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}, Timeout: 2 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return repo.Root(), start.RunID, res
}

// TestAnImplementerKilledWithItsWorkCommittedKeepsTheLane: the lane stays at
// its stage, the await out and not paused, no fallback receipt counts a
// failed start, and the record names the kill, the commits and the spend.
func TestAnImplementerKilledWithItsWorkCommittedKeepsTheLane(t *testing.T) {
	root, id, res := killOnClaude(t, 2, true)
	if res.Paused != nil {
		t.Fatalf("a role killed with its work standing is not a failed start, so the lane is not paused: %s", res.Next)
	}
	if res.Awaiting == nil || res.Awaiting.Role != RoleImplementer || res.Stage != StageImplement {
		t.Fatalf("the lane stays at its stage, the implementer's receipt awaited: %+v", res)
	}
	if !strings.Contains(res.Next, "killed") || !strings.Contains(res.Next, "receipt") {
		t.Errorf("the step says the role was killed with its work standing and its receipt missing: %s", res.Next)
	}
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Fallbacks) != 0 {
		t.Errorf("no failed start is counted: %+v", st.Fallbacks)
	}
	a := st.Lanes[0].Awaits[0]
	if a.Paused != nil || a.Running != nil {
		t.Errorf("the await is neither paused nor held by a runner: %+v", a)
	}
	if !recordSays(t, root, id, StageKilled, "2 commit(s)", "report", "charged as 1 agent run of the claude runner's budget") {
		t.Errorf("the record names the kill, the commits and the spend: %+v", st.Record)
	}
}

// TestAnImplementerKilledWithNoWorkStandingPausesAndKeepsItsSpend: with no
// commit past the base, or with no report, the kill is a failed start as
// before, and its fallback receipt keeps what it spent.
func TestAnImplementerKilledWithNoWorkStandingPausesAndKeepsItsSpend(t *testing.T) {
	for name, tc := range map[string]struct {
		commits int
		report  bool
	}{"no commits": {0, true}, "no report": {1, false}} {
		t.Run(name, func(t *testing.T) {
			root, id, res := killOnClaude(t, tc.commits, tc.report)
			if res.Paused == nil {
				t.Fatalf("a kill with no work standing pauses the lane: %+v", res)
			}
			st, err := ReadState(root, id)
			if err != nil {
				t.Fatal(err)
			}
			if len(st.Fallbacks) != 1 || st.Fallbacks[0].Spent == nil || st.Fallbacks[0].Spent.ChargedRuns != 1 {
				t.Fatalf("the failed start's receipt keeps the killed launch's spend: %+v", st.Fallbacks)
			}
			if c := runner.Tally(st.Fallbacks); c.ChargedRuns != 1 {
				t.Errorf("the tally counts it: %+v", c)
			}
			if rec, err := ReadRecord(root, id); err != nil || rec.FallbackCounts.ChargedRuns != 1 {
				t.Errorf("the run record's tally counts it: %+v %v", rec.FallbackCounts, err)
			}
			// A fallback's `spent` is state version 19's: a version-18 file
			// carrying one is refused naming it.
			path := filepath.Join(root, filepath.FromSlash(StateRelPath(id)))
			old := strings.Replace(string(stateBytes(t, root, id)), fmt.Sprintf(`"schema_version": %d,`, SchemaVersion), `"schema_version": 18,`, 1)
			if err := os.WriteFile(path, []byte(old), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := ReadState(root, id); err == nil || !strings.Contains(err.Error(), "`spent`") {
				t.Fatalf("a version-18 file carrying a spend is refused naming it: %v", err)
			}
		})
	}
}

// TestARunnersClaimOutlastsItsLongestRun: the claim a step takes on the await
// it starts a runner for stands past the longest the run can last, its bound
// and the most its clock stands still at the load gate, so no second step
// starts a second runner while the first waits there.
func TestARunnersClaimOutlastsItsLongestRun(t *testing.T) {
	root, id, steps, o := startToImplement(t)
	newDriveEnv(t, "hang", "claude")
	const bound = 2 * time.Second
	done := make(chan error, 1)
	go func() {
		_, err := Drive(context.Background(), root, id, steps, o,
			Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}, Timeout: bound})
		done <- err
	}()
	var claim *RunnerClaim
	for deadline := time.Now().Add(10 * time.Second); claim == nil && time.Now().Before(deadline); {
		if st, err := ReadState(root, id); err == nil && len(st.Lanes) > 0 && len(st.Lanes[0].Awaits) > 0 {
			claim = st.Lanes[0].Awaits[0].Running
		}
		if claim == nil {
			time.Sleep(20 * time.Millisecond)
		}
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if claim == nil {
		t.Fatal("no claim was seen while the runner ran")
	}
	if want := o.now().Add(runner.LongestRun(bound) + claimGrace); !claim.Until.Equal(want) {
		t.Errorf("the claim lapses at %s, want %s: past the run's longest", claim.Until, want)
	}
}
