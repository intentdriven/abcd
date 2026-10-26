---
schema_version: 1
id: "iss-2610262043310309"
slug: "itd-26-names-abcd-memory-ingest-as-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1143 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:70"
remedy: "none (filed automatically)"
related_intents: [itd-26, itd-2610090831227812]
finding_ref: cfk-32ee102961448952
---

itd-26 names /abcd:memory ingest as the user-facing knowledge-ingest verb; itd-2610090831227812 makes /abcd:memory an agent's command for memory notes

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:70` — "Two distinct user-facing verbs (`/abcd:loot` for code, `/abcd:memory ingest` for knowledge)"
- End B: `.abcd/development/intents/shipped/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md:97` — "`/abcd:memory` becomes an agent's command for memory notes"

itd-26 relies on two user-facing verbs, /abcd:loot for code and /abcd:memory ingest for knowledge. The shipped menu intent merged the old memory command into library and gave the name /abcd:memory to an agent-only command for memory notes. The one name now means two different things, and the knowledge-ingest verb itd-26 pairs with is no longer called that.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1143.
