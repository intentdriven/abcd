//go:build unix

package matrix

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/testsecret"
)

// A collection that fails once the copy's findings are recorded can be tried
// again: a record already written with the same bytes counts as written, so
// the retry goes on to the gates and deletes the clone. A copy whose findings
// the secret screen refuses is recorded as failed, with nothing of its answer
// kept, so its clone is deleted too and no retry is stuck on it.
func TestACheckCopysCollectionCanBeRetriedOnceItsFindingsAreRecorded(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("a read-only directory does not stop root deleting in it")
	}
	r, home, m, rec, store := checkLab(t)
	key := TrialKey{Task: checkTask, Cell: "host-big", Repeat: 1}
	gates, err := snapshotGates(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	clone, err := makeClone(r.Root(), store, rec.RunID, rec.FrozenCommit, key)
	if err != nil {
		t.Fatal(err)
	}
	in := checkInputs(t, r, rec, store, key, CheckDocFidelity, gates, dfAnswer(t, "It refuses a hazard"))

	// The clone cannot be deleted the first time: its parent is read-only.
	parent := filepath.Dir(clone)
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })
	if _, err := collectCheckCopy(m.ID, in); err == nil || !strings.Contains(err.Error(), "could not be deleted") {
		t.Fatalf("a clone that cannot be deleted = %v; want a refusal saying so", err)
	}
	recordPath := filepath.Join(runDir(home, m, rec.RunID), "check", checkTask, "copies", "host-big", "1.json")
	first, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatalf("the findings are not recorded: %v", err)
	}
	if err := os.Chmod(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	c, err := collectCheckCopy(m.ID, in)
	if err != nil {
		t.Fatalf("the retry = %v; want the record taken as written and the clone deleted", err)
	}
	if _, err := os.Stat(clone); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the retry left the clone: %v", err)
	}
	if again, err := os.ReadFile(recordPath); err != nil || !bytes.Equal(again, first) || !c.Answered {
		t.Fatalf("the retry's record = %s (%v); want the first, unchanged", again, err)
	}

	// The screen refuses a credential in a finding: the copy fails, its
	// clone is deleted, and the credential is nowhere in the record.
	token := "ghp_" + testsecret.Synthetic(11, 36)
	sKey := TrialKey{Task: checkTask, Cell: "host-big", Repeat: 2}
	sClone, err := makeClone(r.Root(), store, rec.RunID, rec.FrozenCommit, sKey)
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
