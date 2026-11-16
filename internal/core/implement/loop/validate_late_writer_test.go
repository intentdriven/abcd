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

// collectedImplementer makes the lane's first receipt one a step collected
// where it stood, as receipt_standing.go records it: its route names the
// runner asked and none as having run it, and the record's line for it ends
// with the digest of the bytes taken. It returns the receipt's path.
func collectedImplementer(t *testing.T, root, id string) string {
	t.Helper()
	st, err := ReadState(root, id)
	if err != nil {
		t.Fatal(err)
	}
	rel := st.Lanes[0].Receipts[0].Receipt
	path := absIn(root, rel)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	editState(t, root, id, func(doc map[string]any) {
		lane := doc["lanes"].([]any)[0].(map[string]any)
		lane["receipts"].([]any)[0].(map[string]any)["route"] = map[string]any{"asked": runner.Claude, "ran": "", "model": ""}
		for _, e := range doc["record"].([]any) {
			e := e.(map[string]any)
			if note, _ := e["note"].(string); e["stage"] == "receipt" && strings.Contains(note, " verified at "+rel) {
				e["note"] = note + collectedNote(answerDigest(data))
			}
		}
	})
	return path
}

// TestAModelLearnedLateIsAWriterFromThen: a runner's report reaching a
// receipt a step collected before it, after the round's reviewers were handed
// out, is settled in the same write: a report naming the writer's model in
// other bytes leaves a file that reads and moves no reviewer; one naming a
// reviewer's own model withdraws that reviewer, so its verdict does not count,
// and the next step hands it out again on a class other than every writer's.
func TestAModelLearnedLateIsAWriterFromThen(t *testing.T) {
	const writer = "claude-fable-5"
	report := func(model string) *runner.RouteRecord {
		return &runner.RouteRecord{Asked: runner.Claude, Ran: runner.Claude, Model: model}
	}
	t.Run("the same model in other bytes", func(t *testing.T) {
		repo, id := otherModelRun(t, "", writer)
		path := collectedImplementer(t, repo.Root(), id)
		stages := DefaultStages()
		first := stepAwait(t, repo, id, stages, Options{})
		second := stepAwait(t, repo, id, stages, Options{})
		returnAt(t, repo, id, stages, first, reviewerReturn("SHIP"))
		if _, err := receipt(repo.Root(), id, path, stages, Options{}, report("anthropic/claude-fable-5")); err != nil {
			t.Fatalf("the runner's report is answered: %v", err)
		}
		st, err := ReadState(repo.Root(), id)
		if err != nil {
			t.Fatalf("the file the report leaves reads: %v", err)
		}
		v := validatorAt(t, repo.Root(), id, first.Receipt)
		if v.Verdict != "SHIP" || v.Class != first.Class || !slices.Equal(v.WriterModels, []string{"anthropic/claude-fable-5", writer}) {
			t.Fatalf("the verdict stands, the class is kept, and the writers name both spellings: %+v", v)
		}
		if w := validatorAt(t, repo.Root(), id, second.Receipt); w.Class != second.Class || !st.Lanes[0].awaited(second.Receipt) {
			t.Fatalf("the reviewer still out stays out on its class: %+v", w)
		}
	})
	t.Run("a reviewer's own model", func(t *testing.T) {
		repo, id := otherModelRun(t, "", writer)
		path := collectedImplementer(t, repo.Root(), id)
		stages := DefaultStages()
		first := stepAwait(t, repo, id, stages, Options{})
		second := stepAwait(t, repo, id, stages, Options{})
		if first.Class != ClassTop {
			t.Fatalf("premise: the research reviewer moves to top: %+v", first)
		}
		returnAt(t, repo, id, stages, first, reviewerReturn("SHIP"))
		if _, err := receipt(repo.Root(), id, path, stages, Options{}, report(BundledClasses[ClassTop])); err != nil {
			t.Fatalf("the runner's report is answered: %v", err)
		}
		st, err := ReadState(repo.Root(), id)
		if err != nil {
			t.Fatalf("the file the report leaves reads: %v", err)
		}
		v := validatorAt(t, repo.Root(), id, first.Receipt)
		if v.Verdict != "" || len(v.WriterModels) != 0 || v.ReviewerModel != "" || st.Lanes[0].awaited(first.Receipt) {
			t.Fatalf("the reviewer on the reported model is withdrawn and its verdict does not count: %+v", v)
		}
		if w := validatorAt(t, repo.Root(), id, second.Receipt); !slices.Equal(w.WriterModels, []string{BundledClasses[ClassTop], writer}) {
			t.Fatalf("the reviewer whose model still differs keeps its hand-out under the writers recomputed: %+v", w)
		}
		if !recordSays(t, repo.Root(), id, string(StageValidate), "was withdrawn", "does not count") {
			t.Fatalf("the record says the reviewer was withdrawn: %+v", st.Record)
		}
		again := stepAwait(t, repo, id, stages, Options{})
		if again.Receipt != first.Receipt || again.Class == ClassTop || again.Class == ClassResearch {
			t.Fatalf("the next step hands the withdrawn reviewer out again on a class other than every writer's: %+v", again)
		}
	})
}

