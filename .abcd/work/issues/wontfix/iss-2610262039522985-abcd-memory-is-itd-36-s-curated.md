---
schema_version: 1
id: "iss-2610262039522985"
slug: "abcd-memory-is-itd-36-s-curated"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 893 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md:48"
remedy: "none (filed automatically)"
related_intents: [itd-36]
finding_ref: cfk-4a3091aa9f4e301a
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262038000309. Same itd-36 scope line against the register's memory row."
duplicates: [iss-2610262038000309]
---

`/abcd:memory` is itd-36's curated knowledge store with `ingest`, `ask` and `lint`, and also the README's command for short lessons agents keep.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/README.md:48` — "Keep short lessons an agent would otherwise relearn, stamped with who wrote them"
- End B: `.abcd/development/intents/shipped/itd-36-memory-unification.md:41` — "**`/abcd:memory` command** with sub-verbs `ingest` / `ask` / `lint`."

itd-36's scope defines `/abcd:memory` as the command with sub-verbs `ingest`, `ask` and `lint` over the `.abcd/memory/` store, whose verbs have since merged into library. The README's register lists `/abcd:memory` as the notes command that keeps short lessons an agent would otherwise relearn, stored in `memory.json`. itd-36 records the merge but not the reuse of the name, so one name stands for two different stores with different verbs.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 893.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262038000309. Same itd-36 scope line against the register's memory row.
