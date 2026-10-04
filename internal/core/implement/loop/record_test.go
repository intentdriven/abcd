package loop

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeCapture is a transcript capturer that stores nothing and reports each
// path as stored under its base name, failing on a path named fail.
func fakeCapture(calls *[]string) TranscriptCapturer {
	return func(path string) (Transcript, error) {
		*calls = append(*calls, path)
		if strings.Contains(path, "fail") {
			return Transcript{}, errors.New("the transcript does not redact")
		}
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		return Transcript{Path: path, Session: base, Stored: "~/.abcd.noindex/transcripts/x/records/" + base + ".jsonl", Wrote: true}, nil
	}
}

// TestTheRunRecordNamesEveryLaneReceiptVerdictModelAndTranscript is criterion
// 10: a completed run's record names every lane, the receipts the loop
// verified with the model each runner reported, every verdict the loop
// recorded, what the landing did, and the transcripts captured into the
// history store, one capture per path.
func TestTheRunRecordNamesEveryLaneReceiptVerdictModelAndTranscript(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.landedToArmed(t)
	var calls []string
	if _, err := CaptureTranscripts(f.repo.Root(), f.runID, []string{"/t/main.jsonl"}, fakeCapture(&calls), Options{}); err == nil {
		t.Fatal("a run that is not complete has no transcripts captured yet")
	} else if r := mustRefusal(t, err); r.Stage != StageRecord {
		t.Fatalf("the refusal is the record's: %+v", r)
	}
	if len(calls) != 0 {
		t.Fatalf("nothing is captured for a run that is not complete: %v", calls)
	}
	f.merged(t, l.HeadSHA)
	if res := f.step(t); !res.Complete {
		t.Fatalf("the run completes once its lane lands: %+v", res)
	}

	rec, err := CaptureTranscripts(f.repo.Root(), f.runID, []string{"/t/main.jsonl", "/t/agent-a.jsonl"}, fakeCapture(&calls), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 {
		t.Fatalf("one history capture per path: %v", calls)
	}
	rec, err = ReadRecord(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if !rec.Complete || len(rec.Lanes) != 1 {
		t.Fatalf("the record names the run's lane: %+v", rec)
	}
	lane := rec.Lanes[0]
	if len(lane.Receipts) != 1 || lane.Receipts[0].Model != f.model || lane.Receipts[0].Role != RoleImplementer {
		t.Fatalf("the record names the receipt and the model its runner reported: %+v", lane.Receipts)
	}
	var verdicts []string
	for _, v := range lane.Verdicts {
		verdicts = append(verdicts, v.Role+" "+v.Verdict)
	}
	if got := strings.Join(verdicts, ", "); got != "ruthless-reviewer SHIP, security-reviewer APPROVE, intent-auditor MET" {
		t.Fatalf("the record names every verdict the loop recorded: %s", got)
	}
	if len(lane.Resolves) != 1 || lane.Resolves[0] != f.issue || lane.PR != 7 || lane.Landing == nil || lane.Landing.Merged == "" {
		t.Fatalf("the record names what the landing did: %+v", lane)
	}
	if len(rec.Transcripts) != 2 || rec.Transcripts[1].Session != "agent-a" {
		t.Fatalf("the record names the transcripts captured into the history store: %+v", rec.Transcripts)
	}
	n := 0
	for _, e := range rec.Record {
		if e.Stage == StageTranscript {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("the record carries a line per captured transcript: %+v", rec.Record)
	}

	// A capture that fails keeps the ones before it and names what failed.
	_, err = CaptureTranscripts(f.repo.Root(), f.runID, []string{"/t/second.jsonl", "/t/fail.jsonl", "/t/never.jsonl"}, fakeCapture(&calls), Options{})
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, "does not redact") {
		t.Fatalf("the failed capture is named: %+v", r)
	}
	rec, _ = ReadRecord(f.repo.Root(), f.runID)
	if len(rec.Transcripts) != 3 || calls[len(calls)-1] != "/t/fail.jsonl" {
		t.Fatalf("the captures before the failure are recorded and none after it is made: %+v %v", rec.Transcripts, calls)
	}
}

// TestAVersion6StateReadsAsOneNothingHasLanded: version 7 added the landing,
// the lane's verified receipts and declared fixes and the run's transcripts, so
// a version-6 file is read as a run nothing has landed yet, and a version-6
// file carrying any of them is not one version 6 wrote, and is refused.
func TestAVersion6StateReadsAsOneNothingHasLanded(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	f.validated(t)
	f.step(t)
	path := filepath.Join(f.repo.Root(), filepath.FromSlash(StateRelPath(f.runID)))
	current := stateBytes(t, f.repo.Root(), f.runID)
	if err := os.WriteFile(path, downgraded(t, current, schemaVersionUnlanded), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil || st.SchemaVersion != SchemaVersion || st.Lanes[0].Landing != nil {
		t.Fatalf("a version-6 file reads as a run nothing has landed: %v", err)
	}
	carrying := strings.Replace(string(current), fmt.Sprintf(`"schema_version": %d,`, SchemaVersion), `"schema_version": 6,`, 1)
	if err := os.WriteFile(path, []byte(carrying), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = ReadState(f.repo.Root(), f.runID)
	if r := mustRefusal(t, err); r.Stage != "state" || !strings.Contains(r.Reason, "landing") {
		t.Fatalf("a version-6 file carrying a landing is refused: %+v", r)
	}
}
