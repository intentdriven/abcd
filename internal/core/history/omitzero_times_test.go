package history

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestAZeroTimePrintsNoKey (iss-2611252133341605, dcn-2611271510101433): a
// listing's time that was never set carries no key in its JSON, and a set one
// carries its instant. omitempty never drops a zero time, so `abcd history
// staged --all-repos --json` printed oldest_staged_at as 0001-01-01T00:00:00Z
// for a repository holding a quarantined transcript and nothing staged, which
// a reader takes for an entry staged at that instant.
func TestAZeroTimePrintsNoKey(t *testing.T) {
	when := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	const stamp = `"2026-11-25T12:00:00Z"`
	for _, c := range []struct {
		name string
		v    any
		keys []string
		set  bool
	}{
		{"a backlog of quarantined transcripts only", RepoBacklog{RootSHA: "a1b2c3", Quarantined: 1, QuarantinedBytes: 10},
			[]string{"oldest_staged_at"}, false},
		{"a backlog with a staged entry", RepoBacklog{RootSHA: "a1b2c3", Staged: 1, OldestStagedAt: when},
			[]string{"oldest_staged_at"}, true},
		{"a quarantined entry with no times", Quarantined{SessionID: "s1", Path: "q/s1.jsonl"},
			[]string{"staged_at", "quarantined_at"}, false},
		{"a quarantined entry with both times", Quarantined{SessionID: "s1", Path: "q/s1.jsonl", StagedAt: when, QuarantinedAt: when},
			[]string{"staged_at", "quarantined_at"}, true},
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
