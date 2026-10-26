---
schema_version: 1
id: "iss-2610262033514039"
slug: "the-memory-substrate-has-session-memory"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 443 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/07-memory.md:50"
remedy: "none (filed automatically)"
finding_ref: cfk-7e29ed375ac87a9f
---

The memory substrate has session_memory pages distilled via dev-sync memory; the board chapter says no dev-sync verb ships and no dev-sync code exists.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/07-memory.md:50` — "distilled by the `internal/core/memory` adapter via `dev-sync memory`"
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:583` — "no `dev-sync` verb is on any shipped surface, and no dev-sync code exists in the tree."

07-memory's source-class enum has session_memory pages 'distilled by the internal/core/memory adapter via dev-sync memory', re-distilled on each sync, and names the vendor memory directory as a harvest source for dev-sync memory. 08-abcd states that no dev-sync verb is on any shipped surface and no dev-sync code exists in the tree. A page class produced only through a verb that does not exist cannot be described as working, so one of the two pages is wrong about the tree.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 443.
