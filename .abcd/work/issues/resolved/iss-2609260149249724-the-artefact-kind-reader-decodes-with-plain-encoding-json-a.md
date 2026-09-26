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
resolution: "The artefact declaration refuses a repeated key at any level through jsonstrict, and reads a lockstep entry's keys exactly, so a case-folded spelling is refused."
impact: fix
resolved_by:
  commit: "bd52f005c"
---

The artefact-kind reader decodes with plain encoding/json: a duplicate top-level key takes the last value ({"kind":"wasm","kind":"binary"} reads as binary) and lockstep object entries decode case-insensitively ({"PATH":..,"JSON_POINTER":..} accepted) while the top level refuses Kind.

## Grounds

- pursued: a declaration with a repeated kind, a repeated entry path or an upper-case entry key is refused before any verb writes; TestParseArtefactRefusesRepeatedAndCaseFoldedKeys accepting any of them would show it wrong
