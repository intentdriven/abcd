---
schema_version: 1
id: "iss-2610262033301184"
slug: "the-board-s-now-row-shows-one-lane-and"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 413 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/08-abcd.md:325"
remedy: "none (filed automatically)"
finding_ref: cfk-6456226a2d16226f
---

The board's Now row shows one lane and one awaited role per run; the implement chapter runs a run's lanes in parallel up to its ceiling.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/08-abcd.md:325` — "a run in progress, with the lane the loop works on, its next stage and the role it waits on"
- End B: `.abcd/development/brief/04-surfaces/27-implement.md:386` — "A run works in parallel up to its ceiling (ruling DR6)"

08-abcd's status block lists each intent in a lane as a run in progress, with the lane the loop works on, its next stage and the role it waits on, and its JSON carries a single `lane` object. 27-implement says a run works in parallel up to its ceiling: a stage moves on any lane, several lanes can each await agents, and the status render names every lane alive with each agent it awaits. A run with several lanes alive has no single lane the loop works on. The board's row shape assumes the one-lane-at-a-time loop that the implement chapter replaced.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 413.
