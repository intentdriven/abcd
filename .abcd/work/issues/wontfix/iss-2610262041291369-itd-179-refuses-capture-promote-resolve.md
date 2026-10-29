---
schema_version: 1
id: "iss-2610262041291369"
slug: "itd-179-refuses-capture-promote-resolve"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1004 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-179-the-reasoning-behind-what-was-pursued-no.md:28"
remedy: "none (filed automatically)"
related_intents: [itd-179]
finding_ref: cfk-cb5e0d90d2e1c1fe
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The matrix, the chapter and the resolved issue agree."
---

itd-179 refuses capture promote, resolve and wontfix without grounds; the verification matrix says resolve records grounds only when they are given

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-179-the-reasoning-behind-what-was-pursued-no.md:28` — "(promote / resolve / wontfix), refusal on absence."
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:64` — "A note and an impact are required, grounds are recorded when given"

itd-179 puts a grounds argument on the capture triage routes (promote, resolve and wontfix) and refuses any of them when it is absent. The verification matrix's capture-resolve row says a note and an impact are required and grounds are recorded when given, which makes grounds optional on resolve. Its wontfix row names no grounds at all. A resolve with no grounds cannot be both refused and accepted.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1004.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The matrix, the chapter and the resolved issue agree.
