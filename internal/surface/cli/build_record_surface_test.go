package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
)

// completedRun writes the state of a run whose one lane has landed, as the
// loop leaves it, into the checkout's run tier, and returns its id.
func completedRun(t *testing.T, root string) string {
	t.Helper()
	id := "run-2609300000000001"
	at := time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC)
	head := strings.Repeat("b", 40)
	st := loop.State{
		SchemaVersion: loop.SchemaVersion, RunID: id, Key: "itd-10", Intent: "itd-10", Spec: "spc-1", Driver: loop.DriverHost,
		CreatedAt: at, UpdatedAt: at, Pending: []loop.PendingStep{},
		Lanes: []loop.Lane{{
			ID: "lane-1", Key: "itd-10", SpecStep: 1, StepTitle: "The state file", Stage: loop.StageDone,
			Branch: "build/" + id + "-lane-1", BaseSHA: strings.Repeat("a", 40), HeadSHA: head, PR: 7,
			Receipts: []loop.ReceiptRecord{{Role: loop.RoleImplementer, Receipt: "receipt.json", Model: "a-model"}},
			Validation: []loop.ValidationRound{{Round: 1, HeadSHA: head, Validators: []loop.ValidatorRun{
				{Role: loop.RoleRuthless, Brief: "b.md", Return: "r.md", Verdict: "SHIP", Pass: true},
				{Role: loop.RoleSecurity, Brief: "b.md", Return: "s.md", Verdict: "APPROVE", Pass: true},
			}}},
			Resolves: []loop.Resolution{{Issue: "iss-5", Commit: head, Note: "n", Impact: "fix", Grounds: "pursued: g"}},
			Landing:  &loop.Landing{RecordsDone: true, Records: head, Pushed: head, BodyChecked: true, Merge: "left open", Merged: head},
		}},
		Record: []loop.Entry{{At: at, Lane: "lane-1", Stage: "land", Note: "pull request #7 landed"}},
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, filepath.FromSlash(loop.RunRelDir), id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, loop.StateFileName), data, 0o600); err != nil {
		t.Fatal(err)
	}
	return id
}

// TestImplementRecordRendersTheRunAndCapturesItsTranscripts is criterion 10 at
// the front door: `implement record` renders a completed run's record in text
// and --json, naming its lane, receipt, model, verdicts, fixes and landing; with
// --transcript it captures each transcript into the history store, one capture
// per path, and the record then names it.
func TestImplementRecordRendersTheRunAndCapturesItsTranscripts(t *testing.T) {
	repo := buildRepo(t)
	id := completedRun(t, repo.Root())

	text := mustImplement(t, "implement", "record")
	for _, want := range []string{"run " + id, "complete", "lane-1", "model a-model", "ruthless-reviewer SHIP",
		"security-reviewer APPROVE", "resolves iss-5", "pull request #7", "transcripts: 0"} {
		if !strings.Contains(text, want) {
			t.Fatalf("the record names %q:\n%s", want, text)
		}
	}

	tp := filepath.Join(t.TempDir(), "session-one.jsonl")
	if err := os.WriteFile(tp, []byte(`{"role":"user","text":"build it"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := mustImplement(t, "--json", "implement", "record", "--run", id, "--transcript", tp)
	var rec loop.RunRecord
	if err := json.Unmarshal([]byte(out), &rec); err != nil {
		t.Fatalf("--json is the record: %v\n%s", err, out)
	}
	if len(rec.Transcripts) != 1 || rec.Transcripts[0].Session != "session-one" || !rec.Transcripts[0].Wrote {
		t.Fatalf("the transcript is captured into the history store and recorded: %+v", rec.Transcripts)
	}
	if len(rec.Lanes) != 1 || rec.Lanes[0].Receipts[0].Model != "a-model" || len(rec.Lanes[0].Verdicts) != 2 {
		t.Fatalf("the --json record names the lane's receipts and verdicts: %+v", rec.Lanes)
	}
	if !strings.Contains(mustImplement(t, "history", "list"), "session-one") {
		t.Fatal("the history store holds the captured transcript")
	}
}

// TestImplementRecordRefusesTranscriptsForARunInProgress: the transcripts are
// the run's, captured at its end, so a run in progress is refused in the
// refusal shape and nothing is captured.
func TestImplementRecordRefusesTranscriptsForARunInProgress(t *testing.T) {
	buildRepo(t)
	mustImplement(t, "build", "itd-10")
	tp := filepath.Join(t.TempDir(), "early.jsonl")
	if err := os.WriteFile(tp, []byte(`{"role":"user","text":"x"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ref := refusalDocs(t, 2, "--json", "implement", "record", "--transcript", tp)
	if ref["stage"] != "record" || !strings.Contains(ref["reason"].(string), "not complete") {
		t.Fatalf("a run in progress is refused at the record: %v", ref)
	}
}
