---
schema_version: 1
id: "iss-2609260908572219"
slug: "the-kept-original-is-written-by-path-after-the-store-lock-lapses"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory/ingest.go"
---

Ingest --keep-original writes the kept original by path after every vetting has lapsed: storeOriginal Lstats the leaf sources directory and calls fsutil.WriteFileAtomic on <repo>/.abcd/memory/sources/<hash><ext> after WritePages has released the store lock and its validatedMemoryDir walk, and the store handle Ingest opened is not used for it. The Lstat guard binds the leaf, not an ancestor, so a .abcd/memory swapped for a directory symlink between WritePages and storeOriginal lands redacted source material outside the repository while Ingest reports status=ingested with kept naming the in-repo path the file is not at. Neither the store-handle record iss-2608291814572914 (reads, plus the locked writer as the residual) nor iss-2609252100150846 (the coverage index) covers it. Fix: after WritePages, open a store handle and write sources/<hash><ext> through fsutil.WriteFileAtomicInRoot, keeping the symlink refusal as root.Lstat("sources").
