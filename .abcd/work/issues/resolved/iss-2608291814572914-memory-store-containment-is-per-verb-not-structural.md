---
schema_version: 1
id: "iss-2608291814572914"
slug: "memory-store-containment-is-per-verb-not-structural"
severity: "minor"
category: "architectural-insight"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/core/memory/writer.go"
resolution: "Every memory read (Bare, QueryPages, Ingest dedup, fileBack, Lint crawl and residue, coverage crawl, headroom) goes through one os.Root store handle opened inside the repository root; fileBack now opens it before reading. The locked writer keeps validatedMemoryDir for its writes."
impact: fix
resolved_by:
  commit: "2475b570"
---

ultra-v0.6.8 altitude 3: the memory store's symlink guard is a per-verb pre-check repeated at five entry points (Bare, QueryPages, Ingest, Lint, runMemoryCoverageLint) rather than a containment mechanism, and fileBack in ask.go reached Dir(root) and existingPageFrontmatter without it. The site package fixed the identical class (gh #487) by opening one os.Root and routing every read through fsutil.ReadGuardedInRoot. Deeper fix: memory holds a store-root handle the same way so containment is structural rather than remembered at each verb.

## Grounds

- pursued: a store swapped for a symlink after the handle opened redirects no read, and fileBack refuses a symlinked store before reading its registry; a memory read by path outside store.go reappearing would show it wrong
