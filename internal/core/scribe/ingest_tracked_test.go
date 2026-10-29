package scribe

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestScribeIngestRefusesAParkedSessionACommitCarries: a parked session is the
// local tier's, so one git tracks came with a commit, not from this checkout's
// assembly, and nothing is ingested against it.
func TestScribeIngestRefusesAParkedSessionACommitCarries(t *testing.T) {
	s := assembleSession(t, positionDetection, 1, "{0}: accepted — "+groundA+".\n")
	for _, args := range [][]string{{"init", "-q"}, {"add", "-f", "--", DefaultRunDir + "/" + fixtureRun}} {
		cmd := exec.Command("git", append([]string{"-C", s.repo}, args...)...)
		cmd.Env = gittest.Env(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	before := s.ledger(t)
	raw := `{"_type":"` + OutputType + `","context_sha256":"` + s.res.ContextSHA256 + `","run":"` + fixtureRun +
		`","dispositions":[{"item":"` + s.items[0] + `","state":"accepted","grounds":"` + groundA + `"}]}`
	if _, err := s.ingest(t, s.writeRaw(t, raw)); err == nil || !strings.Contains(err.Error(), "git tracks") {
		t.Fatalf("ingest against a parked session git tracks = %v; want it refused as tracked", err)
	}
	if s.ledger(t) != before {
		t.Fatal("a refused ingest changed the ledger")
	}
}