// TestTheLoopNeverSavesARunItsReadRefuses: a save that would leave stored
// models the read refuses is refused naming where, and the file already there
// stands.
func TestTheLoopNeverSavesARunItsReadRefuses(t *testing.T) {
	repo, id := otherModelRun(t, "", "claude-fable-5")
	stepAwait(t, repo, id, DefaultStages(), Options{})
	before := stateBytes(t, repo.Root(), id)
	err := mutate(repo.Root(), id, func(_ *os.Root, st *State) (bool, error) {
		v := &st.Lanes[0].Validation[0].Validators[0]
		v.WriterModels = []string{"claude-haiku-4-5"}
		return true, nil
	})
	r := mustRefusal(t, err)
	if r.Stage != "state" || !strings.Contains(r.Reason, "was not saved") || !strings.Contains(r.Reason, "round 1 validator 1") {
		t.Fatalf("the save is refused naming where: %+v", r)
	}
	if after := stateBytes(t, repo.Root(), id); string(after) != string(before) {
		t.Fatal("the file already there stands")
	}
}

// TestEveryLaunchedWriterIsAWriter: a model a runner was started on for the
// lane's implementer is a writer of it whether or not a receipt came back from
// it, so a reviewer whose class holds it moves on; the start records it.
func TestEveryLaunchedWriterIsAWriter(t *testing.T) {
	t.Run("a launch with no receipt of its own", func(t *testing.T) {
		repo, id := otherModelRun(t, "", "claude-haiku-4-5")
		// A gone agent's launch on the default class's model, before the
		// restarted implementer's receipt.
		editState(t, repo.Root(), id, func(doc map[string]any) {
			doc["lanes"].([]any)[0].(map[string]any)["writers_launched"] = []any{map[string]any{"model": BundledClasses[ClassDefault], "after": 0}}
		})
		stages := DefaultStages()
		first := stepAwait(t, repo, id, stages, Options{})
		second := stepAwait(t, repo, id, stages, Options{})
		v := validatorAt(t, repo.Root(), id, second.Receipt)
		if !slices.Equal(v.WriterModels, []string{BundledClasses[ClassDefault], "claude-haiku-4-5"}) {
			t.Fatalf("the launched model is a writer, before the receipt's: %+v", v.WriterModels)
		}
		if first.Class == ClassDefault || second.Class == ClassDefault || second.Class == ClassSmall {
			t.Fatalf("no reviewer runs on a launched or a named writer's model: %+v %+v", first, second)
		}
	})
	t.Run("the start records the launched model", func(t *testing.T) {
		f, _, err := driveReceiptOnClaudeAt(t, `{"roles":{"implementer":{"runner":"claude"}},`+
			`"oracle":{"api":{"local":{"base_url":"http://localhost:11434/v1","models":["qwen3-coder"]}}},`+
			`"runner":{"claude":{"model":"local/qwen3-coder"}}}`)
		if err != nil {
			t.Fatal(err)
		}
		l := currentLane(t, f.repo, f.runID)
		if !slices.Equal(l.Launched, []LaunchedWriter{{Model: "local/qwen3-coder", After: 0}}) {
			t.Fatalf("the lane records the model its implementer's runner was started on: %+v", l.Launched)
		}
		if w, why := laneWriters(l, len(l.Receipts)); why != "" || !slices.Contains(w, "local/qwen3-coder") {
			t.Fatalf("the launched model is among the writers: %v %q", w, why)
		}
	})
}

