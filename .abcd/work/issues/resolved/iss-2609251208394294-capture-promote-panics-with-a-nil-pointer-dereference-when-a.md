---
schema_version: 1
id: "iss-2609251208394294"
slug: "capture-promote-panics-with-a-nil-pointer-dereference-when-a"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/promote.go"
resolution: "The orphan-draft remedy names --grounds only when the promotion was given grounds, so a grounds-less promotion whose stamp fails reports its orphan draft and the link remedy instead of panicking on the nil grounds."
impact: fix
resolved_by:
  commit: "14ab2e07"
---

capture promote panics with a nil pointer dereference when a promotion given no --grounds fails its ledger stamp after minting the draft: the orphan-draft remedy in Promote calls g.String() on the optional grounds, which is nil once grounds became optional (iss-2609091009111294), so the failure path that exists to name the orphan and its repair crashes instead and the orphan draft is left unreported. Confirmed with a stamp-failure test while fixing the remedy's delimiter.

## Grounds

- pursued: a failed promotion never crashes composing its remedy; a stamp failure without grounds that panics would show it wrong
