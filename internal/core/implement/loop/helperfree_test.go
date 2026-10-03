package loop

// helperfree_test.go is ruling DR6d-1 (the product thinker, 2026-10-02):
// "preparing parallel steps' worktrees: ONLY WHEN A HELPER IS FREE — a step's
// worktree is made just before an agent takes it; at most the agent ceiling's
// worth on disk."

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
	"time"
)

// worktreesOnDisk counts the lanes of the run whose worktree is on disk.
func worktreesOnDisk(t *testing.T, st State) int {
	t.Helper()
	n := 0
	for _, l := range st.Lanes {
		if l.Worktree == "" {
			continue
		}
		if _, err := os.Stat(l.Worktree); err == nil {
			n++
		}
	}
	return n
}

// cappedStep steps once and fails when more step worktrees are on disk than
// the run's ceiling.
func (f *parFixture) cappedStep(t *testing.T) StepResult {
	t.Helper()
	res := f.step(t)
	if st := f.state(t); worktreesOnDisk(t, st) > st.Ceiling() {
		t.Fatalf("%d step worktrees are on disk at a ceiling of %d: %+v", worktreesOnDisk(t, st), st.Ceiling(), st.Lanes)
	}
	return res
}

// With a ceiling of two and four steps ready at once, a step's worktree is
// made only when a helper is free to take it: lane 2's worktree is made once
// lane 1's implementer is out and a slot is left for lane 2's, and steps 3 and
// 4 wait for a slot with no worktree, never more than two on disk. When lane 1
// lands its worktree is removed, and only then is step 3's lane opened and its
// worktree made, just before its implementer takes the freed slot; step 4 still
// waits.
func TestAStepsWorktreeIsMadeOnlyWhenAHelperIsFree(t *testing.T) {
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n3. Three\n   - needs: none\n4. Four\n   - needs: none\n", Options{SubAgents: strp("2")})
	for range 12 {
		if r := f.cappedStep(t); r.CeilingReached {
			break
		}
	}
	st := f.state(t)
	if len(st.Lanes) != 2 || st.SlotsInUse() != 2 || len(st.Lanes[1].Awaits) != 1 || st.Lanes[1].Awaits[0].Role != RoleImplementer {
		t.Fatalf("two lanes open, each implementer in a slot: %+v", st.Lanes)
	}
	if worktreesOnDisk(t, st) != 2 {
		t.Fatalf("two step worktrees on disk: %+v", st.Lanes)
	}
	var queue []string
	for _, w := range st.Waiting {
		queue = append(queue, w.Lane+" "+w.Role)
	}
	if want := []string{"step 3 implementer", "step 4 implementer"}; !slices.Equal(queue, want) {
		t.Fatalf("steps 3 and 4 wait for a helper with no worktree: %v, want %v", queue, want)
	}

	// Lane 1's implementer hands back: the freed slot goes to its reviewer (an
	// open lane before a new one), and no new worktree is made.
	f.implement(t, "lane-1", "one.txt")
	for range 20 {
		l := f.lane(t, "lane-1")
		if l.Stage != StageValidate {
			break
		}
		if len(l.Awaits) > 0 {
			f.passAll(t, "lane-1")
			continue
		}
		f.cappedStep(t)
	}
	if st := f.state(t); len(st.Lanes) != 2 || worktreesOnDisk(t, st) != 2 {
		t.Fatalf("lane 1 at its landing still holds its worktree, so no lane opens: %+v", st.Lanes)
	}
	f.landed(t, "lane-1")
	st = f.state(t)
	if len(st.Lanes) != 3 || st.Lanes[2].SpecStep != 3 || st.Lanes[2].Worktree != "" || worktreesOnDisk(t, st) != 1 {
		t.Fatalf("lane 1's landing frees its worktree, and step 3's lane opens with no worktree yet: %+v", st.Lanes)
	}
	if r := f.cappedStep(t); r.PerformedStage != StageWorktree || r.Lane != "lane-3" {
		t.Fatalf("step 3's worktree is made next: %+v", r)
	}
	f.cappedStep(t)
	if r := f.cappedStep(t); r.Lane != "lane-3" || r.Awaiting == nil || r.Awaiting.Role != RoleImplementer {
		t.Fatalf("step 3's implementer takes the freed slot: %+v", r)
	}
	if r := f.cappedStep(t); !r.CeilingReached {
		t.Fatalf("the run is full again: %+v", r)
	}
	if st := f.state(t); len(st.Lanes) != 3 || len(st.Pending) != 1 || st.Pending[0].Number != 4 {
		t.Fatalf("step 4 still waits for a helper: %+v", st.Pending)
	}
}

