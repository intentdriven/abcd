---
schema_version: 1
id: "iss-2610262043043610"
slug: "itd-1-fixes-the-audit-at-four-verdict"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1113 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-1-acceptance-gates.md:46"
remedy: "none (filed automatically)"
related_intents: [itd-1, itd-50]
finding_ref: cfk-95caed7079f0d1bb
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. itd-1's four states are the per-criterion verdicts; itd-50 states that UNACHIEVABLE is not a fifth criterion verdict and the intent is planned, so nothing ships a conflicting enum yet."
---

itd-1 fixes the audit at four verdict states; itd-50 adds UNACHIEVABLE as a fifth terminal audit outcome.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-1-acceptance-gates.md:46` — "four states, lifted from PAUL. Binary pass/fail loses information; four states preserve nuance without exploding."
- End B: `.abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:52` — "A new terminal audit outcome distinct from `NOT_MET`"

itd-1 fixes the escalation states at 'four states' (MET, MET_WITH_CONCERNS, NOT_MET, INCONCLUSIVE) for both per-criterion and overall verdicts, and argues four keep nuance without exploding. itd-50 introduces 'a new terminal audit outcome distinct from NOT_MET', UNACHIEVABLE, and its decisions apply it to a criterion judged unmeetable. Either the audit's verdict vocabulary has four values or it has five; the discipline that governs every audit is not amended, so the Audit Notes writer and the loop would disagree on what a verdict can be.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1113.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. itd-1's four states are the per-criterion verdicts; itd-50 states that UNACHIEVABLE is not a fifth criterion verdict and the intent is planned, so nothing ships a conflicting enum yet.
