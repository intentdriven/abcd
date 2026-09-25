package release

import (
	"strings"

	"github.com/intentdriven/abcd/internal/core/lint"
)

// ReceiptsProtocol is the receipts protocol the emit step ends with (itd-93
// AC8, iss-327): what the operator does between the cut and the merge so the
// release job's receipt gate admits the release. The steps are composed here,
// from the release workflow's own required-gate list; the front door numbers
// and renders them.
//
// It exists because the protocol used to live only in the runbook, and a first
// release is exactly when nobody has read the runbook: a one-commit release
// branch reached a tag and failed there, the most expensive place to learn it.
type ReceiptsProtocol struct {
	// Workflow is the release workflow the gate list was read from.
	Workflow string `json:"workflow"`
	// Armed reports that the workflow runs a receipt gate at all.
	Armed bool `json:"armed"`
	// RequiredGates are the semantic gates the release job requires, in the
	// workflow's order. Empty when the workflow arms none.
	RequiredGates []string `json:"required_gates"`
	// Steps are the checklist, in order, unnumbered.
	Steps []string `json:"steps"`
}

// ReceiptsProtocolFor composes the receipts protocol for the repository at
// root, reading which semantic gates to run from its committed release
// workflow — the same list the release job and `abcd launch receipts` read, so
// the checklist can never ask for a gate the release does not require, or omit
// one it does.
func ReceiptsProtocolFor(root string) (ReceiptsProtocol, error) {
	gate, err := lint.ReadReleaseGate(root)
	if err != nil {
		return ReceiptsProtocol{}, err
	}
	p := ReceiptsProtocol{Workflow: gate.Workflow, Armed: gate.Armed, RequiredGates: gate.Gates}

	roll := "Ingest the composed changelog (`abcd launch ship --changelog-json <file>`) and commit the " +
		"result on a release branch. That commit is the content commit: the commit the release publishes " +
		"from and every receipt names."
	switch {
	case !gate.Present:
		p.Steps = []string{
			roll,
			"No `" + gate.Workflow + "` exists, so no receipt gate is armed and no receipt is required. " +
				"`abcd launch scaffold` writes the release workflows; until then nothing tags or publishes the cut.",
		}
	case !gate.Armed:
		p.Steps = []string{
			roll,
			"`" + gate.Workflow + "` requires no semantic gate, so no receipt is required: the deterministic " +
				"gates alone admit the release.",
			"Open the release pull request and merge it once its checks are green; the auto-release workflow " +
				"tags the merged commit and `" + gate.Workflow + "` publishes it.",
		}
	default:
		p.Steps = []string{
			roll,
			"Run each semantic gate `" + gate.Workflow + "` requires against the content commit: " +
				strings.Join(gate.Gates, ", ") + ".",
			"Record each PROMOTE receipt at `.abcd/work/reviews/<content-sha>/<gate>.json`, keyed to the full " +
				"40-character sha of the content commit (`git rev-parse HEAD` right after step 1) — never the tag, " +
				"and never the merge.",
			"Commit the receipts on top. The release branch is exactly two commits: the roll, then the receipts " +
				"naming it. Amending the roll after this gives it a new sha and orphans every receipt.",
			"Run `abcd launch receipts` on the release branch. It runs the release job's receipt gate locally " +
				"and names each missing or non-PROMOTE receipt; merge only when it exits 0.",
		}
	}
	return p, nil
}
