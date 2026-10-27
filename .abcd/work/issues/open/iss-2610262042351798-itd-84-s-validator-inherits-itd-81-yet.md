---
schema_version: 1
id: "iss-2610262042351798"
slug: "itd-84-s-validator-inherits-itd-81-yet"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1079 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-84-intent-decomposition.md:37"
remedy: "none (filed automatically)"
related_intents: [itd-84, itd-81]
finding_ref: cfk-1f37d9c138d3f48b
---

itd-84's validator inherits itd-81 yet renders a three-outcome verdict; itd-81 requires binary verdicts

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-84-intent-decomposition.md:37` — "The verdict has three outcomes, not pass/fail"
- End B: `.abcd/development/intents/disciplines/itd-81-judge-calibration.md:102` — "Binary verdicts. Severity is structural (a FIX-FIRST list vs a NOTES list)"

itd-84 says its validator is a verdict-rendering agent that inherits itd-81, and gives it three outcomes rather than pass/fail. itd-81 scopes binary verdicts for every judge it governs. The validator cannot satisfy both disciplines as written.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1079.
