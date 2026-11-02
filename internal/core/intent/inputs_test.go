package intent

import (
	"encoding/json"
	"strings"
	"testing"
)

// withVersion is example re-encoded with schema_version set to v, or removed
// when v is nil.
func withVersion(t *testing.T, example string, v any) []byte {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal([]byte(example), &doc); err != nil {
		t.Fatal(err)
	}
	if v == nil {
		delete(doc, "schema_version")
	} else {
		doc["schema_version"] = v
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// TestUnversionedInputsReadAnAbsentVersionAsOne holds the three inputs that
// carried no schema version to the spec's rule, through the decoders the
// writing commands call: absent reads as 1, 1 is read, and any other version
// is refused naming the one this binary reads.
func TestUnversionedInputsReadAnAbsentVersionAsOne(t *testing.T) {
	decoders := []struct {
		name    string
		example string
		decode  func([]byte) error
	}{
		{"audit verdict", VerdictExample, func(raw []byte) error { _, err := decodeVerdict(raw); return err }},
		{"prepass findings", PrepassFindingsExample, func(raw []byte) error { _, err := decodePrepassFindings(raw); return err }},
		{"consistency findings", ConsistencyFindingsExample, func(raw []byte) error { _, err := decodeConsistencyPayload(t.TempDir(), raw); return err }},
	}
	for _, d := range decoders {
		t.Run(d.name, func(t *testing.T) {
			for _, v := range []any{nil, 1} {
				if err := d.decode(withVersion(t, d.example, v)); err != nil {
					t.Errorf("schema_version %v refused: %v", v, err)
				}
			}
			for _, v := range []any{2, -1} {
				err := d.decode(withVersion(t, d.example, v))
				if err == nil || !strings.Contains(err.Error(), "schema_version") || !strings.Contains(err.Error(), "version 1") {
					t.Errorf("schema_version %v: err = %v, want a refusal naming version 1", v, err)
				}
			}
		})
	}
}

func TestInputExamplesPassTheirChecks(t *testing.T) {
	if err := CheckVerdict([]byte(VerdictExample)); err != nil {
		t.Errorf("VerdictExample: %v", err)
	}
	if err := CheckPrepassFindings([]byte(PrepassFindingsExample)); err != nil {
		t.Errorf("PrepassFindingsExample: %v", err)
	}
	if err := CheckConsistencyFindings(t.TempDir(), []byte(ConsistencyFindingsExample)); err != nil {
		t.Errorf("ConsistencyFindingsExample: %v", err)
	}
}

// TestInputChecksRefuseWhatTheIngestRefuses spot-checks that each check is
// the ingest's own validation and not a bare parse.
func TestInputChecksRefuseWhatTheIngestRefuses(t *testing.T) {
	bad := func(example, from, to string) []byte {
		if !strings.Contains(example, from) {
			t.Fatalf("the example carries no %q", from)
		}
		return []byte(strings.Replace(example, from, to, 1))
	}
	if err := CheckVerdict(bad(VerdictExample, `"rcp-0123456789ab"`, `"rcp-x"`)); err == nil {
		t.Error("CheckVerdict accepted a malformed receipt id")
	}
	if err := CheckVerdict(bad(VerdictExample, `"policy"`, `"politics"`)); err == nil {
		t.Error("CheckVerdict accepted an undeclared member")
	}
	if err := CheckPrepassFindings(bad(PrepassFindingsExample, `"home": "intent"`, `"home": "elsewhere"`)); err == nil {
		t.Error("CheckPrepassFindings accepted a decomposition home outside the set")
	}
	if err := CheckConsistencyFindings(t.TempDir(), bad(ConsistencyFindingsExample, `"id": "intent-auditor"`, `"id": " "`)); err == nil {
		t.Error("CheckConsistencyFindings accepted a blank verifier id")
	}
}
