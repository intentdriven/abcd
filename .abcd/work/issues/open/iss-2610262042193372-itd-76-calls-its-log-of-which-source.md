---
schema_version: 1
id: "iss-2610262042193372"
slug: "itd-76-calls-its-log-of-which-source"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1061 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-76-source-provenance-ledger.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-76]
finding_ref: cfk-efa64e6dd16787bd
---

itd-76 calls its log of which source influenced which decision 'the provenance ledger'; the ledger entry keeps that name for another sense and calls this one 'the sources ledger'.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-76-source-provenance-ledger.md:18` — "abcd manages this with a personal corpus and a provenance ledger, guarded mechanically"
- End B: `.abcd/development/brief/glossary/core/ledger.md:29` — "Origin, grounds and disposition stamped at the point of commitment | **the provenance ledger**"

itd-76's append-only JSONL ledger records which source influenced which decision. The ledger glossary fixes one spelling for each sense. 'The provenance ledger' is origin, grounds and disposition stamped at the point of commitment. Source-to-decision provenance for the local sources corpus is 'the sources ledger'. itd-76 gives one sense's name to the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1061.
