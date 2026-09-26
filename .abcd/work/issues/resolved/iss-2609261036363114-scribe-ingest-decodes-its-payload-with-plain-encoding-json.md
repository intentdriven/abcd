---
schema_version: 1
id: "iss-2609261036363114"
slug: "scribe-ingest-decodes-its-payload-with-plain-encoding-json"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-scribe"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/scribe/ingest.go"
deferred_after: "v0.11.0"
deferral_reason: "deferred to the integration step (run A, 2026-09-26): the strict duplicate-key decoder jsonstrict lives on the unmerged lintB lane and copying it here would fork it; once lintB lands, decodeOutput in internal/core/scribe/ingest.go reroutes its decode through jsonstrict at every depth, refusing duplicate keys and case-folded twins, and this record is resolved there. Until then the state a duplicate key selects is still held to the item's own line of the supplied dispositions, and every free text to the supplied text verbatim"
resolution: "scribe's decodeStrict runs jsonstrict.NoDuplicateKeys before it decodes, so the scribe output, manifest and context refuse a key repeated at any depth instead of reading it last-wins"
impact: fix
resolved_by:
  commit: "4a51e87238921d20499fe8fb937bbbfb4e2810b4"
---

scribe ingest decodes its payload with plain encoding/json, so a duplicate key at any depth takes the last value: {"state":"rejected","state":"accepted"} decodes as accepted rather than being refused

## Grounds

- pursued: a payload repeating state or run at any depth is refused before anything lands; a repeated key that still decodes last-wins and writes a record would show it wrong
