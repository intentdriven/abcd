---
schema_version: 1
id: "iss-2611032348317844"
slug: "the-guard-s-merge-rule-refuses-a"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "coordinating the build lanes and the walled subscription test, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "On a fetch failure fall back to the last fetched origin/main's facilitator.mode and name it as stale, and never treat a query naming no mutation as a merge; a test times the fetch out and asserts a read-only query passes."
---

The guard's merge rule refuses a read-only GraphQL watch whenever its own fetch of origin's main times out: it then reads facilitator.mode as human and treats any query it cannot read on the line as a possible merge, so on 2026-11-03 a slow network turned a coordinator's queue watcher off, though a plain git fetch with a longer limit succeeded at the same moment.
