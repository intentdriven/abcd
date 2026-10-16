---
schema_version: 1
id: "iss-2610160608380578"
slug: "the-exit-codes-section-at-abcd"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/27-implement.md"
remedy: "Add load --wait's exit 1 on timeout and the wrapped command's own exit code to the Exit codes section at 27-implement.md:400-404."
resolution: "27-implement.md Exit codes now states the load wait's exit 1 on timeout or an unusable hold and, with a command, that command's code, 128+N or 127"
impact: internal
---

The Exit codes section at .abcd/development/brief/04-surfaces/27-implement.md:400-404 lists 0 done (and every load-check status), 2 refused and 3 contention. `abcd implement load --wait` exits 1 when it gives up after --timeout and, with a command after --, exits with that command's code (implement load --help; the chapter's own lines 280-282).
