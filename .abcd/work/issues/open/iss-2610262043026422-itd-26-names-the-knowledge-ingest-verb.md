---
schema_version: 1
id: "iss-2610262043026422"
slug: "itd-26-names-the-knowledge-ingest-verb"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1110 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:70"
remedy: "none (filed automatically)"
related_intents: [itd-26, itd-39]
finding_ref: cfk-700bb2ff1a143895
---

itd-26 names the knowledge-ingest verb '/abcd:memory ingest'; itd-39 records that 'memory' is now the agents' notes command.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-26-loot-oss-vendor.md:70` — "`/abcd:memory ingest` for knowledge"
- End B: `.abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:78` — "Since the verb split, `memory` is the agents' notes command"

itd-26 builds loot on a provenance substrate it shares with knowledge ingest, which it names '/abcd:memory ingest'. itd-39 records that since the verb split 'memory' is the agents' notes command, and that the source store is reached as 'library ingest', 'library ask' and 'library lint'. The name itd-26 builds on now points at a different command, and the surface it means has another name, so an implementer following itd-26 would wire loot to the wrong verb family.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1110.
