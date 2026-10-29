---
schema_version: 1
id: "iss-2610262037423257"
slug: "itd-177-says-the-readiness-gate-refuses"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 743 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-177-an-intent-s-claims-are-typed-and-its.md:13"
remedy: "none (filed automatically)"
related_intents: [itd-177]
finding_ref: cfk-7edaf0cf6bdafae2
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. internal/core/intent/ready.go marks those checks Advisory until the rethink of the reading work; the resolved issue is the later record that says which holds."
---

itd-177 says the readiness gate refuses an intent with an unrecorded context claim; 34-build says the readiness gate only reports scope conditions and the mechanism prompt as advisory

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-177-an-intent-s-claims-are-typed-and-its.md:13` — "the readiness gate refuses an intent that leaves a context claim unrecorded"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:66` — "advisory; a run is where they bind, because an autonomous lane has nobody to"

Shipped itd-177 is titled as making the readiness gate refuse an intent that leaves a context claim, its scope conditions, unrecorded. 34-build describes the claim-sections check differently. There the readiness gate reports the mechanism prompt and the scope conditions as advisory, and they bind only when a build run starts. The two documents disagree on whether intent ready refuses such an intent or merely reports it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 743.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. internal/core/intent/ready.go marks those checks Advisory until the rethink of the reading work; the resolved issue is the later record that says which holds.
