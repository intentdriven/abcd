package loop

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/statusblock"
)

// TestStatusLanesReadsTheLaneEachRunWorksOn is the state-file half of the
// status block's criterion 1: every run in progress is one row naming its
// intent and its lane's state — the next step, the role it waits on, or
// "pending" between lanes — and a complete run is in no lane.
func TestStatusLanesReadsTheLaneEachRunWorksOn(t *testing.T) {
	repo := loopRepo(t, readyIntent("", settledQuestions),
		specWithSteps("1. The parser\n2. The loop\n"))
	if got, err := StatusLanes(repo.Root()); err != nil || len(got) != 0 {
		t.Fatalf("before any run: %v %v, want no lanes", got, err)
	}
	res, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := StatusLanes(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	want := []statusblock.Started{{Intent: "itd-10", Lane: statusblock.Lane{Run: res.RunID, Lane: "lane-1", Step: string(StepWorktree)}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("StatusLanes = %+v, want %+v", got, want)
	}

	rewrite := func(edit func(*State)) {
		t.Helper()
		st, err := ReadState(repo.Root(), res.RunID)
		if err != nil {
			t.Fatal(err)
		}
		edit(&st)
		root, err := os.OpenRoot(repo.Root())
		if err != nil {
			t.Fatal(err)
		}
		defer root.Close()
		if err := writeState(root, st); err != nil {
			t.Fatal(err)
		}
	}

	rewrite(func(st *State) {
		st.Lanes[0].Step = StepImplement
		st.Lanes[0].Awaiting = &Await{Role: "implementer", Brief: "b", Receipt: "r", Since: time.Unix(0, 0).UTC()}
	})
	if got, _ := StatusLanes(repo.Root()); len(got) != 1 || got[0].Lane.Step != "implement" || got[0].Lane.Awaiting != "implementer" {
		t.Errorf("a lane awaiting its implementer reads %+v", got)
	}

	rewrite(func(st *State) {
		st.Lanes[0].Step, st.Lanes[0].Awaiting = StepDone, nil
	})
	if got, _ := StatusLanes(repo.Root()); len(got) != 1 || got[0].Lane != (statusblock.Lane{Run: res.RunID, Step: "pending"}) {
		t.Errorf("a run between lanes reads %+v, want its step pending and no lane", got)
	}

	rewrite(func(st *State) { st.Pending = nil })
	if got, _ := StatusLanes(repo.Root()); len(got) != 0 {
		t.Errorf("a complete run reads %+v, want no lane", got)
	}
}

// TestTheStatusHeadIsTheIntentBuildNextPicks: the status block's head and its
// Next are read in the one pick order `abcd build next` takes, so over a record
// where every READY intent passes every pre-start check the head is the
// intent the pick chooses (the readiest, the oldest among equals) and Next is
// the candidates in the pick's order; once that intent's run is in progress the
// pick passes over it, and so does the head.
func TestTheStatusHeadIsTheIntentBuildNextPicks(t *testing.T) {
	repo := pickRepo(t, map[string][2]string{
		"20": {pickIntent("20", "", settledQuestions, gwt), pickSpec("20", "")},
		"21": {pickIntent("21", "", settledQuestions, gwt), pickSpec("21", fpSmall)},
		"22": {pickIntent("22", "", settledQuestions, gwt), pickSpec("22", fpSmall)},
	})
	for _, running := range []bool{false, true} {
		if running {
			if _, err := Start(repo.Root(), "itd-21", Options{}); err != nil {
				t.Fatal(err)
			}
		}
		set, err := candidates(repo.Root(), "")
		if err != nil {
			t.Fatal(err)
		}
		pick, ok := intent.Choose(set.Candidates)
		if !ok {
			t.Fatalf("running=%v: no candidate: %+v", running, set)
		}
		b, err := statusblock.Read(repo.Root(), StatusLanes)
		if err != nil {
			t.Fatal(err)
		}
		var head string
		for _, r := range b.Now {
			if r.NextUp {
				head = r.ID
			}
		}
		if head != pick.Chosen.ID {
			t.Errorf("running=%v: the head is %q, build next picks %q", running, head, pick.Chosen.ID)
		}
		if !running {
			var next, cands []string
			for _, r := range b.Next {
				next = append(next, r.ID)
			}
			for _, c := range set.Candidates {
				cands = append(cands, c.ID)
			}
			if !reflect.DeepEqual(next, cands) || pick.Chosen.ID != "itd-21" {
				t.Errorf("Next = %v, want the pick's order %v with itd-21 first", next, cands)
			}
		}
	}
}

// TestTheStatusHeadPassesOverWhatBuildNextExcludesFromTheRecord
// (iss-2609291803334904): the readiest READY intents each fail one of build
// next's record-only pre-start checks — an open question, an unanswered claim
// section, an unshipped blocker, a spec with no step left to build — and no
// peer holds anything, so the head is exactly the intent the pick chooses: the
// less ready one every check passes.
func TestTheStatusHeadPassesOverWhatBuildNextExcludesFromTheRecord(t *testing.T) {
	landed := "\n## Steps\n\n1. The parser\n   - landed: #1\n"
	repo := pickRepo(t, map[string][2]string{
		"20": {pickIntent("20", "", settledQuestions, gwt), pickSpec("20", "")},
		"21": {pickIntent("21", "", unsolved, gwt), pickSpec("21", fpSmall)},
		"22": {strings.Replace(pickIntent("22", "", settledQuestions, gwt), "We expect it to work because it is small; shown wrong if it is not.",
			intent.MechanismPrompt, 1), pickSpec("22", fpSmall)},
		"23": {pickIntent("23", "blocked_by: [itd-20]\n", settledQuestions, gwt), pickSpec("23", fpSmall)},
		"24": {pickIntent("24", "", settledQuestions, gwt), pickSpec("24", fpSmall) + landed},
	})
	set, err := candidates(repo.Root(), "")
	if err != nil {
		t.Fatal(err)
	}
	pick, ok := intent.Choose(set.Candidates)
	if !ok || pick.Chosen.ID != "itd-20" {
		t.Fatalf("precondition: build next picks itd-20, the one every check passes: %+v", set)
	}
	want := map[string]string{"itd-21": CheckOpenQuestions, "itd-22": CheckClaimSections, "itd-23": CheckBlocked, "itd-24": CheckSteps}
	for _, e := range set.Excluded {
		if want[e.ID] != e.Check {
			t.Fatalf("precondition: %s is excluded by %q, got %+v", e.ID, want[e.ID], e)
		}
	}
	b, err := statusblock.Read(repo.Root(), StatusLanes)
	if err != nil {
		t.Fatal(err)
	}
	var head string
	for _, r := range b.Now {
		if r.NextUp {
			head = r.ID
		}
	}
	if head != pick.Chosen.ID {
		t.Errorf("the head is %q, build next picks %q", head, pick.Chosen.ID)
	}
}
