---
schema_version: 1
id: "iss-2610262040451252"
slug: "itd-196-builds-its-rehearsal-rungs-on"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 953 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-196-a-capability-is-rehearsed-end-to-end.md:40"
remedy: "none (filed automatically)"
related_intents: [itd-196]
finding_ref: cfk-a4d3fbd654b76de6
---

itd-196 builds its rehearsal rungs on the phase; the phase glossary entry says the phase is superseded and survives only as history.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-196-a-capability-is-rehearsed-end-to-end.md:40` — "The cumulative rung is what turns a phase into something more than a list of"
- End B: `.abcd/development/brief/glossary/core/phase.md:18` — "the sequencing layer is dependencies plus the lifecycle shelves, rendered as the Now / Next / Later status block; the phase documents stay as history."

itd-196's cumulative rung and close-of-phase rung rehearse what the phase can deliver and treat the phase as a live delivery unit. The phase entry says the sequencing layer is now dependencies plus the lifecycle shelves and that the phase documents stay as history, so the rung is keyed to a unit that no longer exists.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 953.
