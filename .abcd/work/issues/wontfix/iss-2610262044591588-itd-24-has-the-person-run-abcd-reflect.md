---
schema_version: 1
id: "iss-2610262044591588"
slug: "itd-24-has-the-person-run-abcd-reflect"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1244 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-24-reflect-command.md:24"
remedy: "none (filed automatically)"
related_intents: [itd-24, itd-2610090831227812]
finding_ref: cfk-aeb304615363c603
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. commands/reflect.md carries user-invocable false and helpgroups.go lists reflect with the agents."
---

itd-24 has the person run /abcd:reflect < tag>; itd-2610090831227812 makes reflect an agent-only page the host refuses when a person types it

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-24-reflect-command.md:24` — "Run `/abcd:reflect v0.11.0` and abcd walks an interview-driven retrospective"
- End B: `.abcd/development/intents/shipped/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md:94` — "Agent-only: lint, reflect, inbox, implement, history, site, docs, scribe, identity, guard, hook and statusline."

itd-24 has the persona run /abcd:reflect v0.11.0 to start the retrospective interview, and says reflection needs the persona's deliberate engagement. itd-2610090831227812 places reflect on the agent-only list, and in that intent an agent-only page carries user-invocable: false, so it is hidden from the person's menu and refused when a person types it. The invocation itd-24 promises is one the host refuses.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1244.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. commands/reflect.md carries user-invocable false and helpgroups.go lists reflect with the agents.
