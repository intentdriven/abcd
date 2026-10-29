---
schema_version: 1
id: "iss-2610262031003182"
slug: "abcd-memory-names-the-itd-36-curated"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 73 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/03-evidence/04-tradeoffs.md:37"
remedy: "In the Memory unification entry of 04-tradeoffs.md (line 37) name `/abcd:library` (ingest, ask and lint under library since the 2026-10-09 merge) in place of `/abcd:memory`, which today keeps agents' notes."
finding_ref: cfk-adb1d5ff4b7c7c65
---

/abcd:memory names the itd-36 curated memory substrate with ingest/ask/lint in the tradeoffs record, and a separate memory-notes store with add/list/remove in the memory chapter

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/03-evidence/04-tradeoffs.md:37` — "ship `/abcd:memory` command initially with `ingest`/`ask`/`lint` sub-verbs"
- End B: `.abcd/development/brief/04-surfaces/37-memory.md:3` — "`/abcd:memory` keeps memory notes: short lessons an agent writes so that a later session does not learn them again"

The tradeoffs record says abcd chose to ship /abcd:memory with ingest, ask and lint sub-verbs over the widened .abcd/memory/ substrate. The memory chapter gives /abcd:memory to something else: short notes agents write to memory.json layers, with add, list and remove. One command name stands for two stores with different sub-verbs, and the record does not say which one the tradeoff's /abcd:memory means.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 73.
