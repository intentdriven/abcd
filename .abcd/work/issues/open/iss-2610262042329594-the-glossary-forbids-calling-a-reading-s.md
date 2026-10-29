---
schema_version: 1
id: "iss-2610262042329594"
slug: "the-glossary-forbids-calling-a-reading-s"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1076 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/ledger/position.md:25"
remedy: "In itd-143 line 68 name the reading by its position instead of 'the Step 2 reading'."
related_intents: [itd-143]
finding_ref: cfk-039e3daeb4d8a01e
---

The glossary forbids calling a reading's position a step; itd-143 calls the widening reading the Step 2 reading

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/ledger/position.md:25` — "Do not call a position a phase (the retired sequencing unit) or a step (a landable piece of a spec); those are delivery vocabulary."
- End B: `.abcd/development/intents/drafts/itd-143-the-brief-gains-a-framing-chapter-under.md:68` — "dropping the Step 2 reading would forfeit the position of greatest exposure"

The position entry says a position is never a step, which is a landable piece of a spec. itd-143 names the widening reading the Step 2 reading in the same sentence that speaks of its position.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1076.
