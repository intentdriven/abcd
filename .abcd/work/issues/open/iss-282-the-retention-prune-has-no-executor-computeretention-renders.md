---
schema_version: 1
id: "iss-282"
slug: "the-retention-prune-has-no-executor-computeretention-renders"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "manual-capture"
found_at: "internal/core/launch/ship.go"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Plan the retention prune executor (itd-70), or keep the manual first rung as protocol?"
---

The retention prune has no executor: ComputeRetention renders the newest-per-line plan on every cut (brief section 3) and ship.go names GitHub Release + retention prune as a later phase, so superseded patches (v0.5.0, v0.4.0 today) stay published until someone deletes them by hand from the plan. First rung: a documented manual prune after each publish (gh release delete, tags kept for record); the executor belongs in the release workflow after that proves out