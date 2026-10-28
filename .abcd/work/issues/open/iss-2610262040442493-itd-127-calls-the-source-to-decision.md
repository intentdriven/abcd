---
schema_version: 1
id: "iss-2610262040442493"
slug: "itd-127-calls-the-source-to-decision"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 952 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-127-a-paper-is-reconstructed-from-the.md:13"
remedy: "none (filed automatically)"
related_intents: [itd-127]
finding_ref: cfk-27a9503a7e42c159
---

itd-127 calls the source-to-decision trail 'the provenance ledger'; the ledger glossary calls that trail 'the sources ledger'.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-127-a-paper-is-reconstructed-from-the.md:13` — "A Paper Is Reconstructed From the Provenance Ledger"
- End B: `.abcd/development/brief/glossary/core/ledger.md:31` — "Source-to-decision provenance for the local sources corpus"

In the ledger entry, 'the provenance ledger' is origin, grounds and disposition stamped at the point of commitment, and 'the sources ledger' is source-to-decision provenance for the sources collection. itd-127 calls the record of which source influenced which decision (itd-76's store) the provenance ledger, using one named ledger's spelling for the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 952.
