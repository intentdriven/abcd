package cli

import (
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/pendingref"
)

// init registers record-lint's pending-reference lookup for every lint the CLI
// runs (`abcd lint`) and every host-prose ingest that asks the prose-citation
// gate before it writes (iss-2611031731385043), the registration
// cmd/record-lint makes for the gate: an id naming a record on an open pull
// request's branch on origin is a pending reference, reported and not refused,
// so a writer accepts what the gate accepts rather than pushing its author to
// reword the citation.
func init() {
	lint.SetPendingLookup(pendingref.ForLint)
}
