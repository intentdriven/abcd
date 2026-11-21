//go:build unix

package matrix

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/testsecret"
)

// A copy whose findings the secret screen refuses is recorded as failed, with
// nothing of its answer kept, so its clone is deleted too and no retry is
// stuck on it. (A hand-back tried again after its clone could not be deleted
// is TestACheckCopysHandBackDeletesItsLeftoverCloneOnRetry.)
func TestACheckCopyWhoseFindingsTheScreenRefusesIsRecordedAsFailed(t *testing.T) {
	r, home, m, rec, store := checkLab(t)
	gates, err := snapshotGates(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	// The screen refuses a credential in a finding: the copy fails, its
	// clone is deleted, and the credential is nowhere in the record.
	token := "ghp_" + testsecret.Synthetic(11, 36)
	sKey := TrialKey{Task: checkTask, Cell: "host-big", Repeat: 2}
	sClone, err := makeCopyClone(m.ID, r.Root(), store, rec.RunID, rec.FrozenCommit, sKey)
	if err != nil {
		t.Fatal(err)
	}
	answer := strings.Replace(string(dfAnswer(t, "It refuses a hazard")), "internal/core/guard/registry.go:40", token, 1)
	sc, err := collectCheckCopy(m.ID, checkInputs(t, r, rec, store, sKey, CheckDocFidelity, gates, []byte(answer)))
	if err != nil {
		t.Fatalf("a copy whose findings the screen refuses = %v; want it recorded as failed", err)
	}
	if sc.Answered || len(sc.Findings) != 0 || sc.Reason == "" {
		t.Fatalf("copy = %+v; want it the copy's failure, nothing of its answer kept", sc)
	}
	if _, err := os.Stat(sClone); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a copy recorded as failed kept its clone: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(runDir(home, m, rec.RunID), "check", checkTask, "copies", "host-big", "2.json"))
	if err != nil || bytes.Contains(data, []byte(token)) {
		t.Fatalf("the failed copy's record = %s (%v); want it written without the credential", data, err)
	}
}

// A check copy's hand-back whose trial is written but whose clone could not be
// deleted is handed back again to delete it: the retry deletes the clone left
// behind and returns the trial already written, never writing a second.
func TestACheckCopysHandBackDeletesItsLeftoverCloneOnRetry(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("a read-only directory does not stop root deleting in it")
	}
	// The run's check task declares docs/reference as its material.
	r, home, m, rec, _ := checkLabWith(t, map[string]string{"docs/reference/cli.md": "# CLI\n\nIt refuses a hazard.\n"})
	key := TrialKey{Task: checkTask, Cell: "host-big", Repeat: 1}
	gates, err := snapshotGates(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	clone, err := makeCopyClone(m.ID, r.Root(), cloneStore(home, rec.RootSHA), rec.RunID, rec.FrozenCommit, key)
	if err != nil {
		t.Fatal(err)
	}
	h := HandOff{SchemaVersion: HandOffSchema, Run: rec.RunID, Key: key, Kind: KindCheck, Source: pinOf(t, rec, key.Cell).Source,
		State: HandOffWaiting, Clone: clone, Brief: "make the copy", Gates: gates}
	data, err := encodeCheck(h)
	if err != nil {
		t.Fatal(err)
	}
	proc, err := lab.OpenProcedure(r.Root(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := proc.WriteState(handOffRel(rec.RunID, key), data); err != nil {
		t.Fatal(err)
	}
	proc.Close()
	now := time.Now().UTC().Truncate(time.Second)
	receipt, err := json.Marshal(map[string]any{"schema_version": HandOffSchema, "outcome": OutcomeMade, "started_at": now.Add(-time.Minute), "finished_at": now})
	if err != nil {
		t.Fatal(err)
	}
	back := CopyBack{Receipt: receipt, Transcript: []byte("read " + guardChapter + "\n"), Form: CheckDocFidelity, Answer: dfAnswer(t, "It refuses a hazard")}

	// The clone cannot be deleted the first time: its parent is read-only.
	parent := filepath.Dir(clone)
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })
	if _, err := CollectCopy(r.Root(), home, m.ID, rec.RunID, key, back); err == nil || !strings.Contains(err.Error(), "could not be deleted") {
		t.Fatalf("a clone that cannot be deleted = %v; want a refusal saying so", err)
	}
	trialPath := filepath.Join(runDir(home, m, rec.RunID), key.rel())
	first, err := os.ReadFile(trialPath)
	if err != nil {
		t.Fatalf("the trial is not written: %v", err)
	}
	if err := os.Chmod(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := CollectCopy(r.Root(), home, m.ID, rec.RunID, key, back)
	if err != nil {
		t.Fatalf("the retry = %v; want the clone left behind deleted and the trial written returned", err)
	}
	if _, err := os.Stat(clone); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the retry left the clone: %v", err)
	}
	if again, err := os.ReadFile(trialPath); err != nil || !bytes.Equal(again, first) || got.Key != key || got.Status != StatusPassed {
		t.Fatalf("the retry's trial = %+v, on disk %s (%v); want the first, unchanged", got, again, err)
	}
	// A hand-back that lost the trial's write to another says so, rather
	// than that it keeps a clone the other deleted; one whose trial is not
	// written keeps its clone.
	// It promises no retry: the other hand-back deletes the clone, and a
	// hand-back once the clone is gone is refused.
	if err := lostTrialWrite(r.Root(), m.ID, rec.RunID, key, fs.ErrExist); !strings.Contains(err.Error(), "another hand-back") || strings.Contains(err.Error(), "keeps its clone") ||
		strings.Contains(err.Error(), "returns that trial") || !strings.Contains(err.Error(), "nothing more to hand back") {
		t.Errorf("the loser's refusal = %v; want it to name the other hand-back's trial and promise no retry", err)
	}
	if err := lostTrialWrite(r.Root(), m.ID, rec.RunID, TrialKey{Task: checkTask, Cell: "host-big", Repeat: 2}, fs.ErrPermission); !strings.Contains(err.Error(), "keeps its clone") {
		t.Errorf("a refused trial write = %v; want it to keep its clone", err)
	}
	// Once the clone is gone, a further hand-back is refused as before.
	if _, err := CollectCopy(r.Root(), home, m.ID, rec.RunID, key, back); err == nil || !strings.Contains(err.Error(), "already handed back") {
		t.Fatalf("a hand-back after the clone is gone = %v; want it refused as already handed back", err)
	}
}
