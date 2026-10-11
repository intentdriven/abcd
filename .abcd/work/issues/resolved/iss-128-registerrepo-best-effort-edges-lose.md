---
schema_version: 1
id: "iss-128"
slug: "registerrepo-best-effort-edges-lose"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "iss-101/102 reviews (2026-07-24 run queue, burst 3)"
found_at: "internal/core/ahoy/apply.go"
resolution: "registerRepo surfaces a lock-contention failure as a change-note and the refresh branch re-applies a pending lineage link without overwriting the winner's"
impact: fix
resolved_by:
  commit: "c88c94b4"
---

registerRepo's best-effort history paths lose signal silently in two edges the iss-101/102 reviews confirmed: (1) a history-lock contention timeout skips registration with no surfaced note (apply.go registerRepo discards the withHistoryLock error); (2) under concurrent double-install of the same new repo with divergent re-founding answers, the lock loser takes the refresh branch which never consults linkLineage, silently dropping a human-approved lineage link with no lineageConflict message