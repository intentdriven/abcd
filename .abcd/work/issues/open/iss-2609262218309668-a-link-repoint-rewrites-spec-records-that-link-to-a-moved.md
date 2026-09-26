---
schema_version: 1
id: "iss-2609262218309668"
slug: "a-link-repoint-rewrites-spec-records-that-link-to-a-moved"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: fix2-drainL sibling sweep of iss-2609262143209970"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/relink/relink.go"
---

A link repoint rewrites spec records that link to a moved path with no spec lock, and no spec writer takes one either: the spec store's only lock is the mint's (withMintLock in internal/core/spec/store.go). spec.Close renames a spec open/ to closed/ while a concurrent repoint (an intent plan, a capture resolve) that read the spec at open/ writes it back there through an atomic rename, which lands the spec in both status folders; a spec edit landing between the repoint's read and its write is erased. Closing it needs a spec-store writer lock every spec writer takes, ordered against the ledger and intent locks.
