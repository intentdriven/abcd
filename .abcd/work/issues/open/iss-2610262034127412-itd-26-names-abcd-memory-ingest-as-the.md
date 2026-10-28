---
schema_version: 1
id: "iss-2610262034127412"
slug: "itd-26-names-abcd-memory-ingest-as-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 477 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:70"
remedy: "none (filed automatically)"
related_intents: [itd-26]
finding_ref: cfk-4a584a3796d7f3b7
---

itd-26 names `/abcd:memory ingest` as the knowledge verb beside loot; the library surface merged `memory` into `library`

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:70` — "`/abcd:memory ingest` for knowledge"
- End B: `.abcd/development/brief/04-surfaces/07-library.md:11` — "The command merges the two that came before it, `source` and `memory`, under"

itd-26 pairs its loot verb with a memory ingest verb over one shared provenance substrate. The library surface says the command merged the two that came before it, source and memory, into library, with neither old name kept as an alias. itd-26 names a verb the brief says no longer exists.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 477.
