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
---

capture promote panics with a nil pointer dereference when a promotion given no --grounds fails its ledger stamp after minting the draft: the orphan-draft remedy in Promote calls g.String() on the optional grounds, which is nil once grounds became optional (iss-2609091009111294), so the failure path that exists to name the orphan and its repair crashes instead and the orphan draft is left unreported. Confirmed with a stamp-failure test while fixing the remedy's delimiter.
