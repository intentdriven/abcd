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
remedy: "In ledger/regime.md say the supply-regime check holds at three positions and that the generative licence declares no reserved names, so its constraint falls at admission rather than at ingest."
finding_ref: cfk-2ff5b5aeb903a352
resolution: "The regime glossary entry now says the supply-regime check holds at three of the four positions: the generative regime declares no reserved names, so the generative licence's constraint falls at admission rather than at ingest, matching 23-reading."
impact: internal
resolved_by:
  commit: "d209ec2417e1482e9846a2d735f676aab6770bef"
---

The regime glossary says the supply-regime check validates at ingest what a reading may produce; 23-reading says the generative regime's constraint falls at admission, not ingest.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/ledger/regime.md:17` — "The supply-regime check validates at ingest what a reading was licensed to produce"
- End B: `.abcd/development/brief/04-surfaces/23-reading.md:250` — "the generative licence is the widest of the four, with its constraint falling at"

regime.md says the supply-regime check validates at ingest what a reading was licensed to produce, and its definition has the ingest verb refusing or flagging. 23-reading says the generative regime declares no reserved names, that the generative licence is the widest of the four with its constraint falling at admission rather than at ingest, so at that position the gate performs no check of its own, and the prose layer that could flag was removed. The glossary's claim holds at three positions, not four.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 618.

## Grounds

- pursued: regime.md's body no longer claims an unqualified ingest check and agrees with 23-reading's account of the generative regime; a reading of 23-reading or the ingest help text showing the gate does check reserved names at the generative position would show it wrong
