---
schema_version: 1
id: "iss-2609022002241168"
slug: "agent-supplied-verdict-and-refusal-fields-reach-committed-re"
severity: "major"
category: "security"
source: "impl-review"
found_during: "cold-reading Phase A, pre-PR review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/audit.go"
resolution: "Both sites are closed by the same rule: a field is a validated shape only where a validator says so, and everything else is free text and is redacted before it is neutralised. In intent/audit.go sha256FieldRe now validates policy.rubric_hash, policy.prompt_hash and a present input_attestations[].digest as sha256:<64 lowercase hex> in validateVerdict — a payload that fails it is quarantined — while verifier.id, verifier.version and an attestation's kind and ref go through the same proseField redaction the rationales use, via a new orFree helper. In reading, a new payloadField built once per ingest (internal/core/reading/redact.go) redacts then neutralises every payload-derived string bound for run.json or refusal.json: the refused item's criterion and candidate id, the unknown and reserved field names, the out-of-vocabulary value, the item-shape decode error, the regime and instrument mismatches, and the recorded instrument identity. Five tests hold it, and the resolved record of iss-2608300924205748 carries a correction saying what its resolution claimed and what was actually true."
impact: fix
---

Agent-supplied verdict and refusal fields reach committed records unredacted, in two sibling ingests. In internal/core/intent/audit.go the ingested block renders verifier.id, verifier.version, policy.rubric_hash, policy.prompt_hash and every input_attestations[] kind, ref and digest through orDash/oneLine alone: no privacy redaction and no shape validation, though iss-2608300924205748's resolution claims identifiers, enum values and hashes are validated shapes. Nothing validates them; a real attestation ref is free prose (a commit range with a parenthetical), so an absolute home path in a ref or a hostname in a verifier id lands in the shipped intent's Audit Notes verbatim. In internal/core/reading/ingest_regime.go a refused comparative item's free-text criterion is echoed into RunRecord.RefusedItems and lands in the durable run.json and refusal.json unredacted, while an ACCEPTED item's body is redacted on write by the same ingest. Expected: every payload-derived string bound for a durable record is redacted before it is neutralised, and what has a declared shape is validated instead.

## Grounds

- pursued: every agent-supplied string that reaches a committed record is privacy-redacted at the moment it is written, in both ingests, and the fields excused from redaction are exactly those a validator constrains. What would show it wrong: a home path, hostname or person name surviving into a shipped intent's Audit Notes or into a reading run's run.json or refusal.json, or a validated field corrupted by the redactor.
