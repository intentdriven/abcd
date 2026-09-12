package cli

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TestJSONCollectionsAreEmptyArraysNotNull pins that a --json collection is an
// empty list, never bare `null`, on an empty store — the invariant history
// list's fix stated class-wide. A consumer that iterates the value (jq `.[]`, an
// agent following the command doc) errors on null; an empty array is safe.
func TestJSONCollectionsAreEmptyArraysNotNull(t *testing.T) {
	_ = captureLedgerRepo(t)

	cases := []struct {
		name   string
		args   []string
		fields []string
	}{
		{"capture", []string{"capture", "--json"}, []string{"recent_open"}},
		{"capture-list", []string{"capture", "list", "--open", "--json"}, []string{"issues", "skipped"}},
		{"spec", []string{"spec", "--json"}, []string{"specs"}},
		{"intent", []string{"intent", "--json"}, []string{"linked"}},
		{"memory", []string{"memory", "--json"}, []string{"by_class", "contradictions", "drift"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := Run(tc.args, &stdout, &stderr); code != 0 {
				t.Fatalf("%v exit=%d stderr=%s", tc.args, code, stderr.String())
			}
			var obj map[string]json.RawMessage
			if err := json.Unmarshal(stdout.Bytes(), &obj); err != nil {
				t.Fatalf("%v output not a JSON object: %v\n%s", tc.args, err, stdout.String())
			}
			for _, f := range tc.fields {
				raw, ok := obj[f]
				if !ok {
					t.Errorf("%v: field %q absent from envelope", tc.args, f)
					continue
				}
				if string(raw) == "null" {
					t.Errorf("%v: field %q rendered as null, want [] (a collection is never null)", tc.args, f)
				}
			}
		})
	}
}
