---
schema_version: 1
id: "iss-230"
slug: "one-tap-micro-prompt-channel"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "2026-08-15 forward-plans grill: micro-prompt graduation settled"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed H): Plan the one-tap micro-prompt channel for small rulings inside the facilitator-experience plan?"
remedy: "Waits on planning ruling H (plan the channel inside the facilitator-experience plan): if planned, define a micro-prompt record (one question, two or three enumerated options, a default, the record id it rules on) that abcd emits as JSON for the host to render through its own question tool, mapping accept, decline and cancel onto ruled, refused and still owed, proven by a core test that round-trips each outcome into the decision log; if not planned, strike the open question from itd-111 and resolve this record as wontfix with that reason."
---

offer a one-tap micro-prompt channel (survey pattern) for maintainer decisions: when abcd needs a single small human ruling (a yes/no, a pick from two or three options), it renders one micro-prompt the facilitator can answer in one tap instead of a full interview session. Lifted out of itd-111's open questions during the 2026-08-15 forward-plans grill so the staleness intent keeps a clean scope; strike the open question from itd-111 when it goes through intent plan. Facilitator-touch item, homed in the facilitator-experience plan.

## Remedy grounds (2026-09-29)

- MCP elicitation (https://modelcontextprotocol.io/specification/2025-06-18/client/elicitation, consulted 2026-09-29) restricts a request to a flat schema of enums, booleans, strings and numbers and answers with a three-action model (accept, decline, cancel); taking that shape keeps the prompt host-delegated (the host renders, abcd never draws) and lets a later MCP front door carry it unchanged.
- Rejected: abcd drawing its own terminal prompt, which breaks host delegation and cannot reach a person away from the terminal.
