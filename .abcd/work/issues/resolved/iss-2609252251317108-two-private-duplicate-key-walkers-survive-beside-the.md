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
resolution: "layered.refuseDuplicateKeys and reading.refuseDuplicateKeys both call jsonstrict.NoDuplicateKeys and phrase their refusal from its DuplicateKeyError: every depth of a preset file is checked now, and a case twin counts in both."
impact: fix
resolved_by:
  commit: "63bda925"
---

Two private duplicate-key walkers survive beside the canonical jsonstrict check: layered.refuseDuplicateKeys (internal/core/layered/layered.go) and reading.refuseDuplicateKeys (internal/core/reading/scope.go). Neither folds key case the way encoding/json binds struct fields, and the reading-preset walker checks only the keys directly under the presets/positions container, so a key repeated inside one entry (a second kinds or window block under a reviewed position) is read last-wins by the strict decoder: the review-evasion vector the walker's own comment names. One primitive, rerouted, closes both.

## Grounds

- pursued: one duplicate-key primitive serves all four JSON trust boundaries, so a key repeated at any depth or in any case spelling of a preset or layered config file is refused; a key repeated inside a preset entry loading (TestKeysRepeatedInsideAPositionEntryAreRefused) would show it wrong
