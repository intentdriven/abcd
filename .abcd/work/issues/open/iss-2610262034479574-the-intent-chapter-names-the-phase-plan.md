---
schema_version: 1
id: "iss-2610262034479574"
slug: "the-intent-chapter-names-the-phase-plan"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 523 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:74"
remedy: "none (filed automatically)"
related_intents: [itd-24]
finding_ref: cfk-84aab1560491ca18
---

The intent chapter names the phase plan as the canonical build order; itd-24 records that phases are retired.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:74` — "the canonical build order is the phase plan at"
- End B: `.abcd/development/intents/shipped/itd-24-reflect-command.md:90` — "phases are retired, so `<phase-id>` reads as the release tag"

05-intent's id rules say an id number is no execution-order guarantee because the canonical build order is the phase plan under roadmap/phases/. itd-24's decision 4 (adr-2609212115255771, 2026-09-21) records that phases are retired and reads every phase as a release. A retired plan cannot be the canonical build order, which leaves the chapter naming a source for the build order that the record has withdrawn.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 523.
