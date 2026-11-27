package autonomy

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestAZeroTimePrintsNoKey (iss-2611252133341605, dcn-2611271510101433): a
// stop check's time that was never set carries no key in its JSON, and a set
// one carries its instant. omitempty never drops a zero time, so each printed
// 0001-01-01T00:00:00Z, which a reader takes for an instant.
func TestAZeroTimePrintsNoKey(t *testing.T) {
	when := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	const stamp = `"2026-11-25T12:00:00Z"`
	for _, c := range []struct {
		name string
		v    any
		keys []string
		set  bool
	}{
		{"a session state with no times", SessionState{SchemaVersion: SchemaVersion, State: StateWorking},
			[]string{"next_look", "waiting_since", "person_prompt_at"}, false},
		{"a session state with every time", SessionState{SchemaVersion: SchemaVersion, State: StateWaiting,
			NextLook: when, WaitingSince: when, PersonPromptAt: when},
			[]string{"next_look", "waiting_since", "person_prompt_at"}, true},
		{"an owed set with no reservation", Owed{}, []string{"next_renewal", "wake_by"}, false},
		{"an owed set with a reservation", Owed{NextRenewal: when, WakeBy: when}, []string{"next_renewal", "wake_by"}, true},
		{"owned work moved at an unknown time", OwnedRef{Ref: "branch:x"}, []string{"changed"}, false},
		{"owned work moved at a known time", OwnedRef{Ref: "branch:x", Changed: when}, []string{"changed"}, true},
	} {
		b, err := json.Marshal(c.v)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		got := string(b)
		if strings.Contains(got, "0001-01-01") {
			t.Errorf("%s: a zero time was printed: %s", c.name, got)
		}
		for _, k := range c.keys {
			if has := strings.Contains(got, `"`+k+`"`); has != c.set {
				t.Errorf("%s: %s present = %v, want %v, in %s", c.name, k, has, c.set, got)
			}
			if c.set && !strings.Contains(got, `"`+k+`":`+stamp) {
				t.Errorf("%s: %s does not carry its instant: %s", c.name, k, got)
			}
		}
	}
}

// TestAnOlderSessionStateStillLoads: a session state an older abcd wrote
// spells each unset time as the zero instant. The file still loads, each such
// time reads as zero, and the next save drops the keys.
func TestAnOlderSessionStateStillLoads(t *testing.T) {
	ctx := context.Background()
	copyRoot := t.TempDir()
	const id = "sess-old"
	dir := filepath.Join(copyRoot, filepath.FromSlash(SessionsDir))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	old := `{"schema_version":` + strconv.Itoa(SchemaVersion) + `,"state":"working",` +
		`"next_look":"0001-01-01T00:00:00Z","blocks":0,"wakes":0,` +
		`"waiting_since":"0001-01-01T00:00:00Z","person_prompt_at":"0001-01-01T00:00:00Z"}`
	if err := os.WriteFile(filepath.Join(dir, id+".json"), []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := LoadSession(ctx, copyRoot, id)
	if err != nil {
		t.Fatalf("an older session state does not load: %v", err)
	}
	if st.State != StateWorking || !st.NextLook.IsZero() || !st.WaitingSince.IsZero() || !st.PersonPromptAt.IsZero() {
		t.Fatalf("an older session state read as %+v; want working with every time zero", st)
	}
	if err := SaveSession(copyRoot, id, st); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, id+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "0001-01-01") {
		t.Errorf("the saved state still spells a zero time: %s", raw)
	}
}
