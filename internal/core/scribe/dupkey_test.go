package scribe

import (
	"strings"
	"testing"
)

// TestScribeIngestRefusesARepeatedKey: a key the payload repeats, at any depth,
// is refused rather than read last-wins, so {"state":"declined","state":"accepted"}
// is not an accepted disposition, and nothing lands. The decode goes through
// jsonstrict, the one strict-JSON check every trust-boundary reader shares
// (iss-2609261036363114).
func TestScribeIngestRefusesARepeatedKey(t *testing.T) {
	for _, tc := range []struct{ name, run, state string }{
		{"a repeated state on a disposition", `"run":"` + fixtureRun + `"`, `"state":"declined","state":"accepted"`},
		{"a repeated run at the top", `"run":"rdg-1","run":"` + fixtureRun + `"`, `"state":"accepted"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// A session of its own each, so a case read last-wins cannot make the
			// next one fail for another reason.
			s := assembleSession(t, positionDetection, 1, "{0}: accepted — "+groundA+".\n")
			before := s.ledger(t)
			raw := `{"_type":"` + OutputType + `","context_sha256":"` + s.res.ContextSHA256 + `",` + tc.run +
				`,"dispositions":[{"item":"` + s.items[0] + `",` + tc.state + `,"grounds":"` + groundA + `"}]}`
			_, err := s.ingest(t, s.writeRaw(t, raw))
			if err == nil || !strings.Contains(err.Error(), "duplicate key") {
				t.Fatalf("a repeated key was read last-wins rather than refused: %v", err)
			}
			if s.ledger(t) != before {
				t.Fatal("a refused payload changed the ledger")
			}
		})
	}
}
