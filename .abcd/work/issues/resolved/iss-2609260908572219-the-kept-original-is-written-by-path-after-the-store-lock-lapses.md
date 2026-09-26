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
resolution: "Ingest writes the kept original through a store handle it opens after WritePages returns: the open re-runs the segment walk, root.Lstat refuses a symlinked sources/, and fsutil.WriteFileAtomicInRoot writes sources/<hash><ext> inside the directory the handle opened, on the full ingest and the registry-hit fast path alike. The kept path is reported only while the store path still names that directory."
impact: fix
resolved_by:
  commit: "27e45c54"
---

Ingest --keep-original writes the kept original by path after every vetting has lapsed: storeOriginal Lstats the leaf sources directory and calls fsutil.WriteFileAtomic on <repo>/.abcd/memory/sources/<hash><ext> after WritePages has released the store lock and its validatedMemoryDir walk, and the store handle Ingest opened is not used for it. The Lstat guard binds the leaf, not an ancestor, so a .abcd/memory swapped for a directory symlink between WritePages and storeOriginal lands redacted source material outside the repository while Ingest reports status=ingested with kept naming the in-repo path the file is not at. Neither the store-handle record iss-2608291814572914 (reads, plus the locked writer as the residual) nor iss-2609252100150846 (the coverage index) covers it. Fix: after WritePages, open a store handle and write sources/<hash><ext> through fsutil.WriteFileAtomicInRoot, keeping the symlink refusal as root.Lstat("sources").

## Grounds

- pursued: a store swapped for a symlink between WritePages and the kept-original write, or after its handle opened, puts nothing outside the repository and reports no kept path, pinned by TestKeepOriginalIsNotWrittenThroughASwappedStore, TestKeepOriginalOnTheFastPathIsNotWrittenThroughASwappedStore and TestKeepOriginalRefusesAStoreSwappedAfterItsHandleOpened; a sources/ write by path reappearing in ingest.go, or those tests passing with it, would show it wrong
