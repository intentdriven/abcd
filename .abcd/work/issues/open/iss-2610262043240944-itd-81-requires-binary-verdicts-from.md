---
schema_version: 1
id: "iss-2610262043240944"
slug: "itd-81-requires-binary-verdicts-from"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1135 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-81-judge-calibration.md:102"
remedy: "none (filed automatically)"
related_intents: [itd-81, itd-28]
finding_ref: cfk-605aa8215d29eb0d
---

itd-81 requires binary verdicts from every judging agent; itd-28 locks reviewers to the three-valued SHIP / NEEDS_WORK / MAJOR_RETHINK enum

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-81-judge-calibration.md:102` — "Binary verdicts. Severity is structural (a FIX-FIRST list vs a NOTES list)"
- End B: `.abcd/development/intents/shipped/itd-28-rp-reviews-into-flow.md:65` — "the canonical review verdict enum, shared by every oracle adapter"

itd-81 covers every reviewer and auditor that renders a verdict and puts binary verdicts in scope, with severity structural rather than scaled. itd-28 locks the canonical review verdict enum, shared by every oracle adapter, at three values. A review agent cannot satisfy both: its verdict is either binary or one of the three.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1135.
