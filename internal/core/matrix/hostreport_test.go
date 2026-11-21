package matrix

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/gittest"
)

// handOffWithClone makes copy key's clone and records its hand-off to the
// host, as the run's start does, and returns the clone.
func handOffWithClone(t *testing.T, r *gittest.Repo, home string, m lab.Minted, rec RunRecord, key TrialKey, kind TaskKind) string {
	t.Helper()
	clone, err := makeCopyClone(m.ID, r.Root(), cloneStore(home, rec.RootSHA), rec.RunID, rec.FrozenCommit, key)
	if err != nil {
		t.Fatal(err)
	}
	h := HandOff{SchemaVersion: HandOffSchema, Run: rec.RunID, Key: key, Kind: kind, Source: pinOf(t, rec, key.Cell).Source,
		State: HandOffWaiting, Clone: clone, Brief: "make the copy"}
	data, err := encodeCheck(h)
	if err != nil {
		t.Fatal(err)
	}
	proc, err := lab.OpenProcedure(r.Root(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer proc.Close()
	if err := proc.WriteState(handOffRel(rec.RunID, key), data); err != nil {
		t.Fatal(err)
	}
	return clone
}

// trialAt is copy key's trial as written, compacted, so its unpinned entries
// read as one line each.
func trialAt(t *testing.T, home string, m lab.Minted, rec RunRecord, key TrialKey) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(runDir(home, m, rec.RunID), key.rel()))
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err := json.Compact(&b, data); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// hostReceipt is a copy receipt with fields set over a made copy's.
func hostReceipt(t *testing.T, fields map[string]any) []byte {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	rc := map[string]any{"schema_version": HandOffSchema, "outcome": OutcomeMade, "started_at": now.Add(-time.Minute), "finished_at": now}
	for k, v := range fields {
		rc[k] = v
	}
	data, err := json.Marshal(rc)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// A host copy's trial is built from the host's receipt alone, so it says so:
// its hidden tests' result is marked self-reported, as are the model served,
// the harness version and the times the host gave; it carries no call
// receipts and says that too; and every served field the host does not
// report, its tokens and cost among them, is marked not reported.
func TestAHostCopysTrialMarksWhatOnlyTheHostReported(t *testing.T) {
	r, home, m, rec, _ := checkLab(t)
	key := TrialKey{Task: "feature-a", Cell: "host-big", Repeat: 1}
	clone := handOffWithClone(t, r, home, m, rec, key, KindCode)
	receipt := hostReceipt(t, map[string]any{"model_served": "a-served-model", "harness_version": "9.9.9",
		"model_time_ms": 5000, "score": map[string]int{"passed": 4, "total": 4}})
	got, err := CollectCopy(r.Root(), home, m.ID, rec.RunID, key, CopyBack{Receipt: receipt, Transcript: []byte("edited feature.go in " + clone + "\n")})
	if err != nil {
		t.Fatalf("CollectCopy = %v", err)
	}
	data := trialAt(t, home, m, rec, key)
	for _, want := range []string{
		`{"field":"trial.score","reason":"self_reported"}`,
		`{"field":"trial.calls","reason":"not_reported"}`,
		`{"field":"trial.times","reason":"self_reported"}`,
		`{"field":"served.model","reason":"self_reported"}`,
		`{"field":"asked.harness_version","reason":"self_reported"}`,
		`{"field":"served.provider","reason":"not_reported"}`,
		`{"field":"served.prompt_tokens","reason":"not_reported"}`,
		`{"field":"served.cost","reason":"not_reported"}`,
	} {
		if !strings.Contains(data, want) {
			t.Errorf("the host copy's trial does not carry %s:\n%s", want, data)
		}
	}
	if !got.HostReportedScore() {
		t.Errorf("the trial returned does not say its score is the host's report: %+v", got.Unpinned)
	}
	// What the host does not give is not reported, never self-reported.
	key2 := TrialKey{Task: "feature-a", Cell: "host-big", Repeat: 2}
	clone2 := handOffWithClone(t, r, home, m, rec, key2, KindCode)
	bare := hostReceipt(t, map[string]any{"score": map[string]int{"passed": 1, "total": 4}})
	if _, err := CollectCopy(r.Root(), home, m.ID, rec.RunID, key2, CopyBack{Receipt: bare, Transcript: []byte("edited feature.go in " + clone2 + "\n")}); err != nil {
		t.Fatalf("CollectCopy = %v", err)
	}
	data2 := trialAt(t, home, m, rec, key2)
	for _, want := range []string{`{"field":"served.model","reason":"not_reported"}`, `{"field":"asked.harness_version","reason":"not_reported"}`, `{"field":"trial.score","reason":"self_reported"}`} {
		if !strings.Contains(data2, want) {
			t.Errorf("a bare receipt's trial does not carry %s:\n%s", want, data2)
		}
	}
}

// A writing copy's text is kept only once its trial has passed the secret
// screen, so a hand-back the screen refuses keeps nothing, and the copy can
// be handed back again with another document.
func TestAWritingCopyTheScreenRefusedCanBeHandedBackAgain(t *testing.T) {
	r, home, m, rec, _ := checkLab(t)
	key := TrialKey{Task: "doc-page", Cell: "host-big", Repeat: 1}
	clone := handOffWithClone(t, r, home, m, rec, key, KindWriting)
	token := syntheticToken(41)
	transcript := []byte("wrote the page in " + clone + "\n")
	_, err := CollectCopy(r.Root(), home, m.ID, rec.RunID, key, CopyBack{Receipt: hostReceipt(t, map[string]any{"model_served": token}),
		Transcript: transcript, Answer: []byte("# Guide\n\nThe first draft.\n")})
	if err == nil || strings.Contains(err.Error(), token) {
		t.Fatalf("a hand-back whose receipt carries a credential = %v; want it refused, the value never shown", err)
	}
	if _, err := os.Stat(filepath.Join(runDir(home, m, rec.RunID), "writing", "doc-page", "host-big", "1.md")); !os.IsNotExist(err) {
		t.Errorf("a refused hand-back kept the copy's text: %v", err)
	}
	corrected := "# Guide\n\nA corrected draft.\n"
	if _, err := CollectCopy(r.Root(), home, m.ID, rec.RunID, key, CopyBack{Receipt: hostReceipt(t, nil),
		Transcript: transcript, Answer: []byte(corrected)}); err != nil {
		t.Fatalf("the corrected hand-back = %v; want it taken", err)
	}
	kept, err := os.ReadFile(filepath.Join(runDir(home, m, rec.RunID), "writing", "doc-page", "host-big", "1.md"))
	if err != nil || string(kept) != corrected {
		t.Fatalf("the kept text = %q (%v); want the corrected document", kept, err)
	}
}

// A code cell whose hidden tests' results are the host's report says so: the
// cell counts them, a limitation names its trials, and a comparison resting on
// them is qualified, never an unqualified ranking.
func TestTheReportQualifiesPassRatesTheHostReported(t *testing.T) {
	r, _, m, rec := reportLab(t, func(in *runInputs) {
		in.Declaration = []byte(strings.Replace(string(in.Declaration), `"max": 20}`, `"max": 20, "min_difference": 0.95}`, 1))
	})
	rep, err := BuildReport(r.Root(), m.ID, rec.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if big := codeCell(t, rep, "host-big"); big.HostReported != 3 {
		t.Errorf("host-big counts %d host-reported results; want 3", big.HostReported)
	}
	if cheap := codeCell(t, rep, "or-cheap"); cheap.HostReported != 0 {
		t.Errorf("or-cheap counts %d host-reported results; want none", cheap.HostReported)
	}
	c := rep.Code[0].Comparisons[0]
	if !c.Ranked || !c.HostReported || !strings.Contains(c.Note, "host's report") {
		t.Errorf("a ranked comparison on host-reported results = %+v; want it qualified as the host's report", c)
	}
	all := strings.Join(rep.Limitations, "\n")
	for _, want := range []string{
		"the hidden tests' results of feature-a on host-big (repeats 1, 2 and 3) are the host's report",
		"carry no call receipts",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("the limitations say nothing of %q:\n%s", want, all)
		}
	}
	if !strings.Contains(strings.Join(rep.Improvements, "\n"), "run feature-a's hidden tests through abcd") {
		t.Errorf("the improvements do not suggest running the hidden tests through abcd:\n%s", strings.Join(rep.Improvements, "\n"))
	}
}
