package docfidelity

import "reflect"

// inputs.go — `docs fidelity record --verdict-json` as the input registry in
// the front door names it (spc-2610301431075964 step 1): the Go type, the
// schema version, a worked example and the check Record runs first.

// VerdictSchemaVersion is the reviewer verdict's shape version. The verdict
// carried none before it existed, so the decoder reads an absent
// schema_version as 1 and refuses any other version.
const VerdictSchemaVersion = 1

// VerdictPayloadType is the Go type the verdict decodes into (VerdictShape is
// the one-line shape the reviewer's request carries).
var VerdictPayloadType = reflect.TypeOf(payload{})

// VerdictExample is one reviewer verdict at VerdictSchemaVersion that passes
// Check: a HOLD naming the one brief sentence the review found false.
const VerdictExample = `{
  "schema_version": 1,
  "verificationResult": "HOLD",
  "judgeModel": "example-model-2026-10-01",
  "tier": "full",
  "verifier": {"id": "docs-currency-reviewer"},
  "failing": [
    {"disposition": "confirmed", "doc": "brief", "chapter": "17-guard.md",
     "sentence": "The guard reads its registry from the plugin root.",
     "evidence": "internal/core/guard/registry.go:40 reads it from the repository",
     "replacement": "The guard reads its registry from the repository."}
  ]
}`

// Check runs the whole of the verdict's own validation, the part Record runs
// before it reads anything from the checkout, and writes nothing. Record then
// labels the receipt with the commit and the release-gate manifest it reads.
func Check(raw []byte) error {
	_, err := decodePayload(raw)
	return err
}

// ReportFindings runs Check's validation and hands back the verdict and the
// sentences it names, each chapter read through the recorder's own parse, and
// writes nothing: no receipt, no flag, no edit. It is how a model-matrix check
// copy's verdict is read (decision 30 of itd-2610161733316518): validated as a
// recorded verdict is, kept in the run's own record, and never saved where the
// release gates read.
func ReportFindings(raw []byte) (string, []Sentence, error) {
	p, err := decodePayload(raw)
	if err != nil {
		return "", nil, err
	}
	out := make([]Sentence, 0, len(p.Failing))
	for _, f := range p.Failing {
		out = append(out, Sentence{Doc: docOf(f.Doc), Chapter: f.Chapter, Sentence: f.Sentence,
			Evidence: f.Evidence, Replacement: f.Replacement})
	}
	return p.VerificationResult, out, nil
}
