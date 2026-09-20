---
schema_version: 1
id: "iss-2609190337545165"
slug: "the-ingest-json-for-a-dead-lettered-verdict-carries-a-rollup"
severity: "nitpick"
category: "ux"
source: "agent-observation"
found_during: "Gropius autonomous sweep, session gropiusllm-66, relayed to abcd-17 on 2026-09-19"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/audit.go"
---

The ingest JSON for a dead-lettered verdict carries a rollup that reads as a result. abcd intent audit ingest --verdict-json --json returns one result struct for every outcome, so a dead_letter response carries criteria, conditions and the per-verdict counters beside its reason: a lane in the Gropius sweep read "criteria: 0" beside "conditions: 11" on a quarantined verdict and took it for a rollup. The text render is right (it prints the rollup only under ingested and the DEAD_LETTER reason otherwise), the JSON is not: the counters are zero-valued members with no omitempty, and conditions is populated from the intent before the verdict is judged. Reproducible by reading the IngestResult struct in internal/core/intent/audit.go against the --json path. Relayed from session gropiusllm-66 on 2026-09-19 at v0.9.0. Wanted: the rollup members omitted (or nulled) on dead_letter and noop, so a quarantined verdict's JSON says only what happened and where the raw payload went. Separately, the same session reported the ingest refusing a digest of "sha256:unknown" while accepting an empty digest; that is the documented contract (empty where the digest is not known, otherwise sha256:<64 hex>) and the refusal says so, so it is not filed.
