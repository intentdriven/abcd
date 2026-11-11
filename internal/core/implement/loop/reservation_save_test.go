package loop

// reservation_save_test.go proves iss-2611020719494343: a lane whose change
// raises the run state's schema leaves the run that drives it where it was.
// The binary under test plays the lane's newer abcd, and a file at
// schemaVersionUnkeyed plays the run a driver one version older wrote: that
// driver refuses anything above its own version and any field it does not
// name (olderDriverRead), which is exactly how a release refuses a newer
// file.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
)

// runStatePath is the run's state file.
func runStatePath(root, runID string) string {
	return filepath.Join(root, filepath.FromSlash(StateRelPath(runID)))
}

// stateAtVersion rewrites the run's state file at version and returns it.
func stateAtVersion(t *testing.T, root, runID string, version int) []byte {
	t.Helper()
	path := runStatePath(root, runID)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	from := fmt.Sprintf(`"schema_version": %d,`, SchemaVersion)
	if !bytes.Contains(data, []byte(from)) {
		t.Fatalf("the state file is not at version %d:\n%s", SchemaVersion, data)
	}
	data = bytes.Replace(data, []byte(from), []byte(fmt.Sprintf(`"schema_version": %d,`, version)), 1)
	if version <= schemaVersionUnadmitted {
		// The admission joined at version 26, so an older driver's file carries
		// none. It is cut while the file is laid out as writeState lays it.
		data = admittedField.ReplaceAll(data, nil)
	}
	if version <= schemaVersionUnrenamed {
		// The classes that version stored are named as it named them.
		var doc map[string]any
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatal(err)
		}
		unreviewOlder(doc)
		if data, err = json.MarshalIndent(doc, "", "  "); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return data
}

// admittedField is a state file's top-level `admitted`, as writeState lays it out.
var admittedField = regexp.MustCompile(`\n  "admitted": \{[^{}]*\},`)

// topLevelFields is a state file's top-level fields, undecoded.
func topLevelFields(t *testing.T, data []byte) map[string]json.RawMessage {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("the state file does not parse: %v\n%s", err, data)
	}
	return m
}

// olderDriverRead fails unless a driver that wrote before as its newest file
// can read after: at no later version and with no field before did not carry,
// the two things a release refuses a file for. fields other than the record
// and its date are unchanged byte for byte, so nothing the driver wrote was
// reshaped under it, and the record only grew.
func olderDriverRead(t *testing.T, before, after []byte) {
	t.Helper()
	was, is := topLevelFields(t, before), topLevelFields(t, after)
	for k, v := range is {
		old, ok := was[k]
		if !ok {
			t.Errorf("the save added the field %q, which the older driver refuses", k)
			continue
		}
		if k == "record" || k == "updated_at" {
			continue
		}
		if !bytes.Equal(old, v) {
			t.Errorf("the save changed %s from %s to %s; the older driver's file is not its own to rewrite", k, old, v)
		}
	}
	for k := range was {
		if _, ok := is[k]; !ok {
			t.Errorf("the save dropped the field %q", k)
		}
	}
	var oldRec, newRec []json.RawMessage
	if err := json.Unmarshal(was["record"], &oldRec); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(is["record"], &newRec); err != nil {
		t.Fatal(err)
	}
	if len(newRec) < len(oldRec) {
		t.Fatalf("the save dropped record entries: %d, was %d", len(newRec), len(oldRec))
	}
	for i := range oldRec {
		if !bytes.Equal(oldRec[i], newRec[i]) {
			t.Errorf("the save rewrote record entry %d: %s, was %s", i, newRec[i], oldRec[i])
		}
	}
}

