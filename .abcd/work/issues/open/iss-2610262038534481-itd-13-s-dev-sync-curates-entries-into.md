---
schema_version: 1
id: "iss-2610262038534481"
slug: "itd-13-s-dev-sync-curates-entries-into"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 825 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-13-scheduled-dev-sync.md:15"
remedy: "none (filed automatically)"
related_intents: [itd-13]
finding_ref: cfk-06ba6997c3aac6e8
---

itd-13's dev-sync curates entries into the issue ledger; 03-configuration says capture writes the ledger directly and the sync has nothing left to migrate there

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-13-scheduled-dev-sync.md:15` — "curating new entries into the `.abcd/work/issues/` ledger"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:1199` — "`abcd capture` writes the issue ledger directly, so the migration this design was drawn around has nothing left to migrate."

itd-13 has scheduled dev-sync curate new entries into the .abcd/work/issues/ ledger. 03-configuration says abcd capture writes the issue ledger directly, so the sync design has nothing left to migrate into it, and its staged source-to-target pairs do not include the ledger. One of the two sync sources itd-13 promises has no place in the configuration model.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 825.