// driveReceiptOnClaudeAt is driveReceiptOnClaude under the machine
// configuration given for the runner, the receipt naming the fake's model.
func driveReceiptOnClaudeAt(t *testing.T, config string) (*landFixture, StepResult, error) {
	t.Helper()
	f := newLandFixture(t, queueRuleset("MERGE"))
	repo, id := f.repo, f.runID
	l := stepTo(t, repo, id, f.stages, StageImplement)
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), id, l.ID)
	sha := laneCommit(t, repo, l, "one.txt")
	rc := goodReceipt(t, id, l, dir, sha)
	body := map[string]any{
		"schema_version": rc.SchemaVersion, "run_id": rc.RunID, "lane": rc.Lane, "branch": rc.Branch,
		"commits": rc.Commits, "definition_of_done": rc.DefinitionOfDone, "report": rc.Report, "model": loopFakeReceiptModel,
		"resolves": []map[string]string{{"issue": f.issue, "commit": sha, "note": "a blank name reads as absent",
			"impact": "fix", "grounds": "pursued: a blank name is accepted; shown wrong if the widget still refuses one"}},
	}
	t.Setenv(loopFakeReceiptEnv, writeReceipt(t, t.TempDir(), body))
	newDriveEnv(t, "receipt", "claude")
	res, err := Drive(context.Background(), repo.Root(), id, f.stages, Options{},
		Runners{Config: runnerConfig(t, config, ""), Transcripts: &memTranscripts{}})
	return f, res, err
}

