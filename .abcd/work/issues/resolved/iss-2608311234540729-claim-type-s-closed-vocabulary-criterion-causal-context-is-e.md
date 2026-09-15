---
schema_version: 1
id: "iss-2608311234540729"
slug: "claim-type-s-closed-vocabulary-criterion-causal-context-is-e"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "closedVocabulary enforces claim_type's three tokens at item level."
impact: fix
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

claim_type's closed vocabulary (criterion, causal, context) is enforced nowhere. agents/cold-reading-entailment.md instructs it and spc-63 tables it, but validateItems checks non-blank only, capture.validateReadingStrict has no enum, and issueschema declares only the field name, so an entailment payload carrying an arbitrary claim_type lands in the durable record.

## Grounds

- pursued: the finding is closed by a test that fails without the change; a later review or mutation run finding the same shape again would show this wrong
