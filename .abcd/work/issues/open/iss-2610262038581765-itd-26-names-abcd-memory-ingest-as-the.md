---
schema_version: 1
id: "iss-2610262038581765"
slug: "itd-26-names-abcd-memory-ingest-as-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 831 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:70"
remedy: "none (filed automatically)"
related_intents: [itd-26]
finding_ref: cfk-729596fa9d3a4125
---

itd-26 names /abcd:memory ingest as the knowledge-ingest verb; the surfaces register gives the name memory to the notes command

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:70` — "Two distinct user-facing verbs (`/abcd:loot` for code, `/abcd:memory ingest` for knowledge); one shared substrate."
- End B: `.abcd/development/brief/04-surfaces/README.md:494` — "That merge freed the name `memory`, which now belongs to the notes command"

itd-26 pairs /abcd:loot with /abcd:memory ingest for knowledge, over one shared substrate. The surfaces register says the library command took over what the old memory command held, and that the freed name memory now belongs to the notes command. /abcd:memory names two different surfaces.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 831.
