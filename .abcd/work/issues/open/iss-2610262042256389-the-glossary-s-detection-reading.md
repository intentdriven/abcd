---
schema_version: 1
id: "iss-2610262042256389"
slug: "the-glossary-s-detection-reading"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1068 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/ledger/position.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-86]
finding_ref: cfk-13128cf40d3fc4b7
---

The glossary's detection reading compares the shipped tree with the claim record; itd-86's cold reading detects contradictions inside one document

A terminology drift (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/ledger/position.md:17` — "the detection pass (gaps between the shipped tree and the claim record)"
- End B: `.abcd/development/intents/drafts/itd-86-cold-reading-surface.md:29` — "A surface that takes exactly one target document and produces candidate detections against the constraints visible in that document alone."

The position entry fixes the detection position of a cold reading as gaps between the shipped tree and the claim record. itd-86 names a cold reading that takes exactly one document and produces detections only against that document's own constraints. The same instrument name and the same word, detection, are given an incompatible object, so someone building from itd-86 would build a different instrument from the one the ledger glossary defines.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1068.