// TestALandingsModelIsHeldToTheLanesReviewers: the landing's re-read of a
// receipt file is held to the writers the reviewers were compared with: a
// receipt naming a model of 300 bytes beside a runner's nameable report lands
// on the report, keeping no name the read refuses, with the writers left in
// place; and a file naming a reviewer's own model hands the lane back as
// stuck, its review no longer counting.
func TestALandingsModelIsHeldToTheLanesReviewers(t *testing.T) {
	t.Run("a 300-byte name beside a runner's report", func(t *testing.T) {
		f := newLandFixture(t, queueRuleset("MERGE"))
		f.validatedOnce(t)
		long := strings.Repeat("m", 300)
		editState(t, f.repo.Root(), f.runID, func(doc map[string]any) {
			for _, r := range doc["lanes"].([]any)[0].(map[string]any)["receipts"].([]any) {
				r := r.(map[string]any)
				r["model"] = long
				r["route"] = map[string]any{"asked": runner.Claude, "ran": runner.Claude, "model": f.model}
			}
		})
		if _, err := ReadState(f.repo.Root(), f.runID); err != nil {
			t.Fatalf("premise: the edited run reads, its writers in place: %v", err)
		}
		landsWithModel(t, f, "from the route that ran it", "Assisted-by: Claude:"+f.model)
		l := currentLane(t, f.repo, f.runID)
		if r := l.Receipts[0]; r.Named != "" || r.Model != f.model {
			t.Fatalf("the overlong name is kept as nothing: %+v", r)
		}
	})
	t.Run("a file naming a reviewer's own model", func(t *testing.T) {
		f := newLandFixture(t, queueRuleset("MERGE"))
		f.validatedOnce(t)
		st, err := ReadState(f.repo.Root(), f.runID)
		if err != nil {
			t.Fatal(err)
		}
		var reviewer string
		for _, v := range st.Lanes[0].Validation[len(st.Lanes[0].Validation)-1].Validators {
			if v.Class != "" && strings.HasPrefix(v.ReviewerModel, "anthropic/claude-") {
				reviewer = strings.TrimPrefix(v.ReviewerModel, "anthropic/")
			}
		}
		if reviewer == "" {
			t.Fatalf("premise: a reviewer ran on a bundled class: %+v", st.Lanes[0].Validation)
		}
		// The receipt verified naming the writer in a form the trailer cannot
		// take, and its file now names the reviewer's model.
		editState(t, f.repo.Root(), f.runID, func(doc map[string]any) {
			lane := doc["lanes"].([]any)[0].(map[string]any)
			for _, r := range lane["receipts"].([]any) {
				r.(map[string]any)["model"] = "mythos-5"
			}
			for _, round := range lane["validation"].([]any) {
				for _, v := range round.(map[string]any)["validators"].([]any) {
					if v := v.(map[string]any); v["writer_models"] != nil {
						v["writer_models"] = []any{"mythos-5"}
					}
				}
			}
		})
		for _, r := range st.Lanes[0].Receipts {
			p := absIn(f.repo.Root(), r.Receipt)
			raw, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			var m map[string]any
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Fatal(err)
			}
			m["model"] = reviewer
			out, _ := json.Marshal(m)
			if err := os.WriteFile(p, out, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := ReadState(f.repo.Root(), f.runID); err != nil {
			t.Fatalf("premise: the edited run reads: %v", err)
		}
		f.step(t) // prepare
		res, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
		if err != nil {
			t.Fatal(err)
		}
		if hb := res.HandBack; hb == nil || hb.Kind != HandBackStuck || !strings.Contains(hb.Reason, "learned after") {
			t.Fatalf("the lane is handed back as stuck: %+v", res)
		}
		if _, err := ReadState(f.repo.Root(), f.runID); err != nil {
			t.Fatalf("the hand-back leaves a file that reads: %v", err)
		}
	})
}

// TestARunReadAsStartedBeforeModelsSaysSo: a version-27 file edited to read as
// a run started before reviewers' models were recorded (the mark set, the
// class map and the unlaunchable classes removed) reads, and the pull
// request's body says in words that the reviewer's models were not recorded;
// a reviewer handed out after the read is compared and recorded as any other.
func TestARunReadAsStartedBeforeModelsSaysSo(t *testing.T) {
	const writer = "claude-haiku-4-5"
	repo, id := otherModelRun(t, "", writer)
	first := stepAwait(t, repo, id, DefaultStages(), Options{})
	editState(t, repo.Root(), id, func(doc map[string]any) {
		rs := doc["review"].(map[string]any)
		delete(rs, "classes")
		delete(rs, "unlaunchable")
		v := firstValidator(doc, 0)
		delete(v, "writer_models")
		delete(v, "reviewer_model")
		v["models_unrecorded"] = true
		v["class"], v["model"] = ClassResearch, BundledClasses[ClassResearch]
		aw := doc["lanes"].([]any)[0].(map[string]any)["awaits"].([]any)[0].(map[string]any)
		aw["class"], aw["model"] = ClassResearch, BundledClasses[ClassResearch]
	})
	st, err := ReadState(repo.Root(), id)
	if err != nil {
		t.Fatalf("the edited file reads: %v", err)
	}
	body := prBody(st, st.Lanes[0], nil)
	const said = "- " + RoleRuthless + ", reviewer 0: the reviewer's models were not recorded: it was handed out before the loop recorded them."
	if !slices.Contains(strings.Split(body, "\n"), said) {
		t.Fatalf("the body says the models were not recorded, naming none:\n%s", body)
	}
	returnAt(t, repo, id, DefaultStages(), first, reviewerReturn("SHIP"))
	st, err = ReadState(repo.Root(), id)
	if err != nil {
		t.Fatal(err)
	}
	for _, rv := range recordOf(st).Lanes[0].Verdicts {
		if rv.Return == first.Receipt && rv.ModelsPhrase() != "the reviewer's models were not recorded: it was handed out before the loop recorded them" {
			t.Fatalf("the record says so too: %q", rv.ModelsPhrase())
		}
	}
	second := stepAwait(t, repo, id, DefaultStages(), Options{})
	if v := validatorAt(t, repo.Root(), id, second.Receipt); !slices.Equal(v.WriterModels, []string{writer}) || v.ReviewerModel == "" {
		t.Fatalf("a reviewer handed out after the read is compared and recorded: %+v", v)
	}
}

// TestARoundAfterASyncByAnotherModel: a conflict resolver on another model
// joins the writers, so the fresh round over its merge is judged by neither
// the first writer's model nor the resolver's, while the first round keeps
// the writers it was handed out under, and the file reads.
func TestARoundAfterASyncByAnotherModel(t *testing.T) {
	f := syncFixture(t)
	judged := f.passToLandOnce(t, "fix.txt", "gen.txt").HeadSHA
	f.moveMain(t, "generated again on main\n", "gen.txt")
	f.step(t)
	res := f.step(t)
	l := currentLane(t, f.repo, f.runID)
	s := l.pendingSync()
	if s == nil || res.Awaiting == nil || res.Awaiting.Receipt != s.Receipt || l.HeadSHA != judged {
		t.Fatalf("premise: the sync waits on a resolver: %+v %+v", l, res)
	}
	_ = gitErr(f.repo, "-C", l.Worktree, "merge", "--no-edit", s.Merged)
	if err := os.WriteFile(filepath.Join(l.Worktree, "gen.txt"), []byte("regenerated over both sides\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.repo.Git("-C", l.Worktree, "add", "--", "gen.txt")
	f.repo.Git("-C", l.Worktree, "commit", "-q", "--no-edit")
	merge := strings.TrimSpace(f.repo.Git("-C", l.Worktree, "rev-parse", "HEAD"))
	laneDir := filepath.Join(f.repo.Root(), filepath.FromSlash(RunRelDir), f.runID, l.ID)
	receiptPath := filepath.Join(f.repo.Root(), filepath.FromSlash(res.Awaiting.Receipt))
	rel, err := filepath.Rel(laneDir, filepath.Dir(receiptPath))
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{ReportFileName: "regenerated gen.txt in the merge\n", DoDFileName: "ok\n"} {
		if err := os.WriteFile(filepath.Join(laneDir, rel, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	rc := LaneReceipt{SchemaVersion: ReceiptSchemaVersion, RunID: f.runID, Lane: l.ID, Branch: l.Branch, Commits: []string{merge},
		DefinitionOfDone: &DoDRun{Command: "make check", ExitCode: zero(), Output: filepath.ToSlash(filepath.Join(rel, DoDFileName))},
		Report:           filepath.ToSlash(filepath.Join(rel, ReportFileName)), Model: "claude-fable-5"}
	data, err := json.Marshal(rc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receiptPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Receipt(f.repo.Root(), f.runID, receiptPath, f.stages, Options{}); err != nil {
		t.Fatalf("the resolver's receipt verifies: %v", err)
	}
	var outs []Await
	for range 2 {
		outs = append(outs, stepAwait(t, f.repo, f.runID, f.stages, Options{}))
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatalf("the run reads after the fresh round is handed out: %v", err)
	}
	l = st.Lanes[st.current()]
	if len(l.Validation) != 2 {
		t.Fatalf("premise: the fresh round is open: %d rounds", len(l.Validation))
	}
	for _, a := range outs {
		v := validatorAt(t, f.repo.Root(), f.runID, a.Receipt)
		if !slices.Equal(v.WriterModels, []string{"claude-mythos-5", "claude-fable-5"}) || a.Class == ClassResearch {
			t.Fatalf("the fresh round's reviewer differs from the first writer's model and the resolver's: %+v %+v", a, v)
		}
	}
	for _, v := range l.Validation[0].Validators {
		if len(v.WriterModels) > 0 && !slices.Equal(v.WriterModels, []string{"claude-mythos-5"}) {
			t.Fatalf("the first round keeps the writers it was handed out under: %+v", v.WriterModels)
		}
	}
}
