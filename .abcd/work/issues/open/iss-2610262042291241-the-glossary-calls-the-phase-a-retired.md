---
schema_version: 1
id: "iss-2610262042291241"
slug: "the-glossary-calls-the-phase-a-retired"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1073 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/ledger/position.md:25"
remedy: "none (filed automatically)"
related_intents: [itd-196]
finding_ref: cfk-23b9bc79ca1fe3f2
---

The glossary calls the phase a retired sequencing unit; the rehearsal discipline builds its rungs on live phases

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/ledger/position.md:25` — "Do not call a position a phase (the retired sequencing unit) or a step (a landable piece of a spec); those are delivery vocabulary."
- End B: `.abcd/development/intents/disciplines/itd-196-a-capability-is-rehearsed-end-to-end.md:40` — "The cumulative rung is what turns a phase into something more than a list of merged intents"

The position entry says a phase is the retired sequencing unit. itd-196's second and third rungs are defined per phase (what the phase can deliver so far, the phase's close), and it calls turning a phase into a runnable demonstration a deliverable. A discipline cannot be enforced against a unit the glossary has retired.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1073.
