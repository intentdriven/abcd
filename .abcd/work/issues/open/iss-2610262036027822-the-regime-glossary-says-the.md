---
schema_version: 1
id: "iss-2610262036027822"
slug: "the-regime-glossary-says-the"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 618 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/ledger/regime.md:17"
remedy: "none (filed automatically)"
finding_ref: cfk-2ff5b5aeb903a352
---

The regime glossary says the supply-regime check validates at ingest what a reading may produce; 23-reading says the generative regime's constraint falls at admission, not ingest.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/ledger/regime.md:17` — "The supply-regime check validates at ingest what a reading was licensed to produce"
- End B: `.abcd/development/brief/04-surfaces/23-reading.md:250` — "the generative licence is the widest of the four, with its constraint falling at"

regime.md says the supply-regime check validates at ingest what a reading was licensed to produce, and its definition has the ingest verb refusing or flagging. 23-reading says the generative regime declares no reserved names, that the generative licence is the widest of the four with its constraint falling at admission rather than at ingest, so at that position the gate performs no check of its own, and the prose layer that could flag was removed. The glossary's claim holds at three positions, not four.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 618.
