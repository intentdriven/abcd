package intent

import (
	"fmt"
	"reflect"
)

// inputs.go — the three filled-in inputs this package reads, as the input
// registry in the front door names them (spc-2610301431075964 step 1): each
// input's Go type, its schema version, a worked example held beside the type,
// and the check its writing command runs, here run without writing.
//
// None of the three carried a schema version before. Each now has one, at 1:
// the decoder reads an absent schema_version as 1, so every caller that works
// today keeps working, and it refuses any other version.

// The schema versions of the three inputs.
const (
	// VerdictSchemaVersion is `intent audit ingest --verdict-json`'s.
	VerdictSchemaVersion = 1
	// PrepassFindingsSchemaVersion is `intent prepass --findings-json`'s.
	PrepassFindingsSchemaVersion = 1
	// ConsistencySchemaVersion is `intent consistency ingest --findings-json`'s.
	ConsistencySchemaVersion = 1
)

// The Go types the three inputs decode into, for the registry's field lists.
var (
	VerdictShape             = reflect.TypeOf(verdict{})
	PrepassFindingsShape     = reflect.TypeOf(prepassReturn{})
	ConsistencyFindingsShape = reflect.TypeOf(consistencyPayload{})
)

// unversionedSchemaGate is the version check of an input that once carried
// no version: absent (zero) reads as want, want is read, and anything else is
// refused, naming the version this binary reads.
func unversionedSchemaGate(what string, got, want int) error {
	if got == 0 || got == want {
		return nil
	}
	if got > want {
		return fmt.Errorf("%s: schema_version %d is newer than this abcd reads (%d); update abcd, or compose it at version %d", what, got, want, want)
	}
	return fmt.Errorf("%s: schema_version %d is not a version this abcd reads; compose it at version %d (an absent schema_version reads as %d)", what, got, want, want)
}

// VerdictExample is one intent-audit verdict at VerdictSchemaVersion that
// passes CheckVerdict.
const VerdictExample = `{
  "_type": "abcd/intent-fidelity-verdict/v1",
  "schema_version": 1,
  "receipt_id": "rcp-0123456789ab",
  "verifier": {"id": "intent-auditor", "version": "example-model-1"},
  "policy": {
    "rubric_hash": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
    "prompt_hash": "sha256:1111111111111111111111111111111111111111111111111111111111111111"
  },
  "input_attestations": [{"kind": "diff", "ref": "main..build/example", "digest": ""}],
  "criteria": [
    {"criterion_id": "ac-1", "verdict": "MET",
     "rationale": "the plan verb mints the spec and writes the link both ways",
     "evidence": [{"ref": "internal/core/spec/store.go:88", "quote": "func Create("}]}
  ],
  "acceptance_rollup": {"MET": 1, "MET_WITH_CONCERNS": 0, "NOT_MET": 0, "INCONCLUSIVE": 0},
  "gap_audit": {
    "honoured": [{"claim": "the spec is linked from the intent", "evidence": [{"ref": "internal/core/spec/store.go:90", "quote": "spec_id"}]}],
    "diverged": [],
    "missing": []
  },
  "scope_conditions": []
}`

// CheckVerdict runs the part of the ingest's validator that needs no record:
// the strict decode, the schema version, the _type, a receipt id of the
// receipt's shape, the attestation chain's shape and a non-empty criteria
// list. What it cannot judge without the intent (each criterion against its
// Acceptance Criteria, the rollup, the gap audit's evidence and the scope
// conditions) the ingest judges against the record the receipt resolves to.
func CheckVerdict(raw []byte) error {
	v, err := decodeVerdict(raw)
	if err != nil {
		return err
	}
	if !rcpIDRe.MatchString(v.ReceiptID) {
		return fmt.Errorf("intent: verdict has no resolvable receipt_id (malformed or absent)")
	}
	return nil
}

// PrepassFindingsExample is one set of pre-pass findings at
// PrepassFindingsSchemaVersion that passes CheckPrepassFindings.
const PrepassFindingsExample = `{
  "_type": "abcd/intent-prepass-findings/v1",
  "schema_version": 1,
  "intent": "itd-2609010000000001",
  "input_digest": "sha256:2222222222222222222222222222222222222222222222222222222222222222",
  "summary": "the draft adds one verb and reads one new file",
  "decomposition": [{"part": "the verb and its refusals", "home": "intent"}],
  "conflicts": [
    {"invariant": 1, "principle": "", "anchor_quote": "internal/core never writes to stdout",
     "draft_quote": "the core prints the summary", "question": "Should the front door print the summary instead?"}
  ],
  "overlaps": [
    {"sibling": "itd-2609010000000002", "question": "Is this the same verb as the sibling's?",
     "recommendation": {"answer": "keep-both", "reason": "the sibling reads, this one writes"}}
  ],
  "unanchored": [{"question": "Who reads the new file?"}],
  "blocks_planning": []
}`

// CheckPrepassFindings runs the pre-pass's own decoder and validator on a
// findings file, against no input: the strict decode, the schema version, the
// _type, and every item's shape. A conflict or an overlap is anchored against
// the input the pre-pass assembles when the brief is written, and the intent
// and input digest are compared with it then; here no anchor is resolved, so
// none is refused for not resolving.
func CheckPrepassFindings(raw []byte) error {
	f, err := decodePrepassFindings(raw)
	if err != nil {
		return err
	}
	if f.Type != PrepassFindingsType {
		return fmt.Errorf("intent prepass: the findings file's _type is %q, want %q", f.Type, PrepassFindingsType)
	}
	_, _, err = validatePrepass(PrepassInput{}, f)
	return err
}

// ConsistencyFindingsExample is one consistency payload at
// ConsistencySchemaVersion that passes CheckConsistencyFindings.
const ConsistencyFindingsExample = `{
  "_type": "abcd/intent-consistency-findings/v1",
  "schema_version": 1,
  "receipt_id": "rcp-0123456789ab",
  "verifier": {"id": "intent-auditor", "version": "example-model-1"},
  "policy": {
    "rubric_hash": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
    "prompt_hash": "sha256:1111111111111111111111111111111111111111111111111111111111111111"
  },
  "findings": [
    {"class": "premise_contradiction", "severity": "major",
     "summary": "the brief says the core prints; the intent says it never does",
     "explanation": "both sentences describe the same verb's output, and only one can hold",
     "ends": [
       {"path": ".abcd/development/intents/planned/itd-2609010000000001-example.md", "quote": "the core prints the summary"},
       {"path": ".abcd/development/brief/04-surfaces/05-intent.md", "quote": "internal/core never writes to stdout"}
     ]}
  ]
}`

// CheckConsistencyFindings runs the part of the ingest's validator that needs
// no issued request: the strict decode, the schema version, the _type, a
// receipt id of the receipt's shape, the policy hashes' shape, a verifier and
// the findings list. The policy's match with what was issued, and each
// finding's ends located in the corpus, are judged by the ingest against the
// request the receipt names.
func CheckConsistencyFindings(repoRoot string, raw []byte) error {
	p, err := decodeConsistencyPayload(repoRoot, raw)
	if err != nil {
		return err
	}
	if p.Type != ConsistencyType {
		return fmt.Errorf("intent: findings _type is %q, which is not %q; refusing to ingest", p.Type, ConsistencyType)
	}
	if !rcpIDRe.MatchString(p.ReceiptID) {
		return fmt.Errorf("intent: findings carry no resolvable receipt_id (malformed or absent); refusing to ingest")
	}
	return checkConsistencyBody(p)
}
