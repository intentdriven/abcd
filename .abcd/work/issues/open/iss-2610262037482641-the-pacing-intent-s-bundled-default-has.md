---
schema_version: 1
id: "iss-2610262037482641"
slug: "the-pacing-intent-s-bundled-default-has"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 750 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/34-build.md:228"
remedy: "none (filed automatically)"
related_intents: [itd-2609201925079472]
finding_ref: cfk-6e499ca5d772967e
---

The pacing intent's bundled default has a 300-minute pause; the build chapter's bundled default has no pause.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/34-build.md:228` — "the bundled default, 120 minutes of work, no pause, 2 sub-agents and 3 fix rounds"
- End B: `.abcd/development/intents/shipped/itd-2609201925079472-an-autonomous-implementation-run-paces.md:55` — "The bundled default is 120 minutes of work, 300 of pause, two lanes"

itd-2609201925079472 decision 5 gives the bundled default as 120 minutes of work, 300 of pause and two lanes, and carries no amendment. 34-build.md gives the bundled default as 120 minutes of work, no pause, 2 sub-agents and 3 fix rounds, citing that same decision 5 and a ruling of 2026-10-10 that removed the pause. A run left on the bundled default pauses five hours by one document and not at all by the other. The intent also speaks of three layered numbers where the brief resolves four, the fix-round cap included.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 750.
