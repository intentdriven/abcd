package intent

import (
	"encoding/json"
	"strings"
	"testing"
)

// issued_policy_test.go — what an ingest says when a payload echoes policy
// hashes the host did not issue.

// TestStalePolicyRefusalNamesTheReEmit is iss-2609262104070666. A payload
// whose hashes are not the pair the host issues for its receipt is most often
// answering a request an earlier binary wrote: the auditor echoed that
// request's Provenance block faithfully, and this binary composes a different
// prompt body. Telling that auditor to echo the Provenance block names a remedy
// it already followed; the only way through is to re-emit the request and run
// the pass again. Both ingests say so, in one wording, naming the verb as it is
// spelled for that receipt.
func TestStalePolicyRefusalNamesTheReEmit(t *testing.T) {
	for _, c := range []struct{ scope, verb string }{
		{"", "`abcd intent consistency`"},
		{"itd-10", "`abcd intent consistency itd-10`"},
	} {
		t.Run("consistency "+c.verb, func(t *testing.T) {
			root := consistencyRepo(t).Root()
			em, err := EmitConsistency(root, c.scope, ConsistencyEmitOptions{})
			if err != nil {
				t.Fatal(err)
			}
			var m map[string]any
			if err := json.Unmarshal(findingsPayload(t, root, em, contradiction()), &m); err != nil {
				t.Fatal(err)
			}
			m["policy"].(map[string]any)["prompt_hash"] = "sha256:" + strings.Repeat("b", 64)
			payload, _ := json.Marshal(m)
			_, err = ingest(t, root, payload, &fakeFiler{})
			assertReEmitRemedy(t, err, c.verb)
		})
	}
	t.Run("fidelity", func(t *testing.T) {
		root := t.TempDir()
		rcp := shipOne(t, root)
		_, err := IngestVerdict(root, writeVerdictRaw(t, root, validVerdict(rcp)))
		assertReEmitRemedy(t, err, "`abcd intent audit itd-10`")
	})
}

func assertReEmitRemedy(t *testing.T, err error, verb string) {
	t.Helper()
	if err == nil {
		t.Fatal("ingest accepted policy hashes the host never issued")
	}
	for _, want := range []string{"never issued", "Re-emit with " + verb, "run the pass again"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal does not say %q, so a stale request's auditor is told to repeat what it already did:\n%v", want, err)
		}
	}
}
