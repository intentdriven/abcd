---
schema_version: 1
id: "iss-2608261533419897"
slug: "scanintenttree-stat-swallow-alignment-with-fail-closed-doctrine"
severity: "nitpick"
category: "observation"
source: "agent-observation"
found_during: "bughunt-a round 9"
found_at: "internal/core/lint/lint.go"
resolution: "scanIntentTree and the two spec-store stat probes treat only ENOENT as absent and return every other stat error, and the intent-id walk no longer discards its errors."
impact: internal
resolved_by:
  commit: "ea6f7a97"
---

scanIntentTree and the two spec-store stat probes in internal/core/lint swallow every stat error as tree-absent, unlike scanIssueLedger and scanRecordStores which part ENOENT from real faults — the doctrine the round-9 ScanSpecLinks fix states as a tree that is present but cannot be read IS a fault. No leg is currently reachable past markdownFiles, os.ReadDir, and the armed delivery_state floor (adjudicated: the claimed vacuous-blocker triggers are all closed one line later or upstream), so this is a consistency alignment, not a live defect: part ENOENT from other errors at the three sites and stop discarding WalkDir errors, matching the sibling scanners. Recorded for a scoped consolidation rather than fixed mid-hunt.

## Grounds

- pursued: an unreadable subtree or an unstattable store is an error from each leg while an absent tree stays soft; a mode-000 directory read as absent would show it wrong
