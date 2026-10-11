---
schema_version: 1
id: "iss-2609281204381700"
slug: "the-implement-loop-s-run-state-reader"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-integ8"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/state.go"
resolution: "readStateIn decodes through jsonstrict.Decode, the lane receipt's strict decode: a repeated key, a case twin and a second document are each refused as a state refusal naming why."
impact: fix
resolved_by:
  commit: "838739bfc"
---

The implement loop's run-state reader, readStateIn in internal/core/implement/loop/state.go, decodes state.json with DisallowUnknownFields only: a repeated key is read last-wins (encoding/json binds case twins to one field too) and anything after the first JSON document is ignored, so a file that is not the one the loop wrote can pass as the run's state. state.json lives in the primary checkout's .abcd/.work.local/run/<id>/, the directory an implementer is sent to write into, and the loop reads lane status and each lane's awaited receipt path from it. The lane receipt reader was fixed for the same class (iss-2609262123574454) by routing it through jsonstrict.Decode; readStateIn should take the same one primitive, with a repeated key and a trailing document each refused under test.

## Grounds

- pursued: a state.json carrying a repeated key or a trailing document is refused rather than read; TestReadStateFailsClosed's three new cases would pass against a last-wins or first-document decode and show it wrong
