---
schema_version: 1
id: "iss-2609261941039204"
slug: "capture-migrate-writes-intents-outside-the-mint-lock"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: fix2-drainA1 sweep"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/migrate.go"
---

capture migrate --apply rewrites intent records (the related_issues back-edge, any bucket including shipped/) under the ledger lock only, not the intent mint lock (internal/core/capture/migrate.go, Migrate): an intent writer holding withIntentMintLock (a hold, a condition disposition, a verdict ingest, a review emit, a related-issue edge) landing on the same record between migrate's scan and its write is erased. The intent package exports no seam for another package to take its lock, the same gap iss-2609261254247117 names for relink.Repoint.
