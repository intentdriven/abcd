---
schema_version: 1
id: "iss-2610262036211949"
slug: "in-the-mode-chapter-mode-means-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 642 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/26-mode.md:4"
remedy: "none (filed automatically)"
related_intents: [itd-2610150602547271]
finding_ref: cfk-5114c49234f6b384
---

In the mode chapter, mode means the waiting-on badge (managed, facilitator, product-thinker) and the away mode set by abcd mode autonomous; itd-2610150602547271 adds 'the facilitator's mode', facilitator.mode human|auto, shown on the same board.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/26-mode.md:4` — "waiting-on state behind the status line's badge: `managed`, `facilitator` or"
- End B: `.abcd/development/intents/planned/itd-2610150602547271-a-repository-declares-whether-its.md:15` — "The facilitator's mode gates the merge"

26-mode stores the waiting-on state as the mode that abcd mode prints and sets, and one of its values is facilitator; its away mode is entered with the autonomous sub-verb. itd-2610150602547271 names a separate per-repository setting, the facilitator's mode, whose values human and auto decide who merges, and has the bare board name 'the mode'. On one board, mode and facilitator would then name two different facts, and auto sits next to autonomous with an unrelated meaning.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 642.
