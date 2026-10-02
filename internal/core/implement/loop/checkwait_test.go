package loop

// checkwait_test.go is ruling DR6d-2 (the product thinker, 2026-10-02): "a
// missing preflight receipt for one lane: CARRY ON, WAIT SHOWN — other lanes
// continue; the lane waits for its receipt like the merge wait (contend, not
// refuse), and the status shows 'waiting for its full check (since HH:MM)'."

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
)

// checkWaits counts the run record's entries naming a lane's wait for its full
// check.
func checkWaits(st State, laneID string) int {
	n := 0
	for _, e := range st.Record {
		if e.Lane == laneID && strings.Contains(e.Note, "waiting for its full check") {
			n++
		}
	}
	return n
}

// One lane reaches its push with no preflight receipt while another lane has
// an implementer out: the call is a wait for that lane alone (a contention, as
// the merge wait is), the time the wait began is written once and kept across
// calls, the other lane keeps moving with the wait named beside its move, and
// the status reads "waiting for its full check (since HH:MM)". The receipt
// minted, the lane pushes and the wait is over.
func TestALaneWithoutItsFullCheckWaitsWhileTheOthersCarryOn(t *testing.T) {
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n", Options{SubAgents: strp("3")})
	f.stepUntil(t, "both implementers are out", func(st State) bool {
		return len(st.Lanes) == 2 && len(st.Lanes[0].Awaits) == 1 && len(st.Lanes[1].Awaits) == 1
	})
	f.implement(t, "lane-1", "one.txt")
	f.roundPassed(t, "lane-1")

	// Lane 1 lands as far as its push; lane 2's implementer is still out, so
	// nothing else moves and the wait is the call's answer.
	f.now = time.Date(2026, 9, 30, 9, 30, 0, 0, time.UTC)
	var wait *Refusal
	for range 10 {
		_, err := advance(f.repo.Root(), f.runID, f.stages, f.opts())
		if err != nil {
			wait = mustRefusal(t, err)
			break
		}
	}
	if wait == nil || !wait.Contention || wait.Lane != "lane-1" || !strings.Contains(wait.Reason, "waiting for its full check (since 09:30)") {
		t.Fatalf("the lane without its receipt waits, a contention naming the time the wait began: %+v", wait)
	}
	if !strings.Contains(wait.Remedy, "make preflight") {
		t.Fatalf("the remedy names how the receipt is minted: %s", wait.Remedy)
	}
	st := f.state(t)
	l1 := f.lane(t, "lane-1")
	if l1.Landing == nil || l1.Landing.Pushed != "" || l1.Landing.CheckWaitSince == nil || !l1.Landing.CheckWaitSince.Equal(f.now) || checkWaits(st, "lane-1") != 1 {
		t.Fatalf("the time the wait began is written on the lane's landing, and the record names it once: %+v\n%s", l1.Landing, recordText(st))
	}
	if got := l1.CheckWait(); got != "waiting for its full check (since 09:30)" {
		t.Fatalf("the lane's status: %q", got)
	}

	// Seven minutes on, lane 2's implementer hands back: the call moves lane
	// 2, names lane 1's wait beside it, and keeps the time the wait began.
	f.now = f.now.Add(7 * time.Minute)
	f.implement(t, "lane-2", "two.txt")
	res := f.step(t)
	if res.Lane != "lane-2" || res.Awaiting == nil || res.Awaiting.Role != RoleRuthless {
		t.Fatalf("the other lane carries on: %+v", res)
	}
	if len(res.Blocked) != 1 || res.Blocked[0].Lane != "lane-1" || !strings.Contains(res.Next, "waiting for its full check (since 09:30)") {
		t.Fatalf("the wait is named beside the move, with the time it began: %+v", res)
	}
	st = f.state(t)
	if l1 := f.lane(t, "lane-1"); !l1.Landing.CheckWaitSince.Equal(time.Date(2026, 9, 30, 9, 30, 0, 0, time.UTC)) || checkWaits(st, "lane-1") != 1 {
		t.Fatalf("the time the wait began is kept across calls: %+v\n%s", l1.Landing, recordText(st))
	}
	lanes, err := StatusLanes(f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	var row string
	for _, s := range lanes {
		for _, l := range s.Lanes {
			if l.Lane == "lane-1" {
				row = l.Waiting
			}
		}
	}
	if row != "waiting for its full check (since 09:30)" {
		t.Fatalf("the status block's row for lane 1: %q in %+v", row, lanes)
	}

	// The receipt minted, lane 1 pushes and its wait is over.
	preflighted(t, l1, l1.HeadSHA)
	if r := f.step(t); r.Lane != "lane-1" {
		t.Fatalf("lane 1 pushes on its receipt: %+v", r)
	}
	if l1 := f.lane(t, "lane-1"); l1.Landing.Pushed == "" || l1.Landing.CheckWaitSince != nil || l1.CheckWait() != "" {
		t.Fatalf("the push ends the wait: %+v", l1.Landing)
	}
}

// refusing is the stages with name's body refusing, for laneID alone, with a
// plain refusal: a stage the binary performs that fails for a reason of its
// own.
func refusing(stages Stages, name Stage, laneID string) Stages {
	out := slices.Clone(stages)
	for k, d := range out {
		if d.Name != name {
			continue
		}
		run := d.Run
		out[k].Run = func(c Context, l *Lane) (Outcome, error) {
			if l.ID == laneID {
				return Outcome{}, refuse(string(name), "", l.ID, "the "+string(name)+" stage failed", "fix it and step again")
			}
			return run(c, l)
		}
	}
	return out
}

// The call that finds a lane waiting for its full check writes the time the
// wait began even when a later lane's stage refuses in that same call and the
// refusal is the call's answer: the time is when the wait began, so the next
// call keeps it rather than writing its own. Nothing else the refused call did
// is written — a lane it opened for a pending step stays unopened. Both a
// stage the binary performs on a lane already open and the worktree of a lane
// the call opens are covered.
func TestALaterRefusalInTheCallKeepsTheTimeTheCheckWaitBegan(t *testing.T) {
	f := newParFixture(t, "1. One\n2. Two\n   - needs: none\n", Options{SubAgents: strp("3")})
	f.stepUntil(t, "both implementers are out", func(st State) bool {
		return len(st.Lanes) == 2 && len(st.Lanes[0].Awaits) == 1 && len(st.Lanes[1].Awaits) == 1
	})
	f.implement(t, "lane-1", "one.txt")
	f.roundPassed(t, "lane-1")
	for range 10 {
		if _, err := advance(f.repo.Root(), f.runID, f.stages, f.opts()); err != nil {
			mustRefusal(t, err)
			break
		}
	}
	if f.lane(t, "lane-1").CheckWait() == "" {
		t.Fatalf("lane 1 waits for its full check: %+v", f.lane(t, "lane-1").Landing)
	}
	path := f.abs(StateRelPath(f.runID))
	waiting, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	for name, tc := range map[string]struct {
		edit    func(doc map[string]any, lanes []any)
		stage   Stage
		refused string
		lanes   int
	}{
		"a binary stage of an open lane": {func(doc map[string]any, lanes []any) {
			l2 := lanes[1].(map[string]any)
			l2["stage"] = string(StageBrief)
			delete(l2, "awaits")
		}, StageBrief, "lane-2", 2},
		"the worktree of a lane the call opens": {func(doc map[string]any, lanes []any) {
			doc["pending"] = append(doc["pending"].([]any), map[string]any{"number": 3, "title": "Three", "needs": []any{}})
		}, StageWorktree, "lane-3", 2},
	} {
		t.Run(name, func(t *testing.T) {
			// The state as it was before the call that finds the wait, with
			// the lane's case edited in.
			var doc map[string]any
			if err := json.Unmarshal(waiting, &doc); err != nil {
				t.Fatal(err)
			}
			lanes := doc["lanes"].([]any)
			delete(lanes[0].(map[string]any)["landing"].(map[string]any), "check_wait_since")
			tc.edit(doc, lanes)
			out, _ := json.MarshalIndent(doc, "", "  ")
			if err := os.WriteFile(path, out, 0o600); err != nil {
				t.Fatal(err)
			}
			pending := len(f.state(t).Pending)

			began := time.Date(2026, 9, 30, 10, 15, 0, 0, time.UTC)
			f.now = began
			_, err := advance(f.repo.Root(), f.runID, refusing(f.stages, tc.stage, tc.refused), f.opts())
			if r := mustRefusal(t, err); r.Contention || r.Lane != tc.refused {
				t.Fatalf("the later lane's refusal is the call's answer: %+v", r)
			}
			st := f.state(t)
			l1 := f.lane(t, "lane-1")
			if l1.Landing.CheckWaitSince == nil || !l1.Landing.CheckWaitSince.Equal(began) {
				t.Fatalf("the time the wait began is written with the refusal: %+v", l1.Landing)
			}
			if len(st.Lanes) != tc.lanes || len(st.Pending) != pending {
				t.Fatalf("nothing else the refused call did is written: %d lane(s), %d pending\n%s", len(st.Lanes), len(st.Pending), recordText(st))
			}

			// The next call, later, keeps the time the wait began.
			f.now = began.Add(5 * time.Minute)
			if _, err := advance(f.repo.Root(), f.runID, refusing(f.stages, tc.stage, tc.refused), f.opts()); err == nil {
				t.Fatal("the next call refuses as the last did")
			}
			if l1 := f.lane(t, "lane-1"); !l1.Landing.CheckWaitSince.Equal(began) {
				t.Fatalf("the since-time does not drift to the next call's: %+v", l1.Landing)
			}
		})
	}
}
