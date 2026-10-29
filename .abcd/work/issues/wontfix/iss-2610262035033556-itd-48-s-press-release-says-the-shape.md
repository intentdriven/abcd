---
schema_version: 1
id: "iss-2610262035033556"
slug: "itd-48-s-press-release-says-the-shape"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 544 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:24"
remedy: "none (filed automatically)"
related_intents: [itd-48]
finding_ref: cfk-4845ed4f03727f67
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032131693. Same A line and root cause; 05-intent.md's sentence is the one that records the re-homing."
duplicates: [iss-2610262032131693]
---

itd-48's press release says the shape role went live as a working verb; the intent chapter says itd-48 did not build the shape role

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:24` — "With all three roles live, `/abcd:intent consistency` and `/abcd:intent shape` move from documented command surface to working command surface."
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:616` — "The shape role below is not built by itd-48: on 2026-09-21 the product thinker re-homed it to itd-34's kinds lint, and the overlap question to itd-42's pre-pass."

itd-48 announces that with all three roles live, /abcd:intent shape moves to a working command surface. The intent chapter says the shape role is not built by itd-48 and was re-homed to itd-34's kinds lint. Both cannot be true of the same shipped record: one says the shape verb exists because of itd-48, the other says itd-48 never delivered it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 544.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032131693. Same A line and root cause; 05-intent.md's sentence is the one that records the re-homing.
