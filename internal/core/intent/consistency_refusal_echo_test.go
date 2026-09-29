package intent

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestConsistencyRefusalsDoNotEchoThePayload — iss-2609290144116254. Every
// consistency refusal returns to the surface with nothing written, and six of
// them carried the payload's own value back: the _type with a bare %q, a policy
// hash, a finding's class and severity, and an end's path and quote through
// oneLine, which cleans and caps but does not redact. A token or a home path in
// any of them reached the terminal and the transcript. Each is described now,
// and the refusal still names the field and where it sits; the one refusal whose
// value the reader needs — an undeclared field's NAME — is redacted through the
// canonical scanner and still named.
func TestConsistencyRefusalsDoNotEchoThePayload(t *testing.T) {
	const leak = "zzleak-7f3a /Users/zzotherperson/notes long enough to quote" // abcd-lint:allow — a planted home path the refusal must not echo
	finding := func(m map[string]any) map[string]any { return m["findings"].([]any)[0].(map[string]any) }
	end := func(m map[string]any) map[string]any { return finding(m)["ends"].([]any)[1].(map[string]any) }
	cases := []struct {
		name   string
		mutate func(m map[string]any)
		names  string
	}{
		{"_type", func(m map[string]any) { m["_type"] = leak }, "_type"},
		{"rubric_hash", func(m map[string]any) { m["policy"].(map[string]any)["rubric_hash"] = leak }, "policy.rubric_hash"},
		{"class", func(m map[string]any) { finding(m)["class"] = leak }, "class"},
		{"severity", func(m map[string]any) { finding(m)["severity"] = leak }, "severity"},
		{"end path", func(m map[string]any) { end(m)["path"] = leak }, "path"},
		{"end quote", func(m map[string]any) { end(m)["quote"] = leak }, "quote"},
		{"undeclared field", func(m map[string]any) { m["reviewer_notes /Users/zzotherperson/notes"] = "x" }, "reviewer_notes"}, // abcd-lint:allow — a planted home path in a KEY
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := consistencyRepo(t)
			root := r.Root()
			em, err := EmitConsistency(root, "", ConsistencyEmitOptions{})
			if err != nil {
				t.Fatal(err)
			}
			var m map[string]any
			if err := json.Unmarshal(findingsPayload(t, root, em, contradiction()), &m); err != nil {
				t.Fatal(err)
			}
			tc.mutate(m)
			payload, _ := json.Marshal(m)
			_, err = ingest(t, root, payload, &fakeFiler{})
			if err == nil {
				t.Fatalf("a payload carrying the leak in %s was accepted", tc.name)
			}
			for _, part := range []string{"zzleak-7f3a", "zzotherperson"} {
				if strings.Contains(err.Error(), part) {
					t.Errorf("the refusal echoes the refused %s: %v", tc.name, err)
				}
			}
			if !strings.Contains(err.Error(), tc.names) {
				t.Errorf("the refusal no longer names %s: %v", tc.names, err)
			}
		})
	}
}