// waitingQueue is the run's waiting list as "<item> <role>" lines.
func waitingQueue(st State) []string {
	var queue []string
	for _, w := range st.Waiting {
		queue = append(queue, w.Lane+" "+w.Role)
	}
	return queue
}

// A step held back by the worktree cap alone, with a slot free, waits under
// `waiting` as `step <n>` as one held by the slots does: lane 1 waits at its
// landing for its full check and lane 2's implementer is out, so one of the two
// slots is free but both step worktrees are on disk. The call that finds it
// lists the step, keeping the time it was first held, a second call writes
// nothing, and the lane that opens once lane 1 lands is served from the list.
func TestAStepHeldByTheWorktreeCapAloneWaitsUnderItsStep(t *testing.T) {
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n3. Three\n   - needs: none\n", Options{SubAgents: strp("2")})
	for range 12 {
		if r := f.cappedStep(t); r.CeilingReached {
			break
		}
	}
	f.implement(t, "lane-1", "one.txt")
	f.roundPassed(t, "lane-1")
	for range 10 {
		if _, err := advance(f.repo.Root(), f.runID, f.stages, f.opts()); err != nil {
			mustRefusal(t, err)
			break
		}
	}
	if f.lane(t, "lane-1").CheckWait() == "" {
		t.Fatalf("lane 1 waits at its landing for its full check: %+v", f.lane(t, "lane-1").Landing)
	}

	// The list as the slots left it is cleared, so what the call writes is the
	// cap's hold alone.
	path := f.abs(StateRelPath(f.runID))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	delete(doc, "waiting")
	out, _ := json.MarshalIndent(doc, "", "  ")
	if err := os.WriteFile(path, out, 0o600); err != nil {
		t.Fatal(err)
	}
	st := f.state(t)
	if st.SlotsInUse() != 1 || worktreesOnDisk(t, st) != 2 || len(st.Lanes) != 2 || len(st.Waiting) != 0 {
		t.Fatalf("one slot is free and both step worktrees are on disk: %d slot(s), %+v", st.SlotsInUse(), st.Lanes)
	}

	held := f.now.Add(10 * time.Minute)
	f.now = held
	res, err := advance(f.repo.Root(), f.runID, f.stages, f.opts())
	if err != nil {
		t.Fatalf("the call holding the step answers with the run, not the wait: %v", err)
	}
	if res.CeilingReached || len(res.Blocked) != 1 || res.Blocked[0].Lane != "lane-1" {
		t.Fatalf("the ceiling is not reached, and lane 1's wait is named: %+v", res)
	}
	st = f.state(t)
	if got, want := waitingQueue(st), []string{"step 3 implementer"}; !slices.Equal(got, want) || !st.Waiting[0].Since.Equal(held) {
		t.Fatalf("step 3 waits for a helper under its step: %v (%+v), want %v since %s", got, st.Waiting, want, held)
	}
	if len(st.Lanes) != 2 || worktreesOnDisk(t, st) != 2 {
		t.Fatalf("no lane opens for it: %+v", st.Lanes)
	}

	f.now = held.Add(5 * time.Minute)
	if _, err := advance(f.repo.Root(), f.runID, f.stages, f.opts()); err == nil || !mustRefusal(t, err).Contention {
		t.Fatalf("a second call writes nothing and answers with lane 1's wait: %v", err)
	}
	if st := f.state(t); !st.Waiting[0].Since.Equal(held) {
		t.Fatalf("the time the step was first held is kept: %+v", st.Waiting)
	}

	f.landed(t, "lane-1")
	st = f.state(t)
	if len(st.Lanes) != 3 || st.Lanes[2].SpecStep != 3 || len(st.Waiting) != 0 {
		t.Fatalf("lane 1's landing opens step 3's lane and serves its waiting item: %+v %+v", st.Lanes, st.Waiting)
	}
}
