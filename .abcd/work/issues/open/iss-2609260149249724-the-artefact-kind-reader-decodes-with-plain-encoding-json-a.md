---
schema_version: 1
id: "iss-2609260149249724"
slug: "the-artefact-kind-reader-decodes-with-plain-encoding-json-a"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25 (fix round, review of lane launchkind)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/artefact.go"
deferred_after: "v0.10.0"
deferral_reason: "deferred to the integration step (run A, 2026-09-26): the strict duplicate-key decoder jsonstrict lives on the unmerged lintB lane and copying it here would fork it; once lintB lands, the artefact-kind reader (top level and lockstep entries) reroutes its decode through jsonstrict, refusing duplicate keys and case-folded field names, and this record is resolved there"
---

The artefact-kind reader decodes with plain encoding/json: a duplicate top-level key takes the last value ({"kind":"wasm","kind":"binary"} reads as binary) and lockstep object entries decode case-insensitively ({"PATH":..,"JSON_POINTER":..} accepted) while the top level refuses Kind.
