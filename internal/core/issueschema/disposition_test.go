package issueschema

import "testing"

// ParseDisposition reads a value the way the strict ledger parser reads it: a
// trailing YAML comment is not part of the value, and a bare YAML null is no
// value at all. It kept its own `strings.Cut` and took the raw remainder, so
// `state: held  # waiting` reached the standing computation as the state
// `held  # waiting`, a supersession edge with a comment after it read as no edge,
// and `exit_condition: null` read as the exit condition "null" — each a record
// the verb reads one way and the board another (iss-2608241347321759).
func TestParseDispositionHonoursCommentAndNull(t *testing.T) {
	const id = "dsp-2"
	for _, tc := range []struct {
		name       string
		line       string
		state      string
		supersedes string
		exit       string
	}{
		{"comment after state", "state: held  # waiting on the rethink", "held", "", ""},
		{"comment after a supersession", "supersedes_disposition: dsp-1 # replaces the first", "", "dsp-1", ""},
		{"comment after a quoted state", `state: "accepted" # ok`, "accepted", "", ""},
		{"bare null exit condition", "exit_condition: null", "", "", ""},
		{"tilde exit condition with comment", "exit_condition: ~ # none yet", "", "", ""},
		{"NULL exit condition", "exit_condition: NULL", "", "", ""},
		{"quoted null is a string", `exit_condition: "null"`, "", "", "null"},
		{"a hash with no space before it is content", "exit_condition: issue#4", "", "", "issue#4"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := ParseDisposition(id, "---\n"+tc.line+"\n---\n")
			if !rec.WellFormed {
				t.Fatalf("%q read as not well-formed", tc.line)
			}
			if rec.State != tc.state || rec.Supersedes != tc.supersedes || rec.ExitCondition != tc.exit {
				t.Errorf("%q read as state=%q supersedes=%q exit=%q; want state=%q supersedes=%q exit=%q",
					tc.line, rec.State, rec.Supersedes, rec.ExitCondition, tc.state, tc.supersedes, tc.exit)
			}
		})
	}
}
