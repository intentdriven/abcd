package capture

import (
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/lint"
)

// init registers record-lint's prose-citation gate for this package's tests,
// as the front doors register it for every ingest they run.
func init() {
	intent.SetProseCitationGate(lintProseGate)
}

// lintProseGate is lint.UnresolvedProseCitationsInRecord in the intent
// package's vocabulary, named so a test that unregisters it can put it back.
func lintProseGate(repoRoot, rel, text string) ([]intent.UnresolvedCitation, error) {
	cites, err := lint.UnresolvedProseCitationsInRecord(repoRoot, rel, text)
	out := make([]intent.UnresolvedCitation, 0, len(cites))
	for _, c := range cites {
		out = append(out, intent.UnresolvedCitation(c))
	}
	return out, err
}
