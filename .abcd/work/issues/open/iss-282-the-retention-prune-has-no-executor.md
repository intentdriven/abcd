---
schema_version: 1
id: "iss-282"
slug: "the-retention-prune-has-no-executor"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "manual-capture"
found_at: "internal/core/launch/ship.go"
related_intents: [itd-70]
deferred_after: "v0.11.1"
deferral_reason: "The product thinker's ruling J21 of 2026-09-29: automate the release retention (keep the newest release per version line) and plan itd-70. Linked to draft itd-70. Owed: that draft's planning interview (the draft stays in drafts/ until a person plans it)."
---

The retention prune has no executor: ComputeRetention renders the newest-per-line plan on every cut (brief section 3) and ship.go names GitHub Release + retention prune as a later phase, so superseded patches (v0.5.0, v0.4.0 today) stay published until someone deletes them by hand from the plan. First rung: a documented manual prune after each publish (gh release delete, tags kept for record); the executor belongs in the release workflow after that proves out