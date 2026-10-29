---
schema_version: 1
id: "iss-2610262039173971"
slug: "itd-8-extends-a-code-rescuer-agent-it"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 850 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-8-with-code-bundling.md:31"
remedy: "none (filed automatically)"
related_intents: [itd-8]
finding_ref: cfk-ebc1fbc41a0cdd76
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262033030496. Same draft edit as iss-2610262033030496, set against a different statement of the same current state."
duplicates: [iss-2610262033030496]
---

itd-8 extends a `code-rescuer` agent it calls currently principle-only, but the agent catalogue lists `code-rescuer` among the agents still to be built

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-8-with-code-bundling.md:31` — "`code-rescuer` agent extended (currently principle-only)"
- End B: `.abcd/development/brief/05-internals/01-agents.md:76` — "The design roster still to be built"

itd-8's scope extends `code-rescuer`, which it calls currently principle-only, so it assumes the agent ships. The agent catalogue lists `code-rescuer` in the design roster still to be built, none of whose agents exists in `agents/`. itd-8 extends an agent that has to be built first.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 850.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262033030496. Same draft edit as iss-2610262033030496, set against a different statement of the same current state.
