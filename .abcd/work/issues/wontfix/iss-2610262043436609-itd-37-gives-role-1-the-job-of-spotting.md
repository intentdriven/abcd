---
schema_version: 1
id: "iss-2610262043436609"
slug: "itd-37-gives-role-1-the-job-of-spotting"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1157 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-37-modification-grammar.md:69"
remedy: "none (filed automatically)"
related_intents: [itd-37, itd-48]
finding_ref: cfk-c9590045e90b9fc6
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. 05-intent.md (Role 1, 'Two passes, two destinations') already says Role 1 judges two document kinds, the itd-1 acceptance pass on a shipped intent and the itd-37 MG004 pass on a native spec, and marks the second as staged; itd-48 describes the shipped pass only. The remaining gap, that itd-37 says the pass runs now, is iss-2610262038495354."
---

itd-37 gives Role 1 the job of spotting spec boilerplate at plan-review; itd-48 defines Role 1 as grading a shipped intent's criteria against delivered reality

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-37-modification-grammar.md:69` — "Role 1 boilerplate detection runs at plan-review and ship time"
- End B: `.abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:24` — "`Role 1` (per-intent fidelity) grades a shipped intent's acceptance criteria against delivered reality"

itd-37 has Role 1 judge a spec's Modification Grammar for boilerplate, at plan-review and again at ship time. itd-48 defines Role 1 as per-intent fidelity: it grades a shipped intent's acceptance criteria against what was delivered. A plan-review judgement of spec prose happens before anything is delivered, so it is not the Role 1 that itd-48 defines. Either Role 1 has a second job that no shipped record gives it, or the boilerplate check has no owner.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1157.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. 05-intent.md (Role 1, 'Two passes, two destinations') already says Role 1 judges two document kinds, the itd-1 acceptance pass on a shipped intent and the itd-37 MG004 pass on a native spec, and marks the second as staged; itd-48 describes the shipped pass only. The remaining gap, that itd-37 says the pass runs now, is iss-2610262038495354.
