---
schema_version: 1
id: "iss-2608291814572914"
slug: "memory-store-containment-is-per-verb-not-structural"
severity: "minor"
category: "architectural-insight"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/core/memory/writer.go"
resolution: "Every memory read (Bare and its headroom, QueryPages, the Ingest dedup and registry load, fileBack, the Lint crawl, residue and quotation checks, and the coverage crawl with its budget and stored fingerprint) goes through one os.Root store handle opened inside the repository root, and one Lint holds one handle for both passes; fileBack opens it before reading. The crawls moved in 2475b570; the config, registry and fingerprint reads that commit left by path moved in a8652c12. The coverage-index write goes through the handle too (iss-2609252100150846), and so does the --keep-original write of sources/<hash>, through a handle Ingest opens after WritePages returns, since the lock and its walk lapse there (iss-2609260908572219). The locked writer keeps validatedMemoryDir and writes by path, a residual behind the store lock; it is the only write into the store by path."
impact: fix
resolved_by:
  commit: "a8652c12"
---

ultra-v0.6.8 altitude 3: the memory store's symlink guard is a per-verb pre-check repeated at five entry points (Bare, QueryPages, Ingest, Lint, runMemoryCoverageLint) rather than a containment mechanism, and fileBack in ask.go reached Dir(root) and existingPageFrontmatter without it. The site package fixed the identical class (gh #487) by opening one os.Root and routing every read through fsutil.ReadGuardedInRoot. Deeper fix: memory holds a store-root handle the same way so containment is structural rather than remembered at each verb.

## Grounds

- pursued: a store swapped for a symlink after the handle opened redirects no read, pinned by TestStoreHandleReadsOnlyTheDirectoryItOpened and by the store_swap_test.go set, which swaps the store at the moment openStore returns and drives Lint, Bare and Ingest over it; fileBack refuses a symlinked store before reading its registry; a memory read by path outside store.go and the locked writer reappearing would show it wrong
