---
schema_version: 1
id: "iss-2608300257424138"
slug: "reading-paths-lock-scope-and-symlink-follow"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "itd-180 second-round reviews, 2026-08-30"
found_at: "internal/core/capture/reading.go (IngestReading, readingItemPaths), internal/core/capture/promote.go (promoteReadingItem)"
resolution: "Redaction moved outside the ledger lock with one scanner per batch (50 items now build one, none while locked); promote recomputes the standing state inside its locked closure and refuses anything but accepted there; and the readings root, every run directory and an item's own disposition directory are Lstat-refused when symlinked — the dispositions FAMILY root and the record files below it were left unguarded until iss-2608300326346554 and iss-2608300320011217."
impact: internal
---

IngestReading redacts every free-text field inside the ledger lock, constructing a fresh scanner per field (each shells out to git config), so a 100-item batch holds the lock about 8 seconds against a 5-second lockTimeout and any concurrent capture, disposition or promote fails with allocator contention; the issue-capture path redacts before taking the lock. Also, promoteReadingItem reads the standing disposition state outside the lock and re-checks only promoted_to inside it, so a disposition landing between preflight and stamp leaves a standing rejected beside a promoted_to; and the readings tree is read through symlinks on the disposition and promote paths (only ingest calls ensureFamilyDir), so a symlinked readings root lets promote write its stamp through the link to a file outside the ledger.