// TestARenewalFromALaneNeverRaisesTheRunThatDrivesIt is the issue's own case:
// the implementer's `implement load --wait`, run through the lane's newer
// abcd, renews the reservation of the run that lives in the driver's checkout.
// The run stays at the version it was read at, so the driver steps.
func TestARenewalFromALaneNeverRaisesTheRunThatDrivesIt(t *testing.T) {
	f := newNarrowFixture(t)
	f.awaitImplementer(t)
	l := currentLane(t, f.repo, f.runID)
	root := f.repo.Root()
	before := stateAtVersion(t, root, f.runID, schemaVersionUnkeyed)

	at := f.tip(t).claimed.Add(61 * time.Minute)
	got := RenewRuns(l.Worktree, at)
	if len(got) != 1 || !got[0].Renewed {
		t.Fatalf("RenewRuns from the lane's worktree = %+v; want the run's reservation renewed", got)
	}

	after, err := os.ReadFile(runStatePath(root, f.runID))
	if err != nil {
		t.Fatal(err)
	}
	olderDriverRead(t, before, after)
	if v := string(topLevelFields(t, after)["schema_version"]); v != fmt.Sprint(schemaVersionUnkeyed) {
		t.Fatalf("the renewal raised the run's schema to %s; the driver reads up to %d", v, schemaVersionUnkeyed)
	}
	if note := recordNote(t, f.repo, f.runID, StageReservationRenewed); !strings.Contains(note, "renewed the reservation") {
		t.Fatalf("the run record logs the renewal: %q", note)
	}

	// The driver takes its next step: a re-told await, with no refusal.
	res, err := advance(root, f.runID, f.stages, Options{Now: func() time.Time { return at }})
	if err != nil {
		t.Fatalf("the driver's next step is refused: %v", err)
	}
	if res.Awaiting == nil || res.Awaiting.Role != RoleImplementer {
		t.Fatalf("the driver's next step finds the implementer still out: %+v", res)
	}
}

// TestEveryWriterALaneReachesKeepsTheVersionItRead: implement status, whose
// renewal runs before it renders, and implement widen, which an issue lane's
// brief has the implementer run from the checkout the run lives in, are the
// other two writers a lane's process can reach. Neither raises the run.
func TestEveryWriterALaneReachesKeepsTheVersionItRead(t *testing.T) {
	for name, write := range map[string]func(t *testing.T, f *narrowFixture, l Lane, at time.Time){
		"implement status": func(t *testing.T, f *narrowFixture, _ Lane, at time.Time) {
			if got := RenewRunsIn(f.repo.Root(), f.runID, at); len(got) != 1 || !got[0].Renewed {
				t.Fatalf("RenewRunsIn = %+v; want the reservation renewed", got)
			}
		},
		"implement widen": func(t *testing.T, f *narrowFixture, l Lane, at time.Time) {
			res, err := Widen(f.repo.Root(), f.runID, l.ID, []string{"internal/widened.go"}, Options{Now: func() time.Time { return at }})
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Contains(res.Added, "internal/widened.go") {
				t.Fatalf("the widening adds the path: %+v", res)
			}
		},
		"a drain's renewal": func(t *testing.T, f *narrowFixture, _ Lane, at time.Time) {
			st := DrainState{Lanes: []DrainLane{{Issue: secondIssue, RunID: f.runID, Outcome: DrainLaneInProgress}}}
			if notes := renewInProgress(f.repo.Root(), st, nil, at); len(notes) != 0 {
				t.Fatalf("the renewal says %q", notes)
			}
			if got := recordNote(t, f.repo, f.runID, StageReservationRenewed); got == "" {
				t.Fatal("the renewal is not logged")
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newNarrowFixture(t)
			f.awaitImplementer(t)
			l := currentLane(t, f.repo, f.runID)
			root := f.repo.Root()
			before := stateAtVersion(t, root, f.runID, schemaVersionUnkeyed)

			write(t, f, l, f.tip(t).claimed.Add(61*time.Minute))

			after, err := os.ReadFile(runStatePath(root, f.runID))
			if err != nil {
				t.Fatal(err)
			}
			olderDriverRead(t, before, after)
			if v := string(topLevelFields(t, after)["schema_version"]); v != fmt.Sprint(schemaVersionUnkeyed) {
				t.Fatalf("%s raised the run's schema to %s", name, v)
			}
		})
	}
}

// TestARunANewerDriverWroteStaysRefusedAndUntouched: the other direction. A
// lane cut before the driver moved on reads a run at a version it does not
// know. It writes nothing, so no field it cannot name is dropped and no
// version is lowered, it says so rather than renewing nothing in silence, and
// the run stays refused to this abcd in the same words as ever.
func TestARunANewerDriverWroteStaysRefusedAndUntouched(t *testing.T) {
	f := newNarrowFixture(t)
	f.awaitImplementer(t)
	l := currentLane(t, f.repo, f.runID)
	root := f.repo.Root()
	newer := SchemaVersion + 1

	// A field this build does not name: the newer driver's.
	data := stateAtVersion(t, root, f.runID, newer)
	data = bytes.Replace(data, []byte("{\n"), []byte("{\n  \"a_field_of_the_newer_driver\": true,\n"), 1)
	path := runStatePath(root, f.runID)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	tip := f.tip(t)

	got := RenewRuns(l.Worktree, tip.claimed.Add(61*time.Minute))
	if len(got) != 1 || got[0].Renewed || got[0].RunID != f.runID || !strings.Contains(got[0].Note, "newer abcd") {
		t.Fatalf("RenewRuns over a newer driver's run = %+v; want one note naming the newer abcd, and no renewal", got)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, after) {
		t.Fatalf("a run this abcd cannot read was rewritten:\n%s", after)
	}
	if now := f.tip(t); now.commit != tip.commit {
		t.Fatalf("a renewal of a run this abcd cannot read pushed %+v over %+v", now, tip)
	}
	if _, err := ReadState(root, f.runID); err == nil || !strings.Contains(err.Error(), "which a newer abcd wrote") {
		t.Fatalf("the run stays refused as a newer abcd's: %v", err)
	}
}

