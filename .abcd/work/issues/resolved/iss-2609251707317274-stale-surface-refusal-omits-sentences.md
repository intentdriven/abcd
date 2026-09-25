---
schema_version: 1
id: "iss-2609251707317274"
slug: "stale-surface-refusal-omits-sentences"
severity: "nitpick"
category: "ux"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/changelog/guard.go"
resolution: "The stale-surface refusal and the drift test name each command whose sentence differs from HEAD's snapshot, as they name a moved verb."
impact: fix
resolved_by:
  commit: "40081a4a"
---

The stale-surface refusal names each verb whose help placement moved but not a reworded sentence: a snapshot differing from HEAD's only in a sentence yields the generic does-not-match line with nothing to act on

## Grounds

- pursued: a sentence-only snapshot difference is named in the refusal; shown wrong by TestGuardSurfaceNamesARewordedSentence failing
