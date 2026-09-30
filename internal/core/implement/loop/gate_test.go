package loop

import (
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/lint"
)

// init registers record-lint's prose-citation gate for this package's tests,
// as the front doors register it for every ingest they run: the landing
// ingests the closing lane's audit verdict.
func init() {
	intent.SetProseCitationGate(func(repoRoot, rel, text string) ([]intent.UnresolvedCitation, error) {
		cites, err := lint.UnresolvedProseCitationsInRecord(repoRoot, rel, text)
		out := make([]intent.UnresolvedCitation, 0, len(cites))
		for _, c := range cites {
			out = append(out, intent.UnresolvedCitation(c))
		}
		return out, err
	})
}
