package loop

// ratelimit_test.go is criterion 8 of itd-2609201925079472 (spc-2609301921521360,
// "The rate-limit checkpoint"): a runner's rate-limit response ends the
// window early for the whole run, since every lane spends the same budget;
// every lane with work in flight is checkpointed to its own branch,
// next_eligible_at is written once, and the record names the lane the
// response came from.

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/runner"
)

// claudeImplementer routes the implementer alone to the claude runner.
const claudeImplementer = `{"roles":{"implementer":{"runner":"claude"}},"runner":{"claude":{}}}`

// rateLimitedPair is a run of two lanes side by side whose second
// implementer, started through the claude runner, meets a rate limit
// mid-work: lane-1's implementer, on the host, is out with a commit made and
// an edit left uncommitted. It returns the fixture, the driven step's result
// and lane-1's commit.
func rateLimitedPair(t *testing.T) (*parFixture, StepResult, string) {
	t.Helper()
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n", Options{SubAgents: strp("3")})
	f.stepUntil(t, "lane-1's implementer is out and lane-2 is at its implement stage", func(st State) bool {
		return len(st.Lanes) == 2 && len(st.Lanes[0].Awaits) == 1 && st.Lanes[1].Stage == StageImplement && len(st.Lanes[1].Awaits) == 0
	})
	l1 := f.lane(t, "lane-1")
	head1 := laneCommit(t, f.repo, l1, "one.txt")
	if err := os.WriteFile(filepath.Join(l1.Worktree, "wip.txt"), []byte("lane-1 at work\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	if err := os.Symlink(self, filepath.Join(bin, "claude")); err != nil {
		t.Fatal(err)
	}
	// The fake claude comes first on PATH, so no real harness is reached;
	// git and the forge stub stay where the fixture put them.
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv(loopFakeEnv, "ratelimit")
	t.Setenv(loopFakeLogEnv, "")
	res, err := Drive(context.Background(), f.repo.Root(), f.runID, f.stages, f.opts(),
		Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatalf("a rate-limit response is the step's answer, not its refusal: %v", err)
	}
	return f, res, head1
}

// entries are the run record's notes at stage, lane by lane.
func entries(st State, stage string) map[string][]string {
	out := map[string][]string{}
	for _, e := range st.Record {
		if e.Stage == stage {
			out[e.Lane] = append(out[e.Lane], e.Note)
		}
	}
	return out
}

func TestARateLimitMidLaneCheckpointsEveryLaneAndEndsTheWindow(t *testing.T) {
	f, res, head1 := rateLimitedPair(t)
	until := f.now.Add(BundledPauseMinutes * time.Minute)

	// The window ends early, for the whole run, with the response named.
	if res.NextEligibleAt == nil || !res.NextEligibleAt.Equal(until) {
		t.Fatalf("next_eligible_at = %v, want %v", res.NextEligibleAt, until)
	}
	if rl := res.RateLimit; rl == nil || rl.Lane != "lane-2" || rl.Role != RoleImplementer || rl.Runner != runner.Claude ||
		!strings.Contains(rl.Response, "rejected rate_limit_event") {
		t.Fatalf("the step names the rate limit: %+v", res.RateLimit)
	}
	st := f.state(t)
	if st.NextEligibleAt == nil || !st.NextEligibleAt.Equal(until) {
		t.Fatalf("the state's next_eligible_at = %v, want %v", st.NextEligibleAt, until)
	}
	limits := entries(st, StageRateLimit)
	if len(limits["lane-2"]) != 1 || !strings.Contains(limits["lane-2"][0], "claude") || !strings.Contains(limits["lane-2"][0], "rejected rate_limit_event") {
		t.Fatalf("the record names the response and the lane it came from: %+v", limits)
	}
	if p := entries(st, "pause")[""]; len(p) != 1 || !strings.Contains(p[0], until.Format(time.RFC3339)) {
		t.Fatalf("the record names the pause once: %+v", p)
	}

	// The lane the response came from is checkpointed to its branch: its
	// agent's uncommitted work and partial receipt are saved aside, its
	// worktree is back at its last commit, and its implementer is out no more.
	l2 := f.lane(t, "lane-2")
	head2 := strings.TrimSpace(f.repo.Git("-C", l2.Worktree, "rev-parse", "HEAD"))
	if len(l2.Awaits) != 0 {
		t.Fatalf("the rate-limited agent's slot is freed: %+v", l2.Awaits)
	}
	if _, err := os.Stat(filepath.Join(l2.Worktree, "halfway.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("lane-2's worktree is reset to its last commit (%v)", err)
	}
	if s := f.repo.Git("-C", l2.Worktree, "status", "--porcelain"); s != "" {
		t.Fatalf("lane-2's worktree is clean at its branch: %q", s)
	}
	dir := f.abs(RunRelDir + "/" + f.runID + "/lane-2")
	at, meta := asideOf(t, dir)
	if meta["head"] != head2 || !strings.Contains(meta["why"].(string), "rate limit") {
		t.Fatalf("aside.json names the head and the rate limit: %v", meta)
	}
	if patch, err := os.ReadFile(filepath.Join(at, AsidePatchName)); err != nil || !bytes.Contains(patch, []byte("half done")) {
		t.Fatalf("the uncommitted work is saved aside: %v", err)
	}
	cps := entries(st, StageCheckpoint)
	if len(cps["lane-2"]) != 1 || !strings.Contains(cps["lane-2"][0], shortSHA(head2)) || !strings.Contains(cps["lane-2"][0], at[len(f.repo.Root())+1:]) {
		t.Fatalf("the record names lane-2's checkpoint, its head and its aside: %+v", cps)
	}

	// Every other lane with work in flight is checkpointed at its branch's
	// head: its agent is not stopped and its worktree is not touched.
	l1 := f.lane(t, "lane-1")
	if len(l1.Awaits) != 1 || l1.Awaits[0].Role != RoleImplementer {
		t.Fatalf("lane-1's implementer is still out: %+v", l1.Awaits)
	}
	if _, err := os.Stat(filepath.Join(l1.Worktree, "wip.txt")); err != nil {
		t.Fatalf("lane-1's worktree is untouched: %v", err)
	}
	if len(cps["lane-1"]) != 1 || !strings.Contains(cps["lane-1"][0], shortSHA(head1)) || !strings.Contains(cps["lane-1"][0], RoleImplementer) {
		t.Fatalf("the record names lane-1's checkpoint at its head %s: %+v", shortSHA(head1), cps)
	}

	// Inside the pause a step refuses naming the time and changes nothing,
	// while an agent already out may still hand back its receipt.
	before := stateBytes(t, f.repo.Root(), f.runID)
	_, err := advance(f.repo.Root(), f.runID, f.stages, f.opts())
	if r, ok := AsRefusal(err); !ok || r.Stage != "pause" || !strings.Contains(r.Reason, until.Format(time.RFC3339)) {
		t.Fatalf("a step inside the pause refuses naming the time: %v", err)
	}
	if !bytes.Equal(before, stateBytes(t, f.repo.Root(), f.runID)) {
		t.Fatal("a refused step inside the pause changes no state")
	}
	if err := os.Remove(filepath.Join(l1.Worktree, "wip.txt")); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(10 * time.Minute)
	f.receipt(t, "lane-1", f.await(t, "lane-1", RoleImplementer), head1)

	// After the pause, a fresh implementer takes lane-2's brief again, from
	// the checkpoint.
	f.now = until
	f.stepUntil(t, "lane-2's implementer is handed out again", func(st State) bool {
		for _, l := range st.Lanes {
			if l.ID == "lane-2" && l.awaitsRole(RoleImplementer) {
				return true
			}
		}
		return false
	})
	if a := f.await(t, "lane-2", RoleImplementer); a.Brief != l2.Brief {
		t.Fatalf("the fresh implementer takes the same brief: %+v", a)
	}
}

// A second lane's rate-limit response inside the pause checkpoints that lane
// too, and next_eligible_at stays as it was written.
func TestASecondRateLimitInsideThePauseKeepsNextEligibleAt(t *testing.T) {
	f, res, head1 := rateLimitedPair(t)
	until := *res.NextEligibleAt
	l1 := f.lane(t, "lane-1")
	f.now = f.now.Add(20 * time.Minute)
	again, err := rateLimitWindow(f.repo.Root(), f.runID, "lane-1", l1.Awaits[0],
		&runner.Failure{Runner: runner.Claude, Reason: runner.ReasonRateLimited, Detail: "its event stream reports a rejected rate_limit_event and the run did not finish"}, f.opts())
	if err != nil {
		t.Fatal(err)
	}
	st := f.state(t)
	if again.NextEligibleAt == nil || !again.NextEligibleAt.Equal(until) || !st.NextEligibleAt.Equal(until) {
		t.Fatalf("next_eligible_at is written once: %v, state %v, want %v", again.NextEligibleAt, st.NextEligibleAt, until)
	}
	if p := entries(st, "pause")[""]; len(p) != 1 {
		t.Fatalf("one pause is recorded: %+v", p)
	}
	limits := entries(st, StageRateLimit)
	if len(limits["lane-1"]) != 1 || len(limits["lane-2"]) != 1 {
		t.Fatalf("the record names each response with its lane: %+v", limits)
	}
	l1 = f.lane(t, "lane-1")
	if len(l1.Awaits) != 0 {
		t.Fatalf("lane-1's rate-limited agent's slot is freed: %+v", l1.Awaits)
	}
	if _, err := os.Stat(filepath.Join(l1.Worktree, "wip.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("lane-1's uncommitted work is saved aside and its worktree reset (%v)", err)
	}
	if got := strings.TrimSpace(f.repo.Git("-C", l1.Worktree, "rev-parse", "HEAD")); got != head1 {
		t.Fatalf("lane-1 keeps its committed work: HEAD %s, want %s", got, head1)
	}
}

// A validator that meets the rate limit edits nothing, and may share the
// worktree with its round: only its partial return is saved aside, the
// worktree is left as it is, and the first step after the pause hands its
// review to a fresh validator.
func TestARateLimitedValidatorLeavesTheWorktreeAndIsHandedOutAgain(t *testing.T) {
	f := newParFixture(t, "1. One\n", Options{})
	f.stepUntil(t, "the implementer is out", func(st State) bool { return len(st.Lanes) == 1 && len(st.Lanes[0].Awaits) == 1 })
	f.implement(t, "lane-1", "one.txt")
	f.stepUntil(t, "the ruthless reviewer is out", func(st State) bool { return st.Lanes[0].awaitsRole(RoleRuthless) })
	a := f.await(t, "lane-1", RoleRuthless)
	if err := os.WriteFile(f.abs(a.Receipt), []byte("# Review\n\nhalf"), 0o600); err != nil {
		t.Fatal(err)
	}
	l := f.lane(t, "lane-1")
	if err := os.WriteFile(filepath.Join(l.Worktree, "note.txt"), []byte("left by no one in particular\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := rateLimitWindow(f.repo.Root(), f.runID, "lane-1", a,
		&runner.Failure{Runner: runner.Claude, Reason: runner.ReasonRateLimited, Detail: "its event stream reports a rejected rate_limit_event and the run did not finish"}, f.opts())
	if err != nil {
		t.Fatal(err)
	}
	if res.NextEligibleAt == nil || res.RateLimit == nil || res.RateLimit.Role != RoleRuthless {
		t.Fatalf("the validator's rate limit ends the window: %+v", res)
	}
	if f.lane(t, "lane-1").awaitsRole(RoleRuthless) {
		t.Fatal("the rate-limited validator's slot is freed")
	}
	if _, err := os.Stat(f.abs(a.Receipt)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the partial return leaves its path (%v)", err)
	}
	at, _ := asideOf(t, f.abs(RunRelDir+"/"+f.runID+"/lane-1"))
	if got, err := os.ReadFile(filepath.Join(at, filepath.Base(a.Receipt))); err != nil || string(got) != "# Review\n\nhalf" {
		t.Fatalf("the partial return is saved aside: %q %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(l.Worktree, "note.txt")); err != nil {
		t.Fatalf("a validator's rate limit leaves the worktree as it is: %v", err)
	}
	f.now = *res.NextEligibleAt
	f.stepUntil(t, "the ruthless reviewer is handed out again", func(st State) bool { return st.Lanes[0].awaitsRole(RoleRuthless) })
}
