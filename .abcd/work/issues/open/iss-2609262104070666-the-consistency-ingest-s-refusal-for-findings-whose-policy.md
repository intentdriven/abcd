---
schema_version: 1
id: "iss-2609262104070666"
slug: "the-consistency-ingest-s-refusal-for-findings-whose-policy"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainA2"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/consistency.go"
---

The consistency ingest's refusal for findings whose policy hashes the request never issued tells the auditor to echo the two values the request's Provenance block states, and never names the re-emit. For a request emitted by an earlier binary (before the findings shape joined the prompt body) the auditor DID echo them, so the remedy it gives cannot succeed: the only way through is to re-emit with abcd intent consistency and run the pass again. The sibling refusal in the fidelity ingest (checkIssuedPolicy) names the re-emit; the two should share one derivation of that wording.
