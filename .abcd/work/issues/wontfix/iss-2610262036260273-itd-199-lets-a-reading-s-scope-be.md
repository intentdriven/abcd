---
schema_version: 1
id: "iss-2610262036260273"
slug: "itd-199-lets-a-reading-s-scope-be"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 647 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-199-a-reading-is-about-something-narrower.md:19"
remedy: "none (filed automatically)"
related_intents: [itd-199]
finding_ref: cfk-fd467e20bfe7b653
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The later ADR says which holds."
---

itd-199 lets a reading's scope be widened or narrowed at invocation and records the override; the reading chapter says the invocation is a position and a target with no override and nothing to stamp

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-199-a-reading-is-about-something-narrower.md:19` — "A scope may be widened or narrowed at invocation while calibrating, and the manifest then records the scope a run actually had"
- End B: `.abcd/development/brief/04-surfaces/23-reading.md:59` — "There is no override at the invocation and nothing to stamp."

itd-199, in shipped/, adds a scope operand the verb refuses to run without, lets it be widened or narrowed at invocation, and stamps an overridden run in the manifest. 23-reading says assembly takes a position and a target state and nothing else, that what a run is handed is not an operand, and that there is no override at the invocation. An assembler cannot both require and refuse a third operand.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 647.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The later ADR says which holds.
