---
schema_version: 1
id: "iss-2610262045201587"
slug: "itd-94-has-the-agent-run-abcd-intent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1268 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-94-an-intent-that-is-not-planned-cannot-be.md:19"
remedy: "none (filed automatically)"
related_intents: [itd-94, itd-179]
finding_ref: cfk-6f64f4eb9f9cbfe2
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. internal/core/intent/ready.go marks the grounds check Advisory, and commands/intent.md documents --grounds as a flag that records an entry and reports the gate unchanged; the resolved issue is the later record that says which holds."
---

itd-94 has the agent run `abcd intent ready itd-93` with no other argument and read its verdict; itd-179 makes `intent ready` refuse without a grounds argument

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-94-an-intent-that-is-not-planned-cannot-be.md:19` — "it runs `abcd intent ready itd-93`,"
- End B: `.abcd/development/intents/shipped/itd-179-the-reasoning-behind-what-was-pursued-no.md:27` — "A grounds argument on `intent ready` and on the capture triage routes"

itd-94's readiness flow runs `abcd intent ready itd-93` with no other argument and acts on the NOT READY verdict, with exit 1 a journaled SKIP for drains. itd-179 adds a grounds argument on `intent ready` and on the capture triage routes, refused when absent. Under itd-179 the invocation itd-94 describes is refused before any verdict comes back.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1268.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. internal/core/intent/ready.go marks the grounds check Advisory, and commands/intent.md documents --grounds as a flag that records an entry and reports the gate unchanged; the resolved issue is the later record that says which holds.
