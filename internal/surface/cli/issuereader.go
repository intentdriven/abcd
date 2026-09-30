package cli

import (
	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/site"
)

// init registers the issue ledger's reader and the site renderer's body check
// with the lint for every lint the CLI runs (`abcd lint docs`, `abcd lint`), so a config arming record_schema over an
// issue store gets the reader-parity and body legs the record-lint gate runs,
// and a config arming stale_edge follows supersession chains through the intent
// package's one chain reader.
// It registers record-lint's prose-citation gate the same way, once, for every
// ingest that copies host prose into a record: the intent audit's verdict
// ingest, and the consistency and reading ingests in the ledger, which ask it
// through intent.UnresolvedProseCitations. Each is held to the gate the record
// it writes must pass, and each refuses when no gate is registered.
func init() {
	lint.SetIssueReader(capture.ReadRefusal)
	lint.SetRecordBodyCheck(site.CheckRecordBody)
	lint.SetSupersessionChain(intent.SupersessionChainOf)
	intent.SetProseCitationGate(proseCitationGate)
}

// proseCitationGate is lint.UnresolvedProseCitationsInRecord in the intent
// package's vocabulary.
func proseCitationGate(repoRoot, rel, text string) ([]intent.UnresolvedCitation, error) {
	cites, err := lint.UnresolvedProseCitationsInRecord(repoRoot, rel, text)
	out := make([]intent.UnresolvedCitation, 0, len(cites))
	for _, c := range cites {
		out = append(out, intent.UnresolvedCitation(c))
	}
	return out, err
}
