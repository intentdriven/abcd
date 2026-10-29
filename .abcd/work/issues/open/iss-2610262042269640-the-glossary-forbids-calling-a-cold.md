---
schema_version: 1
id: "iss-2610262042269640"
slug: "the-glossary-forbids-calling-a-cold"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1069 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/ledger/cold-reading.md:25"
remedy: "In itd-90 line 40 say the cold reading reads a document instead of reviews it, matching the glossary's ban on review for a cold reading."
related_intents: [itd-90]
finding_ref: cfk-fe7a580813a9259b
---

The glossary forbids calling a cold reading a review; itd-90 says the cold reading reviews a document

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/ledger/cold-reading.md:25` — "Do not call a cold reading a review or an audit: a review weighs and recommends, and a reading is forbidden both."
- End B: `.abcd/development/intents/drafts/itd-90-brief-interview-for-the-blanks.md:40` — "That surface *reviews* a document for internal contradictions and is deliberately denied context."

The cold-reading entry lists review as a forbidden synonym and says a review weighs and recommends, which a reading may not do. itd-90 describes the cold reading of itd-86 as a surface that reviews a document, using the forbidden word for the very instrument the entry defines.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1069.
