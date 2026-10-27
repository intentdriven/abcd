---
schema_version: 1
id: "iss-2610262042495955"
slug: "itd-81-puts-judge-panels-out-of-scope"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1095 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-81-judge-calibration.md:117"
remedy: "none (filed automatically)"
related_intents: [itd-81, itd-98]
finding_ref: cfk-0398c0e4bf431b3c
---

itd-81 puts judge panels out of scope; itd-98 runs a judge panel under the itd-81 discipline

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-81-judge-calibration.md:117` — "errors across models are correlated, so the Condorcet argument does not apply"
- End B: `.abcd/development/intents/drafts/itd-98-solo-vs-duo-is-measured-not-debated.md:54` — "An outcome judge panel under the itd-81 discipline"

itd-81 rejects judge panels and juries because errors across models are correlated, treating disagreement as a triage signal and not a vote. itd-98 places an outcome judge panel with independent votes under that same discipline. The discipline it cites excludes the mechanism it builds.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1095.
