package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Merged lanes are finished (spc-2609212015054359 step 3, iss-2610090936460789):
// a lane parked at land whose pushed head is on the default branch is removed
// and marked done on the next drain move, outside the pacing window, since the
// cleanup does no new work.

// drainLandFixture is drainRepo on a local bare remote with a merge-queue
// ruleset mirror and the stub forge client first on PATH, so a drain's lane
// can land.
func drainLandFixture(t *testing.T) *landFixture {
	t.Helper()
	repo := drainRepo(t)
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_COMMITTER_NAME"} {
		t.Setenv(k, "Pat Example")
	}
	for _, k := range []string{"GIT_AUTHOR_EMAIL", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(k, "pat@example.com")
	}
	repo.Write(".abcd/work/rulesets/main-protection.json", queueRuleset("MERGE"))
	repo.Commit("the ruleset mirror")
	bare := filepath.Join(t.TempDir(), "origin.git")
	repo.Git("init", "-q", "--bare", "--initial-branch=main", bare)
	repo.Git("remote", "add", "origin", bare)
	repo.Git("push", "-q", "origin", "main")
	repo.Git("fetch", "-q", "origin")
	gh := t.TempDir()
	if err := os.WriteFile(filepath.Join(gh, "gh"), []byte(stubGH), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", gh+string(os.PathListSeparator)+os.Getenv("PATH"))
	return &landFixture{repo: repo, bare: bare, gh: gh, stages: DefaultStages()}
}

// armIssueLane drives the drain's issue lane from its implement await to an
// armed pull request: a fix commit and a receipt resolving the issue, a
// passing round, the records, the push, the pull request and the arming.
func (f *landFixture) armIssueLane(t *testing.T) Lane {
	t.Helper()
	repo := f.repo
	stepTo(t, repo, f.runID, f.stages, StageImplement)
	if res, err := advance(repo.Root(), f.runID, f.stages, Options{}); err != nil || res.Awaiting == nil {
		t.Fatalf("the implement stage awaits an implementer: %+v %v", res, err)
	}
	l := currentLane(t, repo, f.runID)
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), f.runID, l.ID)
	sha := laneCommit(t, repo, l, "fix.txt")
	rc := goodReceipt(t, f.runID, l, dir, sha)
	rc.Model = "claude-test-5"
	rc.Resolves = []Resolution{{Issue: f.issue, Commit: sha, Note: "the page names the flag", Impact: "fix",
		Grounds: "pursued: the page names the flag; shown wrong if it still does not"}}
	if _, err := Receipt(repo.Root(), f.runID, writeReceipt(t, dir, rc), f.stages, Options{}); err != nil {
		t.Fatal(err)
	}
	passRound(t, repo, f.runID, f.stages, RoleRuthless, RoleSecurity)
	stepTo(t, repo, f.runID, f.stages, StageLand)
	f.step(t)
	f.step(t)
	l = currentLane(t, repo, f.runID)
	preflighted(t, l, l.HeadSHA)
	f.step(t)
	f.step(t)
	f.step(t)
	l = currentLane(t, repo, f.runID)
	if l.Landing == nil || !l.Landing.Armed || l.PR != 7 {
		t.Fatalf("the lane's pull request is armed: %+v", l.Landing)
	}
	return l
}

// pauseBoth puts the drain and the lane's run in a pause until a time an hour
// ahead, as a closed window leaves them.
func pauseBoth(t *testing.T, f *landFixture) time.Time {
	t.Helper()
	until := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	st, live, err := readDrain(f.repo.Root())
	if err != nil || !live {
		t.Fatalf("a live drain: %v", err)
	}
	st.NextEligibleAt = &until
	if err := writeDrain(f.repo.Root(), st); err != nil {
		t.Fatal(err)
	}
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		st.NextEligibleAt = &until
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	return until
}

