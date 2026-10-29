---
schema_version: 1
id: "iss-2610262032113433"
slug: "itd-42-ships-its-coherence-tier-as-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 234 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:71"
remedy: "none (filed automatically)"
related_intents: [itd-42]
finding_ref: cfk-7a418575d0ac7a5c
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The record says its press release and scope are read through the re-scope, and its fidelity audit lists the grill-verb wording as diverged; the CLI registers prepass and no grill."
---

itd-42 ships its coherence tier as a capability of an existing `/abcd:intent grill`; the naming register says the verb is staged and no `intent grill` sub-verb is registered

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:71` — "this is a capability of the existing `/abcd:intent grill`, not a sibling verb."
- End B: `.abcd/development/brief/02-constraints/04-naming.md:44` — "**staged** (itd-27), and no `intent grill` sub-verb is registered."

itd-42, on the shipped shelf, scopes itself as 'a capability of the existing `/abcd:intent grill`, not a sibling verb', which presupposes that the grill sub-verb exists. 04-naming says the opposite: `/abcd:intent grill` is staged (itd-27) and no `intent grill` sub-verb is registered. A shipped intent cannot extend a sub-verb that the register says does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 234.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The record says its press release and scope are read through the re-scope, and its fidelity audit lists the grill-verb wording as diverged; the CLI registers prepass and no grill.
