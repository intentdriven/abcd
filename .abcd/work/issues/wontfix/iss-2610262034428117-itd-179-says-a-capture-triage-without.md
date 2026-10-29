---
schema_version: 1
id: "iss-2610262034428117"
slug: "itd-179-says-a-capture-triage-without"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 516 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-179-the-reasoning-behind-what-was-pursued-no.md:23"
remedy: "none (filed automatically)"
related_intents: [itd-179]
finding_ref: cfk-5fcc4ce01dcd6115
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The resolved issue is the later record that names the change."
---

itd-179 says a capture triage without grounds is refused; the capture chapter says resolving records grounds only when given, their absence parked.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-179-the-reasoning-behind-what-was-pursued-no.md:23` — "A triage without grounds is refused."
- End B: `.abcd/development/brief/04-surfaces/06-capture.md:319` — "grounds are recorded when given, their absence parked by iss-2609091009111294"

itd-179 promises that capture's triage routes (promote, resolve, wontfix) take a grounds argument and that a triage without grounds is refused. 06-capture states that resolving records grounds when given with their absence parked by iss-2609091009111294, that grounds are optional on a wontfix, and that a promotion's missing grounds are reported rather than refused. The shipped intent's refusal and the chapter's acceptance of a groundless triage cannot both describe the verbs.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 516.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The resolved issue is the later record that names the change.
