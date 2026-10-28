---
schema_version: 1
id: "iss-2610262043429522"
slug: "itd-81-requires-binary-verdicts-from"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1156 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-81-judge-calibration.md:102"
remedy: "none (filed automatically)"
related_intents: [itd-81, itd-80]
finding_ref: cfk-c4501af4b0d0dd1d
---

itd-81 requires binary verdicts from every verdict-rendering agent, the intent auditor included; itd-80 ships that reviewer with four-state verdicts

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-81-judge-calibration.md:102` — "Binary verdicts. Severity is structural (a FIX-FIRST list vs a NOTES list)"
- End B: `.abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:19` — "per-criterion verdicts (MET / MET_WITH_CONCERNS / NOT_MET / INCONCLUSIVE)"

itd-81's rule covers every agent that renders a verdict and names intent-auditor. It puts binary verdicts in scope and carries severity in structure. itd-80 ships that same reviewer emitting MET / MET_WITH_CONCERNS / NOT_MET / INCONCLUSIVE for each criterion. A four-valued verdict per criterion is not binary. Either the shipped reviewer breaks the discipline, or the discipline does not cover the agent it names.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1156.