// TestAMergedLaneParkedAtLandIsFinishedOnTheNextDrainMove: while the pull
// request is open the drain leaves the lane alone and asks the forge nothing;
// once the lane's pushed head is on the default branch, the next drain move,
// made while the drain and the run are both paused, removes the lane's
// worktree and branch, marks the run's lane done and the drain's lane done
// with its pull request, and says so, still opening nothing before
// next_eligible_at.
func TestAMergedLaneParkedAtLandIsFinishedOnTheNextDrainMove(t *testing.T) {
	f := drainLandFixture(t)
	first := drainToLane(t, f.repo, Options{}, DrainOptions{})
	if first.Start == nil || first.Lane.Issue != secondIssue {
		t.Fatalf("the drain opens the documentation issue's lane: %+v", first)
	}
	f.runID, f.issue = first.Start.RunID, secondIssue
	l := f.armIssueLane(t)

	waiting, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if waiting.Lanes[0].Outcome != DrainLanePullRequest || len(waiting.Finished) != 0 {
		t.Fatalf("an armed lane not yet merged is a pull request, not finished: %+v %+v", waiting.Lanes, waiting.Finished)
	}
	if strings.Contains(f.ghLog(t), "--json state") {
		t.Fatalf("the drain asks the forge nothing about a lane whose head is not on the default branch:\n%s", f.ghLog(t))
	}
	if _, err := os.Stat(l.Worktree); err != nil {
		t.Fatalf("an unmerged lane keeps its worktree: %v", err)
	}

	until := pauseBoth(t, f)
	f.merged(t, l.Landing.Pushed)
	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatalf("the drain move during the pause: %v", err)
	}
	if len(res.Finished) != 1 || !res.Finished[0].Done || res.Finished[0].RunID != f.runID || res.Finished[0].Issue != secondIssue || res.Finished[0].PR != 7 {
		t.Fatalf("the merged lane is finished on the next drain move: %+v", res.Finished)
	}
	if _, err := os.Stat(l.Worktree); !os.IsNotExist(err) {
		t.Fatalf("the merged lane's worktree is removed: %v", err)
	}
	if out := f.repo.Git("branch", "--list", l.Branch); strings.TrimSpace(out) != "" {
		t.Fatalf("the merged lane's branch is deleted: %q", out)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Complete() || st.Lanes[0].Landing.Merged == "" {
		t.Fatalf("the run's lane is done, its landing naming the merge: %+v", st.Lanes[0])
	}
	if st.NextEligibleAt == nil || !st.NextEligibleAt.Equal(until) {
		t.Fatalf("finishing the lane leaves the run's pause as it was: %v", st.NextEligibleAt)
	}
	if res.Lanes[0].Outcome != DrainLaneDone || res.Lanes[0].PR != 7 {
		t.Fatalf("the drain marks its lane done with its pull request: %+v", res.Lanes[0])
	}
	if res.NextEligibleAt == nil || !res.NextEligibleAt.Equal(until) || res.Start != nil {
		t.Fatalf("the drain is still paused and opens nothing: %+v", res)
	}
	if !strings.Contains(res.Next, "finished") || !strings.Contains(res.Next, f.runID) || !strings.Contains(res.Next, "nothing before") {
		t.Fatalf("the next move names the finished lane and the pause: %s", res.Next)
	}

	again, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil || len(again.Finished) != 0 {
		t.Fatalf("a finished lane is not finished twice: %+v %v", again.Finished, err)
	}
}

// TestAMergedLaneOfAnEndedDrainIsFinishedByTheNextDrain: the lanes of a drain
// that has ended (here, the person's own `abcd build <iss-N>` run, which no
// drain state names) are finished by the next drain move all the same, since
// the cleanup reads the checkout's runs, not one drain's lanes.
func TestAMergedLaneOfAnEndedDrainIsFinishedByTheNextDrain(t *testing.T) {
	f := drainLandFixture(t)
	start, err := Start(f.repo.Root(), secondIssue, Options{})
	if err != nil {
		t.Fatal(err)
	}
	f.runID, f.issue = start.RunID, secondIssue
	l := f.armIssueLane(t)
	f.merged(t, l.Landing.Pushed)

	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Finished) != 1 || !res.Finished[0].Done || res.Finished[0].RunID != f.runID {
		t.Fatalf("a merged lane no drain names is finished by the drain move: %+v", res.Finished)
	}
	if st, err := ReadState(f.repo.Root(), f.runID); err != nil || !st.Complete() {
		t.Fatalf("its run is complete: %+v %v", st, err)
	}
}

// TestAMergedLaneTheLoopCannotProveIsNamedNotFatal: a lane whose branch moved
// past what landed is not cleaned up; the drain names the refusal beside its
// move and goes on.
func TestAMergedLaneTheLoopCannotProveIsNamedNotFatal(t *testing.T) {
	f := drainLandFixture(t)
	start, err := Start(f.repo.Root(), secondIssue, Options{})
	if err != nil {
		t.Fatal(err)
	}
	f.runID, f.issue = start.RunID, secondIssue
	l := f.armIssueLane(t)
	f.merged(t, l.Landing.Pushed)
	laneCommit(t, f.repo, l, "late.txt")

	res, err := Drain(f.repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatalf("a lane the loop cannot finish does not refuse the drain: %v", err)
	}
	if len(res.Finished) != 1 || res.Finished[0].Done || !strings.Contains(res.Finished[0].Note, "past what landed") {
		t.Fatalf("the refusal is named: %+v", res.Finished)
	}
	if _, err := os.Stat(l.Worktree); err != nil {
		t.Fatalf("nothing is cleaned up: %v", err)
	}
	if !strings.Contains(res.Next, "could not finish") {
		t.Fatalf("the next move names the lane it could not finish: %s", res.Next)
	}
}
