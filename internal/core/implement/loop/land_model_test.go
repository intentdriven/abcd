package loop

import (
	"context"
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

// setModel sets the model of every receipt the lane recorded to model, as a
// receipt verified before the model was required left it, and each receipt's
// route to route. With inFile, each receipt's file reports model too (none
// when it is empty).
func setModel(t *testing.T, f *landFixture, model string, route *runner.RouteRecord, inFile bool) {
	t.Helper()
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	i := st.current()
	for j := range st.Lanes[i].Receipts {
		r := &st.Lanes[i].Receipts[j]
		r.Model, r.Route = model, route
		if !inFile {
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
		if model != "" {
			m["model"] = model
		}
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
	setModel(t, f, "", nil, false)
	landsWithModel(t, f, "from its file", "Assisted-by: Claude:claude-test-5")
}

// TestALandingTakesTheModelOfTheRouteThatRanIt: a receipt a runner's
// implementer wrote with no model lands with the model the runner reported.
func TestALandingTakesTheModelOfTheRouteThatRanIt(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.validated(t)
	setModel(t, f, "", &runner.RouteRecord{Asked: runner.Claude, Ran: runner.Claude, Model: "claude-route-1"}, true)
	landsWithModel(t, f, "from the route that ran it", "Assisted-by: Claude:claude-route-1")
}

// TestALandingReadsAnUnnameableModelAgain: a receipt that verified with a
// model the trailer cannot name, and was corrected in its file, lands with the
// corrected model, as the refusal's remedy says it will.
func TestALandingReadsAnUnnameableModelAgain(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.model = "claude-test-5"
	f.validated(t)
	setModel(t, f, "unprefixed-model", nil, false)
	landsWithModel(t, f, "from its file", "Assisted-by: Claude:claude-test-5")
}

// TestAFixBriefAsksForTheModel: a fix round's implementer writes a receipt
// the same verifier reads, which refuses one with no model, so its brief asks
// for the model as required rather than optional (iss-2611022247103872).
func TestAFixBriefAsksForTheModel(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	id, stages := start.RunID, DefaultStages()
	implemented(t, repo, id, stages, "one.txt")
	passRound(t, repo, id, stages, RoleRuthless, RoleSecurity)
	handBack(t, repo, id, stages, RoleAuditor, "INCONCLUSIVE")
	res, err := advance(repo.Root(), id, stages, Options{})
	if err != nil || res.Awaiting == nil || res.Awaiting.Role != RoleImplementer {
		t.Fatalf("the failed round hands the lane to a fresh implementer: %+v %v", res, err)
	}
	brief, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(res.Awaiting.Brief)))
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(strings.Fields(string(brief)), " ")
	if strings.Contains(text, "optional `model`") || !strings.Contains(text, "`model`, the model id your harness reports (required") {
		t.Fatalf("the fix brief asks for the model as required:\n%s", brief)
	}
}

// An implementer a runner launched whose receipt names no model is not paused
// for a person to edit the file (iss-2611022247103872): the runner reports the
// model it ran, so the receipt verifies under that model and the record says
// where it came from. A model the receipt does name is never replaced, and a
// runner's report that differs from it is named beside it in the record.

// driveReceiptOnClaude takes a landing fixture's lane to its implement stage,
// commits its work, and drives the stage to the claude runner, whose agent
// writes a lane receipt naming model (none when it is empty). It returns the
// fixture, the step's result and its error.
func driveReceiptOnClaude(t *testing.T, model string) (*landFixture, StepResult, error) {
	t.Helper()
	f := newLandFixture(t, queueRuleset("MERGE"))
	repo, id := f.repo, f.runID
	l := stepTo(t, repo, id, f.stages, StageImplement)
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), id, l.ID)
	sha := laneCommit(t, repo, l, "one.txt")
	rc := goodReceipt(t, id, l, dir, sha)
	body := map[string]any{
		"schema_version": rc.SchemaVersion, "run_id": rc.RunID, "lane": rc.Lane, "branch": rc.Branch,
		"commits": rc.Commits, "definition_of_done": rc.DefinitionOfDone, "report": rc.Report,
		"resolves": []map[string]string{{"issue": f.issue, "commit": sha, "note": "a blank name reads as absent",
			"impact": "fix", "grounds": "pursued: a blank name is accepted; shown wrong if the widget still refuses one"}},
	}
	if model != "" {
		body["model"] = model
	}
	t.Setenv(loopFakeReceiptEnv, writeReceipt(t, t.TempDir(), body))
	newDriveEnv(t, "receipt", "claude")
	res, err := Drive(context.Background(), repo.Root(), id, f.stages, Options{},
		Runners{Config: runnerConfig(t, claudeImplementer, ""), Transcripts: &memTranscripts{}})
	return f, res, err
}

