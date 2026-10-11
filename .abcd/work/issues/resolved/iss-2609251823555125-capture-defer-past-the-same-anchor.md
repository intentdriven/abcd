---
schema_version: 1
id: "iss-2609251823555125"
slug: "capture-defer-past-the-same-anchor"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
resolution: "capture defer past the same anchor rewrites that cycle's Deferral section in place instead of appending a second; a later anchor still appends"
impact: fix
resolved_by:
  commit: "543100249"
---

capture defer past the SAME anchor appends a second ## Deferral section (internal/core/capture/deferral.go:114), where the doc says one per cycle (review-capture 3).

## Grounds

- pursued: a record deferred twice past one anchor carries one Deferral section naming the latest reason, and a later anchor adds a second; TestReDeferringPastTheSameAnchorKeepsOneSection failing would show it wrong
