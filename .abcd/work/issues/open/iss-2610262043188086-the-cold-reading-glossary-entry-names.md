---
schema_version: 1
id: "iss-2610262043188086"
slug: "the-cold-reading-glossary-entry-names"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1129 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/ledger/cold-reading.md:11"
remedy: "none (filed automatically)"
related_intents: [itd-2609211913453478]
finding_ref: cfk-0c11fd5a93282a07
---

The cold-reading glossary entry names ledger/warm in not_to_be_confused_with, which itd-2609211913453478 restricts to record families

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/ledger/cold-reading.md:11` — "not_to_be_confused_with: [ledger/warm, core/record-families]"
- End B: `.abcd/development/intents/shipped/itd-2609211913453478-one-page-in-the-glossary-maps-abcd-s.md:39` — "may name only a family on the page"

The shipped intent rules that every other glossary entry's not_to_be_confused_with may name only a family on the record-families page (intent, spec, step, bundle, issue, release, status). cold-reading.md names ledger/warm and core/record-families, neither of which is a family on that page. The entry and the rule cannot both stand; the other ledger entries share the same shape.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1129.
