---
schema_version: 1
id: "iss-2609261036354146"
slug: "scribe-ingest-authenticates-nothing-about-the-parked-context"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-scribe"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/scribe/ingest.go"
resolution: "scribe ingest requires --dispositions, the researcher's file assemble was handed; the manifest's supplied hash and the context's supplied copy must both equal it, and every authoring check reads it, so a rewritten parked pair is refused"
impact: internal
resolved_by:
  commit: "21dce529"
---

scribe ingest authenticates nothing about the parked context and manifest pair: a scribe that rewrites context.json's supplied dispositions and recomputes the manifest's context hash passes the verbatim check against text it wrote itself, while manifest.supplied.dispositions_sha256 is recorded and never read

## Grounds

- pursued: a parked context and manifest rewritten together to carry a scribe-authored ground are refused before anything is written; a rewritten pair that lands a record would show it wrong
