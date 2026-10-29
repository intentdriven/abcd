---
schema_version: 1
id: "iss-2610262034431772"
slug: "itd-179-has-intent-ready-refuse-when"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 517 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-179-the-reasoning-behind-what-was-pursued-no.md:27"
remedy: "none (filed automatically)"
related_intents: [itd-179]
finding_ref: cfk-83b18c71a922c4d8
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. ready.go documents the grounds check as advisory."
---

itd-179 has intent ready refuse when grounds are absent; the intent chapter reports the readiness gate's grounds row as advisory, never withholding readiness.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-179-the-reasoning-behind-what-was-pursued-no.md:27` — "A grounds argument on `intent ready` and on the capture triage routes (promote / resolve / wontfix), refusal on absence."
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:325` — "and the grounds row (a discipline record is exempt: it carries no conjecture of its own) are reported as advisory and never withhold readiness"

itd-179's scope puts a grounds argument on intent ready with refusal on absence. 05-intent's readiness gate lists the grounds row among the rows reported as advisory that never withhold readiness, their refusals parked, and makes recording grounds an optional write of the gate. A gate that refuses without grounds and one that reports READY without them are different gates.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 517.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. ready.go documents the grounds check as advisory.