// TestOneUnreadableRunDoesNotStopTheRenewalOfAnother: a run a newer abcd wrote
// in the checkout holds no other run's reservation hostage.
func TestOneUnreadableRunDoesNotStopTheRenewalOfAnother(t *testing.T) {
	f := newNarrowFixture(t)
	f.awaitImplementer(t)
	root := f.repo.Root()

	other := "run-2611040000000001"
	dir := filepath.Join(root, filepath.FromSlash(runRel(other)))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"schema_version": %d, "run_id": %q}`+"\n", SchemaVersion+1, other)
	if err := os.WriteFile(filepath.Join(dir, StateFileName), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	got := RenewRunsIn(root, "", f.tip(t).claimed.Add(61*time.Minute))
	var renewed, said bool
	for _, r := range got {
		renewed = renewed || (r.RunID == f.runID && r.Renewed)
		said = said || (r.RunID == other && !r.Renewed && strings.Contains(r.Note, "newer abcd"))
	}
	if !renewed || !said {
		t.Fatalf("RenewRunsIn = %+v; want %s renewed and %s named as a newer abcd's", got, f.runID, other)
	}
}

// TestAReservationSaveRefusesToDropAnotherChange: the save writes the record
// and its date, so a writer that changed anything else is refused rather than
// having the change dropped.
func TestAReservationSaveRefusesToDropAnotherChange(t *testing.T) {
	f := newNarrowFixture(t)
	f.awaitImplementer(t)
	root := f.repo.Root()
	path := runStatePath(root, f.runID)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	err = mutateReservation(root, f.runID, func(st *State) (bool, error) {
		st.Lanes[0].Stage = StageValidate
		return true, nil
	})
	if err == nil || !strings.Contains(err.Error(), "only the run's record") {
		t.Fatalf("a reservation save that changed a lane = %v; want a refusal", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("a refused save wrote the file:\n%s", after)
	}
}

// TestAReservationSaveMasksWhatItAddsAndChangesNothingElse: what it appends is
// stored as every record line is, and at the file's own version the file
// differs from what was read in the record and its date only, the unchanged
// fields keeping their order.
func TestAReservationSaveMasksWhatItAddsAndChangesNothingElse(t *testing.T) {
	f := newNarrowFixture(t)
	f.awaitImplementer(t)
	root := f.repo.Root()
	path := runStatePath(root, f.runID)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 11, 4, 1, 2, 3, 0, time.UTC)
	if err := mutateReservation(root, f.runID, func(st *State) (bool, error) {
		st.Record = append(st.Record, Entry{At: at, Stage: StageReservationRenewed, Note: "a \x1b[31mcoloured\x1b[0m note"})
		st.UpdatedAt = at
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	olderDriverRead(t, before, after)
	if bytes.Contains(after, []byte(`\u001b`)) || bytes.ContainsRune(after, 0x1b) {
		t.Fatalf("the note was stored raw:\n%s", after)
	}
	if !bytes.HasSuffix(after, []byte("}\n")) {
		t.Fatalf("the file keeps its shape: %q", after[len(after)-8:])
	}
	if got, want := fieldOrder(t, after), fieldOrder(t, before); !slices.Equal(got, want) {
		t.Fatalf("the fields moved: %v, was %v", got, want)
	}
	st, err := ReadState(root, f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if !st.UpdatedAt.Equal(at) || st.Record[len(st.Record)-1].Stage != StageReservationRenewed {
		t.Fatalf("the save did not take: %+v", st.Record[len(st.Record)-1])
	}
}

// fieldOrder is a state file's top-level field names, in file order.
func fieldOrder(t *testing.T, data []byte) []string {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(data))
	if _, err := dec.Token(); err != nil {
		t.Fatal(err)
	}
	var out []string
	for dec.More() {
		k, err := dec.Token()
		if err != nil {
			t.Fatal(err)
		}
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			t.Fatal(err)
		}
		out = append(out, k.(string))
	}
	return out
}
