---
schema_version: 1
id: "iss-2608220750029992"
slug: "two-concurrent-sessions-shared-one"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
resolution: "the conventions content shipped into AGENTS.md; the observation's ask is satisfied there"
impact: internal
---

Two concurrent sessions shared one checkout (2026-08-22, itd-112 banner session vs filing session): a branch switch under a mid-commit peer unstaged its index and swapped the CHANGELOG state; both sessions' tree-wide lint gates failed on each other's work-in-progress. Recovered by coordination with no loss. Root principle: branches do not isolate the working tree, HEAD, or index — the checkout is the unit of isolation. Conventions land in AGENTS.md § Concurrent sessions in the same change.