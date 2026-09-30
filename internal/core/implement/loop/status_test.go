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
	want := []statusblock.Started{{Intent: "itd-10", Lane: statusblock.Lane{Run: res.RunID, Lane: "lane-1", Stage: string(StageWorktree)}}}
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
		st.Lanes[0].Stage = StageImplement
		st.Lanes[0].Awaiting = &Await{Role: "implementer", Brief: "b", Receipt: "r", Since: time.Unix(0, 0).UTC()}
	})
	if got, _ := StatusLanes(repo.Root()); len(got) != 1 || got[0].Lane.Stage != "implement" || got[0].Lane.Awaiting != "implementer" {
		t.Errorf("a lane awaiting its implementer reads %+v", got)
	}

	rewrite(func(st *State) {
		st.Lanes[0].Stage, st.Lanes[0].Awaiting = StageDone, nil
	})
	if got, _ := StatusLanes(repo.Root()); len(got) != 1 || got[0].Lane != (statusblock.Lane{Run: res.RunID, Stage: "pending"}) {
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
		b, err := statusblock.Read(repo.Root(), StatusLanes, StatusPeers)
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
	b, err := statusblock.Read(repo.Root(), StatusLanes, StatusPeers)
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

// TestTheStatusHeadPassesOverAnIntentAPeerHolds (ruling CC1): the bare board
// pays build next's peers read, so "next up" always equals the pick. Another
// checkout holding the readiest intent in another bucket excludes it from the
// pick, and the head passes over it exactly as the pick does; a peer the
// listing names and cannot read fails closed on every record, for the head as
// for the pick, so neither names an intent.
func TestTheStatusHeadPassesOverAnIntentAPeerHolds(t *testing.T) {
	records := map[string][2]string{
		"20": {pickIntent("20", "", settledQuestions, gwt), pickSpec("20", "")},
		"21": {pickIntent("21", "", settledQuestions, gwt), pickSpec("21", fpSmall)},
	}
	head := func(b statusblock.Block) string {
		for _, r := range b.Now {
			if r.NextUp {
				return r.ID
			}
		}
		return ""
	}

	t.Run("a branch holds the readiest intent", func(t *testing.T) {
		repo := pickRepo(t, records)
		ir, _ := pickRel("21")
		repo.Git("checkout", "-q", "-b", "lane-alpha")
		repo.Remove(ir)
		repo.Write(".abcd/development/intents/shipped/itd-21-i21.md", records["21"][0])
		repo.Commit("deliver itd-21")
		repo.Git("checkout", "-q", "main")

		set, err := candidates(repo.Root(), "")
		if err != nil {
			t.Fatal(err)
		}
		pick, ok := intent.Choose(set.Candidates)
		if !ok || pick.Chosen.ID != "itd-20" {
			t.Fatalf("precondition: build next passes over itd-21, which lane-alpha holds, and picks itd-20: %+v", set)
		}
		b, err := statusblock.Read(repo.Root(), StatusLanes, StatusPeers)
		if err != nil {
			t.Fatal(err)
		}
		if got := head(b); got != pick.Chosen.ID {
			t.Errorf("the head is %q, build next picks %q: the board must pay the pick's peers read", got, pick.Chosen.ID)
		}
	})

	t.Run("a peer cannot be read", func(t *testing.T) {
		repo := pickRepo(t, records)
		repo.Git("checkout", "-q", "-b", "lane-beta")
		beta := "---\nid: itd-30\nslug: beta\n---\n# beta\n"
		repo.Write(".abcd/development/intents/drafts/itd-30-beta.md", beta)
		repo.Write(".abcd/development/intents/planned/itd-30-beta.md", beta)
		repo.Commit("split beta")
		repo.Git("checkout", "-q", "main")

		set, err := candidates(repo.Root(), "")
		if err != nil {
			t.Fatal(err)
		}
		if len(set.Candidates) != 0 {
			t.Fatalf("precondition: an unreadable peer leaves the pick no candidate: %+v", set)
		}
		b, err := statusblock.Read(repo.Root(), StatusLanes, StatusPeers)
		if err != nil {
			t.Fatalf("an unreadable peer fails closed on each record, never the board: %v", err)
		}
		if got := head(b); got != "" {
			t.Errorf("the head is %q; the pick has no candidate, so nothing is next up", got)
		}
		if len(b.Next) != 2 {
			t.Errorf("Next = %+v, want both READY intents still listed", b.Next)
		}
	})
}
