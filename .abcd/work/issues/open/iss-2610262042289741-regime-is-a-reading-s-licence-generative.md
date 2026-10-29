---
schema_version: 1
id: "iss-2610262042289741"
slug: "regime-is-a-reading-s-licence-generative"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1072 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/ledger/regime.md:4"
remedy: "In itd-142 lines 19 and 60 rename 'question regimes' to 'question kinds' so regime keeps the glossary's four licences; the 'entailment' kind also borrows a position name and should be renamed."
related_intents: [itd-142]
finding_ref: cfk-4caff5ff9e9e159f
---

Regime is a reading's licence (generative, explicative, evaluative, registrative) in the glossary; itd-142 names four question regimes with other values

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/ledger/regime.md:4` — "naming what it may produce and what the ingest verb refuses or flags: generative, explicative, evaluative or registrative"
- End B: `.abcd/development/intents/drafts/itd-142-the-brief-creation-interview-abcd.md:60` — "Four question regimes — narrative / options / entailment / defaults"

The regime entry fixes the term to the licence a reading holds, with exactly four values. itd-142 uses regime for interview question kinds (narrative, options, entailment, defaults), one of which also borrows the entailment position's name. The same term has two incompatible value sets.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1072.
