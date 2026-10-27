---
schema_version: 1
id: "iss-2610262044045987"
slug: "itd-8-leaves-adding-code-to-old"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1181 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-8-with-code-bundling.md:35"
remedy: "none (filed automatically)"
related_intents: [itd-8, itd-9]
finding_ref: cfk-bf68cf2a75ba332f
---

itd-8 leaves adding code to old lifeboats to itd-9; itd-9 migrates structural schema only and excludes lifeboat content

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-8-with-code-bundling.md:35` — "Migrating existing lifeboats forward to include code retroactively (covered by itd-9 schema migration)"
- End B: `.abcd/development/intents/drafts/itd-9-schema-migration.md:35` — "Migrating lifeboat content (principles, press releases) — only structural schema"

itd-8 puts 'migrating existing lifeboats forward to include code' out of its own scope as covered by itd-9. itd-9 limits itself to structural schema migration and excludes migrating lifeboat content. Code a lifeboat never carried is content, not a schema change, so the dependency names a record whose scope excludes it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1181.
