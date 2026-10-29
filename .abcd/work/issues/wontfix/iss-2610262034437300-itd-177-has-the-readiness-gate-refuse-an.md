---
schema_version: 1
id: "iss-2610262034437300"
slug: "itd-177-has-the-readiness-gate-refuse-an"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 518 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-177-an-intent-s-claims-are-typed-and-its.md:13"
remedy: "none (filed automatically)"
related_intents: [itd-177]
finding_ref: cfk-532cc7d83ab20756
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. internal/core/intent/ready.go marks those checks Advisory until the rethink of the reading work; the resolved issue is the later record that says which holds."
---

itd-177 has the readiness gate refuse an intent that leaves a context claim unrecorded; the intent chapter reports the claim rows as advisory, never withholding readiness.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-177-an-intent-s-claims-are-typed-and-its.md:13` — "the readiness gate refuses an intent that leaves a context claim unrecorded"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:325` — "are reported as advisory and never withhold readiness, their refusals parked by iss-2609091009111294"

itd-177's title and scope promise readiness-gate refusals per the claim gradient: an intent leaving its scope-condition claim unrecorded is refused, and so is a duplicated or missing condition marker, by name. 05-intent's readiness gate reports the mechanism and scope-condition rows as advisory rows that never withhold readiness, their refusals parked. An intent the shipped intent says the gate refuses is READY under the chapter.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 518.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. internal/core/intent/ready.go marks those checks Advisory until the rethink of the reading work; the resolved issue is the later record that says which holds.
