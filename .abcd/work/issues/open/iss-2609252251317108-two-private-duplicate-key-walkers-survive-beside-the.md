---
schema_version: 1
id: "iss-2609252251317108"
slug: "two-private-duplicate-key-walkers-survive-beside-the"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/scope.go"
---

Two private duplicate-key walkers survive beside the canonical jsonstrict check: layered.refuseDuplicateKeys (internal/core/layered/layered.go) and reading.refuseDuplicateKeys (internal/core/reading/scope.go). Neither folds key case the way encoding/json binds struct fields, and the reading-preset walker checks only the keys directly under the presets/positions container, so a key repeated inside one entry (a second kinds or window block under a reviewed position) is read last-wins by the strict decoder: the review-evasion vector the walker's own comment names. One primitive, rerouted, closes both.
