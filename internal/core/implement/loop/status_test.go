package loop

import (
	"os"
	"reflect"
	"testing"
	"time"

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
