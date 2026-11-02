package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/runner"
)

// A receipt's model reaches the landing's Assisted-by: trailer, so it is asked
// for where the implementer can still add it, and a lane whose receipt
// verified without one is given a way to land that edits no run state by hand
// (iss-2611012309408370).

// TestAReceiptThatReportsNoModelIsRefused: the receipt verifier names the
// missing model, so the implementer adds it before the lane moves on.
func TestAReceiptThatReportsNoModelIsRefused(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	repo, id := f.repo, f.runID
	stepTo(t, repo, id, f.stages, StageImplement)
	if res, err := advance(repo.Root(), id, f.stages, Options{}); err != nil || res.Awaiting == nil {
		t.Fatalf("the implement stage awaits an implementer: %+v %v", res, err)
	}
	l := currentLane(t, repo, id)
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), id, l.ID)
	sha := laneCommit(t, repo, l, "one.txt")
	rc := goodReceipt(t, id, l, dir, sha)
	path := writeReceipt(t, dir, map[string]any{
		"schema_version": rc.SchemaVersion, "run_id": rc.RunID, "lane": rc.Lane, "branch": rc.Branch,
		"commits": rc.Commits, "definition_of_done": rc.DefinitionOfDone, "report": rc.Report,
		"resolves": []map[string]string{{"issue": f.issue, "commit": sha, "note": "a blank name reads as absent",
			"impact": "fix", "grounds": "pursued: a blank name is accepted; shown wrong if the widget still refuses one"}},
	})
	_, err := Receipt(repo.Root(), id, path, f.stages, Options{})
	r := mustRefusal(t, err)
	if !strings.Contains(r.Reason, "the model your harness runs") {
		t.Fatalf("a receipt with no model is refused naming it: %+v", r)
	}
	if got := currentLane(t, repo, id).Receipts; len(got) != 0 {
		t.Fatalf("a refused receipt is not recorded: %+v", got)
	}
}

// forgetModel clears the model of every receipt the lane recorded, as a
// receipt verified before the model was required left it, and sets each
// receipt's route to route. With strip, the model is taken out of each
// receipt's file too.
func forgetModel(t *testing.T, f *landFixture, route *runner.RouteRecord, strip bool) {
	t.Helper()
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	i := st.current()
	for j := range st.Lanes[i].Receipts {
		r := &st.Lanes[i].Receipts[j]
		r.Model, r.Route = "", route
		if !strip {
			continue
		}
		p := filepath.Join(f.repo.Root(), filepath.FromSlash(r.Receipt))
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatal(err)
		}
		delete(m, "model")
		out, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, out, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.OpenRoot(f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := writeState(root, st); err != nil {
		t.Fatal(err)
	}
}

// landsWithModel steps the lane from land's first step to its records commit
// and checks the run log says where the model came from and the commit
// discloses it.
func landsWithModel(t *testing.T, f *landFixture, from, trailer string) {
	t.Helper()
	f.step(t) // prepare
	f.step(t) // the model is taken
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(st.Record, func(e Entry) bool {
		return strings.Contains(e.Note, "took the model of") && strings.Contains(e.Note, from)
	}) {
		t.Fatalf("the run log says the model was taken %s: %+v", from, st.Record)
	}
	f.step(t) // the records commit
	l := currentLane(t, f.repo, f.runID)
	msg := f.repo.Git("-C", l.Worktree, "log", "-1", "--format=%B", l.HeadSHA)
	if !slices.Contains(strings.Split(msg, "\n"), trailer) {
		t.Fatalf("the records commit discloses %s:\n%s", trailer, msg)
	}
}

// TestALandingReadsACorrectedReceiptsModelAgain: a receipt that verified with
// no model and was corrected in its file lands with the model the file now
// reports, and the run log says so.
func TestALandingReadsACorrectedReceiptsModelAgain(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.model = "claude-test-5"
	f.validated(t)
	forgetModel(t, f, nil, false)
	landsWithModel(t, f, "from its file", "Assisted-by: Claude:claude-test-5")
}

// TestALandingTakesTheModelOfTheRouteThatRanIt: a receipt a runner's
// implementer wrote with no model lands with the model the runner reported.
func TestALandingTakesTheModelOfTheRouteThatRanIt(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.validated(t)
	forgetModel(t, f, &runner.RouteRecord{Asked: runner.Claude, Ran: runner.Claude, Model: "claude-route-1"}, true)
	landsWithModel(t, f, "from the route that ran it", "Assisted-by: Claude:claude-route-1")
}