// TestARunnersImplementerReceiptWithNoModelTakesTheModelTheRunnerRan: the
// receipt verifies, the lane records the model the claude runner reported,
// the landing's trailer can name it, the record says it came from the runner,
// and the receipt file stays the implementer's own words.
func TestARunnersImplementerReceiptWithNoModelTakesTheModelTheRunnerRan(t *testing.T) {
	f, res, err := driveReceiptOnClaude(t, "")
	if err != nil {
		t.Fatalf("a runner's receipt with no model is not refused: %v", err)
	}
	if res.PerformedStage != StageImplement || res.Stage != StageValidate {
		t.Fatalf("the receipt verifies and the lane moves on: %+v", res)
	}
	l := currentLane(t, f.repo, f.runID)
	if len(l.Receipts) != 1 || l.Receipts[0].Model != loopFakeReceiptModel {
		t.Fatalf("the receipt is recorded with the model the runner ran: %+v", l.Receipts)
	}
	if r := l.Receipts[0].Route; r == nil || r.Ran != runner.Claude || r.Model != loopFakeReceiptModel {
		t.Fatalf("the receipt names the route that ran it: %+v", r)
	}
	if got, gap := assistedByTrailers(l.Receipts); gap != "" || !slices.Equal(got, []string{"Assisted-by: Claude:" + loopFakeReceiptModel}) {
		t.Fatalf("the landing's trailer names the model the runner ran: %v %q", got, gap)
	}
	if !recordSays(t, f.repo.Root(), f.runID, StageRunner, "names no model", loopFakeReceiptModel, "claude runner reported") {
		st, _ := ReadState(f.repo.Root(), f.runID)
		t.Fatalf("the record says the model came from the runner: %+v", st.Record)
	}
	raw, err := os.ReadFile(filepath.Join(f.repo.Root(), filepath.FromSlash(l.Receipts[0].Receipt)))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"model"`) {
		t.Fatalf("the receipt file is not rewritten: %s", raw)
	}
}

// TestARunnersImplementerReceiptKeepsTheModelItNames: a receipt naming a model
// keeps it, and a runner's report that differs is named beside it in the
// record, never in its place.
func TestARunnersImplementerReceiptKeepsTheModelItNames(t *testing.T) {
	f, res, err := driveReceiptOnClaude(t, "claude-sonnet-5-1")
	if err != nil {
		t.Fatal(err)
	}
	if res.PerformedStage != StageImplement {
		t.Fatalf("the receipt verifies: %+v", res)
	}
	l := currentLane(t, f.repo, f.runID)
	if len(l.Receipts) != 1 || l.Receipts[0].Model != "claude-sonnet-5-1" {
		t.Fatalf("the receipt's own model is kept: %+v", l.Receipts)
	}
	if !recordSays(t, f.repo.Root(), f.runID, StageRunner, "claude-sonnet-5-1", loopFakeReceiptModel, "the receipt's is kept") {
		st, _ := ReadState(f.repo.Root(), f.runID)
		t.Fatalf("the record names the disagreement: %+v", st.Record)
	}
}
