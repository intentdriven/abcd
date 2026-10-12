package embargo

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func strp(s string) *string { return &s }

// sampleState uses reserved-looking example ids: no real advisory is named.
func sampleState() State {
	at := time.Date(2026, 10, 12, 6, 0, 0, 0, time.UTC)
	return State{
		SchemaVersion: SchemaVersion,
		Repo:          "example/example",
		IntakeAt:      &at,
		Advisories: []Advisory{{
			GHSAID:     "GHSA-xxxx-xxxx-xxxx",
			IssueID:    "iss-2610120000000001",
			Title:      "an example title",
			Severity:   "major",
			ForgeState: "draft",
		}, {
			GHSAID:     "GHSA-wwww-wwww-wwww",
			IssueID:    "iss-2610120000000002",
			Title:      "another example title",
			CVEID:      strp("CVE-2026-0000"),
			Severity:   "minor",
			ForgeState: "triage",
			Verdict:    strp("not-a-vulnerability"),
			DecidedBy:  strp("Alice"),
			DecidedAt:  &at,
			Reason:     strp("the behaviour is documented"),
			ClosedAt:   &at,
			History: []HistoryEntry{{
				Verdict: "vulnerability", DecidedBy: "Alice", DecidedAt: at,
			}},
		}},
		Pending: []PendingAdvisory{{
			GHSAID: "GHSA-mmmm-mmmm-mmmm", Title: "a pending example title", CVEID: strp("CVE-2026-0002"), ForgeState: "triage",
		}},
	}
}

func openedStore(t *testing.T) Store {
	t.Helper()
	f := newFixture(t)
	s, _, err := Open(f.home, f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// TestStateRoundTrip: what SaveState writes, Verify reads back unchanged.
func TestStateRoundTrip(t *testing.T) {
	s := openedStore(t)
	want := sampleState()
	if err := s.SaveState(want); err != nil {
		t.Fatal(err)
	}
	got, err := s.Verify()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip:\n got %+v\nwant %+v", got, want)
	}
}

// TestStateAbsentIsEmpty: a store with no state yet reads as an empty one.
func TestStateAbsentIsEmpty(t *testing.T) {
	s := openedStore(t)
	got, err := s.Verify()
	if err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != SchemaVersion || len(got.Advisories) != 0 {
		t.Fatalf("absent state read as %+v", got)
	}
}

// TestStateRefusesUnknownKey: a key this version does not name is refused,
// never ignored (unrecognized-input-never-writes).
func TestStateRefusesUnknownKey(t *testing.T) {
	s := openedStore(t)
	writeState(t, s, `{"schema_version":1,"repo":"example/example","intake_at":null,"advisories":[],"note":"x"}`)
	_, err := s.Verify()
	mustRefuse(t, err, "state")
}

// TestStateRefusesMalformedEntries: each field the state's readers rely on is
// held to its shape.
func TestStateRefusesMalformedEntries(t *testing.T) {
	for name, mutate := range map[string]func(*State){
		"schema":        func(st *State) { st.SchemaVersion = SchemaVersion + 1 },
		"repo":          func(st *State) { st.Repo = "example/../x" },
		"ghsa":          func(st *State) { st.Advisories[0].GHSAID = "GHSA-1111-1111-1111" },
		"issue":         func(st *State) { st.Advisories[0].IssueID = "iss-x" },
		"severity":      func(st *State) { st.Advisories[0].Severity = "high" },
		"forge state":   func(st *State) { st.Advisories[0].ForgeState = "open" },
		"verdict":       func(st *State) { st.Advisories[0].Verdict = strp("maybe") },
		"cve":           func(st *State) { st.Advisories[0].CVEID = strp("CVE-26-1") },
		"duplicate":     func(st *State) { st.Advisories[1].GHSAID = st.Advisories[0].GHSAID },
		"duplicate iss": func(st *State) { st.Advisories[1].IssueID = st.Advisories[0].IssueID },
		"pending ghsa":  func(st *State) { st.Pending = []PendingAdvisory{{GHSAID: "GHSA-1111-1111-1111", ForgeState: "triage"}} },
		"pending state": func(st *State) { st.Pending = []PendingAdvisory{{GHSAID: "GHSA-mmmm-mmmm-mmmm", ForgeState: "open"}} },
		"pending cve": func(st *State) {
			st.Pending = []PendingAdvisory{{GHSAID: "GHSA-mmmm-mmmm-mmmm", ForgeState: "triage", CVEID: strp("x")}}
		},
		"pending held": func(st *State) {
			st.Pending = []PendingAdvisory{{GHSAID: st.Advisories[0].GHSAID, ForgeState: "triage"}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			s := openedStore(t)
			st := sampleState()
			mutate(&st)
			if err := s.SaveState(st); err == nil {
				t.Fatal("SaveState wrote a malformed state")
			}
			if _, err := os.Lstat(s.StatePath()); !os.IsNotExist(err) {
				t.Fatalf("a refused save left a file: %v", err)
			}
		})
	}
}

// TestStateWrittenOwnerOnly: the state is written 0600, and a state file
// another account could write, or a symlink in its place, is refused.
func TestStateWrittenOwnerOnly(t *testing.T) {
	s := openedStore(t)
	if err := s.SaveState(sampleState()); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Lstat(s.StatePath())
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("state written %v, want 0600", fi.Mode().Perm())
	}
	if err := os.Chmod(s.StatePath(), 0o622); err != nil {
		t.Fatal(err)
	}
	_, err = s.Verify()
	mustRefuse(t, err, "state")

	if err := os.Remove(s.StatePath()); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "elsewhere.json")
	if err := os.WriteFile(target, []byte(`{"schema_version":1,"repo":"example/example","intake_at":null,"advisories":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, s.StatePath()); err != nil {
		t.Fatal(err)
	}
	_, err = s.Verify()
	mustRefuse(t, err, "state")
}

// TestStateReadsSchemaOne: a state written before the pending list existed
// (schema 1) is read as the current schema with nothing pending, and the next
// save writes the current schema; a schema-1 file that carries a pending list
// was not written by any abcd and is refused.
func TestStateReadsSchemaOne(t *testing.T) {
	s := openedStore(t)
	writeState(t, s, `{"schema_version":1,"repo":"example/example","intake_at":null,"advisories":[`+
		`{"ghsa_id":"GHSA-xxxx-xxxx-xxxx","issue_id":"iss-2610120000000001","title":"an example title","cve_id":null,`+
		`"severity":"major","forge_state":"draft","verdict":null,"decided_by":null,"decided_at":null,"reason":null,`+
		`"closed_at":null,"published_at":null}]}`)
	st, err := s.Verify()
	if err != nil {
		t.Fatalf("a schema-1 state was refused: %v", err)
	}
	if st.SchemaVersion != SchemaVersion || len(st.Advisories) != 1 || len(st.Pending) != 0 {
		t.Fatalf("schema-1 state read as %+v", st)
	}
	if err := s.SaveState(st); err != nil {
		t.Fatal(err)
	}
	if data := readFile(t, s.StatePath()); !strings.Contains(data, `"schema_version": 2`) {
		t.Fatalf("the save did not write the current schema:\n%s", data)
	}
	writeState(t, s, `{"schema_version":1,"repo":"example/example","intake_at":null,"advisories":[],"pending":[]}`)
	_, err = s.Verify()
	mustRefuse(t, err, "state")
}

func writeState(t *testing.T, s Store, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(s.StatePath()), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.StatePath(), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
