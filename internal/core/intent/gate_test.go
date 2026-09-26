package intent

import "github.com/intentdriven/abcd/internal/core/lint"

// init registers record-lint's prose-citation gate for this package's tests,
// as the front doors register it for every ingest they run.
func init() {
	SetProseCitationGate(func(repoRoot, rel, text string) ([]UnresolvedCitation, error) {
		cites, err := lint.UnresolvedProseCitationsInRecord(repoRoot, rel, text)
		out := make([]UnresolvedCitation, 0, len(cites))
		for _, c := range cites {
			out = append(out, UnresolvedCitation(c))
		}
		return out, err
	})
}
