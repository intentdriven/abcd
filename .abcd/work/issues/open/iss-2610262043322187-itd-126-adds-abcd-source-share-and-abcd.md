---
schema_version: 1
id: "iss-2610262043322187"
slug: "itd-126-adds-abcd-source-share-and-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1145 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-126-a-team-shares-one-bibliography-without.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-126, itd-2610090831227812]
finding_ref: cfk-30992871b6572ed1
---

itd-126 adds abcd source share and abcd source ingest; itd-2610090831227812 merged the source family into library with no alias

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-126-a-team-shares-one-bibliography-without.md:17` — "`abcd source ingest` imports the repo's shared entries into the local corpus"
- End B: `.abcd/development/intents/shipped/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md:48` — "`consult`, `ingest`, `memory` and `source` merge into `library`"

itd-126's press release builds on the abcd source family (share, ingest), while the shipped intent merged consult, ingest, memory and source into library and, under adr-40, kept no old name as an alias. Two names stand for one family, and a builder following the draft would revive a retired verb.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1145.
