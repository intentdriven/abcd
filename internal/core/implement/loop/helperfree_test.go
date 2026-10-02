package loop

// helperfree_test.go is ruling DR6d-1 (the product thinker, 2026-10-02):
// "preparing parallel steps' worktrees: ONLY WHEN A HELPER IS FREE — a step's
// worktree is made just before an agent takes it; at most the agent ceiling's
// worth on disk."

import (
	"os"
	"slices"
	"testing"
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
