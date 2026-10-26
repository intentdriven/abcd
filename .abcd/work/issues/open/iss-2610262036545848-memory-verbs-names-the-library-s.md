---
schema_version: 1
id: "iss-2610262036545848"
slug: "memory-verbs-names-the-library-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 683 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md:1139"
remedy: "none (filed automatically)"
finding_ref: cfk-ac9b32a165d1bf3a
---

'Memory verbs' names the library's curation of `.abcd/memory/`, while `/abcd:memory` is a separate verb over memory notes

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/03-configuration.md:1139` — "New projects use `.abcd/memory/`, the substrate `abcd library` curates through its memory verbs."
- End B: `.abcd/development/brief/04-surfaces/37-memory.md:3` — "`/abcd:memory` keeps memory notes: short lessons an agent writes so that a later session does not learn them again"

The configuration chapter says the repository's `.abcd/memory/` substrate is curated by `abcd library` through its memory verbs. The memory chapter gives the name `memory` to a top-level verb that keeps short notes in `memory.json` files. The one name now covers two different stores side by side in `.abcd/`. A reader looking for the memory verbs that curate `.abcd/memory/` lands on a verb that writes a different file for a different purpose.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 683.
