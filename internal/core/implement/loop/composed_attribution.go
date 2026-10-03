package loop

// composed_attribution.go is the trailer of the commits the loop composes from
// record facts (ruling PC1, the technical facilitator, 2026-10-02): the pick's
// record-only commit (pickcommit.go) and the sync's merge commit (sync.go).
// Their text is computed by abcd from the run's state and records, so no model
// wrote it and `Assisted-by: None` (no tool touched it) would be a false
// disclosure. They carry the third form the attribution convention accepts,
// `Assisted-by: abcd:<version>`, beside a model's `<Vendor>:<model-version>`
// and None.
//
// The landing's records commit (land.go) is not one of them: it carries prose
// a model composed, so it names the models the lane's receipts reported, and a
// receipt that names abcd as its model is refused there rather than copied.
// The landing's pull-request body (prBody, land.go) carries both: abcd composed
// its text, so it declares the label, and the change it describes carries the
// models' work, so the label is followed by their lines (ruling R4).

import (
	"github.com/intentdriven/abcd/internal/core"
	"github.com/intentdriven/abcd/internal/core/assistedby"
)

// composedAssistedBy is the trailer line of a commit the loop composes from
// record facts, naming the running binary's version (what `abcd version`
// prints). The label's grammar is internal/core/assistedby's, the one Go home
// the site's authorship tally reads too.
func composedAssistedBy() string {
	return "Assisted-by: " + assistedby.ComposedValue(core.Version)
}
