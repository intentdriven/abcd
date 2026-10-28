---
schema_version: 1
id: "iss-2610262033287288"
slug: "the-library-chapter-says-the-memory"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 410 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/07-library.md:11"
remedy: "none (filed automatically)"
finding_ref: cfk-09d2e3715bfed6a3
---

The library chapter says the `memory` command was merged into `library`; the memory chapter documents a live `/abcd:memory` verb for a different store.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/07-library.md:11` — "The command merges the two that came before it, `source` and `memory`, under the product thinker's ruling of 2026-10-09"
- End B: `.abcd/development/brief/04-surfaces/37-memory.md:3` — "`/abcd:memory` keeps memory notes: short lessons an agent writes so that a later session does not learn them again"

07-library says `library` merges the two commands that came before it, `source` and `memory`, and calls `.abcd/memory/` (distilled, cited pages) 'the memory store'. 37-memory documents `/abcd:memory` as a shipped verb for memory notes kept in `memory.json` layers (`~/.abcd.noindex/memory.json` and `.abcd/memory.json`), a different store with a different shape. So 'memory' and the memory store each name two concepts. 07-library also lists user-scope memory outside the repo as not built, yet 37-memory's machine layer is exactly that.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 410.
