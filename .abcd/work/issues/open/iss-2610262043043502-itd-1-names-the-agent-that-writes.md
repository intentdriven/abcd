---
schema_version: 1
id: "iss-2610262043043502"
slug: "itd-1-names-the-agent-that-writes"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1112 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-1-acceptance-gates.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-1, itd-50]
finding_ref: cfk-6d85b7da1097148b
---

itd-1 names the agent that writes criterion verdicts 'intent-auditor'; itd-50 names it 'intent-fidelity-reviewer'.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-1-acceptance-gates.md:17` — "emitted by the `intent-auditor` agent's single-document role"
- End B: `.abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:25` — "the `intent-fidelity-reviewer` records a `NOT_MET` verdict"

itd-1 states that every shipped intent's per-criterion verdicts are 'emitted by the intent-auditor agent's single-document role'. itd-50, whose whole loop is driven by those verdicts, says 'the intent-fidelity-reviewer records a NOT_MET verdict'. Two names are used for the one agent the loop depends on, and the second carries the 'review' word the audit was renamed away from, so a builder of itd-50 has to guess which agent the loop calls.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1112.
