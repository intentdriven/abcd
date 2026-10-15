---
id: dcn-2607140000000006
slug: iss-93-adr-35-promises-disembark-is
date: 2026-07-14
---
2026-07-14 — iss-93: adr-35 promises disembark is READ-ONLY over the source (a test
hashes the tree before and after), but two paths in the design still write into it —
Pass-0 dev-sync (`.abcd/work/reviews/`, `.abcd/memory/`, `.abcd/work/issues/`) and the
backgrounded-execution checkpoint (`.abcd/logbook/disembark/<ts>/_state.json`). Either
they move out-of-tree (under `<dest>` or the operator-level voyage) or they leave the
disembark path entirely. adr-35 does not settle it; the decision is owed before the
packer ships, and the read-only test is what will force it.
