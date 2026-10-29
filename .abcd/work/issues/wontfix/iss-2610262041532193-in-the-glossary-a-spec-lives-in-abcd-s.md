---
schema_version: 1
id: "iss-2610262041532193"
slug: "in-the-glossary-a-spec-lives-in-abcd-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1032 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/spec.md:4"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-143153d8487146f8
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both ends were true when written and adr-26 says which holds today; the glossary and the code describe the native store."
---

In the glossary a spec lives in abcd's native spec store; itd-34 maps each intent to a flow-next spec planned through /flow-next:plan.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/spec.md:4` — "native spec store that implements one or more intents"
- End B: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:21` — "A standalone intent maps 1:1 to a flow-next `spec`"

The spec entry defines a spec as a block of work in abcd's native spec store. Elsewhere the brief has `abcd intent plan` scaffold the spec there, with no external tool. itd-34 says a standalone intent 'maps 1:1 to a flow-next spec', that planning runs `/flow-next:plan` and `plan-review`, and that a discipline never gets a flow-next spec. The two documents name different stores and different planners for the same record.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1032.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Both ends were true when written and adr-26 says which holds today; the glossary and the code describe the native store.
