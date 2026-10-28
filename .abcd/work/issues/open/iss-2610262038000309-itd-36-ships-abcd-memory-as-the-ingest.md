---
schema_version: 1
id: "iss-2610262038000309"
slug: "itd-36-ships-abcd-memory-as-the-ingest"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 765 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/37-memory.md:3"
remedy: "none (filed automatically)"
related_intents: [itd-36]
finding_ref: cfk-61720440f511092a
---

itd-36 ships /abcd:memory as the ingest, ask and lint knowledge substrate under .abcd/memory/; the memory chapter's /abcd:memory keeps agents' memory notes with add, list and remove.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/37-memory.md:3` — "`/abcd:memory` keeps memory notes: short lessons an agent writes so that a later session does not learn them again"
- End B: `.abcd/development/intents/shipped/itd-36-memory-unification.md:23` — "abcd ships `/abcd:memory` — a per-`voyage` curated knowledge substrate"

itd-36's press release and scope ship /abcd:memory as a per-voyage knowledge substrate in the .abcd/memory/ namespace, with ingest, ask and lint, and its out-of-scope text still has the user invoke /abcd:memory ingest. Its scope adds only that those verbs merged into /abcd:library. 37-memory.md uses the same name for something else: short lessons agents write, kept in a memory.json beside each rules layer, with add, list and remove. One command name stands for two concepts, and a reader of itd-36 reaches for sub-verbs the shipped /abcd:memory does not have.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 765.
